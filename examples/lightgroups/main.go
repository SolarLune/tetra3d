package main

import (
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed lighting.gltf
var gltfData []byte

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	Time float64
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(gltfData, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 2, 15)
	g.System = examples.NewBasicSystemHandler(g)

	gt := g.Scene.Root.Get("OnlyGreen").(*tetra3d.Model)
	gt.LightGroup = tetra3d.NewLightGroup(g.Scene.Root.Get("GreenLight").(*tetra3d.PointLight))

	rt := g.Scene.Root.Get("OnlyRed").(*tetra3d.Model)
	rt.LightGroup = tetra3d.NewLightGroup(g.Scene.Root.Get("RedLight").(*tetra3d.PointLight))

}

func (g *Game) Update() error {

	armature := g.Scene.Root.Get("Armature")

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		armature.Move(-0.1, 0, 0)
	} else if ebiten.IsKeyPressed(ebiten.KeyRight) {
		armature.Move(0.1, 0, 0)
	}

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		green := g.Scene.Root.Get("OnlyGreen").(*tetra3d.Model)
		green.LightGroup.Active = !green.LightGroup.Active

		red := g.Scene.Root.Get("OnlyRed").(*tetra3d.Model)
		red.LightGroup.Active = !red.LightGroup.Active
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.Scene.World.LightingOn = !g.Scene.World.LightingOn
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
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This example shows light groups, which
allow you to control how lights color specific Models.

There's two point lights in this scene - a red one
and a green one. By using LightGroups, the left cube 
is only lit by the green light, while the right
is only lit by the red one. The center cube is lit by both.

1 Key: Toggle light groups being active
2 Key: Toggle all lighting`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - LightGroup Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
