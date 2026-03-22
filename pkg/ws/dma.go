package ws

import "github.com/system8bit/ws-go/pkg/memory"

// DMA holds state for the WonderSwan's general-purpose and sound DMA engines.
type DMA struct {
	// General purpose DMA
	SrcAddr uint32 // 20-bit source (I/O 0xA0-0xA3)
	DstAddr uint16 // 16-bit dest in IRAM (I/O 0xA4-0xA5)
	Length  uint16 // 16-bit transfer length (I/O 0xA6-0xA7)
	Control byte   // I/O 0xA8: bit 7 = start

	// Sound DMA
	SndSrcAddr uint32
	SndLength  uint32
	SndControl byte
}

// Reset clears all DMA state.
func (d *DMA) Reset() {
	d.SrcAddr = 0
	d.DstAddr = 0
	d.Length = 0
	d.Control = 0
	d.SndSrcAddr = 0
	d.SndLength = 0
	d.SndControl = 0
}

// WritePort handles writes to DMA I/O ports (0x40-0x48).
func (d *DMA) WritePort(port byte, val byte) {
	switch port {
	case 0x40:
		d.SrcAddr = (d.SrcAddr & 0xFFF00) | uint32(val)
	case 0x41:
		d.SrcAddr = (d.SrcAddr & 0xF00FF) | (uint32(val) << 8)
	case 0x42:
		d.SrcAddr = (d.SrcAddr & 0x0FFFF) | (uint32(val&0x0F) << 16)
	case 0x43:
		// unused high bits
	case 0x44:
		d.DstAddr = (d.DstAddr & 0xFF00) | uint16(val)
	case 0x45:
		d.DstAddr = (d.DstAddr & 0x00FF) | (uint16(val) << 8)
	case 0x46:
		d.Length = (d.Length & 0xFF00) | uint16(val)
	case 0x47:
		d.Length = (d.Length & 0x00FF) | (uint16(val) << 8)
	case 0x48:
		d.Control = val
	}
}

// ReadPort handles reads from DMA I/O ports (0x40-0x48).
func (d *DMA) ReadPort(port byte) byte {
	switch port {
	case 0x40:
		return byte(d.SrcAddr)
	case 0x41:
		return byte(d.SrcAddr >> 8)
	case 0x42:
		return byte(d.SrcAddr >> 16)
	case 0x44:
		return byte(d.DstAddr)
	case 0x45:
		return byte(d.DstAddr >> 8)
	case 0x46:
		return byte(d.Length)
	case 0x47:
		return byte(d.Length >> 8)
	case 0x48:
		return d.Control
	}
	return 0
}

// Execute performs the DMA transfer if bit 7 of Control is set.
// Reads from the bus (ROM/SRAM) and writes to IRAM.
func (d *DMA) Execute(bus *memory.Bus) {
	if d.Control&0x80 == 0 {
		return
	}

	length := d.Length
	if length == 0 {
		d.Control &^= 0x80 // clear start bit
		return
	}

	src := d.SrcAddr
	dst := d.DstAddr

	// Direction: bit 6 controls increment direction
	// 0 = increment, 1 = decrement
	inc := int32(1)
	if d.Control&0x40 != 0 {
		inc = -1
	}

	for i := uint16(0); i < length; i++ {
		val := bus.ReadLinear(src)
		// Write directly to IRAM (destination is always IRAM)
		if int(dst) < len(bus.IRAM) {
			bus.IRAM[dst] = val
		}
		src = (src + uint32(inc)) & 0xFFFFF
		dst += uint16(inc)
	}

	// Update registers after transfer
	d.SrcAddr = src
	d.DstAddr = dst
	d.Length = 0
	d.Control &^= 0x80 // clear start bit
}
