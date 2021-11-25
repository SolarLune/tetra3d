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

func (quat *Quaternion) Slerp(other *Quaternion, percent float64) *Quaternion {

	if percent <= 0 {
		return quat.Clone()
	} else if percent >= 1 {
		return other.Clone()
	}

	newQuat := quat.Clone()

	angle := quat.Dot(other)

	if math.Abs(angle) >= 1 {
		return newQuat
	}

	sinHalfTheta := math.Sqrt(1 - angle*angle)
	halfTheta := math.Atan2(sinHalfTheta, angle)

	if angle < 0 {
		newQuat.W = -other.W
		newQuat.X = -other.X
		newQuat.Y = -other.Y
		newQuat.Z = -other.Z
	}

	if angle >= 1 {
		return quat.Clone()
	}

	ratioA := math.Sin((1-percent)*halfTheta) / sinHalfTheta
	ratioB := math.Sin(percent*halfTheta) / sinHalfTheta

	newQuat.W = quat.W*ratioA + other.W*ratioB
	newQuat.X = quat.X*ratioA + other.X*ratioB
	newQuat.Y = quat.Y*ratioA + other.Y*ratioB
	newQuat.Z = quat.Z*ratioA + other.Z*ratioB

	// Attempt 2

	// angle := quat.Dot(other)
	// denom := math.Sin(angle)

	// if denom == 0 {
	// 	return newQuat
	// }

	// newQuat.X = quat.X*math.Sin((1-percent)*angle) + other.X*math.Sin(percent*angle)/denom
	// newQuat.Y = quat.Y*math.Sin((1-percent)*angle) + other.Y*math.Sin(percent*angle)/denom
	// newQuat.Z = quat.Z*math.Sin((1-percent)*angle) + other.Z*math.Sin(percent*angle)/denom
	// newQuat.W = quat.W*math.Sin((1-percent)*angle) + other.W*math.Sin(percent*angle)/denom

	// Attempt 1

	// angle := quat.W*other.W + quat.X + other.X + quat.Y + other.Y + quat.Z + other.Z

	// if math.Abs(angle) >= 1 {
	// 	return newQuat
	// }

	// halfTheta := math.Acos(angle)
	// sinHalfTheta := math.Sqrt(1 - angle*angle)

	// if math.Abs(sinHalfTheta) < 0.001 {
	// 	newQuat.W = quat.W*0.5 + other.W*0.5
	// 	newQuat.X = quat.X*0.5 + other.X*0.5
	// 	newQuat.Y = quat.Y*0.5 + other.Y*0.5
	// 	newQuat.Z = quat.Z*0.5 + other.Z*0.5
	// 	return newQuat
	// }

	// ratioA := math.Sin((1-percent)*halfTheta) / sinHalfTheta
	// ratioB := math.Sin(percent*halfTheta) / sinHalfTheta

	// newQuat.W = quat.W*ratioA + other.W*ratioB
	// newQuat.X = quat.X*ratioA + other.X*ratioB
	// newQuat.Y = quat.Y*ratioA + other.Y*ratioB
	// newQuat.Z = quat.Z*ratioA + other.Z*ratioB

	return newQuat

}

func (quat *Quaternion) Dot(other *Quaternion) float64 {
	return quat.X*other.X + quat.Y*other.Y + quat.Z*other.Z + quat.W*other.W
}
