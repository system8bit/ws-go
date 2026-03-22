package cpu

import "testing"

func TestINT_Instruction(t *testing.T) {
	c, bus := newTestCPU()
	// Set up IVT: INT 0x10 → CS:IP = 0x2000:0x0050
	bus.Write16(0, 0x40, 0x0050) // IP for INT 0x10 (offset = 0x10 * 4)
	bus.Write16(0, 0x42, 0x2000) // CS for INT 0x10
	// INT 0x10: opcode CD 10
	c.IP = 0x100
	bus.mem[0x100] = 0xCD
	bus.mem[0x101] = 0x10
	c.InterruptEnable = true
	c.SetFlag(FlagIF, true)
	c.SP = 0xFFF0
	c.Step()
	if c.CS != 0x2000 || c.IP != 0x0050 {
		t.Errorf("CS:IP = %04X:%04X, want 2000:0050", c.CS, c.IP)
	}
	if c.InterruptEnable {
		t.Error("IF should be cleared after INT")
	}
	// Stack should have: flags, old CS, old IP
	if c.SP != 0xFFF0-6 {
		t.Errorf("SP = 0x%04X, want 0x%04X (3 words pushed)", c.SP, 0xFFF0-6)
	}
}

func TestIRET(t *testing.T) {
	c, bus := newTestCPU()
	// IRET: opcode CF
	c.IP = 0x200
	bus.mem[0x200] = 0xCF
	// Push return state: flags, CS, IP
	c.SP = 0x100
	c.push16(0xF202) // flags with IF set (bit 9 = 0x0200)
	c.push16(0x3000) // return CS
	c.push16(0x0010) // return IP
	c.Step()
	if c.CS != 0x3000 || c.IP != 0x0010 {
		t.Errorf("CS:IP = %04X:%04X, want 3000:0010", c.CS, c.IP)
	}
	if !c.InterruptEnable {
		t.Error("IF should be restored from saved flags")
	}
}

func TestHLT(t *testing.T) {
	c, bus := newTestCPU()
	loadCode(bus, 0xF4) // HLT
	c.Step()
	if !c.Halted {
		t.Error("CPU should be halted after HLT")
	}
	if c.IP != 1 {
		t.Errorf("IP = %d, want 1", c.IP)
	}
}

func TestSTI_CLI(t *testing.T) {
	c, bus := newTestCPU()
	// STI: FB, CLI: FA
	loadCode(bus, 0xFB, 0xFA)
	c.Step() // STI
	if !c.InterruptEnable {
		t.Error("IF should be set after STI")
	}
	c.Step() // CLI
	if c.InterruptEnable {
		t.Error("IF should be cleared after CLI")
	}
}

func TestInterruptDispatch_Cycles(t *testing.T) {
	c, bus := newTestCPU()
	// Set up IVT: INT 5 → 0x1000:0x0000
	bus.Write16(0, 20, 0x0000) // IP for INT 5
	bus.Write16(0, 22, 0x1000) // CS for INT 5
	c.SP = 0xFFF0
	c.Interrupt(5)
	// Interrupt() sets PendingCycles=32 (Mednafen CLK(32))
	if c.PendingCycles != 32 {
		t.Errorf("PendingCycles = %d, want 32", c.PendingCycles)
	}
	if c.CS != 0x1000 {
		t.Errorf("CS = 0x%04X, want 0x1000", c.CS)
	}
}
