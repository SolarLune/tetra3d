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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene

	Camera       *tetra3d.Camera
	CameraTilt   float64
	CameraRotate float64

	DrawDebugText     bool
	DrawDebugDepth    bool
	PrevMousePosition vector.Vector

	Time float64

	FireParticleSystem  *tetra3d.ParticleSystem
	FieldParticleSystem *tetra3d.ParticleSystem
	RingParticleSystem  *tetra3d.ParticleSystem
}

func NewGame() *Game {
	game := &Game{
		Width:             786,
		Height:            448,
		PrevMousePosition: vector.Vector{},
		DrawDebugText:     true,
	}

	game.Init()

	return game
}

//go:embed particles.gltf
var particlesScene []byte

func (g *Game) Init() {

	g.Scene = tetra3d.NewScene("particle test")

	g.Scene.World.LightingOn = false

	lib, err := tetra3d.LoadGLTFData(particlesScene, nil)
	if err != nil {
		panic(err)
	}

	g.Scene = lib.ExportedScene

	// OK, so the way particle systems work is that you create a particle system using one or more source particles (models).
	// If you use multiple particles, the particle system will spawn one of them randomly.
	// The particles will render using the first material of each Particle, so you can change their visual properties (blend mode,
	// color, etc) as you would any other model.

	// You then change the ParticleSystemSettings struct to make the particles spawn and move as you wish.

	partSystem := g.Scene.Root.Get("Fire").(*tetra3d.Model)
	g.FireParticleSystem = tetra3d.NewParticleSystem(partSystem, g.Scene.Root.Get("Particle").(*tetra3d.Model))

	settings := g.FireParticleSystem.Settings
	settings.Lifetime.Set(1, 1)        // Lifetime can vary randomly; we're setting both the minimum and maximum bounds here to 1.
	settings.SpawnRate = 0.025         // How often particles are spawned
	settings.SpawnCount = 2            // How many particles are spawned each time
	settings.Scale.SetRanges(0.2, 0.4) // The scale of a particle can vary randomly...
	settings.Scale.Uniform = true      // But we also want the particles to scale uniformly, so they don't appear squashed or stretched
	settings.Growth.SetAll(0.02)       // Growth is how quickly particles grow in size. If a particle reaches a scale of 0, it will die (unless you disable ParticleSystemSettings.AllowNegativeScale).
	settings.Growth.Uniform = true

	settings.SpawnOffset.SetRangeX(-0.1, 0.1) // SpawnOffset is how far the particle can spawn away from the center of the particle system
	settings.SpawnOffset.SetRangeZ(-0.1, 0.1)

	settings.Velocity.SetRangeY(0.04, 0.04) // Velocity controls a Particle's linear movement, while settings.Acceleration controls a Particle's acceleration (it gathers over time)

	// ColorCurve controls how the particle changes color as time passes. It is controlled by adding points to the curve, consisting of a color
	// and the percentage of time through the particle's life where that color appears. This is very similar to how Godot does particles.
	settings.ColorCurve.Add(colors.Red(), 0)
	settings.ColorCurve.Add(colors.Yellow(), 0.25)
	settings.ColorCurve.Add(colors.Gray(), 0.35)
	settings.ColorCurve.Add(colors.Gray().SetAlpha(0), 1.0)

	// Now, for the Field particle system.

	partSystem = g.Scene.Root.Get("Field").(*tetra3d.Model)
	g.FieldParticleSystem = tetra3d.NewParticleSystem(partSystem, g.Scene.Root.Get("Particle").(*tetra3d.Model), g.Scene.Root.Get("Particle2").(*tetra3d.Model))

	settings = g.FieldParticleSystem.Settings
	settings.SpawnRate = 0.1
	settings.SpawnCount = 4
	settings.Lifetime.Set(2, 3)
	settings.Scale.SetRanges(0.05, 0.2)
	settings.Scale.Uniform = true
	settings.SpawnOffset.SetRanges(-3, 3)
	settings.SpawnOffset.SetRangeY(0, 0)
	settings.Velocity.SetRangeY(0.02, 0.02)
	// Particles in the field system rotate - it's most noticeable on the "sparkle", cross-shaped particle. We're rotating on the Z axis here because
	// it's facing the camera (as the particles have billboarding on), so we spin on the Z axis.
	settings.Rotation.SetRangeZ(0.1, 0.1)
	settings.Friction = 0.0001 // Friction controls how the particle's movement dies down over time.

	// However, field particles move in a little bit more complex manner than the other particle systems, so we'll make use of the MovementFunction to tweak
	// how the particles move. In the below function, we move them out from the center of the particle system after they spawn.
	settings.MovementFunction = func(particle *tetra3d.Particle) {
		diff := particle.Model.WorldPosition().Sub(particle.ParticleSystem.Root.WorldPosition())
		diff[1] = 0
		vector.In(diff).Unit().Scale(0.01)
		particle.Model.MoveVec(diff)
	}

	particleColor := tetra3d.NewColor(0.25, 0.75, 1, 1)
	settings.ColorCurve.Add(particleColor.Clone().SetAlpha(0), 0)
	settings.ColorCurve.Add(particleColor, 0.1)
	settings.ColorCurve.Add(colors.White(), 0.9)
	settings.ColorCurve.Add(colors.White().SetAlpha(0), 1)

	// And now, finally, the ring system.

	partSystem = g.Scene.Root.Get("Ring").(*tetra3d.Model)
	g.RingParticleSystem = tetra3d.NewParticleSystem(partSystem, g.Scene.Root.Get("Particle").(*tetra3d.Model))
	settings = g.RingParticleSystem.Settings

	// Similarly to the field system, the ring system spawns particles in a different manner - we want them to spawn in a ring that spins.
	// To do this, we'll make use of a vector that controls how far out the particles spawn, and then rotate that ring after each spawn.

	ring := vector.Vector{4, 0, 0}

	settings.SpawnOffsetFunction = func(particle *tetra3d.Particle) {
		particle.Model.MoveVec(ring)
		// 181 degrees here because 1: we're slowly spinning the ring by 1 degree, and 2: we're spawning two rings (so we spin the ring by 180 degrees)
		ring = ring.Rotate(tetra3d.ToRadians(181), vector.Vector{0, 1, 0})
	}

	settings.SpawnOffset.SetRanges(-0.1, 0.1)
	settings.SpawnRate = 0.025
	settings.SpawnCount = 8
	settings.Scale.SetRanges(0.25, 0.5)
	settings.Scale.Uniform = true
	settings.Growth.SetRanges(0.002, 0.004) // Steadily grow just a little bit, with some randomness
	settings.Growth.Uniform = true
	settings.Velocity.SetRangeY(0.01, 0.02)
	settings.Lifetime.Set(0.5, 1)

	partColor := colors.White()
	settings.ColorCurve.Add(partColor.Clone().SetAlpha(0), 0)
	settings.ColorCurve.Add(partColor, 0.1)
	settings.ColorCurve.Add(colors.SkyBlue(), 0.5)
	settings.ColorCurve.Add(colors.Blue(), 0.8)
	settings.ColorCurve.Add(colors.Blue().MultiplyRGBA(0.25, 0.25, 0.25, 0), 1)

	// And that's about it!

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.SetLocalPositionVec(vector.Vector{0, 2, 5})
	g.Scene.Root.AddChildren(g.Camera)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {
	var err error

	moveSpd := 0.05

	g.Time += 1.0 / 60.0

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Update all of our particle systems, and we're done~
	g.FireParticleSystem.Update(1.0 / 60.0)
	g.FieldParticleSystem.Update(1.0 / 60.0)
	g.RingParticleSystem.Update(1.0 / 60.0)

	g.FireParticleSystem.Root.Move(math.Sin(g.Time)*0.05, 0, 0)

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
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
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
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nThis example shows how particle systems work.\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		g.Camera.DebugDrawText(screen, txt, 0, 120, 1, colors.LightGray())
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
	ebiten.SetWindowTitle("Tetra3d - Particles Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
