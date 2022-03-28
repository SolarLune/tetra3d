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

//go:embed bounds.gltf
var gltfData []byte

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene

	Controlling *tetra3d.Model

	Camera       *tetra3d.Camera
	CameraTilt   float64
	CameraRotate float64

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugBounds    bool
	DrawDebugWireframe bool
	DrawDebugNormals   bool
	PrevMousePosition  vector.Vector
}

func NewGame() *Game {
	game := &Game{
		Width:             1920,
		Height:            1080,
		PrevMousePosition: vector.Vector{},
		DrawDebugText:     true,
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(gltfData, nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.FindScene("Scene")

	g.Controlling = g.Scene.Root.Get("YellowCapsule").(*tetra3d.Model)

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.SetLocalPosition(vector.Vector{0, 6, 15})
	g.Camera.Far = 40

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	// g.Scene.Root.Get("Ground").SetVisible(false, false)

}

func (g *Game) Update() error {

	var err error

	moveSpd := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

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

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.Controlling.Move(moveSpd, 0, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.Controlling.Move(-moveSpd, 0, 0)
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.Controlling.Move(0, 0, -moveSpd)
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.Controlling.Move(0, 0, moveSpd)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		sphere := g.Scene.Root.Get("Sphere").(*tetra3d.Model)
		capsule := g.Scene.Root.Get("YellowCapsule").(*tetra3d.Model)
		if g.Controlling == sphere {
			g.Controlling = capsule
		} else {
			g.Controlling = sphere
		}
	}

	// Gravity
	g.Controlling.Move(0, -0.1, 0)

	bounds := g.Controlling.Children()[0].(tetra3d.BoundingObject)

	for _, o := range g.Scene.Root.ChildrenRecursive().ByType(tetra3d.NodeTypeBounding) {

		other := o.(tetra3d.BoundingObject)

		if inter := bounds.Intersection(other); inter != nil {

			maxMTV := vector.Vector{0, 0, 0}
			for _, i := range inter.Intersections {
				if i.MTV.Magnitude() > maxMTV.Magnitude() {
					maxMTV = i.MTV
				}
			}

			g.Controlling.MoveVec(maxMTV)

		}

	}

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

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugBounds = !g.DrawDebugBounds
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear, but with a color
	// screen.Fill(color.RGBA{20, 25, 30, 255})
	screen.Fill(color.Black)

	g.Camera.Clear()

	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	opt := &ebiten.DrawImageOptions{}
	w, h := g.Camera.ColorTexture().Size()

	// We rescale the depth or color textures just in case we render at a different resolution than the window's.
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), opt)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), opt)
	}

	if g.DrawDebugWireframe {
		g.Camera.DrawDebugWireframe(screen, g.Scene.Root, colors.White())
	}

	if g.DrawDebugNormals {
		g.Camera.DrawDebugNormals(screen, g.Scene.Root, 0.25, colors.SkyBlue())
	}

	if g.DrawDebugBounds {
		g.Camera.DrawDebugBoundsColored(screen, g.Scene.Root, colors.White(), colors.White(), colors.White(), colors.Gray())
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugText(screen, 1, colors.White())
		shapeName := g.Controlling.Name()
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nArrow keys: Move " + shapeName + "\nF: switch between capsule and sphere\nF1, F2, F3: Debug views\nF5: Display bounds shapes\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 128, color.RGBA{192, 192, 192, 255})
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
