package main

import (
	"bytes"
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	PathStepper *tetra3d.PathStepper
}

//go:embed paths.gltf
var libraryData []byte

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(bytes.NewReader(libraryData), nil)
	if err != nil {
		panic(err)
	}

	g.Library = library

	// We clone the scene so we have an original to work from
	g.Scene = library.ExportedScene.Clone()

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 5, 10)

	g.System = examples.NewBasicSystemHandler(g)

	g.PathStepper = tetra3d.NewPathStepper(g.Scene.Root.Get("Path").(*tetra3d.Path))

}

func (g *Game) Update() error {

	cube := g.Scene.Root.Get("Cube")
	cubePos := cube.WorldPosition()
	pathNodePos := g.PathStepper.CurrentWorldPosition()

	// Advance the stepper if you reach the target node...
	if cubePos.Equals(pathNodePos) {
		g.PathStepper.Next()
	}

	// ... Or if you press right or left.
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.PathStepper.Next()
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.PathStepper.Prev()
	}

	g.Scene.Root.Get("PointMarker").SetLocalPositionVec(pathNodePos)

	diff := pathNodePos.Sub(cubePos).ClampMagnitude(0.1)

	cube.MoveVec(diff)

	g.Camera.Update()
	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
	g.Camera.RenderScene(g.Scene)

	// Draw the result
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	if g.System.DrawDebugText {
		txt := `This demo shows how paths work.
The cube will follow the path (which is invisible,
as it is made up of Nodes).
Left, Right keys: Step 1 unit forward or
back through the path`
		g.Camera.DebugDrawText(screen, txt, 0, 220, 1, colors.LightGray())
	}

	g.System.Draw(screen, g.Camera.Camera)

	g.Camera.DrawDebugWireframe(screen, g.Scene.Root.Get("Path"), colors.White())

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Paths Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
