package jank3d

import "github.com/kvartborg/vector"

type Sphere struct {
	Position vector.Vector
	Radius   float64
}

func NewSphere(position vector.Vector, radius float64) *Sphere {
	return &Sphere{
		Position: position,
		Radius:   radius,
	}
}
