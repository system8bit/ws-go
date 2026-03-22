package ws

import (
	"bytes"
	"testing"

	"github.com/system8bit/ws-go/pkg/cart"
)

// makeTestSystem creates a minimal System for testing (no ROM file needed).
func makeTestSystem() *System {
	// Create a minimal 64KB ROM with valid header at the end.
	rom := make([]byte, 65536)
	// ROM header at last 10 bytes: jmp far 0xFFFF:0x0000
	rom[0xFFF0] = 0xEA       // JMP FAR
	rom[0xFFF1] = 0x00       // IP low
	rom[0xFFF2] = 0x00       // IP high
	rom[0xFFF3] = 0xFF       // CS low
	rom[0xFFF4] = 0xFF       // CS high
	rom[0xFFF5] = 0x00       // maintenance
	rom[0xFFF6] = 0x01       // developer
	rom[0xFFF7] = 0x00       // minimum system (WS mono)
	rom[0xFFF8] = 0x00       // cart number
	rom[0xFFF9] = 0x00       // ROM size
	rom[0xFFFA] = 0x00       // save size
	rom[0xFFFB] = 0x00       // flags
	rom[0xFFFC] = 0x00       // reserved
	rom[0xFFFD] = 0x00       // reserved
	rom[0xFFFE] = 0x00       // checksum low
	rom[0xFFFF] = 0x00       // checksum high

	c := &cart.Cartridge{
		ROM: rom,
	}
	c.Header = cart.ParseHeaderExported(rom)
	return New(c)
}

