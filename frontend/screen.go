package frontend

import "github.com/system8bit/ws-go/pkg/ppu"

const (
	// DefaultScale is the integer scaling factor for the window.
	// 224*3 = 672 pixels wide, 144*3 = 432 pixels tall.
	DefaultScale = 3

	// RotationNormal is the standard horizontal (landscape) orientation.
	RotationNormal = 0
	// RotationLeft90 is the vertical (portrait) orientation, rotated 90 degrees counter-clockwise.
	RotationLeft90 = 1
)

// WindowSize returns the window dimensions for the given rotation mode.
// In portrait mode (RotationLeft90), width and height are swapped.
func WindowSize(rotation int) (int, int) {
	if rotation == RotationLeft90 {
		// Portrait: 144*3 = 432 wide, 224*3 = 672 tall
		return ppu.ScreenHeight * DefaultScale, ppu.ScreenWidth * DefaultScale
	}
	return ppu.ScreenWidth * DefaultScale, ppu.ScreenHeight * DefaultScale
}
