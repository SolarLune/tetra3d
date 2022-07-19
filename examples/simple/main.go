package main

import (
	"errors"
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

type Game struct {
	Width, Height  int
	Scene          *tetra3d.Scene
	Camera         *tetra3d.Camera
	DrawDebugText  bool
	DrawDebugDepth bool
}

func NewGame() *Game {
	game := &Game{
		Width:         796,
		Height:        448,
		DrawDebugText: true,
	}

	game.Init()

	return game
}

// In this example, we will simply create a cube and place it in the scene.

func (g *Game) Init() {

	// Create a new Scene and name it.
	g.Scene = tetra3d.NewScene("cube example")

	// Turn off lighting.
	g.Scene.LightingOn = false

	// Create a cube, set the color, add it to the scene.
	cube := tetra3d.NewModel(tetra3d.NewCube(), "Cube")
	cube.Color.Set(0, 0.5, 1, 1)
	g.Scene.Root.AddChildren(cube)

	// Create a camera, move it back.
	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.Move(0, 0, 5)

	// Again, we don't need to actually add the camera to the scenegraph, but we'll do it anyway because why not.
	g.Scene.Root.AddChildren(g.Camera)

}

func (g *Game) Update() error {

	var err error

	// Quit if we press Escape.
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Fullscreen toggling.
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// Spinning the cube.
	cube := g.Scene.Root.Get("Cube")
	cube.SetLocalRotation(cube.LocalRotation().Rotated(0, 1, 0, 0.05))

	// Debug views

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.DrawDebugText = !g.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear the screen with a color.
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera.
	g.Camera.Clear()

	// Render the scene.
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// Draw depth texture if the debug option is enabled; draw color texture otherwise.
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), nil)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), nil)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := "F1 to toggle this text\nThis is a very simple example showing\na simple 3D cube,\ncreated through code.\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 130, color.RGBA{200, 200, 200, 255})
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
