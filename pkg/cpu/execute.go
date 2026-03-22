package cpu

import "fmt"

// executeInstruction fetches and executes one instruction, handling prefixes.
func (c *CPU) executeInstruction() {
	// Save IP before prefix loop for REP interrupt interleave rewind.
	c.instrStartIP = c.IP

	// Fetch opcode, handling prefixes in a loop
	var opcode byte
	for {
		opcode = c.fetchByte()
		switch opcode {
		case 0x26: // ES: segment override
			c.segOverride = 0
			continue
		case 0x2E: // CS: segment override
			c.segOverride = 1
			continue
		case 0x36: // SS: segment override
			c.segOverride = 2
			continue
		case 0x3E: // DS: segment override
			c.segOverride = 3
			continue
		case 0xF0: // LOCK prefix — NOP on V30MZ
			continue
		case 0xF2: // REPNE/REPNZ
			c.repPrefix = 2
			continue
		case 0xF3: // REP/REPE/REPZ
			c.repPrefix = 1
			continue
		}
		break
	}

	switch opcode {

	// ========== BCD/misc opcodes ==========
	case 0x27: // DAA
		c.daa()
	case 0x2F: // DAS
		c.das()
	case 0x37: // AAA
		c.aaa()
	case 0x3F: // AAS
		c.aas()
	case 0xD4: // AAM — Mednafen ignores imm8, always divides by 10
		c.fetchByte() // consume and discard immediate
		if c.AL() == 0 {
			c.Interrupt(0)
		} else {
			c.SetAH(c.AL() / 10)
			c.SetAL(c.AL() % 10)
			c.SetFlagsSZP16(c.AX) // Mednafen: SetSZPF_Word(AW)
		}
		c.Cycles += 5
	case 0xD5: // AAD — Mednafen ignores imm8, always multiplies by 10
		c.fetchByte() // consume and discard immediate
		c.SetAL(c.AH()*10 + c.AL())
		c.SetAH(0)
		c.SetFlagsSZP8(c.AL()) // Mednafen: SetSZPF_Byte(AL)
		c.Cycles += 5

	// ========== Misc undefined / V30MZ specific ==========
	case 0xF1: // V30MZ: undefined / INT1 — treat as NOP
		c.Cycles += 1
	case 0x65: // GS: prefix (386+ — not on V30MZ) — treat as NOP
		c.Cycles += 1
	case 0x64: // FS: prefix (386+ — not on V30MZ) — treat as NOP
		c.Cycles += 1
	case 0xD6: // SALC (undocumented: set AL from carry)
		if c.GetFlag(FlagCF) {
			c.SetAL(0xFF)
		} else {
			c.SetAL(0x00)
		}
		c.Cycles += 1

	// ========== ALU operations (0x00-0x3F) ==========
	// Pattern: each ALU op has 6 encodings spaced 8 apart
	// ADD (0x00-0x05), OR (0x08-0x0D), ADC (0x10-0x15), SBB (0x18-0x1D)
	// AND (0x20-0x25), SUB (0x28-0x2D), XOR (0x30-0x35), CMP (0x38-0x3D)

	case 0x00: // ADD r/m8, r8
		c.aluRM8R8(opcode)
	case 0x01: // ADD r/m16, r16
		c.aluRM16R16(opcode)
	case 0x02: // ADD r8, r/m8
		c.aluR8RM8(opcode)
	case 0x03: // ADD r16, r/m16
		c.aluR16RM16(opcode)
	case 0x04: // ADD AL, imm8
		c.aluALImm8(opcode)
	case 0x05: // ADD AX, imm16
		c.aluAXImm16(opcode)

	case 0x08: // OR r/m8, r8
		c.aluRM8R8(opcode)
	case 0x09:
		c.aluRM16R16(opcode)
	case 0x0A:
		c.aluR8RM8(opcode)
	case 0x0B:
		c.aluR16RM16(opcode)
	case 0x0C:
		c.aluALImm8(opcode)
	case 0x0D:
		c.aluAXImm16(opcode)

	case 0x10: // ADC r/m8, r8
		c.aluRM8R8(opcode)
	case 0x11:
		c.aluRM16R16(opcode)
	case 0x12:
		c.aluR8RM8(opcode)
	case 0x13:
		c.aluR16RM16(opcode)
	case 0x14:
		c.aluALImm8(opcode)
	case 0x15:
		c.aluAXImm16(opcode)

	case 0x18: // SBB r/m8, r8
		c.aluRM8R8(opcode)
	case 0x19:
		c.aluRM16R16(opcode)
	case 0x1A:
		c.aluR8RM8(opcode)
	case 0x1B:
		c.aluR16RM16(opcode)
	case 0x1C:
		c.aluALImm8(opcode)
	case 0x1D:
		c.aluAXImm16(opcode)

	case 0x20: // AND r/m8, r8
		c.aluRM8R8(opcode)
	case 0x21:
		c.aluRM16R16(opcode)
	case 0x22:
		c.aluR8RM8(opcode)
	case 0x23:
		c.aluR16RM16(opcode)
	case 0x24:
		c.aluALImm8(opcode)
	case 0x25:
		c.aluAXImm16(opcode)

	case 0x28: // SUB r/m8, r8
		c.aluRM8R8(opcode)
	case 0x29:
		c.aluRM16R16(opcode)
	case 0x2A:
		c.aluR8RM8(opcode)
	case 0x2B:
		c.aluR16RM16(opcode)
	case 0x2C:
		c.aluALImm8(opcode)
	case 0x2D:
		c.aluAXImm16(opcode)

	case 0x30: // XOR r/m8, r8
		c.aluRM8R8(opcode)
	case 0x31:
		c.aluRM16R16(opcode)
	case 0x32:
		c.aluR8RM8(opcode)
	case 0x33:
		c.aluR16RM16(opcode)
	case 0x34:
		c.aluALImm8(opcode)
	case 0x35:
		c.aluAXImm16(opcode)

	case 0x38: // CMP r/m8, r8
		c.aluRM8R8(opcode)
	case 0x39:
		c.aluRM16R16(opcode)
	case 0x3A:
		c.aluR8RM8(opcode)
	case 0x3B:
		c.aluR16RM16(opcode)
	case 0x3C:
		c.aluALImm8(opcode)
	case 0x3D:
		c.aluAXImm16(opcode)

	// ========== 0x0F two-byte opcodes ==========
	case 0x0F:
		c.executeTwoByteOpcode()

	// ========== PUSH/POP segment registers ==========
	case 0x06: // PUSH ES
		c.push16(c.ES)
		c.Cycles += 1
	case 0x07: // POP ES
		c.ES = c.pop16()
		c.Cycles += 1
	case 0x0E: // PUSH CS
		c.push16(c.CS)
		c.Cycles += 1
	case 0x16: // PUSH SS
		c.push16(c.SS)
		c.Cycles += 1
	case 0x17: // POP SS
		c.SS = c.pop16()
		c.Cycles += 1
	case 0x1E: // PUSH DS
		c.push16(c.DS)
		c.Cycles += 1
	case 0x1F: // POP DS
		c.DS = c.pop16()
		c.Cycles += 1

	// ========== INC/DEC reg16 (0x40-0x4F) ==========
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47: // INC reg16
		reg := int(opcode - 0x40)
		val := c.getReg16(reg)
		result := uint32(val) + 1
		// INC does not affect CF
		saveCF := c.GetFlag(FlagCF)
		c.SetFlagsArith16(result, val, 1, false)
		c.SetFlag(FlagCF, saveCF)
		c.setReg16(reg, uint16(result))
		c.Cycles += 1

	case 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F: // DEC reg16
		reg := int(opcode - 0x48)
		val := c.getReg16(reg)
		result := uint32(val) - 1
		saveCF := c.GetFlag(FlagCF)
		c.SetFlagsArith16(result, val, 1, true)
		c.SetFlag(FlagCF, saveCF)
		c.setReg16(reg, uint16(result))
		c.Cycles += 1

	// ========== PUSH/POP reg16 (0x50-0x5F) ==========
	case 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57: // PUSH reg16
		reg := int(opcode - 0x50)
		if reg == 4 { // PUSH SP: V30/8086 pushes the decremented value
			c.SP -= 2
			c.Bus.Write16(c.SS, c.SP, c.SP)
		} else {
			c.push16(c.getReg16(reg))
		}
		c.Cycles += 1

	case 0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F: // POP reg16
		reg := int(opcode - 0x58)
		c.setReg16(reg, c.pop16())
		c.Cycles += 1

	// ========== PUSHA/POPA (186+) ==========
	case 0x60: // PUSHA
		temp := c.SP
		c.push16(c.AX)
		c.push16(c.CX)
		c.push16(c.DX)
		c.push16(c.BX)
		c.push16(temp)
		c.push16(c.BP)
		c.push16(c.SI)
		c.push16(c.DI)
		c.Cycles += 8

	case 0x61: // POPA
		c.DI = c.pop16()
		c.SI = c.pop16()
		c.BP = c.pop16()
		c.pop16() // discard SP
		c.BX = c.pop16()
		c.DX = c.pop16()
		c.CX = c.pop16()
		c.AX = c.pop16()
		c.Cycles += 8

	// ========== BOUND (186+) ==========
	case 0x62: // BOUND r16, m16&16
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		low := c.readModRM16(mod, rm)
		// GetnextRMWord: read the word at [effective address + 2]
		high := c.Bus.Read16(c.modrmSeg, c.modrmOff+2)
		val := c.getReg16(reg)
		c.Cycles += 13
		if int16(val) < int16(low) || int16(val) > int16(high) {
			c.Interrupt(5)
		}

	// ========== PUSH imm (186+) ==========
	case 0x68: // PUSH imm16
		imm := c.fetchWord()
		c.push16(imm)
		c.Cycles += 1

	case 0x6A: // PUSH imm8 (sign-extended to 16-bit)
		imm := int8(c.fetchByte())
		c.push16(uint16(int16(imm)))
		c.Cycles += 1

	// ========== IMUL imm (186+) ==========
	case 0x69: // IMUL r16, r/m16, imm16
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		src := c.readModRM16(mod, rm)
		imm := c.fetchWord()
		result := int32(int16(src)) * int32(int16(imm))
		c.setReg16(reg, uint16(result))
		overflow := result > 32767 || result < -32768
		c.SetFlag(FlagCF, overflow)
		c.SetFlag(FlagOF, overflow)
		c.Cycles += 4

	case 0x6B: // IMUL r16, r/m16, imm8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		src := c.readModRM16(mod, rm)
		imm := int8(c.fetchByte())
		result := int32(int16(src)) * int32(imm)
		c.setReg16(reg, uint16(result))
		overflow := result > 32767 || result < -32768
		c.SetFlag(FlagCF, overflow)
		c.SetFlag(FlagOF, overflow)
		c.Cycles += 4

	// ========== TEST r/m, r (0x84-0x85) ==========
	case 0x84: // TEST r/m8, r8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.SetFlagsLogic8(c.readModRM8(mod, rm) & c.getReg8(reg))
		c.Cycles += 1

	case 0x85: // TEST r/m16, r16
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.SetFlagsLogic16(c.readModRM16(mod, rm) & c.getReg16(reg))
		c.Cycles += 1

	// ========== I/O string operations (186+) ==========
	case 0x6C: // INSB — read byte from port DX to ES:DI
		c.execIOStringOp(opcode)
	case 0x6D: // INSW — read word from port DX to ES:DI
		c.execIOStringOp(opcode)
	case 0x6E: // OUTSB — write byte from DS:SI to port DX
		c.execIOStringOp(opcode)
	case 0x6F: // OUTSW — write word from DS:SI to port DX
		c.execIOStringOp(opcode)

	// ========== Conditional jumps (0x70-0x7F) ==========
	case 0x70: // JO
		c.jccShort(c.GetFlag(FlagOF))
	case 0x71: // JNO
		c.jccShort(!c.GetFlag(FlagOF))
	case 0x72: // JB/JNAE/JC
		c.jccShort(c.GetFlag(FlagCF))
	case 0x73: // JNB/JAE/JNC
		c.jccShort(!c.GetFlag(FlagCF))
	case 0x74: // JE/JZ
		c.jccShort(c.GetFlag(FlagZF))
	case 0x75: // JNE/JNZ
		c.jccShort(!c.GetFlag(FlagZF))
	case 0x76: // JBE/JNA
		c.jccShort(c.GetFlag(FlagCF) || c.GetFlag(FlagZF))
	case 0x77: // JNBE/JA
		c.jccShort(!c.GetFlag(FlagCF) && !c.GetFlag(FlagZF))
	case 0x78: // JS
		c.jccShort(c.GetFlag(FlagSF))
	case 0x79: // JNS
		c.jccShort(!c.GetFlag(FlagSF))
	case 0x7A: // JP/JPE
		c.jccShort(c.GetFlag(FlagPF))
	case 0x7B: // JNP/JPO
		c.jccShort(!c.GetFlag(FlagPF))
	case 0x7C: // JL/JNGE
		c.jccShort(c.GetFlag(FlagSF) != c.GetFlag(FlagOF))
	case 0x7D: // JNL/JGE
		c.jccShort(c.GetFlag(FlagSF) == c.GetFlag(FlagOF))
	case 0x7E: // JLE/JNG
		c.jccShort(c.GetFlag(FlagZF) || (c.GetFlag(FlagSF) != c.GetFlag(FlagOF)))
	case 0x7F: // JNLE/JG
		c.jccShort(!c.GetFlag(FlagZF) && (c.GetFlag(FlagSF) == c.GetFlag(FlagOF)))

	// ========== Group 1: ALU r/m, imm (0x80-0x83) ==========
	case 0x80: // Group1 r/m8, imm8
		c.group1_8(false)
	case 0x81: // Group1 r/m16, imm16
		c.group1_16(false)
	case 0x82: // Group1 r/m8, imm8 (alias of 0x80)
		c.group1_8(false)
	case 0x83: // Group1 r/m16, imm8 (sign-extended)
		c.group1_16(true)

	// ========== XCHG (0x86-0x87) ==========
	case 0x86: // XCHG r/m8, r8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val1 := c.readModRM8(mod, rm)
		val2 := c.getReg8(reg)
		c.writeModRM8(mod, rm, val2)
		c.setReg8(reg, val1)
		c.Cycles += 3

	case 0x87: // XCHG r/m16, r16
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val1 := c.readModRM16(mod, rm)
		val2 := c.getReg16(reg)
		c.writeModRM16(mod, rm, val2)
		c.setReg16(reg, val1)
		c.Cycles += 3

	// ========== MOV (0x88-0x8B) ==========
	case 0x88: // MOV r/m8, r8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.writeModRM8(mod, rm, c.getReg8(reg))
		c.Cycles += 1

	case 0x89: // MOV r/m16, r16
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.writeModRM16(mod, rm, c.getReg16(reg))
		c.Cycles += 1

	case 0x8A: // MOV r8, r/m8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.setReg8(reg, c.readModRM8(mod, rm))
		c.Cycles += 1

	case 0x8B: // MOV r16, r/m16
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.setReg16(reg, c.readModRM16(mod, rm))
		c.Cycles += 1

	// ========== MOV Sreg (0x8C, 0x8E) ==========
	case 0x8C: // MOV r/m16, Sreg
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.writeModRM16(mod, rm, c.getSegReg(reg))
		c.Cycles += 1

	case 0x8D: // LEA r16, m — V30MZ: mod=3 computes EA from register values
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		if mod == 3 {
			// V30MZ undocumented: LEA with mod=3 computes EA as
			// base_for_rm + register_value(rm).
			c.setReg16(reg, c.leaMod3(rm))
		} else {
			_, off := c.getModRMAddress(mod, rm)
			c.setReg16(reg, off)
		}
		c.Cycles += 1

	case 0x8E: // MOV Sreg, r/m16
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.setSegReg(reg, c.readModRM16(mod, rm))
		c.Cycles += 1

	// ========== POP r/m16 ==========
	case 0x8F: // POP r/m16
		modrm := c.fetchByte()
		mod, _, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		c.writeModRM16(mod, rm, c.pop16())
		c.Cycles += 1

	// ========== NOP / XCHG AX, reg (0x90-0x97) ==========
	case 0x90: // NOP (XCHG AX, AX) — 3 cycles on V30MZ
		c.Cycles += 3

	case 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97: // XCHG AX, reg
		reg := int(opcode - 0x90)
		tmp := c.AX
		c.AX = c.getReg16(reg)
		c.setReg16(reg, tmp)
		c.Cycles += 1

	// ========== CBW, CWD ==========
	case 0x98: // CBW
		if c.AL()&0x80 != 0 {
			c.SetAH(0xFF)
		} else {
			c.SetAH(0x00)
		}
		c.Cycles += 1

	case 0x99: // CWD
		if c.AX&0x8000 != 0 {
			c.DX = 0xFFFF
		} else {
			c.DX = 0x0000
		}
		c.Cycles += 1

	// ========== LAHF / SAHF ==========
	case 0x9E: // SAHF — Store AH into Flags (low byte, mask 0xD5: CF,PF,AF,ZF,SF)
		c.Flags = (c.Flags & 0xFF00) | uint16(c.AH()&0xD5)
		c.Cycles += 1

	case 0x9F: // LAHF — Load Flags (low byte) into AH
		c.SetAH(byte(c.Flags))
		c.Cycles += 1

	// ========== CALL far ==========
	case 0x9A: // CALL far imm
		newIP := c.fetchWord()
		newCS := c.fetchWord()
		c.push16(c.CS)
		c.push16(c.IP)
		c.CS = newCS
		c.IP = newIP
		c.Cycles += 4

	// ========== PUSHF / POPF ==========
	case 0x9C: // PUSHF — bits 1,12-15 always set (8086/V30 hardwired)
		c.push16(c.Flags | 0xF002)
		c.Cycles += 2

	case 0x9D: // POPF — bits 3,5 hardwired to 0 on V30MZ
		c.Flags = c.pop16() &^ 0x0028
		c.InterruptEnable = c.GetFlag(FlagIF)
		c.Cycles += 1

	// ========== MOV AL/AX, moffs (0xA0-0xA3) ==========
	case 0xA0: // MOV AL, [moffs16]
		off := c.fetchWord()
		seg := c.getEffectiveSegment(c.DS)
		c.SetAL(c.Bus.Read8(seg, off))
		c.Cycles += 1

	case 0xA1: // MOV AX, [moffs16]
		off := c.fetchWord()
		seg := c.getEffectiveSegment(c.DS)
		c.AX = c.Bus.Read16(seg, off)
		c.Cycles += 1

	case 0xA2: // MOV [moffs16], AL
		off := c.fetchWord()
		seg := c.getEffectiveSegment(c.DS)
		c.Bus.Write8(seg, off, c.AL())
		c.Cycles += 1

	case 0xA3: // MOV [moffs16], AX
		off := c.fetchWord()
		seg := c.getEffectiveSegment(c.DS)
		c.Bus.Write16(seg, off, c.AX)
		c.Cycles += 1

	// ========== String operations ==========
	case 0xA4: // MOVSB
		c.execStringOp(opcode)
	case 0xA5: // MOVSW
		c.execStringOp(opcode)
	case 0xA6: // CMPSB
		c.execStringOp(opcode)
	case 0xA7: // CMPSW
		c.execStringOp(opcode)
	case 0xAA: // STOSB
		c.execStringOp(opcode)
	case 0xAB: // STOSW
		c.execStringOp(opcode)
	case 0xAC: // LODSB
		c.execStringOp(opcode)
	case 0xAD: // LODSW
		c.execStringOp(opcode)
	case 0xAE: // SCASB
		c.execStringOp(opcode)
	case 0xAF: // SCASW
		c.execStringOp(opcode)

	// ========== TEST AL/AX, imm ==========
	case 0xA8: // TEST AL, imm8
		imm := c.fetchByte()
		c.SetFlagsLogic8(c.AL() & imm)
		c.Cycles += 1

	case 0xA9: // TEST AX, imm16
		imm := c.fetchWord()
		c.SetFlagsLogic16(c.AX & imm)
		c.Cycles += 1

	// ========== MOV reg, imm (0xB0-0xBF) ==========
	case 0xB0, 0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7: // MOV reg8, imm8
		reg := int(opcode - 0xB0)
		c.setReg8(reg, c.fetchByte())
		c.Cycles += 1

	case 0xB8, 0xB9, 0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF: // MOV reg16, imm16
		reg := int(opcode - 0xB8)
		c.setReg16(reg, c.fetchWord())
		c.Cycles += 1

	// ========== Shifts by imm8 (186+) ==========
	case 0xC0: // Group2 r/m8, imm8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM8(mod, rm)
		count := c.fetchByte() & 0x1F
		result := c.shiftRotate8(reg, val, count)
		c.writeModRM8(mod, rm, result)
		c.Cycles += 2

	case 0xC1: // Group2 r/m16, imm8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM16(mod, rm)
		count := c.fetchByte() & 0x1F
		result := c.shiftRotate16(reg, val, count)
		c.writeModRM16(mod, rm, result)
		c.Cycles += 2

	// ========== RET near/far ==========
	case 0xC2: // RET near imm16
		imm := c.fetchWord()
		c.IP = c.pop16()
		c.SP += imm
		c.Cycles += 3

	case 0xC3: // RET near
		c.IP = c.pop16()
		c.Cycles += 3

	// ========== LES, LDS ==========
	case 0xC4: // LES r16, m16:16 — V30MZ: mod=3 uses leaMod3 address
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		if mod == 3 {
			off := c.leaMod3(rm)
			seg := c.leaMod3Seg(rm)
			c.setReg16(reg, c.Bus.Read16(seg, off))
			c.ES = c.Bus.Read16(seg, off+2)
		} else {
			seg, off := c.getModRMAddress(mod, rm)
			c.setReg16(reg, c.Bus.Read16(seg, off))
			c.ES = c.Bus.Read16(seg, off+2)
		}
		c.Cycles += 2

	case 0xC5: // LDS r16, m16:16 — V30MZ: mod=3 uses leaMod3 address
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		if mod == 3 {
			off := c.leaMod3(rm)
			seg := c.leaMod3Seg(rm)
			c.setReg16(reg, c.Bus.Read16(seg, off))
			c.DS = c.Bus.Read16(seg, off+2)
		} else {
			seg, off := c.getModRMAddress(mod, rm)
			c.setReg16(reg, c.Bus.Read16(seg, off))
			c.DS = c.Bus.Read16(seg, off+2)
		}
		c.Cycles += 2

	// ========== MOV r/m, imm ==========
	case 0xC6: // MOV r/m8, imm8
		modrm := c.fetchByte()
		mod, _, rm := c.decodeModRM(modrm)
		// Must resolve address (consuming displacement) BEFORE reading immediate
		if mod != 3 {
			seg, off := c.getModRMAddress(mod, rm)
			imm := c.fetchByte()
			c.Bus.Write8(seg, off, imm)
		} else {
			imm := c.fetchByte()
			c.setReg8(rm, imm)
		}
		c.Cycles += 1

	case 0xC7: // MOV r/m16, imm16
		modrm := c.fetchByte()
		mod, _, rm := c.decodeModRM(modrm)
		// Must resolve address (consuming displacement) BEFORE reading immediate
		if mod != 3 {
			seg, off := c.getModRMAddress(mod, rm)
			imm := c.fetchWord()
			c.Bus.Write16(seg, off, imm)
		} else {
			imm := c.fetchWord()
			c.setReg16(rm, imm)
		}
		c.Cycles += 1

	// ========== ENTER / LEAVE (186+) ==========
	case 0xC8: // ENTER imm16, imm8
		allocSize := c.fetchWord()
		nestingLevel := c.fetchByte() & 0x1F
		c.push16(c.BP)
		framePtr := c.SP
		if nestingLevel > 0 {
			for i := byte(1); i < nestingLevel; i++ {
				c.BP -= 2
				c.push16(c.Bus.Read16(c.SS, c.BP))
			}
			c.push16(framePtr)
		}
		c.BP = framePtr
		c.SP -= allocSize
		c.Cycles += 4

	case 0xC9: // LEAVE
		c.SP = c.BP
		c.BP = c.pop16()
		c.Cycles += 1

	// ========== RET far ==========
	case 0xCA: // RET far imm16
		imm := c.fetchWord()
		c.IP = c.pop16()
		c.CS = c.pop16()
		c.SP += imm
		c.Cycles += 4

	case 0xCB: // RET far
		c.IP = c.pop16()
		c.CS = c.pop16()
		c.Cycles += 4

	// ========== INT ==========
	case 0xCC: // INT 3 — Mednafen: CLK(10) + 32 from Interrupt()
		c.Interrupt(3)
		c.Cycles += 10

	case 0xCD: // INT imm8 — Mednafen: CLK(10) + 32 from Interrupt()
		vector := int(c.fetchByte())
		c.Interrupt(vector)
		c.Cycles += 10

	// ========== IRET ==========
	case 0xCF: // IRET — Mednafen: CLK(10), bits 3,5 hardwired to 0
		c.IP = c.pop16()
		c.CS = c.pop16()
		c.Flags = c.pop16() &^ 0x0028
		c.InterruptEnable = c.GetFlag(FlagIF)
		c.Cycles += 10

	// ========== Shifts/rotates by 1 or CL ==========
	case 0xD0: // Group2 r/m8, 1
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM8(mod, rm)
		result := c.shiftRotate8(reg, val, 1)
		c.writeModRM8(mod, rm, result)
		c.Cycles += 1

	case 0xD1: // Group2 r/m16, 1
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM16(mod, rm)
		result := c.shiftRotate16(reg, val, 1)
		c.writeModRM16(mod, rm, result)
		c.Cycles += 1

	case 0xD2: // Group2 r/m8, CL
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM8(mod, rm)
		count := c.CL() & 0x1F
		result := c.shiftRotate8(reg, val, count)
		c.writeModRM8(mod, rm, result)
		c.Cycles += 2

	case 0xD3: // Group2 r/m16, CL
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM16(mod, rm)
		count := c.CL() & 0x1F
		result := c.shiftRotate16(reg, val, count)
		c.writeModRM16(mod, rm, result)
		c.Cycles += 2

	// ========== XLAT ==========
	case 0xD7: // XLAT
		seg := c.getEffectiveSegment(c.DS)
		off := c.BX + uint16(c.AL())
		c.SetAL(c.Bus.Read8(seg, off))
		c.Cycles += 1

	// ========== FPU escape codes (D8-DF) — V30MZ has no FPU, skip ModRM ==========
	case 0xD8, 0xD9, 0xDA, 0xDB, 0xDC, 0xDD, 0xDE, 0xDF:
		modrm := c.fetchByte()
		mod := int((modrm >> 6) & 0x03)
		rm := int(modrm & 0x07)
		// Skip any displacement bytes in the ModRM encoding
		switch mod {
		case 0:
			if rm == 6 {
				c.IP += 2 // 16-bit direct address
			}
		case 1:
			c.IP += 1 // 8-bit displacement
		case 2:
			c.IP += 2 // 16-bit displacement
		}
		c.Cycles += 1

	// ========== LOOPNZ, LOOPZ, LOOP, JCXZ ==========
	case 0xE0: // LOOPNZ/LOOPNE
		disp := int16(int8(c.fetchByte()))
		c.CX--
		if c.CX != 0 && !c.GetFlag(FlagZF) {
			c.IP = uint16(int16(c.IP) + disp)
			c.Cycles += 3
		} else {
			c.Cycles += 2
		}

	case 0xE1: // LOOPZ/LOOPE
		disp := int16(int8(c.fetchByte()))
		c.CX--
		if c.CX != 0 && c.GetFlag(FlagZF) {
			c.IP = uint16(int16(c.IP) + disp)
			c.Cycles += 3
		} else {
			c.Cycles += 2
		}

	case 0xE2: // LOOP
		disp := int16(int8(c.fetchByte()))
		c.CX--
		if c.CX != 0 {
			c.IP = uint16(int16(c.IP) + disp)
			c.Cycles += 3
		} else {
			c.Cycles += 2
		}

	case 0xE3: // JCXZ
		disp := int16(int8(c.fetchByte()))
		if c.CX == 0 {
			c.IP = uint16(int16(c.IP) + disp)
			c.Cycles += 3
		} else {
			c.Cycles += 2
		}

	// ========== I/O ==========
	case 0xE4: // IN AL, imm8
		port := c.fetchByte()
		c.SetAL(c.Bus.IORead(port))
		c.Cycles += 2

	case 0xE5: // IN AX, imm8
		port := c.fetchByte()
		c.SetAL(c.Bus.IORead(port))
		c.SetAH(c.Bus.IORead(port + 1))
		c.Cycles += 2

	case 0xE6: // OUT imm8, AL
		port := c.fetchByte()
		c.Bus.IOWrite(port, c.AL())
		c.Cycles += 2

	case 0xE7: // OUT imm8, AX
		port := c.fetchByte()
		c.Bus.IOWrite(port, c.AL())
		c.Bus.IOWrite(port+1, c.AH())
		c.Cycles += 2

	// ========== CALL near ==========
	case 0xE8: // CALL near
		disp := c.fetchWord()
		c.push16(c.IP)
		c.IP = c.IP + disp
		c.Cycles += 3

	// ========== JMP ==========
	case 0xE9: // JMP near
		disp := c.fetchWord()
		c.IP = c.IP + disp
		c.Cycles += 2

	case 0xEA: // JMP far
		newIP := c.fetchWord()
		newCS := c.fetchWord()
		c.CS = newCS
		c.IP = newIP
		c.Cycles += 3

	case 0xEB: // JMP short
		disp := int8(c.fetchByte())
		c.IP = uint16(int16(c.IP) + int16(disp))
		c.Cycles += 2

	// ========== I/O via DX ==========
	case 0xEC: // IN AL, DX
		c.SetAL(c.Bus.IORead(byte(c.DX)))
		c.Cycles += 2

	case 0xED: // IN AX, DX
		c.SetAL(c.Bus.IORead(byte(c.DX)))
		c.SetAH(c.Bus.IORead(byte(c.DX) + 1))
		c.Cycles += 2

	case 0xEE: // OUT DX, AL
		c.Bus.IOWrite(byte(c.DX), c.AL())
		c.Cycles += 2

	case 0xEF: // OUT DX, AX
		c.Bus.IOWrite(byte(c.DX), c.AL())
		c.Bus.IOWrite(byte(c.DX)+1, c.AH())
		c.Cycles += 2

	// ========== HLT ==========
	case 0xF4: // HLT
		c.Halted = true
		c.Cycles += 1

	// ========== CMC ==========
	case 0xF5: // CMC
		c.SetFlag(FlagCF, !c.GetFlag(FlagCF))
		c.Cycles += 1

	// ========== Group 3 ==========
	case 0xF6: // Group3 r/m8
		c.group3_8()
	case 0xF7: // Group3 r/m16
		c.group3_16()

	// ========== Flag manipulation ==========
	case 0xF8: // CLC
		c.SetFlag(FlagCF, false)
		c.Cycles += 1
	case 0xF9: // STC
		c.SetFlag(FlagCF, true)
		c.Cycles += 1
	case 0xFA: // CLI — Mednafen: CLK(4)
		c.SetFlag(FlagIF, false)
		c.InterruptEnable = false
		c.Cycles += 4
	case 0xFB: // STI — Mednafen: CLK(4)
		c.SetFlag(FlagIF, true)
		c.InterruptEnable = true
		c.Cycles += 4
	case 0xFC: // CLD
		c.SetFlag(FlagDF, false)
		c.Cycles += 1
	case 0xFD: // STD
		c.SetFlag(FlagDF, true)
		c.Cycles += 1

	// ========== Group 4: INC/DEC r/m8 ==========
	case 0xFE:
		c.group4()

	// ========== Group 5 ==========
	case 0xFF:
		c.group5()

	default:
		fmt.Printf("WARNING: unimplemented opcode 0x%02X at %04X:%04X\n", opcode, c.CS, c.IP-1)
		c.Cycles += 1
	}
}

