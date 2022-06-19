package main

import (
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"math"
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

//go:embed transparency.gltf
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
		ExampleGame:   examples.NewExampleGame(1920, 1080),
		DrawDebugText: true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	opt := tetra3d.DefaultGLTFLoadOptions()
	opt.CameraWidth = g.Width
	opt.CameraHeight = g.Height
	library, err := tetra3d.LoadGLTFData(gltfData, opt)
	if err != nil {
		panic(err)
	}

	g.Library = library
	library.Materials["Gate"].BackfaceCulling = false
	g.Scene = library.Scenes[0]

	g.SetupCameraAt(vector.Vector{0, 5, 15})
	g.Camera.Far = 30

	ambientLight := tetra3d.NewAmbientLight("ambient", 0.8, 0.9, 1, 0.5)
	g.Scene.Root.AddChildren(ambientLight)

	g.Scene.Root.Get("Water").(*tetra3d.Model).Color.A = 0.6

	g.Scene.FogMode = tetra3d.FogOverwrite
	g.Scene.FogColor = tetra3d.NewColor(0.8, 0.9, 1, 1)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	water := g.Scene.Root.Get("Water").(*tetra3d.Model)

	water.Mesh.MeshParts[0].Material.VertexProgram = func(v vector.Vector) vector.Vector {
		v[1] += math.Sin((g.Time*math.Pi)+(v[0]*1.2)+(v[2]*0.739)) * 0.007
		return v
	}

}

func (g *Game) Update() error {

	g.Time += 1.0 / 60.0

	var err error

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// We toggle transparency here; note that we can also just leave the mode set to TransparencyModeAuto for the water, instead; then, we would just need to make the material or object color transparent.
	// Tetra3D would then treat the material as though it had TransparencyModeTransparent set.
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		if g.Library.Materials["TransparentSquare"].TransparencyMode == tetra3d.TransparencyModeOpaque {
			g.Library.Materials["TransparentSquare"].TransparencyMode = tetra3d.TransparencyModeAlphaClip
			g.Library.Materials["Water"].TransparencyMode = tetra3d.TransparencyModeTransparent
		} else {
			g.Library.Materials["TransparentSquare"].TransparencyMode = tetra3d.TransparencyModeOpaque
			g.Library.Materials["Water"].TransparencyMode = tetra3d.TransparencyModeOpaque
		}
	}

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

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear, but with a color
	screen.Fill(g.Scene.FogColor.ToRGBA64())

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

		g.Camera.DrawDebugText(screen, 1, colors.Black())

		transparencyOn := "On"
		if g.Library.Materials["TransparentSquare"].TransparencyMode == tetra3d.TransparencyModeOpaque {
			transparencyOn = "Off"
		}

		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\n" +
			"Transparent materials draw in a second pass;\n" +
			"they don't appear in the depth texture.\n" +
			"They can overlap (bad), but also show things\n" +
			"underneath (good).\n" +
			"1 Key: Toggle transparency, currently: " + transparencyOn +
			"\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"

		text.Draw(screen, txt, basicfont.Face7x13, 0, 120, color.RGBA{255, 0, 0, 255})

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
