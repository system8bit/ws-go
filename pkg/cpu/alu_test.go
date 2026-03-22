package cpu

import "testing"

// TestDoALU8 tests the core ALU operations via direct function call.
func TestDoALU8(t *testing.T) {
	tests := []struct {
		name   string
		op     int // 0=ADD,1=OR,2=ADC,3=SBB,4=AND,5=SUB,6=XOR,7=CMP
		dst    byte
		src    byte
		cfIn   bool
		want   byte
		cf, zf, sf, of bool
	}{
		// ADD
		{"ADD 0+0", 0, 0, 0, false, 0, false, true, false, false},
		{"ADD 0xFF+1 carry", 0, 0xFF, 1, false, 0, true, true, false, false},
		{"ADD 0x7F+1 overflow", 0, 0x7F, 1, false, 0x80, false, false, true, true},

		// SUB
		{"SUB 5-3", 5, 5, 3, false, 2, false, false, false, false},
		{"SUB 0-1 borrow", 5, 0, 1, false, 0xFF, true, false, true, false},
		{"SUB 0x80-1 overflow", 5, 0x80, 1, false, 0x7F, false, false, false, true},

		// ADC with carry
		{"ADC 0xFF+0+CF", 2, 0xFF, 0, true, 0, true, true, false, false},
		{"ADC 0x7F+0+CF", 2, 0x7F, 0, true, 0x80, false, false, true, true},

		// SBB with borrow
		{"SBB 0-0-CF", 3, 0, 0, true, 0xFF, true, false, true, false},
		{"SBB 0x80-0-CF", 3, 0x80, 0, true, 0x7F, false, false, false, true},

		// OR
		{"OR 0x0F|0xF0", 1, 0x0F, 0xF0, false, 0xFF, false, false, true, false},

		// AND
		{"AND 0x0F&0xF0", 4, 0x0F, 0xF0, false, 0, false, true, false, false},

		// XOR
		{"XOR 0xFF^0xFF", 6, 0xFF, 0xFF, false, 0, false, true, false, false},

		// CMP (result not written, same flags as SUB)
		{"CMP 5-5", 7, 5, 5, false, 5, false, true, false, false}, // result = dst (unchanged)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestCPU()
			c.SetFlag(FlagCF, tt.cfIn)
			got := c.doALU8(tt.op, tt.dst, tt.src)
			if got != tt.want {
				t.Errorf("result = 0x%02X, want 0x%02X", got, tt.want)
			}
			assertFlags(t, c, map[uint16]bool{
				FlagCF: tt.cf, FlagZF: tt.zf, FlagSF: tt.sf, FlagOF: tt.of,
			})
		})
	}
}

// TestADD_Instruction tests ADD AL, imm8 via full Step() execution.
func TestADD_AL_Imm8(t *testing.T) {
	c, bus := newTestCPU()
	// ADD AL, 0x42 → opcode 04, imm8=0x42
	loadCode(bus, 0x04, 0x42)
	c.SetAL(0x10)
	cycles := c.Step()
	if c.AL() != 0x52 {
		t.Errorf("AL = 0x%02X, want 0x52", c.AL())
	}
	if cycles < 1 {
		t.Errorf("cycles = %d, want >= 1", cycles)
	}
}

// TestSUB_Instruction tests SUB AX, imm16 via Step().
func TestSUB_AX_Imm16(t *testing.T) {
	c, bus := newTestCPU()
	// SUB AX, 0x0001 → opcode 2D, imm16=0x0001
	loadCode(bus, 0x2D, 0x01, 0x00)
	c.AX = 0x0001
	c.Step()
	if c.AX != 0x0000 {
		t.Errorf("AX = 0x%04X, want 0x0000", c.AX)
	}
	assertFlags(t, c, map[uint16]bool{FlagZF: true, FlagCF: false})
}

// TestCMP_DoesNotWriteback verifies CMP sets flags but doesn't change dst.
func TestCMP_DoesNotWriteback(t *testing.T) {
	c, bus := newTestCPU()
	// CMP AL, 0x10 → opcode 3C, imm8=0x10
	loadCode(bus, 0x3C, 0x10)
	c.SetAL(0x10)
	c.Step()
	if c.AL() != 0x10 {
		t.Errorf("AL changed to 0x%02X, should remain 0x10", c.AL())
	}
	assertFlags(t, c, map[uint16]bool{FlagZF: true, FlagCF: false})
}
