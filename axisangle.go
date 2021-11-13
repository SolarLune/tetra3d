package tetra3d

import (
	"github.com/kvartborg/vector"
)

// AxisAngle represents a rotation in radians around a given 3D axis. This being the case, a AxisAngle can easily also be stored in a 4-dimensional vector; it's separated
// here into a 3D Vector and angle for simplicity and readability.
type AxisAngle struct {
	Axis  vector.Vector // 3 dimensional axis for rotating
	Angle float64       // Rotation in radians
}

// NewAxisAngle creates a new Quaternion out of the given 3D vector axis and angular rotation.
func NewAxisAngle(axis vector.Vector, angle float64) AxisAngle {
	axisAngle := AxisAngle{
		Axis:  axis.Unit(),
		Angle: angle,
	}
	return axisAngle
}

func (aa AxisAngle) Clone() AxisAngle {
	return NewAxisAngle(aa.Axis, aa.Angle)
}

// RotateVector rotates the given Vector by the axis and angle given, returning a rotated copy of it. For example, assuming the AxisAngle had an Axis
// of [0, 1, 0] (+Y, or "Up") and an Angle of pi / 2, axisAngle.RotateVector(vector.Vector{1, 0, 0}) would return vector.Vector{0, 0, -1}.
func (aa AxisAngle) RotateVector(vec vector.Vector) vector.Vector {
	return vec.Rotate(-aa.Angle, aa.Axis) // Angle is negative here to match the coordinate system.
}

func (aa AxisAngle) Add(otherAngle AxisAngle) AxisAngle {

	mat := Rotate(aa.Axis[0], aa.Axis[1], aa.Axis[2], aa.Angle)
	other := Rotate(otherAngle.Axis[0], otherAngle.Axis[1], otherAngle.Axis[2], otherAngle.Angle)
	_, _, a := mat.Mult(other).Decompose()
	return a

}

func (aa AxisAngle) Sub(otherAngle AxisAngle) AxisAngle {

	oa := otherAngle.Clone()
	oa.Angle *= -1
	return aa.Add(oa)

}

func AxisAngleLookAt(from, to, up vector.Vector) AxisAngle {
	_, _, r := LookAt(from, to, up).Decompose()
	return r
}
