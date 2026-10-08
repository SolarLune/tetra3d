package tetra3d

import (
	"flag"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// frames takes the steps of the tests that draw, and runs each one in the
// Update of its own frame, as ReadPixels needs a running game. It is nil
// when the tests do not run inside a game.
var frames chan func()

// TestMain runs the tests inside a small Ebitengine window when a display is
// available and -short is not set, so that the tests that draw can run.
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() || (os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "") {
		os.Exit(m.Run())
	}
	frames = make(chan func())
	code := make(chan int, 1)
	go func() {
		code <- m.Run()
		close(frames)
	}()
	ebiten.SetWindowSize(64, 64)
	if err := ebiten.RunGame(&frameGame{}); err != nil {
		panic(err)
	}
	os.Exit(<-code)
}

type frameGame struct {
	waitDraw bool // A step ran, and the next must wait for the Draw of its frame.
}

func (g *frameGame) Update() error {
	if g.waitDraw {
		return nil
	}
	f, ok := <-frames
	if !ok {
		return ebiten.Termination
	}
	f()
	g.waitDraw = true
	return nil
}

func (g *frameGame) Draw(*ebiten.Image) { g.waitDraw = false }

func (*frameGame) Layout(int, int) (int, int) { return 64, 64 }

// inFrame runs f in a frame of its own, or skips the test when the tests do
// not run inside a game.
func inFrame(t *testing.T, f func()) {
	t.Helper()
	if frames == nil {
		t.Skip("needs a GPU and a display; runs without -short")
	}
	done := make(chan struct{})
	frames <- func() {
		defer close(done)
		f()
	}
	<-done
}
