package ppu

// WriteMonoPalette writes a mono palette register.
// Ports 0x20-0x3F: each port holds 2 palette entries packed as nibbles.
// Port 0x20 + N*2: palette N, entries 0 (low nibble) and 1 (high nibble)
// Port 0x20 + N*2 + 1: palette N, entries 2 (low nibble) and 3 (high nibble)
func (p *PPU) WriteMonoPalette(port byte, val byte) {
	if port < 0x20 || port > 0x3F {
		return
	}
	offset := port - 0x20
	paletteIdx := int(offset / 2)
	entryPair := int(offset % 2)

	p.MonoPalette[paletteIdx][entryPair*2] = val & 0x07
	p.MonoPalette[paletteIdx][entryPair*2+1] = (val >> 4) & 0x07
}

// ReadMonoPalette reads a mono palette register.
func (p *PPU) ReadMonoPalette(port byte) byte {
	if port < 0x20 || port > 0x3F {
		return 0
	}
	offset := port - 0x20
	paletteIdx := int(offset / 2)
	entryPair := int(offset % 2)

	low := p.MonoPalette[paletteIdx][entryPair*2]
	high := p.MonoPalette[paletteIdx][entryPair*2+1]
	return (high << 4) | low
}

// GetColorFromPalette returns 8-bit RGB values for a given palette and color index.
// In mono mode (wsVMode==0), the shade is converted to grayscale via ShadeLUT.
// In color mode (wsVMode!=0), the 12-bit RGB is read from palette RAM in IRAM (0xFE00-0xFFFF).
func (p *PPU) GetColorFromPalette(paletteIdx, colorIdx int) (r, g, b byte) {
	if paletteIdx < 0 || paletteIdx >= 16 {
		return 0xFF, 0xFF, 0xFF
	}

	if p.WsVMode() == 0 {
		// Mono mode: palette entry is a shade value (0-7).
		if colorIdx < 0 || colorIdx >= 4 {
			return 0xFF, 0xFF, 0xFF
		}
		shade := p.MonoPalette[paletteIdx][colorIdx]
		if shade > 7 {
			shade = 7
		}
		lcdLevel := p.ShadeLUT[shade]
		if lcdLevel > 15 {
			lcdLevel = 15
		}
		// LCD level 0 = white (0xFF), LCD level 15 = black (0x00)
		gray := byte(255 - int(lcdLevel)*17)
		return gray, gray, gray
	}

	// Color mode: read 12-bit RGB from palette RAM in IRAM at 0xFE00-0xFFFF.
	// 16 palettes × 16 colors × 2 bytes = 512 bytes.
	if colorIdx < 0 || colorIdx >= 16 {
		return 0x00, 0x00, 0x00
	}
	iram := p.renderIRAM()
	addr := 0xFE00 + (paletteIdx*16+colorIdx)*2
	if addr+1 >= len(iram) {
		return 0x00, 0x00, 0x00
	}
	lo := uint16(iram[addr])
	hi := uint16(iram[addr+1])
	rgb := lo | (hi << 8)

	// 12-bit color: WSC format is xxxx RRRR GGGG BBBB
	// bits 0-3 = blue, bits 4-7 = green, bits 8-11 = red
	blue4 := byte(rgb & 0x0F)
	green4 := byte((rgb >> 4) & 0x0F)
	red4 := byte((rgb >> 8) & 0x0F)

	// Expand 4-bit to 8-bit: multiply by 17 (0x0 -> 0x00, 0xF -> 0xFF)
	return red4 * 17, green4 * 17, blue4 * 17
}
