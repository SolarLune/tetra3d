package tetra3d

import (
	"math"
)

// BoundingAABB represents a 3D AABB (Axis-Aligned Bounding Box), a 3D cube of varying width, height, and depth that cannot rotate.
// The primary purpose of a BoundingAABB is, like the other Bounding* Nodes, to perform intersection testing between itself and other
// BoundingObject Nodes.
type BoundingAABB struct {
	*Node
	internalSize Vector
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
		internalSize: Vector{width, height, depth, 0},
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

	dimensions := NewEmptyDimensions()

	for _, c := range corners {

		position := r.MultVec(Vector{
			box.internalSize.X * c[0] * s.X / 2,
			box.internalSize.Y * c[1] * s.Y / 2,
			box.internalSize.Z * c[2] * s.Z / 2,
			0,
		})

		if dimensions.Min.X > position.X {
			dimensions.Min.X = position.X
		}

		if dimensions.Min.Y > position.Y {
			dimensions.Min.Y = position.Y
		}

		if dimensions.Min.Z > position.Z {
			dimensions.Min.Z = position.Z
		}

		if dimensions.Max.X < position.X {
			dimensions.Max.X = position.X
		}

		if dimensions.Max.Y < position.Y {
			dimensions.Max.Y = position.Y
		}

		if dimensions.Max.Z < position.Z {
			dimensions.Max.Z = position.Z
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

	if box.internalSize.X != newWidth || box.internalSize.Y != newHeight || box.internalSize.Z != newDepth {
		box.internalSize.X = newWidth
		box.internalSize.Y = newHeight
		box.internalSize.Z = newDepth
		box.updateSize()
	}

}

// Clone returns a new BoundingAABB.
func (box *BoundingAABB) Clone() INode {
	clone := NewBoundingAABB(box.name, box.internalSize.X, box.internalSize.Y, box.internalSize.Z)
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

// ClosestPoint returns the closest point, to the point given, on the inside or surface of the BoundingAABB
// in world space.
func (box *BoundingAABB) ClosestPoint(point Vector) Vector {
	out := point
	pos := box.WorldPosition()

	half := box.Dimensions.Size().Scale(0.5)

	if out.X > pos.X+half.X {
		out.X = pos.X + half.X
	} else if out.X < pos.X-half.X {
		out.X = pos.X - half.X
	}

	if out.Y > pos.Y+half.Y {
		out.Y = pos.Y + half.Y
	} else if out.Y < pos.Y-half.Y {
		out.Y = pos.Y - half.Y
	}

	if out.Z > pos.Z+half.Z {
		out.Z = pos.Z + half.Z
	} else if out.Z < pos.Z-half.Z {
		out.Z = pos.Z - half.Z
	}

	return out
}

// aabbNormalGuess guesses which normal to return for an AABB given an MTV vector. Basically, if you have an MTV vector indicating a sphere, for example,
// moves up by 0.1 when colliding with an AABB, it must be colliding with the top, and so the returned normal would be [0, 1, 0].
func aabbNormalGuess(dir Vector) Vector {

	if dir.X == 0 && dir.Y == 0 && dir.Z == 0 {
		return NewVectorZero()
	}

	ax := math.Abs(dir.X)
	ay := math.Abs(dir.Y)
	az := math.Abs(dir.Z)

	if ax > az && ax > ay {
		// X is greatest axis
		if dir.X > 0 {
			return Vector{1, 0, 0, 0}
		} else {
			return Vector{-1, 0, 0, 0}
		}
	}

	if ay > az {
		// Y is greatest axis

		if dir.Y > 0 {
			return Vector{0, 1, 0, 0}
		} else {
			return Vector{0, -1, 0, 0}
		}
	}

	// Z is greatest axis
	if dir.Z > 0 {
		return Vector{0, 0, 1, 0}
	} else {
		return Vector{0, 0, -1, 0}
	}

}

// Colliding returns true if the BoundingAABB collides with another IBoundingObject.
func (box *BoundingAABB) Colliding(other IBoundingObject) bool {
	return box.Collision(other) != nil
}

// Collision returns the Collision between the BoundingAABB and the other IBoundingObject. If
// there is no intersection, the function returns nil. (Note that BoundingAABB > BoundingTriangles collision
// is buggy at the moment.)
func (box *BoundingAABB) Collision(other IBoundingObject) *Collision {

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
				inter.Normal = inter.Normal.Invert()
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
				inter.Normal = inter.Normal.Invert()
			}
			intersection.BoundingObject = otherBounds
		}
		return intersection

	}

	panic("Unimplemented bounds type")

}

// CollisionTest performs a collision test using the provided collision test settings structure.
// As a nicety, CollisionTest also returns a distance-sorted slice of all of the Collisions (but you should rather
// handle collisions with intent using the OnCollision function of the CollisionTestSettings struct).
func (box *BoundingAABB) CollisionTest(settings CollisionTestSettings) []*Collision {
	return CommonCollisionTest(box, settings)
}

// Type returns the NodeType for this object.
func (box *BoundingAABB) Type() NodeType {
	return NodeTypeBoundingAABB
}

func (box *BoundingAABB) PointInside(point Vector) bool {

	position := box.WorldPosition()
	min := box.Dimensions.Min.Add(position)
	max := box.Dimensions.Max.Add(position)
	margin := 0.01

	if point.X >= min.X-margin && point.X <= max.X+margin &&
		point.Y >= min.Y-margin && point.Y <= max.Y+margin &&
		point.Z >= min.Z-margin && point.Z <= max.Z+margin {
		return true
	}

	return false
}
