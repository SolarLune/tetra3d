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
var assets embed.FS

type Player struct {
	Model       *tetra3d.Model
	Bounds      *tetra3d.BoundingCapsule
	VerticalSpd float32
	Facing      tetra3d.Vector3
}

func NewPlayer(model *tetra3d.Model) *Player {
	// model.Get("PlayerModelHead").(*tetra3d.Model).Mesh.MeshParts[0].Material.LightVolumeShadingMode = tetra3d.LightVolumeShadingModePerVertexWithoutNormal
	return &Player{Model: model, Bounds: model.Get("BoundingCapsule").(*tetra3d.BoundingCapsule)}
}

func (p *Player) Update() {

	cam := p.Model.Scene().Root.Search(tetra3d.SearchOptions{}.ByType(tetra3d.NodeTypeCamera)).First().(*tetra3d.Camera)
	// cam := p.Model.Scene().Get("Camera").(*tetra3d.Camera)

	camRight := cam.Transform().Right()
	camForward := cam.Transform().Forward().Invert()

	camRight.Y = 0
	camForward.Y = 0

	camRight = camRight.Unit()
	camForward = camForward.Unit()

	p.Model.Get("PlayerModelBody").Rotate(0, 1, 0, 0.01)

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

	if mv.MagnitudeSquared() > 0.1 {
		p.Facing = mv.Unit()
		p.Model.Get("PlayerModelHead").SetWorldRotation(tetra3d.NewMatrix4LookAt(tetra3d.NewVector3(0, 0, 0), p.Facing, tetra3d.WorldUp))
	}

	movespd := float32(0.1)

	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		movespd = 0.2
	}

	p.Model.MoveVec(mv.Unit().Scale(movespd))

	sceneTree := p.Bounds.Scene().Root.Children(true)
	solids := p.Bounds.Scene().Root.Search(tetra3d.SearchOptions{}.ByPropNamesParent("solid"))

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

	if p.VerticalSpd <= 0 {

		tetra3d.RayTest(tetra3d.RayTestOptions{
			From:        p.Model.WorldPosition().AddY(p.VerticalSpd),
			To:          p.Model.WorldPosition().AddY(p.VerticalSpd).SubY(2),
			TestAgainst: solids,
			OnHit: func(hit tetra3d.RayHit, index, count int) bool {
				onGround = true
				p.VerticalSpd = 0
				p.Model.SetLocalY(hit.Position.Y + 1.2)
				return false
			},
		})

	}

	if !onGround {
		p.VerticalSpd -= 0.02
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		p.VerticalSpd = 0.1
	}

	p.Model.Move(0, p.VerticalSpd, 0)

}

type Guy struct {
	Model       *tetra3d.Model
	Bounds      *tetra3d.BoundingCapsule
	PathStepper *tetra3d.PathStepper
}

func NewGuy(node *tetra3d.Model) *Guy {
	return &Guy{
		Model:       node,
		PathStepper: tetra3d.NewPathStepper(nil),
		Bounds:      node.Get("BoundingCapsule").(*tetra3d.BoundingCapsule),
	}
}

func (g *Guy) Update() {

	grid := g.Model.Scene().Get("Grid").(*tetra3d.Grid)

	if g.PathStepper.Path() == nil {

		point := grid.ClosestGridPoint(g.Model.WorldPosition())
		g.PathStepper.SetPath(point.PathTo(grid.RandomPoint()))

	} else {

		if g.PathStepper.Current().DistanceTo(g.Model.WorldPosition()) < 0.1 {
			g.PathStepper.GotoNext()
			if g.PathStepper.AtEnd() {
				g.PathStepper.SetPath(nil)
			}
		} else {
			g.Model.MoveTowardsVec(g.PathStepper.Current(), 0.05)
		}

	}

	// sceneTree := g.Model.Scene().Root.SearchTree()

	// g.Bounds.CollisionTest(tetra3d.CollisionTestSettings{
	// 	TestAgainst: sceneTree,
	// 	OnCollision: func(col *tetra3d.Collision, index, count int) bool {
	// 		mtv := col.AverageMTV()
	// 		mtv.Y = 0
	// 		g.Model.MoveVec(mtv)
	// 		return true
	// 	},
	// })

	// grid := g.Model.Scene().Get("Grid").(*tetra3d.Grid)

}

