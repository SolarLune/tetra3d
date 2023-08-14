package main

import (
	"bytes"
	"image/png"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	Scene *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	BG *ebiten.Image
}

//go:embed compositing.gltf
var compositingGLTF []byte

//go:embed bg.png
var bgPng []byte

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	// Load the GLTF.
	data, err := tetra3d.LoadGLTFData(compositingGLTF, nil)
	if err != nil {
		panic(err)
	}

	g.Scene = data.Scenes[0].Clone()

	// Load the background image.
	br := bytes.NewReader(bgPng)
	img, err := png.Decode(br)
	if err != nil {
		panic(err)
	}
	g.BG = ebiten.NewImageFromImage(img)

	// Set up a camera.
	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 0, 5)

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {
	g.Camera.Update()
	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	screen.Clear()

	// Draw BG
	opt := &ebiten.DrawImageOptions{}
	w, h := g.BG.Size()
	camW, camH := g.Camera.Size()
	opt.GeoM.Scale(float64(camW)/float64(w), float64(camH)/float64(h))
	screen.DrawImage(g.BG, opt)

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
	g.Camera.RenderScene(g.Scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		g.Camera.DebugDrawText(screen,
			`This demo shows how composite modes work.
The blue plane is opaque.
The red one is additive.
The green one is transparent.
The white square cuts out all objects to show the background.`,
			0, 200, 1, colors.LightGray(),
		)
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Alpha Ordering Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
