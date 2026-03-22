package cpu

import "testing"

func TestNEG8Flags(t *testing.T) {
	tests := []struct {
		name   string
		val    byte
		wantCF bool
		wantOF bool
		wantAF bool
		wantZF bool
	}{
		{"NEG 0", 0x00, false, false, false, true},
		{"NEG 1", 0x01, true, false, true, false},
		{"NEG 0x80", 0x80, true, true, false, false},  // -128 overflows
		{"NEG 0xFF", 0xFF, true, false, true, false},
		{"NEG 0x10", 0x10, true, false, false, false},  // AF: low nibble=0
		{"NEG 0x0F", 0x0F, true, false, true, false},   // AF: low nibble!=0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, bus := newTestCPU()
			// NEG BL: F6 /3, modrm=0xDB (mod=3, reg=3, rm=3=BL)
			loadCode(bus, 0xF6, 0xDB)
			c.SetBL(tt.val)
			// Set all flags to opposite of expected to verify they're changed
			c.SetFlag(FlagCF, !tt.wantCF)
			c.SetFlag(FlagOF, !tt.wantOF)
			c.SetFlag(FlagAF, !tt.wantAF)
			c.Step()

			if c.GetFlag(FlagCF) != tt.wantCF {
				t.Errorf("CF = %v, want %v", c.GetFlag(FlagCF), tt.wantCF)
			}
			if c.GetFlag(FlagOF) != tt.wantOF {
				t.Errorf("OF = %v, want %v", c.GetFlag(FlagOF), tt.wantOF)
			}
			if c.GetFlag(FlagAF) != tt.wantAF {
				t.Errorf("AF = %v, want %v", c.GetFlag(FlagAF), tt.wantAF)
			}
			if c.GetFlag(FlagZF) != tt.wantZF {
				t.Errorf("ZF = %v, want %v", c.GetFlag(FlagZF), tt.wantZF)
			}
		})
	}
}

func TestShiftOFAllCounts(t *testing.T) {
	tests := []struct {
		name   string
		opcode byte // D2=shift r/m8 by CL
		modrm  byte
		val    byte
		count  byte
		wantOF bool
	}{
		// SHL: OF = CF XOR result_MSB
		{"SHL 0x40,1 → OF=1 (CF=0,MSB=1)", 0xD2, 0xE3, 0x40, 1, true},  // result=0x80
		{"SHL 0x80,1 → OF=1 (CF=1,MSB=0)", 0xD2, 0xE3, 0x80, 1, true},  // result=0x00
		{"SHL 0xC0,1 → OF=0 (CF=1,MSB=1)", 0xD2, 0xE3, 0xC0, 1, false}, // result=0x80
		{"SHL 0x01,8 → OF=1 (CF=1,MSB=0)", 0xD2, 0xE3, 0x01, 8, true},  // result=0x00

		// SHR: OF = result_MSB XOR result_(MSB-1)
		{"SHR 0x80,1 → OF=1", 0xD2, 0xEB, 0x80, 1, true},   // result=0x40 (bit6=1,bit7=0)
		{"SHR 0x40,1 → OF=0", 0xD2, 0xEB, 0x40, 1, false},  // result=0x20 (bit6=0,bit7=0)
		{"SHR 0x80,2 → OF=0", 0xD2, 0xEB, 0x80, 2, false},  // result=0x20

		// SAR: OF = result_MSB XOR result_(MSB-1)
		{"SAR 0x80,1 → OF=0", 0xD2, 0xFB, 0x80, 1, false},  // result=0xC0 (sign preserved)
		{"SAR 0x40,1 → OF=0", 0xD2, 0xFB, 0x40, 1, false},  // result=0x20 (bit7=0,bit6=0)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, bus := newTestCPU()
			loadCode(bus, tt.opcode, tt.modrm)
			c.SetBL(tt.val)
			c.SetCL(tt.count)
			c.Step()

			if c.GetFlag(FlagOF) != tt.wantOF {
				t.Errorf("OF = %v, want %v", c.GetFlag(FlagOF), tt.wantOF)
			}
		})
	}
}

