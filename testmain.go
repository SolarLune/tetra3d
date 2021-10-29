package main

// import (
// 	"fmt"
// 	"image/color"
// 	"os"
// 	"runtime/pprof"
// 	"time"

// 	"github.com/hajimehoshi/ebiten/v2"
// 	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
// 	"github.com/solarlune/ebiten3d/ebiten3d"

// 	_ "image/png"
// )

// type Game struct {
// 	Bunny   *ebiten3d.Model
// 	TestImg *ebiten.Image
// }

// func NewGame() *Game {

// 	go func() {

// 		for {
// 			fmt.Println(ebiten.CurrentFPS())
// 			time.Sleep(time.Second)
// 		}

// 	}()

// 	game := &Game{
// 		Bunny: ebiten3d.NewModel(ebiten3d.LoadMeshFromOBJFile("ebiten3d/bunny.obj")),
// 	}

// 	testImg, _, err := ebitenutil.NewImageFromFile("ebiten3d/testimage.png")

// 	if err != nil {
// 		panic(err)
// 	}

// 	game.TestImg = testImg

// 	game.StartProfiling()

// 	return game
// }

// func (g *Game) Update() error {

// 	return nil
// }

// func (g *Game) Draw(screen *ebiten.Image) {

// 	// screen.Clear()

// 	screen.Fill(color.White)

// 	// ebiten3d.ReloadRendererShader()

// 	// screen.DrawImage(g.TestImg, &ebiten.DrawImageOptions{})

// 	screen.DrawTriangles(
// 		[]ebiten.Vertex{
// 			{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
// 			{DstX: 150, DstY: 100, SrcX: 16, SrcY: 16, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
// 			{DstX: 0, DstY: 100, SrcX: 0, SrcY: 16, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
// 		},
// 		[]uint16{0, 1, 2},
// 		g.TestImg,
// 		&ebiten.DrawTrianglesOptions{},
// 	)

// 	// fmt.Println("1 : ", ebiten3d.NewMatrix4())
// 	// fmt.Println("2 : ", ebiten3d.NewMatrix4().Dot(ebiten3d.Scale(1, 1, 1)))

// 	// camera := ebiten3d.NewCamera(screen)

// 	// camera.UpdateTransform()

// 	// // camera.Draw()

// 	// // ebiten3d.SetModel(g.Bunny)

// 	// // ebiten3d.DrawModel(camera)

// 	// for i := 0; i < 20; i++ {
// 	// 	// ebiten3d.DrawModel(camera, ebiten3d.NewCube())
// 	// 	ebiten3d.DrawModel(camera, g.Bunny)
// 	// }

// 	// ebiten3d.DrawModel(camera, g.Bunny)

// 	// triangle := []vector.Vector{
// 	// 	{0, 0, 0},
// 	// 	{1, -1, 0},
// 	// 	{0, -1, 0},
// 	// }

// 	// ebiten3d.DrawTriangle3D(camera, triangle)

// }

// func (g *Game) StartProfiling() {

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

// func (g *Game) Layout(w, h int) (int, int) {
// 	return 1024, 768
// }

// func main() {

// 	ebiten.SetWindowTitle("ebiten3D Test")
// 	// ebiten.SetWindowSize(640, 480)
// 	ebiten.SetWindowResizable(true)
// 	// ebiten.SetMaxTPS(1)

// 	game := NewGame()

// 	if err := ebiten.RunGame(game); err != nil {
// 		panic(err)
// 	}

// }
