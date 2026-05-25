package main

import (
	"bytes"
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

//go:embed sectors.glb
var sceneData []byte

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {
	library, err := tetra3d.LoadGLTFData(bytes.NewReader(sceneData), nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.ExportedScene.Clone()

	g.Camera = examples.NewBasicFreeCam(g.Scene)

	// We use the existing camera for this.
	g.Camera.Camera = g.Scene.Root.Get("Camera").(*tetra3d.Camera)
	g.Camera.Camera.SectorRendering = true

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	g.Camera.Update()

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.Camera.SectorRendering = !g.Camera.SectorRendering
	}

	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene first
	g.Camera.RenderScene(g.Scene)

	// Draw the result
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {

		txt := `This example shows sector-based rendering, allowing you to control which
part of a game scene renders. Here, each colored room is a sector.
Sectors and their children only render when the camera is inside a Sector
or a neighboring Sector. In this example the neighbor depth is 1,
so the camera will render the current sector (room), + 1 neighboring room.

Objects of Standalone sector type always render, regardless of their location.
Press 1 to toggle sector-based rendering. It is: `

		if g.Camera.SectorRendering {
			txt += "On"
		} else {
			txt += "Off"
		}

		tetra3d.DrawDebugText(screen, txt, 0, 220, 1, colors.White())

	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Sectors Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
