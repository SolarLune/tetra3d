package tetra3d

import "github.com/solarlune/tetra3d/math32"

// ColliderSphere represents a 3D sphere.
type ColliderSphere struct {
	*Node
	radius float32
}

// NewColliderSphere returns a new ColliderSphere instance.
func NewColliderSphere(name string, radius float32) *ColliderSphere {
	sphere := &ColliderSphere{
		Node:   NewNode(name),
		radius: radius,
	}
	sphere.owner = sphere
	return sphere
}

// Clone returns a new ColliderSphere instance.
func (sphere *ColliderSphere) Clone() INode {
	clone := NewColliderSphere(sphere.name, sphere.radius)
	clone.Node = sphere.Node.clone(clone).(*Node)
	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}
	return clone
}

// Returns a Dimensions set that fully contains the sphere object.
func (sphere *ColliderSphere) Dimensions() Dimensions {
	pos := sphere.WorldPosition()
	r := sphere.radius / 2
	return Dimensions{
		Min: NewVector3(
			pos.X-r,
			pos.Y-r,
			pos.Z-r,
		),

		Max: NewVector3(
			pos.X+r,
			pos.Y+r,
			pos.Z+r,
		),
	}
}

// Returns the radius of the ColliderSphere in world units, after taking into account its scale.
func (sphere *ColliderSphere) WorldRadius() float32 {
	var scale Vector3
	maxScale := float32(1.0)

	scale = sphere.Node.WorldScale() // We don't want to have to decompose the transform if we can help it

	maxScale = math32.Max(math32.Max(math32.Abs(scale.X), math32.Abs(scale.Y)), math32.Abs(scale.Z))

	return sphere.radius * maxScale
}

// Returns the local radius of the ColliderSphere object.
func (sphere *ColliderSphere) LocalRadius() float32 {
	return sphere.radius
}

// Sets the local radius of the ColliderSphere object.
func (sphere *ColliderSphere) SetRadius(radius float32) {
	sphere.radius = radius
}

// Colliding returns true if the ColliderSphere is intersecting the other Collider.
func (sphere *ColliderSphere) Colliding(other Collider) bool {
	return sphere.Collision(other) != nil
}

// Collision returns a Collision if the ColliderSphere is intersecting another Collider. If
// no intersection is reported, Collision returns nil.
func (sphere *ColliderSphere) Collision(other Collider) *Collision {

	if other == sphere || other == nil {
		return nil
	}

	switch otherCollider := other.(type) {

	case *ColliderSphere:
		return btSphereSphere(sphere, otherCollider)

	case *ColliderAABB:
		return btSphereAABB(sphere, otherCollider)

	case *ColliderTriangles:
		return btSphereTriangles(sphere, otherCollider)

	case *ColliderCapsule:
		return btSphereCapsule(sphere, otherCollider)

	}

	panic("Unimplemented collider type")

}

// CollisionTest performs a collision test using the provided collision test settings structure.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the object; if no collision is found, then it returns nil.
func (sphere *ColliderSphere) CollisionTest(settings CollisionTestSettings) *Collision {
	return commonCollisionTest(sphere, settings)
}

// PointInside returns whether the given point is inside of the sphere or not.
func (sphere *ColliderSphere) PointInside(point Vector3) bool {
	return sphere.Node.WorldPosition().Sub(point).Magnitude() < sphere.WorldRadius()
}

/////

// Type returns the NodeType for this object.
func (sphere *ColliderSphere) Type() NodeType {
	return NodeTypeColliderSphere
}
