package cpu

// Flag bit positions in the Flags register (x86-compatible).
const (
	FlagCF uint16 = 1 << 0  // Carry Flag
	FlagPF uint16 = 1 << 2  // Parity Flag
	FlagAF uint16 = 1 << 4  // Auxiliary Carry Flag
	FlagZF uint16 = 1 << 6  // Zero Flag
	FlagSF uint16 = 1 << 7  // Sign Flag
	FlagTF uint16 = 1 << 8  // Trap Flag
	FlagIF uint16 = 1 << 9  // Interrupt Enable Flag
	FlagDF uint16 = 1 << 10 // Direction Flag
	FlagOF uint16 = 1 << 11 // Overflow Flag
)

// GetFlag returns true if the specified flag bit is set.
func (c *CPU) GetFlag(flag uint16) bool {
	return c.Flags&flag != 0
}

// SetFlag sets or clears the specified flag bit.
func (c *CPU) SetFlag(flag uint16, v bool) {
	if v {
		c.Flags |= flag
	} else {
		c.Flags &^= flag
	}
}

// SetFlagsArith8 sets CF, ZF, SF, PF, AF, OF for an 8-bit arithmetic operation.
// result is the full (possibly >8-bit) result, op1 and op2 are the original operands.
// isSub should be true for SUB/SBB/CMP, false for ADD/ADC.
func (c *CPU) SetFlagsArith8(result uint16, op1, op2 byte, isSub bool) {
	res8 := byte(result)

	// CF: carry/borrow out of bit 7
	c.SetFlag(FlagCF, result > 0xFF)

	// ZF: result is zero
	c.SetFlag(FlagZF, res8 == 0)

	// SF: sign of result
	c.SetFlag(FlagSF, res8&0x80 != 0)

	// PF: parity of low byte
	c.SetFlag(FlagPF, parity(res8))

	// AF: carry/borrow from bit 3 to bit 4
	if isSub {
		c.SetFlag(FlagAF, (op1&0x0F) < (op2&0x0F))
	} else {
		c.SetFlag(FlagAF, (op1&0x0F)+(op2&0x0F) > 0x0F)
	}

	// OF: signed overflow
	if isSub {
		c.SetFlag(FlagOF, (op1^op2)&0x80 != 0 && (op1^res8)&0x80 != 0)
	} else {
		c.SetFlag(FlagOF, (^(op1 ^ op2))&(op1^res8)&0x80 != 0)
	}
}

// SetFlagsArith16 sets CF, ZF, SF, PF, AF, OF for a 16-bit arithmetic operation.
func (c *CPU) SetFlagsArith16(result uint32, op1, op2 uint16, isSub bool) {
	res16 := uint16(result)

	// CF: carry/borrow out of bit 15
	c.SetFlag(FlagCF, result > 0xFFFF)

	// ZF
	c.SetFlag(FlagZF, res16 == 0)

	// SF
	c.SetFlag(FlagSF, res16&0x8000 != 0)

	// PF: parity of low byte only
	c.SetFlag(FlagPF, parity(byte(res16)))

	// AF
	if isSub {
		c.SetFlag(FlagAF, (op1&0x0F) < (op2&0x0F))
	} else {
		c.SetFlag(FlagAF, uint16(op1&0x0F)+uint16(op2&0x0F) > 0x0F)
	}

	// OF
	if isSub {
		c.SetFlag(FlagOF, (op1^op2)&0x8000 != 0 && (op1^res16)&0x8000 != 0)
	} else {
		c.SetFlag(FlagOF, (^(op1 ^ op2))&(op1^res16)&0x8000 != 0)
	}
}

// SetFlagsLogic8 sets ZF, SF, PF and clears CF, OF for an 8-bit logical operation.
func (c *CPU) SetFlagsLogic8(result byte) {
	c.SetFlag(FlagCF, false)
	c.SetFlag(FlagOF, false)
	c.SetFlag(FlagZF, result == 0)
	c.SetFlag(FlagSF, result&0x80 != 0)
	c.SetFlag(FlagPF, parity(result))
	c.SetFlag(FlagAF, false)
}

// SetFlagsLogic16 sets ZF, SF, PF and clears CF, OF for a 16-bit logical operation.
func (c *CPU) SetFlagsLogic16(result uint16) {
	c.SetFlag(FlagCF, false)
	c.SetFlag(FlagOF, false)
	c.SetFlag(FlagZF, result == 0)
	c.SetFlag(FlagSF, result&0x8000 != 0)
	c.SetFlag(FlagPF, parity(byte(result)))
	c.SetFlag(FlagAF, false)
}

// SetFlagsSZP8 sets only ZF, SF, PF for an 8-bit result (no CF/OF/AF change).
// Used by shifts, NEG, and other ops where CF is handled separately.
func (c *CPU) SetFlagsSZP8(result byte) {
	c.SetFlag(FlagZF, result == 0)
	c.SetFlag(FlagSF, result&0x80 != 0)
	c.SetFlag(FlagPF, parity(result))
}

// SetFlagsSZP16 sets only ZF, SF, PF for a 16-bit result (no CF/OF/AF change).
func (c *CPU) SetFlagsSZP16(result uint16) {
	c.SetFlag(FlagZF, result == 0)
	c.SetFlag(FlagSF, result&0x8000 != 0)
	c.SetFlag(FlagPF, parity(byte(result)))
}

// parity returns true if the byte has an even number of set bits.
func parity(v byte) bool {
	v ^= v >> 4
	v ^= v >> 2
	v ^= v >> 1
	return v&1 == 0
}
