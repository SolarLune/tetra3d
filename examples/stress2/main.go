package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime/pprof"
	"time"

	_ "embed"

	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	examples "github.com/solarlune/tetra3d/examples"
)

//go:embed testimage.png
var testImage []byte

type Game struct {
	examples.ExampleGame

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugWireframe bool
	DrawDebugNormals   bool
}

func NewGame() *Game {

	game := &Game{
		ExampleGame: examples.NewExampleGame(398, 224),
	}
	game.CamMoveSpeed = 0.1

	game.Init()

	return game
}

func (g *Game) Init() {

	g.Scene = tetra3d.NewScene("Test Scene")

	img, _, err := image.Decode(bytes.NewReader(testImage))
	if err != nil {
		panic(err)
	}

	// We can reuse the mesh for all of the models.
	cubeMesh := tetra3d.NewCube()

	mat := cubeMesh.MeshParts[0].Material
	mat.Shadeless = true
	mat.Texture = ebiten.NewImageFromImage(img)

	for i := 0; i < 21; i++ {
		for j := 0; j < 21; j++ {
			// Create a new Model, position it, and add it to the cubes slice.
			cube := tetra3d.NewModel(cubeMesh, "Cube")
			cube.SetLocalPosition(vector.Vector{float64(i * 3), 0, float64(-j * 3)})
			g.Scene.Root.AddChildren(cube)
		}
	}

	g.SetupCameraAt(vector.Vector{0, 0, 15})
	g.Camera.Far = 120

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

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.DrawDebugWireframe = !g.DrawDebugWireframe
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.DrawDebugNormals = !g.DrawDebugNormals
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	return err

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
		time.Sleep(10 * time.Second)
		pprof.StopCPUProfile()
		fmt.Println("CPU profiling finished.")
	}()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the non-screen Models
	// g.Camera.Render(g.Scene, g.Scene.Models...)

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

	if g.DrawDebugWireframe {
		g.Camera.DrawDebugWireframe(screen, g.Scene.Root, colors.Red())
	}

	if g.DrawDebugNormals {
		g.Camera.DrawDebugNormals(screen, g.Scene.Root, 0.25, colors.Blue())
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugText(screen, 1, colors.White())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {

	ebiten.SetWindowTitle("Tetra3d Test - Stress Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
