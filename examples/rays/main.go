package main

import (
	"bytes"
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed rays.glb
var blendFile []byte

type Game struct {
	Scene *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	// Load the GLTF file and turn it into a Library, which is a collection of scenes and data shared between them (like meshes or animations).

	library, err := tetra3d.LoadGLTFData(bytes.NewReader(blendFile), nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.FindScene("Scene")

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.System = examples.NewBasicSystemHandler(g)
	// g.Camera.Resize(320, 180)

	g.Camera.SetFieldOfView(100)

}

func (g *Game) Update() error {

	results := g.Camera.MouseRayTest(g.Camera.Far(), g.Scene.Root.SearchTree().IBoundingObjects()...)
	if len(results) > 0 {

		marker := g.Scene.Root.Get("Marker")

		marker.SetWorldPositionVec(results[0].Position.Add(results[0].Normal.Scale(0.5)))

		mat := tetra3d.NewLookAtMatrix(marker.WorldPosition(), results[0].Position, tetra3d.WorldUp)

		marker.SetWorldRotation(mat)

	}

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.FogColor.ToRGBA64())

	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `In this example, a ray is cast
from the mouse, forward. Whatever is hit will
be marked with a blue arrow. If the mouse is locked
to the game window, then the ray shoots directly forward
from the center of the screen.
`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Ray Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
