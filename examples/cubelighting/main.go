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

//go:embed cubeLighting.gltf
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
		Width:             796,
		Height:            448,
		PrevMousePosition: vector.Vector{},
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
	g.Scene = library.Scenes[0]

	g.Scene.World.AmbientLight.On = false

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.SetLocalPositionVec(vector.Vector{0, 10, 15})
	g.Scene.Root.AddChildren(g.Camera)

	for _, cubeLightModel := range g.Scene.Root.ChildrenRecursive().ByName("CubeLightVolume", false).Models() {

		cubeLight := tetra3d.NewCubeLightFromModel("cube light", cubeLightModel)
		cubeLight.Energy = 3
		g.Scene.Root.AddChildren(cubeLight)

	}

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {

	// for _, cube := range g.Scene.Root.ChildrenRecursive().ByTags("cube") {
	// 	// cube.Move(0, math.Sin(g.Time*math.Pi)*0.1, 0)
	// 	cube.Move(0, 0, -0.1)
	// }

	cubeLight := g.Scene.Root.Get("cube light").(*tetra3d.CubeLight)
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		vector.In(cubeLight.LightingAngle).Rotate(0.1, vector.X)
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		vector.In(cubeLight.LightingAngle).Rotate(-0.1, vector.X)
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		cubeLight.Bleed += 0.05
		if cubeLight.Bleed > 1 {
			cubeLight.Bleed = 1
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		cubeLight.Bleed -= 0.05
		if cubeLight.Bleed < 0 {
			cubeLight.Bleed = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		if cubeLight.Distance == 0 {
			cubeLight.Distance = 25
		} else {
			cubeLight.Distance = 0
		}
	}

	g.Time += 1.0 / 60.0

	var err error

	moveSpd := 0.15

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

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
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

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.Scene.World.LightingOn = !g.Scene.World.LightingOn
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color - we can use the world lighting color for this.
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), nil)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), nil)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\n\nThis example shows a Cube Light.\nCube Lights are volumes that shine from the top down.\nIf the light's distance is greater than 0, then the\nlight will be brighter towards the top.\nTriangles that lie outside the (AABB)\nvolume remain unlit.\n\nE Key: Toggle light distance\nLeft / Right Arrow Key: Rotate Light\nUp / Down Arrow Key: Increase / Decrease Bleed\n2 Key: Toggle all lighting\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		g.Camera.DebugDrawText(screen, txt, 0, 150, 1, colors.LightGray())
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
	ebiten.SetWindowTitle("Tetra3d - LightGroup Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
