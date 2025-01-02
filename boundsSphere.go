package tetra3d

import "github.com/solarlune/tetra3d/math32"

// BoundingSphere represents a 3D sphere.
type BoundingSphere struct {
	*Node
	Radius float32
}

// NewBoundingSphere returns a new BoundingSphere instance.
func NewBoundingSphere(name string, radius float32) *BoundingSphere {
	sphere := &BoundingSphere{
		Node:   NewNode(name),
		Radius: radius,
	}
	sphere.owner = sphere
	return sphere
}

// Clone returns a new BoundingSphere instance.
func (sphere *BoundingSphere) Clone() INode {
	clone := NewBoundingSphere(sphere.name, sphere.Radius)
	clone.Node = sphere.Node.clone(clone).(*Node)
	if clone.Callbacks() != nil && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}
	return clone
}

// WorldRadius returns the radius of the BoundingSphere in world units, after taking into account its scale.
func (sphere *BoundingSphere) WorldRadius() float32 {
	var scale Vector3
	maxScale := float32(1.0)
	if sphere.Node.Parent() != nil {
		scale = sphere.Node.WorldScale() // We don't want to have to decompose the transform if we can help it
	} else {
		scale = sphere.Node.scale // We don't want to have to make a memory duplicate if we don't have to
	}
	maxScale = math32.Max(math32.Max(math32.Abs(scale.X), math32.Abs(scale.Y)), math32.Abs(scale.Z))
	return sphere.Radius * maxScale
}

// Colliding returns true if the BoundingSphere is intersecting the other BoundingObject.
func (sphere *BoundingSphere) Colliding(other IBoundingObject) bool {
	return sphere.Collision(other) != nil
}

// Collision returns a Collision if the BoundingSphere is intersecting another BoundingObject. If
// no intersection is reported, Collision returns nil.
func (sphere *BoundingSphere) Collision(other IBoundingObject) *Collision {

	if other == sphere || other == nil {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingSphere:
		return btSphereSphere(sphere, otherBounds)

	case *BoundingAABB:
		return btSphereAABB(sphere, otherBounds)

	case *BoundingTriangles:
		return btSphereTriangles(sphere, otherBounds)

	case *BoundingCapsule:
		return btSphereCapsule(sphere, otherBounds)

	}

	panic("Unimplemented bounds type")

}

// CollisionTest performs a collision test using the provided collision test settings structure.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return if a collision was found with the sphere at the settings specified.
func (sphere *BoundingSphere) CollisionTest(settings CollisionTestSettings) bool {
	return commonCollisionTest(sphere, settings)
}

// PointInside returns whether the given point is inside of the sphere or not.
func (sphere *BoundingSphere) PointInside(point Vector3) bool {
	return sphere.Node.WorldPosition().Sub(point).Magnitude() < sphere.WorldRadius()
}

/////

// Type returns the NodeType for this object.
func (sphere *BoundingSphere) Type() NodeType {
	return NodeTypeBoundingSphere
}
