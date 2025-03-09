package tetra3d

import (
	"strconv"

	"github.com/solarlune/tetra3d/math32"
)

// PathStepper is an object that steps through points in a set path.
// It returns the position of the current node and has the ability to go to the next or previous node in the path.
type PathStepper struct {
	path   IPath
	points []Vector3
	Index  int
}

// NewPathStepper returns a new PathStepper object.
func NewPathStepper(path IPath) *PathStepper {
	ps := &PathStepper{}
	ps.SetPath(path)
	return ps
}

// SetPath sets the path of the PathStepper; this should be called whenever the path updates.
// Doing this will reset the PathStepper's index to 0.
func (ps *PathStepper) SetPath(path IPath) {
	ps.path = path
	if path != nil {
		ps.points = ps.path.Points()
	}
	ps.SetIndexToStart()
}

// Path returns the path used by the PathStepper.
func (ps *PathStepper) Path() IPath {
	return ps.path
}

// SetIndexToStart resets the PathStepper to point to the beginning of the path.
func (ps *PathStepper) SetIndexToStart() {
	ps.Index = 0
}

// SetIndexToEnd sets the PathStepper to the point at the end of the path.
func (ps *PathStepper) SetIndexToEnd() {
	ps.Index = len(ps.points) - 1
}

// CurrentWorldPosition returns the current node's world position for the PathStepper.
func (ps *PathStepper) CurrentWorldPosition() Vector3 {
	return ps.points[ps.Index]
}

// Next steps to the next point in the Path for the PathStepper.
// If the PathStepper is at the end of the path, then it will loop through the path again.
func (ps *PathStepper) Next() {

	ps.Index++

	if ps.Index > len(ps.points)-1 {
		ps.Index = 0
	}

}

// Next steps to the previous point in the Path for the PathStepper.
// If the PathStepper is at the beginning of the path, then it will loop through the path again.
func (ps *PathStepper) Prev() {

	ps.Index--

	if ps.Index < 0 {
		ps.Index = len(ps.points) - 1
	}

}

// AtEnd returns if the PathStepper is at the end of its path.
func (ps *PathStepper) AtEnd() bool {
	return ps.Index == len(ps.points)-1
}

// AtStart returns if the PathStepper is at the start of its path.
func (ps *PathStepper) AtStart() bool {
	return ps.Index == 0
}

// ProgressToWorldPosition returns a position on the PathStepper's current Path, if given a percentage value that ranges from 0 to 1.
// The percentage is weighted for distance, not for number of points.
// For example, say you had a path comprised of four points: {0, 0, 0}, {9, 0, 0}, {9.5, 0, 0}, and {10, 0, 0}. If you called PathStepper.ProgressToWorldPosition(0.9), you'd get {9, 0, 0} (90% of the way through the path).
// If the PathStepper has a nil Path or its Path has no points, this function returns an empty Vector.
func (ps *PathStepper) ProgressToWorldPosition(perc float32) Vector3 {

	if ps.path == nil || len(ps.points) == 0 {
		return Vector3{}
	}

	if len(ps.points) == 1 {
		return ps.points[0]
	}

	perc = math32.Clamp(perc, 0, 1)

	points := append(make([]Vector3, 0, len(ps.points)), ps.points...)

	if ps.path.isClosed() {
		points = append(points, ps.points[0])
	}

	totalDistance := ps.path.Length()
	d := perc

	worldPos := points[len(points)-1]

	for i := 0; i < len(points)-1; i++ {
		segment := points[i+1].Sub(points[i])
		segmentDistance := segment.Magnitude() / totalDistance
		if d > segmentDistance {
			d -= segmentDistance
		} else {
			worldPos = points[i].Add(segment.Scale(d / segmentDistance))
			break
		}
	}

	return worldPos

}

// IPath represents an object that implements a path, returning points of a given length across
// a distance.
type IPath interface {
	// Length returns the length of the overall path.
	Length() float32
	// Points returns the points of the IPath in a slice.
	Points() []Vector3
	HopCount() int // HopCount returns the number of hops in the path.
	isClosed() bool
}

// A Path represents a Node that represents a sequential path. All children of the Path are considered its points, in order.
type Path struct {
	*Node
	Closed    bool // Closed indicates if a Path is closed (and so going to the end will return to the start) or not.
	PathIndex int  // Index of the path points; used to reset the path when navigating through Path.Next().
}

// NewPath returns a new Path object. A Path is a Node whose children represent points on a path. A Path can be stepped through
// spatially using a Navigator. The passed point vectors will become Nodes, children of the Path.
func NewPath(name string, points ...Vector3) *Path {
	path := &Path{
		Node: NewNode(name),
	}
	path.owner = path
	for i, point := range points {
		pointNode := NewNode("_point." + strconv.Itoa(i))
		pointNode.SetLocalPositionVec(point)
		path.AddChildren(pointNode)
	}
	return path
}

// Clone creates a clone of the Path and its points.
func (path *Path) Clone() INode {

	clone := NewPath(path.name)
	clone.Closed = path.Closed

	clone.Node = path.Node.clone(clone).(*Node)

	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}

	return clone

}

// Length returns the total distance that a Path covers by stepping through all of the children under the Path.
func (path *Path) Length() float32 {
	dist := float32(0.0)
	points := path.Children()

	if len(points) <= 1 {
		return 0
	}

	start := points[0].WorldPosition()
	for i := 1; i < len(points); i++ {
		next := points[i].WorldPosition()
		dist += next.Sub(start).Magnitude()
		start = next
	}

	if path.Closed {
		dist += points[len(points)-1].WorldPosition().Sub(points[0].WorldPosition()).Magnitude()
	}

	return dist
}

// Points returns the Vector world positions of each point in the Path.
func (path *Path) Points() []Vector3 {
	points := make([]Vector3, 0, len(path.Children()))
	for _, c := range path.children {
		points = append(points, c.WorldPosition())
	}
	return points
}

// HopCount returns the number of hops in the path (i.e. number of nodes - 1).
func (path *Path) HopCount() int {
	return len(path.Children()) - 1
}

func (path *Path) isClosed() bool {
	return path.Closed
}

// Next returns the next point in the path as an iterator - if the boolean value is true, you have reached the end of the path.
// Useful if you don't want to use a Navigator to navigate through it.
func (path *Path) Next() (Vector3, bool) {
	points := path.Points()
	path.PathIndex++
	atEnd := false
	if path.PathIndex >= len(points) {
		path.PathIndex = len(points) - 1
		atEnd = true
	}
	return points[path.PathIndex], atEnd
}

/////

// Type returns the NodeType for this object.
func (path *Path) Type() NodeType {
	return NodeTypePath
}

var _ IPath = &Path{} // Sanity check to ensure Path implements IPath.
