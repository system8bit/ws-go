package ppu

// RenderScanline renders a single scanline into the framebuffer.
func (p *PPU) RenderScanline(line int) {
	if line < 0 || line >= ScreenHeight {
		return
	}

	p.CurrentLine = byte(line)
	p.Scanline = line

	if byte(line) == p.LineCompare {
		p.LineMatchFlag = true
	}

	p.renderBackground(line)
	p.FGDrawn = [ScreenWidth]bool{}

	if p.DispCtrl&0x01 != 0 {
		p.renderBGLayer(line, 1, false, false)
	}
	if p.DispCtrl&0x02 != 0 {
		winEnable := p.DispCtrl&0x20 != 0
		winOutside := p.DispCtrl&0x10 != 0
		p.renderBGLayer(line, 2, winEnable, winOutside)
	}
	if p.DispCtrl&0x04 != 0 {
		p.renderSprites(line)
	}
}

// renderBackground fills the scanline with the background color.
func (p *PPU) renderBackground(line int) {
	backColor := p.RenderBackColor
	palIdx := int((backColor >> 4) & 0x0F)
	colIdx := int(backColor & 0x0F)
	if p.WsVMode() == 0 {
		colIdx &= 0x03
	}

	r, g, b := p.GetColorFromPalette(palIdx, colIdx)
	baseOffset := line * ScreenWidth * 4
	for x := 0; x < ScreenWidth; x++ {
		off := baseOffset + x*4
		p.Framebuffer[off+0] = r
		p.Framebuffer[off+1] = g
		p.Framebuffer[off+2] = b
		p.Framebuffer[off+3] = 0xFF
	}
}

// isColor0Transparent returns true if color index 0 should be transparent
// for the given layer and palette. Mednafen gfx.cpp verified.
//
// Rules:
//   - SCR1 color mode: color 0 always opaque
//   - SCR2 color mode: color 0 always transparent
//   - Mono mode (both layers): transparent if palette bit 2 set
//   - Sprites color mode: color 0 always transparent
//   - Sprites mono mode: transparent if palette bit 2 set
func (p *PPU) isColor0Transparent(layer int, palIdx int) bool {
	if p.WsVMode()&0x02 != 0 {
		// Color mode: SCR1 opaque, SCR2/sprites transparent
		return layer != 1
	}
	// Mono mode: palette bit 2 controls transparency
	return palIdx&0x04 != 0
}

// layerScroll returns scroll registers and map base for the given layer.
func (p *PPU) layerScroll(layer int) (scrollX, scrollY byte, mapBase int) {
	if layer == 1 {
		return p.SCR1ScrollX, p.SCR1ScrollY, int(p.RenderMapBase&0x0F) << 11
	}
	return p.SCR2ScrollX, p.SCR2ScrollY, int((p.RenderMapBase>>4)&0x0F) << 11
}

// mapEntry contains decoded fields from a 16-bit background map entry.
type mapEntry struct {
	tileNum int
	palIdx  int
	bank    int
	hFlip   bool
	vFlip   bool
}

// decodeMapEntry decodes a 16-bit little-endian map entry from IRAM.
func decodeMapEntry(iram []byte, addr int) mapEntry {
	lo := uint16(iram[addr])
	hi := uint16(iram[addr+1])
	e := lo | (hi << 8)
	return mapEntry{
		tileNum: int(e & 0x01FF),
		palIdx:  int((e >> 9) & 0x0F),
		bank:    int((e >> 13) & 0x01),
		hFlip:   e&0x4000 != 0,
		vFlip:   e&0x8000 != 0,
	}
}

// renderBGLayer renders one background layer (SCR1=1 or SCR2=2).
// When winEnable is true, applies SCR2 window clipping.
func (p *PPU) renderBGLayer(line, layer int, winEnable, winOutside bool) {
	iram := p.renderIRAM()
	if len(iram) == 0 {
		return
	}

	scrollX, scrollY, mapBase := p.layerScroll(layer)
	baseOffset := line * ScreenWidth * 4

	// Window coordinates (only used when winEnable is true)
	winX0, winY0 := int(p.SCR2WinX0), int(p.SCR2WinY0)
	winX1, winY1 := int(p.SCR2WinX1), int(p.SCR2WinY1)

	for x := 0; x < ScreenWidth; x++ {
		// Window clipping for SCR2
		if winEnable {
			inWindow := x >= winX0 && x <= winX1 && line >= winY0 && line <= winY1
			if winOutside == inWindow {
				continue
			}
		}

		sX := (int(scrollX) + x) & 0xFF
		sY := (int(scrollY) + line) & 0xFF

		mapAddr := mapBase + (sY/8*32+sX/8)*2
		if mapAddr+1 >= len(iram) {
			continue
		}
		me := decodeMapEntry(iram, mapAddr)

		pixelX, pixelY := sX%8, sY%8
		if me.hFlip {
			pixelX = 7 - pixelX
		}
		if me.vFlip {
			pixelY = 7 - pixelY
		}

		colorIdx := p.getTilePixel(me.tileNum, pixelX, pixelY, me.bank)

		if colorIdx == 0 && p.isColor0Transparent(layer, me.palIdx) {
			continue
		}

		r, g, b := p.GetColorFromPalette(me.palIdx, colorIdx)
		off := baseOffset + x*4
		p.Framebuffer[off+0] = r
		p.Framebuffer[off+1] = g
		p.Framebuffer[off+2] = b
		p.Framebuffer[off+3] = 0xFF

		if layer == 2 {
			p.FGDrawn[x] = true
		}
	}
}

