package main

import (
	"bytes"
	"image"
	"image/color"
	"math"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed testimage.png
var testImage []byte

//go:embed character.png
var character []byte

type Game struct {
	Scene *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	Cubes []*tetra3d.Model

	Time float64
}

func NewGame() *Game {

	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	// OK, so in this stress test, we're rendering a lot of individual, low-poly objects (cubes).

	// To make this run faster, we can batch objects together dynamically - unlike static batching,
	// this will allow them to move and act individually while also allowing us to render them more
	// efficiently as they will render in a single batch.
	// However, dynamic batching has some limitations.

	// 1) Dynamically batched Models can't have individual material / object properties (like object
	// color, texture, material blend mode, or texture filtering mode). Vertex colors still
	// work for fading individual triangles / a dynamically batched Model, though.

	// 2) Dynamically batched objects all count up to a single vertex count, so we can't have
	// more than the max number of vertices after batching all objects together into a single
	// Model (which at this stage is 65535 vertices, or 21845 triangles).

	// 3) Dynamically batched objects render using the batching object's first meshpart's material
	// for rendering. This is to make it predictable as to how all batched objects appear at all times.

	// 4) Because we're batching together dynamically batched objects, they can't write to the depth
	// texture individually. This means dynamically batched objects cannot visually intersect with one
	// another - they simply draw behind or in front of one another, and so are sorted according to their
	// objects' depths in comparison to the camera.

	// While there are some restrictions, the advantages are weighty enough to make it worth it - this
	// functionality is very useful for things like drawing a lot of small, simple models, like NPCs,
	// particles, or map icons.

	// Create the scene (we're doing this ourselves in this case).
	g.Scene = tetra3d.NewScene("Test Scene")

	// Load the test cube texture.
	img, _, err := image.Decode(bytes.NewReader(testImage))
	if err != nil {
		panic(err)
	}

	// We can reuse the same mesh for all of the models.
	cubeMesh := tetra3d.NewCubeMesh()

	// Set up how the cube should appear.
	mat := cubeMesh.MeshParts[0].Material
	mat.Shadeless = true
	mat.Texture = ebiten.NewImageFromImage(img)

	// The batched model will hold all of our batching results. The batched objects keep their mesh-level
	// details (vertex positions, UV, normals, etc), but will mimic the batching object's material-level
	// and object-level properties (texture, material / object color, texture filtering, etc).

	// When a model is dynamically batching other models, the batching model doesn't render.

	// In this example, you will see it as a plane, floating above all the other Cubes before batching.

	planeMesh := tetra3d.NewPlaneMesh(2, 2)

	// Load the character texture.
	img, _, err = image.Decode(bytes.NewReader(character))
	if err != nil {
		panic(err)
	}

	mat = planeMesh.MeshParts[0].Material
	mat.Shadeless = true
	mat.Texture = ebiten.NewImageFromImage(img)

	batched := tetra3d.NewModel(planeMesh, "DynamicBatching")
	batched.Move(0, 4, 0)
	batched.Rotate(1, 0, 0, tetra3d.ToRadians(90))
	batched.Rotate(0, 1, 0, tetra3d.ToRadians(180))

	batched.FrustumCulling = false // For now, disallow frustum culling

	g.Cubes = []*tetra3d.Model{}

	for i := 0; i < 21; i++ {
		for j := 0; j < 21; j++ {
			// Create a new Cube, position it, add it to the scene, and add it to the cubes slice.
			cube := tetra3d.NewModel(cubeMesh, "Cube")
			cube.SetLocalPosition(float64(i)*1.5, 0, float64(-j*3))
			g.Scene.Root.AddChildren(cube)
			g.Cubes = append(g.Cubes, cube)
		}
	}

	g.Scene.Root.AddChildren(batched)

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetFar(120)
	g.Camera.SetLocalPosition(0, 0, 15)

	g.System = examples.NewBasicSystemHandler(g)

	ebiten.SetVsyncEnabled(false)

}

func (g *Game) Update() error {

	tps := 1.0 / 60.0 // 60 ticks per second, regardless of display FPS

	g.Time += tps

	for cubeIndex, cube := range g.Cubes {
		wp := cube.WorldPosition()

		wp.Y = math.Sin(g.Time*math.Pi + float64(cubeIndex))

		cube.SetWorldPositionVec(wp)
		cube.Color.R = float32(cubeIndex) / 100
		cube.Color.G = float32(cubeIndex) / 10
	}

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		dyn := g.Scene.Root.Get("DynamicBatching").(*tetra3d.Model)

		if len(dyn.DynamicBatchModels) == 0 {
			dyn.DynamicBatchAdd(dyn.Mesh.MeshParts[0], g.Cubes...) // Note that Model.DynamicBatchAdd() can return an error if batching the specified objects would push it over the vertex limit.
		} else {
			dyn.DynamicBatchRemove(g.Cubes...)
		}

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

	// Render the result
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `Stress Test 2 - Here, cubes are moving. We can render them
efficiently by dynamically batching them, though they
will mimic the batching object (the character plane) visually - 
they no longer have their own texture,
blend mode, or texture filtering (as they all
take these properties from the floating character plane).
They also can no longer intersect; rather, they
will just draw in front of or behind each other.

1 Key: Toggle batching cubes together`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {

	ebiten.SetWindowTitle("Tetra3d Test - Stress Test 2")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
