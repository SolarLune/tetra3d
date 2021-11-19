package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

// BoundingSphere represents a 3D Sphere.
type BoundingSphere struct {
	Model         *Model
	LocalPosition vector.Vector
	LocalRadius   float64
}

// NewBoundingSphere returns a new BoundingSphere instance.
func NewBoundingSphere(model *Model, localPosition vector.Vector, radius float64) *BoundingSphere {
	return &BoundingSphere{
		Model:         model,
		LocalPosition: localPosition,
		LocalRadius:   radius,
	}
}

func (sphere *BoundingSphere) Clone() *BoundingSphere {
	return NewBoundingSphere(sphere.Model, sphere.LocalPosition, sphere.LocalRadius)
}

func (sphere *BoundingSphere) Position() vector.Vector {
	pos := sphere.LocalPosition
	if sphere.Model != nil {
		pos = pos.Add(sphere.Model.LocalPosition)
	}
	return pos
}

func (sphere *BoundingSphere) Radius() float64 {
	maxScale := 1.0
	if sphere.Model != nil {
		maxScale = math.Max(math.Max(math.Abs(sphere.Model.LocalScale[0]), math.Abs(sphere.Model.LocalScale[1])), math.Abs(sphere.Model.LocalScale[2]))
	}
	return sphere.LocalRadius * maxScale
}
