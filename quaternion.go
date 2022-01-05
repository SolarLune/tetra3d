package tetra3d

import "math"

type Quaternion struct {
	X, Y, Z, W float64
}

func NewQuaternion(x, y, z, w float64) *Quaternion {
	return &Quaternion{x, y, z, w}
}

func (quat *Quaternion) Clone() *Quaternion {
	return NewQuaternion(quat.X, quat.Y, quat.Z, quat.W)
}

// func (quat *Quaternion) Slerp(other *Quaternion, percent float64) *Quaternion {

// 	if percent <= 0 {
// 		return quat.Clone()
// 	} else if percent >= 1 {
// 		return other.Clone()
// 	}

// 	newQuat := quat.Clone()

// 	angle := quat.Dot(other)

// 	if math.Abs(angle) >= 1 {
// 		return newQuat
// 	}

// 	sinHalfTheta := math.Sqrt(1 - angle*angle)
// 	halfTheta := math.Atan2(sinHalfTheta, angle)

// 	if angle < 0 {
// 		newQuat.W = -other.W
// 		newQuat.X = -other.X
// 		newQuat.Y = -other.Y
// 		newQuat.Z = -other.Z
// 	}

// 	if angle >= 1 {
// 		return quat.Clone()
// 	}

// 	ratioA := math.Sin((1-percent)*halfTheta) / sinHalfTheta
// 	ratioB := math.Sin(percent*halfTheta) / sinHalfTheta

// 	newQuat.W = quat.W*ratioA + other.W*ratioB
// 	newQuat.X = quat.X*ratioA + other.X*ratioB
// 	newQuat.Y = quat.Y*ratioA + other.Y*ratioB
// 	newQuat.Z = quat.Z*ratioA + other.Z*ratioB

// 	return newQuat

// }

func (quat *Quaternion) Lerp(end *Quaternion, percent float64) *Quaternion {

	if percent <= 0 {
		return quat.Clone()
	} else if percent >= 1 {
		return end.Clone()
	}

	if quat.Dot(end) < 0 {
		end = end.Negated()
	}

	x := quat.X - percent*(quat.X-end.X)
	y := quat.Y - percent*(quat.Y-end.Y)
	z := quat.Z - percent*(quat.Z-end.Z)
	w := quat.W - percent*(quat.W-end.W)

	return NewQuaternion(x, y, z, w)

}

func (quat *Quaternion) Dot(other *Quaternion) float64 {
	return quat.X*other.X + quat.Y*other.Y + quat.Z*other.Z + quat.W*other.W
}

func (quat *Quaternion) Magnitude() float64 {
	return math.Sqrt(
		(quat.X * quat.X) +
			(quat.Y * quat.Y) +
			(quat.Z * quat.Z) +
			(quat.W * quat.W),
	)
}

func (quat *Quaternion) Normalized() *Quaternion {
	newQuat := quat.Clone()
	m := newQuat.Magnitude()
	newQuat.X /= m
	newQuat.Y /= m
	newQuat.Z /= m
	newQuat.W /= m
	return newQuat
}

func (quat *Quaternion) Negated() *Quaternion {
	return NewQuaternion(-quat.X, -quat.Y, -quat.Z, -quat.W)
}
