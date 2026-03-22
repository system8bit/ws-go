package apu

// tickSweep handles frequency sweep for channel 3 (index 2).
// Mednafen: sweep uses an 8192-cycle divider. When divider expires,
// sweep_counter decrements. When sweep_counter reaches 0, frequency
// is updated and counter reloads from (sweep_step + 1).
func (a *APU) tickSweep(cycles int) {
	// Sweep is only active when bit 6 of SoundCtrl is set.
	if a.SoundCtrl&0x40 == 0 {
		return
	}
	if a.SweepTime == 0 {
		return
	}

	a.sweepDivider -= cycles

	for a.sweepDivider <= 0 {
		a.sweepDivider += 8192

		a.SweepCounter--
		if a.SweepCounter <= 0 {
			a.SweepCounter = int(a.SweepTime) + 1

			// Apply sweep: period[ch] = (period[ch] + (int8)sweep_value) & 0x7FF
			ch := &a.Channels[2] // channel 3 (index 2)
			newFreq := (int(ch.Frequency) + int(a.SweepValue)) & 0x7FF
			ch.Frequency = uint16(newFreq)
		}
	}
}
