package cart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// BankSize is the size of a single ROM bank (64 KB).
	BankSize = 64 * 1024
	// MinROMSize is the minimum accepted ROM size (64 KB).
	MinROMSize = 64 * 1024
	// HeaderSize is the size of the WonderSwan ROM header at the end of the file.
	HeaderSize = 16
)

// Cartridge holds the loaded ROM data, parsed header, and save data.
type Cartridge struct {
	ROM        []byte
	Header     *Header
	SRAM       []byte // SRAM save data (for SRAM-type cartridges)
	EEPROMData []byte // Game EEPROM data (for EEPROM-type cartridges)
	ROMPath    string // Path to the ROM file, used for .sav path derivation
}

// LoadROM reads a WonderSwan ROM file, validates it, parses the header, and
// returns a ready-to-use Cartridge.
func LoadROM(filename string) (*Cartridge, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("cart: failed to read ROM: %w", err)
	}

	size := len(data)
	if size < MinROMSize {
		return nil, fmt.Errorf("cart: ROM too small (%d bytes, minimum %d)", size, MinROMSize)
	}

	if !isValidROMSize(size) {
		return nil, fmt.Errorf("cart: invalid ROM size %d bytes", size)
	}

	header := parseHeader(data)

	c := &Cartridge{
		ROM:     data,
		Header:  header,
		ROMPath: filename,
	}

	// Allocate save storage based on header type
	if eepromSize := EEPROMSize(header.SaveSizeCode); eepromSize > 0 {
		c.EEPROMData = make([]byte, eepromSize)
	} else {
		sramSize := SaveSize(header.SaveSizeCode)
		c.SRAM = make([]byte, sramSize)
	}

	return c, nil
}

// ReadROM reads a byte from the ROM at the given bank and offset.
// Each bank is 64 KB. A bank value of -1 (ROMLastBank) maps to the last bank.
// The address wraps around if it exceeds the ROM size.
func (c *Cartridge) ReadROM(bank int, offset uint16) byte {
	if bank < 0 {
		// Last bank: map to end of ROM
		bank = len(c.ROM)/BankSize - 1
		if bank < 0 {
			bank = 0
		}
	}
	addr := bank*BankSize + int(offset)
	addr %= len(c.ROM)
	return c.ROM[addr]
}

// ReadSRAM reads a byte from SRAM. Returns 0xFF if no SRAM is present or the
// address is out of range.
func (c *Cartridge) ReadSRAM(addr uint16) byte {
	if len(c.SRAM) == 0 {
		return 0xFF
	}
	return c.SRAM[int(addr)%len(c.SRAM)]
}

// WriteSRAM writes a byte to SRAM. The write is silently ignored if no SRAM is
// present.
func (c *Cartridge) WriteSRAM(addr uint16, val byte) {
	if len(c.SRAM) == 0 {
		return
	}
	c.SRAM[int(addr)%len(c.SRAM)] = val
}

// IsColor returns true if the cartridge requires a WonderSwan Color system.
func (c *Cartridge) IsColor() bool {
	return c.Header.MinimumSystem == 1
}

// SaveData returns the game save data (EEPROM or SRAM).
// Returns nil if no save capability exists.
func (c *Cartridge) SaveData() []byte {
	if len(c.EEPROMData) > 0 {
		return c.EEPROMData
	}
	return c.SRAM
}

// HasSaveData returns true if the cartridge has any save capability.
func (c *Cartridge) HasSaveData() bool {
	return len(c.EEPROMData) > 0 || len(c.SRAM) > 0
}

// SavePath returns the .sav file path derived from the ROM path.
func (c *Cartridge) SavePath() string {
	ext := filepath.Ext(c.ROMPath)
	return strings.TrimSuffix(c.ROMPath, ext) + ".sav"
}

// LoadSave reads save data from the .sav file into the appropriate buffer.
// Does nothing if the file doesn't exist or the cartridge has no save capability.
func (c *Cartridge) LoadSave() error {
	if !c.HasSaveData() {
		return nil
	}
	data, err := os.ReadFile(c.SavePath())
	if err != nil {
		return nil // file doesn't exist, use fresh save
	}
	saveData := c.SaveData()
	if len(data) > len(saveData) {
		data = data[:len(saveData)]
	}
	copy(saveData, data)
	return nil
}

// WriteSave writes save data to the .sav file.
// Does nothing if the cartridge has no save capability.
func (c *Cartridge) WriteSave() error {
	if !c.HasSaveData() {
		return nil
	}
	return os.WriteFile(c.SavePath(), c.SaveData(), 0644)
}

// isValidROMSize checks whether size is a power of two or a known WonderSwan
// ROM size (some sizes like 3 MB and 6 MB are not powers of two).
func isValidROMSize(size int) bool {
	// Known non-power-of-two sizes accepted by WonderSwan.
	known := map[int]bool{
		768 * 1024:      true, // 6Mbit
		3 * 1024 * 1024: true,
		6 * 1024 * 1024: true,
	}
	if known[size] {
		return true
	}
	// Power-of-two check.
	return size > 0 && (size&(size-1)) == 0
}
