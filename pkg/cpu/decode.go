package cpu

// fetchByte reads a byte at CS:IP and increments IP.
func (c *CPU) fetchByte() byte {
	v := c.Bus.Read8(c.CS, c.IP)
	c.IP++
	return v
}

// fetchWord reads a 16-bit word at CS:IP and increments IP by 2.
func (c *CPU) fetchWord() uint16 {
	v := c.Bus.Read16(c.CS, c.IP)
	c.IP += 2
	return v
}

// decodeModRM splits a ModR/M byte into its mod, reg, and rm fields.
func (c *CPU) decodeModRM(modrm byte) (mod, reg, rm int) {
	mod = int((modrm >> 6) & 0x03)
	reg = int((modrm >> 3) & 0x07)
	rm = int(modrm & 0x07)
	return
}

// getEffectiveSegment returns the segment to use for a memory access.
func (c *CPU) getEffectiveSegment(defaultSeg uint16) uint16 {
	if c.segOverride >= 0 {
		return c.getSegReg(c.segOverride)
	}
	return defaultSeg
}

// rmBaseAddr computes the base address for mod 0 (no displacement).
// Returns the base address and whether SS is the default segment.
func (c *CPU) rmBaseAddr(rm int) (addr uint16, useSS bool) {
	switch rm {
	case 0:
		return c.BX + c.SI, false
	case 1:
		return c.BX + c.DI, false
	case 2:
		return c.BP + c.SI, true
	case 3:
		return c.BP + c.DI, true
	case 4:
		return c.SI, false
	case 5:
		return c.DI, false
	case 6:
		return c.BP, true // mod=0 uses direct address (caller handles), mod=1/2 uses BP
	case 7:
		return c.BX, false
	}
	return 0, false
}

// leaMod3 computes the effective address for LEA with mod=3 (V30MZ undocumented).
// The base register from the standard addressing mode is added to the register value
// selected by rm in mod=3 mode.
func (c *CPU) leaMod3(rm int) uint16 {
	// Base register for each rm (from standard EA calculation)
	var base uint16
	switch rm {
	case 0:
		base = c.BX // [BX+SI] → BX
	case 1:
		base = c.BX // [BX+DI] → BX
	case 2:
		base = c.BP // [BP+SI] → BP
	case 3:
		base = c.BP // [BP+DI] → BP
	case 4:
		base = c.SI // [SI] → SI
	case 5:
		base = c.DI // [DI] → DI
	case 6:
		base = c.BP // [BP+disp] → BP
	case 7:
		base = c.BX // [BX] → BX
	}
	return base + c.getReg16(rm)
}

// leaMod3Seg returns the default segment for leaMod3 addressing.
// rm 2,3,6 use SS (BP-based); others use DS.
func (c *CPU) leaMod3Seg(rm int) uint16 {
	switch rm {
	case 2, 3, 6:
		return c.getEffectiveSegment(c.SS)
	default:
		return c.getEffectiveSegment(c.DS)
	}
}

// getModRMAddress computes the effective address from ModR/M fields.
func (c *CPU) getModRMAddress(mod, rm int) (seg, offset uint16) {
	defaultSeg := c.DS
	var addr uint16

	if mod == 0 && rm == 6 {
		// Special case: 16-bit direct address
		addr = c.fetchWord()
	} else {
		base, useSS := c.rmBaseAddr(rm)
		if useSS || (rm == 6 && mod != 0) {
			defaultSeg = c.SS
		}
		addr = base

		switch mod {
		case 1:
			addr += uint16(int16(int8(c.fetchByte())))
		case 2:
			addr += c.fetchWord()
		}
	}

	return c.getEffectiveSegment(defaultSeg), addr
}

// resolveModRM computes and caches the effective address for a ModR/M operand.
func (c *CPU) resolveModRM(mod, rm int) {
	if mod == 3 {
		return
	}
	c.modrmSeg, c.modrmOff = c.getModRMAddress(mod, rm)
}

