package main

import (
	"bytes"
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

	DrawDebugText  bool
	DrawDebugDepth bool

	BG *ebiten.Image
}

//go:embed alpha-ordering.gltf
var alphaOrdering []byte

//go:embed bg.png
var bgPng []byte

func NewGame() *Game {
	game := &Game{
		ExampleGame:   examples.NewExampleGame(796, 448),
		DrawDebugText: true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {
	data, err := tetra3d.LoadGLTFData(alphaOrdering, nil)
	if err != nil {
		panic(err)
	}

	g.Scene = data.Scenes[0]

	br := bytes.NewReader(bgPng)
	img, err := png.Decode(br)
	if err != nil {
		panic(err)
	}
	g.BG = ebiten.NewImageFromImage(img)

	// This is another way to do it
	// screen := g.Scene.Root.Get("Screen").(*tetra3d.Model)
	// screen.Mesh.FindMeshPartByMaterialName("ScreenTexture").Material.Image = g.Offscreen

	g.SetupCameraAt(vector.Vector{0, 0, 5})

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)
}

func (g *Game) Update() error {
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

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	screen.Clear()

	// Draw BG
	opt := &ebiten.DrawImageOptions{}
	w, h := g.BG.Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	screen.DrawImage(g.BG, opt)

	// Clear the Camera
	g.Camera.Clear()

	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	opt = &ebiten.DrawImageOptions{}
	w, h = g.Camera.ColorTexture().Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), opt)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), opt)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugText(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nThis demo shows how composite modes work.\nThe blue plane is opaque.\nThe red one is additive.\nThe green one is transparent.\nThe closest plane cuts out all objects to show the background.\n\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 108, color.RGBA{255, 0, 0, 255})
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
