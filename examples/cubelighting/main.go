package main

import (
	"bytes"
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed cubeLighting.glb
var gltfData []byte

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFData(bytes.NewReader(gltfData), nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.Camera = examples.NewBasicFreeCam(g.Scene)

	g.Camera.Move(0, 10, 10)
	g.System = examples.NewBasicSystemHandler(g)

	for _, cubeLightModel := range g.Scene.Root.SearchTree().ByPropRegex("cubelight").Models() {

		cubeLight := tetra3d.NewCubeLightFromModel("cube light", cubeLightModel)
		cubeLight.SetEnergy(3)
		g.Scene.Root.AddChildren(cubeLight)

	}

	g.Scene.Root.Get("SunLight").Unparent()

}

func (g *Game) Update() error {

	cubeLight := g.Scene.Root.Get("cube light").(*tetra3d.CubeLight)

	angle := cubeLight.LightingAngle.Modify()

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		angle.RotateVec(tetra3d.WorldRight.Modify(), 0.1)
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		angle.RotateVec(tetra3d.WorldRight.Modify(), -0.1)
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		cubeLight.Bleed += 0.05
		if cubeLight.Bleed > 1 {
			cubeLight.Bleed = 1
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		cubeLight.Bleed -= 0.05
		if cubeLight.Bleed < 0 {
			cubeLight.Bleed = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.Scene.World.LightingOn = !g.Scene.World.LightingOn
	}

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color - we can use the world lighting color for this.
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
	g.Camera.RenderScene(g.Scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This example shows a Cube Light.
Cube Lights are volumes that shine from the top down.
Triangles that lie outside the (AABB)
volume remain unlit.
Left / Right Arrow Key: Rotate Light
Up / Down Arrow Key: Increase / Decrease Bleed
2 Key: Toggle all lighting
`

		g.Camera.DrawDebugText(screen, txt, 0, 220, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - LightGroup Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
