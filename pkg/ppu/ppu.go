package ppu

const (
	ScreenWidth  = 224
	ScreenHeight = 144
	TotalLines   = 159 // 144 visible + 15 VBlank
)

// PPU represents the WonderSwan Picture Processing Unit.
type PPU struct {
	// Framebuffer: 224*144 pixels, RGBA (4 bytes per pixel)
	// Rendering writes to Framebuffer; the frontend reads DisplayBuffer.
	// SwapBuffers() copies Framebuffer → DisplayBuffer at frame end.
	Framebuffer   [ScreenWidth * ScreenHeight * 4]byte
	DisplayBuffer [ScreenWidth * ScreenHeight * 4]byte

	// Current scanline
	Scanline int

	// Control registers (from I/O ports)
	DispCtrl    byte // 0x00 - Display control
	BackColor   byte // 0x01 - Background color (palette index)
	CurrentLine byte // 0x02 - Current scanline (read-only)
	LineCompare byte // 0x03 - Line compare (for line interrupt)
	SpriteBase  byte // 0x04 - Sprite table base address
	SpriteFirst byte // 0x05 - First sprite to display
	SpriteCount byte // 0x06 - Number of sprites to display
	MapBase     byte // 0x07 - SCR1/SCR2 map base addresses

	// Scroll registers
	SCR1ScrollX byte // 0x10
	SCR1ScrollY byte // 0x11
	SCR2ScrollX byte // 0x12
	SCR2ScrollY byte // 0x13

	// Window registers
	SCR2WinX0 byte // 0x08 - SCR2 window left
	SCR2WinY0 byte // 0x09 - SCR2 window top
	SCR2WinX1 byte // 0x0A - SCR2 window right
	SCR2WinY1 byte // 0x0B - SCR2 window bottom
	SprWinX0  byte // 0x0C - Sprite window left
	SprWinY0  byte // 0x0D - Sprite window top
	SprWinX1  byte // 0x0E - Sprite window right
	SprWinY1  byte // 0x0F - Sprite window bottom

	// LCD control
	LCDCtrl   byte // 0x60 — also determines video mode (wsVMode = LCDCtrl >> 5)
	LCDIcons  byte // 0x15
	LCDVtotal byte // 0x16 — total vertical lines (Mednafen: vtotal = max(143, LCDVtotal) + 1)

	// WSC-specific
	IsColor bool

	// Shade LUT: maps shade values (0-7) to LCD gray levels (0-15)
	// Written via ports 0x1C-0x1F. Each port holds two 4-bit entries.
	ShadeLUT [8]byte

	// Palettes
	MonoPalette [16][4]byte // 16 palettes, 4 entries each (shade 0-15 for mono)

	// Reference to IRAM for tile/map data access
	IRAM []byte

	// Shadow copy of IRAM snapshotted on MapBase change for stable rendering.
	RenderIRAM []byte

	// Snapshotted display registers (captured with IRAM snapshot)
	RenderMapBase  byte
	RenderBackColor byte

	// Interrupt flags
	VBlankFlag    bool
	LineMatchFlag bool

	// Per-pixel FG flag for the current scanline. Set by SCR2 (FG layer)
	// rendering so sprites can check whether FG has drawn a pixel.
	// Mednafen uses b_bg[] |0x10 for this purpose.
	FGDrawn [ScreenWidth]bool

	// Sprite table double-buffering (Mednafen-verified).
	// Sprite data is cached at line 142 for use in the NEXT frame.
	SpriteTableCache [2][128][4]byte // two buffers, 128 sprites, 4 bytes each
	SpriteCountCache [2]int
	SpriteFrameActive bool // toggles at VBlank (line 144)
}

// New creates a new PPU instance.
func New(isColor bool) *PPU {
	p := &PPU{
		IsColor: isColor,
	}
	p.Reset()
	return p
}

// Reset resets the PPU to its initial state.
func (p *PPU) Reset() {
	p.Scanline = 0
	p.DispCtrl = 0
	p.BackColor = 0
	p.CurrentLine = 0
	p.LineCompare = 0
	p.SpriteBase = 0
	p.SpriteFirst = 0
	p.SpriteCount = 0
	p.MapBase = 0
	p.SCR1ScrollX = 0
	p.SCR1ScrollY = 0
	p.SCR2ScrollX = 0
	p.SCR2ScrollY = 0
	p.LCDCtrl = 0
	p.LCDIcons = 0
	p.LCDVtotal = 158 // default: 159 total lines (Mednafen wsDefaultVtotal=159)
	p.VBlankFlag = false
	p.LineMatchFlag = false

	// Clear framebuffer to white
	for i := range p.Framebuffer {
		p.Framebuffer[i] = 0xFF
	}

	// Allocate render snapshot buffer
	if len(p.IRAM) > 0 {
		p.RenderIRAM = make([]byte, len(p.IRAM))
	}

	// Clear palettes
	p.MonoPalette = [16][4]byte{}
}

