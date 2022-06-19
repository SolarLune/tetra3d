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
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	examples "github.com/solarlune/tetra3d/examples"
)

type Game struct {
	examples.ExampleGame

	DrawDebugText  bool
	DrawDebugDepth bool
	Time           float64
}

func NewGame() *Game {

	game := &Game{
		ExampleGame:   examples.NewExampleGame(796, 448),
		DrawDebugText: true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	g.Scene = tetra3d.NewScene("shader test scene")

	mesh := tetra3d.NewCube()

	_, err := mesh.MeshParts[0].Material.SetShader([]byte(`
	package main

	func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
		scrSize := imageDstTextureSize()
		return vec4(position.x / scrSize.x, position.y / scrSize.y, 0, 1)
	}`))

	if err != nil {
		panic(err)
	}

	model := tetra3d.NewModel(mesh, "fragmentcube")
	model.Move(-2, 0, 0)
	g.Scene.Root.AddChildren(model)

	vertCube := tetra3d.NewCube()
	mat := vertCube.MeshParts[0].Material
	mat.Shadeless = true
	mat.VertexProgram = func(v vector.Vector) vector.Vector {
		waveHeight := 0.1
		v[1] += math.Sin(g.Time*math.Pi+v[0])*waveHeight + (waveHeight / 2)
		return v
	}
	model = tetra3d.NewModel(vertCube, "vertexcube")
	model.Move(2, 0, 0)
	g.Scene.Root.AddChildren(model)

	g.SetupCameraAt(vector.Vector{0, 2, 5})

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {
	var err error

	g.Time += 1.0 / 60.0

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	g.ProcessCameraInputs()

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

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
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	opt := &ebiten.DrawImageOptions{}
	w, h := g.Camera.ColorTexture().Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), opt)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), opt)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugText(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nThis demo shows how custom shaders work.\nThere are two kinds: fragment shaders and vertex programs.\nFragment shaders are written in Kage, while\nvertex programs are written in pure Go and are\ndone on the CPU.\nThe cube on the left is running a fragment shader,\nwhile the cube on the right runs a vertex program.\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 120, color.RGBA{255, 0, 0, 255})
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
	ebiten.SetWindowTitle("Tetra3d - Logo Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
