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
)

//go:embed lighting.gltf
var gltfData []byte

type Game struct {
	Width, Height int
	Library       *tetra3d.Library
	Scene         *tetra3d.Scene

	Camera       *tetra3d.Camera
	CameraTilt   float64
	CameraRotate float64

	DrawDebugText     bool
	DrawDebugDepth    bool
	PrevMousePosition vector.Vector

	Time float64
}

func NewGame() *Game {
	game := &Game{
		Width:             1920,
		Height:            1080,
		PrevMousePosition: vector.Vector{},
		DrawDebugText:     true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	opt := tetra3d.DefaultGLTFLoadOptions()
	opt.CameraWidth = 1920
	opt.CameraHeight = 1080
	opt.LoadBackfaceCulling = true
	library, err := tetra3d.LoadGLTFData(gltfData, opt)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.Camera = tetra3d.NewCamera(1920, 1080)
	g.Camera.SetLocalPosition(vector.Vector{0, 2, 15})
	g.Scene.Root.AddChildren(g.Camera)

	// g.Scene.FogMode = tetra3d.FogMultiply

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {

	g.Time += 1.0 / 60.0

	var err error

	moveSpd := 0.05

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	spin := g.Scene.Root.Get("Spin").(*tetra3d.Model)
	spin.Rotate(0, 1, 0, 0.025)

	light := g.Scene.Root.Get("Point light").(*tetra3d.Node)
	light.AnimationPlayer().Play(g.Library.Animations["LightAction"])
	light.AnimationPlayer().Update(1.0 / 60.0)

	// g.Scene.Root.Get("plane").Rotate(1, 0, 0, 0.04)

	// light := g.Scene.Root.Get("third point light")
	// light.Move(math.Sin(g.Time*math.Pi)*0.1, 0, math.Cos(g.Time*math.Pi*0.19)*0.03)

	// Moving the Camera

	// We use Camera.Rotation.Forward().Invert() because the camera looks down -Z (so its forward vector is inverted)
	forward := g.Camera.LocalRotation().Forward().Invert()
	right := g.Camera.LocalRotation().Right()

	pos := g.Camera.LocalPosition()

	// g.Scene.Root.Get("point light").(*tetra3d.PointLight).Distance = 10 + (math.Sin(g.Time*math.Pi) * 5)

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

	g.Camera.SetLocalPosition(pos)

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
	w, h := g.Camera.ColorTexture.Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture, opt)
	} else {
		screen.DrawImage(g.Camera.ColorTexture, opt)
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
