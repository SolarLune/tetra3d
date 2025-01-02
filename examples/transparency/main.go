package main

import (
	"bytes"
	"math"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"
	"github.com/solarlune/tetra3d/math32"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed transparency.glb
var gltfData []byte

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	Time float32
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(bytes.NewReader(gltfData), nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetFar(30)
	g.Camera.SetLocalPosition(0, 5, 15)

	g.System = examples.NewBasicSystemHandler(g)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	water := g.Scene.Root.Get("Water").(*tetra3d.Model)

	water.VertexTransformFunction = func(v *tetra3d.Vector3, vertID int) {
		v.Y += math32.Sin((g.Time*math.Pi)+(v.X*1.2)+(v.Z*0.739)) * 0.1
	}

}

func (g *Game) Update() error {

	g.Time += 1.0 / 60.0
	g.Camera.Update()
	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear, but with a color
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
	g.Camera.RenderScene(g.Scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `Transparent materials draw in a second pass,
meaning they don't appear in the depth texture.
Because of this, they can overlap at certain
angles (which is bad), but also show things
underneath (which is, of course, good).`

		g.Camera.DrawDebugText(screen, txt, 0, 220, 1, colors.White())

	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Transparency Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
