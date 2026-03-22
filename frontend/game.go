package frontend

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/system8bit/ws-go/pkg/ppu"
	"github.com/system8bit/ws-go/pkg/ws"
)

// WonderSwan runs at ~75.47 Hz (3,072,000 / 40,704 cycles per frame).
// To sync with the display (typically 60 Hz) without flickering, we use
// Ebitengine's SyncWithFPS mode and run a time-based frame accumulator.
const (
	wsFrameTime = 1.0 / 75.47 // ~13.25 ms per WS frame
)

// Game implements ebiten.Game for the WonderSwan emulator.
type Game struct {
	System      *ws.System
	ROMPath     string  // used to derive save state path
	accumulator float64 // seconds of emulation time to catch up
}

// NewGame creates a new Game that drives the given System each frame.
func NewGame(sys *ws.System) *Game {
	return &Game{System: sys}
}

// Update polls input and runs enough emulation frames to keep pace with
// wall-clock time. Called once per display frame (SyncWithFPS mode).
func (g *Game) Update() error {
	// F1 = reset, F2 = save state, F3 = load state
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.System.Reset()
		g.accumulator = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		path := g.saveStatePath()
		if err := g.System.SaveToFile(path); err != nil {
			fmt.Printf("Save state failed: %v\n", err)
		} else {
			fmt.Printf("State saved to %s\n", path)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		path := g.saveStatePath()
		if err := g.System.LoadFromFile(path); err != nil {
			fmt.Printf("Load state failed: %v\n", err)
		} else {
			fmt.Printf("State loaded from %s\n", path)
		}
	}

	UpdateInput(g.System.Input)

	// Add one display-frame's worth of time.
	// TPS = SyncWithFPS, so Update is called at display refresh rate.
	displayDt := 1.0 / float64(ebiten.TPS())
	if displayDt <= 0 || displayDt > 0.1 {
		displayDt = 1.0 / 60.0 // fallback
	}
	g.accumulator += displayDt

	// Run emulation frames to catch up. Typically 1-2 per display frame
	// at 60Hz display (75.47/60 ≈ 1.26 frames per update).
	// Cap at 8 to allow catch-up after brief stalls while preventing spiral-of-death.
	for n := 0; g.accumulator >= wsFrameTime && n < 8; n++ {
		g.System.RunFrame()
		g.accumulator -= wsFrameTime
	}

	return nil
}

// Draw copies the PPU display buffer to the Ebitengine screen.
// Uses DisplayBuffer (not Framebuffer) to avoid showing partially-rendered frames.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.WritePixels(g.System.PPU.DisplayBuffer[:])
}

// Layout returns the native WonderSwan resolution.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ppu.ScreenWidth, ppu.ScreenHeight
}

// saveStatePath returns the save state file path derived from the ROM path.
func (g *Game) saveStatePath() string {
	path := g.ROMPath
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		path = path[:idx]
	}
	return path + ".state"
}
