package apu

import "sync"

const (
	SampleRate = 48000   // Output sample rate for Ebitengine (exact divisor of CPUClock)
	CPUClock   = 3072000 // 3.072 MHz

	ringBufferSize = 32768 // ring buffer capacity in samples (~682ms at 48kHz)

	// Master volume scale. Converts internal sample values to int16 range.
	// Voice D/A max: 128 × 80 = 10240 (~31% of int16).
	// Wavetable 4ch max: 900 × 80 = 72000 (clamped).
	masterScale = 80
)

// APU emulates the WonderSwan's audio processing unit.
type APU struct {
	Channels      [4]Channel
	ChannelVolume [4]byte // I/O 0x88-0x8B: L/R nibbles

	// Control registers
	NoiseCfg  byte   // I/O 0x8E
	SoundCtrl byte   // I/O 0x90
	OutputCtrl byte  // I/O 0x91
	NoiseLFSR uint16 // 15-bit LFSR state

	// Voice/D/A
	VoiceVolume byte // I/O 0x94

	// HyperVoice
	HyperVoice     byte // I/O 0x95
	HVoiceCtrl     byte // I/O 0x6A
	HVoiceChanCtrl byte // I/O 0x6B

	// Sweep (channel 3)
	SweepValue   int8 // I/O 0x8C
	SweepTime    byte // I/O 0x8D
	SweepCounter int
	sweepDivider int

	// BlipBuf band-limited synthesis
	blip         *BlipBuf
	cyclePos     int
	lastBlipMono int32

	// Ring buffer (consumed by Stream.Read)
	ringBuffer [ringBufferSize]int16
	bufRead    int
	bufWrite   int
	bufMu      sync.Mutex
	tmpSamples [8]int16

	// Shared state
	IRAM          []byte
	WaveTableBase uint16 // I/O 0x8F
}

func New() *APU {
	a := &APU{}
	a.blip = NewBlipBuf(1024)
	a.blip.SetRates(float64(CPUClock), float64(SampleRate))
	a.Reset()
	return a
}

func (a *APU) Reset() {
	a.Channels = [4]Channel{}
	a.ChannelVolume = [4]byte{}
	a.NoiseCfg = 0
	a.SoundCtrl = 0
	a.OutputCtrl = 0
	a.NoiseLFSR = 0x7FFF
	a.VoiceVolume = 0
	a.HyperVoice = 0
	a.HVoiceCtrl = 0
	a.HVoiceChanCtrl = 0
	a.SweepValue = 0
	a.SweepTime = 0
	a.SweepCounter = 0
	a.sweepDivider = 8192
	a.cyclePos = 0
	a.lastBlipMono = 0
	a.bufRead = 0
	a.bufWrite = 0
	a.WaveTableBase = 0
	if a.blip != nil {
		a.blip.Clear()
	}
}

// State accessors for save/load
func (a *APU) GetSweepDivider() int  { return a.sweepDivider }
func (a *APU) SetSweepDivider(v int) { a.sweepDivider = v }
func (a *APU) GetCyclePos() int      { return a.cyclePos }
func (a *APU) SetCyclePos(v int)     { a.cyclePos = v }
func (a *APU) GetLastBlipMono() int32  { return a.lastBlipMono }
func (a *APU) SetLastBlipMono(v int32) { a.lastBlipMono = v }

// Tick advances APU by cpuCycles, updating channels and recording BlipBuf deltas.
func (a *APU) Tick(cpuCycles int) {
	a.tickSweep(cpuCycles)
	a.tickAllChannels(cpuCycles)

	mono := a.computeMix()
	if delta := mono - a.lastBlipMono; delta != 0 {
		a.blip.AddDelta(a.cyclePos, delta)
		a.lastBlipMono = mono
	}
	a.cyclePos += cpuCycles
}

// tickAllChannels advances all 4 channels by cpuCycles.
func (a *APU) tickAllChannels(cpuCycles int) {
	waveBase := uint16(a.WaveTableBase) << 6

	for i := 0; i < 4; i++ {
		ch := &a.Channels[i]
		ch.Enabled = a.SoundCtrl&(1<<uint(i)) != 0

		switch {
		case i == 1 && a.SoundCtrl&0x20 != 0:
			a.tickVoiceDA(ch)
		case i == 3 && a.NoiseCfg&0x80 != 0:
			a.tickNoise(ch, i, cpuCycles)
		default:
			a.tickWavetable(ch, i, cpuCycles, waveBase+uint16(i)*16)
		}
	}
}

