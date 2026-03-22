package apu

// Channel represents one of the WonderSwan's 4 wavetable sound channels.
type Channel struct {
	Frequency uint16 // 11-bit frequency value (0-2047)
	Enabled   bool

	// Internal state
	Counter  int   // frequency counter (counts up each CPU cycle)
	Position byte  // position in wavetable (0-31)
	Output   int16 // current raw output sample (before volume)
	OutputL  int16 // left output (after volume)
	OutputR  int16 // right output (after volume)
}

// tickChannel advances the channel's frequency counter and reads from the wavetable
// when the counter overflows. iram is the WonderSwan's internal RAM and waveAddr is
// the byte address of this channel's 16-byte (32-nibble) wavetable.
func (ch *Channel) tickChannel(cpuCycles int, iram []byte, waveAddr uint16) {
	if !ch.Enabled {
		ch.Output = 0
		return
	}

	period := 2048 - int(ch.Frequency)
	if period <= 0 {
		period = 1
	}

	// Don't tick if period is too small (Mednafen: skip if tmp_pt <= 4)
	if period <= 4 {
		return
	}

	ch.Counter += cpuCycles
	for ch.Counter >= period {
		ch.Counter -= period
		ch.Position = (ch.Position + 1) & 0x1F // wrap 0-31
		ch.Output = readWavetableSample(iram, waveAddr, ch.Position)
	}
}

// readWavetableSample reads a 4-bit sample from the wavetable in IRAM.
// Each byte holds two 4-bit samples: the low nibble is the even-indexed
// sample, the high nibble is the odd-indexed sample.
// Returns raw unsigned nibble 0-15 (Mednafen-compatible).
func readWavetableSample(iram []byte, waveAddr uint16, position byte) int16 {
	if iram == nil {
		return 0
	}

	byteOffset := uint16(position >> 1)
	addr := waveAddr + byteOffset
	if int(addr) >= len(iram) {
		return 0
	}

	raw := iram[addr]
	// Mednafen: >> ((sample_pos & 1) ? 4 : 0)
	// Even positions: low nibble, Odd positions: high nibble
	if position&1 == 0 {
		return int16(raw & 0x0F)
	}
	return int16((raw >> 4) & 0x0F)
}
