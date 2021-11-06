package main

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"os"
	"runtime/pprof"
	"time"

	_ "image/png"

	"github.com/kvartborg/vector"
	"github.com/solarlune/jank3d"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Width, Height int
	Models        []*jank3d.Model
	Camera        *jank3d.Camera
	Time          float64
	DrawDebugText bool
}

func NewGame() *Game {

	game := &Game{
		Width:  1920,
		Height: 1080,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	g.Models = []*jank3d.Model{}

	meshes, _ := jank3d.LoadMeshesFromDAEFile("examples.dae")

	mesh := meshes["Suzanne"]
	mesh.ApplyMatrix(jank3d.Rotate(1, 0, 0, -math.Pi/2))
	model := jank3d.NewModel(mesh)
	// model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("outdoorstuff.png")
	g.Models = append(g.Models, model)

	// mesh = meshes["Suzanne"]
	// mesh.ApplyMatrix(jank3d.Rotate(1, 0, 0, -math.Pi/2))
	// model = jank3d.NewModel(mesh)
	// model.Position[0] += 4
	// g.Models = append(g.Models, model)

	// mesh = meshes["Sphere"]
	// mesh.ApplyMatrix(jank3d.Rotate(1, 0, 0, -math.Pi/2))
	// model = jank3d.NewModel(mesh)
	// model.Position[0] += 8
	// g.Models = append(g.Models, model)

	g.Camera = jank3d.NewCamera(g.Width, g.Height)
	g.Camera.Position[1] = 5
	g.Camera.Position[2] = 5
	g.Camera.Rotation.Axis = vector.Vector{1, 0, 0}
	g.Camera.Rotation.Angle = -math.Pi / 4

}

func (g *Game) Update() error {

	var err error

	moveSpd := 0.2

	g.Time += 1.0 / 60

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Movement is global; you could use Camera.Forward(), Camera.Right(), and Camera.Up() for local axis movement.

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.Camera.Position[0] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.Camera.Position[0] -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.Camera.Position[2] -= moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.Camera.Position[2] += moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.Camera.Position[1] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		g.Camera.Position[1] -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		g.Camera.Rotation.Angle -= 0.01
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		g.Camera.Rotation.Angle += 0.01
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		for _, m := range g.Models {
			m.Rotation.Angle += 0.025
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		for _, m := range g.Models {
			m.Rotation.Angle -= 0.025
		}
	}

	// for _, m := range g.Models {
	// 	m.Position[1] = math.Sin((((m.Position[0] + m.Position[2]) * 0.01) + g.Time) * math.Pi)
	// }

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

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		g.Camera.DebugDrawBoundingSphere = !g.Camera.DebugDrawBoundingSphere
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{20, 30, 40, 255})

	g.Camera.Clear()

	g.Camera.Render(g.Models...)

	screen.DrawImage(g.Camera.ColorTexture, nil)

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

	ebiten.SetWindowTitle("jank3d Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
