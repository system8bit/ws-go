package cpu

import "testing"

func TestGetSetReg8(t *testing.T) {
	c, _ := newTestCPU()
	// Write distinct values to each 8-bit register and verify readback
	vals := [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	for i, v := range vals {
		c.setReg8(i, v)
	}
	for i, want := range vals {
		if got := c.getReg8(i); got != want {
			t.Errorf("getReg8(%d) = 0x%02X, want 0x%02X", i, got, want)
		}
	}
	// Verify AX = AL|AH
	if c.AL() != 0x11 || c.AH() != 0x55 {
		t.Errorf("AX byte access: AL=0x%02X AH=0x%02X", c.AL(), c.AH())
	}
}

func TestGetSetReg16(t *testing.T) {
	c, _ := newTestCPU()
	vals := [8]uint16{0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666, 0x7777, 0x8888}
	for i, v := range vals {
		c.setReg16(i, v)
	}
	// 0=AX, 1=CX, 2=DX, 3=BX, 4=SP, 5=BP, 6=SI, 7=DI
	if c.AX != 0x1111 || c.CX != 0x2222 || c.DX != 0x3333 || c.BX != 0x4444 {
		t.Errorf("reg16 0-3: AX=%04X CX=%04X DX=%04X BX=%04X", c.AX, c.CX, c.DX, c.BX)
	}
	if c.SP != 0x5555 || c.BP != 0x6666 || c.SI != 0x7777 || c.DI != 0x8888 {
		t.Errorf("reg16 4-7: SP=%04X BP=%04X SI=%04X DI=%04X", c.SP, c.BP, c.SI, c.DI)
	}
	for i, want := range vals {
		if got := c.getReg16(i); got != want {
			t.Errorf("getReg16(%d) = 0x%04X, want 0x%04X", i, got, want)
		}
	}
}

func TestGetSetSegReg(t *testing.T) {
	c, _ := newTestCPU()
	c.setSegReg(0, 0x1000) // ES
	c.setSegReg(1, 0x2000) // CS
	c.setSegReg(2, 0x3000) // SS
	c.setSegReg(3, 0x4000) // DS
	if c.ES != 0x1000 || c.CS != 0x2000 || c.SS != 0x3000 || c.DS != 0x4000 {
		t.Errorf("ES=%04X CS=%04X SS=%04X DS=%04X", c.ES, c.CS, c.SS, c.DS)
	}
}

func TestPush16Pop16(t *testing.T) {
	c, bus := newTestCPU()
	c.SP = 0x100
	c.push16(0x1234)
	if c.SP != 0xFE {
		t.Errorf("SP after push: got 0x%04X, want 0x00FE", c.SP)
	}
	if v := bus.Read16(c.SS, c.SP); v != 0x1234 {
		t.Errorf("stack value: got 0x%04X, want 0x1234", v)
	}
	val := c.pop16()
	if val != 0x1234 || c.SP != 0x100 {
		t.Errorf("pop16: got 0x%04X, SP=0x%04X", val, c.SP)
	}
}

func TestResolveModRM_DirectAddress(t *testing.T) {
	c, bus := newTestCPU()
	// mod=0, rm=6: direct 16-bit address
	// Place displacement 0x0100 at CS:IP
	bus.mem[0] = 0x00
	bus.mem[1] = 0x01
	c.IP = 0
	c.resolveModRM(0, 6)
	if c.modrmOff != 0x0100 {
		t.Errorf("modrmOff = 0x%04X, want 0x0100", c.modrmOff)
	}
	if c.IP != 2 {
		t.Errorf("IP = %d, want 2 (displacement consumed)", c.IP)
	}
}

func TestResolveModRM_BP_UsesSS(t *testing.T) {
	c, bus := newTestCPU()
	c.SS = 0x0010
	c.DS = 0x0020
	c.BP = 0x0050
	// mod=1, rm=6: [BP + disp8], default segment = SS
	bus.mem[0] = 0x04 // disp8 = 4
	c.IP = 0
	c.resolveModRM(1, 6)
	if c.modrmSeg != 0x0010 {
		t.Errorf("segment = 0x%04X, want SS=0x0010", c.modrmSeg)
	}
	if c.modrmOff != 0x0054 {
		t.Errorf("offset = 0x%04X, want BP+4=0x0054", c.modrmOff)
	}
}
