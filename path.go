package tetra3d

import (
	"strconv"

	"github.com/kvartborg/vector"
)

// PathFollower follows a Path, stepping through the Path spatially to completion. You can use this to make objects that follow a path's positions in
// space.
type PathFollower struct {
	Percentage float64 // Percentage is how far along the Path the PathFollower is, ranging from 0 to 1.
	FinishMode int     // FinishMode indicates what should happen when the PathFollower finishes running across the given Path.
	Direction  int     // Direction indicates the playback direction.
	Path       *Path   // A Reference to the Path
	OnFinish   func()
	// OnFinish is a function that gets called when the PathFollower completes moving across the provided Path.
	// If the FinishMode is set to FinishModePingPong, then this triggers when the animation plays once both forwards and backwards.
	onIndex int
}

// NewPathFollower returns a new PathFollower object.
func NewPathFollower(path *Path) *PathFollower {
	follower := &PathFollower{
		FinishMode: FinishModeLoop,
		Path:       path,
		Direction:  1,
	}
	return follower
}

// AdvancePercentage advances the PathFollower by a certain percentage of the path. This being a percentage, the larger the path, the further
// in space a percentage advances the PathFollower.
func (follower *PathFollower) AdvancePercentage(percentage float64) {
	follower.Percentage += percentage * float64(follower.Direction)

	if follower.FinishMode == FinishModeLoop {

		looped := false

		for follower.Percentage > 1 {
			follower.Percentage--
			looped = true
		}
		for follower.Percentage < 0 {
			follower.Percentage++
			looped = true
		}

		if looped && follower.OnFinish != nil {
			follower.OnFinish()
		}

	} else {

		finish := false

		if follower.Percentage > 1 {
			follower.Percentage = 1
			finish = true
		} else if follower.Percentage < 0 {
			follower.Percentage = 0
			finish = true
		}

		if finish {

			if follower.FinishMode == FinishModePingPong {
				if follower.Direction < 0 && follower.OnFinish != nil {
					follower.OnFinish()
				}
				follower.Direction *= -1
			} else if follower.OnFinish != nil {
				follower.OnFinish()
			}

		}

	}

}

// AdvanceDistance advances the PathFollower by a certain distance in absolute movement units on the path.
func (follower *PathFollower) AdvanceDistance(distance float64) {
	follower.AdvancePercentage(distance / follower.Path.Distance())
}

// WorldPosition returns the position of the PathFollower in world space.
func (follower *PathFollower) WorldPosition() vector.Vector {

	totalDistance := follower.Path.Distance()
	d := follower.Percentage

	points := follower.Path.Children()

	if follower.Path.Closed {
		points = append(points, points[0])
	}

	for i := 0; i < len(points)-1; i++ {
		segment := points[i+1].WorldPosition().Sub(points[i].WorldPosition())
		segmentDistance := segment.Magnitude() / totalDistance
		if d > segmentDistance {
			d -= segmentDistance
		} else {
			follower.onIndex = i
			return points[i].WorldPosition().Add(segment.Scale(d / segmentDistance))
		}
	}

	return points[len(points)-1].WorldPosition()
}

// Index returns the index of the point / child that the PathFollower is on.
func (follower *PathFollower) Index() int {
	return follower.onIndex
}

// Clone returns a clone of this PathFollower.
func (follower *PathFollower) Clone() *PathFollower {
	newFollower := NewPathFollower(follower.Path)
	newFollower.FinishMode = follower.FinishMode
	newFollower.Percentage = follower.Percentage
	newFollower.Direction = follower.Direction
	newFollower.onIndex = follower.onIndex
	newFollower.OnFinish = follower.OnFinish
	return newFollower
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
		pointNode.SetLocalPosition(point)
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
