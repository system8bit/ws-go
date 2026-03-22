package cpu

// Interrupt dispatches a software or hardware interrupt.
// Pushes flags, CS, IP onto the stack, then loads the new CS:IP from the IVT.
// Clears IF and TF. Costs 32 cycles (Mednafen v30mz_int CLK(32)).
func (c *CPU) Interrupt(vector int) {
	c.Halted = false

	// Push flags, CS, IP
	c.push16(c.Flags)
	c.push16(c.CS)
	c.push16(c.IP)

	// Clear IF and TF
	c.SetFlag(FlagIF, false)
	c.SetFlag(FlagTF, false)
	c.InterruptEnable = false

	// Load new IP:CS from interrupt vector table at 0000:vector*4
	addr := uint16(vector * 4)
	c.IP = c.Bus.Read16(0x0000, addr)
	c.CS = c.Bus.Read16(0x0000, addr+2)

	// Interrupt dispatch costs 32 cycles (Mednafen-verified).
	// Use PendingCycles so they survive the next Step() reset.
	c.PendingCycles += 32
}
