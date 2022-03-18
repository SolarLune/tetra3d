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

//go:embed shapes.gltf
var shapes []byte

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene

	Camera       tetra3d.INode
	CameraTilt   float64
	CameraRotate float64

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugWireframe bool
	DrawDebugNormals   bool
	PrevMousePosition  vector.Vector
}

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

	// Load the GLTF file and turn it into a Library, which is a collection of scenes and data shared between them (like meshes or animations).

	options := tetra3d.DefaultGLTFLoadOptions()
	options.CameraWidth = g.Width
	options.CameraHeight = g.Height

	library, err := tetra3d.LoadGLTFData(shapes, options)
	if err != nil {
		panic(err)
	}

	g.Scene = library.FindScene("Scene")

	// Cameras are loaded from the GLTF file. By default, there's a correction so that they point in the correct direction
	// when compared with Blender.
	g.Camera = g.Scene.Root.Get("Camera")

	// g.Camera.RenderDepth = false // You can turn off depth rendering if your computer doesn't do well with shaders or rendering to offscreen buffers,
	// but this will turn off inter-object depth sorting. Instead, Tetra's Camera will render objects in order of distance to camera.

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {
	var err error

	moveSpd := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	camera := g.Camera.Get("Camera_Orientation").(*tetra3d.Camera)

	forward := g.Camera.LocalRotation().Up().Invert()
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

	// Rotate and tilt the camera according to mouse movements
	mx, my := ebiten.CursorPosition()

	mv := vector.Vector{float64(mx), float64(my)}

	diff := mv.Sub(g.PrevMousePosition)

	g.CameraTilt -= diff[1] * 0.005
	g.CameraRotate -= diff[0] * 0.005

	g.CameraTilt = math.Max(math.Min(g.CameraTilt, math.Pi-0.1), -0.1)

	tilt := tetra3d.NewMatrix4Rotate(1, 0, 0, g.CameraTilt)
	rotate := tetra3d.NewMatrix4Rotate(0, 1, 0, g.CameraRotate)

	// Order of this is important - tilt * rotate works, rotate * tilt does not, lol
	g.Camera.SetLocalRotation(tilt.Mult(rotate))

	g.PrevMousePosition = mv.Clone()

	// Fog controls
	if ebiten.IsKeyPressed(ebiten.Key1) {
		g.Scene.FogColor.Set(1, 0, 0, 1)
		g.Scene.FogMode = tetra3d.FogAdd
	} else if ebiten.IsKeyPressed(ebiten.Key2) {
		g.Scene.FogColor.Set(0, 0, 0, 1)
		g.Scene.FogMode = tetra3d.FogMultiply
	} else if ebiten.IsKeyPressed(ebiten.Key3) {
		g.Scene.FogColor.Set(0, 0, 0, 1)
		g.Scene.FogMode = tetra3d.FogOverwrite
	} else if ebiten.IsKeyPressed(ebiten.Key4) {
		g.Scene.FogColor = colors.White()
		g.Scene.FogMode = tetra3d.FogOverwrite
	} else if ebiten.IsKeyPressed(ebiten.Key5) {
		g.Scene.FogMode = tetra3d.FogOff
		g.Scene.FogColor = colors.Black() // Fog being off, this doesn't do anything directly, but the clear color is set below to the fog color
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

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear, but with a color
	screen.Fill(g.Scene.FogColor.ToRGBA64())

	camera := g.Camera.Get("Camera_Orientation").(*tetra3d.Camera)

	camera.Clear()

	camera.RenderNodes(g.Scene, g.Scene.Root)

	opt := &ebiten.DrawImageOptions{}
	w, h := camera.ColorTexture.Size()

	// We rescale the depth or color textures just in case we render at a different resolution than the window's.
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(camera.DepthTexture, opt)
	} else {
		screen.DrawImage(camera.ColorTexture, opt)
	}

	if g.DrawDebugText {
		camera.DrawDebugText(screen, 1, tetra3d.NewColor(0, 0.5, 1, 1))
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\n1, 2, 3, 4: Change fog\nF1, F2, F3, F5: Debug views\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 128, color.RGBA{255, 0, 0, 255})
	}

	if g.DrawDebugWireframe {
		camera.DrawDebugWireframe(screen, g.Scene.Root, colors.Red())
	}

	if g.DrawDebugNormals {
		camera.DrawDebugNormals(screen, g.Scene.Root, 0.25, colors.Blue())
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
	ebiten.SetWindowTitle("Tetra3d - Shapes Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
