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

// The goal of this example is to show how a simple object-oriented game engine-type flow could be approximated in Tetra3D.

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	GameObjects []GameObject
	ToRemove    []GameObject
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

	// This example's focus is on creating an "engine" to easily update game objects.

	// There's two ways to do this. One is to create data objects that represent our game objects, and then place our models in our scenes from there,
	// or to go in the reverse direction and create models in our game scenes, and then create data objects to match. This example uses the latter approach.

	// We will treat the original scenes in our tetra3d.Library as "prototype" scenes that we clone to instantiate your game level / map.
	// We loop through objects that we have tagged to be game objects and simply set callbacks so that when they are cloned or altered, we add or remove
	// them from a GameObjects slice. You could also just loop through the scene tree directly for a simpler approach, but you would be leaving performance
	// on the table in that case.

	for _, scene := range library.Scenes {

		// Set up callbacks for each relevant node that creates the necessary data.
		scene.Root.SearchTree().ByPropNameRegex("gameobject").ForEach(func(node tetra3d.INode) bool {
			switch node.Properties().Get("gameobject").AsString() {
			case "player":

				node.Callbacks().OnClone = func(newNode tetra3d.INode) {
					// When the Player node is cloned, we create a new Player object for its data holder.
					p := NewPlayer(newNode)
					newNode.SetData(p)
					g.GameObjects = append(g.GameObjects, p)
				}

				node.Callbacks().OnReparent = func(node, oldParent, newParent tetra3d.INode) {
					// When the Player node is unparented, we remove the Player game object from the GameObjects slice.
					if newParent == nil {
						g.ToRemove = append(g.ToRemove, node.Data().(GameObject))
					}
				}

			}
			return true
		})

	}

	g.Library = library

	// We clone the scene, and that's it!
	g.Scene = library.Scenes[0].Clone()

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 2, 5)

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		// Cloning a Node is as simple as getting it (from any Scene), Cloning it, and adding it under the hierarchy of your target Scene.
		player := g.Library.Scenes[0].Root.Get("Player")
		g.Scene.Root.AddChildren(player.Clone())
	}

	for i := len(g.GameObjects) - 1; i >= 0; i-- {
		g.GameObjects[i].Update()
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

		g.Camera.DrawDebugText(screen, txt, 0, 220, 1, colors.LightGray())
	}

	for _, r := range g.ToRemove {
		for i, obj := range g.GameObjects {
			if obj == r {
				g.GameObjects[i] = nil
				g.GameObjects = append(g.GameObjects[:i], g.GameObjects[i+1:]...)
			}
		}
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
