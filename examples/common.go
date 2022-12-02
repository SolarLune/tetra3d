package examples

import (
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
)

type GameI interface {
	Init()
}

type BasicSystemHandler struct {
	DrawDebugText  bool
	DrawDebugDepth bool
	Game           GameI
}

func NewBasicSystemHandler(game GameI) BasicSystemHandler {
	return BasicSystemHandler{
		DrawDebugText: true,
		Game:          game,
	}
}

func (system *BasicSystemHandler) Update() error {

	var err error

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		system.Game.Init()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		system.DrawDebugText = !system.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		system.DrawDebugDepth = !system.DrawDebugDepth
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		StartProfiling()
	}

	return err

}

func (system *BasicSystemHandler) Draw(screen *ebiten.Image, camera *tetra3d.Camera) {

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		f, err := os.Create("screenshot" + time.Now().Format("2006-01-02 15:04:05") + ".png")
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		png.Encode(f, camera.ColorTexture())
	}

	if system.DrawDebugDepth {
		screen.DrawImage(camera.DepthTexture(), nil)
	}

	if system.DrawDebugText {
		camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		camera.DebugDrawText(screen, txt, 0, 120, 1, colors.LightGray())
	}

}

// BasicFreeCam represents a basic freely moving camera, which allows you to easily explore a scene.
type BasicFreeCam struct {
	*tetra3d.Camera
	CameraTilt        float64
	CameraRotate      float64
	PrevMousePosition tetra3d.Vector
	Locked            bool
}

// NewBasicFreeCam creates a new BasicFreeCam struct.
func NewBasicFreeCam() BasicFreeCam {

	freecam := BasicFreeCam{
		Locked: true,
	}

	freecam.Camera = tetra3d.NewCamera(480, 270)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	freecam.Node.SetLocalPosition(0, 0, 5)

	return freecam

}

// Update updates the free cam.
func (cc *BasicFreeCam) Update() {

	// Moving the Camera

	moveSpd := 0.05

	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		moveSpd *= 3
	}

	// We use Camera.Rotation.Forward().Invert() because the camera looks down -Z (so its forward vector is inverted)
	forward := cc.Node.LocalRotation().Forward().Invert()
	right := cc.Node.LocalRotation().Right()

	pos := cc.Node.LocalPosition()

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
		pos.Y += moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		pos.Y -= moveSpd
	}

	cc.Node.SetLocalPositionVec(pos)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cc.Locked = !cc.Locked
		if cc.Locked {
			ebiten.SetCursorMode(ebiten.CursorModeCaptured)
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		}
	}

	// Rotating the camera with the mouse

	// Rotate and tilt the camera according to mouse movements
	mx, my := ebiten.CursorPosition()

	mv := tetra3d.NewVector(float64(mx), float64(my), 0)

	if cc.Locked {

		diff := mv.Sub(cc.PrevMousePosition)

		cc.CameraTilt -= diff.Y * 0.005
		cc.CameraRotate -= diff.X * 0.005

		cc.CameraTilt = math.Max(math.Min(cc.CameraTilt, math.Pi/2-0.1), -math.Pi/2+0.1)

		tilt := tetra3d.NewMatrix4Rotate(1, 0, 0, cc.CameraTilt)
		rotate := tetra3d.NewMatrix4Rotate(0, 1, 0, cc.CameraRotate)

		// Order of this is important - tilt * rotate works, rotate * tilt does not, lol
		cc.Node.SetLocalRotation(tilt.Mult(rotate))

	}

	cc.PrevMousePosition = mv

}

