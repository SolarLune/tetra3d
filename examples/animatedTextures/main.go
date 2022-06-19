package main

import (
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"runtime/pprof"
	"time"

	_ "embed"

	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	examples "github.com/solarlune/tetra3d/examples"
)

type Game struct {
	examples.ExampleGame

	Library *tetra3d.Library

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugWireframe bool

	AnimatedTexture *tetra3d.TexturePlayer
}

//go:embed animatedTextures.gltf
var libraryData []byte

func NewGame() *Game {
	game := &Game{
		ExampleGame:   examples.NewExampleGame(796, 448),
		DrawDebugText: true,
	}

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

	// Turn off lighting
	g.Scene.LightingOn = false

	g.SetupCameraAt(vector.Vector{0, 5, 10})

	// Firstly, we create a TexturePlayer, which animates a collection of vertices' UV values to
	// animate a texture on them.
	g.AnimatedTexture = tetra3d.NewTexturePlayer(library.Meshes["Plane"].MeshParts[0].Vertices)

	// We create the animation here; rather than having a function to create the struct, it's simpler to do so
	// manually, as can be seen below.
	bloopAnim := &tetra3d.TextureAnimation{
		FPS: 15,
		Frames: []vector.Vector{
			{0, 0},     // UV offset for frame 0
			{0.5, 0},   // ... For frame 1,
			{0, 0.5},   // ... For frame 2,
			{0.5, 0.5}, // ... And for frame 3.
		},
	}

	// The above offsets are relative to the starting UV values as they are set when we pass our vertices into the TexturePlayer instance.

	// Begin playing the animation.
	g.AnimatedTexture.Play(bloopAnim)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {

	// Update the TexturePlayer
	g.AnimatedTexture.Update(1.0 / 60.0)

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.AnimatedTexture.Playing = !g.AnimatedTexture.Playing
	}

	var err error

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	g.ProcessCameraInputs()

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		f, err := os.Create("screenshot" + time.Now().Format("2006-01-02 15:04:05") + ".png")
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		png.Encode(f, g.Camera.ColorTexture())
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.Init()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.StartProfiling()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.DrawDebugText = !g.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.DrawDebugWireframe = !g.DrawDebugWireframe
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	opt := &ebiten.DrawImageOptions{}
	w, h := g.Camera.ColorTexture().Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), opt)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), opt)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugText(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\n\nThis demo shows how animated textures work.\nThere are several planes, but they all share\nthe same mesh, which is animated by the\nTexturePlayer.\n1 key: Toggle playback\n\nF2:Toggle wireframe\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 108, color.RGBA{255, 0, 0, 255})
	}

	if g.DrawDebugWireframe {
		g.Camera.DrawDebugWireframe(screen, g.Scene.Root, colors.Gray())
	}

}

func (g *Game) StartProfiling() {
	outFile, err := os.Create("./cpu.pprof")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Beginning CPU profiling...")
	pprof.StartCPUProfile(outFile)
	go func() {
		time.Sleep(2 * time.Second)
		pprof.StopCPUProfile()
		fmt.Println("CPU profiling finished.")
	}()
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Logo Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
