package tetra3d

import (
	"math"
	"math/rand"
	"slices"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// GridPoint represents a point on a Grid, used for pathfinding or connecting points in space.
// GridPoints are parented to a Grid and the connections are created seperate from their positions,
// which means you can move GridPoints freely after creation. Note that GridPoints consider themselves
// to be in the same Grid only if they have the same direct parent (being the Grid).
type GridPoint struct {
	*Node
	passable    bool
	cost        float32
	Connections []*GridPoint
}

// NewGridPoint creates a new GridPoint.
func NewGridPoint(name string) *GridPoint {
	return NewGridPointFromNode(NewNode(name))
}

// NewGridPoint creates a new GridPoint from an existing Node.
func NewGridPointFromNode(node *Node) *GridPoint {
	gridPoint := &GridPoint{
		Node:        node,
		Connections: []*GridPoint{},
		passable:    true,
	}
	gridPoint.owner = gridPoint
	return gridPoint
}

// Clone clones the given GridPoint.
func (point *GridPoint) Clone() INode {
	newPoint := &GridPoint{}
	newPoint.Node = point.Node.clone(point).(*Node)
	newPoint.Connections = append([]*GridPoint{}, point.Connections...)

	newPoint.passable = point.passable
	newPoint.cost = point.cost

	if runCallbacks && newPoint.Callbacks().OnClone != nil {
		newPoint.Callbacks().OnClone(newPoint)
	}

	return newPoint
}

func (point *GridPoint) Cost() float32 {
	return point.cost
}

func (point *GridPoint) SetCost(cost float32) {
	point.cost = cost
}

func (point *GridPoint) Passable() bool {
	return point.passable
}

func (point *GridPoint) SetPassable(passable bool) {
	point.passable = passable
}

// IsConnected returns if the provided GridPoint is connected to the given other GridPoint.
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

// An internal struct to assist with pathfinding on a Grid.
type pathNode struct {
	gp                  *GridPoint
	next                *pathNode
	runningCostDistance float32
}

// PathTo creates a path going from the GridPoint to the given goal GridPoint.
// It'll be the shortest-possible route out of all possible paths to the goal,
// taking into account both the overall cumulative length (in world units) and costs of each GridPoint on the path.
// If a path is not possible from the starting point to the end point, then PathTo will return nil.

func (point *GridPoint) PathTo(goal *GridPoint) *GridPath {

	if point.parent.Type() != NodeTypeGrid || !point.IsOnSameGrid(goal) || !goal.passable || !point.passable {
		return nil
	}

	if point == goal {
		return &GridPath{
			gridPoints: []*GridPoint{point},
		}
	}

	checked := []*GridPoint{goal}
	toCheck := []*pathNode{&pathNode{gp: goal}}

	// If we've checked a grid point already, then we don't have to check it again
	alreadyChecked := func(node *pathNode, conn *GridPoint) bool {
		for _, c := range checked {
			if c == conn {
				return true
			}
		}
		return false
	}

	// Represents a linked series of nodes, representing the final path.
	var foundPath *pathNode

	for {

		if len(toCheck) == 0 {
			break
		}

		next := toCheck[0]
		toCheck = toCheck[1:]

		if next.gp == point {
			// Managed to find the start; this should be the most ideal path.
			foundPath = next
			break
		}

		// We loop through connections and insert them to the toCheck list with the position depending on their overall running cost.
		// The longer the path and higher the cost, the lower priority (later in the list) it is.
		for _, conn := range next.gp.Connections {

			if conn.passable && !alreadyChecked(next, conn) {

				node := &pathNode{
					gp:                  conn,
					next:                next,
					runningCostDistance: conn.cost + conn.DistanceSquaredTo(next.gp) + next.runningCostDistance,
				}

				insertIndex := len(toCheck)

				for i, c := range toCheck {
					if node.runningCostDistance < c.runningCostDistance {
						insertIndex = i
						break
					}
				}

				toCheck = slices.Insert(toCheck, insertIndex, node)
				checked = append(checked, conn)

			}

		}

	}

	if foundPath == nil {
		return nil
	}

	path := &GridPath{
		gridPoints: []*GridPoint{},
	}

	for foundPath != nil {
		path.gridPoints = append(path.gridPoints, foundPath.gp)
		foundPath = foundPath.next
	}

	return path

}

func (point *GridPoint) insertIntoSlice(slice []*GridPoint, index int, value *GridPoint) []*GridPoint {
	if len(slice) == index { // nil or empty slice or after last element
		return append(slice, value)
	}
	slice = append(slice[:index+1], slice[index:]...) // index < len(a)
	slice[index] = value
	return slice
}

////////////

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
	g := &Grid{Node: NewNode(name)}
	g.owner = g
	return g
}

