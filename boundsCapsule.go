package tetra3d

import (
	"math"
)

// BoundingCapsule represents a 3D capsule, whose primary purpose is to perform intersection testing between itself and other Bounding Nodes.
type BoundingCapsule struct {
	*Node
	Height         float64
	Radius         float64
	internalSphere *BoundingSphere
}

// NewBoundingCapsule returns a new BoundingCapsule instance. Name is the name of the underlying Node for the Capsule, height is the total
// height of the Capsule, and radius is how big around the capsule is. Height has to be at least radius (otherwise, it would no longer be a capsule).
func NewBoundingCapsule(name string, height, radius float64) *BoundingCapsule {
	return &BoundingCapsule{
		Node:           NewNode(name),
		Height:         math.Max(radius, height),
		Radius:         radius,
		internalSphere: NewBoundingSphere("internal sphere", 0),
	}
}

// Clone returns a new BoundingCapsule.
func (capsule *BoundingCapsule) Clone() INode {
	clone := NewBoundingCapsule(capsule.name, capsule.Height, capsule.Radius)
	clone.Node = capsule.Node.Clone().(*Node)
	return clone
}

// WorldRadius is the radius of the Capsule in world units, after taking into account its scale.
func (capsule *BoundingCapsule) WorldRadius() float64 {
	maxScale := 1.0
	if capsule.Node != nil {
		scale := capsule.Node.LocalScale()
		maxScale = math.Max(math.Max(math.Abs(scale.X), math.Abs(scale.Y)), math.Abs(scale.Z))
	}
	return capsule.Radius * maxScale
}

// Colliding returns true if the BoundingCapsule is intersecting the other BoundingObject.
func (capsule *BoundingCapsule) Colliding(other IBoundingObject) bool {
	return capsule.Collision(other) != nil
}

// Collision returns a Collision struct if the BoundingCapsule is intersecting another BoundingObject. If
// no intersection is reported, Collision returns nil.
func (capsule *BoundingCapsule) Collision(other IBoundingObject) *Collision {

	if other == capsule {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingCapsule:
		return btCapsuleCapsule(capsule, otherBounds)

	case *BoundingSphere:
		intersection := btSphereCapsule(otherBounds, capsule)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				inter.Normal = inter.Normal.Invert()
			}
			intersection.BoundingObject = otherBounds
		}
		return intersection

	case *BoundingAABB:
		return btCapsuleAABB(capsule, otherBounds)

	case *BoundingTriangles:
		return btCapsuleTriangles(capsule, otherBounds)

	}

	panic("Unimplemented bounds type")

}

// CollisionTest performs a collision test using the provided collision test settings structure.
// As a nicety, CollisionTest also returns a distance-sorted slice of all of the Collisions (but you should rather
// handle collisions with intent using the OnCollision function of the CollisionTestSettings struct).
func (capsule *BoundingCapsule) CollisionTest(settings CollisionTestSettings) []*Collision {
	return CommonCollisionTest(capsule, settings)
}

// PointInside returns true if the point provided is within the capsule.
func (capsule *BoundingCapsule) PointInside(point Vector) bool {
	return capsule.ClosestPoint(point).Sub(point).Magnitude() < capsule.WorldRadius()
}

// ClosestPoint returns the closest point on the capsule's "central line" to the point provided. Essentially, ClosestPoint returns a point
// along the capsule's line in world coordinates, capped between its bottom and top.
func (capsule *BoundingCapsule) ClosestPoint(point Vector) Vector {

	up := capsule.Node.WorldRotation().Up()
	pos := capsule.Node.WorldPosition()

	start := pos
	start.X += up.X * (-capsule.Height/2 + capsule.Radius)
	start.Y += up.Y * (-capsule.Height/2 + capsule.Radius)
	start.Z += up.Z * (-capsule.Height/2 + capsule.Radius)

	end := pos
	end.X += up.X * (capsule.Height/2 - capsule.Radius)
	end.Y += up.Y * (capsule.Height/2 - capsule.Radius)
	end.Z += up.Z * (capsule.Height/2 - capsule.Radius)

	segment := end.Sub(start)

	point = point.Sub(start)
	t := point.Dot(segment) / segment.Dot(segment)
	if t > 1 {
		t = 1
	} else if t < 0 {
		t = 0
	}

	start.X += segment.X * t
	start.Y += segment.Y * t
	start.Z += segment.Z * t

	return start

}

// lineTop returns the world position of the internal top end of the BoundingCapsule's line (i.e. this subtracts the
// capsule's radius).
func (capsule *BoundingCapsule) lineTop() Vector {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(capsule.Height/2 - capsule.Radius))
}

// Top returns the world position of the top of the BoundingCapsule.
func (capsule *BoundingCapsule) Top() Vector {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(capsule.Height / 2))
}

// lineBottom returns the world position of the internal bottom end of the BoundingCapsule's line (i.e. this subtracts the
// capsule's radius).
func (capsule *BoundingCapsule) lineBottom() Vector {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(-capsule.Height/2 + capsule.Radius))
}

// Bottom returns the world position of the bottom of the BoundingCapsule.
func (capsule *BoundingCapsule) Bottom() Vector {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(-capsule.Height / 2))
}

/////

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (capsule *BoundingCapsule) AddChildren(children ...INode) {
	capsule.addChildren(capsule, children...)
}

// Unparent unparents the Camera from its parent, removing it from the scenegraph.
func (capsule *BoundingCapsule) Unparent() {
	if capsule.parent != nil {
		capsule.parent.RemoveChildren(capsule)
	}
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (capsule *BoundingCapsule) Index() int {
	if capsule.parent != nil {
		for i, c := range capsule.parent.Children() {
			if c == capsule {
				return i
			}
		}
	}
	return -1
}

// Type returns the NodeType for this object.
func (capsule *BoundingCapsule) Type() NodeType {
	return NodeTypeBoundingCapsule
}
