package main

import (
	"image/color"
	"math"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed orthographic.gltf
var sceneData []byte

type Game struct {
	Scene     *tetra3d.Scene
	CamHandle tetra3d.INode

	System examples.BasicSystemHandler
}

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

	g.Scene = library.Scenes[0]

	g.CamHandle = g.Scene.Root.Get("CameraHandle")

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	moveSpd := 0.1

	camera := g.CamHandle.Get("Camera").(*tetra3d.Camera)

	// Moving the Camera

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.CamHandle.Move(0, 0, -moveSpd)
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.CamHandle.Move(moveSpd, 0, 0)
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.CamHandle.Move(0, 0, moveSpd)
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.CamHandle.Move(-moveSpd, 0, 0)
	}

	scale := camera.OrthoScale()

	// Adjusting orthoscale changes how much you can see from an orthographic camera - this is both zooming and FOV changes.
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		scale += 0.5
	} else if ebiten.IsKeyPressed(ebiten.KeyW) {
		scale -= 0.5
	}

	// Limit orthoscale size.
	camera.SetOrthoScale(math.Max(math.Min(scale, 80), 10))
	camera.SetPerspective(false)

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		g.CamHandle.Rotate(0, 1, 0, -0.025)
	} else if ebiten.IsKeyPressed(ebiten.KeyQ) {
		g.CamHandle.Rotate(0, 1, 0, 0.025)
	}

	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	camera := g.CamHandle.Get("Camera").(*tetra3d.Camera)

	// Clear the Camera
	camera.Clear()

	camera.RenderScene(g.Scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	screen.DrawImage(camera.ColorTexture(), nil)

	g.System.Draw(screen, camera)

	if g.System.DrawDebugText {
		camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := `Arrow Keys: Pan in cardinal directions
W, S: Zoom in and Out
Q, E: Rotate View
R: Restart`
		camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	camera := g.CamHandle.Get("Camera").(*tetra3d.Camera)
	return camera.Size()
}

func main() {

	ebiten.SetWindowTitle("Tetra3d Test - Orthographic Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
