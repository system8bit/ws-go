package memory

// I/O port addresses for bank registers.
const (
	PortROMLinearBank = 0xC0 // Upper bits for ROM linear area (0x40000-0xEFFFF)
	PortSRAMBank      = 0xC1 // SRAM bank select
	PortROM0Bank      = 0xC2 // ROM bank 0 select (0x20000-0x2FFFF)
	PortROM1Bank      = 0xC3 // ROM bank 1 select (0x30000-0x3FFFF)
)

// ROMLastBank is a sentinel value passed to CartRead for accesses in the
// 0xF0000-0xFFFFF region. The cartridge implementation should map this to
// its highest bank number (totalBanks - 1).
const ROMLastBank = -1

// ROMBankSize is the size of a single ROM bank (64KB).
const ROMBankSize = 0x10000

// ComputeLastBank returns the index of the last 64KB bank for a ROM of
// the given total size in bytes. Returns 0 if romSize is smaller than one bank.
func ComputeLastBank(romSize int) int {
	banks := romSize / ROMBankSize
	if banks <= 0 {
		return 0
	}
	return banks - 1
}
