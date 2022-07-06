package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"time"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// The easiest way to do sprites currently is to just make a 2D plane and consistently orient it to face the camera.

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene
	Camera        *tetra3d.Camera

	DrawDebugText  bool
	DrawDebugDepth bool
}

func NewGame() *Game {
	game := &Game{
		Width:         796,
		Height:        448,
		DrawDebugText: true,
	}

	game.Init()

	return game
}

//go:embed testimage.png
var testImageData []byte

func loadImage(data []byte) *ebiten.Image {
	reader := bytes.NewReader(data)
	img, _, err := ebitenutil.NewImageFromReader(reader)
	if err != nil {
		panic(err)
	}
	return img
}

func (g *Game) Init() {

	g.Scene = tetra3d.NewScene("cube example")

	g.Scene.LightingOn = false

	cube := tetra3d.NewModel(tetra3d.NewCube(), "Cube")
	cube.Color.Set(0, 0.5, 1, 1)
	g.Scene.Root.AddChildren(cube)

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.Move(0, 0, 5)
	g.Scene.Root.AddChildren(g.Camera)

}

func (g *Game) Update() error {
	var err error

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Moving the Camera

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// Spinning the cube
	cube := g.Scene.Root.Get("Cube")
	cube.SetLocalRotation(cube.LocalRotation().Rotated(0, 1, 0, 0.05))

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

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.DrawDebugText = !g.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear the screen with a color.
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera.
	g.Camera.Clear()

	// Render the scene.
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
		txt := "F1 to toggle this text\nThis is a very simple example showing\na simple 3D cube,\ncreated through code.\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 130, color.RGBA{200, 200, 200, 255})
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Simple Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
