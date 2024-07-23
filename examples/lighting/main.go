package main

import (
	"embed"
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed assets/*
var assets embed.FS

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

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFFileSystem(assets, "assets/lighting.gltf", nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.Move(0, 5, 0)

	g.System = examples.NewBasicSystemHandler(g)

	light := tetra3d.NewPointLight("camera light", 1, 1, 1, 2)
	light.Range = 10
	light.Move(0, 1, -2)
	light.On = false
	g.Camera.AddChildren(light)

}

func (g *Game) Update() error {

	spin := g.Scene.Root.Get("Spin").(*tetra3d.Model)
	spin.Rotate(0, 1, 0, 0.025)

	light := g.Scene.Root.Get("Point light").(*tetra3d.PointLight)
	light.AnimationPlayer().PlayByName("LightAction")
	light.AnimationPlayer().Update(1.0 / 60.0)

	armature := g.Scene.Root.Get("Armature")

	player := armature.AnimationPlayer()
	player.PlayByName("ArmatureAction")
	player.Update(1.0 / 60.0)

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.Scene.World.LightingOn = !g.Scene.World.LightingOn
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		pointLight := g.Camera.Get("camera light").(*tetra3d.PointLight)
		pointLight.On = !pointLight.On
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

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.

	if g.System.DrawDebugText {
		txt := "This example simply shows dynamic\nvertex-based lighting.\nThere are six lights in this scene:\nan ambient light, three point lights, \na single directional (sun) light,\nand one more point light parented to the camera.\n1 Key: Toggle all lighting\n2 Key: Toggle camera light"
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

	g.System.Draw(screen, g.Camera.Camera)
}

func (g *Game) Layout(w, h int) (int, int) {
	nw, nh := g.Camera.ColorTexture().Size()
	return nw, nh
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Lighting Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
