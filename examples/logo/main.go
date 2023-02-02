package main

import (
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

//go:embed tetra3d.gltf
var logoModel []byte

func NewGame() *Game {

	game := &Game{}

	game.Init()

	return game

}

func (g *Game) Init() {

	data, err := tetra3d.LoadGLTFData(logoModel, nil)

	if err != nil {
		panic(err)
	}

	g.Scene = data.Scenes[0]

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.SystemHandler = examples.NewBasicSystemHandler(g)

	g.Offscreen = ebiten.NewImage(g.Camera.Size())

	// Set the screen objects' screen texture to the offscreen buffer we just created above:
	data.Materials["ScreenTexture"].Texture = g.Offscreen

	// This is another way to do it:

	// screen := g.Scene.Root.Get("Screen").(*tetra3d.Model)
	// screen.Mesh().FindMeshPart("ScreenTexture").Material.Texture = g.Offscreen

	g.Scene.World.LightingOn = false

}

func (g *Game) Update() error {

	// Let the camera move around, handle locking and unlocking, etc.
	g.Camera.Update()

	// Spin the tetrahedrons in the logos around their local orientation:
	for _, g := range g.Scene.Root.SearchTree().ByProperties("spin").INodes() {
		g.Rotate(0, 1, 0, 0.05)
	}

	// Let the system handler handle quitting, fullscreening, restarting, etc.
	return g.SystemHandler.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear with a color:
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera:
	g.Camera.Clear()

	// Render the logos first:
	g.Camera.RenderNodes(g.Scene, g.Scene.Root.Get("LogoGroup"))

	// Clear the Offscreen, then draw the camera's color texture output to it as well:
	g.Offscreen.Fill(color.Black)
	g.Offscreen.DrawImage(g.Camera.ColorTexture(), nil)

	// Render the screen objects individually after drawing the others; this way, we can ensure the TVs don't show up onscreen:
	g.Camera.Render(g.Scene, g.Scene.Root.SearchTree().ByName("screen").Models()...)

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
