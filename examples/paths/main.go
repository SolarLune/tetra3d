package main

import (
	"fmt"
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

	PathFollower *tetra3d.Navigator
	AutoAdvance  bool
}

//go:embed paths.gltf
var libraryData []byte

func NewGame() *Game {
	game := &Game{
		AutoAdvance: true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(libraryData, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library

	// We clone the scene so we have an original to work from
	g.Scene = library.ExportedScene.Clone()

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 5, 10)

	g.System = examples.NewBasicSystemHandler(g)

	g.PathFollower = tetra3d.NewNavigator(g.Scene.Root.Get("Path").(*tetra3d.Path))

	fmt.Println(g.Scene.Root.HierarchyAsString())

}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.PathFollower.AdvanceDistance(1)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.PathFollower.AdvanceDistance(-1)
	}

	if g.AutoAdvance {
		g.PathFollower.AdvancePercentage(0.01) // Advance one percent of the path per-frame
	}

	cube := g.Scene.Root.Get("Cube")
	cube.SetWorldPositionVec(g.PathFollower.WorldPosition())

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.AutoAdvance = !g.AutoAdvance
	}

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
1 key: Toggle running through path
Left, Right keys: Step 1 unit forward or
back through the path`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

	g.System.Draw(screen, g.Camera.Camera)

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