// Clone creates a clone of this GridPoint.
func (grid *Grid) Clone() INode {

	newGrid := &Grid{}
	newGrid.Node = grid.Node.clone(newGrid).(*Node)

	for _, c := range newGrid.Points() {
		c.Connections = []*GridPoint{}
	}

	for _, c := range grid.Points() {

		start := newGrid.ClosestGridPoint(c.WorldPosition())
		for _, connect := range c.Connections {
			end := newGrid.ClosestGridPoint(connect.WorldPosition())
			start.Connect(end)
		}

	}

	if runCallbacks && newGrid.Callbacks().OnClone != nil {
		newGrid.Callbacks().OnClone(newGrid)
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

func (grid *Grid) MergeDuplicatePoints(margin float32) {
	grid.ForEachPoint(func(point *GridPoint) {

		grid.ForEachPoint(func(point2 *GridPoint) {

			if point == point2 {
				return
			}

			if point.WorldPosition().DistanceSquaredTo(point2.WorldPosition()) < margin {
				for _, c := range point2.Connections {
					point.Connections = append(point.Connections, c)

					// nc := c.Clone()
					// point.Connections = append(point.Connections, c)

					// nc2 := c.To.Connection(point2).Clone()
					// nc2.To = point
				}
				point2.DisconnectAll()
				point2.Unparent()
			}

		})

	})
}

// ClosestPositionOnGrid returns the nearest world position on the Grid to the given world position.
// This position can be directly on a GridPoint, or on a connection between GridPoints.
func (grid *Grid) ClosestPositionOnGrid(position Vector3) Vector3 {

	nearestPoint := grid.ClosestGridPoint(position)

	start := nearestPoint.WorldPosition()

	dist := float32(math.MaxFloat32)
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

		nd := newPos.DistanceSquaredTo(position)
		if nd < dist {
			dist = nd
			endPos = newPos
		}

	}

	return endPos

}

// ClosestGridPoint returns the nearest grid point to the given world position.
func (grid *Grid) ClosestGridPoint(position Vector3) *GridPoint {

	points := grid.Points()

	sort.Slice(points, func(i, j int) bool {
		return points[i].WorldPosition().Sub(position).MagnitudeSquared() < points[j].WorldPosition().Sub(position).MagnitudeSquared()
	})

	return points[0]

}

// FurthestGridPoint returns the furthest grid point to the given world position.
func (grid *Grid) FurthestGridPoint(position Vector3) *GridPoint {

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

// FirstPoint returns the first point out of the Grid's GridPoints.
// If the Grid has no GridPoints, then it will return nil.
func (grid *Grid) FirstPoint() *GridPoint {
	gridPoints := grid.Points()
	if len(gridPoints) == 0 {
		return nil
	}
	return gridPoints[0]
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

// Combine combines the Grid with the other Grids provided. This reparents the other' Grid's GridPoints (and other children)
// to be under the calling Grid's. If two GridPoints share the same position, they will be merged together.
// After combining a Grid with others, the other Grids will automatically be unparented from the scene nodegraph tree
// (as their GridPoints will have been absorbed).
func (grid *Grid) Combine(others ...*Grid) {

	for _, other := range others {

		if grid == other {
			continue
		}

		other.ForEachChild(false, func(p INode, index int) bool {
			pos := p.WorldPosition()
			grid.AddChildren(p)
			p.SetWorldPositionVec(pos)
			return true
		})

		// for _, p := range grid.Points() {

		// 	for _, p2 := range grid.Points() {

		// 		if p == p2 {
		// 			continue
		// 		}

		// 		if p.WorldPosition().Equals(p2.WorldPosition()) {
		// 			for _, connect := range p2.Connections {
		// 				p.Connect(connect.To)
		// 				connect.To.Disconnect(p2)
		// 			}
		// 			p2.Unparent()
		// 		}
		// 	}

		// }

		grid.MergeDuplicatePoints(0.001)

		other.Unparent()

	}

}

// Center returns the center point of the Grid, given the positions of its GridPoints.
func (grid *Grid) Center() Vector3 {
	pos := Vector3{0, 0, 0}
	points := grid.Points()
	for _, p := range points {
		pos = pos.Add(p.WorldPosition())
	}

	pos = pos.Divide(float32(len(points)))
	return pos
}

// Dimensions returns a Dimensions struct, indicating the overall "spread" of the GridPoints composing the Grid.
func (grid *Grid) Dimensions() Dimensions {
	gridPoints := grid.Points()
	points := make([]Vector3, 0, len(gridPoints))
	for _, p := range gridPoints {
		points = append(points, p.WorldPosition())
	}
	return NewDimensionsFromPoints(points...)
}

////////

// Type returns the NodeType for this object.
func (grid *Grid) Type() NodeType {
	return NodeTypeGrid
}

// GridPath represents a sequence of grid points, used to traverse a path.
// GridPath implements IPath.
type GridPath struct {
	gridPoints []*GridPoint
}

// Returns the length of the overall path.
func (gp *GridPath) Length() float32 {

	dist := float32(0)

	if len(gp.gridPoints) <= 1 {
		return 0
	}

	start := gp.gridPoints[0]

	for i := 1; i < len(gp.gridPoints); i++ {
		next := gp.gridPoints[i]
		dist += next.WorldPosition().Sub(start.WorldPosition()).Magnitude()
		start = next
	}

	return dist

}

// Returns the grid points of the GridPath in a slice.
func (gp *GridPath) GridPoints() []*GridPoint {
	points := make([]*GridPoint, 0, len(gp.gridPoints))
	for _, gp := range gp.gridPoints {
		points = append(points, gp)
	}
	return points
}

// Returns the positions of the GridPoints of the GridPath in a slice.
func (gp *GridPath) Points() []Vector3 {
	points := make([]Vector3, 0, len(gp.gridPoints))
	for _, gp := range gp.gridPoints {
		points = append(points, gp.WorldPosition())
	}
	return points
}

// HopCount returns the number of hops in the path (i.e. the number of nodes - 1).
func (gp *GridPath) HopCount() int {
	return len(gp.gridPoints) - 1
}

func (gp *GridPath) isClosed() bool {
	return false
}

func (gp *GridPath) DebugDraw(screen *ebiten.Image, camera *Camera, color Color4) {

	points := gp.Points()
	for i := 0; i < len(points)-1; i++ {
		p1 := camera.WorldToScreenPixels(points[i])
		p2 := camera.WorldToScreenPixels(points[i+1])
		vector.StrokeLine(screen, p1.X, p1.Y, p2.X, p2.Y, 1, color.ToRGBA64(), false)
		vector.StrokeCircle(screen, p1.X, p1.Y, 8, 1, color.ToRGBA64(), false)
		if i == len(points)-2 {
			vector.StrokeCircle(screen, p2.X, p2.Y, 8, 1, color.ToRGBA64(), false)
		}
	}

}

var _ IPath = &GridPath{} // Sanity check to ensure GridPaths implement paths
