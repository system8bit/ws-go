package ws

// EEPROM emulates the WonderSwan's internal and game EEPROM hardware.
//
// Two independent EEPROMs are supported:
//   - Internal EEPROM (iEEPROM): 1 KB, always present, ports 0xBA-0xBE.
//     Stores system settings (owner name, birth date, etc.).
//   - Game EEPROM (wsEEPROM): 128/1024/2048 bytes, ports 0xC4-0xC8.
//     Stores game save data when the cartridge uses EEPROM (not SRAM).
//
// Both use word-level addressing: the address register selects a 16-bit word,
// and the data ports access the low/high bytes of that word.
//
// Mednafen reference: src/wswan/eeprom.cpp, memory.cpp

const iEEPROMSize = 1024 // internal EEPROM is always 1 KB

// EEPROM represents both the internal and game EEPROM hardware.
type EEPROM struct {
	IData [iEEPROMSize]byte // internal EEPROM data
	GData []byte            // game EEPROM data (from cartridge, may be nil)

	iAddr uint16 // internal EEPROM address register
	gAddr uint16 // game EEPROM address register
	iCmd  byte   // internal EEPROM command/status register
	gCmd  byte   // game EEPROM command/status register
}

// State accessors for save/load
func (e *EEPROM) GetIAddr() uint16 { return e.iAddr }
func (e *EEPROM) SetIAddr(v uint16) { e.iAddr = v }
func (e *EEPROM) GetGAddr() uint16 { return e.gAddr }
func (e *EEPROM) SetGAddr(v uint16) { e.gAddr = v }
func (e *EEPROM) GetICmd() byte     { return e.iCmd }
func (e *EEPROM) SetICmd(v byte)    { e.iCmd = v }
func (e *EEPROM) GetGCmd() byte     { return e.gCmd }
func (e *EEPROM) SetGCmd(v byte)    { e.gCmd = v }

// NewEEPROM creates an EEPROM component. gameData should be the cartridge's
// EEPROMData slice (may be nil for SRAM-type or no-save cartridges).
func NewEEPROM(gameData []byte) *EEPROM {
	e := &EEPROM{
		GData: gameData,
	}
	e.initInternal()
	return e
}

// initInternal initializes the internal EEPROM with default system settings.
// Mednafen: WSwan_EEPROMInit() sets owner name, birth date, and other fields.
func (e *EEPROM) initInternal() {
	// Default owner name "WONDERSWAN" at offset 0x360 (custom character encoding)
	name := "WONDERSWAN"
	for i := 0; i < 16; i++ {
		if i < len(name) {
			e.IData[0x360+i] = encodeNameChar(name[i])
		}
	}

	// Birth year 2000 in BCD: 0x20, 0x00
	e.IData[0x370] = 0x20
	e.IData[0x371] = 0x00
	// Birth month 1 (January) in BCD
	e.IData[0x372] = 0x01
	// Birth day 1 in BCD
	e.IData[0x373] = 0x01
	// Sex: 0 (unspecified)
	e.IData[0x374] = 0x00
	// Blood type: 0 (unspecified)
	e.IData[0x375] = 0x00
}

// encodeNameChar converts an ASCII character to the WonderSwan internal
// EEPROM name encoding (Mednafen-verified).
func encodeNameChar(ch byte) byte {
	switch {
	case ch == ' ':
		return 0x00
	case ch >= '0' && ch <= '9':
		return ch - '0' + 0x01
	case ch >= 'A' && ch <= 'Z':
		return ch - 'A' + 0x0B
	case ch >= 'a' && ch <= 'z':
		return ch - 'a' + 0x25
	default:
		return 0x00
	}
}