type Game struct {
	Scene *tetra3d.Scene

	Player *Player
	Guys   []*Guy

	Time float32

	Camera                  examples.BasicTargetedCam
	System                  examples.BasicSystemHandler
	LightVolume             *tetra3d.LightVolume
	DebugDrawingLightVolume bool
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

	library, err := tetra3d.LoadGLTFFileSystem(assets, "lightvolume.glb", nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.SceneByName("Scene")

	g.Player = NewPlayer(g.Scene.Root.Search(tetra3d.SearchOptions{}.ByPropNames("player")).First().(*tetra3d.Model))

	g.Scene.Root.Search(tetra3d.SearchOptions{}.ByPropNames("guy")).ForEach(func(node tetra3d.INode) bool {
		g.Guys = append(g.Guys, NewGuy(node.(*tetra3d.Model)))
		return true
	})

	g.Camera = examples.NewBasicTargetedCam(g.Player.Model)

	t := time.Now()

	g.Scene.Root.Search(tetra3d.SearchOptions{}.ByPropNames("solid")).ForEachModel(func(model *tetra3d.Model) bool {
		model.Mesh().ForEachMaterial(func(mat *tetra3d.Material) bool {
			mat.LightVolumeShadingMode = tetra3d.LightVolumeShadingModePerVertexWithNormal
			return true
		})
		return true
	})

	// So we get the LightVolume, which is a blank slate at this point.
	// Now to populate it with light data!
	lightvolume := g.Scene.Root.Get("LightVolume").(*tetra3d.LightVolume)

	// For each grid cell in the LightVolume, we take in the position of the grid cell and the current color,
	// and then we shade that cell - anything that passes through the cell is lightened, darkened, or otherwise colored
	// depending on the color we put in the cell.

	solids := g.Scene.Root.Search(tetra3d.SearchOptions{}.ByPropNamesParent("solid"))

	lightvolume.LightVolumeResize(lightvolume.Dimensions(), 2, 2, 2)

	// Let's define some tags; tags are just numbers indicating information in a cell.
	const TAG_LIGHT_FROM_ABOVE = 0
	const TAG_LIGHT_FROM_BELOW = 1

	// Loop through each cell and do something with it.
	lightvolume.LightVolumeForEachCell(func(cell *tetra3d.LightVolumeCell) bool {

		// Check for collision; if the cell collides with a wall, then we know it's against a ceiling, wall, floor, etc.
		tetra3d.CollisionTestSphereVec(cell.WorldPosition(), 0.5, tetra3d.CollisionTestSettings{

			TestAgainst: solids,

			OnCollision: func(col *tetra3d.Collision, index, count int) bool {

				// Loop through each collided material and see what we get.
				col.ForEachCollidedMaterial(func(mat *tetra3d.Material) bool {
					if mat.Name() == "Sky" {
						// Touching the sky, so we can make the cell bright.
						cell.AddTag(TAG_LIGHT_FROM_ABOVE)
						cell.SetColorMonochrome(2)
					} else if mat.Name() == "LightFloor" {
						// Touching the LightFloor material, so it deals with light coming up from below.
						cell.AddTag(TAG_LIGHT_FROM_BELOW)
					} else if mat.Name() == "LightCeiling" {
						cell.AddTag(TAG_LIGHT_FROM_ABOVE)
						cell.SetColorMonochrome(3)
					} else {
						cell.SetBlockedAll(true)
					}
					return true
				})

				return true
			},
		})

		return true

	})

	underLight := g.Scene.Get("Underlight").(tetra3d.ILight)

	// Propagate the light downwards for the sky or ceiling
	lightvolume.LightVolumeForEachCell(func(cell *tetra3d.LightVolumeCell) bool {

		if cell.HasTag(TAG_LIGHT_FROM_ABOVE) {
			cell.ForEachNeighborInDirection(tetra3d.LightVolumeDirectionDown, func(neighbor *tetra3d.LightVolumeCell) bool {
				if neighbor.Blocked(tetra3d.LightVolumeDirectionNone) {
					return false
				}
				neighbor.SetColor(cell.Color())
				return true
			})
		}

		// Light floor

		if cell.HasTag(TAG_LIGHT_FROM_BELOW) {

			// We can call `cell.UseLights()` to simply light an object with another actual Light if it's within a Cell.
			cell.UseLights(underLight)
			cell.ForEachNeighborInDirection(tetra3d.LightVolumeDirectionUp, func(neighbor *tetra3d.LightVolumeCell) bool {
				if neighbor.Blocked(0) {
					return false
				}
				neighbor.UseLights(underLight)
				return true
			})
		}

		return true
	})

	lightvolume.LightVolumeBlur(1, 1, 1)

	fmt.Println("time to check light cells:", time.Since(t))

	g.LightVolume = lightvolume

}

func (g *Game) Update() error {

	g.Player.Update()
	for _, guy := range g.Guys {
		guy.Update()
	}

	g.Time += 1.0 / 60.0

	g.Camera.Update()

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.LightVolume.SetVisible(!g.LightVolume.IsVisible(), false)
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.DebugDrawingLightVolume = !g.DebugDrawingLightVolume
	}

	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		m := g.Scene.Get("Map").(*tetra3d.Model)
		m.Mesh().AutoSubdivide = !m.Mesh().AutoSubdivide
	}

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	g.Camera.Clear()
	// g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// g.Scene.Get("LightVolume").(*tetra3d.LightVolume).DebugDraw(screen, g.Camera.Camera)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This example showcases light volumes.
As you walk around the map, the character
gets darker or lighter depending on his
environment.
Press 1 to toggle the light volume.`
		g.Camera.DrawDebugText(screen, txt, 0, 230, 1, colors.LightGray())
	}

	if g.DebugDrawingLightVolume {
		g.LightVolume.LightVolumeDebugDraw(screen, g.Camera.Camera, g.Camera.Parent().WorldPosition(), 8)
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
