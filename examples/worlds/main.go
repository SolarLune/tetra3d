package main

import (
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Scene   *tetra3d.Scene
	Library *tetra3d.Library

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

//go:embed worlds.gltf
var sceneData []byte

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {
	library, err := tetra3d.LoadGLTFData(sceneData, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.ExportedScene.Clone()

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.Far = 30
	g.Camera.SetLocalPosition(0, 5, 5)

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.Scene.World = g.Library.Worlds["Dark"]
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.Scene.World = g.Library.Worlds["Bright"]
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.Scene.World = g.Library.Worlds["Blue"]
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		g.Scene.World = g.Library.Worlds["Red"]
	}

	g.Camera.Update()

	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene first
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// Draw the result
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {

		txt := `1-4 Keys: Cycle Worlds
Each World can have its own clear color,
fog color and range, and ambient color.`

		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.White())

	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Worlds Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
