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

type Game struct {
	Library *tetra3d.Library
	Time    float64

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

//go:embed assets
var assets embed.FS

func NewGame() *Game {
	game := &Game{}
	game.Init()
	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFFile(assets, "assets/animations.glb", nil)
	if err != nil {
		panic(err)
	}

	g.Library = library

	scene := g.Library.Scenes[0].Clone()

	g.Camera = examples.NewBasicFreeCam(scene)
	g.Camera.Move(0, 0, 10)

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	scene := g.Library.Scenes[0]

	armature := scene.Root.SearchTree().ByName("Armature").First()
	armature.Rotate(0, 1, 0, 0.01)

	ap := armature.AnimationPlayer()

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if ap.Animation != nil && ap.Animation.Name == "ArmatureAction" {
			ap.Playing = !ap.Playing
		} else {
			ap.Play("ArmatureAction")
		}
	}

	ap.Update(1.0 / 60)

	table := scene.Root.Get("Table").(*tetra3d.Model)

	ap = table.AnimationPlayer()
	ap.BlendTime = 0.1
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		ap.Play("SmoothRoll")
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		ap.Play("StepRoll")
	}

	ap.Update(1.0 / 60)

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	scene := g.Library.Scenes[0]

	// Clear, but with a color
	screen.Fill(scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
	g.Camera.RenderScene(scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := `1 Key: Play [SmoothRoll] Animation On Table
2 Key: Play [StepRoll] Animation on Table
Note that models can blend between two different
animations (as shown on the table).
F Key: Play Animation on Skinned Mesh
Note that the nodes move as well (as can be
seen in the debug mode).`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Animations Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
