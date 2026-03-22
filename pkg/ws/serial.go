package ws

// Serial emulates the WonderSwan serial communication port.
// Ports 0xB1 (data) and 0xB3 (control). Mednafen comm.cpp reference.
//
// In single-player mode (no external device), TX completes immediately
// on the next Comm_Process() call and RX never receives data.
type Serial struct {
	Control     byte // port 0xB3: bit 7=TX enable, bit 5=RX enable
	SendBuf     byte
	RecvBuf     byte
	SendLatched bool
	RecvLatched bool
}

// NewSerial creates a serial port in disconnected (single-player) mode.
func NewSerial() *Serial {
	return &Serial{}
}

// Reset clears all serial state.
func (s *Serial) Reset() {
	s.Control = 0
	s.SendBuf = 0
	s.RecvBuf = 0
	s.SendLatched = false
	s.RecvLatched = false
}

// WritePort handles writes to serial I/O ports.
func (s *Serial) WritePort(port byte, val byte) {
	switch port {
	case 0xB1:
		// Data register write: queue byte for TX if enabled
		if s.Control&0x80 != 0 {
			s.SendBuf = val
			s.SendLatched = true
		}
	case 0xB3:
		// Control register: only upper nibble is writable (Mednafen: V & 0xF0)
		s.Control = val & 0xF0
	}
}

// ReadPort handles reads from serial I/O ports.
func (s *Serial) ReadPort(port byte) byte {
	switch port {
	case 0xB1:
		// Data register read: return received byte, clear RX latch
		// Mednafen: reading clears RecvLatched and de-asserts WSINT_SERIAL_RECV
		s.RecvLatched = false
		return s.RecvBuf
	case 0xB3:
		// Status register:
		//   bits 7-4: control flags
		//   bit 2: TX ready (TX enabled AND not latched)
		//   bit 0: RX data available (RX enabled AND latched)
		status := s.Control & 0xF0
		if s.Control&0x80 != 0 && !s.SendLatched {
			status |= 0x04 // TX ready
		}
		if s.Control&0x20 != 0 && s.RecvLatched {
			status |= 0x01 // RX data available
		}
		return status
	}
	return 0
}

// Process is called once per scanline (Mednafen: Comm_Process).
// Returns true if WSINT_SERIAL_SEND (IRQ0) should fire.
// In single-player mode (no external device), TX completes immediately
// and RX never receives data.
func (s *Serial) Process() bool {
	// TX path (checked first, higher priority than RX)
	if s.SendLatched && s.Control&0x80 != 0 {
		// No external device: "dummy send" — discard byte, fire TX IRQ
		s.SendLatched = false
		return true
	}

	// RX path: no external device connected, so no data ever arrives.
	// RecvLatched stays false, WSINT_SERIAL_RECV is never asserted.

	return false
}
