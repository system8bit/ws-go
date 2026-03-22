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

// DefaultMapping provides the default keyboard-to-button mapping.
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
	{ebiten.KeyZ, input.ButtonB},
	{ebiten.KeyX, input.ButtonA},
	{ebiten.KeyEnter, input.ButtonStart},
}

// UpdateInput polls the current Ebitengine key states
// and updates button pressed states using the DefaultMapping.
func UpdateInput(i *input.Input) {
	for _, m := range DefaultMapping {
		i.SetButton(m.Button, ebiten.IsKeyPressed(m.Key))
	}
}
