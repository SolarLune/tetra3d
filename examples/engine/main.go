package main

import (
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

//go:embed engine.gltf
var libraryData []byte

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(libraryData, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library

	// We clone the scene so we have an original to work from
	g.Scene = library.ExportedScene.Clone()

	for _, o := range g.Scene.Root.Children() {

		if o.Properties().Has("gameobject") {
			g.SetGameObject(o)
		}

	}

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 2, 5)

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) SetGameObject(o tetra3d.INode) {

	switch o.Properties().Get("gameobject").AsString() {

	case "player":
		player := NewPlayer(o)
		o.SetData(player)

	}

}

func (g *Game) CreateFromTemplate(node tetra3d.INode) {
	newObj := node.Clone()
	g.Scene.Root.AddChildren(newObj)
	g.SetGameObject(newObj)
}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.CreateFromTemplate(g.Library.Scenes[0].Root.Get("Player"))
	}

	for _, obj := range g.Scene.Root.ChildrenRecursive() {

		if data := obj.Data(); data != nil {
			if gObj, ok := data.(GameObject); ok {
				gObj.Update()
			}
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

	// Render the logo first
	g.Camera.RenderScene(g.Scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This demo is a small example showing how one could design a game
using a traditional 'class-based' approach with Tetra3D.

1 Key: Spawn player objects.
Arrow keys: Move player(s)
Touch spikes to destroy player(s)`

		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Engine Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
