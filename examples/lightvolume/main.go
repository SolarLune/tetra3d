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
	Model   *tetra3d.Model
	Bounds  *tetra3d.BoundingCapsule
	Fallspd float32
	Facing  tetra3d.Vector3
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
	solids := p.Bounds.Scene().Root.Search(tetra3d.SearchOptions{}.ByProps("solid").WithReturnChildren(true))

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
		TestAgainst: solids,
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

	g.Player = NewPlayer(g.Scene.Root.Search(tetra3d.SearchOptions{}.ByProps("player")).First().(*tetra3d.Model))

	g.Scene.Root.Search(tetra3d.SearchOptions{}.ByProps("guy")).ForEach(func(node tetra3d.INode) bool {
		g.Guys = append(g.Guys, NewGuy(node.(*tetra3d.Model)))
		return true
	})

	g.Camera = examples.NewBasicTargetedCam(g.Player.Model)

	t := time.Now()

	// So we get the LightVolume, which is a blank slate at this point.
	// Now to populate it with light data!
	lightvolume := g.Scene.Root.Get("LightVolume").(*tetra3d.LightVolume)

	// For each grid cell in the LightVolume, we take in the position of the grid cell and the current color,
	// and then we shade that cell - anything that passes through the cell is lightened, darkened, or otherwise colored
	// depending on the color we put in the cell.

	solids := g.Scene.Root.Search(tetra3d.SearchOptions{}.ByParentProps("solid"))

	selectedCellsForAnimation := []*tetra3d.LightVolumeCell{}

	lightvolume.LightVolumeForEachCell(func(position tetra3d.Vector3, cell *tetra3d.LightVolumeCell) {

		tetra3d.RayTest(tetra3d.RayTestOptions{
			From:        position,
			To:          position.SubY(8),
			TestAgainst: g.Scene.Root.Children(true),
			OnHit: func(hit tetra3d.RayHit, index, count int) bool {

				if hit.Triangle != nil {
					mat := hit.Triangle.MeshPart.Material
					if mat.Name() == "LightCube" {
						cell.UseLights(g.Scene.Root.Get("Underlight").(*tetra3d.DirectionalLight))

						selectedCellsForAnimation = append(selectedCellsForAnimation, cell)
					}
				}

				return true
			},
		})

		up := tetra3d.RayTest(tetra3d.RayTestOptions{
			From:        position.AddY(0.5),
			To:          position.AddY(1000),
			TestAgainst: solids,
			Doublesided: true,
		})

		// Perform a ray test - if we hit a triangle that has the "Sky" material, then we'll say that this point
		// is exposed to the sky / sun, and so is bright.
		if up != nil {

			mat := up.Triangle.MeshPart.Material

			if mat.Name() == "Sky" {
				cell.SetColorMonochrome(1.25)
			} else if mat.Name() == "LightCube" {
				vertexColor, _ := up.VertexColor(0)
				cell.SetColor(vertexColor.MultiplyScalarRGB(1.5))
			} else {
				cell.SetColorMonochrome(-1)
			}

		} else {
			cell.SetBlocked(true)
		}

	})

	fmt.Println("time to check light cells:", time.Since(t))

	lightvolume.LightVolumeResize(lightvolume.Dimensions(), 2, 2, 2)

	lightvolume.LightVolumeBlur(1, 2, 1)

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
		g.LightVolume.LightVolumeDebugDraw(screen, g.Camera.Camera)
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
