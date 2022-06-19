package tetra3d

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"
)

type ExampleGame struct {
	Width, Height int
	Scene         *tetra3d.Scene

	Camera       *tetra3d.Camera
	CameraTilt   float64
	CameraRotate float64

	PrevMousePosition vector.Vector
	CamMoveSpeed      float64
}

func NewExampleGame(width, height int) ExampleGame {
	return ExampleGame{
		Width:             width,
		Height:            height,
		PrevMousePosition: vector.Vector{},
		CamMoveSpeed:      0.05,
	}
}

func (g *ExampleGame) ProcessCameraInputs() {
	g.processCamMovements()
	g.processCamRotations()
}

func (g *ExampleGame) processCamMovements() {
	// We use Camera.Rotation.Forward().Invert() because the camera looks down -Z (so its forward vector is inverted)
	forward := g.Camera.LocalRotation().Forward().Invert()
	right := g.Camera.LocalRotation().Right()

	pos := g.Camera.LocalPosition()

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		pos = pos.Add(forward.Scale(g.CamMoveSpeed))
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		pos = pos.Add(right.Scale(g.CamMoveSpeed))
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		pos = pos.Add(forward.Scale(-g.CamMoveSpeed))
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		pos = pos.Add(right.Scale(-g.CamMoveSpeed))
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		pos[1] += g.CamMoveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		pos[1] -= g.CamMoveSpeed
	}

	g.Camera.SetLocalPosition(pos)
}

// Rotating the camera with the mouse
func (g *ExampleGame) processCamRotations() {
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
}

func (g *ExampleGame) SetupCameraAt(localPos vector.Vector) {
	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.SetLocalPosition(localPos)
	g.Scene.Root.AddChildren(g.Camera)
}
