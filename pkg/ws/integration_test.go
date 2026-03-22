package ws

import (
	"os"
	"testing"

	"github.com/system8bit/ws-go/pkg/cart"
)

// TestBadAppleIntegration runs Bad Apple for 375 frames and verifies
// the CPU and PPU reach expected states. Skipped if ROM is not available.
func TestBadAppleIntegration(t *testing.T) {
	romPath := "../../bad_apple_ws.ws"
	if _, err := os.Stat(romPath); os.IsNotExist(err) {
		t.Skip("bad_apple_ws.ws not found, skipping integration test")
	}

	cartridge, err := cart.LoadROM(romPath)
	if err != nil {
		t.Fatalf("LoadROM: %v", err)
	}
	sys := New(cartridge)

	for frame := 1; frame <= 375; frame++ {
		sys.RunFrame()
	}

	// At frame 375, Bad Apple should be running the main loop:
	// CS=FF9C (data segment), Halted=false, IE=true
	// DispCtrl=0x22 (BG+FG on), MapBase=0x10 or 0x20 (alternating)
	// BackColor=0x00 (black), SoundCtrl=0x22 (Voice D/A + CH2)
	if sys.CPU.CS != 0xFF9C {
		t.Errorf("CPU.CS = 0x%04X, want 0xFF9C", sys.CPU.CS)
	}
	if sys.CPU.Halted {
		t.Error("CPU should not be halted at frame 375")
	}
	if !sys.CPU.InterruptEnable {
		t.Error("Interrupts should be enabled")
	}
	if sys.PPU.DispCtrl != 0x22 {
		t.Errorf("PPU.DispCtrl = 0x%02X, want 0x22", sys.PPU.DispCtrl)
	}
	if sys.PPU.BackColor != 0x00 {
		t.Errorf("PPU.BackColor = 0x%02X, want 0x00", sys.PPU.BackColor)
	}
	if sys.APU.SoundCtrl != 0x22 {
		t.Errorf("APU.SoundCtrl = 0x%02X, want 0x22", sys.APU.SoundCtrl)
	}

	// IRAM should have tile data in 0x2000-0x3FFF area
	nonZero := 0
	for i := 0x2000; i < 0x4000; i++ {
		if sys.Bus.IRAM[i] != 0 {
			nonZero++
		}
	}
	if nonZero < 100 {
		t.Errorf("Tile data area has only %d non-zero bytes, expected significant data", nonZero)
	}
}

// TestSwandrivingIntegration runs swandriving for 375 frames and verifies
// basic WSC color mode operation. Skipped if ROM is not available.
func TestSwandrivingIntegration(t *testing.T) {
	romPath := "../../swandriving.wsc"
	if _, err := os.Stat(romPath); os.IsNotExist(err) {
		t.Skip("swandriving.wsc not found, skipping integration test")
	}

	cartridge, err := cart.LoadROM(romPath)
	if err != nil {
		t.Fatalf("LoadROM: %v", err)
	}
	sys := New(cartridge)

	for frame := 1; frame <= 375; frame++ {
		sys.RunFrame()
	}

	// swandriving uses WSC color mode (wsVMode=7)
	if sys.PPU.WsVMode() != 7 {
		t.Errorf("PPU.WsVMode() = %d, want 7", sys.PPU.WsVMode())
	}
	if !sys.PPU.IsColor {
		t.Error("PPU.IsColor should be true")
	}
	// DispCtrl should have BG+FG+SPR enabled (0x07)
	if sys.PPU.DispCtrl != 0x07 {
		t.Errorf("PPU.DispCtrl = 0x%02X, want 0x07", sys.PPU.DispCtrl)
	}
	// SpriteCount should be 2 (car sprites)
	if sys.PPU.SpriteCount != 2 {
		t.Errorf("PPU.SpriteCount = %d, want 2", sys.PPU.SpriteCount)
	}
}

// TestSaveLoadContinuation verifies that save/load doesn't break emulation.
func TestSaveLoadContinuation(t *testing.T) {
	sys := makeTestSystem()

	// Run some frames to get into a non-trivial state
	for i := 0; i < 10; i++ {
		sys.RunFrame()
	}

	// Save state
	ss := sys.Snapshot()

	// Run 10 more frames
	for i := 0; i < 10; i++ {
		sys.RunFrame()
	}

	// Restore to the saved state
	sys.Restore(ss)

	// Run 10 more frames — should not panic or hang
	for i := 0; i < 10; i++ {
		sys.RunFrame()
	}

	// Basic sanity: system should still be functional
	if sys.CPU.CS == 0 && sys.CPU.IP == 0 {
		t.Error("CPU seems reset after save/load/continue")
	}
}
