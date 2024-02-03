package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed cameraTest.glb
var shapes []byte

type Game struct {
	Scene  *tetra3d.Scene
	System examples.BasicSystemHandler
	Camera *tetra3d.Camera
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	// Load the GLTF file and turn it into a Library, which is a collection of scenes and data shared between them (like meshes or animations).

	library, err := tetra3d.LoadGLTFData(bytes.NewReader(shapes), nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.FindScene("Scene")

	g.Camera = g.Scene.Root.Get("Shallow").(*tetra3d.Camera)

}

func (g *Game) Update() error {

	cameras := g.Scene.Root.SearchTree().ByType(tetra3d.NodeTypeCamera).INodes()

	n := 0

	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		n = 1
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		n = -1
	}

	if n != 0 {

		for i, c := range cameras {
			if c == g.Camera {
				n += i
				break
			}
		}

		if n >= len(cameras) {
			n = 0
		} else if n < 0 {
			n = len(cameras) - 1
		}

		g.Camera = cameras[n].(*tetra3d.Camera)

	}

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera)

	if g.System.DrawDebugText {
		txt := `This test is a simple camera test,
designed to check out different camera
field of view settings to ensure they
match with Blender's output.

Left and right cycle through the cameras.`

		if g.Camera.Perspective() {
			txt += fmt.Sprintf("\n\nName: %s\nType: Perspective Camera\nFOV: %s", g.Camera.Name(), strconv.Itoa(int(g.Camera.FieldOfView())))
		} else {
			txt += fmt.Sprintf("\n\nName: %s\nType: Orthographic Camera\nOrtho-Scale: %s", g.Camera.Name(), strconv.FormatFloat(g.Camera.OrthoScale(), 'f', 1, 64))
		}
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Camera Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
