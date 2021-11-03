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
	"github.com/solarlune/jank"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const ScreenWidth = 320

type Game struct {
	Width, Height int
	Models        []*jank.Model
	Camera        *jank.Camera
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

	g.Models = []*jank.Model{}

	// g.Models = append(g.Models, g.ConstructGrid(40, 2)...)

	// x := jank.NewModel(jank.NewCube())
	// x.Position[0] = 10
	// x.Scale = vector.Vector{0.25, 0.25, 0.25}
	// for _, t := range x.Mesh.Triangles {
	// 	for _, v := range t.Vertices {
	// 		v.Color = vector.Vector{1, 0, 0, 1}
	// 	}
	// }
	// g.Models = append(g.Models, x)

	// y := jank.NewModel(jank.NewCube())
	// y.Position[1] = 10
	// y.Scale = vector.Vector{0.25, 0.25, 0.25}
	// for _, t := range y.Mesh.Triangles {
	// 	for _, v := range t.Vertices {
	// 		v.Color = vector.Vector{0, 1, 0, 1}
	// 	}
	// }
	// g.Models = append(g.Models, y)

	// z := jank.NewModel(jank.NewCube())
	// z.Position[2] = 10
	// z.Scale = vector.Vector{0.25, 0.25, 0.25}
	// for _, t := range z.Mesh.Triangles {
	// 	for _, v := range t.Vertices {
	// 		v.Color = vector.Vector{0, 0, 1, 1}
	// 	}
	// }
	// g.Models = append(g.Models, z)

	// m := jank.NewModel(jank.NewCube())
	// m.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("jank/testimage.png")
	// g.Models = append(g.Models, m)

	// for i := 0; i < 60; i++ {
	// 	for j := 0; j < 20; j++ {
	// 		model := jank.NewModel(jank.NewCube())
	// 		model.Position[0] = float64(i) * 6
	// 		model.Position[2] = float64(j) * 6
	// 		model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("testimage.png")
	// 		g.Models = append(g.Models, model)
	// 	}
	// }

	meshes, _ := jank.LoadMeshesFromDAEFile("examples.dae")
	testCube := jank.NewModel(meshes["Crates"])
	testCube.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("outdoorstuff.png")
	g.Models = append(g.Models, testCube)
	testCube.Mesh.BackfaceCulling = false
	testCube.Rotation.Axis = vector.Vector{1, 0, 0}
	testCube.Rotation.Angle = math.Pi / 2

	// model := jank.NewModel(jank.NewCube())
	// model.Position[0] = -12
	// model.Position[1] = 4
	// model.Position[2] = 140
	// model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("jank/testimage.png")
	// g.Models = append(g.Models, model)
	// g.Models = append(g.Models, model)

	// model := jank.NewModel(jank.NewCube())
	// model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("jank/testimage.png")
	// g.Models = append(g.Models, model)

	// model = jank.NewModel(jank.NewPlane())
	// model.Position[2] = -5
	// model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("jank/testimage.png")
	// g.Models = append(g.Models, model)

	g.Camera = jank.NewCamera(g.Width, g.Height)
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

	g.Camera.Clear()

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
	return ScreenWidth, g.Height
}

func main() {

	ebiten.SetWindowTitle("jank Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