// ========== ALU helpers ==========

// aluOp returns the ALU operation index (0-7) from an opcode.
// ADD=0, OR=1, ADC=2, SBB=3, AND=4, SUB=5, XOR=6, CMP=7
func aluOpIndex(opcode byte) int {
	return int((opcode >> 3) & 0x07)
}

func (c *CPU) doALU8(op int, dst, src byte) byte {
	var result uint16
	var carry byte
	if c.GetFlag(FlagCF) {
		carry = 1
	}

	switch op {
	case 0: // ADD
		result = uint16(dst) + uint16(src)
		c.SetFlagsArith8(result, dst, src, false)
	case 1: // OR
		result = uint16(dst | src)
		c.SetFlagsLogic8(byte(result))
	case 2: // ADC
		result = uint16(dst) + uint16(src) + uint16(carry)
		res8 := byte(result)
		c.SetFlag(FlagCF, result > 0xFF)
		c.SetFlag(FlagZF, res8 == 0)
		c.SetFlag(FlagSF, res8&0x80 != 0)
		c.SetFlag(FlagPF, parity(res8))
		c.SetFlag(FlagAF, (dst^src^res8)&0x10 != 0)
		c.SetFlag(FlagOF, (^(dst^src))&(dst^res8)&0x80 != 0)
	case 3: // SBB
		result = uint16(dst) - uint16(src) - uint16(carry)
		res8 := byte(result)
		c.SetFlag(FlagCF, result > 0xFF)
		c.SetFlag(FlagZF, res8 == 0)
		c.SetFlag(FlagSF, res8&0x80 != 0)
		c.SetFlag(FlagPF, parity(res8))
		c.SetFlag(FlagAF, (dst^src^res8)&0x10 != 0)
		c.SetFlag(FlagOF, (dst^src)&(dst^res8)&0x80 != 0)
	case 4: // AND
		result = uint16(dst & src)
		c.SetFlagsLogic8(byte(result))
	case 5: // SUB
		result = uint16(dst) - uint16(src)
		c.SetFlagsArith8(result, dst, src, true)
	case 6: // XOR
		result = uint16(dst ^ src)
		c.SetFlagsLogic8(byte(result))
	case 7: // CMP
		result = uint16(dst) - uint16(src)
		c.SetFlagsArith8(result, dst, src, true)
		return dst // CMP doesn't write result
	}
	return byte(result)
}

