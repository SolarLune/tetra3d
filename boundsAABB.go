package tetra3d

import "github.com/solarlune/tetra3d/math32"

// BoundingAABB represents a 3D AABB (Axis-Aligned Bounding Box), a 3D cube of varying width, height, and depth that cannot rotate.
// The primary purpose of a BoundingAABB is, like the other Bounding* Nodes, to perform intersection testing between itself and other
// BoundingObject Nodes.
type BoundingAABB struct {
	*Node
	internalSize Vector3
	Dimensions   Dimensions // Dimensions represents the size of the AABB after transformation.
}

// NewBoundingAABB returns a new BoundingAABB Node.
func NewBoundingAABB(name string, width, height, depth float32) *BoundingAABB {
	min := float32(0.0001)
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
		internalSize: Vector3{width, height, depth},
	}
	bounds.Node.onTransformUpdate = bounds.updateSize
	bounds.updateSize()
	bounds.owner = bounds
	return bounds
}

// updateSize updates the BoundingAABB's external Dimensions property to reflect its size after reposition, rotation, or resizing.
// This is be called automatically internally as necessary after the node's transform is updated.
func (box *BoundingAABB) updateSize() {

	_, s, r := box.Node.Transform().Decompose()

	corners := [][]float32{
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

		position := r.MultVec(Vector3{
			box.internalSize.X * c[0] * s.X / 2,
			box.internalSize.Y * c[1] * s.Y / 2,
			box.internalSize.Z * c[2] * s.Z / 2,
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
func (box *BoundingAABB) SetDimensions(newWidth, newHeight, newDepth float32) {

	min := float32(0.00001)
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
	clone.Node = box.Node.clone(clone).(*Node)
	clone.Node.onTransformUpdate = clone.updateSize
	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}
	return clone
}

// ClosestPoint returns the closest point, to the point given, on the inside or surface of the BoundingAABB
// in world space.
func (box *BoundingAABB) ClosestPoint(point Vector3) Vector3 {
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

// normalFromContactPoint guesses which normal to return for an AABB given an MTV vector. Basically, if you have an MTV vector indicating a sphere, for example,
// moves up by 0.1 when colliding with an AABB, it must be colliding with the top, and so the returned normal would be [0, 1, 0].
func (box *BoundingAABB) normalFromContactPoint(contactPoint Vector3) Vector3 {

	if contactPoint.Equals(box.WorldPosition()) {
		return Vector3{}
	}

	p := contactPoint.Sub(box.WorldPosition())
	d := Vector3{
		box.Dimensions.Width() / 2,
		box.Dimensions.Height() / 2,
		box.Dimensions.Depth() / 2,
	}

	nx := p.X / d.X
	ny := p.Y / d.Y
	nz := p.Z / d.Z

	if math32.Abs(nx) > math32.Abs(ny) && math32.Abs(nx) > math32.Abs(nz) {
		return Vector3{nx, 0, 0}.Unit()
	} else if math32.Abs(ny) > math32.Abs(nx) && math32.Abs(ny) > math32.Abs(nz) {
		return Vector3{0, ny, 0}.Unit()
	}
	return Vector3{0, 0, nz}.Unit()

}

// Colliding returns true if the BoundingAABB collides with another IBoundingObject.
func (box *BoundingAABB) Colliding(other IBoundingObject) bool {
	return box.Collision(other) != nil
}

// ContainsAABB returns if the calling BoundingAABB contains the provided other BoundingAABB.
func (box *BoundingAABB) ContainsAABB(other *BoundingAABB) bool {

	mePos := box.WorldPosition()
	meMin := mePos.Sub(box.Dimensions.Center())
	meMax := mePos.Add(box.Dimensions.Center())

	otherPos := other.WorldPosition()
	otherMin := otherPos.Sub(other.Dimensions.Center())
	otherMax := otherPos.Add(other.Dimensions.Center())

	return otherMin.X > meMin.X && otherMin.Y > meMin.Y && otherMin.Z > meMin.Z && otherMax.X < meMax.X && otherMax.Y < meMax.Y && otherMax.Z < meMax.Z
}

// Collision returns the Collision between the BoundingAABB and the other IBoundingObject. If
// there is no intersection, the function returns nil. (Note that BoundingAABB > BoundingTriangles collision
// is buggy at the moment.)
func (box *BoundingAABB) Collision(other IBoundingObject) *Collision {

	if other == box || other == nil {
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
			intersection.Object = otherBounds
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
			intersection.Object = otherBounds
		}
		return intersection

	}

	panic("Unimplemented bounds type")

}

// CollisionTest performs a collision test using the provided collision test settings structure.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the object; if no collision is found, then it returns nil.
func (box *BoundingAABB) CollisionTest(settings CollisionTestSettings) *Collision {
	return commonCollisionTest(box, settings)
}

func (box *BoundingAABB) PointInside(point Vector3) bool {

	position := box.WorldPosition()
	min := box.Dimensions.Min.Add(position)
	max := box.Dimensions.Max.Add(position)
	margin := float32(0.01)

	if point.X >= min.X-margin && point.X <= max.X+margin &&
		point.Y >= min.Y-margin && point.Y <= max.Y+margin &&
		point.Z >= min.Z-margin && point.Z <= max.Z+margin {
		return true
	}

	return false
}

// Type returns the NodeType for this object.
func (box *BoundingAABB) Type() NodeType {
	return NodeTypeBoundingAABB
}
