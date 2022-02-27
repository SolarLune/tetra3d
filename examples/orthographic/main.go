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

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

//go:embed orthographic.gltf
var sceneData []byte

type Game struct {
	Width, Height int

	Scene     *tetra3d.Scene
	CamHandle tetra3d.INode

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugWireframe bool
	DrawDebugNormals   bool
	DrawDebugCenters   bool
}

func NewGame() *Game {

	game := &Game{
		Width:         398,
		Height:        224,
		DrawDebugText: true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(sceneData, nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.Scenes[0]

	camera := tetra3d.NewCamera(g.Width, g.Height)
	camera.SetOrthographic(40)
	camera.Far = 60

	g.CamHandle = g.Scene.Root.Get("CameraHandle")

	placeholder := g.CamHandle.Get("CameraPlaceholder")

	g.CamHandle.AddChildren(camera)
	camera.SetWorldPosition(placeholder.WorldPosition())
	camera.SetWorldRotation(placeholder.WorldRotation().BlenderToTetra()) // We have to flip around the axes to have the camera match the rotation of the scene's Camera Node

	placeholder.Unparent() // We don't need the placeholder to be there

}

func (g *Game) Update() error {

	var err error

	moveSpd := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	camera := g.CamHandle.Get("Camera").(*tetra3d.Camera)

	// Moving the Camera

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.CamHandle.Move(0, 0, -moveSpd)
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.CamHandle.Move(moveSpd, 0, 0)
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.CamHandle.Move(0, 0, moveSpd)
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.CamHandle.Move(-moveSpd, 0, 0)
	}

	// Note that adjusting the ortho scale makes it so you can see more or less, so the far clipping distance should also
	// be adjusted to compensate. I'll forego that for now for simplicity.

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		camera.OrthoScale += 0.5
	} else if ebiten.IsKeyPressed(ebiten.KeyW) {
		camera.OrthoScale -= 0.5
	}

	camera.OrthoScale = math.Max(math.Min(camera.OrthoScale, 80), 10)

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		g.CamHandle.Rotate(0, 1, 0, -0.025)
	} else if ebiten.IsKeyPressed(ebiten.KeyQ) {
		g.CamHandle.Rotate(0, 1, 0, 0.025)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		f, err := os.Create("screenshot" + time.Now().Format("2006-01-02 15:04:05") + ".png")
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		png.Encode(f, camera.ColorTexture)
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

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		g.DrawDebugCenters = !g.DrawDebugCenters
	}

	return err

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

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	camera := g.CamHandle.Get("Camera").(*tetra3d.Camera)

	// Clear the Camera
	camera.Clear()

	camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	opt := &ebiten.DrawImageOptions{}
	w, h := camera.ColorTexture.Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(camera.DepthTexture, opt)
	} else {
		screen.DrawImage(camera.ColorTexture, opt)
	}

	if g.DrawDebugText {
		camera.DrawDebugText(screen, 1, colors.White())
		txt := "F1 to toggle this text\nArrow Keys: Pan in cardinal directions\nW, S: Zoom in and Out\nQ, E: Rotate View\nR: Restart\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 128, color.RGBA{255, 0, 0, 255})
	}

	if g.DrawDebugWireframe {
		camera.DrawDebugWireframe(screen, g.Scene.Root, colors.White())
	}

	if g.DrawDebugNormals {
		camera.DrawDebugNormals(screen, g.Scene.Root, 0.5, colors.Blue())
	}

	if g.DrawDebugCenters {
		camera.DrawDebugCenters(screen, g.Scene.Root, colors.SkyBlue())
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {

	ebiten.SetWindowTitle("Tetra3d Test - Orthographic Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
