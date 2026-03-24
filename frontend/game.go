package frontend

import (
	"fmt"
	"math"
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
	Rotation    int     // RotationNormal or RotationLeft90
	ScaleIndex  int     // index into Scales (0=1x, 1=1.5x, 2=2x)
}

// NewGame creates a new Game that drives the given System each frame.
func NewGame(sys *ws.System) *Game {
	return &Game{System: sys, ScaleIndex: DefaultScaleIndex}
}

// Update polls input and runs enough emulation frames to keep pace with
// wall-clock time. Called once per display frame (SyncWithFPS mode).
func (g *Game) Update() error {
	// F1 = reset, F2 = save state, F3 = load state, F4 = toggle portrait/landscape, F5 = cycle scale
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
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		if g.Rotation == RotationNormal {
			g.Rotation = RotationLeft90
			fmt.Println("Portrait mode (rotated 90° CCW)")
		} else {
			g.Rotation = RotationNormal
			fmt.Println("Landscape mode")
		}
		w, h := WindowSize(g.Rotation, Scales[g.ScaleIndex])
		ebiten.SetWindowSize(w, h)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.ScaleIndex = (g.ScaleIndex + 1) % len(Scales)
		scale := Scales[g.ScaleIndex]
		w, h := WindowSize(g.Rotation, scale)
		ebiten.SetWindowSize(w, h)
		fmt.Printf("Scale: %.1fx\n", scale)
	}

	UpdateInput(g.System.Input, g.Rotation)

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
// In portrait mode (RotationLeft90), the image is rotated 90° counter-clockwise.
func (g *Game) Draw(screen *ebiten.Image) {
	src := ebiten.NewImage(ppu.ScreenWidth, ppu.ScreenHeight)
	src.WritePixels(g.System.PPU.DisplayBuffer[:])

	op := &ebiten.DrawImageOptions{}
	if g.Rotation == RotationLeft90 {
		// Rotate 90° CCW around the image centre, then translate into view.
		// Rotation centre: (W/2, H/2). After -90° the image is H wide × W tall.
		op.GeoM.Translate(-float64(ppu.ScreenWidth)/2, -float64(ppu.ScreenHeight)/2)
		op.GeoM.Rotate(-math.Pi / 2)
		op.GeoM.Translate(float64(ppu.ScreenHeight)/2, float64(ppu.ScreenWidth)/2)
	}
	screen.DrawImage(src, op)
}

// Layout returns the logical (native) resolution for Ebitengine.
// In portrait mode the axes are swapped so the rotated image fills the window.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if g.Rotation == RotationLeft90 {
		return ppu.ScreenHeight, ppu.ScreenWidth
	}
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

