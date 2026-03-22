package apu

// Mednafen sound.cpp: NoiseByetable[8] = {14,10,13,4,8,6,9,11}
var noiseTaps = [8]uint{14, 10, 13, 4, 8, 6, 9, 11}

// clockNoiseLFSR advances the 15-bit LFSR one step.
// Mednafen: nreg = ((nreg << 1) | ((1 ^ (nreg >> 7) ^ (nreg >> stab[tap])) & 1)) & 0x7FFF
func (a *APU) clockNoiseLFSR() {
	tapIndex := a.NoiseCfg & 0x07
	tap := noiseTaps[tapIndex]
	feedback := (1 ^ (a.NoiseLFSR >> 7) ^ (a.NoiseLFSR >> tap)) & 1
	a.NoiseLFSR = ((a.NoiseLFSR << 1) | feedback) & 0x7FFF
}
