package main

import (
	"embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"
	"github.com/solarlune/tetra3d/math32"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed assets
var assetData embed.FS

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
	ChannelColor    = iota // The default first vertex color channel
	ChannelLight           // The light color channel
	ChannelAO              // The AO color channel
	ChannelCombined        // The combined color channel
)

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFFileSystem(assetData, "assets/baking.glb", nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0].Clone()

	// for _, mat := range library.Materials {
	// 	mat.TextureFilterMode = ebiten.FilterLinear
	// }

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.Move(0, 10, 0)
	g.System = examples.NewBasicSystemHandler(g)

	// OK, so in this example we're going to bake lighting and AO.
	// We will combine them into different vertex color channels to
	// easily preview the differences, as well.

	// We start by collecting all the lights we'll be baking. NodeFilter.Lights() will automatically give us just the lights out of our selection and discard any other INodes.
	lights := g.Scene.Root.SearchTree().ILights()

	// // Don't forget the world's ambient light!
	lights = append(lights, g.Scene.World.AmbientLight)

	// Let's get all the solid, occluding models here.
	// The idea is that we'll bake the lighting and AO of each ao-applicable Model.

	// We'll bake the lighting to vertex channel #0, AO to channel #1, and combine both
	// in vertex channel #2. We will use named consts to make referencing these channels simple.

	model := g.Scene.Get("GameMap").(*tetra3d.Model)

	// First, we'll bake the lighting into the lighting vertex color channel.
	model.BakeLighting(ChannelLight, lights...)

	// Next, we'll bake the AO into the AO vertex color channel.

	// The AOBakeOptions struct controls how our AO should be baked. For readability, we create it
	// here, though we should, of course, create it outside our model loop if the options don't differ
	// for each Model.
	bakeOptions := tetra3d.NewDefaultAOBakeOptions()
	bakeOptions.TargetMeshParts = []*tetra3d.MeshPart{ // Bake AO only to meshparts with the following materials
		model.Mesh.MeshPartByMaterialName("BrickTile"),
		model.Mesh.MeshPartByMaterialName("GroundTile"),
		model.Mesh.MeshPartByMaterialName("Cube"),
	}
	bakeOptions.OcclusionAngle = math32.ToRadians(60)
	bakeOptions.TargetChannel = ChannelAO // Set the target baking channel (as it's not 0)
	// bakeOptions.OtherModels = models      // Specify what other objects should influence the AO (note that this is optional)
	model.Mesh.SetVertexColor(ChannelAO, colors.White())
	model.BakeAO(bakeOptions) // And bake the AO.

	// We'll combine the channels together multiplicatively here.
	model.Mesh.CombineVertexColors(ChannelCombined, true, ChannelColor, ChannelLight, ChannelAO)

	for _, mat := range model.Mesh.Materials() {
		mat.Shadeless = true // We don't need lighting anymore.
	}

	// Finally, we'll set the models' active color channel here. By default, it's -1, indicating no vertex colors are active (unless
	// the mesh was exported from Blender with an active vertex color channel).
	model.Mesh.SetActiveColorChannel(ChannelCombined)

}

func (g *Game) Update() error {

	// Change the active vertex color channels
	gameMap := g.Scene.Root.Get("GameMap").(*tetra3d.Model)

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		gameMap.Mesh.Select().SetActiveColorChannel(ChannelColor)
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		gameMap.Mesh.Select().SetActiveColorChannel(ChannelAO)
	}

	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		gameMap.Mesh.Select().SetActiveColorChannel(ChannelLight)
	}

	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		gameMap.Mesh.Select().SetActiveColorChannel(ChannelCombined)
	}

	if inpututil.IsKeyJustPressed(ebiten.Key5) {
		for _, mp := range gameMap.Mesh.MeshParts {
			mp.Material.UseTexture = !mp.Material.UseTexture
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
		txt := `This example shows how lighting and primitive
ambient occlusion can be baked into vertex colors.
1 Key: Switch to vertex color-only channel 
2 Key: Switch to only AO
3 key: Switch to only lighting
4 Key: Switch to VC + AO + Lighting
5 Key: Enable / disable texture channel`
		g.Camera.DrawDebugText(screen, txt, 0, 220, 1, colors.LightGray())
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