// readModRM8 reads an 8-bit operand from the resolved ModR/M location.
func (c *CPU) readModRM8(mod, rm int) byte {
	if mod == 3 {
		return c.getReg8(rm)
	}
	return c.Bus.Read8(c.modrmSeg, c.modrmOff)
}

// writeModRM8 writes an 8-bit value to the resolved ModR/M location.
func (c *CPU) writeModRM8(mod, rm int, val byte) {
	if mod == 3 {
		c.setReg8(rm, val)
		return
	}
	c.Bus.Write8(c.modrmSeg, c.modrmOff, val)
}

// readModRM16 reads a 16-bit operand from the resolved ModR/M location.
func (c *CPU) readModRM16(mod, rm int) uint16 {
	if mod == 3 {
		return c.getReg16(rm)
	}
	return c.Bus.Read16(c.modrmSeg, c.modrmOff)
}

// writeModRM16 writes a 16-bit value to the resolved ModR/M location.
func (c *CPU) writeModRM16(mod, rm int, val uint16) {
	if mod == 3 {
		c.setReg16(rm, val)
		return
	}
	c.Bus.Write16(c.modrmSeg, c.modrmOff, val)
}

// getReg8 returns the value of an 8-bit register by encoding index.
// 0=AL, 1=CL, 2=DL, 3=BL, 4=AH, 5=CH, 6=DH, 7=BH
func (c *CPU) getReg8(reg int) byte {
	switch reg {
	case 0:
		return c.AL()
	case 1:
		return c.CL()
	case 2:
		return c.DL()
	case 3:
		return c.BL()
	case 4:
		return c.AH()
	case 5:
		return c.CH()
	case 6:
		return c.DH()
	case 7:
		return c.BH()
	}
	return 0
}

// setReg8 sets an 8-bit register by encoding index.
func (c *CPU) setReg8(reg int, val byte) {
	switch reg {
	case 0:
		c.SetAL(val)
	case 1:
		c.SetCL(val)
	case 2:
		c.SetDL(val)
	case 3:
		c.SetBL(val)
	case 4:
		c.SetAH(val)
	case 5:
		c.SetCH(val)
	case 6:
		c.SetDH(val)
	case 7:
		c.SetBH(val)
	}
}

// getReg16 returns the value of a 16-bit register by encoding index.
// 0=AX, 1=CX, 2=DX, 3=BX, 4=SP, 5=BP, 6=SI, 7=DI
func (c *CPU) getReg16(reg int) uint16 {
	regs := [8]*uint16{&c.AX, &c.CX, &c.DX, &c.BX, &c.SP, &c.BP, &c.SI, &c.DI}
	return *regs[reg&7]
}

// setReg16 sets a 16-bit register by encoding index.
func (c *CPU) setReg16(reg int, val uint16) {
	regs := [8]*uint16{&c.AX, &c.CX, &c.DX, &c.BX, &c.SP, &c.BP, &c.SI, &c.DI}
	*regs[reg&7] = val
}

// getSegReg returns a segment register by index. 0=ES, 1=CS, 2=SS, 3=DS
func (c *CPU) getSegReg(reg int) uint16 {
	segs := [4]*uint16{&c.ES, &c.CS, &c.SS, &c.DS}
	return *segs[reg&3]
}

// setSegReg sets a segment register by index.
func (c *CPU) setSegReg(reg int, val uint16) {
	segs := [4]*uint16{&c.ES, &c.CS, &c.SS, &c.DS}
	*segs[reg&3] = val
}

// push16 pushes a 16-bit value onto the stack.
func (c *CPU) push16(val uint16) {
	c.SP -= 2
	c.Bus.Write16(c.SS, c.SP, val)
}

// pop16 pops a 16-bit value from the stack.
func (c *CPU) pop16() uint16 {
	val := c.Bus.Read16(c.SS, c.SP)
	c.SP += 2
	return val
}
