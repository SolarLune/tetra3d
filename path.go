package tetra3d

import (
	"strconv"
)

type IPath interface {
	Distance() float64
	Points() []Vector
	isClosed() bool
}

// Navigator follows a Path or GridPath, stepping through it spatially to completion. You can use this to make objects that follow a path's positions in
// space.
type Navigator struct {
	Percentage float64    // Percentage is how far along the Path the Navigator is, ranging from 0 to 1.
	FinishMode FinishMode // FinishMode indicates what should happen when the Navigator finishes running across the given Path.
	Direction  int        // Direction indicates the playback direction.
	Path       IPath      // A Reference to the Path
	// OnFinish is a function that gets called when the Navigator completes moving across the provided Path.
	// If the FinishMode is set to FinishModePingPong, then this triggers when the animation plays once both forwards and backwards.
	OnFinish func()
	//OnTouchPoint is a function that gets called when the Navigator passes by a path point during an Advance().
	OnTouchPoint func()
	worldPos     Vector
	onIndex      int
	prevIndex    int
	finished     bool
}

// NewNavigator returns a new Navigator object. path is an object that fulfills the IPath interface (so a Path or a GridPath).
func NewNavigator(path IPath) *Navigator {
	navigator := &Navigator{
		FinishMode: FinishModeLoop,
		Path:       path,
		Direction:  1,
	}

	if path != nil {

		if points := path.Points(); len(points) > 0 {
			navigator.worldPos = points[0] // Just to start
		}

	}

	return navigator
}

// AdvancePercentage advances the Navigator by a certain percentage of the path. This being a percentage, the larger the path, the further
// in space a percentage advances the Navigator.
func (navigator *Navigator) AdvancePercentage(percentage float64) {

	navigator.finished = false

	if !navigator.HasPath() || len(navigator.Path.Points()) <= 1 {
		navigator.finished = true
		return
	}

	navigator.Percentage += percentage * float64(navigator.Direction)

	points := navigator.Path.Points()
	totalDistance := navigator.Path.Distance()
	d := navigator.Percentage

	if len(points) == 0 {
		navigator.worldPos = NewVectorZero()
	}

	if len(points) == 1 {
		navigator.worldPos = points[0]
	}

	navigator.worldPos = points[len(points)-1]

	for i := 0; i < len(points)-1; i++ {
		navigator.onIndex = i
		segment := points[i+1].Sub(points[i])
		segmentDistance := segment.Magnitude() / totalDistance
		if d > segmentDistance {
			d -= segmentDistance
		} else {
			navigator.worldPos = points[i].Add(segment.Scale(d / segmentDistance))
			break
		}
	}

	if navigator.prevIndex != navigator.onIndex && navigator.OnTouchPoint != nil {
		navigator.OnTouchPoint()
	}

	navigator.prevIndex = navigator.onIndex

	if navigator.FinishMode == FinishModeLoop {

		looped := false

		for navigator.Percentage > 1 {
			navigator.Percentage--
			looped = true
		}
		for navigator.Percentage < 0 {
			navigator.Percentage++
			looped = true
		}

		if looped {
			navigator.finished = true
			if navigator.OnFinish != nil {
				navigator.OnFinish()
			}
		}

	} else {

		finish := false

		if navigator.Percentage > 1 {
			navigator.Percentage = 1
			finish = true
		} else if navigator.Percentage < 0 {
			navigator.Percentage = 0
			finish = true
		}

		if finish {

			navigator.finished = true

			if navigator.FinishMode == FinishModePingPong {
				if navigator.Direction < 0 && navigator.OnFinish != nil {
					navigator.OnFinish()
				}
				navigator.Direction *= -1
			} else if navigator.OnFinish != nil {
				navigator.OnFinish()
			}

		}

	}

}

// AdvanceDistance advances the Navigator by a certain distance in absolute movement units on the path.
func (navigator *Navigator) AdvanceDistance(distance float64) {
	navigator.AdvancePercentage(distance / navigator.Path.Distance())
}

