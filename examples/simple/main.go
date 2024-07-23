package main

import (
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene
	Camera        *tetra3d.Camera
	SystemHandler examples.BasicSystemHandler
}

func NewGame() *Game {
	game := &Game{
		Width:  796,
		Height: 448,
	}

	game.Init()

	return game
}

// In this example, we will simply create a cube and place it in the scene.

func (g *Game) Init() {

	// Create a new Scene and name it.
	g.Scene = tetra3d.NewScene("cube example")

	// Turn off lighting.
	g.Scene.World.LightingOn = false

	g.SystemHandler = examples.NewBasicSystemHandler(g)

	// Create a cube, set the color, add it to the scene.
	cube := tetra3d.NewModel("Cube", tetra3d.NewCubeMesh())
	cube.Color = tetra3d.NewColor(0, 0.5, 1, 1)
	g.Scene.Root.AddChildren(cube)

	// Create a camera, move it back on the Z axis (depth). The camera automatically looks forward.
	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.Move(0, 0, 5)

	// Again, we don't need to actually add the camera to the scenegraph, but we'll do it anyway because why not.
	g.Scene.Root.AddChildren(g.Camera)

}

func (g *Game) Update() error {

	// Spinning the cube.
	cube := g.Scene.Root.Get("Cube")
	cube.SetLocalRotation(cube.LocalRotation().Rotated(0, 1, 0, 0.05))

	// Debug views

	return g.SystemHandler.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear the screen with a color.
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera.
	g.Camera.Clear()

	// Render the scene.
	g.Camera.RenderScene(g.Scene)

	// Draw the resulting color texture.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// Draw the debug text if enabled, and that's basically it.
	g.SystemHandler.Draw(screen, g.Camera)

	if g.SystemHandler.DrawDebugText {
		txt := `F1 to toggle this text
This is a very simple example showing
a simple 3D cube, created through code.

(Note that the camera can't be moved
in this example.)`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	// This is a fixed aspect ratio; we can change this to, say, extend for wider displays by using the provided w argument and
	// calculating the height from the aspect ratio, then calling Camera.Resize() with the new width and height.
	return g.Width, g.Height
}

func main() {

	ebiten.SetWindowTitle("Tetra3d - Simple Test")

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
