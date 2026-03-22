package frontend

import "github.com/system8bit/ws-go/pkg/ppu"

const (
	// DefaultScale is the integer scaling factor for the window.
	// 224*3 = 672 pixels wide, 144*3 = 432 pixels tall.
	DefaultScale = 3
)

// WindowSize returns the default window dimensions (native resolution * scale).
func WindowSize() (int, int) {
	return ppu.ScreenWidth * DefaultScale, ppu.ScreenHeight * DefaultScale
}
