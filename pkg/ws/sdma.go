package ws

import (
	"github.com/system8bit/ws-go/pkg/apu"
	"github.com/system8bit/ws-go/pkg/memory"
)

// SoundDMA implements the WonderSwan's Sound DMA engine (ports 0x4A-0x52).
// It transfers data from ROM to an APU port at a configurable rate.
//
// Control register (port 0x52) bits:
//   Bit 7: Enable
//   Bit 6: Direction (0=increment, 1=decrement source address)
//   Bit 4: Target (0=port 0x89 CH2 volume, 1=port 0x95 HyperVoice)
//   Bit 3: Auto-reload (loop when length reaches 0)
//   Bits 0-1: Rate (0=4kHz, 1=6kHz, 2=12kHz, 3=24kHz)
type SoundDMA struct {
	Source      uint32 // 20-bit source address
	Length      uint32 // 20-bit transfer length
	SourceSaved uint32 // saved source for auto-reload
	LengthSaved uint32 // saved length for auto-reload
	Control     byte   // control register
	Timer       byte   // countdown timer for rate division
}

// Reset clears all Sound DMA state.
func (sd *SoundDMA) Reset() {
	sd.Source = 0
	sd.Length = 0
	sd.SourceSaved = 0
	sd.LengthSaved = 0
	sd.Control = 0
	sd.Timer = 0
}

// WritePort handles writes to Sound DMA I/O ports (0x4A-0x52).
func (sd *SoundDMA) WritePort(port byte, val byte) {
	switch port {
	case 0x4A:
		sd.Source = (sd.Source & 0xFFF00) | uint32(val)
		sd.SourceSaved = sd.Source
	case 0x4B:
		sd.Source = (sd.Source & 0xF00FF) | (uint32(val) << 8)
		sd.SourceSaved = sd.Source
	case 0x4C:
		sd.Source = (sd.Source & 0x0FFFF) | (uint32(val&0x0F) << 16)
		sd.SourceSaved = sd.Source
	case 0x4E:
		sd.Length = (sd.Length & 0xFFF00) | uint32(val)
		sd.LengthSaved = sd.Length
	case 0x4F:
		sd.Length = (sd.Length & 0xF00FF) | (uint32(val) << 8)
		sd.LengthSaved = sd.Length
	case 0x50:
		sd.Length = (sd.Length & 0x0FFFF) | (uint32(val&0x0F) << 16)
		sd.LengthSaved = sd.Length
	case 0x52:
		sd.Control = val &^ 0x20 // bit 5 is always 0 (Mednafen)
	}
}

// ReadPort handles reads from Sound DMA I/O ports (0x4A-0x52).
func (sd *SoundDMA) ReadPort(port byte) byte {
	switch port {
	case 0x4A:
		return byte(sd.Source)
	case 0x4B:
		return byte(sd.Source >> 8)
	case 0x4C:
		return byte(sd.Source >> 16)
	case 0x4E:
		return byte(sd.Length)
	case 0x4F:
		return byte(sd.Length >> 8)
	case 0x50:
		return byte(sd.Length >> 16)
	case 0x52:
		return sd.Control
	}
	return 0
}

// Check performs one Sound DMA tick (called ~every 128 CPU cycles = 24kHz).
// Transfers one byte from ROM to the APU when the timer expires.
func (sd *SoundDMA) Check(bus *memory.Bus, apuUnit *apu.APU) {
	if sd.Control&0x80 == 0 {
		return
	}

	if sd.Timer > 0 {
		sd.Timer--
		return
	}

	// Transfer one byte
	data := bus.ReadLinear(sd.Source)

	if sd.Control&0x10 != 0 {
		// Target: port 0x95 (HyperVoice)
		apuUnit.WritePort(0x95, data)
	} else {
		// Target: port 0x89 (CH2 volume / Voice D/A)
		apuUnit.WritePort(0x89, data)
	}

	// Advance source address
	if sd.Control&0x40 != 0 {
		sd.Source--
	} else {
		sd.Source++
	}
	sd.Source &= 0x000FFFFF

	// Decrement length
	sd.Length--
	sd.Length &= 0x000FFFFF

	if sd.Length == 0 {
		if sd.Control&0x08 != 0 {
			// Auto-reload
			sd.Length = sd.LengthSaved
			sd.Source = sd.SourceSaved
		} else {
			// Stop
			sd.Control &^= 0x80
		}
	}

	// Reload timer based on rate selection (bits 0-1)
	// Rate divides the 24kHz base: 0→/6=4kHz, 1→/4=6kHz, 2→/2=12kHz, 3→/1=24kHz
	switch sd.Control & 3 {
	case 0:
		sd.Timer = 5
	case 1:
		sd.Timer = 3
	case 2:
		sd.Timer = 1
	case 3:
		sd.Timer = 0
	}
}
