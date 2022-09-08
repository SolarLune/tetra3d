package main

import (
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"math/rand"
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
)

// Shared cube mesh
var cubeMesh = tetra3d.NewCube()

type CubeElement struct {
	Model           *tetra3d.Model
	Root            *tetra3d.Node
	Current, Target *tetra3d.GridPoint
	Path            []*tetra3d.GridPoint
}

func newCubeElement(root tetra3d.INode) *CubeElement {
	element := &CubeElement{
		Model:   tetra3d.NewModel(cubeMesh, "Cube Element"),
		Root:    root.(*tetra3d.Node),
		Current: root.Get("Network").Children()[0].(*tetra3d.GridPoint),
	}
	element.Model.Color.Set(1, rand.Float32()*0.1, rand.Float32()*0.1, 1)
	element.Model.SetWorldScale(0.1, 0.1, 0.1)
	element.ChooseNewTarget()
	root.AddChildren(element.Model)
	return element
}

func (cube *CubeElement) Update() {
	pos := cube.Model.WorldPosition()

	if len(cube.Path) > 0 {

		target := cube.Path[0]

		if pos.Sub(target.WorldPosition()).Magnitude() < 0.1 {
			cube.Path = cube.Path[1:]
			if len(cube.Path) == 0 {
				cube.Current = cube.Target
				cube.ChooseNewTarget()
			}
		} else {
			diff := target.WorldPosition().Sub(cube.Model.WorldPosition()).Unit()
			cube.Model.MoveVec(diff.Scale(0.025))
		}

	} else {
		cube.ChooseNewTarget()
	}

}

func (cube *CubeElement) ChooseNewTarget() {
	gridPoints := cube.Root.Get("Network").Children().GridPoints()
	cube.Target = cube.Root.Get("Network").Children().GridPoints()[rand.Intn(len(gridPoints))]
	cube.Path = cube.Target.PathTo(cube.Current)
}

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene

	Camera       *tetra3d.Camera
	CameraTilt   float64
	CameraRotate float64

	Cubes []*CubeElement

	DrawDebugText     bool
	DrawDebugDepth    bool
	PrevMousePosition vector.Vector
}

//go:embed grids.gltf
var grids []byte

func NewGame() *Game {
	game := &Game{
		Width:             398,
		Height:            224,
		PrevMousePosition: vector.Vector{},
		DrawDebugText:     true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {
	data, err := tetra3d.LoadGLTFData(grids, nil)
	if err != nil {
		panic(err)
	}

	g.Scene = data.Scenes[0]

	g.Scene.World.LightingOn = false

	for i := 0; i < 40; i++ {
		newCube := newCubeElement(g.Scene.Root)
		g.Cubes = append(g.Cubes, newCube)
	}

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.SetLocalPositionVec(vector.Vector{0, 0, 5})
	g.Scene.Root.AddChildren(g.Camera)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {
	var err error

	moveSpd := 0.05

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	for _, cube := range g.Cubes {
		cube.Update()
	}

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
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nThe screen object shows what the\ncamera is looking at.\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 120, color.RGBA{200, 200, 200, 255})
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
	ebiten.SetWindowTitle("Tetra3d - Webs Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
