package cpu

import "testing"

func TestParity(t *testing.T) {
	tests := []struct {
		val  byte
		want bool
	}{
		{0x00, true},  // 0 bits set (even)
		{0x01, false}, // 1 bit set (odd)
		{0x03, true},  // 2 bits set (even)
		{0xFF, true},  // 8 bits set (even)
		{0x80, false}, // 1 bit set (odd)
		{0x55, true},  // 4 bits set (even)
		{0x0F, true},  // 4 bits set (even)
	}
	for _, tt := range tests {
		if got := parity(tt.val); got != tt.want {
			t.Errorf("parity(0x%02X) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestSetFlagsArith8(t *testing.T) {
	tests := []struct {
		name   string
		result uint16
		op1    byte
		op2    byte
		isSub  bool
		cf, zf, sf, of bool
	}{
		{"ADD 0x7F+0x01=0x80 overflow", 0x80, 0x7F, 0x01, false, false, false, true, true},
		{"ADD 0xFF+0x01=0x100 carry", 0x100, 0xFF, 0x01, false, true, true, false, false},
		{"SUB 0x00-0x01 borrow", 0x1FF, 0x00, 0x01, true, true, false, true, false},
		{"SUB 0x80-0x01=0x7F overflow", 0x7F, 0x80, 0x01, true, false, false, false, true},
		{"ADD 0+0=0 zero", 0x00, 0x00, 0x00, false, false, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestCPU()
			c.SetFlagsArith8(tt.result, tt.op1, tt.op2, tt.isSub)
			assertFlags(t, c, map[uint16]bool{
				FlagCF: tt.cf, FlagZF: tt.zf, FlagSF: tt.sf, FlagOF: tt.of,
			})
		})
	}
}

func TestSetFlagsLogic8(t *testing.T) {
	c, _ := newTestCPU()
	c.SetFlag(FlagCF, true)
	c.SetFlag(FlagOF, true)
	c.SetFlagsLogic8(0x00)
	assertFlags(t, c, map[uint16]bool{
		FlagCF: false, FlagOF: false, FlagZF: true, FlagSF: false,
	})

	c.SetFlagsLogic8(0x80)
	assertFlags(t, c, map[uint16]bool{
		FlagZF: false, FlagSF: true,
	})
}

func TestSetFlagsSZP8(t *testing.T) {
	c, _ := newTestCPU()
	c.SetFlag(FlagCF, true) // should NOT be modified
	c.SetFlagsSZP8(0)
	if !c.GetFlag(FlagZF) {
		t.Error("ZF should be set for 0")
	}
	if !c.GetFlag(FlagCF) {
		t.Error("CF should NOT be modified by SetFlagsSZP8")
	}
}
