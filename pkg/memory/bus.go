package memory

// Bus represents the WonderSwan 20-bit memory bus (1MB address space).
//
// Memory map:
//   0x00000-0x0FFFF  Internal RAM (16KB mirrored in mono, 64KB in color)
//   0x10000-0x1FFFF  SRAM (cartridge save, banked via SRAMBank)
//   0x20000-0x2FFFF  ROM bank 0 (via ROM0Bank register, I/O 0xC2)
//   0x30000-0x3FFFF  ROM bank 1 (via ROM1Bank register, I/O 0xC3)
//   0x40000-0xEFFFF  ROM linear area (via ROMLinearBank register, I/O 0xC0)
//   0xF0000-0xFFFFF  ROM last bank (always maps to last 64KB of ROM)
type Bus struct {
	IRAM    []byte // 64KB max (only first 16KB used in mono mode)
	IsColor bool

	// Bank registers (updated via I/O ports 0xC0-0xC3)
	ROMLinearBank byte // I/O 0xC0 — upper address bits for 0x40000-0xEFFFF
	SRAMBank      byte // I/O 0xC1
	ROM0Bank      byte // I/O 0xC2 — bank for 0x20000-0x2FFFF
	ROM1Bank      byte // I/O 0xC3 — bank for 0x30000-0x3FFFF

	// I/O port space (256 bytes, addresses 0x00-0xFF)
	IOPorts [256]byte

	// Cartridge callbacks (set during system init)
	CartRead      func(bank int, addr uint16) byte
	CartWriteSRAM func(addr uint16, val byte)
	CartReadSRAM  func(addr uint16) byte

	// I/O hook callbacks for PPU, APU, input, etc.
	IOReadHook  func(port uint8) byte
	IOWriteHook func(port uint8, val byte)
}

const (
	iramSize      = 0x10000 // 64KB
	iramMonoMask  = 0x3FFF  // 16KB - 1, for mirroring
	addrSpaceMask = 0xFFFFF // 20-bit address space
)

// NewBus creates a new memory bus. If isColor is false, IRAM is 64KB allocated
// but only the lower 16KB are unique (reads/writes mirror).
func NewBus(isColor bool) *Bus {
	b := &Bus{
		IRAM:    make([]byte, iramSize),
		IsColor: isColor,
	}
	return b
}

// linearAddress computes a 20-bit linear address from segment:offset.
func linearAddress(seg uint16, offset uint16) uint32 {
	return (uint32(seg)<<4 + uint32(offset)) & addrSpaceMask
}

// Read8 reads a byte using segment:offset addressing.
func (b *Bus) Read8(seg uint16, offset uint16) byte {
	return b.ReadLinear(linearAddress(seg, offset))
}

// Write8 writes a byte using segment:offset addressing.
func (b *Bus) Write8(seg uint16, offset uint16, val byte) {
	b.WriteLinear(linearAddress(seg, offset), val)
}

// Read16 reads a little-endian 16-bit word at segment:offset.
func (b *Bus) Read16(seg uint16, offset uint16) uint16 {
	addr := linearAddress(seg, offset)
	lo := uint16(b.ReadLinear(addr))
	hi := uint16(b.ReadLinear((addr + 1) & addrSpaceMask))
	return hi<<8 | lo
}

// Write16 writes a little-endian 16-bit word at segment:offset.
func (b *Bus) Write16(seg uint16, offset uint16, val uint16) {
	addr := linearAddress(seg, offset)
	b.WriteLinear(addr, byte(val))
	b.WriteLinear((addr+1)&addrSpaceMask, byte(val>>8))
}

// ReadLinear reads a byte from a 20-bit linear address.
func (b *Bus) ReadLinear(addr uint32) byte {
	addr &= addrSpaceMask

	switch {
	// 0x00000-0x0FFFF: Internal RAM
	case addr < 0x10000:
		return b.readIRAM(uint16(addr))

	// 0x10000-0x1FFFF: SRAM (cartridge save)
	case addr < 0x20000:
		if b.CartReadSRAM != nil {
			offset := uint16(addr & 0xFFFF)
			return b.CartReadSRAM(uint16(uint32(b.SRAMBank)<<16 | uint32(offset)))
		}
		return 0xFF

	// 0x20000-0x2FFFF: ROM bank 0
	case addr < 0x30000:
		if b.CartRead != nil {
			return b.CartRead(int(b.ROM0Bank), uint16(addr&0xFFFF))
		}
		return 0xFF

	// 0x30000-0x3FFFF: ROM bank 1
	case addr < 0x40000:
		if b.CartRead != nil {
			return b.CartRead(int(b.ROM1Bank), uint16(addr&0xFFFF))
		}
		return 0xFF

	// 0x40000-0xFFFFF: ROM linear area (Mednafen-compatible)
	// bank 4..F map to ROMLinearBank+0 .. ROMLinearBank+11
	default:
		if b.CartRead != nil {
			bank := int(b.ROMLinearBank) + int((addr-0x40000)>>16)
			return b.CartRead(bank, uint16(addr&0xFFFF))
		}
		return 0xFF
	}
}

// WriteLinear writes a byte to a 20-bit linear address.
func (b *Bus) WriteLinear(addr uint32, val byte) {
	addr &= addrSpaceMask

	switch {
	// 0x00000-0x0FFFF: Internal RAM
	case addr < 0x10000:
		b.writeIRAM(uint16(addr), val)

	// 0x10000-0x1FFFF: SRAM (cartridge save)
	case addr < 0x20000:
		if b.CartWriteSRAM != nil {
			offset := uint16(addr & 0xFFFF)
			b.CartWriteSRAM(uint16(uint32(b.SRAMBank)<<16|uint32(offset)), val)
		}

	// 0x20000-0xFFFFF: ROM area — writes are ignored
	default:
		// ROM is read-only; writes are silently dropped.
	}
}

// readIRAM reads from internal RAM, applying mirroring in mono mode.
func (b *Bus) readIRAM(offset uint16) byte {
	if !b.IsColor {
		offset &= iramMonoMask
	}
	return b.IRAM[offset]
}

// writeIRAM writes to internal RAM, applying mirroring in mono mode.
func (b *Bus) writeIRAM(offset uint16, val byte) {
	if !b.IsColor {
		offset &= iramMonoMask
	}
	b.IRAM[offset] = val
}