// getTilePixel reads a pixel from tile data in IRAM.
func (p *PPU) getTilePixel(tileNum, pixelX, pixelY, bank int) int {
	iram := p.renderIRAM()
	if len(iram) == 0 {
		return 0
	}

	vMode := p.WsVMode()
	if vMode&0x07 == 0 {
		bank = 0
	}

	switch vMode {
	case 7: // Packed 4bpp
		base := 0x4000 + bank*0x4000
		rowAddr := base + tileNum*32 + pixelY*4
		if rowAddr+3 >= len(iram) {
			return 0
		}
		b := iram[rowAddr+pixelX/2]
		if pixelX%2 == 0 {
			return int(b >> 4)
		}
		return int(b & 0x0F)

	case 6: // Planar 4bpp
		base := 0x4000 + bank*0x4000
		rowAddr := base + tileNum*32 + pixelY*4
		if rowAddr+3 >= len(iram) {
			return 0
		}
		bitPos := uint(7 - pixelX)
		return int((iram[rowAddr]>>bitPos)&1) |
			int((iram[rowAddr+1]>>bitPos)&1)<<1 |
			int((iram[rowAddr+2]>>bitPos)&1)<<2 |
			int((iram[rowAddr+3]>>bitPos)&1)<<3

	default: // Mode 0 (mono 2bpp)
		tileAddr := (0x2000 + tileNum*16) & 0x3FFF
		rowAddr := (tileAddr + pixelY*2) & 0x3FFF
		if rowAddr+1 >= len(iram) {
			return 0
		}
		bitPos := uint(7 - pixelX)
		return int((iram[rowAddr]>>bitPos)&1) |
			int((iram[rowAddr+1]>>bitPos)&1)<<1
	}
}

// renderSprites renders all visible sprites for the given scanline.
func (p *PPU) renderSprites(line int) {
	if len(p.renderIRAM()) == 0 {
		return
	}

	activeBuf := 0
	if p.SpriteFrameActive {
		activeBuf = 1
	}
	count := p.SpriteCountCache[activeBuf]
	if count == 0 {
		return
	}

	baseOffset := line * ScreenWidth * 4
	sprWinEnabled := p.DispCtrl&0x08 != 0

	// Sprites processed in reverse order (last = highest priority)
	for i := count - 1; i >= 0; i-- {
		stab := p.SpriteTableCache[activeBuf][i]
		attr := stab[1]
		sprX, sprY := int(stab[3]), int(stab[2])
		tileNum := int(stab[0]) | int(attr&0x01)<<8
		palIdx := int((attr >> 1) & 0x07)
		sprWinFlag := attr&0x10 != 0
		priority := attr&0x20 != 0
		hFlip := attr&0x40 != 0
		vFlip := attr&0x80 != 0

		if sprX >= 249 {
			sprX -= 256
		}
		if sprY > 150 {
			sprY = int(int8(byte(sprY)))
		}

		relY := line - sprY
		if relY < 0 || relY >= 8 {
			continue
		}

		pixelY := relY
		if vFlip {
			pixelY = 7 - relY
		}

		spritePal := palIdx + 8

		for px := 0; px < 8; px++ {
			screenX := sprX + px
			if screenX < 0 || screenX >= ScreenWidth {
				continue
			}

			pixelX := px
			if hFlip {
				pixelX = 7 - px
			}

			colorIdx := p.getTilePixel(tileNum, pixelX, pixelY, 0)

			// Sprite layer = 3 (not 1 or 2) → isColor0Transparent returns true in color mode
			if colorIdx == 0 && p.isColor0Transparent(3, palIdx) {
				continue
			}

			if !priority && p.FGDrawn[screenX] {
				continue
			}

			if sprWinEnabled {
				inWin := screenX >= int(p.SprWinX0) && screenX <= int(p.SprWinX1) &&
					line >= int(p.SprWinY0) && line <= int(p.SprWinY1)
				if sprWinFlag == inWin {
					continue
				}
			}

			r, g, b := p.GetColorFromPalette(spritePal, colorIdx)
			off := baseOffset + screenX*4
			p.Framebuffer[off+0] = r
			p.Framebuffer[off+1] = g
			p.Framebuffer[off+2] = b
			p.Framebuffer[off+3] = 0xFF
		}
	}
}
