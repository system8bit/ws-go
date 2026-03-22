package cpu

import "testing"

func TestJZ_Taken(t *testing.T) {
	c, bus := newTestCPU()
	// JZ +5: opcode 74, rel8=0x05
	loadCode(bus, 0x74, 0x05)
	c.SetFlag(FlagZF, true)
	c.Step()
	// IP = 2 (after fetch) + 5 = 7
	if c.IP != 7 {
		t.Errorf("IP = 0x%04X, want 0x0007", c.IP)
	}
}

func TestJZ_NotTaken(t *testing.T) {
	c, bus := newTestCPU()
	loadCode(bus, 0x74, 0x05)
	c.SetFlag(FlagZF, false)
	c.Step()
	if c.IP != 2 {
		t.Errorf("IP = 0x%04X, want 0x0002 (not taken)", c.IP)
	}
}

func TestJNZ_Taken(t *testing.T) {
	c, bus := newTestCPU()
	loadCode(bus, 0x75, 0x03)
	c.SetFlag(FlagZF, false)
	c.Step()
	if c.IP != 5 {
		t.Errorf("IP = 0x%04X, want 0x0005", c.IP)
	}
}

func TestJMP_Short(t *testing.T) {
	c, bus := newTestCPU()
	// JMP short +10: opcode EB, rel8=0x0A
	loadCode(bus, 0xEB, 0x0A)
	c.Step()
	if c.IP != 12 {
		t.Errorf("IP = 0x%04X, want 0x000C", c.IP)
	}
}

func TestJMP_Short_Backward(t *testing.T) {
	c, bus := newTestCPU()
	c.IP = 0x10
	bus.mem[0x10] = 0xEB
	bus.mem[0x11] = 0xFE // -2 → jump to 0x10
	c.Step()
	if c.IP != 0x10 {
		t.Errorf("IP = 0x%04X, want 0x0010 (infinite loop)", c.IP)
	}
}

func TestCALL_Near(t *testing.T) {
	c, bus := newTestCPU()
	// CALL near +0x0010: opcode E8, rel16=0x0010
	loadCode(bus, 0xE8, 0x10, 0x00)
	c.SP = 0x100
	c.Step()
	// IP after fetch = 3, pushed to stack, then IP += 0x10 = 0x13
	if c.IP != 0x13 {
		t.Errorf("IP = 0x%04X, want 0x0013", c.IP)
	}
	// Stack should have return address (3)
	retAddr := bus.Read16(c.SS, c.SP)
	if retAddr != 3 {
		t.Errorf("return addr = 0x%04X, want 0x0003", retAddr)
	}
}

func TestRET_Near(t *testing.T) {
	c, bus := newTestCPU()
	// RET: opcode C3
	loadCode(bus, 0xC3)
	c.SP = 0x100
	bus.Write16(0, 0x100, 0x1234) // return address on stack
	c.Step()
	if c.IP != 0x1234 {
		t.Errorf("IP = 0x%04X, want 0x1234", c.IP)
	}
	if c.SP != 0x102 {
		t.Errorf("SP = 0x%04X, want 0x0102", c.SP)
	}
}

func TestLOOP(t *testing.T) {
	c, bus := newTestCPU()
	// LOOP -2: opcode E2, rel8=0xFE → jump back to itself
	c.IP = 0x10
	bus.mem[0x10] = 0xE2
	bus.mem[0x11] = 0xFE
	c.CX = 3
	c.Step() // CX becomes 2, jump taken
	if c.IP != 0x10 || c.CX != 2 {
		t.Errorf("iter1: IP=0x%04X CX=%d, want IP=0x10 CX=2", c.IP, c.CX)
	}
	c.Step() // CX becomes 1
	c.Step() // CX becomes 0, jump NOT taken
	if c.IP != 0x12 || c.CX != 0 {
		t.Errorf("iter3: IP=0x%04X CX=%d, want IP=0x12 CX=0", c.IP, c.CX)
	}
}

func TestNOP(t *testing.T) {
	c, bus := newTestCPU()
	loadCode(bus, 0x90) // NOP
	cycles := c.Step()
	if c.IP != 1 {
		t.Errorf("IP = %d, want 1", c.IP)
	}
	if cycles != 3 {
		t.Errorf("cycles = %d, want 3 (Mednafen NOP=3)", cycles)
	}
}
