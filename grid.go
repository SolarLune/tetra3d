package tetra3d

import (
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// GridPoint represents a point on a Grid, used for pathfinding or connecting points in space.
// GridPoints are parented to a Grid and the connections are created seperate from their positions,
// which means you can move GridPoints freely after creation. Note that GridPoints consider themselves
// to be in the same Grid only if they have the same direct parent (being the Grid), so manually reparenting
// GridPoints is not advised.
type GridPoint struct {
	*Node
	Connections []*GridPoint
	prevLink    *GridPoint
	Cost        int
}

// NewGridPoint creates a new GridPoint.
func NewGridPoint(name string) *GridPoint {
	gridPoint := &GridPoint{
		Node:        NewNode(name),
		Connections: []*GridPoint{},
	}
	return gridPoint
}

// Clone clones the given GridPoint.
func (point *GridPoint) Clone() INode {
	newPoint := &GridPoint{
		Node:        point.Node.Clone().(*Node),
		Connections: append([]*GridPoint{}, point.Connections...),
	}
	for _, child := range newPoint.children {
		child.setParent(newPoint)
	}
	return newPoint
}

// IsConnected returns if the GridPoint is connected to the given other GridPoint.
func (point *GridPoint) IsConnected(other *GridPoint) bool {

	for _, c := range point.Connections {
		if c == other {
			return true
		}
	}

	return false

}

// IsOnSameGrid returns if the grid point is on the same grid as the other given GridPoint.
func (point *GridPoint) IsOnSameGrid(other *GridPoint) bool {
	return point.parent == other.parent
}

// Connect connects the GridPoint to the given other GridPoint.
func (point *GridPoint) Connect(other *GridPoint) {

	if point == other {
		return
	}

	if !point.IsConnected(other) {
		point.Connections = append(point.Connections, other)
	}

	if !other.IsConnected(point) {
		other.Connections = append(other.Connections, point)
	}

}

// Disconnect disconnects the GridPoint from the given other GridPoint.
func (point *GridPoint) Disconnect(other *GridPoint) {

	if point == other {
		return
	}

	for i := len(point.Connections) - 1; i >= 0; i-- {
		if point.Connections[i] == other {
			point.Connections[i] = nil
			point.Connections = append(point.Connections[:i], point.Connections[i+1:]...)
		}
	}

	for i := len(other.Connections) - 1; i >= 0; i-- {
		if other.Connections[i] == point {
			other.Connections[i] = nil
			other.Connections = append(other.Connections[:i], other.Connections[i+1:]...)
		}
	}

}

// DisconnectAll disconnects the GridPoint from all other GridPoints.
func (point *GridPoint) DisconnectAll() {
	for i := len(point.Connections) - 1; i >= 0; i-- {
		point.Disconnect(point.Connections[i])
	}
}

// PathTo creates a path going from the GridPoint to the given other GridPoint. This path is currently generated
// using the best-case metric of smallest number of hops, not necessarily the smallest distance (so Grids should generally
// be composed of evenly spaced points if the purpose is for shortest-distance pathfinding).
// If a path is not possible from the starting point to the end point, then PathTo will return nil.
func (point *GridPoint) PathTo(goal *GridPoint) *GridPath {
	path := &GridPath{
		GridPoints: []Vector{},
	}

	if !point.IsOnSameGrid(goal) {
		return nil
	}

	if point == goal {
		return &GridPath{
			GridPoints: []Vector{point.WorldPosition()},
		}
	}

	toCheck := []*GridPoint{goal}
	checked := map[*GridPoint]bool{}

	goal.prevLink = nil
	point.prevLink = nil

	var next *GridPoint

	for {

		if next == point {
			break
		}

		// No more nodes to check; the two points are on the same grid, but are inaccessible to each other
		if len(toCheck) == 0 {
			return nil
		}

		next = toCheck[0]

		toCheck = toCheck[1:]
		checked[next] = true

		for _, c := range next.Connections {
			// Hasn't been checked yet
			if _, exists := checked[c]; !exists {
				c.prevLink = next

				// A neighbor has a connection to the goal; let's go with it
				if c == point {
					next = c
					break
				}

				if len(toCheck) > 0 && c.Cost < toCheck[0].Cost {
					toCheck = append(append(make([]*GridPoint, 0, len(toCheck)+1), c), toCheck...)
				} else {
					toCheck = append(toCheck, c)
				}
			}
		}

	}

	for next.prevLink != nil {
		path.GridPoints = append(path.GridPoints, next.WorldPosition())
		next = next.prevLink
	}

	path.GridPoints = append(path.GridPoints, goal.WorldPosition())

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

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (point *GridPoint) Index() int {
	if point.parent != nil {
		for i, c := range point.parent.Children() {
			if c == point {
				return i
			}
		}
	}
	return -1
}

// Type returns the NodeType for this object.
func (point *GridPoint) Type() NodeType {
	return NodeTypeGridPoint
}

// Grid represents a collection of points and the connections between them. A Grid can be used for pathfinding
// or simply for connecting points in space (like for a world map in a level-based game, for example).
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

	for _, child := range newGrid.children {
		child.setParent(newGrid)
	}

	for _, c := range newGrid.Points() {
		c.Connections = []*GridPoint{}
	}

	for _, c := range grid.Points() {

		start := newGrid.NearestGridPoint(c.LocalPosition())
		for _, connect := range c.Connections {
			end := newGrid.NearestGridPoint(connect.LocalPosition())
			start.Connect(end)
		}

	}

	return newGrid
}

// Points returns a slice of the children nodes that constitute this Grid's GridPoints.
func (grid *Grid) Points() []*GridPoint {
	points := make([]*GridPoint, 0, len(grid.children))
	for _, n := range grid.children {
		if gp, ok := n.(*GridPoint); ok {
			points = append(points, gp)
		}
	}
	return points
}

// ForEachPoint returns a slice of the children nodes that constitute this Grid's GridPoints.
func (grid *Grid) ForEachPoint(forEach func(gridPoint *GridPoint)) {
	for _, n := range grid.children {
		if gp, ok := n.(*GridPoint); ok {
			forEach(gp)
		}
	}
}

// DisconnectAllPoints disconnects all points from each other in the Grid.
func (grid *Grid) DisconnectAllPoints() {
	for _, point := range grid.Points() {
		point.DisconnectAll()
	}
}

// HopCount indicates an instances of how many hops it takes to get to the specified GridPoint from the starting GridPoint.
type HopCount struct {
	Start       *GridPoint // The start of the path
	Destination *GridPoint // The end of the path
	HopCount    int        // How many hops it takes to get there (i.e. length of the path between the two points minus one)
}

// HopCounts returns the number of hops from the starting position to all other provided positions on the grid.
func (grid *Grid) HopCounts(from Vector, positions ...Vector) []HopCount {

	start := grid.NearestGridPoint(from)

	hopCounts := []HopCount{}

	for _, point := range positions {
		hc := HopCount{
			Start:       start,
			Destination: grid.NearestGridPoint(point),
		}
		hc.HopCount = len(start.PathTo(hc.Destination).GridPoints) - 1
		hopCounts = append(hopCounts, hc)
	}

	sort.Slice(hopCounts, func(i, j int) bool {
		return hopCounts[i].HopCount < hopCounts[j].HopCount
	})

	return hopCounts

}

// NearestPositionOnGrid returns the nearest world position on the Grid to the given world position.
// This position can be directly on a GridPoint, or on a connection between GridPoints.
func (grid *Grid) NearestPositionOnGrid(position Vector) Vector {

	nearestPoint := grid.NearestGridPoint(position)

	start := nearestPoint.WorldPosition()

	dist := math.MaxFloat64
	endPos := position

	for _, connection := range nearestPoint.Connections {
		// diff := connection.WorldPosition().Sub(pos)
		end := connection.WorldPosition()
		segment := end.Sub(start)
		newPos := position.Sub(start)
		t := newPos.Dot(segment) / segment.Dot(segment)
		if t > 1 {
			t = 1
		} else if t < 0 {
			t = 0
		}

		newPos.X = start.X + segment.X*t
		newPos.Y = start.Y + segment.Y*t
		newPos.Z = start.Z + segment.Z*t

		nd := newPos.DistanceSquared(position)
		if nd < dist {
			dist = nd
			endPos = newPos
		}

	}

	return endPos

}

// NearestGridPoint returns the nearest grid point to the given world position.
func (grid *Grid) NearestGridPoint(position Vector) *GridPoint {

	points := grid.Points()

	sort.Slice(points, func(i, j int) bool {
		return points[i].WorldPosition().Sub(position).MagnitudeSquared() < points[j].WorldPosition().Sub(position).MagnitudeSquared()
	})

	return points[0]

}

// FurthestGridPoint returns the furthest grid point to the given world position.
func (grid *Grid) FurthestGridPoint(position Vector) *GridPoint {

	points := grid.Points()

	sort.Slice(points, func(i, j int) bool {
		return points[i].WorldPosition().Sub(position).MagnitudeSquared() < points[j].WorldPosition().Sub(position).MagnitudeSquared()
	})

	return points[len(points)-1]

}

// RandomPoint returns a random grid point in the grid.
func (grid *Grid) RandomPoint() *GridPoint {
	gridPoints := grid.Points()
	return gridPoints[rand.Intn(len(gridPoints))]
}

// LastPoint returns the last point out of the Grid's GridPoints.
// If the Grid has no GridPoints, then it will return nil.
func (grid *Grid) LastPoint() *GridPoint {
	gridPoints := grid.Points()
	if len(gridPoints) == 0 {
		return nil
	}
	return gridPoints[len(gridPoints)-1]
}

// LastPoint returns the first point out of the Grid's GridPoints.
// If the Grid has no GridPoints, then it will return nil.
func (grid *Grid) FirstPoint() *GridPoint {
	gridPoints := grid.Points()
	if len(gridPoints) == 0 {
		return nil
	}
	return gridPoints[0]
}

// Combine combines the Grid with the other Grids provided. This reparents the other' Grid's GridPoints (and other children)
// to be under the calling Grid's. If two GridPoints share the same position, they will be merged together.
// After combining a Grid with others, the other Grids will automatically be unparented (as their GridPoints will
// have been absorbed).
func (grid *Grid) Combine(others ...*Grid) {

	for _, other := range others {

		if grid == other {
			continue
		}

		for _, p := range other.Children() {
			pos := p.WorldPosition()
			grid.AddChildren(p)
			p.SetWorldPositionVec(pos)
		}

		for _, p := range grid.Points() {

			for _, p2 := range grid.Points() {

				if p == p2 {
					continue
				}

				if p.WorldPosition().Equals(p2.WorldPosition()) {
					for _, connect := range p2.Connections {
						p.Connect(connect)
						connect.Disconnect(p2)
					}
					p2.Unparent()
				}
			}

		}

		other.Unparent()

	}

}

// Center returns the center point of the Grid, given the positions of its GridPoints.
func (grid *Grid) Center() Vector {
	pos := Vector{0, 0, 0, 0}
	points := grid.Points()
	for _, p := range points {
		pos = pos.Add(p.WorldPosition())
	}

	pos = pos.Divide(float64(len(points)))
	return pos
}

// Dimensions returns a Dimensions struct, indicating the overall "spread" of the GridPoints composing the Grid.
func (grid *Grid) Dimensions() Dimensions {
	gridPoints := grid.Points()
	points := make([]Vector, 0, len(gridPoints))
	for _, p := range gridPoints {
		points = append(points, p.WorldPosition())
	}
	return NewDimensionsFromPoints(points...)
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

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (grid *Grid) Index() int {
	if grid.parent != nil {
		for i, c := range grid.parent.Children() {
			if c == grid {
				return i
			}
		}
	}
	return -1
}

// Type returns the NodeType for this object.
func (grid *Grid) Type() NodeType {
	return NodeTypeGrid
}

// GridPath represents a sequence of grid points, used to traverse a path.
type GridPath struct {
	GridPoints []Vector
	Index      int
}

// Length returns the length of the overall path.
func (gp *GridPath) Length() float64 {

	dist := 0.0

	if len(gp.GridPoints) <= 1 {
		return 0
	}

	start := gp.GridPoints[0]

	for i := 1; i < len(gp.GridPoints); i++ {
		next := gp.GridPoints[i]
		dist += next.Sub(start).Magnitude()
		start = next
	}

	return dist

}

// Points returns the points of the GridPath in a slice.
func (gp *GridPath) Points() []Vector {
	points := append(make([]Vector, 0, len(gp.GridPoints)), gp.GridPoints...)
	return points
}

func (gp *GridPath) isClosed() bool {
	return false
}

func (gp *GridPath) DebugDraw(screen *ebiten.Image, camera *Camera, color *Color) {

	points := gp.Points()
	for i := 0; i < len(points)-1; i++ {
		p1 := camera.WorldToScreen(points[i])
		p2 := camera.WorldToScreen(points[i+1])
		ebitenutil.DrawLine(screen, p1.X, p1.Y, p2.X, p2.Y, color.ToRGBA64())
		ebitenutil.DrawCircle(screen, p1.X, p1.Y, 8, color.ToRGBA64())
		if i == len(points)-2 {
			ebitenutil.DrawCircle(screen, p2.X, p2.Y, 8, color.ToRGBA64())
		}
	}

}
