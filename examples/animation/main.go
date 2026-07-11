package main

import (
	"embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Library *tetra3d.Library
	Time    float32

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

	library, err := tetra3d.LoadGLTFFileSystem(assets, "assets/animations.glb", nil)
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

	scene := g.Camera.Scene

	armature := scene.Root.Search().ByNames("Armature").GetFirst()
	armature.Rotate(0, 1, 0, 0.01)

	ap := armature.AnimationPlayer()

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if ap.Animation() != nil && ap.Animation().Name() == "ArmatureAction" {
			if ap.IsPlaying() {
				ap.Pause()
			} else {
				ap.Resume()
			}
		} else {
			ap.PlayByName("ArmatureAction")
		}
	}

	ap.Update(1.0 / 60)

	table := scene.Root.Get("Table").(*tetra3d.Model)

	ap = table.AnimationPlayer()
	ap.BlendTime = 0.1
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		ap.PlayByName("SmoothRoll")
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		ap.PlayByName("StepRoll")
	}

	// The morphing cube has shape keys that you can use to animate per-vertex animations for the model
	morpher := scene.Root.Get("MorphingCube").(*tetra3d.Model)

	if ebiten.IsKeyPressed(ebiten.Key3) {
		target := morpher.Mesh().ShapeKeyByName("Diamond")
		w := target.Weight() - 0.01
		if w < 0 {
			w = 0
		}
		target.SetWeight(w)
	}
	if ebiten.IsKeyPressed(ebiten.Key4) {
		target := morpher.Mesh().ShapeKeyByName("Diamond")
		w := target.Weight() + 0.01
		if w > 1 {
			w = 1
		}
		target.SetWeight(w)
	}

	if ebiten.IsKeyPressed(ebiten.Key5) {
		target := morpher.Mesh().ShapeKeyByName("Shift")
		w := target.Weight() - 0.01
		if w < 0 {
			w = 0
		}
		target.SetWeight(w)
	}
	if ebiten.IsKeyPressed(ebiten.Key6) {
		target := morpher.Mesh().ShapeKeyByName("Shift")
		w := target.Weight() + 0.01
		if w > 1 {
			w = 1
		}
		target.SetWeight(w)
	}

	ap.Update(1.0 / 60)

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	scene := g.Camera.Scene

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
		g.Camera.DebugInfo.Draw(screen, 1, colors.White())
		txt := `1 Key: Play [SmoothRoll] Animation On Table
2 Key: Play [StepRoll] Animation on Table
(Note that models can blend between two different
animations (as shown on the table).)
3 / 4 Key: Morph the Cube into diamond, 5 / 6 : Shift its vertices upwards

F Key: Play Animation on Skinned Mesh - note that the nodes making up its bones 
move as well (as can be seen in the debug mode).`
		tetra3d.DrawDebugText(screen, txt, 0, 230, 1, colors.LightGray())
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