// WorldPosition returns the position of the Navigator in world space.
func (navigator *Navigator) WorldPosition() Vector {
	return navigator.worldPos
}

// TouchingNode returns if a Navigator is within a distance of margin units to a point in a Grid or Path. If it is not within that distance, TouchingNode returns nil.
func (navigator *Navigator) TouchingNode(margin float64) (point Vector, touching bool) {

	wp := navigator.WorldPosition()
	for _, p := range navigator.Path.Points() {
		if p.Sub(wp).Magnitude() <= margin {
			return p, true
		}
	}

	return Vector{}, false

}

// Index returns the index of the point / child that the Navigator is on.
func (navigator *Navigator) Index() int {
	return navigator.onIndex
}

// NextIndex returns the index of the next point / child that the Navigator will be heading towards.
func (navigator *Navigator) NextIndex() int {
	nextIndex := navigator.onIndex + navigator.Direction
	pathLen := len(navigator.Path.Points())
	if nextIndex >= pathLen {
		nextIndex -= pathLen
	} else if nextIndex < 0 {
		nextIndex += pathLen
	}
	return nextIndex
}

// Clone returns a clone of this Navigator.
func (navigator *Navigator) Clone() *Navigator {
	newNavigator := NewNavigator(navigator.Path)
	newNavigator.FinishMode = navigator.FinishMode
	newNavigator.Percentage = navigator.Percentage
	newNavigator.Direction = navigator.Direction
	newNavigator.onIndex = navigator.onIndex
	newNavigator.prevIndex = navigator.prevIndex
	newNavigator.worldPos = navigator.worldPos
	newNavigator.OnFinish = navigator.OnFinish
	newNavigator.OnTouchPoint = navigator.OnTouchPoint
	return newNavigator
}

func (navigator *Navigator) HasPath() bool {
	return navigator.Path != nil && len(navigator.Path.Points()) > 0
}

func (navigator *Navigator) SetPath(path IPath) {
	if path != navigator.Path {
		navigator.Path = path
		navigator.Percentage = 0
		navigator.finished = false
		if path != nil {
			if points := path.Points(); len(points) > 0 {
				navigator.worldPos = points[0]
			}
		}
	}
}

func (navigator *Navigator) Finished() bool {
	return navigator.finished
}

// A Path represents a Node that represents a sequential path. All children of the Path are considered its points, in order.
type Path struct {
	*Node
	Closed bool // Closed indicates if a Path is closed (and so going to the end will return to the start) or not.
}

// NewPath returns a new Path object. A Path is a Node whose children represent points on a path. A Path can be stepped through
// spatially using a Navigator. The passed point vectors will become Nodes, children of the Path.
func NewPath(name string, points ...Vector) *Path {
	path := &Path{
		Node: NewNode(name),
	}
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

	clone.Node = path.Node.Clone().(*Node)
	for _, child := range path.children {
		child.setParent(path)
	}

	return clone

}

// Distance returns the total distance that a Path covers by stepping through all of the children under the Path.
func (path *Path) Distance() float64 {
	dist := 0.0
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

func (path *Path) Points() []Vector {
	points := make([]Vector, 0, len(path.Children()))
	for _, c := range path.children {
		points = append(points, c.WorldPosition())
	}
	if path.Closed {
		points = append(points, path.children[0].WorldPosition())
	}
	return points
}

func (path *Path) isClosed() bool {
	return path.Closed
}

/////

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (path *Path) AddChildren(children ...INode) {
	path.addChildren(path, children...)
}

// Unparent unparents the Path from its parent, removing it from the scenegraph.
func (path *Path) Unparent() {
	if path.parent != nil {
		path.parent.RemoveChildren(path)
	}
}

// Type returns the NodeType for this object.
func (path *Path) Type() NodeType {
	return NodeTypePath
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (path *Path) Index() int {
	if path.parent != nil {
		for i, c := range path.parent.Children() {
			if c == path {
				return i
			}
		}
	}
	return -1
}
