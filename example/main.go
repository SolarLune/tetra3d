package main

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	_ "image/png"

	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Width, Height int
	// Models         []*tetra3d.Model
	Scene *tetra3d.Scene

	Camera *tetra3d.Camera

	Time              float64
	DrawDebugText     bool
	DrawDebugDepth    bool
	PrevMousePosition vector.Vector
}

func NewGame() *Game {

	game := &Game{
		Width:             320,
		Height:            180,
		PrevMousePosition: vector.Vector{},
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	dae, _ := tetra3d.LoadDAEFile("examples.dae", nil)

	g.Scene = dae

	for _, m := range g.Scene.Meshes {
		if strings.HasSuffix(m.MaterialName, ".png") {
			m.Image, _, _ = ebitenutil.NewImageFromFile(m.MaterialName)
		}
	}

	// fmt.Println(g.Scene.Meshes["Crates"])

	// g.Scene.Meshes

	// vvvvv Stress Test vvvvv

	// for i := 0; i < 5; i++ {
	// 	for j := 0; j < 5; j++ {

	// 		mesh := meshes["Suzanne"]
	// 		mesh.ApplyMatrix(tetra3d.Rotate(1, 0, 0, -math.Pi/2))
	// 		model := tetra3d.NewModel(mesh)
	// 		model.Position[0] = float64(i) * 2
	// 		model.Position[2] = float64(j) * 2
	// 		g.Models = append(g.Models, model)

	// 	}
	// }

	// mesh := dae.Meshes["Suzanne"]
	// mesh.ApplyMatrix(tetra3d.Rotate(1, 0, 0, -math.Pi/2))
	// model := tetra3d.NewModel(mesh, "Suzanne")
	// g.Models = append(g.Models, model)

	// mesh = dae.Meshes["Crates"]
	// mesh.ApplyMatrix(tetra3d.Rotate(1, 0, 0, -math.Pi/2))
	// mesh.Image, _, _ = ebitenutil.NewImageFromFile("outdoorstuff.png")
	// model = tetra3d.NewModel(mesh, "Suzanne")
	// model.Position[0] += 4
	// g.Models = append(g.Models, model)

	// mesh = dae.Meshes["Sphere"]
	// mesh.ApplyMatrix(tetra3d.Rotate(1, 0, 0, -math.Pi/2))
	// model = tetra3d.NewModel(mesh, "Suzanne")
	// model.Position[0] += 8
	// g.Models = append(g.Models, model)

	// mesh = dae.Meshes["Hallway"]
	// mesh.ApplyMatrix(tetra3d.Rotate(1, 0, 0, -math.Pi/2))
	// mesh.Image, _, _ = ebitenutil.NewImageFromFile("outdoorstuff.png")
	// model = tetra3d.NewModel(mesh, "Suzanne")
	// model.Position[0] += 12
	// g.Models = append(g.Models, model)

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	// g.Camera.Position[0] = 5
	g.Camera.Position[2] = 20
	// g.Camera.Far = 60
	// g.Camera.Rotation.Axis = vector.Vector{1, 0, 0}
	// g.Camera.Rotation.Angle = -math.Pi / 4
	// g.Camera.RenderDepth = false // You can turn off depth rendering if your computer doesn't do well with shaders or rendering to offscreen buffers,
	// but this will turn off inter-object depth sorting. Instead, Tetra's Camera will render objects in order of distance to camera.

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {

	var err error

	moveSpd := 0.2

	g.Time += 1.0 / 60

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Movement is global; you could use Camera.Forward(), Camera.Right(), and Camera.Up() for local axis movement.

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.Camera.Position = g.Camera.Position.Add(g.Camera.Forward().Scale(moveSpd))
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.Camera.Position = g.Camera.Position.Add(g.Camera.Right().Scale(moveSpd))
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.Camera.Position = g.Camera.Position.Add(g.Camera.Forward().Scale(-moveSpd))
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.Camera.Position = g.Camera.Position.Add(g.Camera.Right().Scale(-moveSpd))
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		g.Camera.Rotation.Angle -= 0.025
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		g.Camera.Rotation.Angle += 0.025
	}

	// mx, my := ebiten.CursorPosition()

	// mv := vector.Vector{float64(mx), float64(my)}

	// diff := mv.Sub(g.PrevMousePosition)

	// g.CameraRot.Angle -= diff[0] * 0.01

	// g.CameraTilt.Axis = g.Camera.Right()
	// g.CameraTilt.Angle -= diff[1] * 0.01

	// g.Camera.Rotation = g.CameraTilt.Add(g.CameraTilt)

	// g.PrevMousePosition = mv.Clone()

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.Camera.Position[1] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		g.Camera.Position[1] -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.Key1) {
		g.Scene.SetFog(1, 0, 0, tetra3d.FogAdd)
	} else if ebiten.IsKeyPressed(ebiten.Key2) {
		g.Scene.SetFog(0, 0, 0, tetra3d.FogMultiply)
	} else if ebiten.IsKeyPressed(ebiten.Key3) {
		g.Scene.SetFog(0, 0, 0, tetra3d.FogOff)
	}

	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		if sphere := g.Scene.FindModel("Sphere"); sphere != nil {
			g.Scene.RemoveModels(sphere)
		}
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

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.Black)

	g.Camera.Clear()

	g.Camera.Render(g.Scene)

	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture, nil)
	} else {
		screen.DrawImage(g.Camera.ColorTexture, nil)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugText(screen, 1)
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

	ebiten.SetWindowTitle("tetra3d Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
