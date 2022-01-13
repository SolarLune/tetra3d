package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
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

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (capsule *BoundingCapsule) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	capsule.addChildren(capsule, children...)
}

// WorldRadius is the radius of the Capsule in world units, after taking into account its scale.
func (capsule *BoundingCapsule) WorldRadius() float64 {
	maxScale := 1.0
	if capsule.Node != nil {
		scale := capsule.Node.LocalScale()
		maxScale = math.Max(math.Max(math.Abs(scale[0]), math.Abs(scale[1])), math.Abs(scale[2]))
	}
	return capsule.Radius * maxScale
}

// Intersecting returns true if the BoundingCapsule is intersecting the other BoundingObject.
func (capsule *BoundingCapsule) Intersecting(other BoundingObject) bool {
	return capsule.Intersection(other) != nil
}

// Intersection returns an IntersectionResult if the BoundingCapsule is intersecting another BoundingObject. If
// no intersection is reported, Intersection returns nil.
func (capsule *BoundingCapsule) Intersection(other BoundingObject) *IntersectionResult {

	if other == capsule {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingCapsule:
		return btCapsuleCapsule(capsule, otherBounds)

	case *BoundingSphere:
		intersection := btSphereCapsule(otherBounds, capsule)
		if intersection != nil {
			intersection.MTV = intersection.MTV.Invert()
		}
		return intersection

	case *BoundingAABB:
		return btCapsuleAABB(capsule, otherBounds)

	case *BoundingTriangles:
		return btCapsuleTriangles(capsule, otherBounds)

	}

	panic("Unimplemented bounds type")

}

// PointInside returns true if the point provided is within the capsule.
func (capsule *BoundingCapsule) PointInside(point vector.Vector) bool {
	return capsule.ClosestPoint(point).Sub(point).Magnitude() < capsule.WorldRadius()
}

// ClosestPoint returns the closest point on the capsule's "central line" to the point provided. Essentially, ClosestPoint returns a point
// along the capsule's line, somewhere between BoundingCapsule.Bottom() and BoundingCapsule.Top().
func (capsule *BoundingCapsule) ClosestPoint(point vector.Vector) vector.Vector {

	up := capsule.Node.WorldRotation().Up()
	start := capsule.Node.WorldPosition().Add(up.Scale(-capsule.Height/2 + capsule.Radius))
	end := capsule.Node.WorldPosition().Add(up.Scale(capsule.Height/2 - capsule.Radius))
	segment := end.Sub(start)

	t := point.Sub(start).Dot(segment) / segment.Dot(segment)
	t = math.Max(math.Min(t, 1), 0)

	return start.Add(segment.Scale(t))

}

// Top returns the world position of the top of the BoundingCapsule.
func (capsule *BoundingCapsule) Top() vector.Vector {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(capsule.Height/2 - capsule.Radius))
}

// Bottom returns the world position of the bottom of the BoundingCapsule.
func (capsule *BoundingCapsule) Bottom() vector.Vector {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(-capsule.Height/2 + capsule.Radius))
}