func (c *CPU) doALU16(op int, dst, src uint16) uint16 {
	var result uint32
	var carry uint16
	if c.GetFlag(FlagCF) {
		carry = 1
	}

	switch op {
	case 0: // ADD
		result = uint32(dst) + uint32(src)
		c.SetFlagsArith16(result, dst, src, false)
	case 1: // OR
		result = uint32(dst | src)
		c.SetFlagsLogic16(uint16(result))
	case 2: // ADC
		result = uint32(dst) + uint32(src) + uint32(carry)
		res16 := uint16(result)
		c.SetFlag(FlagCF, result > 0xFFFF)
		c.SetFlag(FlagZF, res16 == 0)
		c.SetFlag(FlagSF, res16&0x8000 != 0)
		c.SetFlag(FlagPF, parity(byte(res16)))
		c.SetFlag(FlagAF, (dst^src^res16)&0x10 != 0)
		c.SetFlag(FlagOF, (^(dst^src))&(dst^res16)&0x8000 != 0)
	case 3: // SBB
		result = uint32(dst) - uint32(src) - uint32(carry)
		res16 := uint16(result)
		c.SetFlag(FlagCF, result > 0xFFFF)
		c.SetFlag(FlagZF, res16 == 0)
		c.SetFlag(FlagSF, res16&0x8000 != 0)
		c.SetFlag(FlagPF, parity(byte(res16)))
		c.SetFlag(FlagAF, (dst^src^res16)&0x10 != 0)
		c.SetFlag(FlagOF, (dst^src)&(dst^res16)&0x8000 != 0)
	case 4: // AND
		result = uint32(dst & src)
		c.SetFlagsLogic16(uint16(result))
	case 5: // SUB
		result = uint32(dst) - uint32(src)
		c.SetFlagsArith16(result, dst, src, true)
	case 6: // XOR
		result = uint32(dst ^ src)
		c.SetFlagsLogic16(uint16(result))
	case 7: // CMP
		result = uint32(dst) - uint32(src)
		c.SetFlagsArith16(result, dst, src, true)
		return dst
	}
	return uint16(result)
}

