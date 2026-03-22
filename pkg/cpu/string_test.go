package cpu

import "testing"

func TestMOVSB_Single(t *testing.T) {
	c, bus := newTestCPU()
	// MOVSB: opcode A4
	loadCode(bus, 0xA4)
	c.SI = 0x100
	c.DI = 0x200
	bus.mem[0x100] = 0x42
	c.Step()
	if bus.mem[0x200] != 0x42 {
		t.Errorf("[DI] = 0x%02X, want 0x42", bus.mem[0x200])
	}
	if c.SI != 0x101 || c.DI != 0x201 {
		t.Errorf("SI=%04X DI=%04X, want SI=0101 DI=0201", c.SI, c.DI)
	}
}

func TestMOVSB_DF(t *testing.T) {
	c, bus := newTestCPU()
	loadCode(bus, 0xA4) // MOVSB
	c.SI = 0x105
	c.DI = 0x205
	c.SetFlag(FlagDF, true) // direction = decrement
	bus.mem[0x105] = 0xAB
	c.Step()
	if bus.mem[0x205] != 0xAB {
		t.Errorf("[DI] = 0x%02X, want 0xAB", bus.mem[0x205])
	}
	if c.SI != 0x104 || c.DI != 0x204 {
		t.Errorf("SI=%04X DI=%04X, want SI=0104 DI=0204", c.SI, c.DI)
	}
}

func TestSTOSB(t *testing.T) {
	c, bus := newTestCPU()
	// STOSB: opcode AA
	loadCode(bus, 0xAA)
	c.DI = 0x300
	c.SetAL(0x77)
	c.Step()
	if bus.mem[0x300] != 0x77 {
		t.Errorf("[DI] = 0x%02X, want 0x77", bus.mem[0x300])
	}
	if c.DI != 0x301 {
		t.Errorf("DI = 0x%04X, want 0x0301", c.DI)
	}
}

func TestLODSB(t *testing.T) {
	c, bus := newTestCPU()
	// LODSB: opcode AC
	loadCode(bus, 0xAC)
	c.SI = 0x100
	bus.mem[0x100] = 0x55
	c.Step()
	if c.AL() != 0x55 {
		t.Errorf("AL = 0x%02X, want 0x55", c.AL())
	}
	if c.SI != 0x101 {
		t.Errorf("SI = 0x%04X, want 0x0101", c.SI)
	}
}

func TestREP_MOVSB(t *testing.T) {
	c, bus := newTestCPU()
	// REP MOVSB: F3 A4
	loadCode(bus, 0xF3, 0xA4)
	c.SI = 0x100
	c.DI = 0x200
	c.CX = 3
	bus.mem[0x100] = 0x11
	bus.mem[0x101] = 0x22
	bus.mem[0x102] = 0x33

	// REP interleave: each Step() does one iteration and rewinds IP
	for c.CX > 0 {
		c.Step()
	}

	if bus.mem[0x200] != 0x11 || bus.mem[0x201] != 0x22 || bus.mem[0x202] != 0x33 {
		t.Errorf("copied: %02X %02X %02X, want 11 22 33",
			bus.mem[0x200], bus.mem[0x201], bus.mem[0x202])
	}
	if c.CX != 0 {
		t.Errorf("CX = %d, want 0", c.CX)
	}
}

func TestREP_STOSB(t *testing.T) {
	c, bus := newTestCPU()
	// REP STOSB: F3 AA
	loadCode(bus, 0xF3, 0xAA)
	c.DI = 0x300
	c.CX = 4
	c.SetAL(0xCC)

	for c.CX > 0 {
		c.Step()
	}

	for i := 0; i < 4; i++ {
		if bus.mem[0x300+i] != 0xCC {
			t.Errorf("[0x%04X] = 0x%02X, want 0xCC", 0x300+i, bus.mem[0x300+i])
		}
	}
}
