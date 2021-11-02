package ebiten3d

import "github.com/kvartborg/vector"

// A Quaternion represents a rotation in radians around a given 3D axis. This being the case, a Quaternion can easily also be stored in a 4-dimensional vector; it's separated
// here for simplicity and readability.
type Quaternion struct {
	Axis  vector.Vector // 3 dimensional axis for rotating
	Angle float64       // Rotation in radians
}

// NewQuaternion creates a new Quaternion out of the given 3D vector axis and angular rotation.
func NewQuaternion(axis vector.Vector, angle float64) Quaternion {
	quat := Quaternion{
		Axis:  axis.Unit(),
		Angle: 0,
	}
	return quat
}

// RotateVector uses the Quaternion to rotate the given Vector by the axis and angle given.
func (quat Quaternion) RotateVector(vec vector.Vector) vector.Vector {
	return vec.Rotate(-quat.Angle, quat.Axis)
}
