package ws

import "time"

// RTC emulates the WonderSwan cartridge real-time clock.
// Ports 0xCA (command) and 0xCB (data). Mednafen rtc.cpp reference.
type RTC struct {
	// BCD time registers
	sec  byte
	min  byte
	hour byte
	wday byte
	mday byte
	mon  byte
	year byte

	// Command state
	command      byte
	commandBuf   [7]byte
	commandIndex byte
	commandCount byte

	// Cycle accumulator — ticks real time at 3,072,000 cycles/sec
	clockCycles uint32
}

// State accessors for save/load
func (r *RTC) GetSec() byte         { return r.sec }
func (r *RTC) SetSec(v byte)        { r.sec = v }
func (r *RTC) GetMin() byte         { return r.min }
func (r *RTC) SetMin(v byte)        { r.min = v }
func (r *RTC) GetHour() byte        { return r.hour }
func (r *RTC) SetHour(v byte)       { r.hour = v }
func (r *RTC) GetWday() byte        { return r.wday }
func (r *RTC) SetWday(v byte)       { r.wday = v }
func (r *RTC) GetMday() byte        { return r.mday }
func (r *RTC) SetMday(v byte)       { r.mday = v }
func (r *RTC) GetMon() byte         { return r.mon }
func (r *RTC) SetMon(v byte)        { r.mon = v }
func (r *RTC) GetYear() byte        { return r.year }
func (r *RTC) SetYear(v byte)       { r.year = v }
func (r *RTC) GetCommand() byte     { return r.command }
func (r *RTC) SetCommand(v byte)    { r.command = v }
func (r *RTC) GetCommandBuf() [7]byte { return r.commandBuf }
func (r *RTC) SetCommandBuf(v [7]byte) { r.commandBuf = v }
func (r *RTC) GetCommandIndex() byte { return r.commandIndex }
func (r *RTC) SetCommandIndex(v byte) { r.commandIndex = v }
func (r *RTC) GetCommandCount() byte { return r.commandCount }
func (r *RTC) SetCommandCount(v byte) { r.commandCount = v }
func (r *RTC) GetClockCycles() uint32 { return r.clockCycles }
func (r *RTC) SetClockCycles(v uint32) { r.clockCycles = v }

// NewRTC creates an RTC seeded from the system local time.
func NewRTC() *RTC {
	r := &RTC{}
	r.initFromTime(time.Now())
	return r
}

func u8toBCD(v int) byte {
	return byte((v/10)<<4 | (v % 10))
}

func (r *RTC) initFromTime(t time.Time) {
	r.sec = u8toBCD(t.Second())
	r.min = u8toBCD(t.Minute())
	r.hour = u8toBCD(t.Hour())
	r.wday = u8toBCD(int(t.Weekday()))
	r.mday = u8toBCD(t.Day())
	r.mon = u8toBCD(int(t.Month()))
	r.year = u8toBCD(t.Year() % 100)

	// Kill the leap second (Mednafen)
	if r.sec >= 0x60 {
		r.sec = 0x59
	}
}

// Reset clears command state but preserves time.
func (r *RTC) Reset() {
	r.command = 0
	r.commandBuf = [7]byte{}
	r.commandIndex = 0
	r.commandCount = 0
}

// Clock adds CPU cycles and advances BCD time every 3,072,000 cycles (1 second).
func (r *RTC) Clock(cycles uint32) {
	r.clockCycles += cycles
	for r.clockCycles >= 3072000 {
		r.clockCycles -= 3072000
		r.tick()
	}
}

// bcdInc increments a BCD value. Returns true if it wrapped past thresh.
func bcdInc(v *byte, thresh byte, resetVal byte) bool {
	*v = ((*v + 1) & 0x0F) | (*v & 0xF0)
	if *v&0x0F >= 0x0A {
		*v &= 0xF0
		*v += 0x10
		if *v&0xF0 >= 0xA0 {
			*v &= 0x0F
		}
	}
	if *v >= thresh {
		*v = resetVal
		return true
	}
	return false
}

// tick advances the clock by one second (Mednafen GenericRTC::Clock).
func (r *RTC) tick() {
	if bcdInc(&r.sec, 0x60, 0) {
		if bcdInc(&r.min, 0x60, 0) {
			if bcdInc(&r.hour, 0x24, 0) {
				mdayThresh := byte(0x32) // 31+1 days default

				switch r.mon {
				case 0x02: // February
					mdayThresh = 0x29
					// Leap year check (Mednafen BCD logic)
					if (r.year&0x0F)%4 == func() byte {
						if r.year&0x10 != 0 {
							return 0x02
						}
						return 0x00
					}() {
						mdayThresh = 0x30
					}
				case 0x04, 0x06, 0x09, 0x11: // 30-day months
					mdayThresh = 0x31
				}

				bcdInc(&r.wday, 0x07, 0)

				if bcdInc(&r.mday, mdayThresh, 0x01) {
					if bcdInc(&r.mon, 0x13, 0x01) {
						bcdInc(&r.year, 0xA0, 0)
					}
				}
			}
		}
	}
}

// WritePort handles writes to RTC ports 0xCA-0xCB.
func (r *RTC) WritePort(port byte, val byte) {
	switch port {
	case 0xCA:
		r.command = val & 0x1F
		switch r.command {
		case 0x15: // Read time — snapshot into buffer
			r.commandBuf[0] = r.year
			r.commandBuf[1] = r.mon
			r.commandBuf[2] = r.mday
			r.commandBuf[3] = r.wday
			r.commandBuf[4] = r.hour
			r.commandBuf[5] = r.min
			r.commandBuf[6] = r.sec
			r.commandIndex = 0
			r.commandCount = 7
		case 0x14: // Program time — prepare to receive 7 bytes
			r.commandIndex = 0
			r.commandCount = 7
		case 0x13:
			// Acknowledged, no action (Mednafen)
		}

	case 0xCB:
		if r.command == 0x14 && r.commandIndex < r.commandCount {
			r.commandBuf[r.commandIndex] = val
			r.commandIndex++
			// Mednafen has the actual time-set code #if 0'd out,
			// but we implement it for completeness.
			if r.commandIndex == r.commandCount {
				r.year = r.commandBuf[0]
				r.mon = r.commandBuf[1]
				r.mday = r.commandBuf[2]
				r.wday = r.commandBuf[3]
				r.hour = r.commandBuf[4]
				r.min = r.commandBuf[5]
				r.sec = r.commandBuf[6]
			}
		}
	}
}

// ReadPort handles reads from RTC ports 0xCA-0xCB.
func (r *RTC) ReadPort(port byte) byte {
	switch port {
	case 0xCA:
		return r.command | 0x80 // bit 7 = ready
	case 0xCB:
		if r.command == 0x15 && r.commandIndex < r.commandCount {
			v := r.commandBuf[r.commandIndex]
			r.commandIndex++
			return v
		}
		return 0x80
	}
	return 0
}
