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

	_ "image/png"

	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene

	Camera       *tetra3d.Camera
	CameraTilt   float64
	CameraRotate float64

	Time              float64
	DrawDebugText     bool
	DrawDebugDepth    bool
	PrevMousePosition vector.Vector
}

func NewGame() *Game {

	game := &Game{
		Width:             398,
		Height:            224,
		PrevMousePosition: vector.Vector{},
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	g.Scene = tetra3d.NewScene("Test Scene")

	loaded, err := tetra3d.LoadDAEFile("../shapes/shapes.dae", nil)
	fmt.Println(err)

	// cubeMesh := tetra3d.NewCube()
	// cubeMesh.Image, _, _ = ebitenutil.NewImageFromFile("testimage.png")

	cube := loaded.FindNode("hpcube").Clone()
	g.Scene.Root.AddChildren(cube)

	// for i := 0; i < 10; i++ {
	// 	for j := 0; j < 3; j++ {
	// 		// cube := tetra3d.NewModel(cubeMesh, "Cube")
	// 		cube := loaded.FindNode("hpcube").Clone()
	// 		fmt.Println(cube)
	// 		cube.LocalPosition[0] = float64(i * 3)
	// 		cube.LocalPosition[2] = float64(j * 3)
	// 		g.Scene.Root.AddChildren(cube)
	// 	}
	// }

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.LocalPosition[2] = 15
	// g.Camera.Far = 20
	// g.Camera.RenderDepth = false

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {

	var err error

	moveSpd := 0.1

	g.Time += 1.0 / 60

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Moving the Camera

	// We use Camera.Rotation.Forward().Invert() because the camera looks down -Z (so its forward vector is inverted)
	forward := g.Camera.LocalRotation.Forward().Invert()
	right := g.Camera.LocalRotation.Right()

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.Camera.LocalPosition = g.Camera.LocalPosition.Add(forward.Scale(moveSpd))
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.Camera.LocalPosition = g.Camera.LocalPosition.Add(right.Scale(moveSpd))
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.Camera.LocalPosition = g.Camera.LocalPosition.Add(forward.Scale(-moveSpd))
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.Camera.LocalPosition = g.Camera.LocalPosition.Add(right.Scale(-moveSpd))
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.Camera.LocalPosition[1] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		g.Camera.LocalPosition[1] -= moveSpd
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// Rotating the camera with the mouse

	// Rotate and tilt the camera according to mouse movements
	mx, my := ebiten.CursorPosition()

	mv := vector.Vector{float64(mx), float64(my)}

	diff := mv.Sub(g.PrevMousePosition)

	g.CameraTilt -= diff[1] * 0.005
	g.CameraRotate -= diff[0] * 0.005

	g.CameraTilt = math.Max(math.Min(g.CameraTilt, math.Pi/2-0.1), -math.Pi/2+0.1)

	tilt := tetra3d.NewMatrix4Rotate(1, 0, 0, g.CameraTilt)
	rotate := tetra3d.NewMatrix4Rotate(0, 1, 0, g.CameraRotate)

	// Order of this is important - tilt * rotate works, rotate * tilt does not, lol
	g.Camera.LocalRotation = tilt.Mult(rotate)

	g.PrevMousePosition = mv.Clone()

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		f, err := os.Create("screenshot" + time.Now().Format("2006-01-02 15:04:05") + ".png")
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		png.Encode(f, g.Camera.ColorTexture)
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
		g.Camera.DebugDrawWireframe = !g.Camera.DebugDrawWireframe
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.Camera.DebugDrawNormals = !g.Camera.DebugDrawNormals
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
		time.Sleep(2 * time.Second)
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
	w, h := g.Camera.ColorTexture.Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture, opt)
	} else {
		screen.DrawImage(g.Camera.ColorTexture, opt)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugText(screen, 1)
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
