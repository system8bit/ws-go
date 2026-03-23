package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/system8bit/ws-go/frontend"
	"github.com/system8bit/ws-go/pkg/cart"
	"github.com/system8bit/ws-go/pkg/ws"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <rom.ws|rom.wsc>\n", os.Args[0])
		os.Exit(1)
	}

	romPath := os.Args[1]

	// Load ROM cartridge.
	cartridge, err := cart.LoadROM(romPath)
	if err != nil {
		log.Fatalf("Failed to load ROM: %v", err)
	}

	kind := "WonderSwan"
	if cartridge.IsColor() {
		kind = "WonderSwan Color"
	}
	fmt.Printf("Loaded %s ROM: %s (%d bytes)\n", kind, romPath, len(cartridge.ROM))

	// Load save data from .sav file (if any).
	if err := cartridge.LoadSave(); err != nil {
		log.Printf("Warning: failed to load save: %v", err)
	}

	// Create the emulated system.
	system := ws.New(cartridge)

	// Setup audio.
	frontend.SetupAudio(system.APU)

	// Create Ebitengine game.
	game := frontend.NewGame(system)
	game.ROMPath = romPath

	// Sync emulation updates with the display refresh rate (vsync).
	// The game loop uses a time accumulator to run the correct number of
	// WonderSwan frames (~75.47fps) per display frame, eliminating the
	// judder/flickering caused by the old fixed TPS=76 approach.
	// Audio adaptive rate control handles the sample-rate mismatch.
	ebiten.SetTPS(ebiten.SyncWithFPS)

	w, h := frontend.WindowSize(frontend.RotationNormal)
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("ws-go — WonderSwan Emulator")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

	// Save game data on exit.
	if err := cartridge.WriteSave(); err != nil {
		log.Printf("Warning: failed to write save: %v", err)
	}
}
