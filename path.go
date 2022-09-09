package tetra3d

import (
	"strconv"

	"github.com/kvartborg/vector"
)

type IPath interface {
	Distance() float64
	Points() []vector.Vector
	isClosed() bool
}

// Navigator follows a Path or GridPath, stepping through it spatially to completion. You can use this to make objects that follow a path's positions in
// space.
type Navigator struct {
	Percentage float64    // Percentage is how far along the Path the PathFollower is, ranging from 0 to 1.
	FinishMode FinishMode // FinishMode indicates what should happen when the PathFollower finishes running across the given Path.
	Direction  int        // Direction indicates the playback direction.
	Path       IPath      // A Reference to the Path
	OnFinish   func()
	// OnFinish is a function that gets called when the PathFollower completes moving across the provided Path.
	// If the FinishMode is set to FinishModePingPong, then this triggers when the animation plays once both forwards and backwards.
	onIndex  int
	finished bool
}

// NewNavigator returns a new PathFollower object.
func NewNavigator(path IPath) *Navigator {
	navigator := &Navigator{
		FinishMode: FinishModeLoop,
		Path:       path,
		Direction:  1,
	}
	return navigator
}

// AdvancePercentage advances the PathFollower by a certain percentage of the path. This being a percentage, the larger the path, the further
// in space a percentage advances the PathFollower.
func (navigator *Navigator) AdvancePercentage(percentage float64) {

	navigator.finished = false

	if len(navigator.Path.Points()) <= 1 {
		navigator.finished = true
		return
	}

	navigator.Percentage += percentage * float64(navigator.Direction)

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

		if looped && navigator.OnFinish != nil {
			navigator.OnFinish()
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

// AdvanceDistance advances the PathFollower by a certain distance in absolute movement units on the path.
func (navigator *Navigator) AdvanceDistance(distance float64) {
	navigator.AdvancePercentage(distance / navigator.Path.Distance())
}

// WorldPosition returns the position of the PathFollower in world space.
func (navigator *Navigator) WorldPosition() vector.Vector {

	totalDistance := navigator.Path.Distance()
	d := navigator.Percentage

	points := navigator.Path.Points()

	if len(points) == 0 {
		return nil
	}

	if len(points) == 1 {
		return points[0]
	}

	if navigator.Path.isClosed() {
		points = append(points, points[0])
	}

	for i := 0; i < len(points)-1; i++ {
		segment := points[i+1].Sub(points[i])
		segmentDistance := segment.Magnitude() / totalDistance
		if d > segmentDistance {
			d -= segmentDistance
		} else {
			navigator.onIndex = i
			return points[i].Add(segment.Scale(d / segmentDistance))
		}
	}

	return points[len(points)-1]
}

// Index returns the index of the point / child that the PathFollower is on.
func (navigator *Navigator) Index() int {
	return navigator.onIndex
}

// Clone returns a clone of this PathFollower.
func (navigator *Navigator) Clone() *Navigator {
	newNavigator := NewNavigator(navigator.Path)
	newNavigator.FinishMode = navigator.FinishMode
	newNavigator.Percentage = navigator.Percentage
	newNavigator.Direction = navigator.Direction
	newNavigator.onIndex = navigator.onIndex
	newNavigator.OnFinish = navigator.OnFinish
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
// spatially using a PathFollower. The passed point vectors will become Nodes, children of the Path.
func NewPath(name string, points ...vector.Vector) *Path {
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

func (path *Path) Points() []vector.Vector {
	points := make([]vector.Vector, 0, len(path.Children()))
	for _, c := range path.children {
		points = append(points, c.WorldPosition())
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
