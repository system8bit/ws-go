package cpu

import "fmt"

// BusInterface defines how the CPU communicates with memory and I/O.
type BusInterface interface {
	Read8(seg, offset uint16) byte
	Write8(seg, offset uint16, val byte)
	Read16(seg, offset uint16) uint16
	Write16(seg, offset uint16, val uint16)
	IORead(port uint8) byte
	IOWrite(port uint8, val byte)
}

// CPU represents the V30MZ processor (NEC 80186-compatible) running at 3.072 MHz.
type CPU struct {
	// General purpose registers (16-bit, accessible as 8-bit pairs)
	AX, BX, CX, DX uint16

	// Index and pointer registers
	SI, DI, SP, BP uint16

	// Segment registers
	CS, DS, ES, SS uint16

	// Instruction pointer
	IP uint16

	// Flags register
	Flags uint16

	// Internal state
	Halted        bool
	Cycles        int    // cycles consumed in current step
	PendingCycles int    // cycles from interrupt dispatch, added to next Step()
	TotalCycles   uint64

	// Segment override (-1 = none, 0=ES, 1=CS, 2=SS, 3=DS)
	segOverride int
	// REP prefix: 0=none, 1=REP/REPE, 2=REPNE
	repPrefix int

	// Cached ModR/M address (set by resolveModRM)
	modrmSeg uint16
	modrmOff uint16

	// IP at the start of the current instruction (before prefixes),
	// used by REP to rewind for interrupt interleave.
	instrStartIP uint16

	// Bus interface for memory and I/O access
	Bus BusInterface

	// Interrupt state
	InterruptEnable bool
	PendingIRQ      int // -1 = no pending, 0-255 = vector number
}

// New creates a new V30MZ CPU connected to the given bus and resets it.
func New(bus BusInterface) *CPU {
	c := &CPU{
		Bus: bus,
	}
	c.Reset()
	return c
}

// Reset sets the CPU to its initial power-on state.
// CS=0xFFFF, IP=0x0000, all other registers zeroed, interrupts disabled.
func (c *CPU) Reset() {
	c.AX = 0
	c.BX = 0
	c.CX = 0
	c.DX = 0
	c.SI = 0
	c.DI = 0
	c.SP = 0
	c.BP = 0
	c.CS = 0xFFFF
	c.DS = 0
	c.ES = 0
	c.SS = 0
	c.IP = 0x0000
	c.Flags = 0
	c.Halted = false
	c.Cycles = 0
	c.TotalCycles = 0
	c.segOverride = -1
	c.repPrefix = 0
	c.InterruptEnable = false
	c.PendingIRQ = -1
}

// Step executes one instruction and returns the number of cycles consumed.
// PendingCycles from interrupt dispatch are included in the return value.
func (c *CPU) Step() int {
	// Include any pending cycles from interrupt dispatch (32 cycles per Interrupt() call)
	c.Cycles = c.PendingCycles
	c.PendingCycles = 0

	if c.Halted {
		// Halted CPU consumes no additional cycles per step;
		// the system loop handles burning remaining cycles.
		if c.Cycles == 0 {
			c.Cycles = 1
		}
		c.TotalCycles += uint64(c.Cycles)
		return c.Cycles
	}

	// Reset prefix state
	c.segOverride = -1
	c.repPrefix = 0

	// Save TF state before instruction execution.
	// Single-step trap fires AFTER the instruction if TF was set BEFORE it.
	tfBefore := c.GetFlag(FlagTF)

	// Execute one instruction (handles prefix loops internally)
	c.executeInstruction()

	// Single-step trap: if TF was set before the instruction, fire INT 1.
	// Interrupt() clears TF and IF, preventing recursive traps.
	if tfBefore {
		c.Interrupt(1)
	}

	c.TotalCycles += uint64(c.Cycles)
	return c.Cycles
}

// --- 8-bit register accessors ---

func (c *CPU) AL() byte        { return byte(c.AX) }
func (c *CPU) AH() byte        { return byte(c.AX >> 8) }
func (c *CPU) BL() byte        { return byte(c.BX) }
func (c *CPU) BH() byte        { return byte(c.BX >> 8) }
func (c *CPU) CL() byte        { return byte(c.CX) }
func (c *CPU) CH() byte        { return byte(c.CX >> 8) }
func (c *CPU) DL() byte        { return byte(c.DX) }
func (c *CPU) DH() byte        { return byte(c.DX >> 8) }

func (c *CPU) SetAL(v byte)    { c.AX = (c.AX & 0xFF00) | uint16(v) }
func (c *CPU) SetAH(v byte)    { c.AX = (c.AX & 0x00FF) | (uint16(v) << 8) }
func (c *CPU) SetBL(v byte)    { c.BX = (c.BX & 0xFF00) | uint16(v) }
func (c *CPU) SetBH(v byte)    { c.BX = (c.BX & 0x00FF) | (uint16(v) << 8) }
func (c *CPU) SetCL(v byte)    { c.CX = (c.CX & 0xFF00) | uint16(v) }
func (c *CPU) SetCH(v byte)    { c.CX = (c.CX & 0x00FF) | (uint16(v) << 8) }
func (c *CPU) SetDL(v byte)    { c.DX = (c.DX & 0xFF00) | uint16(v) }
func (c *CPU) SetDH(v byte)    { c.DX = (c.DX & 0x00FF) | (uint16(v) << 8) }

// String returns a debug representation of the CPU state.
func (c *CPU) String() string {
	return fmt.Sprintf(
		"AX=%04X BX=%04X CX=%04X DX=%04X SI=%04X DI=%04X SP=%04X BP=%04X\n"+
			"CS=%04X DS=%04X ES=%04X SS=%04X IP=%04X FL=%04X [%s]",
		c.AX, c.BX, c.CX, c.DX, c.SI, c.DI, c.SP, c.BP,
		c.CS, c.DS, c.ES, c.SS, c.IP, c.Flags, c.flagsString(),
	)
}

func (c *CPU) flagsString() string {
	s := ""
	if c.GetFlag(FlagOF) {
		s += "O"
	} else {
		s += "-"
	}
	if c.GetFlag(FlagDF) {
		s += "D"
	} else {
		s += "-"
	}
	if c.GetFlag(FlagIF) {
		s += "I"
	} else {
		s += "-"
	}
	if c.GetFlag(FlagSF) {
		s += "S"
	} else {
		s += "-"
	}
	if c.GetFlag(FlagZF) {
		s += "Z"
	} else {
		s += "-"
	}
	if c.GetFlag(FlagAF) {
		s += "A"
	} else {
		s += "-"
	}
	if c.GetFlag(FlagPF) {
		s += "P"
	} else {
		s += "-"
	}
	if c.GetFlag(FlagCF) {
		s += "C"
	} else {
		s += "-"
	}
	return s
}
