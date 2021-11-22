package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

// BoundingSphere represents a 3D Sphere.
type BoundingSphere struct {
	Node          Node
	LocalPosition vector.Vector
	LocalRadius   float64
}

// NewBoundingSphere returns a new BoundingSphere instance.
func NewBoundingSphere(node Node, localPosition vector.Vector, radius float64) *BoundingSphere {
	return &BoundingSphere{
		Node:          node,
		LocalPosition: localPosition,
		LocalRadius:   radius,
	}
}

func (sphere *BoundingSphere) Clone() *BoundingSphere {
	return NewBoundingSphere(sphere.Node, sphere.LocalPosition, sphere.LocalRadius)
}

func (sphere *BoundingSphere) Position() vector.Vector {
	pos := sphere.LocalPosition
	if sphere.Node != nil {
		pos = pos.Add(sphere.Node.LocalPosition())
	}
	return pos
}

func (sphere *BoundingSphere) Radius() float64 {
	maxScale := 1.0
	if sphere.Node != nil {
		scale := sphere.Node.LocalScale()
		maxScale = math.Max(math.Max(math.Abs(scale[0]), math.Abs(scale[1])), math.Abs(scale[2]))
	}
	return sphere.LocalRadius * maxScale
}
