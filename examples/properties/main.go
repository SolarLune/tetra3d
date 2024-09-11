package main

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Scene *tetra3d.Scene
	Time  float64

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

//go:embed properties.glb
var libraryData []byte

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	data, err := tetra3d.LoadGLTFData(bytes.NewReader(libraryData), nil)
	if err != nil {
		panic(err)
	}

	g.Scene = data.ExportedScene

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 5, 15)

	g.System = examples.NewBasicSystemHandler(g)

	for _, o := range g.Scene.Root.Children() {

		// In this example, we use a property named "parented to" to dynamically parent an object to another at run-time.
		if o.Properties().Has("parented to") {

			// Object reference properties are composed of strings, formatted as: [Scene Name]:[Object Name].
			// If the scene to search is not set in Blender, that portion will be blank.
			link := strings.Split(o.Properties().Get("parented to").AsString(), ":")

			// Store the object's original transform
			transform := o.Transform()

			// Reparent it to its new parent
			g.Scene.Root.Get(link[1]).AddChildren(o)

			// Re-apply the original transform, so it's in the same location as before
			o.SetWorldTransform(transform)
		}

	}

}

func (g *Game) Update() error {

	g.Time += 1.0 / 60.0

	// Here we loop through objects as usual, but notice from the blend file that collection instance
	// objects are replaced in the scene tree by their children.
	for _, o := range g.Scene.Root.Children() {

		if o.Properties().Has("turn") {
			o.Rotate(0, 1, 0, 0.02*o.Properties().Get("turn").AsFloat64())
		}

		if o.Properties().Has("wave") && o.Properties().Get("wave").AsBool() {
			o.Move(0, math.Sin(g.Time*math.Pi)*0.08, 0)
		}

	}

	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		fmt.Println(g.Scene.Root.HierarchyAsString())
		g.Scene.Root.Get("Cube.008").Unparent()
	}

	g.Camera.Update()

	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear the screen with the BG fill color.
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera.
	g.Camera.Clear()

	// Render the scene.
	g.Camera.RenderScene(g.Scene)

	// Draw the result.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This demo shows how game properties work with
the Tetra3D Blender add-on.
Game properties are set in the blend file, and
exported from there to a GLTF file.

Collection instances can be used as prefabs,
and properties set on a collection
instance object get propagated to top-level objects
in that collection.`
		g.Camera.DebugDrawText(screen, txt, 0, 220, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Properties Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