func (c *CPU) aluRM8R8(opcode byte) {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	op := aluOpIndex(opcode)
	dst := c.readModRM8(mod, rm)
	src := c.getReg8(reg)
	result := c.doALU8(op, dst, src)
	if op != 7 { // not CMP
		c.writeModRM8(mod, rm, result)
	}
	c.Cycles += 1
}

func (c *CPU) aluRM16R16(opcode byte) {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	op := aluOpIndex(opcode)
	dst := c.readModRM16(mod, rm)
	src := c.getReg16(reg)
	result := c.doALU16(op, dst, src)
	if op != 7 {
		c.writeModRM16(mod, rm, result)
	}
	c.Cycles += 1
}

func (c *CPU) aluR8RM8(opcode byte) {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	op := aluOpIndex(opcode)
	dst := c.getReg8(reg)
	src := c.readModRM8(mod, rm)
	result := c.doALU8(op, dst, src)
	if op != 7 {
		c.setReg8(reg, result)
	}
	c.Cycles += 1
}

func (c *CPU) aluR16RM16(opcode byte) {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	op := aluOpIndex(opcode)
	dst := c.getReg16(reg)
	src := c.readModRM16(mod, rm)
	result := c.doALU16(op, dst, src)
	if op != 7 {
		c.setReg16(reg, result)
	}
	c.Cycles += 1
}

