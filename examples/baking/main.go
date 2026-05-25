package main

import (
	"embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

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
	lights := g.Scene.Root.Search(tetra3d.SearchOptions{}.ByType(tetra3d.NodeTypeLight))

	// Let's get all the solid, occluding models here.
	// The idea is that we'll bake the lighting and AO of each ao-applicable Model.

	// We'll bake the lighting to vertex channel #0, AO to channel #1, and combine both
	// in vertex channel #2. We will use named consts to make referencing these channels simple.

	model := g.Scene.Get("GameMap").(*tetra3d.Model)
	mesh := model.Mesh()

	// First, we'll bake the lighting into the lighting vertex color channel.
	model.BakeLighting(ChannelLight, g.Camera.Camera, lights)

	lights.ForEachLight(func(light tetra3d.ILight) bool {
		light.SetVisible(false, false)
		return true
	})

	// We'll combine the channels together multiplicatively here.
	mesh.CombineVertexColors(ChannelCombined, true, ChannelColor, ChannelLight)

	for _, mat := range mesh.Materials() {
		mat.Shadeless = true // We don't need lighting anymore.
	}

	// Finally, we'll set the models' active color channel here. By default, it's -1, indicating no vertex colors are active (unless
	// the mesh was exported from Blender with an active vertex color channel).
	model.Mesh().VertexActiveColorChannel = ChannelCombined

}

func (g *Game) Update() error {

	// Change the active vertex color channels
	gameMap := g.Scene.Root.Get("GameMap").(*tetra3d.Model)

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		gameMap.Mesh().VertexActiveColorChannel = ChannelColor
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		gameMap.Mesh().VertexActiveColorChannel = ChannelLight
	}

	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		gameMap.Mesh().VertexActiveColorChannel = ChannelCombined
	}

	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		for _, mp := range gameMap.Mesh().MeshParts {
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
		tetra3d.DrawDebugText(screen, txt, 0, 230, 1, colors.LightGray())
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