// tickVoiceDA processes Voice D/A mode for channel 2 (index 1).
func (a *APU) tickVoiceDA(ch *Channel) {
	if !ch.Enabled {
		ch.OutputL, ch.OutputR = 0, 0
		return
	}
	sample := int(a.ChannelVolume[1]) - 128 // center at 0
	half := sample / 2
	var left, right int
	if a.VoiceVolume&4 != 0 {
		left = sample
	} else if a.VoiceVolume&8 != 0 {
		left = half
	}
	if a.VoiceVolume&1 != 0 {
		right = sample
	} else if a.VoiceVolume&2 != 0 {
		right = half
	}
	ch.OutputL, ch.OutputR = int16(left), int16(right)
}

// tickNoise processes noise LFSR mode for channel 4 (index 3).
func (a *APU) tickNoise(ch *Channel, chIdx, cpuCycles int) {
	if !ch.Enabled {
		ch.OutputL, ch.OutputR = 0, 0
		return
	}
	period := 2048 - int(ch.Frequency)
	if period <= 0 {
		period = 1
	}
	ch.Counter += cpuCycles
	for ch.Counter >= period {
		ch.Counter -= period
		a.clockNoiseLFSR()
	}
	sample := -8 // center at 0
	if a.NoiseLFSR&1 != 0 {
		sample = 7
	}
	volL := int(a.ChannelVolume[chIdx] >> 4)
	volR := int(a.ChannelVolume[chIdx] & 0x0F)
	ch.OutputL, ch.OutputR = int16(sample*volL), int16(sample*volR)
}

// tickWavetable processes normal wavetable playback.
func (a *APU) tickWavetable(ch *Channel, chIdx, cpuCycles int, waveAddr uint16) {
	ch.tickChannel(cpuCycles, a.IRAM, waveAddr)
	volL := int(a.ChannelVolume[chIdx] >> 4)
	volR := int(a.ChannelVolume[chIdx] & 0x0F)
	raw := int(ch.Output) - 8 // center at 0
	ch.OutputL, ch.OutputR = int16(raw*volL), int16(raw*volR)
}

// computeMix returns the mono mix of all channels with HyperVoice.
func (a *APU) computeMix() int32 {
	if a.OutputCtrl == 0 {
		return 0
	}
	var mixedL, mixedR int32
	for i := 0; i < 4; i++ {
		mixedL += int32(a.Channels[i].OutputL)
		mixedR += int32(a.Channels[i].OutputR)
	}
	mixedL *= masterScale
	mixedR *= masterScale

	if a.HVoiceCtrl&0x80 != 0 {
		hv := int32(a.computeHyperVoice()) * masterScale
		if a.HVoiceChanCtrl&0x40 != 0 {
			mixedL += hv
		}
		if a.HVoiceChanCtrl&0x20 != 0 {
			mixedR += hv
		}
	}
	mono := (mixedL + mixedR) / 2
	if mono > 32767 {
		mono = 32767
	} else if mono < -32768 {
		mono = -32768
	}
	return mono
}

// EndScanline finalizes one scanline of BlipBuf data and writes to ring buffer.
func (a *APU) EndScanline(scanlineCycles int) {
	a.blip.EndFrame(scanlineCycles)
	a.cyclePos = 0

	n := a.blip.SamplesAvail()
	if n <= 0 {
		return
	}
	if n > len(a.tmpSamples) {
		n = len(a.tmpSamples)
	}
	n = a.blip.ReadSamples(a.tmpSamples[:], n)

	// Correct integrator drift from fixed-point rounding errors
	a.blip.integ = a.lastBlipMono

	a.bufMu.Lock()
	for i := 0; i < n; i++ {
		a.ringBuffer[a.bufWrite] = a.tmpSamples[i]
		a.bufWrite = (a.bufWrite + 1) % ringBufferSize
		if a.bufWrite == a.bufRead {
			a.bufRead = (a.bufRead + 1) % ringBufferSize
		}
	}
	a.bufMu.Unlock()
}

