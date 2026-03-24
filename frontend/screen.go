package frontend

import "github.com/system8bit/ws-go/pkg/ppu"

const (
	// RotationNormal is the standard horizontal (landscape) orientation.
	RotationNormal = 0
	// RotationLeft90 is the vertical (portrait) orientation, rotated 90 degrees counter-clockwise.
	RotationLeft90 = 1
)

// Scales lists the available window scale multipliers cycled by F5.
// Values are relative to the native WonderSwan resolution (224x144).
var Scales = []float64{2.0, 4.0, 6.0}

// DefaultScaleIndex is the index into Scales used at startup (4x).
const DefaultScaleIndex = 1

// WindowSize returns the window dimensions for the given rotation and scale.
// In portrait mode (RotationLeft90), width and height are swapped.
func WindowSize(rotation int, scale float64) (int, int) {
	w := int(float64(ppu.ScreenWidth) * scale)
	h := int(float64(ppu.ScreenHeight) * scale)
	if rotation == RotationLeft90 {
		return h, w
	}
	return w, h
}
