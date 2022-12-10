package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"time"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed testimage.png
var testImage []byte

type Game struct {
	Scene  *tetra3d.Scene
	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

func NewGame() *Game {

	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	g.Scene = tetra3d.NewScene("Test Scene")

	img, _, err := image.Decode(bytes.NewReader(testImage))
	if err != nil {
		panic(err)
	}

	// When creating a new mesh, it has no MeshParts.
	merged := tetra3d.NewModel(tetra3d.NewMesh("merged cubes"), "merged cubes")

	// Here, we'll store all of the cubes we'll merge together.
	cubes := []*tetra3d.Model{}

	// We can reuse the mesh for all of the models.
	cubeMesh := tetra3d.NewCube()

	// Create a new Model, position it, and add it to the cubes slice.
	// for x := 0; x < 100; x++ {
	// 	for j := 0; j < 100; j++ {
	for x := 0; x < 10; x++ {
		for z := 0; z < 10; z++ {
			for y := 0; y < 10; y++ {
				cube := tetra3d.NewModel(cubeMesh, "Cube")
				// cube.SetLocalPosition(3*float64(x), 3*float64(y), -3*float64(z))
				cubes = append(cubes, cube)
			}
		}
	}

	// Here, we merge all of the cubes into one mesh. This is good for when you have elements
	// you want to freely position in your modeler, for example, before combining them for
	// increased render speed in-game. Note that the maximum number of rendered triangles
	// in a single draw call is 21845.
	merged.StaticMerge(cubes...)

	// for i := 0; i < 6; i++ {
	// 	c := merged.Clone()
	// 	c.Move(0, float64(i)*4, 0)
	// 	g.Scene.Root.AddChildren(c)
	// }

	// After merging the cubes, the merged mesh has multiple MeshParts, but because the cube Models
	// shared a single Mesh (which has a single Material (aptly named "Cube")), the MeshParts also
	// share the same Material. This being the case, we don't need to set the image on all MeshParts'
	// Materials; just the first one is fine.

	mat := merged.Mesh.MeshParts[0].Material
	mat.Shadeless = true
	mat.Texture = ebiten.NewImageFromImage(img)

	// Add the merged result model, and we're done, basically.
	g.Scene.Root.AddChildren(merged)

	for x := 0; x < 8; x++ {
		m2 := merged.Clone()
		m2.Move(4*float64(x+1), 0, 0)
		g.Scene.Root.AddChildren(m2)
	}

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.System = examples.NewBasicSystemHandler(g)

	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)

	go func() {
		for {
			fmt.Println("FPS:", ebiten.ActualFPS())
			time.Sleep(time.Second)
		}
	}()

}

func (g *Game) Update() error {

	g.Camera.Update()

	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the non-screen Models
	// g.Camera.Render(g.Scene, g.Scene.Models...)

	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	if g.System.DrawDebugText {

		txt := `This is a simple stress test.
All of the cubes are statically merged
together into as few render calls as possible.`

		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

	g.System.Draw(screen, g.Camera.Camera)

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {

	ebiten.SetWindowTitle("Tetra3d Test - Stress Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
