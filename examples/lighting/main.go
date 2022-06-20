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

//go:embed lighting.gltf
var gltfData []byte

type Game struct {
	examples.ExampleGame

	Library *tetra3d.Library

	DrawDebugText  bool
	DrawDebugDepth bool

	Time float64
}

func NewGame() *Game {
	game := &Game{
		ExampleGame:   examples.NewExampleGame(796, 448),
		DrawDebugText: true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	opt := tetra3d.DefaultGLTFLoadOptions()
	opt.CameraWidth = g.Width
	opt.CameraHeight = g.Height
	opt.LoadBackfaceCulling = true
	library, err := tetra3d.LoadGLTFData(gltfData, opt)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.SetupCameraAt(vector.Vector{0, 2, 15})

	// g.Scene.FogMode = tetra3d.FogMultiply

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {

	g.Time += 1.0 / 60.0

	var err error

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	spin := g.Scene.Root.Get("Spin").(*tetra3d.Model)
	spin.Rotate(0, 1, 0, 0.025)

	light := g.Scene.Root.Get("Point light").(*tetra3d.PointLight)
	light.AnimationPlayer().Play(g.Library.Animations["LightAction"])
	light.AnimationPlayer().Update(1.0 / 60.0)

	// g.Scene.Root.Get("plane").Rotate(1, 0, 0, 0.04)

	// light := g.Scene.Root.Get("third point light")
	// light.Move(math.Sin(g.Time*math.Pi)*0.1, 0, math.Cos(g.Time*math.Pi*0.19)*0.03)

	g.ProcessCameraInputs()

	// g.Scene.Root.Get("point light").(*tetra3d.PointLight).Distance = 10 + (math.Sin(g.Time*math.Pi) * 5)

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

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.Scene.LightingOn = !g.Scene.LightingOn
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color - we can use the world lighting color for this.
	light := g.Scene.Root.Get("World Ambient").(*tetra3d.AmbientLight)
	energy := light.Energy
	bgColor := light.Color.Clone()
	bgColor.MultiplyRGBA(energy, energy, energy, 1)
	screen.Fill(bgColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
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
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nThis example simply shows dynamic vertex-based lighting.\n1 Key: Toggle lighting\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 150, color.RGBA{255, 0, 0, 255})
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
	ebiten.SetWindowTitle("Tetra3d - Lighting Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
