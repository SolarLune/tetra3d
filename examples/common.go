package examples

import (
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
	"github.com/solarlune/tetra3d/math32"
)

type GameI interface {
	Init()
}

type BasicSystemHandler struct {
	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugWireframe bool
	DrawDebugCenters   bool

	Game              GameI
	UsingBasicFreeCam bool
}

func NewBasicSystemHandler(game GameI) BasicSystemHandler {
	return BasicSystemHandler{
		DrawDebugText:     true,
		Game:              game,
		UsingBasicFreeCam: true,
	}
}

func (system *BasicSystemHandler) Update() error {

	var err error

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = ebiten.Termination
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		system.Game.Init()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		system.DrawDebugText = !system.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		system.DrawDebugDepth = !system.DrawDebugDepth
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		system.DrawDebugWireframe = !system.DrawDebugWireframe
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		system.DrawDebugCenters = !system.DrawDebugCenters
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

	if system.DrawDebugWireframe {
		camera.DrawDebugWireframe(screen, camera.Scene().Root, colors.White())
	}

	if system.DrawDebugCenters {
		camera.DrawDebugCenters(screen, camera.Scene().Root, colors.SkyBlue())
	}

	if system.DrawDebugText {
		camera.DrawDebugRenderInfo(screen, 1, colors.White())
		var txt string

		pcOn := "On"
		if !camera.PerspectiveCorrectedTextureMapping {
			pcOn = "Off"
		}

		if system.UsingBasicFreeCam {
			txt = fmt.Sprintf(`WASD: Move, Mouse: Look, Shift: move fast
Right Click to Lock / Unlock Mouse Cursor
F1: Toggle help text - F2: Toggle depth debug,
F3: Wireframe debug - F4: fullscreen - F5: node center debug
F6: Toggle Perpsective Correction: %s
ESC: Quit
`, pcOn)
		} else {
			txt = fmt.Sprintf(`F1: Toggle help text - F2: Toggle depth debug,
F3: Wireframe debug - F4: fullscreen - F5: node center debug
F6: Toggle Perpsective Correction: %s
ESC: Quit
`, pcOn)
		}
		camera.DrawDebugText(screen, txt, 0, 130, 1, colors.LightGray())
	}

}

// BasicFreeCam represents a basic freely moving camera, which allows you to easily explore a scene.
type BasicFreeCam struct {
	*tetra3d.Camera
	Scene             *tetra3d.Scene
	CameraTilt        float32
	CameraRotate      float32
	CameraTiltSpeed   float32
	CameraRotateSpeed float32
	PrevMousePosition tetra3d.Vector3
	Locked            bool
}

// NewBasicFreeCam creates a new BasicFreeCam struct.
func NewBasicFreeCam(scene *tetra3d.Scene) BasicFreeCam {

	freecam := BasicFreeCam{
		Scene: scene,
	}

	freecam.Camera = tetra3d.NewCamera(640, 360)
	freecam.Camera.SetFieldOfView(60)
	freecam.Camera.SetLocalPosition(0, 0, 10)

	scene.Root.AddChildren(freecam.Camera)

	return freecam

}

// Update updates the free cam.
func (cc *BasicFreeCam) Update() {

	// Moving the Camera

	moveSpd := float32(0.075)

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

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		cc.Locked = !cc.Locked
		if cc.Locked {
			ebiten.SetCursorMode(ebiten.CursorModeCaptured)
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		cc.PerspectiveCorrectedTextureMapping = !cc.PerspectiveCorrectedTextureMapping
	}

	// Rotating the camera with the mouse

	// Rotate and tilt the camera according to mouse movements
	mx, my := ebiten.CursorPosition()

	mv := tetra3d.NewVector3(float32(mx), float32(my), 0)

	if cc.Locked {

		// Divide by camera size to get hopefully screen size-independent movements;
		// Note that using ebiten.WindowSize() may not work for this because it returns 0,0 on unsupported platforms (WASM, for example).
		// All of the examples use the camera's size for the layout / "screen" size, so its' fine to use it as well here too.
		w, h := cc.Camera.Size()

		diff := mv.Sub(cc.PrevMousePosition)
		diff.X /= float32(w)
		diff.Y /= float32(h)

		accel := float32(3)
		friction := float32(0.6)

		cc.CameraTiltSpeed *= friction
		cc.CameraRotateSpeed *= friction

		cc.CameraTiltSpeed -= diff.Y * accel
		cc.CameraRotateSpeed -= diff.X * accel

		cc.CameraTilt += cc.CameraTiltSpeed
		cc.CameraRotate += cc.CameraRotateSpeed

		cc.CameraTilt = math32.Clamp(cc.CameraTilt, -math.Pi/2+0.1, math.Pi/2-0.1)

		// Order of this is important - tilt * rotate works, rotate * tilt does not, lol
		rotate := tetra3d.NewMatrix4Rotate(0, 1, 0, cc.CameraRotate).Rotated(1, 0, 0, cc.CameraTilt)

		cc.Node.SetLocalRotation(rotate)

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
		time.Sleep(5 * time.Second)
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
// 	CameraTilt   float32
// 	CameraRotate float32

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

// 	mv := tetra3d.NewVector(float32(mx), float32(my), 0)

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
