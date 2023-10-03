package main

import (
	"bytes"
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Scene *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	DrawDebugWireframe bool
	DrawDebugCenters   bool
}

func NewGame() *Game {

	game := &Game{}

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

	cubeMesh := tetra3d.NewCubeMesh()
	mat := cubeMesh.MeshParts[0].Material
	mat.Shadeless = true
	mat.Texture = loadImage(testImageData)

	parent := tetra3d.NewModel("parent", cubeMesh)
	parent.SetLocalPositionVec(tetra3d.NewVector(0, -3, 0))

	child := tetra3d.NewModel("child", cubeMesh)
	child.SetLocalPositionVec(tetra3d.NewVector(10, 2, 0))

	g.Scene.Root.AddChildren(parent, child)

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 0, 15)

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	g.Camera.Update()

	parent := g.Scene.Root.Get("parent")
	parent.SetLocalRotation(parent.LocalRotation().Rotated(0, 1, 0, 0.05))

	child := g.Scene.Root.SearchTree().ByName("child").First()
	position := parent.LocalPosition()

	moveSpd := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		position.X -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		position.X += moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		position.Z -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		position.Z += moveSpd
	}

	parent.SetLocalPositionVec(position)

	if ebiten.IsKeyPressed(ebiten.KeyG) {
		child.SetWorldPosition(10, 2, 0)
		child.SetWorldRotation(tetra3d.NewMatrix4Rotate(0, 1, 0, 0))
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		parent.SetVisible(!parent.Visible(), true)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyO) {

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

	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	if g.System.DrawDebugText {
		txt := "Arrow Keys:Move parent\nO: Toggle parenting\nG:Reset child position\nI:Toggle visibility on parent and child"
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

	g.System.Draw(screen, g.Camera.Camera)

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {

	ebiten.SetWindowTitle("Tetra3d - Parenting Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
