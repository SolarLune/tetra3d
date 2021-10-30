package main

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"runtime/pprof"
	"time"

	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/solarlune/ebiten3d/ebiten3d"
)

type Game struct {
	Width, Height int
	Model         *ebiten3d.Model
	Camera        *ebiten3d.Camera
}

func NewGame() *Game {

	game := &Game{
		Width:  640,
		Height: 360,
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

	// g.Bunny = ebiten3d.NewModel(ebiten3d.LoadMeshFromOBJFile("ebiten3d/cube.obj"))
	g.Model = ebiten3d.NewModel(ebiten3d.NewCube())
	// g.Model = ebiten3d.NewModel(ebiten3d.NewPlane())
	g.Model.Mesh.Image, _, _ = ebitenutil.NewImageFromFile("ebiten3d/testimage.png")
	// g.Model = ebiten3d.NewModel(ebiten3d.NewCube())
	// g.Model.Rotation = vector.Vector{1, 0, 0, 0}
	// g.Model.Mesh.BackfaceCulling = false
	// g.Model.Scale = g.Model.Scale.Scale(0.1)

	g.Camera = ebiten3d.NewCamera(g.Width, g.Height)
	g.Camera.SetPerspectiveView(90)

	g.Camera.Position[2] = -10
	g.Camera.Position[1] = 1

}

func (g *Game) Update() error {

	var err error

	moveSpd := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.Camera.Position[0] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.Camera.Position[0] -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.Camera.Position[2] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.Camera.Position[2] -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.Camera.Position[1] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		g.Camera.Position[1] -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.Model.Rotation[3] += 0.025
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.Model.Rotation[3] -= 0.025
	}

	// g.Camera.LookAt(g.Model.Position, vector.Y)

	// fmt.Println(g.Model.Position)
	// fmt.Println(g.Camera.EyeMatrix)

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.Init()
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.White)
	// screen.Fill(color.Black)

	// g.Camera.Transform = ebiten3d.LookAt(vector.Vector{0, 0, 0}, vector.Vector{0, 5, 5}, vector.Vector{0, 1, 0})

	// g.Camera.Position[2] += 0.05
	// g.Camera.Position[0] += 0.01

	// g.Camera.Position[1] += 0.01
	// g.Bunny.Position[0] += 0.01

	// Begin updates the camera transforms and clears the backbuffer

	g.Camera.Begin()

	// for i := 0; i < 20; i++ {
	// g.Bunny.Position = vector.Vector{0, 0, 0}

	g.Camera.Render(g.Model)

	// g.Bunny.Position = vector.Vector{10, 0, 10}
	// ebiten3d.DrawModel(g.Camera, g.Bunny)

	// }

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

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
