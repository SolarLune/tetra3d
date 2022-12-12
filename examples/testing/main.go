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

	System examples.BasicSystemHandler
}

func NewGame() *Game {
	game := &Game{
		Width:  640,
		Height: 360,
	}

	game.Init()

	return game
}

// In this example, we will simply create a cube and place it in the scene.

func (g *Game) Init() {

	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)

	// Create a new Scene and name it.
	g.Scene = tetra3d.NewScene("cube example")

	// Turn off lighting.
	g.Scene.World.LightingOn = false

	// Create a cube, set the color, add it to the scene.
	cube := tetra3d.NewModel(tetra3d.NewCubeMesh(), "Cube")
	cube.Color.RandomizeRGB(0.5, 1, false)
	g.Scene.Root.AddChildren(cube)

	g.System = examples.NewBasicSystemHandler(g)
	g.System.UsingBasicFreeCam = false

	// Create a camera, move it back.
	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.Move(0, 2, 15)

	// We don't need to actually add the camera to the scenegraph, but we'll do it anyway because why not.
	g.Scene.Root.AddChildren(g.Camera)

	cube.Rotate(0, 1, 0, 1)

}

func (g *Game) Update() error {
	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear the screen with a color.
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera.
	g.Camera.Clear()

	// Render the scene.
	g.Camera.RenderScene(g.Scene)

	// Draw depth texture if the debug option is enabled; draw color texture otherwise.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera)

	if g.System.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := `This is a very simple example showing
a simple 3D cube,
created through code.`
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
