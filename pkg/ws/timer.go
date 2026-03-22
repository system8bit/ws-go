package ws

// Timer manages the WonderSwan HBlank and VBlank countdown timers.
//
// I/O ports:
//
//	0xA2:      Timer control
//	0xA4-0xA5: HBlank timer preset (16-bit LE)
//	0xA6-0xA7: VBlank timer preset (16-bit LE)
//	0xA8-0xA9: HBlank timer counter (16-bit LE, read-only)
//	0xAA-0xAB: VBlank timer counter (16-bit LE, read-only)
//
// Timer control bits:
//
//	Bit 0: HBlank timer enable
//	Bit 1: HBlank timer auto-preset (repeat on expiry)
//	Bit 2: VBlank timer enable
//	Bit 3: VBlank timer auto-preset (repeat on expiry)
type Timer struct {
	Control      byte
	HBlankPreset uint16
	VBlankPreset uint16
	HBlankCount  uint16
	VBlankCount  uint16
}

// Reset clears all timer state.
func (t *Timer) Reset() {
	t.Control = 0
	t.HBlankPreset = 0
	t.VBlankPreset = 0
	t.HBlankCount = 0
	t.VBlankCount = 0
}

// WritePort handles writes to timer I/O ports.
func (t *Timer) WritePort(port byte, val byte) {
	switch port {
	case 0xA2:
		t.Control = val
		// Writing control reloads counters from presets for enabled timers
		if val&0x01 != 0 {
			t.HBlankCount = t.HBlankPreset
		}
		if val&0x04 != 0 {
			t.VBlankCount = t.VBlankPreset
		}
	case 0xA4:
		t.HBlankPreset = (t.HBlankPreset & 0xFF00) | uint16(val)
		t.HBlankCount = t.HBlankPreset // Mednafen: writing preset also loads counter
	case 0xA5:
		t.HBlankPreset = (t.HBlankPreset & 0x00FF) | (uint16(val) << 8)
		t.HBlankCount = t.HBlankPreset // Mednafen: writing preset also loads counter
	case 0xA6:
		t.VBlankPreset = (t.VBlankPreset & 0xFF00) | uint16(val)
		t.VBlankCount = t.VBlankPreset
	case 0xA7:
		t.VBlankPreset = (t.VBlankPreset & 0x00FF) | (uint16(val) << 8)
		t.VBlankCount = t.VBlankPreset
	}
}

// ReadPort handles reads from timer I/O ports.
func (t *Timer) ReadPort(port byte) byte {
	switch port {
	case 0xA2:
		return t.Control
	case 0xA4:
		return byte(t.HBlankPreset)
	case 0xA5:
		return byte(t.HBlankPreset >> 8)
	case 0xA6:
		return byte(t.VBlankPreset)
	case 0xA7:
		return byte(t.VBlankPreset >> 8)
	case 0xA8:
		return byte(t.HBlankCount)
	case 0xA9:
		return byte(t.HBlankCount >> 8)
	case 0xAA:
		return byte(t.VBlankCount)
	case 0xAB:
		return byte(t.VBlankCount >> 8)
	}
	return 0
}

// TickHBlank is called once per scanline. If the HBlank timer is enabled
// (bit 0), it decrements the counter and returns true when it reaches zero
// (indicating IRQ7 should fire). If bit 1 (repeat) is set, the counter
// auto-reloads from the preset value.
func (t *Timer) TickHBlank() bool {
	if t.Control&0x01 == 0 {
		return false
	}
	if t.HBlankCount == 0 {
		return false
	}
	t.HBlankCount--
	if t.HBlankCount == 0 {
		if t.Control&0x02 != 0 {
			// Auto-preset: reload counter
			t.HBlankCount = t.HBlankPreset
		}
		return true
	}
	return false
}

// TickVBlank is called once per VBlank. If the VBlank timer is enabled
// (bit 2), it decrements the counter and returns true when it reaches zero
// (indicating IRQ5 should fire). If bit 3 (repeat) is set, the counter
// auto-reloads from the preset value.
func (t *Timer) TickVBlank() bool {
	if t.Control&0x04 == 0 {
		return false
	}
	if t.VBlankCount == 0 {
		return false
	}
	t.VBlankCount--
	if t.VBlankCount == 0 {
		if t.Control&0x08 != 0 {
			// Auto-preset: reload counter
			t.VBlankCount = t.VBlankPreset
		}
		return true
	}
	return false
}
