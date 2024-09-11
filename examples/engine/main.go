package main

import (
	"bytes"
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The goal of this example is to show how a simple engine object-oriented game engine flow could be approximated in Tetra3D.

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	Watcher *tetra3d.TreeWatcher
}

//go:embed engine.gltf
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

	// We clone the scene so we have an original to work from
	g.Scene = library.ExportedScene.Clone()

	// By creating and updating the Watcher, whenever anything under
	// the watched Node's hierarchy changes, the attached function will run.
	g.Watcher = tetra3d.NewTreeWatcher(g.Scene.Root, func(node tetra3d.INode) {

		if node.Properties().Has("gameobject") {

			switch node.Properties().Get("gameobject").AsString() {

			case "player":
				player := NewPlayer(node)
				node.SetData(player)

			}

		}

	})

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 2, 5)

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		// Cloning a Node is as simple as getting it (from any Scene), Cloning it, and adding it under the hierarchy of your target Scene.
		player := g.Library.Scenes[0].Root.Get("Player")
		newPlayer := player.Clone()
		g.Scene.Root.AddChildren(newPlayer)
	}

	g.Watcher.Update() // Update the TreeWatcher.

	// By using NodeFilter.ForEach(), we avoid allocating a new Slice just to iterate through a tree.
	g.Scene.Root.SearchTree().ForEach(func(node tetra3d.INode) bool {

		if node.Data() != nil {

			d, ok := node.Data().(GameObject)
			if ok {
				d.Update()
			}

		}

		return true
	})

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

		g.Camera.DebugDrawText(screen, txt, 0, 220, 1, colors.LightGray())
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
