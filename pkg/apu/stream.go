package apu

import (
	"encoding/binary"
	"sync"
)

// Stream implements io.Reader to feed audio data to Ebitengine's audio.Context.
// It reads samples from the APU's ring buffer and converts them to 16-bit stereo
// little-endian PCM.
type Stream struct {
	apu        *APU
	mu         sync.Mutex
	lastSample int16 // held when buffer underruns to avoid clicks
}

// NewStream creates a new audio stream backed by the given APU.
func NewStream(apu *APU) *Stream {
	return &Stream{
		apu: apu,
	}
}

// Read fills buf with 16-bit stereo little-endian PCM data.
// Each sample frame is 4 bytes: 2 bytes left + 2 bytes right (mono duplicated).
// When the ring buffer is empty, the last sample is held (sample-and-hold)
// instead of outputting silence, preventing audible clicks at buffer underruns.
func (s *Stream) Read(buf []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	framesToWrite := len(buf) / 4
	written := 0

	s.apu.bufMu.Lock()
	for i := 0; i < framesToWrite; i++ {
		if s.apu.bufRead != s.apu.bufWrite {
			s.lastSample = s.apu.ringBuffer[s.apu.bufRead]
			s.apu.bufRead = (s.apu.bufRead + 1) % len(s.apu.ringBuffer)
		}
		// When buffer is empty, lastSample holds the previous value
		// instead of jumping to 0 (which would cause a click)

		offset := i * 4
		binary.LittleEndian.PutUint16(buf[offset:], uint16(s.lastSample))
		binary.LittleEndian.PutUint16(buf[offset+2:], uint16(s.lastSample))
		written += 4
	}
	s.apu.bufMu.Unlock()

	return written, nil
}
