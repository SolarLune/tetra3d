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
)

type Game struct {
	Width, Height int
	Library       *tetra3d.Library

	Time              float64
	Camera            *tetra3d.Camera
	CameraTilt        float64
	CameraRotate      float64
	PrevMousePosition vector.Vector

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugNormals   bool
	DrawDebugCenters   bool
	DrawDebugWireframe bool
}

//go:embed animations.gltf
var gltf []byte

func NewGame() *Game {

	game := &Game{
		Width:             796,
		Height:            448,
		PrevMousePosition: vector.Vector{},
		DrawDebugText:     true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(gltf, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.Move(0, 0, 10)
	scene := g.Library.Scenes[0]
	// Turn off lighting
	scene.World.LightingOn = false
	scene.Root.AddChildren(g.Camera)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	// newCube := scenes.Scenes[0].Root.Get("Cube.001").Clone()
	// scenes.Scenes[0].Root.Get("Armature/Root/1/2/3/4/5").AddChildren(newCube)
	// newCube.SetLocalPosition(vector.Vector{0, 2, 0})

}

func (g *Game) Update() error {
	var err error

	moveSpd := 0.05

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
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

	g.Camera.SetLocalPosition(pos)

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

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.DrawDebugWireframe = !g.DrawDebugWireframe
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.DrawDebugNormals = !g.DrawDebugNormals
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		g.DrawDebugCenters = !g.DrawDebugCenters
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	scene := g.Library.Scenes[0]

	armature := scene.Root.ChildrenRecursive().ByName("Armature", true)[0].(*tetra3d.Node)
	armature.Rotate(0, 1, 0, 0.01)

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		// armature.AnimationPlayer.FinishMode = tetra3d.FinishModeStop
		armature.AnimationPlayer().Play(g.Library.Animations["ArmatureAction"])
	}

	armature.AnimationPlayer().Update(1.0 / 60)

	table := scene.Root.Get("Table").(*tetra3d.Model)
	table.AnimationPlayer().BlendTime = 0.1
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		table.AnimationPlayer().Play(g.Library.Animations["SmoothRoll"])
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		table.AnimationPlayer().Play(g.Library.Animations["StepRoll"])
	}

	table.AnimationPlayer().Update(1.0 / 60)

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
	scene := g.Library.Scenes[0]
	g.Camera.RenderNodes(scene, scene.Root)

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

	if g.DrawDebugWireframe {
		g.Camera.DrawDebugWireframe(screen, scene.Root, colors.Red())
	}

	if g.DrawDebugNormals {
		g.Camera.DrawDebugNormals(screen, scene.Root, 0.5, colors.Blue())
	}

	if g.DrawDebugCenters {
		g.Camera.DrawDebugCenters(screen, scene.Root, colors.SkyBlue())
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\n1 Key: Play [SmoothRoll] Animation On Table\n2 Key: Play [StepRoll] Animation on Table\nNote the animations can blend\nF Key: Play Animation on Skinned Mesh\nNote that the nodes move as well\nF4: Toggle fullscreen\nF6: Node Debug View\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 140, color.RGBA{255, 0, 0, 255})
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
	ebiten.SetWindowTitle("Tetra3d - Animations Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
