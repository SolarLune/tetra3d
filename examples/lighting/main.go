package main

import (
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed lighting.gltf
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

	library, err := tetra3d.LoadGLTFData(gltfData, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.Camera = examples.NewBasicFreeCam()
	g.System = examples.NewBasicSystemHandler(g)

	light := tetra3d.NewPointLight("camera light", 1, 1, 1, 2)
	light.Distance = 10
	light.Move(0, 1, -2)
	light.On = false
	g.Camera.Node.AddChildren(light)

}

func (g *Game) Update() error {

	spin := g.Scene.Root.Get("Spin").(*tetra3d.Model)
	spin.Rotate(0, 1, 0, 0.025)

	light := g.Scene.Root.Get("Point light").(*tetra3d.PointLight)
	light.AnimationPlayer().Play(g.Library.Animations["LightAction"])
	light.AnimationPlayer().Update(1.0 / 60.0)

	// g.Scene.Root.Get("plane").Rotate(1, 0, 0, 0.04)

	// light := g.Scene.Root.Get("third point light")
	// light.Move(math.Sin(g.Time*math.Pi)*0.1, 0, math.Cos(g.Time*math.Pi*0.19)*0.03)

	// g.Scene.Root.Get("point light").(*tetra3d.PointLight).Distance = 10 + (math.Sin(g.Time*math.Pi) * 5)

	armature := g.Scene.Root.Get("Armature")

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		armature.Move(-0.1, 0, 0)
	} else if ebiten.IsKeyPressed(ebiten.KeyRight) {
		armature.Move(0.1, 0, 0)
	}

	player := armature.AnimationPlayer()
	player.Play(g.Library.Animations["ArmatureAction"])
	player.Update(1.0 / 60.0)

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.Scene.World.LightingOn = !g.Scene.World.LightingOn
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		pointLight := g.Camera.Get("camera light").(*tetra3d.PointLight)
		pointLight.On = !pointLight.On
	}

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color - we can use the world lighting color for this.
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
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
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nThis example simply shows dynamic vertex-based lighting.\nThere are six lights in this scene:\nan ambient light, three point lights, \na single directional (sun) light,\nand one more point light parented to the camera.\n1 Key: Toggle all lighting\n2 Key: Toggle camera light\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		g.Camera.DebugDrawText(screen, txt, 0, 150, 1, colors.Red())
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	nw, nh := g.Camera.ColorTexture().Size()
	return nw, nh
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Lighting Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
