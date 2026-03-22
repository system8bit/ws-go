package ppu

import "testing"

func TestPortWriteMask(t *testing.T) {
	p := New(false) // mono mode

	tests := []struct {
		port byte
		mask byte
	}{
		{0x00, 0x3F}, // DispCtrl
		{0x01, 0x07}, // BackColor mono
		{0x02, 0x00}, // CurrentLine read-only
		{0x04, 0x1F}, // SpriteBase mono
		{0x05, 0x7F}, // SpriteFirst
		{0x07, 0x77}, // MapBase
		{0x14, 0x01}, // LCD IF Ctrl
		{0x15, 0x3F}, // LCD Icons
		{0x18, 0x00}, // unused
		{0x19, 0x00}, // unused
		{0x28, 0x70}, // SPR palette even
		{0x29, 0x77}, // SPR palette odd
		{0x20, 0x77}, // BG palette even
	}
	for _, tt := range tests {
		got := p.PortWriteMask(tt.port)
		if got != tt.mask {
			t.Errorf("PortWriteMask(0x%02X) = 0x%02X, want 0x%02X", tt.port, got, tt.mask)
		}
	}
}

func TestPortWriteMaskColor(t *testing.T) {
	p := New(true) // color mode
	// Force color video mode by setting LCDCtrl
	p.LCDCtrl = 0xE0 // wsVMode = 7

	if m := p.PortWriteMask(0x01); m != 0xFF {
		t.Errorf("Color BackColor mask = 0x%02X, want 0xFF", m)
	}
	if m := p.PortWriteMask(0x04); m != 0x3F {
		t.Errorf("Color SpriteBase mask = 0x%02X, want 0x3F", m)
	}
}

func TestWritePortMasking(t *testing.T) {
	p := New(false)

	// Write 0xFF to DispCtrl — only bits 0-5 should be stored
	p.WritePort(0x00, 0xFF)
	if p.DispCtrl != 0x3F {
		t.Errorf("DispCtrl after write 0xFF = 0x%02X, want 0x3F", p.DispCtrl)
	}

	// Write 0xFF to MapBase — bits 3,7 should be 0
	p.WritePort(0x07, 0xFF)
	if p.MapBase != 0x77 {
		t.Errorf("MapBase after write 0xFF = 0x%02X, want 0x77", p.MapBase)
	}

	// Write 0xFF to SpriteFirst — bit 7 should be 0
	p.WritePort(0x05, 0xFF)
	if p.SpriteFirst != 0x7F {
		t.Errorf("SpriteFirst after write 0xFF = 0x%02X, want 0x7F", p.SpriteFirst)
	}
}

func TestMonoPaletteWriteMask(t *testing.T) {
	p := New(false)

	// Write 0xFF to BG palette port 0x20 — each nibble masked to 3 bits (0x07)
	p.WritePort(0x20, 0xFF)
	got := p.ReadMonoPalette(0x20)
	if got != 0x77 {
		t.Errorf("MonoPalette 0x20 after write 0xFF = 0x%02X, want 0x77", got)
	}

	// Write 0xFF to SPR palette port 0x28 — low nibble masked by PortWriteMask (0x70)
	p.WritePort(0x28, 0xFF)
	got = p.ReadMonoPalette(0x28)
	// PortWriteMask(0x28) = 0x70, so val = 0xFF & 0x70 = 0x70
	// WriteMonoPalette stores: low = 0x70 & 0x07 = 0, high = (0x70>>4) & 0x07 = 7
	// ReadMonoPalette: (7<<4) | 0 = 0x70
	if got != 0x70 {
		t.Errorf("MonoPalette SPR 0x28 after write 0xFF = 0x%02X, want 0x70", got)
	}
}
