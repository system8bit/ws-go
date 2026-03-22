package cpu

import "testing"

func TestDAA(t *testing.T) {
	tests := []struct {
		name string
		al   byte
		cfIn bool
		afIn bool
		want byte
		cf   bool
	}{
		{"0x0A→0x10", 0x0A, false, false, 0x10, false},
		{"0x9A→0x00 CF", 0x9A, false, false, 0x00, true},
		{"0x15 no adjust", 0x15, false, false, 0x15, false},
		{"0x0F AF set", 0x0F, false, true, 0x15, false},
		{"0xA0 CF in", 0xA0, true, false, 0x00, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, bus := newTestCPU()
			loadCode(bus, 0x27) // DAA
			c.SetAL(tt.al)
			c.SetFlag(FlagCF, tt.cfIn)
			c.SetFlag(FlagAF, tt.afIn)
			c.Step()
			if c.AL() != tt.want {
				t.Errorf("AL = 0x%02X, want 0x%02X", c.AL(), tt.want)
			}
			if c.GetFlag(FlagCF) != tt.cf {
				t.Errorf("CF = %v, want %v", c.GetFlag(FlagCF), tt.cf)
			}
		})
	}
}

func TestDAS(t *testing.T) {
	tests := []struct {
		name string
		al   byte
		cfIn bool
		afIn bool
		want byte
		cf   bool
	}{
		{"0x10→0x10 no adj", 0x10, false, false, 0x10, false},
		{"0x0A AF→0x04", 0x0A, false, true, 0x04, false},
		{"0xA0 CF→0x40", 0xA0, true, false, 0x40, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, bus := newTestCPU()
			loadCode(bus, 0x2F) // DAS
			c.SetAL(tt.al)
			c.SetFlag(FlagCF, tt.cfIn)
			c.SetFlag(FlagAF, tt.afIn)
			c.Step()
			if c.AL() != tt.want {
				t.Errorf("AL = 0x%02X, want 0x%02X", c.AL(), tt.want)
			}
			if c.GetFlag(FlagCF) != tt.cf {
				t.Errorf("CF = %v, want %v", c.GetFlag(FlagCF), tt.cf)
			}
		})
	}
}

func TestAAM(t *testing.T) {
	c, bus := newTestCPU()
	// AAM: opcode D4 0A (imm8 ignored, always /10)
	loadCode(bus, 0xD4, 0x0A)
	c.SetAL(35)
	c.Step()
	if c.AH() != 3 || c.AL() != 5 {
		t.Errorf("AAM 35: AH=%d AL=%d, want AH=3 AL=5", c.AH(), c.AL())
	}
}

func TestAAD(t *testing.T) {
	c, bus := newTestCPU()
	// AAD: opcode D5 0A
	loadCode(bus, 0xD5, 0x0A)
	c.SetAH(3)
	c.SetAL(5)
	c.Step()
	if c.AL() != 35 || c.AH() != 0 {
		t.Errorf("AAD 3,5: AH=%d AL=%d, want AH=0 AL=35", c.AH(), c.AL())
	}
}

func TestAAA(t *testing.T) {
	c, bus := newTestCPU()
	loadCode(bus, 0x37) // AAA
	c.AX = 0x000A       // AL=0x0A > 9 → adjust
	c.Step()
	// AAA: AX += 0x106, AL &= 0x0F
	// AX = 0x010A + 0x0106 = nah, AX starts at 0x000A, += 0x106 = 0x0110, AL &= 0x0F → 0x0100
	if c.AL() != 0x00 || c.AH() != 0x01 {
		t.Errorf("AAA: AX=0x%04X, want AH=1 AL=0", c.AX)
	}
	assertFlags(t, c, map[uint16]bool{FlagAF: true, FlagCF: true})
}

func TestAAS(t *testing.T) {
	c, bus := newTestCPU()
	loadCode(bus, 0x3F) // AAS
	c.AX = 0x010A       // AL=0x0A > 9 → adjust
	c.Step()
	// AAS: AX -= 0x106 = 0x010A - 0x0106 = 0x0004, AL &= 0x0F → 0x0004
	if c.AL() != 0x04 || c.AH() != 0x00 {
		t.Errorf("AAS: AX=0x%04X, want AH=0 AL=4", c.AX)
	}
	assertFlags(t, c, map[uint16]bool{FlagAF: true, FlagCF: true})
}
