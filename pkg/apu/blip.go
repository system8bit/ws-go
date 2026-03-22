package apu

import "math"

// BlipBuf implements band-limited audio synthesis for sample-rate conversion.
// Amplitude deltas at the input clock rate are converted to anti-aliased
// output samples using a Blackman-windowed sinc kernel and integration.
//
// Usage:
//  1. Call AddDelta() for each amplitude change at its clock time
//  2. Call EndFrame() at the end of each emulation frame
//  3. Call ReadSamples() to retrieve anti-aliased output

const (
	bbFracBits  = 16                // fractional bits for sub-sample time tracking
	bbPhaseBits = 5                 // log2(phases)
	bbPhases    = 1 << bbPhaseBits  // 32 sub-sample phase positions
	bbHalfWidth = 8                 // sinc kernel half-width in output samples
	bbWidth     = bbHalfWidth * 2   // 16 total taps per phase
	bbScaleBits = 15                // kernel coefficient fixed-point precision
)

// Pre-computed Blackman-windowed sinc impulse response.
// bbImpulse[phase][tap]: phase in [0, bbPhases], tap in [0, bbWidth).
// For each phase, the tap coefficients sum to (1 << bbScaleBits).
var bbImpulse [bbPhases + 1][bbWidth]int32

func init() {
	hw := float64(bbHalfWidth)
	for p := 0; p <= bbPhases; p++ {
		frac := float64(p) / float64(bbPhases)
		var raw [bbWidth]float64
		sum := 0.0
		for t := 0; t < bbWidth; t++ {
			x := float64(t) - hw + 1.0 - frac

			var s float64
			if math.Abs(x) < 1e-10 {
				s = 1.0
			} else {
				s = math.Sin(math.Pi*x) / (math.Pi * x)
			}

			// Blackman window: excellent stopband attenuation (~-58 dB)
			wn := (float64(t) + 1.0 - frac) / float64(bbWidth)
			w := 0.42 - 0.5*math.Cos(2*math.Pi*wn) + 0.08*math.Cos(4*math.Pi*wn)

			raw[t] = s * w
			sum += raw[t]
		}

		if sum > 0 {
			scale := float64(int32(1)<<bbScaleBits) / sum
			for t := 0; t < bbWidth; t++ {
				bbImpulse[p][t] = int32(math.Round(raw[t] * scale))
			}
			// Fix rounding: adjust the center tap so the sum is exactly (1 << bbScaleBits).
			// This prevents DC drift in the integrator.
			actualSum := int32(0)
			for t := 0; t < bbWidth; t++ {
				actualSum += bbImpulse[p][t]
			}
			bbImpulse[p][bbHalfWidth-1] += (int32(1) << bbScaleBits) - actualSum
		}
	}
}

// BlipBuf is a band-limited synthesis buffer.
type BlipBuf struct {
	factor int64   // clock→sample conversion in fixed point (<<bbFracBits)
	offset int64   // accumulated fractional sample position across frames
	avail  int     // completed output samples available for reading
	buf    []int32 // sample accumulator (size + kernel overlap)
	integ  int32   // integrator: running sum converts impulses to steps
}

// NewBlipBuf creates a buffer that can hold up to size output samples.
func NewBlipBuf(size int) *BlipBuf {
	return &BlipBuf{
		buf: make([]int32, size+bbWidth),
	}
}

// SetRates configures the input clock rate and output sample rate.
// For WonderSwan: clockRate=3072000, sampleRate=48000 → factor=1024.
func (b *BlipBuf) SetRates(clockRate, sampleRate float64) {
	b.factor = int64(sampleRate/clockRate*float64(int64(1)<<bbFracBits) + 0.5)
}

// AddDelta records an amplitude change of delta at the given clock time
// (relative to the start of the current frame). The delta is convolved
// with the windowed sinc kernel and spread across nearby output samples.
func (b *BlipBuf) AddDelta(clockTime int, delta int32) {
	if delta == 0 {
		return
	}

	fixedPos := int64(clockTime)*b.factor + b.offset
	sIdx := int(fixedPos >> bbFracBits)
	phase := int(fixedPos>>(bbFracBits-bbPhaseBits)) & (bbPhases - 1)

	idx := b.avail + sIdx
	if idx < 0 || idx+bbWidth > len(b.buf) {
		return
	}

	imp := &bbImpulse[phase]
	for t := 0; t < bbWidth; t++ {
		b.buf[idx+t] += int32(int64(delta) * int64(imp[t]) >> bbScaleBits)
	}
}

// EndFrame marks the end of a frame of the given clock duration,
// converting accumulated time into available output samples.
func (b *BlipBuf) EndFrame(clockDuration int) {
	off := int64(clockDuration)*b.factor + b.offset
	b.avail += int(off >> bbFracBits)
	b.offset = off & ((1 << bbFracBits) - 1)
}

// SamplesAvail returns the number of completed output samples.
func (b *BlipBuf) SamplesAvail() int {
	return b.avail
}

// ReadSamples reads up to count anti-aliased output samples into out.
// The integrator converts the impulse-encoded buffer into a proper step
// waveform (running sum). Returns the number of samples actually read.
func (b *BlipBuf) ReadSamples(out []int16, count int) int {
	if count > b.avail {
		count = b.avail
	}
	if count <= 0 {
		return 0
	}

	sum := b.integ
	for i := 0; i < count; i++ {
		sum += b.buf[i]
		s := sum
		if s > 32767 {
			s = 32767
		} else if s < -32768 {
			s = -32768
		}
		out[i] = int16(s)
	}
	b.integ = sum

	// Shift remaining buffer data left to make room for new frame data
	remaining := b.avail - count + bbWidth
	copy(b.buf[:remaining], b.buf[count:count+remaining])
	for i := remaining; i < remaining+count && i < len(b.buf); i++ {
		b.buf[i] = 0
	}
	b.avail -= count

	return count
}

// Clear resets the buffer to silence.
func (b *BlipBuf) Clear() {
	b.offset = 0
	b.avail = 0
	b.integ = 0
	for i := range b.buf {
		b.buf[i] = 0
	}
}
