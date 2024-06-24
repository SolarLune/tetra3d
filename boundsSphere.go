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
func (sphere *BoundingSphere) PointInside(point Vector) bool {
	return sphere.Node.WorldPosition().Sub(point).Magnitude() < sphere.WorldRadius()
}

/////

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (sphere *BoundingSphere) AddChildren(children ...INode) {
	sphere.addChildren(sphere, children...)
}

// Unparent unparents the Camera from its parent, removing it from the scenegraph.
func (sphere *BoundingSphere) Unparent() {
	if sphere.parent != nil {
		sphere.parent.RemoveChildren(sphere)
	}
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (sphere *BoundingSphere) Index() int {
	if sphere.parent != nil {
		for i, c := range sphere.parent.Children() {
			if c == sphere {
				return i
			}
		}
	}
	return -1
}

// Type returns the NodeType for this object.
func (sphere *BoundingSphere) Type() NodeType {
	return NodeTypeBoundingSphere
}