func (c *CPU) aluALImm8(opcode byte) {
	imm := c.fetchByte()
	op := aluOpIndex(opcode)
	result := c.doALU8(op, c.AL(), imm)
	if op != 7 {
		c.SetAL(result)
	}
	c.Cycles += 1
}

func (c *CPU) aluAXImm16(opcode byte) {
	imm := c.fetchWord()
	op := aluOpIndex(opcode)
	result := c.doALU16(op, c.AX, imm)
	if op != 7 {
		c.AX = result
	}
	c.Cycles += 1
}

// ========== Group 1: ALU r/m, imm ==========

func (c *CPU) group1_8(signExtend bool) {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	dst := c.readModRM8(mod, rm)
	imm := c.fetchByte()
	_ = signExtend // 8-bit op, sign extension not applicable
	result := c.doALU8(reg, dst, imm)
	if reg != 7 { // not CMP
		c.writeModRM8(mod, rm, result)
	}
	c.Cycles += 1
}

func (c *CPU) group1_16(signExtend bool) {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	dst := c.readModRM16(mod, rm)
	var imm uint16
	if signExtend {
		imm = uint16(int16(int8(c.fetchByte())))
	} else {
		imm = c.fetchWord()
	}
	result := c.doALU16(reg, dst, imm)
	if reg != 7 {
		c.writeModRM16(mod, rm, result)
	}
	c.Cycles += 1
}

// ========== Conditional jump helper ==========

func (c *CPU) jccShort(cond bool) {
	disp := int8(c.fetchByte())
	if cond {
		c.IP = uint16(int16(c.IP) + int16(disp))
		c.Cycles += 2
	} else {
		c.Cycles += 1
	}
}

// ========== Shift/Rotate operations ==========

// shiftRotateCore implements all shift/rotate operations for any operand width (8 or 16).
// signBit is 0x80 for 8-bit or 0x8000 for 16-bit.
// width is 8 or 16 (used for SHL/SHR/SAR overflow guards).
func (c *CPU) shiftRotateCore(op int, val uint16, count byte, signBit uint16, width byte) uint16 {
	var cf bool
	secondBit := signBit >> 1 // 0x40 or 0x4000

	if count == 0 {
		// V30MZ: count=0 still computes flags
		switch op {
		case 0: // ROL: OF = CF XOR MSB
			c.SetFlag(FlagOF, c.GetFlag(FlagCF) != (val&signBit != 0))
		case 1: // ROR: OF = MSB XOR (MSB-1)
			c.SetFlag(FlagOF, (val&signBit != 0) != (val&secondBit != 0))
		case 2: // RCL: OF = CF XOR MSB
			c.SetFlag(FlagOF, c.GetFlag(FlagCF) != (val&signBit != 0))
		case 3: // RCR: OF = MSB XOR (MSB-1)
			c.SetFlag(FlagOF, (val&signBit != 0) != (val&secondBit != 0))
		case 4: // SHL count=0: CF preserved, OF=CF^MSB, SZP from value, AF=0
			c.setFlagsSZPByWidth(val, width)
			c.SetFlag(FlagAF, false)
			c.SetFlag(FlagOF, c.GetFlag(FlagCF) != (val&signBit != 0))
		case 5: // SHR count=0: CF preserved, OF=original_MSB^original_(MSB-1), SZP from value, AF=0
			c.setFlagsSZPByWidth(val, width)
			c.SetFlag(FlagAF, false)
			c.SetFlag(FlagOF, (val&signBit != 0) != (val&secondBit != 0))
		case 7: // SAR count=0: CF preserved, OF=result_MSB^result_(MSB-1), SZP from value, AF=0
			c.setFlagsSZPByWidth(val, width)
			c.SetFlag(FlagAF, false)
			c.SetFlag(FlagOF, (val&signBit != 0) != (val&secondBit != 0))
		}
		return val
	}

	switch op {
	case 0: // ROL
		for i := byte(0); i < count; i++ {
			cf = val&signBit != 0
			val = val << 1
			if cf {
				val |= 1
			}
		}
		c.SetFlag(FlagCF, cf)
		c.SetFlag(FlagOF, (val&signBit != 0) != cf)
		return val

	case 1: // ROR
		for i := byte(0); i < count; i++ {
			cf = val&1 != 0
			val = val >> 1
			if cf {
				val |= signBit
			}
		}
		c.SetFlag(FlagCF, cf)
		c.SetFlag(FlagOF, (val&signBit != 0) != (val&secondBit != 0))
		return val

	case 2: // RCL
		for i := byte(0); i < count; i++ {
			oldCF := c.GetFlag(FlagCF)
			cf = val&signBit != 0
			c.SetFlag(FlagCF, cf)
			val = val << 1
			if oldCF {
				val |= 1
			}
		}
		c.SetFlag(FlagOF, (val&signBit != 0) != c.GetFlag(FlagCF))
		return val

	case 3: // RCR
		for i := byte(0); i < count; i++ {
			oldCF := c.GetFlag(FlagCF)
			cf = val&1 != 0
			c.SetFlag(FlagCF, cf)
			val = val >> 1
			if oldCF {
				val |= signBit
			}
		}
		c.SetFlag(FlagOF, (val&signBit != 0) != (val&secondBit != 0))
		return val

	case 4: // SHL/SAL — V30MZ: OF = CF XOR result_MSB (all counts)
		var result uint16
		if count >= width {
			cf = count == width && val&1 != 0
			result = 0
		} else {
			cf = val&(1<<(width-count)) != 0
			result = val << count
		}
		c.SetFlag(FlagCF, cf)
		c.setFlagsSZPByWidth(result, width)
		c.SetFlag(FlagAF, false)
		c.SetFlag(FlagOF, cf != (result&signBit != 0))
		return result

	case 5: // SHR — V30MZ: OF = result_MSB XOR result_(MSB-1)
		var result uint16
		if count == 0 {
			result = val
		} else if count >= width {
			cf = count == width && val&signBit != 0
			result = 0
		} else {
			cf = val&(1<<(count-1)) != 0
			result = val >> count
		}
		c.SetFlag(FlagCF, cf)
		c.setFlagsSZPByWidth(result, width)
		c.SetFlag(FlagAF, false)
		c.SetFlag(FlagOF, (result&signBit != 0) != (result&secondBit != 0))
		return result

	case 7: // SAR — V30MZ: OF = result_MSB XOR result_(MSB-1)
		var result uint16
		if count == 0 {
			result = val
		} else if count >= width {
			cf = val&signBit != 0
			if cf {
				result = (1 << width) - 1 // 0xFF or 0xFFFF
			}
		} else {
			cf = val&(1<<(count-1)) != 0
			if width == 8 {
				result = uint16(byte(int8(byte(val)) >> count))
			} else {
				result = uint16(int16(val) >> count)
			}
		}
		c.SetFlag(FlagCF, cf)
		c.setFlagsSZPByWidth(result, width)
		c.SetFlag(FlagAF, false)
		c.SetFlag(FlagOF, (result&signBit != 0) != (result&secondBit != 0))
		return result
	}
	return val
}

