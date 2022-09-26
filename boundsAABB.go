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
	Dimensions   Dimensions
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
	bounds.Node.onTransformUpdate = bounds.updateSize
	bounds.updateSize()
	return bounds
}

// updateSize updates the BoundingAABB's external Dimensions property to reflect its size after reposition, rotation, or resizing.
// This is be called automatically internally as necessary after the node's transform is updated.
func (box *BoundingAABB) updateSize() {

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

	dimensions := Dimensions{
		vector.Vector{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64},
		vector.Vector{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64},
	}

	for _, c := range corners {

		position := r.MultVec(vector.Vector{
			box.internalSize[0] * c[0] * s[0] / 2,
			box.internalSize[1] * c[1] * s[1] / 2,
			box.internalSize[2] * c[2] * s[2] / 2,
		})

		if dimensions[0][0] > position[0] {
			dimensions[0][0] = position[0]
		}

		if dimensions[0][1] > position[1] {
			dimensions[0][1] = position[1]
		}

		if dimensions[0][2] > position[2] {
			dimensions[0][2] = position[2]
		}

		if dimensions[1][0] < position[0] {
			dimensions[1][0] = position[0]
		}

		if dimensions[1][1] < position[1] {
			dimensions[1][1] = position[1]
		}

		if dimensions[1][2] < position[2] {
			dimensions[1][2] = position[2]
		}

	}

	box.Dimensions = dimensions

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
		box.updateSize()
	}

}

// Clone returns a new BoundingAABB.
func (box *BoundingAABB) Clone() INode {
	clone := NewBoundingAABB(box.name, box.internalSize[0], box.internalSize[1], box.internalSize[2])
	clone.Node = box.Node.Clone().(*Node)
	clone.Node.onTransformUpdate = clone.updateSize
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

	half := box.Dimensions.Size().Scale(0.5)

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

// aabbNormalGuess guesses which normal to return for an AABB given an MTV vector. Basically, if you have an MTV vector indicating a sphere, for example,
// moves up by 0.1 when colliding with an AABB, it must be colliding with the top, and so the returned normal would be [0, 1, 0].
func aabbNormalGuess(dir vector.Vector) vector.Vector {

	if dir[0] == 0 && dir[1] == 0 && dir[2] == 0 {
		return vector.Vector{0, 0, 0}
	}

	ax := math.Abs(dir[0])
	ay := math.Abs(dir[1])
	az := math.Abs(dir[2])

	if ax > az && ax > ay {
		// X is greatest axis
		if dir[0] > 0 {
			return vector.Vector{1, 0, 0}
		} else {
			return vector.Vector{-1, 0, 0}
		}
	}

	if ay > az {
		// Y is greatest axis

		if dir[1] > 0 {
			return vector.Vector{0, 1, 0}
		} else {
			return vector.Vector{0, -1, 0}
		}
	}

	// Z is greatest axis
	if dir[2] > 0 {
		return vector.Vector{0, 0, 1}
	} else {
		return vector.Vector{0, 0, -1}
	}

}

// Colliding returns true if the BoundingAABB is colliding with another BoundingObject.
func (box *BoundingAABB) Colliding(other BoundingObject) bool {
	return box.Collision(other) != nil
}

// Collision returns the Collision between the BoundingAABB and the other BoundingObject. If
// there is no intersection, the function returns nil. (Note that BoundingAABB > BoundingTriangles collision
// is buggy at the moment.)
func (box *BoundingAABB) Collision(other BoundingObject) *Collision {

	if other == box {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingAABB:
		return btAABBAABB(box, otherBounds)

	case *BoundingSphere:
		intersection := btSphereAABB(otherBounds, box)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				vector.In(inter.Normal).Invert()
			}
			intersection.BoundingObject = otherBounds
		}
		return intersection

	case *BoundingTriangles:
		return btAABBTriangles(box, otherBounds)

	case *BoundingCapsule:
		intersection := btCapsuleAABB(otherBounds, box)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				vector.In(inter.Normal).Invert()
			}
			intersection.BoundingObject = otherBounds
		}
		return intersection

	}

	panic("Unimplemented bounds type")

}

// CollisionTest performs an collision test if the bounding object were to move in the given direction in world space.
// It returns all valid Collisions across all recursive children of the INodes slice passed in as others, testing against BoundingObjects in those trees.
// To exemplify this, if you had a Model that had a BoundingObject child, and then tested the Model for collision,
// the Model's children would be tested for collision (which means the BoundingObject), and the Model would be the
// collided object. Of course, if you simply tested the BoundingObject directly, then it would return the BoundingObject as the collided
// object.
// Collisions will be sorted in order of distance. If no Collisions occurred, it will return an empty slice.
func (box *BoundingAABB) CollisionTest(dx, dy, dz float64, others ...INode) []*Collision {
	return commonCollisionTest(box, dx, dy, dz, others...)
}

// CollisionTestVec performs an collision test if the bounding object were to move in the given direction in world space using a vector.
// It returns all valid Collisions across all recursive children of the INodes slice passed in as others, testing against BoundingObjects in those trees.
// To exemplify this, if you had a Model that had a BoundingObject child, and then tested the Model for collision,
// the Model's children would be tested for collision (which means the BoundingObject), and the Model would be the
// collided object. Of course, if you simply tested the BoundingObject directly, then it would return the BoundingObject as the collided
// object.
// Collisions will be sorted in order of distance. If no Collisions occurred, it will return an empty slice.
func (box *BoundingAABB) CollisionTestVec(moveVec vector.Vector, others ...INode) []*Collision {
	if moveVec == nil {
		return commonCollisionTest(box, 0, 0, 0, others...)
	}
	return commonCollisionTest(box, moveVec[0], moveVec[1], moveVec[2], others...)
}

// Type returns the NodeType for this object.
func (box *BoundingAABB) Type() NodeType {
	return NodeTypeBoundingAABB
}

func (box *BoundingAABB) PointInside(point vector.Vector) bool {

	position := box.WorldPosition()
	min := box.Dimensions[0].Add(position)
	max := box.Dimensions[1].Add(position)
	margin := 0.01

	if point[0] >= min[0]-margin && point[0] <= max[0]+margin &&
		point[1] >= min[1]-margin && point[1] <= max[1]+margin &&
		point[2] >= min[2]-margin && point[2] <= max[2]+margin {
		return true
	}

	return false
}
