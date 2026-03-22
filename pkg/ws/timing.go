package ws

// Frame and scanline timing constants for the WonderSwan.
const (
	CyclesPerLine  = 256               // CPU cycles executed per scanline
	VisibleLines   = 144               // Number of visible (rendered) scanlines
	TotalLines     = 159               // Total scanlines per frame (visible + VBlank)
	CyclesPerFrame = CyclesPerLine * TotalLines // 40704 cycles per frame
)

// Interrupt vector assignments relative to the base stored in I/O 0xB0.
// The bit position in the IntEnable/IntStatus registers matches the IRQ number.
const (
	IRQSerialTX    = 0 // Bit 0: Serial transmit complete
	IRQKeyPress    = 1 // Bit 1: Key press
	IRQCartridge   = 2 // Bit 2: Cartridge IRQ (RTC alarm)
	IRQSerialRX    = 3 // Bit 3: Serial receive ready
	IRQLineMatch   = 4 // Bit 4: Scanline compare match
	IRQVBlankTimer = 5 // Bit 5: VBlank timer / sprite complete
	IRQVBlank      = 6 // Bit 6: VBlank start
	IRQHBlankTimer = 7 // Bit 7: HBlank timer countdown
)

// Interrupt I/O port addresses.
const (
	IOIntBase   = 0xB0 // Interrupt vector base number
	IOIntEnable = 0xB2 // Interrupt enable mask
	IOIntAck    = 0xB4 // Interrupt acknowledge (write 1 to clear)
	IOIntStatus = 0xB6 // Interrupt status (read)
)