// setFlagsSZPByWidth calls the appropriate SetFlagsSZP for the given width.
func (c *CPU) setFlagsSZPByWidth(result uint16, width byte) {
	if width == 8 {
		c.SetFlagsSZP8(byte(result))
	} else {
		c.SetFlagsSZP16(result)
	}
}

func (c *CPU) shiftRotate8(op int, val byte, count byte) byte {
	return byte(c.shiftRotateCore(op, uint16(val), count, 0x80, 8))
}

func (c *CPU) shiftRotate16(op int, val uint16, count byte) uint16 {
	return c.shiftRotateCore(op, val, count, 0x8000, 16)
}

// ========== Group 3: TEST, NOT, NEG, MUL, IMUL, DIV, IDIV ==========

func (c *CPU) group3_8() {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	val := c.readModRM8(mod, rm)

	switch reg {
	case 0: // TEST r/m8, imm8
		imm := c.fetchByte()
		c.SetFlagsLogic8(val & imm)
		c.Cycles += 1
	case 1: // TEST r/m8, imm8 (undocumented alias)
		imm := c.fetchByte()
		c.SetFlagsLogic8(val & imm)
		c.Cycles += 1
	case 2: // NOT r/m8
		c.writeModRM8(mod, rm, ^val)
		c.Cycles += 1
	case 3: // NEG r/m8 — all arithmetic flags set
		c.SetFlag(FlagCF, val != 0)
		c.SetFlag(FlagOF, val == 0x80)
		c.SetFlag(FlagAF, val&0x0F != 0)
		result := byte(^val) + 1
		c.SetFlagsSZP8(result)
		c.writeModRM8(mod, rm, result)
		c.Cycles += 1
	case 4: // MUL r/m8
		result := uint16(c.AL()) * uint16(val)
		c.AX = result
		c.SetFlag(FlagCF, c.AH() != 0)
		c.SetFlag(FlagOF, c.AH() != 0)
		c.Cycles += 25
	case 5: // IMUL r/m8 — Mednafen: CF=OF=(AH!=0)
		result := int16(int8(c.AL())) * int16(int8(val))
		c.AX = uint16(result)
		c.SetFlag(FlagCF, c.AH() != 0)
		c.SetFlag(FlagOF, c.AH() != 0)
		c.Cycles += 25
	case 6: // DIV r/m8
		if val == 0 {
			c.Interrupt(0) // divide by zero
			c.Cycles += 1
			return
		}
		quotient := c.AX / uint16(val)
		remainder := c.AX % uint16(val)
		if quotient > 0xFF {
			c.Interrupt(0)
			c.Cycles += 1
			return
		}
		c.SetAL(byte(quotient))
		c.SetAH(byte(remainder))
		c.Cycles += 25
	case 7: // IDIV r/m8
		if val == 0 {
			c.Interrupt(0)
			c.Cycles += 1
			return
		}
		dividend := int16(c.AX)
		divisor := int16(int8(val))
		quotient := dividend / divisor
		remainder := dividend % divisor
		if quotient > 127 || quotient < -128 {
			c.Interrupt(0)
			c.Cycles += 1
			return
		}
		c.SetAL(byte(int8(quotient)))
		c.SetAH(byte(int8(remainder)))
		c.Cycles += 25
	}
}

func (c *CPU) group3_16() {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	val := c.readModRM16(mod, rm)

	switch reg {
	case 0: // TEST r/m16, imm16
		imm := c.fetchWord()
		c.SetFlagsLogic16(val & imm)
		c.Cycles += 1
	case 1: // TEST (undocumented alias)
		imm := c.fetchWord()
		c.SetFlagsLogic16(val & imm)
		c.Cycles += 1
	case 2: // NOT r/m16
		c.writeModRM16(mod, rm, ^val)
		c.Cycles += 1
	case 3: // NEG r/m16 — all arithmetic flags set
		c.SetFlag(FlagCF, val != 0)
		c.SetFlag(FlagOF, val == 0x8000)
		c.SetFlag(FlagAF, val&0x0F != 0)
		result := uint16(^val) + 1
		c.SetFlagsSZP16(result)
		c.writeModRM16(mod, rm, result)
		c.Cycles += 1
	case 4: // MUL r/m16
		result := uint32(c.AX) * uint32(val)
		c.AX = uint16(result)
		c.DX = uint16(result >> 16)
		c.SetFlag(FlagCF, c.DX != 0)
		c.SetFlag(FlagOF, c.DX != 0)
		c.Cycles += 35
	case 5: // IMUL r/m16 — Mednafen: CF=OF=(DX!=0)
		result := int32(int16(c.AX)) * int32(int16(val))
		c.AX = uint16(result)
		c.DX = uint16(uint32(result) >> 16)
		c.SetFlag(FlagCF, c.DX != 0)
		c.SetFlag(FlagOF, c.DX != 0)
		c.Cycles += 35
	case 6: // DIV r/m16
		if val == 0 {
			c.Interrupt(0)
			c.Cycles += 1
			return
		}
		dividend := (uint32(c.DX) << 16) | uint32(c.AX)
		quotient := dividend / uint32(val)
		remainder := dividend % uint32(val)
		if quotient > 0xFFFF {
			c.Interrupt(0)
			c.Cycles += 1
			return
		}
		c.AX = uint16(quotient)
		c.DX = uint16(remainder)
		c.Cycles += 35
	case 7: // IDIV r/m16
		if val == 0 {
			c.Interrupt(0)
			c.Cycles += 1
			return
		}
		dividend := int32((uint32(c.DX) << 16) | uint32(c.AX))
		divisor := int32(int16(val))
		quotient := dividend / divisor
		remainder := dividend % divisor
		if quotient > 32767 || quotient < -32768 {
			c.Interrupt(0)
			c.Cycles += 1
			return
		}
		c.AX = uint16(int16(quotient))
		c.DX = uint16(int16(remainder))
		c.Cycles += 35
	}
}

// ========== Group 4: INC/DEC r/m8 ==========

func (c *CPU) group4() {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)
	val := c.readModRM8(mod, rm)

	switch reg {
	case 0: // INC r/m8
		result := uint16(val) + 1
		saveCF := c.GetFlag(FlagCF)
		c.SetFlagsArith8(result, val, 1, false)
		c.SetFlag(FlagCF, saveCF)
		c.writeModRM8(mod, rm, byte(result))
		c.Cycles += 1
	case 1: // DEC r/m8
		result := uint16(val) - 1
		saveCF := c.GetFlag(FlagCF)
		c.SetFlagsArith8(result, val, 1, true)
		c.SetFlag(FlagCF, saveCF)
		c.writeModRM8(mod, rm, byte(result))
		c.Cycles += 1
	default:
		fmt.Printf("WARNING: unimplemented Group4 reg=%d\n", reg)
		c.Cycles += 1
	}
}

// ========== Group 5: INC/DEC/CALL/JMP/PUSH r/m16 ==========