func TestRotateOFCount0(t *testing.T) {
	tests := []struct {
		name   string
		modrm  byte
		val    byte
		cfIn   bool
		wantOF bool
	}{
		// ROL count=0: OF = CF XOR MSB
		{"ROL 0x00 CF=0 → OF=0", 0xC3, 0x00, false, false},
		{"ROL 0x80 CF=0 → OF=1", 0xC3, 0x80, false, true},
		{"ROL 0x00 CF=1 → OF=1", 0xC3, 0x00, true, true},
		{"ROL 0x80 CF=1 → OF=0", 0xC3, 0x80, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, bus := newTestCPU()
			// ROL BL, CL with CL=0: D2 C3 (mod=3, reg=0=ROL, rm=3=BL)
			loadCode(bus, 0xD2, tt.modrm)
			c.SetBL(tt.val)
			c.SetCL(0) // count = 0
			c.SetFlag(FlagCF, tt.cfIn)
			c.Step()

			if c.GetFlag(FlagOF) != tt.wantOF {
				t.Errorf("OF = %v, want %v", c.GetFlag(FlagOF), tt.wantOF)
			}
		})
	}
}

func TestSHLCount0SetsFlags(t *testing.T) {
	c, bus := newTestCPU()
	// SHL BL, CL with CL=0: D2 E3 (mod=3, reg=4=SHL, rm=3=BL)
	loadCode(bus, 0xD2, 0xE3)
	c.SetBL(0x00) // value = 0
	c.SetCL(0)    // count = 0
	c.SetFlag(FlagCF, true) // should be PRESERVED
	c.SetFlag(FlagZF, false)
	c.SetFlag(FlagPF, false)
	c.Step()

	// SHL count=0: SZP from value, CF preserved, AF cleared
	if !c.GetFlag(FlagCF) {
		t.Error("CF should be preserved (true) for SHL count=0")
	}
	if !c.GetFlag(FlagZF) {
		t.Error("ZF should be set for value 0")
	}
	if !c.GetFlag(FlagPF) {
		t.Error("PF should be set for value 0 (even parity)")
	}
	if c.GetFlag(FlagAF) {
		t.Error("AF should be cleared for SHL count=0")
	}
}

func TestDAABoundary(t *testing.T) {
	tests := []struct {
		name   string
		al     byte
		cfIn   bool
		afIn   bool
		wantAL byte
		wantCF bool
	}{
		{"DAA 0x00 no flags", 0x00, false, false, 0x00, false},
		{"DAA 0x09 no flags", 0x09, false, false, 0x09, false},
		{"DAA 0x0A → +6", 0x0A, false, false, 0x10, false},
		{"DAA 0x9A → +66", 0x9A, false, false, 0x00, true},
		{"DAA 0x99 → 0x99", 0x99, false, false, 0x99, false},
		{"DAA 0x00 CF=1 → +60", 0x00, true, false, 0x60, true},
		{"DAA 0x00 AF=1 → +6", 0x00, false, true, 0x06, false},
		{"DAA 0x7A → 0x80", 0x7A, false, false, 0x80, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, bus := newTestCPU()
			loadCode(bus, 0x27) // DAA
			c.SetAL(tt.al)
			c.SetFlag(FlagCF, tt.cfIn)
			c.SetFlag(FlagAF, tt.afIn)
			c.Step()

			if c.AL() != tt.wantAL {
				t.Errorf("AL = 0x%02X, want 0x%02X", c.AL(), tt.wantAL)
			}
			if c.GetFlag(FlagCF) != tt.wantCF {
				t.Errorf("CF = %v, want %v", c.GetFlag(FlagCF), tt.wantCF)
			}
		})
	}
}

func TestDASBoundary(t *testing.T) {
	tests := []struct {
		name   string
		al     byte
		cfIn   bool
		afIn   bool
		wantAL byte
		wantCF bool
	}{
		{"DAS 0x00 no flags", 0x00, false, false, 0x00, false},
		{"DAS 0x09 no flags", 0x09, false, false, 0x09, false},
		{"DAS 0x0A → -6", 0x0A, false, false, 0x04, false},
		{"DAS 0x9A → -66", 0x9A, false, false, 0x34, true},
		{"DAS 0x00 CF=1 → -60", 0x00, true, false, 0xA0, true},
		{"DAS 0x00 AF=1 → -6", 0x00, false, true, 0xFA, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, bus := newTestCPU()
			loadCode(bus, 0x2F) // DAS
			c.SetAL(tt.al)
			c.SetFlag(FlagCF, tt.cfIn)
			c.SetFlag(FlagAF, tt.afIn)
			c.Step()

			if c.AL() != tt.wantAL {
				t.Errorf("AL = 0x%02X, want 0x%02X", c.AL(), tt.wantAL)
			}
			if c.GetFlag(FlagCF) != tt.wantCF {
				t.Errorf("CF = %v, want %v", c.GetFlag(FlagCF), tt.wantCF)
			}
		})
	}
}