func TestSaveLoadRoundTrip(t *testing.T) {
	sys := makeTestSystem()

	// Set up some non-default state across all components.
	sys.CPU.AX = 0x1234
	sys.CPU.BX = 0x5678
	sys.CPU.CX = 0x9ABC
	sys.CPU.DX = 0xDEF0
	sys.CPU.SI = 0x1111
	sys.CPU.DI = 0x2222
	sys.CPU.SP = 0x3333
	sys.CPU.BP = 0x4444
	sys.CPU.CS = 0xF000
	sys.CPU.DS = 0x0100
	sys.CPU.ES = 0x0200
	sys.CPU.SS = 0x0300
	sys.CPU.IP = 0x0042
	sys.CPU.Flags = 0xF246
	sys.CPU.Halted = true
	sys.CPU.InterruptEnable = true
	sys.CPU.PendingIRQ = 5
	sys.CPU.TotalCycles = 123456789

	sys.Bus.IRAM[0] = 0xAA
	sys.Bus.IRAM[0xFFFF] = 0xBB
	sys.Bus.IOPorts[0x00] = 0x3F
	sys.Bus.IOPorts[0xB6] = 0x40
	sys.Bus.ROMLinearBank = 3
	sys.Bus.SRAMBank = 1
	sys.Bus.ROM0Bank = 7
	sys.Bus.ROM1Bank = 11

	sys.PPU.DispCtrl = 0x07
	sys.PPU.MapBase = 0x76
	sys.PPU.Scanline = 42
	sys.PPU.ShadeLUT = [8]byte{0, 2, 5, 6, 8, 10, 13, 15}

	sys.Timer.Control = 0x05
	sys.Timer.HBlankPreset = 300
	sys.Timer.HBlankCount = 150
	sys.Timer.VBlankPreset = 75
	sys.Timer.VBlankCount = 30

	sys.APU.SoundCtrl = 0x22
	sys.APU.NoiseLFSR = 0x1234

	sys.Serial.Control = 0x80
	sys.Serial.SendBuf = 0x42

	// Save to buffer
	var buf bytes.Buffer
	if err := sys.SaveToWriter(&buf); err != nil {
		t.Fatalf("SaveToWriter: %v", err)
	}
	savedSize := buf.Len()
	t.Logf("Save state size: %d bytes", savedSize)

	if savedSize < 1000 {
		t.Errorf("Save state too small: %d bytes", savedSize)
	}

	// Create a fresh system and load the state
	sys2 := makeTestSystem()
	if err := sys2.LoadFromReader(&buf); err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	// Verify CPU
	if sys2.CPU.AX != 0x1234 {
		t.Errorf("CPU.AX = 0x%04X, want 0x1234", sys2.CPU.AX)
	}
	if sys2.CPU.BX != 0x5678 {
		t.Errorf("CPU.BX = 0x%04X, want 0x5678", sys2.CPU.BX)
	}
	if sys2.CPU.SP != 0x3333 {
		t.Errorf("CPU.SP = 0x%04X, want 0x3333", sys2.CPU.SP)
	}
	if sys2.CPU.Flags != 0xF246 {
		t.Errorf("CPU.Flags = 0x%04X, want 0xF246", sys2.CPU.Flags)
	}
	if sys2.CPU.Halted != true {
		t.Errorf("CPU.Halted = %v, want true", sys2.CPU.Halted)
	}
	if sys2.CPU.InterruptEnable != true {
		t.Errorf("CPU.InterruptEnable = %v, want true", sys2.CPU.InterruptEnable)
	}
	if sys2.CPU.PendingIRQ != 5 {
		t.Errorf("CPU.PendingIRQ = %d, want 5", sys2.CPU.PendingIRQ)
	}
	if sys2.CPU.TotalCycles != 123456789 {
		t.Errorf("CPU.TotalCycles = %d, want 123456789", sys2.CPU.TotalCycles)
	}

	// Verify Bus
	if sys2.Bus.IRAM[0] != 0xAA {
		t.Errorf("Bus.IRAM[0] = 0x%02X, want 0xAA", sys2.Bus.IRAM[0])
	}
	if sys2.Bus.IRAM[0xFFFF] != 0xBB {
		t.Errorf("Bus.IRAM[0xFFFF] = 0x%02X, want 0xBB", sys2.Bus.IRAM[0xFFFF])
	}
	if sys2.Bus.IOPorts[0x00] != 0x3F {
		t.Errorf("Bus.IOPorts[0] = 0x%02X, want 0x3F", sys2.Bus.IOPorts[0x00])
	}
	if sys2.Bus.ROMLinearBank != 3 {
		t.Errorf("Bus.ROMLinearBank = %d, want 3", sys2.Bus.ROMLinearBank)
	}
	if sys2.Bus.ROM0Bank != 7 {
		t.Errorf("Bus.ROM0Bank = %d, want 7", sys2.Bus.ROM0Bank)
	}

	// Verify PPU
	if sys2.PPU.DispCtrl != 0x07 {
		t.Errorf("PPU.DispCtrl = 0x%02X, want 0x07", sys2.PPU.DispCtrl)
	}
	if sys2.PPU.MapBase != 0x76 {
		t.Errorf("PPU.MapBase = 0x%02X, want 0x76", sys2.PPU.MapBase)
	}
	if sys2.PPU.Scanline != 42 {
		t.Errorf("PPU.Scanline = %d, want 42", sys2.PPU.Scanline)
	}
	if sys2.PPU.ShadeLUT != [8]byte{0, 2, 5, 6, 8, 10, 13, 15} {
		t.Errorf("PPU.ShadeLUT = %v, want bad_apple values", sys2.PPU.ShadeLUT)
	}

	// Verify Timer
	if sys2.Timer.Control != 0x05 {
		t.Errorf("Timer.Control = 0x%02X, want 0x05", sys2.Timer.Control)
	}
	if sys2.Timer.HBlankCount != 150 {
		t.Errorf("Timer.HBlankCount = %d, want 150", sys2.Timer.HBlankCount)
	}

	// Verify APU
	if sys2.APU.SoundCtrl != 0x22 {
		t.Errorf("APU.SoundCtrl = 0x%02X, want 0x22", sys2.APU.SoundCtrl)
	}
	if sys2.APU.NoiseLFSR != 0x1234 {
		t.Errorf("APU.NoiseLFSR = 0x%04X, want 0x1234", sys2.APU.NoiseLFSR)
	}

	// Verify Serial
	if sys2.Serial.Control != 0x80 {
		t.Errorf("Serial.Control = 0x%02X, want 0x80", sys2.Serial.Control)
	}
	if sys2.Serial.SendBuf != 0x42 {
		t.Errorf("Serial.SendBuf = 0x%02X, want 0x42", sys2.Serial.SendBuf)
	}
}

func TestSaveLoadPreservesIRAM(t *testing.T) {
	sys := makeTestSystem()

	// Write a known pattern to IRAM
	for i := range sys.Bus.IRAM {
		sys.Bus.IRAM[i] = byte(i ^ (i >> 8))
	}

	var buf bytes.Buffer
	if err := sys.SaveToWriter(&buf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	sys2 := makeTestSystem()
	if err := sys2.LoadFromReader(&buf); err != nil {
		t.Fatalf("Load: %v", err)
	}

	for i := range sys.Bus.IRAM {
		if sys2.Bus.IRAM[i] != sys.Bus.IRAM[i] {
			t.Fatalf("IRAM mismatch at 0x%04X: got 0x%02X, want 0x%02X", i, sys2.Bus.IRAM[i], sys.Bus.IRAM[i])
		}
	}
}
