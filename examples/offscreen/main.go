package main

import (
	"bytes"
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"

	"github.com/hajimehoshi/ebiten/v2"

	examples "github.com/solarlune/tetra3d/examples"
)

type Game struct {
	Scene     *tetra3d.Scene
	Offscreen *ebiten.Image

	SystemHandler examples.BasicSystemHandler
	Camera        examples.BasicFreeCam
}

//go:embed offscreen.glb
var sceneData []byte

func NewGame() *Game {

	game := &Game{}

	game.Init()

	return game

}

func (g *Game) Init() {

	data, err := tetra3d.LoadGLTFData(bytes.NewReader(sceneData), nil)

	if err != nil {
		panic(err)
	}

	g.Scene = data.Scenes[0]

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.SystemHandler = examples.NewBasicSystemHandler(g)

	// Create an offscreen texture to render the camera to.
	g.Offscreen = ebiten.NewImage(g.Camera.Size())

	// Substitute the texture on the ScreenTexture material with our offscreen texture.
	data.MaterialByName("ScreenTexture").Texture = g.Offscreen

	// That's it!

	g.Scene.World.LightingOn = false

}

func (g *Game) Update() error {

	// Let the camera move around, handle locking and unlocking, etc.
	g.Camera.Update()

	// Spin the tetrahedron in the logo around their local orientation:
	g.Scene.Root.Search(tetra3d.SearchOptions{}.ByPropNames("spin")).ForEach(func(node tetra3d.INode) bool {
		node.Rotate(0, 1, 0, 0.05)
		return true
	})

	// Let the system handler handle quitting, fullscreening, restarting, etc.
	return g.SystemHandler.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear with a color:
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera:
	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	// Clear the Offscreen, then draw the camera's color texture output to it as well:
	g.Offscreen.Fill(color.Black)
	g.Offscreen.DrawImage(g.Camera.ColorTexture(), nil)

	// And then just draw the color texture output:
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// Finally, do any debug rendering that might be necessary. Because BasicFreeCam embeds *tetra3d.Camera, we pass
	// its embedded Camera instance.
	g.SystemHandler.Draw(screen, g.Camera.Camera)

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Logo Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
