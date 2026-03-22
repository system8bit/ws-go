package input

// WonderSwan buttons
const (
	ButtonY1    = iota // Up (horizontal mode) / Y1
	ButtonY2           // Right
	ButtonY3           // Down
	ButtonY4           // Left
	ButtonX1           // Up2 (vertical mode) / X1
	ButtonX2           // Right2
	ButtonX3           // Down2
	ButtonX4           // Left2
	ButtonA
	ButtonB
	ButtonStart
	ButtonCount
)

// Input manages WonderSwan button state and I/O port 0xB5 multiplexing.
type Input struct {
	Buttons [ButtonCount]bool
	keyCtrl byte // I/O port 0xB5 control value (written by game)
}

// New creates a new Input instance with all buttons released.
func New() *Input {
	return &Input{}
}

// SetButton sets the pressed state of a button.
func (i *Input) SetButton(btn int, pressed bool) {
	if btn >= 0 && btn < ButtonCount {
		i.Buttons[btn] = pressed
	}
}

// ReadPort reads from I/O port 0xB5.
// The returned value depends on which button group was selected via WritePort.
func (i *Input) ReadPort(port byte) byte {
	if port != 0xB5 {
		return 0
	}

	var val byte

	// Bit 4: Y button group (Y1-Y4 in bits 0-3)
	if i.keyCtrl&0x10 != 0 {
		if i.Buttons[ButtonY1] {
			val |= 0x01
		}
		if i.Buttons[ButtonY2] {
			val |= 0x02
		}
		if i.Buttons[ButtonY3] {
			val |= 0x04
		}
		if i.Buttons[ButtonY4] {
			val |= 0x08
		}
	}

	// Bit 5: X button group (X1-X4 in bits 0-3)
	if i.keyCtrl&0x20 != 0 {
		if i.Buttons[ButtonX1] {
			val |= 0x01
		}
		if i.Buttons[ButtonX2] {
			val |= 0x02
		}
		if i.Buttons[ButtonX3] {
			val |= 0x04
		}
		if i.Buttons[ButtonX4] {
			val |= 0x08
		}
	}

	// Bit 6: A/B/Start group (Start=bit1, A=bit2, B=bit3)
	if i.keyCtrl&0x40 != 0 {
		if i.Buttons[ButtonStart] {
			val |= 0x02
		}
		if i.Buttons[ButtonA] {
			val |= 0x04
		}
		if i.Buttons[ButtonB] {
			val |= 0x08
		}
	}

	return val
}

// WritePort writes to I/O port 0xB5, setting the multiplex control value
// that determines which button group will be returned on the next read.
func (i *Input) WritePort(port byte, val byte) {
	if port == 0xB5 {
		i.keyCtrl = val
	}
}
