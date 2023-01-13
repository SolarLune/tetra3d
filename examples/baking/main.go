package main

import (
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed baking.gltf
var gltfData []byte

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

const (
	ChannelLight    = iota // The light color channel
	ChannelAO              // The AO color channel
	ChannelCombined        // The combined color channel
	ChannelNone     = -1   // No color channel
)

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(gltfData, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.Move(0, 10, 0)
	g.System = examples.NewBasicSystemHandler(g)

	// OK, so in this example we're going to bake lighting and AO.
	// We will combine them into different vertex color channels to
	// easily preview the differences, as well.

	// We start by collecting all the lights we'll be baking. NodeFilter.Lights() will automatically give us just the lights out of our selection and discard any other INodes.
	lights := g.Scene.Root.ChildrenRecursive().Lights()

	// // Don't forget the world's ambient light!
	lights = append(lights, g.Scene.World.AmbientLight)

	// Let's get all the solid, occluding models here.
	models := g.Scene.Root.ChildrenRecursive().ByProperties("ao").Models()

	// The idea is that we'll bake the lighting and AO of each ao-applicable Model.

	// We'll bake the lighting to vertex channel #0, AO to channel #1, and combine both
	// in vertex channel #2. We will use named consts to make referencing these channels simple.

	for _, model := range models {

		// First, we'll bake the lighting into the lighting vertex color channel.
		model.BakeLighting(ChannelLight, lights...)

		// Next, we'll bake the AO into the AO vertex color channel.

		// The AOBakeOptions struct controls how our AO should be baked. For readability, we create it
		// here, though we should, of course, create it outside our model loop if the options don't differ
		// for each Model.
		bakeOptions := tetra3d.NewDefaultAOBakeOptions()
		bakeOptions.OcclusionAngle = tetra3d.ToRadians(60)
		bakeOptions.TargetChannel = ChannelAO // Set the target baking channel (as it's not 0)
		// bakeOptions.OtherModels = models      // Specify what other objects should influence the AO (note that this is optional)
		model.Mesh.SetVertexColor(ChannelAO, colors.White())
		model.BakeAO(bakeOptions) // And bake the AO.

		// We'll combine the channels together multiplicatively here.
		model.Mesh.CombineVertexColors(ChannelCombined, true, ChannelLight, ChannelAO)

		for _, mat := range model.Mesh.Materials() {
			mat.Shadeless = true // We don't need lighting anymore.
		}

		// Finally, we'll set the models' active color channel here. By default, it's -1, indicating no vertex colors are active (unless
		// the mesh was exported from Blender with an active vertex color channel).
		model.Mesh.SetActiveColorChannel(ChannelCombined)

	}

	// We can turn off the lights now, as we won't need them anymore.
	for _, light := range lights {
		light.SetOn(false)
	}

}

func (g *Game) Update() error {

	// Change the active vertex color channels
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		for _, model := range g.Scene.Root.ChildrenRecursive().Models() {
			model.Mesh.SelectVertices().SelectAll().SetActiveColorChannel(ChannelNone) // Switch to unlit
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		for _, model := range g.Scene.Root.ChildrenRecursive().Models() {
			model.Mesh.SelectVertices().SelectAll().SetActiveColorChannel(ChannelAO) // Switch to AO only
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		for _, model := range g.Scene.Root.ChildrenRecursive().Models() {
			model.Mesh.SelectVertices().SelectAll().SetActiveColorChannel(ChannelLight) // Switch to Lighting
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		for _, model := range g.Scene.Root.ChildrenRecursive().Models() {
			model.Mesh.SelectVertices().SelectAll().SetActiveColorChannel(ChannelCombined) // Switch to lighting+AO
		}
	}

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color - we can use the world lighting color for this.
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
	g.Camera.RenderScene(g.Scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := "This example shows how lighting and primitive\nambient occlusion can be baked into vertex colors.\n1 Key: Switch to unlit channel\n2 Key: Switch to only AO\n3 key: Switch to only lighting\n4 Key: Switch to lighting+AO"
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Baked Lighting Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