// ReadPort handles reads from EEPROM I/O ports (0xBA-0xBE and 0xC4-0xC8).
func (e *EEPROM) ReadPort(port byte) byte {
	switch port {
	// Internal EEPROM
	case 0xBA:
		return e.readDataLow(e.IData[:], e.iAddr, iEEPROMSize)
	case 0xBB:
		return e.readDataHigh(e.IData[:], e.iAddr, iEEPROMSize)
	case 0xBC:
		return byte(e.iAddr)
	case 0xBD:
		return byte(e.iAddr >> 8)
	case 0xBE:
		return e.readStatus(e.iCmd)

	// Game EEPROM
	case 0xC4:
		if len(e.GData) == 0 {
			return 0
		}
		return e.readDataLow(e.GData, e.gAddr, len(e.GData))
	case 0xC5:
		if len(e.GData) == 0 {
			return 0
		}
		return e.readDataHigh(e.GData, e.gAddr, len(e.GData))
	case 0xC6:
		return byte(e.gAddr)
	case 0xC7:
		return byte(e.gAddr >> 8)
	case 0xC8:
		return e.readStatus(e.gCmd)
	}
	return 0
}

// WritePort handles writes to EEPROM I/O ports (0xBA-0xBE and 0xC4-0xC8).
func (e *EEPROM) WritePort(port byte, val byte) {
	switch port {
	// Internal EEPROM
	case 0xBA:
		e.writeDataLow(e.IData[:], e.iAddr, iEEPROMSize, val)
	case 0xBB:
		e.writeDataHigh(e.IData[:], e.iAddr, iEEPROMSize, val)
	case 0xBC:
		e.iAddr = (e.iAddr & 0xFF00) | uint16(val)
	case 0xBD:
		e.iAddr = (e.iAddr & 0x00FF) | (uint16(val) << 8)
	case 0xBE:
		e.iCmd = val

	// Game EEPROM
	case 0xC4:
		if len(e.GData) > 0 {
			e.writeDataLow(e.GData, e.gAddr, len(e.GData), val)
		}
	case 0xC5:
		if len(e.GData) > 0 {
			e.writeDataHigh(e.GData, e.gAddr, len(e.GData), val)
		}
	case 0xC6:
		e.gAddr = (e.gAddr & 0xFF00) | uint16(val)
	case 0xC7:
		e.gAddr = (e.gAddr & 0x00FF) | (uint16(val) << 8)
	case 0xC8:
		e.gCmd = val
	}
}

// readDataLow reads the low byte of the word at the current address.
// Address is word-based: byte offset = address << 1.
func (e *EEPROM) readDataLow(data []byte, addr uint16, size int) byte {
	byteAddr := int(addr<<1) & (size - 1)
	if byteAddr < len(data) {
		return data[byteAddr]
	}
	return 0
}

// readDataHigh reads the high byte of the word at the current address.
func (e *EEPROM) readDataHigh(data []byte, addr uint16, size int) byte {
	byteAddr := (int(addr<<1) | 1) & (size - 1)
	if byteAddr < len(data) {
		return data[byteAddr]
	}
	return 0
}

// writeDataLow writes the low byte of the word at the current address.
func (e *EEPROM) writeDataLow(data []byte, addr uint16, size int, val byte) {
	byteAddr := int(addr<<1) & (size - 1)
	if byteAddr < len(data) {
		data[byteAddr] = val
	}
}

// writeDataHigh writes the high byte of the word at the current address.
func (e *EEPROM) writeDataHigh(data []byte, addr uint16, size int, val byte) {
	byteAddr := (int(addr<<1) | 1) & (size - 1)
	if byteAddr < len(data) {
		data[byteAddr] = val
	}
}

// readStatus returns the command/status register value.
// Mednafen: status bits 0-1 indicate operation readiness.
// Since we implement instant access, operations are always complete.
func (e *EEPROM) readStatus(cmd byte) byte {
	if cmd&0x20 != 0 {
		return cmd | 0x02 // write/erase complete
	}
	if cmd&0x10 != 0 {
		return cmd | 0x01 // read complete
	}
	return cmd | 0x03 // idle/ready
}

// Reset clears command and address registers (data is preserved).
func (e *EEPROM) Reset() {
	e.iAddr = 0
	e.gAddr = 0
	e.iCmd = 0
	e.gCmd = 0
}
