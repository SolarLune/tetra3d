package main

import (
	"embed"
	_ "embed"
	"fmt"
	"time"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed *.glb
var shapes embed.FS

type Player struct {
	Model   *tetra3d.Model
	Bounds  *tetra3d.BoundingCapsule
	Fallspd float32
}

func NewPlayer(model *tetra3d.Model) *Player {
	return &Player{Model: model, Bounds: model.Get("BoundingCapsule").(*tetra3d.BoundingCapsule)}
}

func (p *Player) Update() {

	cam := p.Model.Scene().Root.SearchTree().ByType(tetra3d.NodeTypeCamera).First().(*tetra3d.Camera)
	// cam := p.Model.Scene().Get("Camera").(*tetra3d.Camera)

	camRight := cam.Transform().Right()
	camForward := cam.Transform().Forward().Invert()

	camRight.Y = 0
	camForward.Y = 0

	camRight = camRight.Unit()
	camForward = camForward.Unit()

	// p.Model.Rotate(0, 1, 0, 0.01)

	mv := tetra3d.NewVector3(0, 0, 0)
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		mv = mv.Add(camForward)
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		mv = mv.Add(camForward.Invert())
	}

	if ebiten.IsKeyPressed(ebiten.KeyD) {
		mv = mv.Add(camRight)
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		mv = mv.Add(camRight.Invert())
	}

	movespd := float32(0.1)

	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		movespd = 0.2
	}

	p.Model.MoveVec(mv.Unit().Scale(movespd))

	sceneTree := p.Bounds.Scene().Root.SearchTree()

	p.Bounds.CollisionTest(tetra3d.CollisionTestSettings{
		TestAgainst: sceneTree,
		OnCollision: func(col *tetra3d.Collision, index, count int) bool {
			mtv := col.AverageMTV()
			mtv.Y = 0
			p.Model.MoveVec(mtv)
			return true
		},
	})

	onGround := false

	tetra3d.RayTest(tetra3d.RayTestOptions{
		From:        p.Model.WorldPosition(),
		To:          p.Model.WorldPosition().SubY(2),
		TestAgainst: sceneTree.ByParentProps("solid"),
		OnHit: func(hit tetra3d.RayHit, index, count int) bool {
			onGround = true
			p.Fallspd = 0
			p.Model.SetLocalY(hit.Position.Y + 1.2)
			return false
		},
	})

	if !onGround {
		p.Fallspd += 0.02
	}

	p.Model.Move(0, -p.Fallspd, 0)

}

type Game struct {
	Scene *tetra3d.Scene

	Player *Player

	Camera      examples.BasicTargetedCam
	System      examples.BasicSystemHandler
	LightVolume *tetra3d.LightVolume
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	// Load the GLTF file and turn it into a Library, which is a collection of scenes and data shared between them (like meshes or animations).

	g.System.UsingBasicFreeCam = true

	g.System = examples.NewBasicSystemHandler(g)

	library, err := tetra3d.LoadGLTFFileSystem(shapes, "lightvolume.glb", nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.SceneByName("Scene")

	g.Player = NewPlayer(g.Scene.Root.SearchTree().ByNamePartial("Player").First().(*tetra3d.Model))

	g.Camera = examples.NewBasicTargetedCam(g.Player.Model)

	// Bake the AO on the map to make it look a little nicer.
	m := g.Scene.Get("Map").(*tetra3d.Model)
	opt := tetra3d.NewDefaultAOBakeOptions()
	opt.TargetVertices = tetra3d.NewVertexSelection().SelectByMeshPartName(m.Mesh, "Walls", "Grass")
	m.BakeAO(opt)

	lightVolumeCube := g.Scene.Get("LightVolume").(*tetra3d.Model)
	// Create a light volume from the LightVolume model. It will use its current name and dimensions.
	// A light volume is essentially a grid of cells - in this case, the cell size is 2x2x2.
	lightvolume := tetra3d.NewLightVolumeFromModel(lightVolumeCube, 2, 2, 2)
	g.LightVolume = lightvolume
	g.Scene.Root.AddChildren(lightvolume)

	// Once we've added the LightVolume to the scene, we don't need the original cube anymore.
	lightVolumeCube.Unparent()

	// This gets the objects to perform the ray test against.
	testAgainst := g.Scene.Root.SearchTree()

	// For each grid cell in the LightVolume, we take in the position of the grid cell and the current color,
	// and then we shade that cell - anything that passes through the cell is lightened, darkened, or otherwise colored
	// depending on the color we put in the cell.

	t := time.Now()
	lightvolume.LightCellReadWriteAll(func(position tetra3d.Vector3, in tetra3d.Color) tetra3d.Color {

		// Start off assuming darkness, with a light value of -1 (negative light / darkness).
		v := float32(-1)

		up := tetra3d.RayTest(tetra3d.RayTestOptions{
			From:        position,
			To:          position.AddY(1000),
			TestAgainst: testAgainst,
			Doublesided: true,
		})

		// Perform a ray test - if we hit a triangle that has the "Sky" material, then we'll say that this point
		// is exposed to the sky / sun, and so is bright.
		if up != nil && up.Triangle != nil {

			if up.Triangle.MeshPart.Material.Name() == "Sky" {
				// Set the light value to 0 (no change).
				v = 0
			} else if up.Triangle.MeshPart.Material.Name() == "LightCube" {
				v = 2
			}
		}

		in.R = v
		in.G = v
		in.B = v

		return in
	})
	fmt.Println(time.Since(t))

	// It's a little ugly, so let's blur the result.
	lightvolume.LightCellBlur(1, 1)

}

func (g *Game) Update() error {

	g.Player.Update()

	g.Camera.Update()

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.LightVolume.SetOn(!g.LightVolume.On())
	}

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	g.Camera.ClearWithColor(g.Scene.World.FogColor)
	// g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// g.Scene.Get("LightVolume").(*tetra3d.LightVolume).DebugDraw(screen, g.Camera.Camera)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This example showcases light volumes.
As you walk around the map, the character
gets darker or lighter depending on his
environment.`
		g.Camera.DrawDebugText(screen, txt, 0, 220, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Light Volume Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
