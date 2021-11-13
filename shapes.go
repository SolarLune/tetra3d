package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

type BoundingSphere struct {
	Model         *Model
	LocalPosition vector.Vector
	LocalRadius   float64
}

func NewBoundingSphere(model *Model, localPosition vector.Vector, radius float64) *BoundingSphere {
	return &BoundingSphere{
		Model:         model,
		LocalPosition: localPosition,
		LocalRadius:   radius,
	}
}

func (sphere *BoundingSphere) Position() vector.Vector {
	pos := sphere.LocalPosition
	if sphere.Model != nil {
		pos = pos.Add(sphere.Model.Position)
	}
	return pos
}

func (sphere *BoundingSphere) Radius() float64 {
	maxScale := 1.0
	if sphere.Model != nil {
		maxScale = math.Max(math.Max(math.Abs(sphere.Model.Scale[0]), math.Abs(sphere.Model.Scale[1])), math.Abs(sphere.Model.Scale[2]))
	}
	return sphere.LocalRadius * maxScale
}