// WsVMode returns the current video mode derived from LCDCtrl (port 0x60).
// 0 = mono 2bpp, 6 = color planar 4bpp, 7 = color packed 4bpp.
func (p *PPU) WsVMode() int {
	return int(p.LCDCtrl >> 5)
}

// TotalLinesForFrame returns the dynamic total lines per frame.
// Mednafen: vtotal = max(144, LCDVtotal) + 1, default 159 when LCDVtotal=0.
func (p *PPU) TotalLinesForFrame() int {
	vt := int(p.LCDVtotal)
	if vt < 144 {
		vt = 143 // Mednafen: max(143, LCDVtotal)
	}
	return vt + 1
}

// renderIRAM returns the snapshotted IRAM if available, otherwise the live IRAM.
func (p *PPU) renderIRAM() []byte {
	if len(p.RenderIRAM) > 0 {
		return p.RenderIRAM
	}
	return p.IRAM
}

// SnapshotTiles copies the current IRAM and display registers into the
// render snapshot buffer. Called when MapBase changes (buffer swap) so
// rendering uses consistent tile+map+register data.
func (p *PPU) SnapshotTiles() {
	if len(p.IRAM) > 0 {
		if len(p.RenderIRAM) == 0 {
			p.RenderIRAM = make([]byte, len(p.IRAM))
		}
		copy(p.RenderIRAM, p.IRAM)
	}
	p.RenderMapBase = p.MapBase
	p.RenderBackColor = p.BackColor
}

// SwapBuffers copies the completed Framebuffer to DisplayBuffer.
// Called at the end of each frame so the frontend always reads a complete frame.
func (p *PPU) SwapBuffers() {
	copy(p.DisplayBuffer[:], p.Framebuffer[:])
}

// CacheSpriteTable snapshots the sprite table into the inactive buffer.
// Called at line 142 (Mednafen: wsExecuteLine, wsLine==142).
func (p *PPU) CacheSpriteTable() {
	bufIdx := 0
	if p.SpriteFrameActive {
		bufIdx = 1
	}
	// Cache into the INACTIVE buffer (opposite of current active)
	inactiveBuf := 1 - bufIdx

	count := int(p.SpriteCount)
	if count > 0x80 {
		count = 0x80
	}
	p.SpriteCountCache[inactiveBuf] = count

	base := int(p.SpriteBase) << 9
	first := int(p.SpriteFirst)
	for i := 0; i < count; i++ {
		idx := ((first + i) & 0x7F)
		addr := base + idx*4
		if addr+3 < len(p.IRAM) {
			p.SpriteTableCache[inactiveBuf][i][0] = p.IRAM[addr]
			p.SpriteTableCache[inactiveBuf][i][1] = p.IRAM[addr+1]
			p.SpriteTableCache[inactiveBuf][i][2] = p.IRAM[addr+2]
			p.SpriteTableCache[inactiveBuf][i][3] = p.IRAM[addr+3]
		}
	}
}

// FlipSpriteFrame toggles the active sprite buffer. Called at VBlank (line 144).
func (p *PPU) FlipSpriteFrame() {
	p.SpriteFrameActive = !p.SpriteFrameActive
}

// ReadPort reads a PPU I/O register.
// HandlesRead returns true if the PPU explicitly handles reads for this port.
// Ports in the PPU range (0x00-0x3F, 0x60) that are NOT handled here
// will fall back to IOPorts[] stored value.
func (p *PPU) HandlesRead(port byte) bool {
	switch {
	case port <= 0x13, port == 0x15, port == 0x16,
		port >= 0x1C && port <= 0x1F,
		port >= 0x20 && port <= 0x3F,
		port == 0x60:
		return true
	default:
		return false
	}
}

func (p *PPU) ReadPort(port byte) byte {
	switch {
	case port == 0x00:
		return p.DispCtrl
	case port == 0x01:
		return p.BackColor
	case port == 0x02:
		return p.CurrentLine
	case port == 0x03:
		return p.LineCompare
	case port == 0x04:
		return p.SpriteBase
	case port == 0x05:
		return p.SpriteFirst
	case port == 0x06:
		return p.SpriteCount
	case port == 0x07:
		return p.MapBase
	case port == 0x08:
		return p.SCR2WinX0
	case port == 0x09:
		return p.SCR2WinY0
	case port == 0x0A:
		return p.SCR2WinX1
	case port == 0x0B:
		return p.SCR2WinY1
	case port == 0x0C:
		return p.SprWinX0
	case port == 0x0D:
		return p.SprWinY0
	case port == 0x0E:
		return p.SprWinX1
	case port == 0x0F:
		return p.SprWinY1
	case port == 0x10:
		return p.SCR1ScrollX
	case port == 0x11:
		return p.SCR1ScrollY
	case port == 0x12:
		return p.SCR2ScrollX
	case port == 0x13:
		return p.SCR2ScrollY
	case port == 0x15:
		return p.LCDIcons
	case port == 0x16:
		return p.LCDVtotal
	case port >= 0x1C && port <= 0x1F:
		idx := (port - 0x1C) * 2
		return p.ShadeLUT[idx] | (p.ShadeLUT[idx+1] << 4)
	case port >= 0x20 && port <= 0x3F:
		return p.ReadMonoPalette(port)
	case port == 0x60:
		return p.LCDCtrl
	default:
		return 0
	}
}

