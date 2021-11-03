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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/solarlune/ebiten3d/ebiten3d"
)

type Game struct {
	Width, Height int
	Models        []*ebiten3d.Model
	Camera        *ebiten3d.Camera
	Time          float64
}

func NewGame() *Game {

	game := &Game{
		Width:  320,
		Height: 180,
	}

	game.Init()

	go func() {

		for {
			fmt.Println(ebiten.CurrentFPS())
			time.Sleep(time.Second)
		}

	}()

	return game
}

func (g *Game) Init() {

	g.Models = []*ebiten3d.Model{}

	// g.Models = append(g.Models, g.ConstructGrid(40, 2)...)

	// x := ebiten3d.NewModel(ebiten3d.NewCube())
	// x.Position[0] = 10
	// x.Scale = vector.Vector{0.25, 0.25, 0.25}
	// for _, t := range x.Mesh.Triangles {
	// 	for _, v := range t.Vertices {
	// 		v.Color = vector.Vector{1, 0, 0, 1}
	// 	}
	// }
	// g.Models = append(g.Models, x)

	// y := ebiten3d.NewModel(ebiten3d.NewCube())
	// y.Position[1] = 10
	// y.Scale = vector.Vector{0.25, 0.25, 0.25}
	// for _, t := range y.Mesh.Triangles {
	// 	for _, v := range t.Vertices {
	// 		v.Color = vector.Vector{0, 1, 0, 1}
	// 	}
	// }
	// g.Models = append(g.Models, y)

	// z := ebiten3d.NewModel(ebiten3d.NewCube())
	// z.Position[2] = 10
	// z.Scale = vector.Vector{0.25, 0.25, 0.25}
	// for _, t := range z.Mesh.Triangles {
	// 	for _, v := range t.Vertices {
	// 		v.Color = vector.Vector{0, 0, 1, 1}
	// 	}
	// }
	// g.Models = append(g.Models, z)

	// m := ebiten3d.NewModel(ebiten3d.NewCube())
	// m.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("ebiten3d/testimage.png")
	// g.Models = append(g.Models, m)

	for i := 0; i < 60; i++ {
		for j := 0; j < 20; j++ {
			model := ebiten3d.NewModel(ebiten3d.NewCube())
			model.Position[0] = float64(i) * 6
			model.Position[2] = float64(j) * 6
			model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("ebiten3d/testimage.png")
			g.Models = append(g.Models, model)
		}
	}

	// meshes, _ := ebiten3d.LoadMeshFromDAEFile("ebiten3d/examples.dae")
	// testCube := ebiten3d.NewModel(meshes["Sphere"])
	// g.Models = append(g.Models, testCube)
	// testCube.Mesh.BackfaceCulling = false
	// testCube.Rotation.Axis = vector.Vector{1, 0, 0}
	// testCube.Rotation.Angle = math.Pi / 2

	// model := ebiten3d.NewModel(ebiten3d.NewCube())
	// model.Position[0] = -12
	// model.Position[1] = 4
	// model.Position[2] = 140
	// model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("ebiten3d/testimage.png")
	// g.Models = append(g.Models, model)
	// g.Models = append(g.Models, model)

	// model := ebiten3d.NewModel(ebiten3d.NewCube())
	// model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("ebiten3d/testimage.png")
	// g.Models = append(g.Models, model)

	// model = ebiten3d.NewModel(ebiten3d.NewPlane())
	// model.Position[2] = -5
	// model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("ebiten3d/testimage.png")
	// g.Models = append(g.Models, model)

	g.Camera = ebiten3d.NewCamera(g.Width, g.Height)
	g.Camera.Position[0] = 0
	g.Camera.Position[1] = 0
	g.Camera.Position[2] = 12

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

	for _, m := range g.Models {
		m.Position[1] = math.Sin((((m.Position[0] + m.Position[2]) * 0.01) + g.Time) * math.Pi)
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.Init()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.StartProfiling()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.Camera.Wireframe = !g.Camera.Wireframe
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{20, 30, 40, 255})

	g.Camera.Begin()

	g.Camera.Render(g.Models...)

	screen.DrawImage(g.Camera.ColorTexture, nil)

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

	ebiten.SetWindowTitle("ebiten3D Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	game.StartProfiling()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