func StartProfiling() {
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

// type DemoSceneI interface {
// 	Update() error
// 	ActiveScene() *tetra3d.Scene
// 	Draw(screen *ebiten.Image)
// 	Init()
// }

// type ExampleBase struct {
// 	Level DemoSceneI

// 	Width, Height int

// 	Camera       *tetra3d.Camera
// 	CameraTilt   float64
// 	CameraRotate float64

// 	DrawDebugText     bool
// 	DrawDebugDepth    bool
// 	PrevMousePosition tetra3d.Vector
// }

// func NewExampleBase(level DemoSceneI) *ExampleBase {
// 	base := &ExampleBase{Level: level}
// 	base.Level.Init()
// 	return base
// }

// func (base *ExampleBase) Update() error {

// 	var err error

// 	moveSpd := 0.05

// 	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
// 		err = errors.New("quit")
// 	}

// 	// Moving the Camera

// 	// We use Camera.Rotation.Forward().Invert() because the camera looks down -Z (so its forward vector is inverted)
// 	forward := base.Camera.LocalRotation().Forward().Invert()
// 	right := base.Camera.LocalRotation().Right()

// 	pos := base.Camera.LocalPosition()

// 	if ebiten.IsKeyPressed(ebiten.KeyW) {
// 		pos = pos.Add(forward.Scale(moveSpd))
// 	}

// 	if ebiten.IsKeyPressed(ebiten.KeyD) {
// 		pos = pos.Add(right.Scale(moveSpd))
// 	}

// 	if ebiten.IsKeyPressed(ebiten.KeyS) {
// 		pos = pos.Add(forward.Scale(-moveSpd))
// 	}

// 	if ebiten.IsKeyPressed(ebiten.KeyA) {
// 		pos = pos.Add(right.Scale(-moveSpd))
// 	}

// 	if ebiten.IsKeyPressed(ebiten.KeySpace) {
// 		pos.Y += moveSpd
// 	}

// 	if ebiten.IsKeyPressed(ebiten.KeyControl) {
// 		pos.Y -= moveSpd
// 	}

// 	base.Camera.SetLocalPositionVec(pos)

// 	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
// 		ebiten.SetFullscreen(!ebiten.IsFullscreen())
// 	}

// 	// Rotating the camera with the mouse

// 	// Rotate and tilt the camera according to mouse movements
// 	mx, my := ebiten.CursorPosition()

// 	mv := tetra3d.NewVector(float64(mx), float64(my), 0)

// 	diff := mv.Sub(base.PrevMousePosition)

// 	base.CameraTilt -= diff.Y * 0.005
// 	base.CameraRotate -= diff.X * 0.005

// 	base.CameraTilt = math.Max(math.Min(base.CameraTilt, math.Pi/2-0.1), -math.Pi/2+0.1)

// 	tilt := tetra3d.NewMatrix4Rotate(1, 0, 0, base.CameraTilt)
// 	rotate := tetra3d.NewMatrix4Rotate(0, 1, 0, base.CameraRotate)

// 	// Order of this is important - tilt * rotate works, rotate * tilt does not, lol
// 	base.Camera.SetLocalRotation(tilt.Mult(rotate))

// 	base.PrevMousePosition = mv

// 	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
// 		f, err := os.Create("screenshot" + time.Now().Format("2006-01-02 15:04:05") + ".png")
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		defer f.Close()
// 		png.Encode(f, base.Camera.ColorTexture())
// 	}

// 	if ebiten.IsKeyPressed(ebiten.KeyR) {
// 		base.Level.Init()
// 	}

// 	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
// 		base.StartProfiling()
// 	}

// 	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
// 		base.DrawDebugText = !base.DrawDebugText
// 	}

// 	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
// 		base.DrawDebugDepth = !base.DrawDebugDepth
// 	}

// 	if newErr := base.Level.Update(); newErr != nil {
// 		err = newErr
// 	}

// 	return err

// }

// func (g *ExampleBase) Layout(w, h int) (int, int) {
// 	return w, h
// }

// func (g *ExampleBase) StartProfiling() {
// 	outFile, err := os.Create("./cpu.pprof")
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		return
// 	}

// 	fmt.Println("Beginning CPU profiling...")
// 	pprof.StartCPUProfile(outFile)
// 	go func() {
// 		time.Sleep(2 * time.Second)
// 		pprof.StopCPUProfile()
// 		fmt.Println("CPU profiling finished.")
// 	}()
// }

// func (base *ExampleBase) Draw(screen *ebiten.Image) {

// 	// Clear, but with a color
// 	screen.Fill(base.Level.ActiveScene().World.ClearColor.ToRGBA64())

// 	// Clear the Camera
// 	base.Camera.Clear()

// 	base.Level.Draw(base.Camera, screen)

// }
