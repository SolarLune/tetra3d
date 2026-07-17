package tetra3d

import (
	"github.com/solarlune/tetra3d/math32"
)

// ColliderAABB represents a 3D AABB (Axis-Aligned Bounding Box), a 3D cube of varying width, height, and depth that cannot rotate.
// The primary purpose of a ColliderAABB is, like the other colliders, to perform intersection testing between itself and other
// Collider Nodes.
type ColliderAABB struct {
	*Node
	internalSize Vector3
	dimensions   Dimensions // Dimensions represents the size of the AABB after transformation.
}

// NewColliderAABB returns a new ColliderAABB Node.
func NewColliderAABB(name string, width, height, depth float32) *ColliderAABB {
	min := float32(0.001)
	if width <= min {
		width = min
	}
	if height <= min {
		height = min
	}
	if depth <= min {
		depth = min
	}
	collider := &ColliderAABB{
		Node:         NewNode(name),
		internalSize: Vector3{width, height, depth},
	}
	collider.Node.onTransformUpdate = collider.updateSize
	collider.updateSize()
	collider.owner = collider
	return collider
}

func NewColliderFromDimensions(name string, dim Dimensions) *ColliderAABB {
	collider := &ColliderAABB{
		Node:         NewNode(name),
		internalSize: dim.Size(),
	}
	collider.SetWorldPositionVec(dim.Center())
	collider.Node.onTransformUpdate = collider.updateSize
	collider.updateSize()
	collider.owner = collider
	return collider
}

// updateSize updates the ColliderAABB's external Dimensions property to reflect its size after reposition, rotation, or resizing.
// This is be called automatically internally as necessary after the node's transform is updated.
func (box *ColliderAABB) updateSize() {

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

	dimensions := newEmptyDimensions()

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

	box.dimensions = dimensions

}

// SetDimensions sets the ColliderAABB's internal dimensions (prior to resizing or rotating the Node).
func (box *ColliderAABB) SetDimensions(newWidth, newHeight, newDepth float32) {

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

func (box *ColliderAABB) Dimensions() Dimensions {
	// The dimensions are already scaled and include rotation, so rather than transform,
	// they just need to be moved by the AABB's world position to be accurate
	dim := box.dimensions
	dim.Max = dim.Max.Add(box.WorldPosition())
	dim.Min = dim.Min.Add(box.WorldPosition())
	return dim.Canon()
}

// Clone returns a new ColliderAABB.
func (box *ColliderAABB) Clone() INode {
	clone := NewColliderAABB(box.name, box.internalSize.X, box.internalSize.Y, box.internalSize.Z)
	clone.Node = box.Node.clone(clone).(*Node)
	clone.Node.onTransformUpdate = clone.updateSize
	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}
	return clone
}

// ClosestPoint returns the closest point, to the point given, on the inside or surface of the ColliderAABB
// in world space.
func (box *ColliderAABB) ClosestPoint(point Vector3) Vector3 {
	pos := box.WorldPosition()

	half := box.dimensions.Size().Scale(0.5)

	if point.X > pos.X+half.X {
		point.X = pos.X + half.X
	} else if point.X < pos.X-half.X {
		point.X = pos.X - half.X
	}

	if point.Y > pos.Y+half.Y {
		point.Y = pos.Y + half.Y
	} else if point.Y < pos.Y-half.Y {
		point.Y = pos.Y - half.Y
	}

	if point.Z > pos.Z+half.Z {
		point.Z = pos.Z + half.Z
	} else if point.Z < pos.Z-half.Z {
		point.Z = pos.Z - half.Z
	}

	return point
}

// normalFromContactPoint guesses which normal to return for an AABB given an MTV vector. Basically, if you have an MTV vector indicating a sphere, for example,
// moves up by 0.1 when colliding with an AABB, it must be colliding with the top, and so the returned normal would be [0, 1, 0].
func (box *ColliderAABB) normalFromContactPoint(contactPoint Vector3) Vector3 {

	if contactPoint.Equals(box.WorldPosition()) {
		return Vector3{}
	}

	p := contactPoint.Sub(box.WorldPosition())
	d := Vector3{
		box.dimensions.Width() / 2,
		box.dimensions.Height() / 2,
		box.dimensions.Depth() / 2,
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

// Colliding returns true if the ColliderAABB collides with another Collider.
func (box *ColliderAABB) Colliding(other Collider) bool {
	return box.Collision(other) != nil
}

// ContainsAABB returns if the calling ColliderAABB contains the provided other ColliderAABB.
func (box *ColliderAABB) ContainsAABB(other *ColliderAABB) bool {

	mePos := box.WorldPosition()
	meMin := mePos.Sub(box.dimensions.Center())
	meMax := mePos.Add(box.dimensions.Center())

	otherPos := other.WorldPosition()
	otherMin := otherPos.Sub(other.dimensions.Center())
	otherMax := otherPos.Add(other.dimensions.Center())

	return otherMin.X > meMin.X && otherMin.Y > meMin.Y && otherMin.Z > meMin.Z && otherMax.X < meMax.X && otherMax.Y < meMax.Y && otherMax.Z < meMax.Z
}

// Collision returns the Collision between the ColliderAABB and the other Collider. If
// there is no intersection, the function returns nil. (Note that ColliderAABB > ColliderTriangles collision
// is buggy at the moment.)
func (box *ColliderAABB) Collision(other Collider) *Collision {

	if other == box || other == nil {
		return nil
	}

	switch otherCollider := other.(type) {

	case *ColliderAABB:
		return btAABBAABB(box, otherCollider)

	case *ColliderSphere:
		intersection := btSphereAABB(otherCollider, box)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				inter.Normal = inter.Normal.Invert()
			}
			intersection.Object = otherCollider
		}
		return intersection

	case *ColliderTriangles:
		return btAABBTriangles(box, otherCollider)

	case *ColliderCapsule:
		intersection := btCapsuleAABB(otherCollider, box)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				inter.Normal = inter.Normal.Invert()
			}
			intersection.Object = otherCollider
		}
		return intersection

	}

	panic("Unimplemented collider type")

}

// CollisionTest performs a collision test using the provided collision test settings structure.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the object; if no collision is found, then it returns nil.
func (box *ColliderAABB) CollisionTest(settings CollisionTestSettings) *Collision {
	return commonCollisionTest(box, settings)
}

func (box *ColliderAABB) PointInside(point Vector3) bool {

	position := box.WorldPosition()
	min := box.dimensions.Min.Add(position)
	max := box.dimensions.Max.Add(position)
	margin := float32(0.01)

	if point.X >= min.X-margin && point.X <= max.X+margin &&
		point.Y >= min.Y-margin && point.Y <= max.Y+margin &&
		point.Z >= min.Z-margin && point.Z <= max.Z+margin {
		return true
	}

	return false
}

// Type returns the NodeType for this object.
func (box *ColliderAABB) Type() NodeType {
	return NodeTypeColliderAABB
}
