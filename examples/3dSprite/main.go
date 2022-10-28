package main

import (
	"bytes"
	"errors"
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed heart.png
var heartImg []byte

//go:embed scene.gltf
var scene []byte

type Game struct {
	Width, Height  int
	Scene          *tetra3d.Scene
	Camera         *tetra3d.Camera
	DrawDebugText  bool
	DrawDebugDepth bool

	WireframeDrawHeart bool
	HeartSprite        *ebiten.Image
}

func NewGame() *Game {
	game := &Game{
		DrawDebugText: true,
	}

	game.Init()

	return game
}

// In this example, we will simply create a cube and place it in the scene.

func (g *Game) Init() {

	scene, err := tetra3d.LoadGLTFData(scene, nil)
	if err != nil {
		panic(err)
	}
	g.Scene = scene.FindScene("Scene").Clone()

	g.Camera = g.Scene.Root.Get("Camera").(*tetra3d.Camera)

	reader := bytes.NewReader(heartImg)
	newImg, _, err := ebitenutil.NewImageFromReader(reader)
	if err != nil {
		panic(err)
	}
	g.HeartSprite = newImg

}

func (g *Game) Update() error {

	var err error

	moveSpd := 0.05
	dx := 0.0
	dz := 0.0

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		dx -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		dx += moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		dz -= moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		dz += moveSpd
	}

	g.Scene.Root.Get("Heart").Move(dx, 0, dz)

	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.WireframeDrawHeart = !g.WireframeDrawHeart
	}

	// Quit if we press Escape.
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Fullscreen toggling.
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.Init()
	}

	// Debug views

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

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(-float64(g.HeartSprite.Bounds().Dx())/2, -float64(g.HeartSprite.Bounds().Dy())/2)
	g.Camera.DrawImageIn3D(
		g.Camera.ColorTexture(),
		tetra3d.SpriteRender3d{
			Image:         g.HeartSprite,
			Options:       opt,
			WorldPosition: g.Scene.Root.Get("Heart").WorldPosition(),
		},
	)

	// Draw depth texture if the debug option is enabled; draw color texture otherwise.
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), nil)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), nil)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		g.Camera.DebugDrawText(screen, "F1 to toggle this text.\n\nThis is an example showing\nhow you can render a sprite in 2D, while\nmaintaining its ability to render over or under\nother 3D objects by simply moving\nit through 3D space.\n\nA: Toggle wireframe view of heart position\nR: Restart\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit", 0, 130, 1, colors.LightGray())
	}

	if g.WireframeDrawHeart {
		g.Camera.DrawDebugWireframe(screen, g.Scene.Root.Get("Heart"), colors.White())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	// Here, we simply return the camera's width and height.
	return g.Camera.Size()
}

func main() {

	ebiten.SetWindowTitle("Tetra3d - 3D Sprite Test")

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
