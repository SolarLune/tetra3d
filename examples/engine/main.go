package main

import (
	"bytes"
	"image/color"
	"slices"

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

	// There's two ways to do this. One is to create data objects that represent our game objects, and then place our models in our scenes from there,
	// or to go in the reverse direction and create models in our game scenes, and then create data objects to match. This example uses the latter approach.

	// An easy way to do this is to treat the original scenes in our tetra3d.Library as "prototype" scenes that we clone to instantiate your game level / map.

	// We register relevant objects as reporting callbacks so that when the scene they're in is cloned, or they are
	// reparented, we handle this to add game object logic controls to our g.GameObjects slice.

	for _, scene := range library.Scenes {

		// Register nodes that have a "gameobject" property for scene tree callbacks. Now whenever it's reparented (or part of a Scene that is cloned),
		// tetra3d.Callbacks.OnReparent will be called, allowing us to easily create the data object easily.
		scene.Root.SearchTree().ByPropNameRegex("gameobject").ForEach(func(node tetra3d.INode) bool {
			node.SetRegisteredForSceneTreeCallbacks(true)
			return true
		})

		// Called when a relevant gameobject node is reparented or in a scene that is cloned.
		// In other words, whenever a node
		tetra3d.Callbacks.OnNodeReparent = func(node tetra3d.INode, oldParent tetra3d.INode, newParent tetra3d.INode) {

			if node.Properties().Has("gameobject") {

				if newParent != nil && oldParent == nil && node.Data() == nil {
					// Reparenting into scene tree

					switch node.Properties().Get("gameobject").AsString() {

					case "player":
						player := NewPlayer(node)
						g.GameObjects = append(g.GameObjects, player)
						node.SetData(player)
					}

				} else if newParent == nil && oldParent != nil && node.Data() != nil {
					// Reparenting out of scene tree (removing)

					for i := len(g.GameObjects) - 1; i >= 0; i-- {
						if g.GameObjects[i] == node.Data() {
							// g.GameObjects[i] = nil
							// g.GameObjects = append(g.GameObjects[:i], g.GameObjects[i+1:]...)
							// break
							g.GameObjects = slices.Delete(g.GameObjects, i, i+1)
						}
					}

				}

			}

		}

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
		newPlayer := player.Clone()
		g.Scene.Root.AddChildren(newPlayer)
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
