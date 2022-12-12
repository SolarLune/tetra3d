package main

import (
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed shapes.gltf
var shapes []byte

type Game struct {
	Scene *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	// Load the GLTF file and turn it into a Library, which is a collection of scenes and data shared between them (like meshes or animations).

	library, err := tetra3d.LoadGLTFData(shapes, nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.FindScene("Scene")

	g.Camera = examples.NewBasicFreeCam(g.Scene)

	g.Camera.Far = 30

}

func (g *Game) Update() error {

	// Fog controls
	if ebiten.IsKeyPressed(ebiten.Key1) {
		g.Scene.World.FogColor.Set(1, 0, 0, 1)
		g.Scene.World.FogMode = tetra3d.FogAdd
	} else if ebiten.IsKeyPressed(ebiten.Key2) {
		g.Scene.World.FogColor.Set(0, 0, 0, 1)
		g.Scene.World.FogMode = tetra3d.FogMultiply
	} else if ebiten.IsKeyPressed(ebiten.Key3) {
		g.Scene.World.FogColor.Set(0, 0, 0, 1)
		g.Scene.World.FogMode = tetra3d.FogOverwrite
	} else if ebiten.IsKeyPressed(ebiten.Key4) {
		g.Scene.World.FogColor = colors.White()
		g.Scene.World.FogMode = tetra3d.FogOverwrite
	} else if ebiten.IsKeyPressed(ebiten.Key5) {
		g.Scene.World.FogMode = tetra3d.FogOff
		g.Scene.World.FogColor = colors.Black() // With the fog being off, setting the color doesn't do anything directly, but the clear color is set below to the fog color
	}

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.FogColor.ToRGBA64())

	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `1: Change fog to red additive
2: Change fog to black multiply
3: Change fog to black overwrite
4: Change fog to white overwrite
5: Turn off fog`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Shapes Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
