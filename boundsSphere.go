package tetra3d

import (
	"math"
)

// BoundingSphere represents a 3D sphere.
type BoundingSphere struct {
	*Node
	Radius float64
}

// NewBoundingSphere returns a new BoundingSphere instance.
func NewBoundingSphere(name string, radius float64) *BoundingSphere {
	return &BoundingSphere{
		Node:   NewNode(name),
		Radius: radius,
	}
}

// Clone returns a new BoundingSphere instance.
func (sphere *BoundingSphere) Clone() INode {
	clone := NewBoundingSphere(sphere.name, sphere.Radius)
	clone.Node = sphere.Node.Clone().(*Node)
	return clone
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (sphere *BoundingSphere) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	sphere.addChildren(sphere, children...)
}

// WorldRadius returns the radius of the BoundingSphere in world units, after taking into account its scale.
func (sphere *BoundingSphere) WorldRadius() float64 {
	var scale Vector
	maxScale := 1.0
	if sphere.Node.Parent() != nil {
		scale = sphere.Node.WorldScale() // We don't want to have to decompose the transform if we can help it
	} else {
		scale = sphere.Node.scale // We don't want to have to make a memory duplicate if we don't have to
	}
	maxScale = math.Max(math.Max(math.Abs(scale.X), math.Abs(scale.Y)), math.Abs(scale.Z))
	return sphere.Radius * maxScale
}

// Colliding returns true if the BoundingSphere is intersecting the other BoundingObject.
func (sphere *BoundingSphere) Colliding(other IBoundingObject) bool {
	return sphere.Collision(other) != nil
}

// Collision returns a Collision if the BoundingSphere is intersecting another BoundingObject. If
// no intersection is reported, Collision returns nil.
func (sphere *BoundingSphere) Collision(other IBoundingObject) *Collision {

	if other == sphere {
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

// CollisionTest performs an collision test if the bounding object were to move in the given direction in world space.
// It returns all valid Collisions across all recursive children of the INodes slice passed in as others, testing against BoundingObjects in those trees.
// To exemplify this, if you had a Model that had a BoundingObject child, and then tested the Model for collision,
// the Model's children would be tested for collision (which means the BoundingObject), and the Model would be the
// collided object. Of course, if you simply tested the BoundingObject directly, then it would return the BoundingObject as the collided
// object.
// Collisions will be sorted in order of distance. If no Collisions occurred, it will return an empty slice.
func (sphere *BoundingSphere) CollisionTest(dx, dy, dz float64, others ...INode) []*Collision {
	return commonCollisionTest(sphere, dx, dy, dz, others...)
}

// CollisionTestVec performs an collision test if the bounding object were to move in the given direction in world space using a vector.
// It returns all valid Collisions across all recursive children of the INodes slice passed in as others, testing against BoundingObjects in those trees.
// To exemplify this, if you had a Model that had a BoundingObject child, and then tested the Model for collision,
// the Model's children would be tested for collision (which means the BoundingObject), and the Model would be the
// collided object. Of course, if you simply tested the BoundingObject directly, then it would return the BoundingObject as the collided
// object.
// Collisions will be sorted in order of distance. If no Collisions occurred, it will return an empty slice.
func (sphere *BoundingSphere) CollisionTestVec(moveVec Vector, others ...INode) []*Collision {
	return commonCollisionTest(sphere, moveVec.X, moveVec.Y, moveVec.Z, others...)
}

// PointInside returns whether the given point is inside of the sphere or not.
func (sphere *BoundingSphere) PointInside(point Vector) bool {
	return sphere.Node.WorldPosition().Sub(point).Magnitude() < sphere.WorldRadius()
}

// Type returns the NodeType for this object.
func (sphere *BoundingSphere) Type() NodeType {
	return NodeTypeBoundingSphere
}
