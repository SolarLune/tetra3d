package main

import (
	"bytes"
	"image/color"

	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// This is a relatively complex-looking example, but in truth, it's rather simple.
// It mainly exists to show how you can toggle passibility (or change costs, for that matter), for
// one side of a connection between GridPoints in a grid.

// GridPointMarker is just there to show which nodes are connected.

type GridPointMarker struct {
	Model     *tetra3d.Model
	Gridpoint *tetra3d.GridPoint
	lines     []*tetra3d.Model
}

func NewGridPointMarker(gridPoint *tetra3d.GridPoint) *GridPointMarker {

	scene := gridPoint.Scene()
	model := scene.Get("GridPointMarker").Clone()
	gridPoint.AddChildren(model)
	model.SetLocalPosition(0, 0, 0)
	marker := &GridPointMarker{
		Model:     model.(*tetra3d.Model),
		Gridpoint: gridPoint,
	}

	for _, connection := range gridPoint.Connections {
		line := scene.Get("Line").Clone()
		line.SetLocalPosition(0, 0, 0)
		model.AddChildren(line)
		rot := tetra3d.NewLookAtMatrix(gridPoint.WorldPosition(), connection.To.WorldPosition(), tetra3d.WorldUp)
		dist := gridPoint.DistanceTo(connection.To)
		line.SetLocalScale(1, 1, dist)
		line.SetWorldRotation(rot)
		marker.lines = append(marker.lines, line.(*tetra3d.Model))
	}

	return marker
}

func (m *GridPointMarker) Update() {

	lib := m.Model.Library()

	red := false

	for i, connection := range m.Gridpoint.Connections {
		if connection.Passable {
			m.lines[i].Mesh = lib.Meshes["Line"]
		} else {
			red = true
			m.lines[i].Mesh = lib.Meshes["BrokenLine"]
		}
		if connection.Cost > 0 {
			m.lines[i].Color = colors.Green()
		} else {
			m.lines[i].Color = colors.White()
		}
	}

	m.Model.Color = colors.White()

	if red {
		m.Model.Color = colors.PaleRed()
	}

	// for _, connection := range m.

}

// PathAgent is our pathfinding agent. It moves from one point towards another.
type PathAgent struct {
	Model       *tetra3d.Model
	Root        *tetra3d.Node
	PathStepper *tetra3d.PathStepper
}

func NewPathAgent(modelNode tetra3d.INode) *PathAgent {

	// We specify no path for the PathStepper at first, as it's still at the beginning of the example.
	cube := &PathAgent{
		Model:       modelNode.(*tetra3d.Model),
		PathStepper: tetra3d.NewPathStepper(nil),
	}

	return cube
}

func (p *PathAgent) Update() {

	if p.PathStepper.Path() != nil {

		moveSpd := 0.05

		currentPos := p.PathStepper.CurrentWorldPosition()
		diff := currentPos.Sub(p.Model.WorldPosition()).ClampMagnitude(moveSpd)
		p.Model.MoveVec(diff)

		if currentPos.Equals(p.Model.WorldPosition()) {
			if !p.PathStepper.AtEnd() {
				p.PathStepper.Next()
			}

		}

	}

}

type Game struct {
	Scene  *tetra3d.Scene
	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	Cursor           *tetra3d.Model
	Grid             *tetra3d.Grid
	Cube             *PathAgent
	GridPointMarkers []*GridPointMarker
}

//go:embed grids.gltf
var grids []byte

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	data, err := tetra3d.LoadGLTFData(bytes.NewReader(grids), nil)
	if err != nil {
		panic(err)
	}

	g.Scene = data.Scenes[0]

	// g.Camera = g.Scene.Root.Get("Camera").(*tetra3d.Camera)
	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.Move(0, 2, 0)
	g.System = examples.NewBasicSystemHandler(g)
	g.Cube = NewPathAgent(g.Scene.Root.Get("Cube"))

	g.Grid = g.Scene.Root.Get("Network").(*tetra3d.Grid)

	// Create markers for each point in a Grid
	g.Grid.ForEachPoint(
		func(gridPoint *tetra3d.GridPoint) {
			g.GridPointMarkers = append(g.GridPointMarkers, NewGridPointMarker(gridPoint))
		},
	)

}

func (g *Game) Update() error {

	cursor := g.Scene.Root.Get("Cursor")

	g.Cube.Update()

	for _, marker := range g.GridPointMarkers {
		marker.Update()
	}

	cursor.Rotate(0, 1, 0, 0.04)

	// See which grid point marker we're hovering over with the mouse.
	g.Camera.MouseRayTest(tetra3d.MouseRayTestOptions{
		Depth: 100,
		OnHit: func(hit tetra3d.RayHit, hitIndex int, hitCount int) bool {

			// The bounding objects's parent (model)'s parent is the grid point node
			targetPoint := hit.Object.Get("../../").(*tetra3d.GridPoint)

			cursor.SetLocalPositionVec(targetPoint.LocalPosition())
			cursor.SetVisible(true, true)

			if inpututil.IsKeyJustPressed(ebiten.KeyE) {
				for _, c := range targetPoint.Connections {
					c.Passable = !c.Passable
				}
			}

			if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
				for _, c := range targetPoint.Connections {
					// c.Passable = !c.Passable
					if c.Cost == 0 {
						c.Cost = 100
					} else {
						c.Cost = 0
					}
				}
			}

			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				from := g.Grid.ClosestGridPoint(g.Cube.Model.LocalPosition())
				to := targetPoint
				path := from.PathTo(to)

				if path != nil {
					g.Cube.PathStepper.SetPath(path)
				}
			}

			return false // Don't continue stepping through ray hits beyond the first one

		},
		TestAgainst: g.Scene.Root.SearchTree().ByParentProps(false, "gridPoint"),
	})

	g.Camera.Update()

	return g.System.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
	g.Camera.RenderScene(g.Scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This example shows how GridPoint connections work.
Connections between GridPoints are two-fold.
This is so that each connection (from A to B, and B to A)
can have different costs or passability values.

Mouse over nodes to select them.
The E key toggles passability >from< the selected Node.
The Q key toggles increased costs from the selected Node.
Left-click moves the cube to the selected Node.
`
		g.Camera.DebugDrawText(screen, txt, 0, 220, 1, colors.White())

	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Webs Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
