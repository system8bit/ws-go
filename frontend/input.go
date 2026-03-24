package frontend

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/system8bit/ws-go/pkg/input"
)

// KeyMapping maps an Ebitengine key to a WonderSwan button.
type KeyMapping struct {
	Key    ebiten.Key
	Button int
}

// DefaultMapping provides the default keyboard-to-button mapping for landscape mode.
// Arrow keys map to the X pad, WASD maps to the Y pad.
var DefaultMapping = []KeyMapping{
	{ebiten.KeyArrowUp, input.ButtonX1},
	{ebiten.KeyArrowRight, input.ButtonX2},
	{ebiten.KeyArrowDown, input.ButtonX3},
	{ebiten.KeyArrowLeft, input.ButtonX4},
	{ebiten.KeyW, input.ButtonY1},
	{ebiten.KeyD, input.ButtonY2},
	{ebiten.KeyS, input.ButtonY3},
	{ebiten.KeyA, input.ButtonY4},
	{ebiten.KeyZ, input.ButtonA},
	{ebiten.KeyX, input.ButtonB},
	{ebiten.KeyEnter, input.ButtonStart},
}

// VerticalMapping provides the keyboard-to-button mapping for portrait mode (rotated 90° CCW).
// In portrait mode:
//   - Arrow keys map to the Y pad (Up=Y2, Left=Y1, Right=Y3, Down=Y4)
//   - X pad X3 → A button (key X), X pad X4 → B button (key Z)
var VerticalMapping = []KeyMapping{
	{ebiten.KeyArrowUp, input.ButtonY2},
	{ebiten.KeyArrowRight, input.ButtonY3},
	{ebiten.KeyArrowDown, input.ButtonY4},
	{ebiten.KeyArrowLeft, input.ButtonY1},
	{ebiten.KeyZ, input.ButtonX3},
	{ebiten.KeyX, input.ButtonX4},
	{ebiten.KeyEnter, input.ButtonStart},
}

// UpdateInput polls the current Ebitengine key states and updates button pressed
// states. The rotation parameter selects the appropriate key mapping:
// RotationNormal uses DefaultMapping, RotationLeft90 uses VerticalMapping.
func UpdateInput(i *input.Input, rotation int) {
	mapping := DefaultMapping
	if rotation == RotationLeft90 {
		mapping = VerticalMapping
	}
	for _, m := range mapping {
		i.SetButton(m.Button, ebiten.IsKeyPressed(m.Key))
	}
}

