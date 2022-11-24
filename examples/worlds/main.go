package main

import (
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"runtime/pprof"
	"time"

	_ "embed"

	"github.com/quartercastle/vector"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene
	Library       *tetra3d.Library

	Camera       *tetra3d.Camera
	CameraTilt   float64
	CameraRotate float64
	Offscreen    *ebiten.Image

	DrawDebugText     bool
	DrawDebugDepth    bool
	PrevMousePosition vector.Vector
}

//go:embed worlds.gltf
var sceneData []byte

func NewGame() *Game {
	game := &Game{
		Width:             398,
		Height:            224,
		PrevMousePosition: vector.Vector{},
		DrawDebugText:     true,
	}
	game.Offscreen = ebiten.NewImage(game.Width, game.Height)

	game.Init()

	return game
}

func (g *Game) Init() {
	library, err := tetra3d.LoadGLTFData(sceneData, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.ExportedScene.Clone()

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.Far = 30
	g.Camera.SetLocalPositionVec(vector.Vector{0, 0, 5})
	g.Scene.Root.AddChildren(g.Camera)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {
	var err error

	moveSpd := 0.05

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Moving the Camera

	// We use Camera.Rotation.Forward().Invert() because the camera looks down -Z (so its forward vector is inverted)
	forward := g.Camera.LocalRotation().Forward().Invert()
	right := g.Camera.LocalRotation().Right()

	pos := g.Camera.LocalPosition()

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		pos = pos.Add(forward.Scale(moveSpd))
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		pos = pos.Add(right.Scale(moveSpd))
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		pos = pos.Add(forward.Scale(-moveSpd))
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		pos = pos.Add(right.Scale(-moveSpd))
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		pos[1] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		pos[1] -= moveSpd
	}

	g.Camera.SetLocalPositionVec(pos)

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
	g.Camera.SetLocalRotation(tilt.Mult(rotate))

	g.PrevMousePosition = mv.Clone()

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		f, err := os.Create("screenshot" + time.Now().Format("2006-01-02 15:04:05") + ".png")
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		png.Encode(f, g.Camera.ColorTexture())
	}

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.Scene.World = g.Library.Worlds["Dark"]
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.Scene.World = g.Library.Worlds["Bright"]
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.Scene.World = g.Library.Worlds["Blue"]
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		g.Scene.World = g.Library.Worlds["Red"]
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
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), nil)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), nil)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\n1-4 Keys: Cycle Worlds\nEach World can have its own clear color,\nfog color and range, and ambient color.\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		g.Camera.DebugDrawText(screen, txt, 0, 120, 1, colors.White())
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
	ebiten.SetWindowTitle("Tetra3d - Worlds Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
