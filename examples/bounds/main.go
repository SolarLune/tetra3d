package main

import (
	"bytes"
	"fmt"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed bounds.glb
var gltfData []byte

type Game struct {
	Scene *tetra3d.Scene

	Controlling   *tetra3d.Model
	Movement      tetra3d.Vector3
	VerticalSpeed float32

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

	g.Scene = library.SceneByName("Scene")

	g.Controlling = g.Scene.Root.Get("YellowCapsule").(*tetra3d.Model)

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.SetLocalPosition(0, 6, 15)
	g.Camera.SetFar(40)

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	friction := float32(0.05)
	maxSpd := float32(0.25)
	accel := float32(0.05) + friction
	gravity := float32(0.05)

	bounds := g.Controlling.Children()[0].(tetra3d.IBoundingObject)
	solids := g.Scene.Root.SearchTree().IBoundingObjects()

	movement := g.Movement.Modify() // Modification Vector

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		movement.X += accel
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		movement.X -= accel
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		movement.Z -= accel
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		movement.Z += accel
	}

	movement.SubMagnitude(friction)

	movement.ClampMagnitude(maxSpd)

	if g.VerticalSpeed < -0.3 {
		g.VerticalSpeed = -0.3
	}

	g.VerticalSpeed -= gravity

	// Now, check for collisions.

	// Horizontal check (walls):

	g.Controlling.MoveVec(g.Movement)

	margin := float32(0.01)

	bounds.CollisionTest(tetra3d.CollisionTestSettings{

		OnCollision: func(col *tetra3d.Collision, index, count int) bool {
			mtv := col.AverageMTV()
			mtv.Y = 0                                       // We don't want to move up to avoid collision
			g.Controlling.MoveVec(mtv.Expand(margin, 0.01)) // Move out of the collision, but add a little margin
			return true
		},

		TestAgainst: solids,
	})

	// Vertical check (floors):

	g.Controlling.Move(0, g.VerticalSpeed, 0)

	bounds.CollisionTest(tetra3d.CollisionTestSettings{

		OnCollision: func(col *tetra3d.Collision, index, count int) bool {
			g.Controlling.Move(0, col.AverageMTV().Y+margin, 0) // Move the object up, so that it's on the ground, plus a little margin
			g.VerticalSpeed = 0
			return true
		},

		TestAgainst: solids,
	})

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		sphere := g.Scene.Root.Get("Sphere").(*tetra3d.Model)
		capsule := g.Scene.Root.Get("YellowCapsule").(*tetra3d.Model)
		if g.Controlling == sphere {
			g.Controlling = capsule
		} else {
			g.Controlling = sphere
		}
	}

	g.Camera.Update()
	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := fmt.Sprintf(`This demo shows some basic movement 
and collision detection. The red and white cubes
have BoundingAABB nodes, while the capsule and sphere are,
naturally, capsule and sphere BoundingObjects.
Arrow keys: Move %s
F: switch between capsule and sphere`, g.Controlling.Name())
		g.Camera.DrawDebugText(screen, txt, 0, 220, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Shapes Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
