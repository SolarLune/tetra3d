package main

import (
	"embed"
	"fmt"

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

	library, err := tetra3d.LoadGLTFFileSystem(assets, "assets/anim_queueing.glb", nil)
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

	cube := scene.Root.Search().ByNames("Cube").GetFirst()
	ap := cube.AnimationPlayer()

	// You can also sequence animation playback by calling AnimationPlayer.Play() or PlayByName() with multiple animations,
	// or by playing an animation and then using the returned AnimationSequence object to do other things.

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ap.Pause() // Stops the currently playing animation if there is one to start from the beginning
		ap.PlayByName("Slam", "Slam", "Slam", "Spin").Then(func(ap *tetra3d.AnimationPlayer, loopCount int) { fmt.Println("done!") })
	}

	// Append an animation to the existing sequence
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		ap.Trigger().ThenPlayByName("Slam")
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		ap.Trigger().ThenPlayByName("Spin")
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
		txt := `F Key: Play "Slam" 3 times, then "Spin", then print a message
1 / 2 Key: Append "Slam" or "Spin" animation onto the queue`
		tetra3d.DrawDebugText(screen, txt, 0, 230, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Animation Queueing Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
