package main

import (
	"bytes"
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	FlashingVertices tetra3d.VertexSelection

	Time float64

	Cube *tetra3d.Model
}

//go:embed vertexSelection.glb
var libraryData []byte

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(bytes.NewReader(libraryData), nil)
	if err != nil {
		panic(err)
	}

	g.Library = library

	// We clone the scene as it's best practice not to mess with the original.
	g.Scene = library.ExportedScene.Clone()

	g.Cube = g.Scene.Root.Get("Cube").(*tetra3d.Model)

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 5, 10)

	g.System = examples.NewBasicSystemHandler(g)

	// ------

	// OK, so this demo is about selecting vertices.

	// The easiest way to select vertices is to just use Mesh.SelectVertices() - it allows us to create a VertexSelection object, which is our
	// current vertex capture. We can use it to select vertices that fulfill a set of criteria (and yes, it's method-chainable as well).

	// Internally for the VertexSelection object, the vertices are simply index numbers, with their properties (position, UV, normals, etc.)
	// stored on the Mesh as a series of slices (i.e. Mesh.VertexPositions[], Mesh.VertexNormals[], Mesh.VertexUVs[], etc).

	// Oh, and by default, when we export a mesh with vertex colors, the first vertex color channel is active.

	// Here, we'll select our vertices and store it in the Game struct.
	mesh := g.Cube.Mesh

	// VertexSelection.SelectInVertexColorChannel selects all vertices that have greater than 0 influence to a particular vertex group.
	// You can also use VertexSelection.SelectInVertexColorChannel() to select vertices that have non-black vertex color.
	vertices, err := tetra3d.NewVertexSelection().SelectInVertexColorChannel(mesh, "Flash")

	// An error could happen if the color channel index passed to SelectInChannel is too high to be beyond the vertex color channel count set on the Model.
	if err != nil {
		panic(err)
	}

	g.FlashingVertices = vertices

	// Once we know which vertices are not black in the "Flash" channel in Blender, we're good to go.

}

func (g *Game) Update() error {

	g.Time += 1.0 / 60.0

	// Here we'll create a new glowing color.
	glow := tetra3d.NewColorFromHSV(g.Time/10, 1, 1)

	// Once we've gotten our indices, we can use the helper SetColor() function to set the color of all vertices at the same time.
	// There are other functions to, for example, move or set the normal of the vertices, but if we wanted to do something else a
	// bit more special, we could do so manually by looping through the indices in the *VertexSelection instance and modifying the
	// mesh's vertex properties.
	g.FlashingVertices.SetColor(g.Cube.Mesh.VertexColorChannelNames["Color"], glow)

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

	// Now draw the result to the screen
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This demo shows how to easily select specific
vertices using vertex color in Tetra3D.
In this example, the glowing faces are colored in the 'Flash'
vertex color channel in Blender (which
becomes channel 1 in Tetra3D as it's in the second slot).
The vertices that have a non-black color
in the "Flash" channel are then assigned
a flashing color here in Tetra3D.`
		g.Camera.DrawDebugText(screen, txt, 0, 220, 1, colors.LightGray())
	}

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
