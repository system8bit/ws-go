package frontend

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/system8bit/ws-go/pkg/apu"
)

// audioPlayer is kept at package level to prevent the GC from collecting
// the player (and stopping playback) after SetupAudio returns.
var audioPlayer *audio.Player

// SetupAudio initialises an Ebitengine audio context and starts streaming
// samples from the APU.
func SetupAudio(a *apu.APU) {
	ctx := audio.NewContext(apu.SampleRate)
	stream := apu.NewStream(a)
	audioPlayer, _ = ctx.NewPlayer(stream)
	// Larger buffer reduces dropouts from frame-timing jitter.
	// Default is ~46ms; 100ms gives comfortable headroom.
	audioPlayer.SetBufferSize(100 * time.Millisecond)
	audioPlayer.Play()
}
