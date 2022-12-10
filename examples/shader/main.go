package main

import (
	"math"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	Scene *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	Time float64
}

func NewGame() *Game {

	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	g.Scene = tetra3d.NewScene("shader test scene")

	mesh := tetra3d.NewCube()

	// Here we specify a fragment shader
	_, err := mesh.MeshParts[0].Material.SetShader([]byte(`
	package main

	func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
		scrSize := imageDstTextureSize()
		return vec4(position.x / scrSize.x, position.y / scrSize.y, 0, 1)
	}`))

	if err != nil {
		panic(err)
	}

	model := tetra3d.NewModel(mesh, "fragmentcube")
	model.Move(-2, 0, 0)
	g.Scene.Root.AddChildren(model)

	vertCube := tetra3d.NewCube()
	mat := vertCube.MeshParts[0].Material
	mat.Shadeless = true

	model = tetra3d.NewModel(vertCube, "vertexcube")
	model.Move(2, 0, 0)
	g.Scene.Root.AddChildren(model)

	// ... And here we specify a "vertex program" - in truth, this operates on CPU, rather than the GPU, but it still is useful.
	// Much like a Fragment shader, it operates on all vertices that render with the material.
	model.VertexTransformFunction = func(v tetra3d.Vector, id int) tetra3d.Vector {
		waveHeight := 0.1
		v.Y += math.Sin(g.Time*math.Pi+v.X)*waveHeight + (waveHeight / 2)
		return v
	}

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 2, 5)

}

func (g *Game) Update() error {

	g.Time += 1.0 / 60.0

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera.
	g.Camera.Clear()

	// Render the scene.
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// Draw the result.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This demo shows how custom shaders work.

There are two kinds of shader programs: 
fragment shaders and vertex programs.
Fragment shaders are written in Kage, executing
on the GPU, while vertex programs are written in 
pure Go and are executed on the CPU.

The cube on the left is running a fragment shader,
while the cube on the right runs a vertex program.`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Shader Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
