package tetra3d

import (
	"math/rand"
	"sort"

	"github.com/kvartborg/vector"
)

type GridPoint struct {
	*Node
	Connections []*GridPoint
	prevLink    *GridPoint
}

func NewGridPoint(name string) *GridPoint {
	gridPoint := &GridPoint{
		Node:        NewNode(name),
		Connections: []*GridPoint{},
	}
	return gridPoint
}

func (point *GridPoint) Clone() INode {
	newPoint := NewGridPoint(point.name)
	newPoint.Connections = append(newPoint.Connections, point.Connections...)
	return newPoint
}

func (point *GridPoint) IsConnected(other *GridPoint) bool {

	for _, c := range point.Connections {
		if c == other {
			return true
		}
	}

	return false

}

func (point *GridPoint) IsOnSameGrid(other *GridPoint) bool {
	return point.parent == other.parent
}

func (point *GridPoint) Connect(other *GridPoint) {

	if !point.IsConnected(other) {
		point.Connections = append(point.Connections, other)
	}

	if !other.IsConnected(point) {
		other.Connections = append(other.Connections, point)
	}

}

func (point *GridPoint) PathTo(other *GridPoint) *GridPath {
	path := &GridPath{
		points: []vector.Vector{},
	}

	if !point.IsOnSameGrid(other) {
		return nil
	}

	if point == other {
		return &GridPath{
			points: []vector.Vector{point.WorldPosition()},
		}
	}

	toCheck := []*GridPoint{other}
	checked := map[*GridPoint]bool{}

	other.prevLink = nil
	point.prevLink = nil

	var next *GridPoint

	for true {

		next = toCheck[0]

		toCheck = toCheck[1:]
		checked[next] = true

		if next == point {
			break
		}

		for _, c := range next.Connections {
			if _, exists := checked[c]; !exists {
				c.prevLink = next
				toCheck = append(toCheck, c)
			}
		}

	}

	for next.prevLink != nil {
		path.points = append(path.points, next.WorldPosition())
		next = next.prevLink
	}

	path.points = append(path.points, other.WorldPosition())

	return path

}

////////////

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (point *GridPoint) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	point.addChildren(point, children...)
}

// Unparent unparents the Model from its parent, removing it from the scenegraph.
func (point *GridPoint) Unparent() {
	if point.parent != nil {
		point.parent.RemoveChildren(point)
	}
}

// Type returns the NodeType for this object.
func (point *GridPoint) Type() NodeType {
	return NodeTypeGridPoint
}

// Grid represents a collection of points and the connections between them.
type Grid struct {
	*Node
}

// NewGrid creates a new Grid.
func NewGrid(name string) *Grid {
	return &Grid{Node: NewNode(name)}
}

// Clone creates a clone of this GridPoint.
func (grid *Grid) Clone() INode {
	newGrid := &Grid{}
	newGrid.Node = grid.Node.Clone().(*Node)
	return newGrid
}

// Points returns a slice of the children nodes that constitute this Grid's GridPoints.
func (grid *Grid) Points() []*GridPoint {
	points := make([]*GridPoint, 0, len(grid.children))
	for _, n := range grid.children {
		if g, ok := n.(*GridPoint); ok {
			points = append(points, g)
		}
	}
	return points
}

// NearestPoint returns the nearest grid point to the given world position.
func (grid *Grid) NearestPoint(position vector.Vector) *GridPoint {

	points := grid.Points()

	sort.Slice(points, func(i, j int) bool {
		return points[i].WorldPosition().Sub(position).Magnitude() < points[j].WorldPosition().Sub(position).Magnitude()
	})

	return points[0]

}

// FurthestPoint returns the furthest grid point to the given world position.
func (grid *Grid) FurthestPoint(position vector.Vector) *GridPoint {

	points := grid.Points()

	sort.Slice(points, func(i, j int) bool {
		return points[i].WorldPosition().Sub(position).Magnitude() < points[j].WorldPosition().Sub(position).Magnitude()
	})

	return points[len(points)-1]

}

// RandomPoint returns a random grid point in the grid.
func (grid *Grid) RandomPoint() *GridPoint {
	gridPoints := grid.Points()
	return gridPoints[rand.Intn(len(gridPoints))]
}

////////

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (grid *Grid) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	grid.addChildren(grid, children...)
}

// Unparent unparents the Model from its parent, removing it from the scenegraph.
func (grid *Grid) Unparent() {
	if grid.parent != nil {
		grid.parent.RemoveChildren(grid)
	}
}

// Type returns the NodeType for this object.
func (grid *Grid) Type() NodeType {
	return NodeTypeGrid
}

// GridPath represents a sequence of grid points, used to traverse a path.
type GridPath struct {
	points []vector.Vector
}

func (gp *GridPath) Distance() float64 {

	dist := 0.0

	if len(gp.points) <= 1 {
		return 0
	}

	start := gp.points[0]

	for i := 1; i < len(gp.points); i++ {
		next := gp.points[i]
		dist += next.Sub(start).Magnitude()
		start = next
	}

	return dist

}

func (gp *GridPath) Points() []vector.Vector {
	points := make([]vector.Vector, 0, len(gp.points))
	for _, p := range gp.points {
		points = append(points, p.Clone())
	}
	return points
}

func (gp *GridPath) isClosed() bool {
	return false
}