// WritePort writes a PPU I/O register.
// ppuWriteMask returns the writable bit mask for a PPU port.
// Bits not in the mask are ignored on write.
func (p *PPU) PortWriteMask(port byte) byte {
	switch port {
	case 0x00: // DispCtrl: bits 0-5 writable
		return 0x3F
	case 0x01: // BackColor: mono=0x07, color=0xFF
		if p.WsVMode() == 0 {
			return 0x07
		}
		return 0xFF
	case 0x02: // CurrentLine: read-only
		return 0x00
	case 0x04: // SpriteBase: mono=0x1F, color=0x3F
		if p.WsVMode() == 0 {
			return 0x1F
		}
		return 0x3F
	case 0x05: // SpriteFirst: 7 bits
		return 0x7F
	case 0x07: // MapBase: two 4-bit fields, bit3 of each always 0
		return 0x77
	case 0x14: // LCD IF Ctrl: bit 0 only
		return 0x01
	case 0x15: // LCD Icons: bits 0-5
		return 0x3F
	case 0x17: // unused but writable on SPHINX
		return 0xFF
	case 0x18, 0x19, 0x1A, 0x1B: // unused gap before ShadeLUT
		return 0x00
	case 0x20, 0x22, 0x24, 0x26: // BG palette even (entry 0+1): both nibbles writable
		return 0x77
	case 0x21, 0x23, 0x25, 0x27: // BG palette odd (entry 2+3): both nibbles writable
		return 0x77
	case 0x28, 0x2A, 0x2C, 0x2E: // SPR palette even: entry 0 (low nibble) = color 0 (unused), high writable
		return 0x70
	case 0x29, 0x2B, 0x2D, 0x2F: // SPR palette odd: both nibbles writable
		return 0x77
	case 0x30, 0x32, 0x34, 0x36: // BG palette 8-15 even
		return 0x77
	case 0x31, 0x33, 0x35, 0x37: // BG palette 8-15 odd
		return 0x77
	case 0x38, 0x3A, 0x3C, 0x3E: // SPR palette 8-15 even
		return 0x70
	case 0x39, 0x3B, 0x3D, 0x3F: // SPR palette 8-15 odd
		return 0x77
	default:
		return 0xFF
	}
}

func (p *PPU) WritePort(port byte, val byte) {
	val &= p.PortWriteMask(port)
	switch {
	case port == 0x00:
		p.DispCtrl = val
	case port == 0x01:
		p.BackColor = val
	case port == 0x02:
		// Current line is read-only; ignore writes
	case port == 0x03:
		p.LineCompare = val
	case port == 0x04:
		p.SpriteBase = val
	case port == 0x05:
		p.SpriteFirst = val
	case port == 0x06:
		p.SpriteCount = val
	case port == 0x07:
		p.MapBase = val
	case port == 0x08:
		p.SCR2WinX0 = val
	case port == 0x09:
		p.SCR2WinY0 = val
	case port == 0x0A:
		p.SCR2WinX1 = val
	case port == 0x0B:
		p.SCR2WinY1 = val
	case port == 0x0C:
		p.SprWinX0 = val
	case port == 0x0D:
		p.SprWinY0 = val
	case port == 0x0E:
		p.SprWinX1 = val
	case port == 0x0F:
		p.SprWinY1 = val
	case port == 0x10:
		p.SCR1ScrollX = val
	case port == 0x11:
		p.SCR1ScrollY = val
	case port == 0x12:
		p.SCR2ScrollX = val
	case port == 0x13:
		p.SCR2ScrollY = val
	case port == 0x15:
		p.LCDIcons = val
	case port == 0x16:
		p.LCDVtotal = val
	case port >= 0x1C && port <= 0x1F:
		idx := (port - 0x1C) * 2
		p.ShadeLUT[idx] = val & 0x0F
		p.ShadeLUT[idx+1] = (val >> 4) & 0x0F
	case port >= 0x20 && port <= 0x3F:
		p.WriteMonoPalette(port, val)
	case port == 0x60:
		p.LCDCtrl = val
	}
}
