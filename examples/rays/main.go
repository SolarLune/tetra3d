package main

import (
	"bytes"
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed rays.glb
var blendFile []byte

type Game struct {
	Scene *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	// Load the GLTF file and turn it into a Library, which is a collection of scenes and data shared between them (like meshes or animations).

	library, err := tetra3d.LoadGLTFData(bytes.NewReader(blendFile), nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.SceneByName("Scene")

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.System = examples.NewBasicSystemHandler(g)
	// g.Camera.Resize(320, 180)

	g.Camera.SetFieldOfView(100)

}

func (g *Game) Update() error {

	g.Camera.MouseRayTest(tetra3d.MouseRayTestOptions{
		// Depth: g.Camera.Far(),
		TestAgainst: g.Scene.Root.SearchTree().IBoundingObjects(),

		OnHit: func(hit tetra3d.RayHit, hitIndex int, hitCount int) bool {

			marker := g.Scene.Root.Get("Marker")

			// Here, we'll set the color of the marker.
			vc, err := hit.VertexColor(0)

			// If there's an error, it's because we're not colliding against a BoundingTriangles,
			// so there's no triangle to pull UV or vertex color data from.

			// If that's the case, we can just go with the material color instead.
			if err != nil && err.Error() == tetra3d.ErrorObjectHitNotBoundingTriangles {
				vc = hit.Object.Parent().(*tetra3d.Model).Mesh.MeshParts[0].Material.Color
			}

			tetra3d.NewVertexSelection().SelectMeshPartByIndex(marker.(*tetra3d.Model).Mesh, 1).SetColor(0, vc)

			marker.SetWorldPositionVec(hit.Position.Add(hit.Normal.Scale(0.5)))

			mat := tetra3d.NewLookAtMatrix(marker.WorldPosition(), hit.Position, tetra3d.WorldUp)

			marker.SetWorldRotation(mat)

			return false
		},
	})

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.FogColor.ToRGBA64())

	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `In this example, a ray is cast
forward from the mouse's position. Whatever is hit will
be marked with the arrow, which will also change color 
to match the object struck. If the mouse is locked
to the game window, then the ray shoots directly forward
from the center of the screen.
`
		g.Camera.DrawDebugText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Ray Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
