package cpu

import "testing"

func TestShiftRotate8(t *testing.T) {
	tests := []struct {
		name    string
		op      int
		val     byte
		count   byte
		cfIn    bool
		wantVal byte
		wantCF  bool
	}{
		// ROL (op=0)
		{"ROL 0x80,1", 0, 0x80, 1, false, 0x01, true},
		{"ROL 0x01,1", 0, 0x01, 1, false, 0x02, false},
		{"ROL count=0", 0, 0x55, 0, false, 0x55, false},

		// ROR (op=1)
		{"ROR 0x01,1", 1, 0x01, 1, false, 0x80, true},
		{"ROR 0x80,1", 1, 0x80, 1, false, 0x40, false},

		// RCL (op=2)
		{"RCL 0x80,1 CF=0", 2, 0x80, 1, false, 0x00, true},
		{"RCL 0x80,1 CF=1", 2, 0x80, 1, true, 0x01, true},
		{"RCL 0x00,1 CF=1", 2, 0x00, 1, true, 0x01, false},

		// RCR (op=3)
		{"RCR 0x01,1 CF=0", 3, 0x01, 1, false, 0x00, true},
		{"RCR 0x01,1 CF=1", 3, 0x01, 1, true, 0x80, true},

		// SHL (op=4)
		{"SHL 0x80,1", 4, 0x80, 1, false, 0x00, true},
		{"SHL 0x01,1", 4, 0x01, 1, false, 0x02, false},
		{"SHL count=8", 4, 0x01, 8, false, 0x00, true},
		{"SHL count=9", 4, 0xFF, 9, false, 0x00, false},

		// SHR (op=5)
		{"SHR 0x01,1", 5, 0x01, 1, false, 0x00, true},
		{"SHR 0x80,1", 5, 0x80, 1, false, 0x40, false},
		{"SHR count=8", 5, 0x80, 8, false, 0x00, true},

		// SAR (op=7)
		{"SAR 0x80,1 sign extend", 7, 0x80, 1, false, 0xC0, false},
		{"SAR 0x80,8 all sign", 7, 0x80, 8, false, 0xFF, true},
		{"SAR 0x7F,8 positive", 7, 0x7F, 8, false, 0x00, false}, // positive → CF=val&0x80=false
		{"SAR 0x02,1", 7, 0x02, 1, false, 0x01, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestCPU()
			c.SetFlag(FlagCF, tt.cfIn)
			got := c.shiftRotate8(tt.op, tt.val, tt.count)
			if got != tt.wantVal {
				t.Errorf("result = 0x%02X, want 0x%02X", got, tt.wantVal)
			}
			if tt.count > 0 {
				if cf := c.GetFlag(FlagCF); cf != tt.wantCF {
					t.Errorf("CF = %v, want %v", cf, tt.wantCF)
				}
			}
		})
	}
}

func TestShiftRotate16(t *testing.T) {
	tests := []struct {
		name    string
		op      int
		val     uint16
		count   byte
		cfIn    bool
		wantVal uint16
		wantCF  bool
	}{
		// SHL 16-bit
		{"SHL16 0x8000,1", 4, 0x8000, 1, false, 0x0000, true},
		{"SHL16 count=16", 4, 0x0001, 16, false, 0x0000, true},
		{"SHL16 count=17", 4, 0xFFFF, 17, false, 0x0000, false},

		// SHR 16-bit
		{"SHR16 0x0001,1", 5, 0x0001, 1, false, 0x0000, true},
		{"SHR16 count=16", 5, 0x8000, 16, false, 0x0000, true},

		// SAR 16-bit
		{"SAR16 0x8000,1", 7, 0x8000, 1, false, 0xC000, false},
		{"SAR16 0x8000,16", 7, 0x8000, 16, false, 0xFFFF, true},

		// ROL 16-bit
		{"ROL16 0x8000,1", 0, 0x8000, 1, false, 0x0001, true},

		// ROR 16-bit
		{"ROR16 0x0001,1", 1, 0x0001, 1, false, 0x8000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestCPU()
			c.SetFlag(FlagCF, tt.cfIn)
			got := c.shiftRotate16(tt.op, tt.val, tt.count)
			if got != tt.wantVal {
				t.Errorf("result = 0x%04X, want 0x%04X", got, tt.wantVal)
			}
			if tt.count > 0 {
				if cf := c.GetFlag(FlagCF); cf != tt.wantCF {
					t.Errorf("CF = %v, want %v", cf, tt.wantCF)
				}
			}
		})
	}
}

// TestSHL8_Instruction tests SHL via full instruction execution (opcode D0 /4).
func TestSHL8_Instruction(t *testing.T) {
	c, bus := newTestCPU()
	// SHL AL, 1 → opcode D0, modrm C0 (mod=3, reg=4, rm=0=AL)
	loadCode(bus, 0xD0, 0xE0)
	c.SetAL(0x80)
	c.Step()
	if c.AL() != 0x00 {
		t.Errorf("AL = 0x%02X, want 0x00", c.AL())
	}
	assertFlags(t, c, map[uint16]bool{FlagCF: true, FlagZF: true})
}
