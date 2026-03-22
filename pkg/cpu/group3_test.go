package cpu

import "testing"

// TestMUL8 tests MUL r/m8 via instruction execution.
func TestMUL8(t *testing.T) {
	c, bus := newTestCPU()
	// MUL BL → opcode F6 /4, modrm=0xE3 (mod=3, reg=4, rm=3=BL)
	loadCode(bus, 0xF6, 0xE3)
	c.SetAL(0xFF)
	c.SetBL(0xFF)
	c.Step()
	if c.AX != 0xFE01 {
		t.Errorf("AX = 0x%04X, want 0xFE01", c.AX)
	}
	assertFlags(t, c, map[uint16]bool{FlagCF: true, FlagOF: true})
}

// TestIMUL8 tests IMUL r/m8.
func TestIMUL8(t *testing.T) {
	c, bus := newTestCPU()
	// IMUL CL → opcode F6, modrm=0xE9 (mod=3, reg=5, rm=1=CL)
	loadCode(bus, 0xF6, 0xE9)
	c.SetAL(0xFF) // -1
	c.SetCL(0xFF) // -1
	c.Step()
	if c.AX != 0x0001 {
		t.Errorf("AX = 0x%04X, want 0x0001 (-1*-1=1)", c.AX)
	}
	// AH=0 → CF=OF=0
	assertFlags(t, c, map[uint16]bool{FlagCF: false, FlagOF: false})
}

// TestDIV8 tests DIV r/m8.
func TestDIV8(t *testing.T) {
	c, bus := newTestCPU()
	// DIV BL → opcode F6, modrm=0xF3 (mod=3, reg=6, rm=3=BL)
	loadCode(bus, 0xF6, 0xF3)
	c.AX = 0x0107 // 263
	c.SetBL(2)
	c.Step()
	if c.AL() != 131 || c.AH() != 1 {
		t.Errorf("AL=%d AH=%d, want AL=131 AH=1", c.AL(), c.AH())
	}
}

// TestDIV8_DivByZero tests that DIV by 0 triggers INT 0.
func TestDIV8_DivByZero(t *testing.T) {
	c, bus := newTestCPU()
	// Set up IVT: INT 0 vector at 0000:0000 → CS:IP = 0x1000:0x0000
	bus.Write16(0, 0, 0x0000) // IP for INT 0
	bus.Write16(0, 2, 0x1000) // CS for INT 0
	// Place code at 0x0100 to avoid IVT collision
	c.IP = 0x0100
	bus.mem[0x0100] = 0xF6 // DIV r/m8
	bus.mem[0x0101] = 0xF3 // modrm: mod=3, reg=6, rm=3=BL
	c.AX = 0x0001
	c.SetBL(0)
	c.Step()
	if c.CS != 0x1000 || c.IP != 0x0000 {
		t.Errorf("CS:IP = %04X:%04X, want 1000:0000 (INT 0 handler)", c.CS, c.IP)
	}
}

// TestNEG8 tests NEG r/m8.
func TestNEG8(t *testing.T) {
	tests := []struct {
		val  byte
		want byte
		cf   bool
	}{
		{0x01, 0xFF, true},
		{0x00, 0x00, false},
		{0x80, 0x80, true},
	}
	for _, tt := range tests {
		c, bus := newTestCPU()
		// NEG AL → opcode F6, modrm=0xD8 (mod=3, reg=3, rm=0=AL)
		loadCode(bus, 0xF6, 0xD8)
		c.SetAL(tt.val)
		c.Step()
		if c.AL() != tt.want {
			t.Errorf("NEG 0x%02X: got 0x%02X, want 0x%02X", tt.val, c.AL(), tt.want)
		}
		if cf := c.GetFlag(FlagCF); cf != tt.cf {
			t.Errorf("NEG 0x%02X: CF=%v, want %v", tt.val, cf, tt.cf)
		}
	}
}

// TestNOT8 tests NOT r/m8.
func TestNOT8(t *testing.T) {
	c, bus := newTestCPU()
	// NOT AL → opcode F6, modrm=0xD0 (mod=3, reg=2, rm=0=AL)
	loadCode(bus, 0xF6, 0xD0)
	c.SetAL(0x55)
	c.Step()
	if c.AL() != 0xAA {
		t.Errorf("NOT 0x55: got 0x%02X, want 0xAA", c.AL())
	}
}
