package cart

import "fmt"

// Header represents the WonderSwan ROM header, stored in the last 16 bytes of
// the ROM file.
type Header struct {
	DeveloperID   byte
	MinimumSystem byte // 0 = WS mono, 1 = WSC
	GameID        byte
	GameVersion   byte
	ROMSizeCode   byte // encoded ROM size
	SaveSizeCode  byte // 0=none, 1=64Kbit SRAM, 2=256Kbit SRAM, ...
	Flags         byte // bit0: orientation, bit2: ROM bus width
	RTC           byte // 1 = has RTC
	Checksum      uint16
}

// Orientation returns true if the cartridge is designed for vertical play.
func (h *Header) Orientation() bool {
	return h.Flags&0x01 != 0
}

// ROMBusWidth16 returns true if the ROM bus is 16-bit (bit 2 of flags).
func (h *Header) ROMBusWidth16() bool {
	return h.Flags&0x04 != 0
}

// HasRTC returns true if the cartridge contains a real-time clock.
func (h *Header) HasRTC() bool {
	return h.RTC == 1
}

// ROMSize returns the actual ROM size in bytes derived from the encoded
// rom_size field.
//
// Mapping:
//
//	0 = 128 KB   (1 Mbit)
//	1 = 256 KB   (2 Mbit)
//	2 = 512 KB   (4 Mbit)
//	3 = 1 MB     (8 Mbit)
//	4 = 2 MB     (16 Mbit)
//	5 = 3 MB     (24 Mbit)
//	6 = 4 MB     (32 Mbit)
//	7 = 6 MB     (48 Mbit)
//	8 = 8 MB     (64 Mbit)
//	9 = 16 MB    (128 Mbit)
func ROMSize(code byte) (int, error) {
	sizes := map[byte]int{
		0: 128 * 1024,
		1: 256 * 1024,
		2: 512 * 1024,
		3: 1 * 1024 * 1024,
		4: 2 * 1024 * 1024,
		5: 3 * 1024 * 1024,
		6: 4 * 1024 * 1024,
		7: 6 * 1024 * 1024,
		8: 8 * 1024 * 1024,
		9: 16 * 1024 * 1024,
	}
	s, ok := sizes[code]
	if !ok {
		return 0, fmt.Errorf("unknown ROM size code: %d", code)
	}
	return s, nil
}

// SaveSize returns the SRAM size in bytes for the given save_size code.
// Returns 0 for EEPROM-type codes (use EEPROMSize instead).
//
// Mapping:
//
//	0 = 0        (none)
//	1 = 8 KB     (64 Kbit)
//	2 = 32 KB    (256 Kbit)
//	3 = 128 KB   (1 Mbit)
//	4 = 256 KB   (2 Mbit)
//	5 = 512 KB   (4 Mbit)
func SaveSize(code byte) int {
	// EEPROM codes are handled separately
	if IsEEPROM(code) {
		return 0
	}
	sizes := map[byte]int{
		0: 0,
		1: 8 * 1024,
		2: 32 * 1024,
		3: 128 * 1024,
		4: 256 * 1024,
		5: 512 * 1024,
	}
	if s, ok := sizes[code]; ok {
		return s
	}
	return 0
}

// EEPROMSize returns the EEPROM size in bytes for the given save_size code.
// Returns 0 if the code indicates SRAM (not EEPROM).
//
// Mapping (Mednafen-verified):
//
//	0x10 = 128 bytes
//	0x20 = 2048 bytes (2 KB)
//	0x50 = 1024 bytes (1 KB)
func EEPROMSize(code byte) int {
	switch code {
	case 0x10:
		return 128
	case 0x20:
		return 2048
	case 0x50:
		return 1024
	default:
		return 0
	}
}

// IsEEPROM returns true if the save type is EEPROM rather than SRAM.
func IsEEPROM(code byte) bool {
	return EEPROMSize(code) > 0
}

// parseHeader reads the last 10 bytes of rom and populates a Header.
// WonderSwan ROM header layout (offset from end of ROM):
//
//	-10: Developer ID
//	 -9: Minimum system (0=WS, 1=WSC)
//	 -8: Game ID
//	 -7: Game version
//	 -6: ROM size code
//	 -5: Save size code
//	 -4: Flags
//	 -3: RTC
//	 -2: Checksum (16-bit LE)
// ParseHeaderExported is the exported wrapper for parseHeader (used in tests).
func ParseHeaderExported(rom []byte) *Header {
	return parseHeader(rom)
}

func parseHeader(rom []byte) *Header {
	n := len(rom)
	h := &Header{
		DeveloperID:   rom[n-10],
		MinimumSystem: rom[n-9],
		GameID:        rom[n-8],
		GameVersion:   rom[n-7],
		ROMSizeCode:   rom[n-6],
		SaveSizeCode:  rom[n-5],
		Flags:         rom[n-4],
		RTC:           rom[n-3],
		Checksum:      uint16(rom[n-2]) | uint16(rom[n-1])<<8,
	}
	return h
}
