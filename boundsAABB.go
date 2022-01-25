package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

// BoundingAABB represents a 3D AABB (Axis-Aligned Bounding Box), a 3D cube of varying width, height, and depth that cannot rotate.
// The primary purpose of a BoundingAABB is, like the other Bounding* Nodes, to perform intersection testing between itself and other
// BoundingObject Nodes.
type BoundingAABB struct {
	*Node
	internalSize vector.Vector
	Size         vector.Vector
}

// NewBoundingAABB returns a new BoundingAABB Node.
func NewBoundingAABB(name string, width, height, depth float64) *BoundingAABB {
	min := 0.0001
	if width <= 0 {
		width = min
	}
	if height <= 0 {
		height = min
	}
	if depth <= 0 {
		depth = min
	}
	bounds := &BoundingAABB{
		Node:         NewNode(name),
		internalSize: vector.Vector{width, height, depth},
	}
	bounds.onTransformUpdate = bounds.UpdateSize
	return bounds
}

// UpdateSize updates the BoundingAABB's external Size property to reflect its size after reposition, rotation, or resizing. \
// This is be called automatically internally as necessary after the node's transform is updated.
func (box *BoundingAABB) UpdateSize() {

	_, s, r := box.Node.Transform().Decompose()

	corners := [][]float64{
		{1, 1, 1},
		{1, -1, 1},
		{-1, 1, 1},
		{-1, -1, 1},
		{1, 1, -1},
		{1, -1, -1},
		{-1, 1, -1},
		{-1, -1, -1},
	}

	box.Size = vector.Vector{0, 0, 0}

	for _, c := range corners {
		cs := vector.Vector{
			box.internalSize[0] * c[0],
			box.internalSize[1] * c[1],
			box.internalSize[2] * c[2],
		}
		cs = r.MultVec(cs)
		box.Size[0] = math.Max(box.Size[0], math.Abs(cs[0]))
		box.Size[1] = math.Max(box.Size[1], math.Abs(cs[1]))
		box.Size[2] = math.Max(box.Size[2], math.Abs(cs[2]))
	}

	box.Size[0] *= s[0]
	box.Size[1] *= s[1]
	box.Size[2] *= s[2]

}

// SetDimensions sets the BoundingAABB's internal dimensions (prior to resizing or rotating the Node).
func (box *BoundingAABB) SetDimensions(newWidth, newHeight, newDepth float64) {

	min := 0.00001
	if newWidth <= 0 {
		newWidth = min
	}
	if newHeight <= 0 {
		newHeight = min
	}
	if newDepth <= 0 {
		newDepth = min
	}

	if box.internalSize[0] != newWidth || box.internalSize[1] != newHeight || box.internalSize[2] != newDepth {
		box.internalSize[0] = newWidth
		box.internalSize[1] = newHeight
		box.internalSize[2] = newDepth
		box.UpdateSize()
	}

}

// Clone returns a new BoundingAABB.
func (box *BoundingAABB) Clone() INode {
	clone := NewBoundingAABB(box.name, box.internalSize[0], box.internalSize[1], box.internalSize[2])
	clone.Node = box.Node.Clone().(*Node)
	return clone
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (box *BoundingAABB) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	box.addChildren(box, children...)
}

// ClosestPoint returns the closest point, to the point given, on the inside or surface of the BoundingAABB.
func (box *BoundingAABB) ClosestPoint(point vector.Vector) vector.Vector {
	out := point.Clone()
	pos := box.WorldPosition()

	half := box.Size.Scale(0.5)

	if out[0] > pos[0]+half[0] {
		out[0] = pos[0] + half[0]
	} else if out[0] < pos[0]-half[0] {
		out[0] = pos[0] - half[0]
	}

	if out[1] > pos[1]+half[1] {
		out[1] = pos[1] + half[1]
	} else if out[1] < pos[1]-half[1] {
		out[1] = pos[1] - half[1]
	}

	if out[2] > pos[2]+half[2] {
		out[2] = pos[2] + half[2]
	} else if out[2] < pos[2]-half[2] {
		out[2] = pos[2] - half[2]
	}

	return out
}

// Intersecting returns true if the BoundingAABB is intersecting with another BoundingObject.
func (box *BoundingAABB) Intersecting(other BoundingObject) bool {
	return box.Intersection(other) != nil
}

// Intersection returns the intersection between the BoundingAABB and the other BoundingObject. If
// there is no intersection, the function returns nil. (Note that BoundingAABB > BoundingTriangles collision
// is buggy at the moment.)
func (box *BoundingAABB) Intersection(other BoundingObject) *IntersectionResult {

	if other == box {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingAABB:
		return btAABBAABB(box, otherBounds)

	case *BoundingSphere:
		intersection := btSphereAABB(otherBounds, box)
		if intersection != nil {
			intersection.MTV = intersection.MTV.Invert()
		}
		return intersection

	case *BoundingTriangles:
		return btAABBTriangles(box, otherBounds)

	case *BoundingCapsule:
		intersection := btCapsuleAABB(otherBounds, box)
		if intersection != nil {
			intersection.MTV = intersection.MTV.Invert()
		}
		return intersection

	}

	return nil

}

// Type returns the NodeType for this object.
func (box *BoundingAABB) Type() NodeType {
	return NodeTypeBoundingAABB
}