func (c *CPU) group5() {
	modrm := c.fetchByte()
	mod, reg, rm := c.decodeModRM(modrm)
	c.resolveModRM(mod, rm)

	switch reg {
	case 0: // INC r/m16
		val := c.readModRM16(mod, rm)
		result := uint32(val) + 1
		saveCF := c.GetFlag(FlagCF)
		c.SetFlagsArith16(result, val, 1, false)
		c.SetFlag(FlagCF, saveCF)
		c.writeModRM16(mod, rm, uint16(result))
		c.Cycles += 1
	case 1: // DEC r/m16
		val := c.readModRM16(mod, rm)
		result := uint32(val) - 1
		saveCF := c.GetFlag(FlagCF)
		c.SetFlagsArith16(result, val, 1, true)
		c.SetFlag(FlagCF, saveCF)
		c.writeModRM16(mod, rm, uint16(result))
		c.Cycles += 1
	case 2: // CALL r/m16 (near indirect)
		target := c.readModRM16(mod, rm)
		c.push16(c.IP)
		c.IP = target
		c.Cycles += 3
	case 3: // CALL far m16:16
		newIP := c.Bus.Read16(c.modrmSeg, c.modrmOff)
		newCS := c.Bus.Read16(c.modrmSeg, c.modrmOff+2)
		c.push16(c.CS)
		c.push16(c.IP)
		c.CS = newCS
		c.IP = newIP
		c.Cycles += 5
	case 4: // JMP r/m16 (near indirect)
		target := c.readModRM16(mod, rm)
		c.IP = target
		c.Cycles += 2
	case 5: // JMP far m16:16
		newIP := c.Bus.Read16(c.modrmSeg, c.modrmOff)
		newCS := c.Bus.Read16(c.modrmSeg, c.modrmOff+2)
		c.CS = newCS
		c.IP = newIP
		c.Cycles += 3
	case 6: // PUSH r/m16
		val := c.readModRM16(mod, rm)
		c.push16(val)
		c.Cycles += 1
	case 7: // undefined on 186, treat as NOP
		c.Cycles += 1
	}
}

// ========== String operations ==========

func (c *CPU) execStringOp(opcode byte) {
	if c.repPrefix != 0 {
		if c.CX == 0 {
			c.Cycles += 5
			return
		}
		// Mednafen: CLK(5) base overhead on each entry to REP handler
		c.Cycles += 5
		// Execute one iteration
		c.doStringOp(opcode)
		c.CX--
		// Per-iteration cycle costs (Mednafen-verified)
		switch opcode {
		case 0xA4, 0xA5: // MOVS
			c.Cycles += 5
		case 0xA6, 0xA7: // CMPS
			c.Cycles += 9 // 6 + 3 extra in REP
		case 0xAA, 0xAB: // STOS
			c.Cycles += 3
		case 0xAC, 0xAD: // LODS
			c.Cycles += 3
		case 0xAE, 0xAF: // SCAS
			c.Cycles += 4
		}
		// Check termination for CMPS/SCAS
		isCmpSca := opcode == 0xA6 || opcode == 0xA7 || opcode == 0xAE || opcode == 0xAF
		if isCmpSca {
			if c.repPrefix == 1 && !c.GetFlag(FlagZF) {
				return // REPE: stop if ZF=0
			}
			if c.repPrefix == 2 && c.GetFlag(FlagZF) {
				return // REPNE: stop if ZF=1
			}
		}
		// If more iterations remain, rewind IP to the REP prefix
		// so the system loop can check interrupts before re-entering.
		if c.CX > 0 {
			c.IP = c.instrStartIP
		}
	} else {
		c.doStringOp(opcode)
		c.Cycles += 1
	}
}

func (c *CPU) doStringOp(opcode byte) {
	srcSeg := c.getEffectiveSegment(c.DS)
	delta := uint16(1)
	if opcode == 0xA5 || opcode == 0xA7 || opcode == 0xAB || opcode == 0xAD || opcode == 0xAF {
		delta = 2 // word operations
	}
	if c.GetFlag(FlagDF) {
		delta = uint16(-int16(delta))
	}

	switch opcode {
	case 0xA4: // MOVSB
		val := c.Bus.Read8(srcSeg, c.SI)
		c.Bus.Write8(c.ES, c.DI, val)
		c.SI += delta
		c.DI += delta
	case 0xA5: // MOVSW
		val := c.Bus.Read16(srcSeg, c.SI)
		c.Bus.Write16(c.ES, c.DI, val)
		c.SI += delta
		c.DI += delta
	case 0xA6: // CMPSB
		val1 := c.Bus.Read8(srcSeg, c.SI)
		val2 := c.Bus.Read8(c.ES, c.DI)
		result := uint16(val1) - uint16(val2)
		c.SetFlagsArith8(result, val1, val2, true)
		c.SI += delta
		c.DI += delta
	case 0xA7: // CMPSW
		val1 := c.Bus.Read16(srcSeg, c.SI)
		val2 := c.Bus.Read16(c.ES, c.DI)
		result := uint32(val1) - uint32(val2)
		c.SetFlagsArith16(result, val1, val2, true)
		c.SI += delta
		c.DI += delta
	case 0xAA: // STOSB
		c.Bus.Write8(c.ES, c.DI, c.AL())
		c.DI += delta
	case 0xAB: // STOSW
		c.Bus.Write16(c.ES, c.DI, c.AX)
		c.DI += delta
	case 0xAC: // LODSB
		c.SetAL(c.Bus.Read8(srcSeg, c.SI))
		c.SI += delta
	case 0xAD: // LODSW
		c.AX = c.Bus.Read16(srcSeg, c.SI)
		c.SI += delta
	case 0xAE: // SCASB
		val := c.Bus.Read8(c.ES, c.DI)
		result := uint16(c.AL()) - uint16(val)
		c.SetFlagsArith8(result, c.AL(), val, true)
		c.DI += delta
	case 0xAF: // SCASW
		val := c.Bus.Read16(c.ES, c.DI)
		result := uint32(c.AX) - uint32(val)
		c.SetFlagsArith16(result, c.AX, val, true)
		c.DI += delta
	}
}

// executeTwoByteOpcode handles the 0x0F prefix opcodes.
func (c *CPU) executeTwoByteOpcode() {
	op2 := c.fetchByte()
	switch op2 {
	// Jcc near (16-bit displacement)
	case 0x80: // JO near
		c.jccNear(c.GetFlag(FlagOF))
	case 0x81: // JNO near
		c.jccNear(!c.GetFlag(FlagOF))
	case 0x82: // JB near
		c.jccNear(c.GetFlag(FlagCF))
	case 0x83: // JNB near
		c.jccNear(!c.GetFlag(FlagCF))
	case 0x84: // JZ near
		c.jccNear(c.GetFlag(FlagZF))
	case 0x85: // JNZ near
		c.jccNear(!c.GetFlag(FlagZF))
	case 0x86: // JBE near
		c.jccNear(c.GetFlag(FlagCF) || c.GetFlag(FlagZF))
	case 0x87: // JA near
		c.jccNear(!c.GetFlag(FlagCF) && !c.GetFlag(FlagZF))
	case 0x88: // JS near
		c.jccNear(c.GetFlag(FlagSF))
	case 0x89: // JNS near
		c.jccNear(!c.GetFlag(FlagSF))
	case 0x8A: // JP near
		c.jccNear(c.GetFlag(FlagPF))
	case 0x8B: // JNP near
		c.jccNear(!c.GetFlag(FlagPF))
	case 0x8C: // JL near
		c.jccNear(c.GetFlag(FlagSF) != c.GetFlag(FlagOF))
	case 0x8D: // JGE near
		c.jccNear(c.GetFlag(FlagSF) == c.GetFlag(FlagOF))
	case 0x8E: // JLE near
		c.jccNear(c.GetFlag(FlagZF) || (c.GetFlag(FlagSF) != c.GetFlag(FlagOF)))
	case 0x8F: // JG near
		c.jccNear(!c.GetFlag(FlagZF) && (c.GetFlag(FlagSF) == c.GetFlag(FlagOF)))

	// SETcc r/m8
	case 0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97,
		0x98, 0x99, 0x9A, 0x9B, 0x9C, 0x9D, 0x9E, 0x9F:
		modrm := c.fetchByte()
		mod, _, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		cond := c.evaluateCondition(op2 & 0x0F)
		if cond {
			c.writeModRM8(mod, rm, 1)
		} else {
			c.writeModRM8(mod, rm, 0)
		}
		c.Cycles += 1

	// MOVZX / MOVSX
	case 0xB6: // MOVZX r16, r/m8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM8(mod, rm)
		c.setReg16(reg, uint16(val))
		c.Cycles += 1

	case 0xB7: // MOVZX r16, r/m16 (NOP-like on 16-bit)
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM16(mod, rm)
		c.setReg16(reg, val)
		c.Cycles += 1

	case 0xBE: // MOVSX r16, r/m8
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM8(mod, rm)
		c.setReg16(reg, uint16(int16(int8(val))))
		c.Cycles += 1

	case 0xBF: // MOVSX r16, r/m16 (NOP-like on 16-bit)
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		val := c.readModRM16(mod, rm)
		c.setReg16(reg, val)
		c.Cycles += 1

	// IMUL r16, r/m16
	case 0xAF:
		modrm := c.fetchByte()
		mod, reg, rm := c.decodeModRM(modrm)
		c.resolveModRM(mod, rm)
		src := c.readModRM16(mod, rm)
		dst := c.getReg16(reg)
		result := int32(int16(dst)) * int32(int16(src))
		c.setReg16(reg, uint16(result))
		overflow := result > 32767 || result < -32768
		c.SetFlag(FlagCF, overflow)
		c.SetFlag(FlagOF, overflow)
		c.Cycles += 4

	default:
		fmt.Printf("WARNING: unimplemented 0x0F 0x%02X at %04X:%04X\n", op2, c.CS, c.IP)
		c.Cycles += 1
	}
}

