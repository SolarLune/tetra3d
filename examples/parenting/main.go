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

	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	examples "github.com/solarlune/tetra3d/examples"
)

type Game struct {
	examples.ExampleGame

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugWireframe bool
	DrawDebugCenters   bool
}

func NewGame() *Game {

	game := &Game{
		ExampleGame:   examples.NewExampleGame(398, 224),
		DrawDebugText: true,
	}
	game.CamMoveSpeed = 0.1

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

	// In this example, we'll construct the scene ourselves by hand.
	g.Scene = tetra3d.NewScene("Test Scene")

	cubeMesh := tetra3d.NewCube()
	mat := cubeMesh.MeshParts[0].Material
	mat.Shadeless = true
	mat.Texture = loadImage(testImageData)

	parent := tetra3d.NewModel(cubeMesh, "parent")
	parent.SetLocalPosition(vector.Vector{0, -3, 0})

	child := tetra3d.NewModel(cubeMesh, "child")
	child.SetLocalPosition(vector.Vector{10, 2, 0})

	g.Scene.Root.AddChildren(parent, child)

	g.SetupCameraAt(vector.Vector{0, 0, 15})

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {

	var err error

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	parent := g.Scene.Root.Get("parent")
	parent.SetLocalRotation(parent.LocalRotation().Rotated(0, 1, 0, 0.05))

	child := g.Scene.Root.ChildrenRecursive().ByName("child", true)[0]

	g.ProcessCameraInputs()

	position := parent.LocalPosition()
	nodeSpeed := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		position[0] -= nodeSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		position[0] += nodeSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		position[2] -= nodeSpeed
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		position[2] += nodeSpeed
	}

	parent.SetLocalPosition(position)

	if ebiten.IsKeyPressed(ebiten.KeyG) {
		child.SetWorldPosition(vector.Vector{10, 2, 0})
		child.SetWorldRotation(tetra3d.NewMatrix4Rotate(0, 1, 0, 0))
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		parent.SetVisible(!parent.Visible(), true)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {

		if child.Parent() == parent {
			transform := child.Transform()
			// After changing parenting, the child's position would change because of a different parent (so now
			// 0, 0, 0 corresponds to a different location). We counteract this by manually setting the world transform to be the original transform.
			g.Scene.Root.AddChildren(child)
			child.SetWorldTransform(transform)
		} else {
			transform := child.Transform()
			parent.AddChildren(child)
			child.SetWorldTransform(transform)
		}

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

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.DrawDebugText = !g.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.DrawDebugWireframe = !g.DrawDebugWireframe
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugCenters = !g.DrawDebugCenters
	}

	return err

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the non-screen Models
	// g.Camera.Render(g.Scene, g.Scene.Models...)

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
	}

	if g.DrawDebugWireframe {
		g.Camera.DrawDebugWireframe(screen, g.Scene.Root, colors.White())
	}

	if g.DrawDebugCenters {
		g.Camera.DrawDebugCenters(screen, g.Scene.Root, colors.SkyBlue())
	}

	if g.DrawDebugText {
		txt := "F1 to toggle this text\nWASD: Move, Space: Go Up, Ctrl: Go Down\nMouse: Look\nArrow Keys:Move parent\nP: Toggle parenting\nG:Reset child position\nI:Toggle visibility on parent\nF1, F2, F3, F5: Debug views\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 104, color.RGBA{255, 0, 0, 255})
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {

	ebiten.SetWindowTitle("Tetra3d - Parenting Test")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
