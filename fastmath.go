package tetra3d

import "github.com/kvartborg/vector"

// The goal of fastmath.go is to provide vector operations that don't clone the vector to use. Be careful with it!

var standin = vector.Vector{0, 0, 0}

func fastSub(a, b vector.Vector) vector.Vector {

	standin[0] = a[0] - b[0]
	standin[1] = a[1] - b[1]
	standin[2] = a[2] - b[2]
	return standin

}