func (c *CPU) jccNear(cond bool) {
	disp := int16(c.fetchWord())
	if cond {
		c.IP = uint16(int16(c.IP) + disp)
		c.Cycles += 2
	} else {
		c.Cycles += 1
	}
}

// execIOStringOp handles INS/OUTS instructions with optional REP prefix.
func (c *CPU) execIOStringOp(opcode byte) {
	if c.repPrefix != 0 {
		if c.CX == 0 {
			c.Cycles += 5
			return
		}
		// Mednafen: CLK(5) base overhead
		c.Cycles += 5
		c.doIOStringOp(opcode)
		c.CX--
		// Per-iteration cycle costs (Mednafen-verified)
		switch opcode {
		case 0x6C, 0x6D: // INS
			c.Cycles += 6
		case 0x6E, 0x6F: // OUTS
			c.Cycles += 7
		}
		// If more iterations remain, rewind IP for interrupt interleave.
		if c.CX > 0 {
			c.IP = c.instrStartIP
		}
	} else {
		c.doIOStringOp(opcode)
		c.Cycles += 2
	}
}

func (c *CPU) doIOStringOp(opcode byte) {
	port := byte(c.DX)
	delta := uint16(1)
	if opcode == 0x6D || opcode == 0x6F {
		delta = 2
	}
	if c.GetFlag(FlagDF) {
		delta = uint16(-int16(delta))
	}

	switch opcode {
	case 0x6C: // INSB
		val := c.Bus.IORead(port)
		c.Bus.Write8(c.ES, c.DI, val)
		c.DI += delta
	case 0x6D: // INSW
		lo := c.Bus.IORead(port)
		hi := c.Bus.IORead(port + 1)
		c.Bus.Write16(c.ES, c.DI, uint16(hi)<<8|uint16(lo))
		c.DI += delta
	case 0x6E: // OUTSB
		srcSeg := c.getEffectiveSegment(c.DS)
		val := c.Bus.Read8(srcSeg, c.SI)
		c.Bus.IOWrite(port, val)
		c.SI += delta
	case 0x6F: // OUTSW
		srcSeg := c.getEffectiveSegment(c.DS)
		val := c.Bus.Read16(srcSeg, c.SI)
		c.Bus.IOWrite(port, byte(val))
		c.Bus.IOWrite(port+1, byte(val>>8))
		c.SI += delta
	}
}

// BCD adjust helpers
// daa matches Mednafen's ADJ4(6, 0x60) macro.
func (c *CPU) daa() {
	original := c.AL()
	oldCF := c.GetFlag(FlagCF)
	oldAF := c.GetFlag(FlagAF)

	// Determine adjustments needed based on original value and input flags
	needHighAdj := oldCF || original > 0x99
	needLowAdj := oldAF || (original&0x0F) > 9

	// Apply high adjustment first (matching V30MZ order)
	if needHighAdj {
		c.SetAL(c.AL() + 0x60)
	}
	// Apply low adjustment
	if needLowAdj {
		c.SetAL(c.AL() + 0x06)
	}

	// Set flags based on conditions (not carry)
	c.SetFlag(FlagCF, needHighAdj)
	c.SetFlag(FlagAF, needLowAdj)

	adjusted := c.AL()
	c.SetFlagsSZP8(adjusted)
	// V30MZ: OF = signed overflow of (adjusted - original)
	diff := adjusted - original
	c.SetFlag(FlagOF, (adjusted^original)&(adjusted^diff)&0x80 != 0)
	c.Cycles += 1
}

// das — V30MZ: CF/AF based on original conditions, not adjustment borrow.
func (c *CPU) das() {
	original := c.AL()
	oldCF := c.GetFlag(FlagCF)
	oldAF := c.GetFlag(FlagAF)

	// Determine adjustments needed based on original value and input flags
	needHighAdj := oldCF || original > 0x99
	needLowAdj := oldAF || (original&0x0F) > 9

	// Apply high adjustment first (matching test ROM order)
	if needHighAdj {
		c.SetAL(c.AL() - 0x60)
	}
	// Apply low adjustment
	if needLowAdj {
		c.SetAL(c.AL() - 0x06)
	}

	// Set flags based on conditions (not borrow)
	c.SetFlag(FlagCF, needHighAdj)
	c.SetFlag(FlagAF, needLowAdj)

	adjusted := c.AL()
	c.SetFlagsSZP8(adjusted)
	// V30MZ: OF = signed overflow of (adjusted - original)
	diff := adjusted - original
	c.SetFlag(FlagOF, (adjusted^original)&(adjusted^diff)&0x80 != 0)
	c.Cycles += 1
}

func (c *CPU) aaa() {
	// V30MZ: mask AL to low nibble FIRST, then adjust
	c.SetAL(c.AL() & 0x0F)
	if c.AL() > 9 || c.GetFlag(FlagAF) {
		c.SetAL(c.AL() + 6)
		c.SetAH(c.AH() + 1)
		c.SetAL(c.AL() & 0x0F)
		c.SetFlag(FlagAF, true)
		c.SetFlag(FlagCF, true)
		c.SetFlag(FlagZF, true)
		c.SetFlag(FlagSF, false)
	} else {
		c.SetFlag(FlagAF, false)
		c.SetFlag(FlagCF, false)
		c.SetFlag(FlagZF, false)
		c.SetFlag(FlagSF, true)
	}
	c.SetFlag(FlagOF, false)
	c.SetFlag(FlagPF, true)
	c.Cycles += 1
}

func (c *CPU) aas() {
	// V30MZ: mask AL to low nibble FIRST, then adjust
	c.SetAL(c.AL() & 0x0F)
	if c.AL() > 9 || c.GetFlag(FlagAF) {
		c.SetAL(c.AL() - 6)
		c.SetAH(c.AH() - 1)
		c.SetAL(c.AL() & 0x0F)
		c.SetFlag(FlagAF, true)
		c.SetFlag(FlagCF, true)
		c.SetFlag(FlagZF, true)
		c.SetFlag(FlagSF, false)
	} else {
		c.SetFlag(FlagAF, false)
		c.SetFlag(FlagCF, false)
		c.SetFlag(FlagZF, false)
		c.SetFlag(FlagSF, true)
	}
	c.SetFlag(FlagOF, false)
	c.SetFlag(FlagPF, true)
	c.Cycles += 1
}

func (c *CPU) evaluateCondition(cc byte) bool {
	switch cc {
	case 0x0:
		return c.GetFlag(FlagOF)
	case 0x1:
		return !c.GetFlag(FlagOF)
	case 0x2:
		return c.GetFlag(FlagCF)
	case 0x3:
		return !c.GetFlag(FlagCF)
	case 0x4:
		return c.GetFlag(FlagZF)
	case 0x5:
		return !c.GetFlag(FlagZF)
	case 0x6:
		return c.GetFlag(FlagCF) || c.GetFlag(FlagZF)
	case 0x7:
		return !c.GetFlag(FlagCF) && !c.GetFlag(FlagZF)
	case 0x8:
		return c.GetFlag(FlagSF)
	case 0x9:
		return !c.GetFlag(FlagSF)
	case 0xA:
		return c.GetFlag(FlagPF)
	case 0xB:
		return !c.GetFlag(FlagPF)
	case 0xC:
		return c.GetFlag(FlagSF) != c.GetFlag(FlagOF)
	case 0xD:
		return c.GetFlag(FlagSF) == c.GetFlag(FlagOF)
	case 0xE:
		return c.GetFlag(FlagZF) || (c.GetFlag(FlagSF) != c.GetFlag(FlagOF))
	case 0xF:
		return !c.GetFlag(FlagZF) && (c.GetFlag(FlagSF) == c.GetFlag(FlagOF))
	}
	return false
}
