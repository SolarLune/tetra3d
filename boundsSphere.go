package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
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
	return NewBoundingSphere(sphere.name, sphere.Radius)
}

// WorldRadius returns the radius of the BoundingSphere in world units, after taking into account its scale.
func (sphere *BoundingSphere) WorldRadius() float64 {
	maxScale := 1.0
	if sphere.Node != nil {
		scale := sphere.Node.LocalScale()
		maxScale = math.Max(math.Max(math.Abs(scale[0]), math.Abs(scale[1])), math.Abs(scale[2]))
	}
	return sphere.Radius * maxScale
}

// Intersecting returns true if the BoundingSphere is intersecting the other BoundingObject.
func (sphere *BoundingSphere) Intersecting(other BoundingObject) bool {
	return sphere.Intersection(other) != nil
}

// Intersection returns an IntersectionResult if the BoundingSphere is intersecting another BoundingObject. If
// no intersection is reported, Intersection returns nil.
func (sphere *BoundingSphere) Intersection(other BoundingObject) *IntersectionResult {

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

// PointInside returns whether the given point is inside of the sphere or not.
func (sphere *BoundingSphere) PointInside(point vector.Vector) bool {
	return sphere.Node.WorldPosition().Sub(point).Magnitude() < sphere.WorldRadius()
}
