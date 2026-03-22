package ws

import "testing"

func TestTimerPresetLoadsCounter(t *testing.T) {
	timer := &Timer{}

	// Writing to preset should also load the counter
	timer.WritePort(0xA4, 0x2C) // low byte of 300 = 0x012C
	timer.WritePort(0xA5, 0x01) // high byte
	if timer.HBlankPreset != 300 {
		t.Errorf("HBlankPreset = %d, want 300", timer.HBlankPreset)
	}
	if timer.HBlankCount != 300 {
		t.Errorf("HBlankCount = %d, want 300 (should be loaded on preset write)", timer.HBlankCount)
	}

	// VBlank timer too
	timer.WritePort(0xA6, 0x4B) // low byte of 75 = 0x004B
	timer.WritePort(0xA7, 0x00) // high byte
	if timer.VBlankPreset != 75 {
		t.Errorf("VBlankPreset = %d, want 75", timer.VBlankPreset)
	}
	if timer.VBlankCount != 75 {
		t.Errorf("VBlankCount = %d, want 75", timer.VBlankCount)
	}
}

func TestTimerDisabledDoesNotTick(t *testing.T) {
	timer := &Timer{}

	// Set counter to 300 via preset write
	timer.WritePort(0xA4, 0x2C)
	timer.WritePort(0xA5, 0x01)

	// Timer disabled (Control = 0)
	timer.Control = 0

	// Tick 80 times — counter should not change
	for i := 0; i < 80; i++ {
		timer.TickHBlank()
	}
	if timer.HBlankCount != 300 {
		t.Errorf("HBlankCount = %d after 80 ticks with timer disabled, want 300", timer.HBlankCount)
	}
}

func TestTimerEnabledOneShotFires(t *testing.T) {
	timer := &Timer{}

	// Set counter to 3
	timer.WritePort(0xA4, 3)
	timer.WritePort(0xA5, 0)

	// Enable one-shot mode
	timer.WritePort(0xA2, 0x01)

	// Tick 1: 3→2
	if fired := timer.TickHBlank(); fired {
		t.Error("Timer fired at count 2")
	}
	// Tick 2: 2→1
	if fired := timer.TickHBlank(); fired {
		t.Error("Timer fired at count 1")
	}
	// Tick 3: 1→0 — should fire
	if fired := timer.TickHBlank(); !fired {
		t.Error("Timer did not fire at count 0")
	}
	// Tick 4: counter is 0, should not fire again (one-shot)
	if fired := timer.TickHBlank(); fired {
		t.Error("Timer fired again in one-shot mode")
	}
}

func TestTimerRepeatReloads(t *testing.T) {
	timer := &Timer{}

	// Set counter to 2 with repeat
	timer.WritePort(0xA4, 2)
	timer.WritePort(0xA5, 0)
	timer.WritePort(0xA2, 0x03) // enable + repeat

	// Tick 1: 2→1
	timer.TickHBlank()
	// Tick 2: 1→0, fires, reloads to 2
	if fired := timer.TickHBlank(); !fired {
		t.Error("Timer did not fire")
	}
	if timer.HBlankCount != 2 {
		t.Errorf("HBlankCount = %d after repeat, want 2", timer.HBlankCount)
	}

	// Should fire again after 2 more ticks
	timer.TickHBlank()
	if fired := timer.TickHBlank(); !fired {
		t.Error("Timer did not fire on second cycle")
	}
}

func TestTimerControlReloadsCounter(t *testing.T) {
	timer := &Timer{}

	// Set preset but don't enable
	timer.WritePort(0xA4, 0x0A) // preset = 10
	timer.WritePort(0xA5, 0x00)

	// Enable timer — should reload counter
	timer.WritePort(0xA2, 0x01)
	if timer.HBlankCount != 10 {
		t.Errorf("HBlankCount = %d after enable, want 10", timer.HBlankCount)
	}
}