// EndFrame is a no-op kept for API compatibility.
func (a *APU) EndFrame() {}

func (a *APU) computeHyperVoice() int16 {
	s := uint16(a.HyperVoice)
	shift := uint(a.HVoiceCtrl & 3)
	switch a.HVoiceCtrl & 0x0C {
	case 0x00:
		s = s << (8 - shift)
	case 0x04:
		s = (s | 0xFF00) << (8 - shift)
	case 0x08:
		s = uint16(int8(a.HyperVoice)) << (8 - shift)
	case 0x0C:
		s = s << 8
	}
	return int16(s) >> 5
}

// BufWritePos/BufReadPos for diagnostics.
func (a *APU) BufWritePos() int { return a.bufWrite }
func (a *APU) BufReadPos() int  { return a.bufRead }

// WritePort handles writes to APU I/O ports.
func (a *APU) WritePort(port byte, val byte) {
	// Frequency registers 0x80-0x87: paired low/high bytes per channel
	if port >= 0x80 && port <= 0x87 {
		ch := int((port - 0x80) / 2)
		if port&1 == 0 {
			a.Channels[ch].Frequency = (a.Channels[ch].Frequency & 0x0700) | uint16(val)
		} else {
			a.Channels[ch].Frequency = (a.Channels[ch].Frequency & 0x00FF) | (uint16(val&0x07) << 8)
		}
		return
	}
	// Volume registers 0x88-0x8B
	if port >= 0x88 && port <= 0x8B {
		a.ChannelVolume[port-0x88] = val
		return
	}

	switch port {
	case 0x8C:
		a.SweepValue = int8(val)
	case 0x8D:
		a.SweepTime = val
		a.SweepCounter = int(val) + 1
		a.sweepDivider = 8192
	case 0x8E:
		if val&0x08 != 0 {
			a.NoiseLFSR = 0
		}
		a.NoiseCfg = val & 0x17
	case 0x8F:
		a.WaveTableBase = uint16(val)
	case 0x90:
		for n := 0; n < 4; n++ {
			if a.SoundCtrl&(1<<uint(n)) == 0 && val&(1<<uint(n)) != 0 {
				a.Channels[n].Counter = 1
				a.Channels[n].Position = 0x1F
			}
		}
		a.SoundCtrl = val
	case 0x91:
		a.OutputCtrl = val & 0x0F
	case 0x92:
		a.NoiseLFSR = (a.NoiseLFSR & 0xFF00) | uint16(val)
	case 0x93:
		a.NoiseLFSR = (a.NoiseLFSR & 0x00FF) | (uint16(val&0x7F) << 8)
	case 0x94:
		a.VoiceVolume = val & 0x0F
	case 0x95:
		a.HyperVoice = val
	case 0x6A:
		a.HVoiceCtrl = val
	case 0x6B:
		a.HVoiceChanCtrl = val & 0x6F
	}
}

// ReadPort handles reads from APU I/O ports.
func (a *APU) ReadPort(port byte) byte {
	// Frequency registers 0x80-0x87
	if port >= 0x80 && port <= 0x87 {
		ch := int((port - 0x80) / 2)
		if port&1 == 0 {
			return byte(a.Channels[ch].Frequency & 0xFF)
		}
		return byte(a.Channels[ch].Frequency >> 8)
	}
	// Volume registers 0x88-0x8B
	if port >= 0x88 && port <= 0x8B {
		return a.ChannelVolume[port-0x88]
	}

	switch port {
	case 0x8C:
		return byte(a.SweepValue)
	case 0x8D:
		return a.SweepTime
	case 0x8E:
		return a.NoiseCfg
	case 0x8F:
		return byte(a.WaveTableBase)
	case 0x90:
		return a.SoundCtrl
	case 0x91:
		return a.OutputCtrl | 0x80
	case 0x92:
		return byte(a.NoiseLFSR & 0xFF)
	case 0x93:
		return byte(a.NoiseLFSR >> 8)
	case 0x94:
		return a.VoiceVolume
	case 0x6A:
		return a.HVoiceCtrl
	case 0x6B:
		return a.HVoiceChanCtrl
	}
	return 0
}
