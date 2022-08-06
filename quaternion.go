package tetra3d

import (
	"math"
)

// Quaternion is a tool to rotate objects, similar to rotation Matrix4s. However, a difference is that they can very easily be lerped without
// losing data - if you were to lerp two rotation matrices, you can easily end up with a zero matrix, making your rotating object disappear.
// Instead, you can create the two Quaternions you need (either from Matrix4s or directly), and then lerp them together.
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

func (q1 *Quaternion) Mult(q2 *Quaternion) *Quaternion {
	// Cribbed from euclidean space: http://www.euclideanspace.com/maths/algebra/realNormedAlgebra/quaternions/code/index.htm#mul
	return NewQuaternion(
		q1.X*q2.W+q1.Y*q2.Z-q1.Z*q2.Y+q1.W*q2.X,
		-q1.X*q2.Z+q1.Y*q2.W+q1.Z*q2.X+q1.W*q2.Y,
		q1.X*q2.Y-q1.Y*q2.X+q1.Z*q2.W+q1.W*q2.Z,
		-q1.X*q2.X-q1.Y*q2.Y-q1.Z*q2.Z+q1.W*q2.W,
	)
}

// ToMatrix4 generates a rotation Matrx4 from the given Quaternion.
func (quat *Quaternion) ToMatrix4() Matrix4 {

	// See this page for where this formula comes from: https://www.euclideanspace.com/maths/geometry/rotations/conversions/quaternionToMatrix/jay.htm

	m1 := NewMatrix4()
	m1[0][0] = quat.W
	m1[0][1] = quat.Z
	m1[0][2] = -quat.Y
	m1[0][3] = quat.X

	m1[1][0] = -quat.Z
	m1[1][1] = quat.W
	m1[1][2] = quat.X
	m1[1][3] = quat.Y

	m1[2][0] = quat.Y
	m1[2][1] = -quat.X
	m1[2][2] = quat.W
	m1[2][3] = quat.Z

	m1[3][0] = -quat.X
	m1[3][1] = -quat.Y
	m1[3][2] = -quat.Z
	m1[3][3] = quat.W

	m2 := NewMatrix4()

	m2[0][0] = quat.W
	m2[0][1] = quat.Z
	m2[0][2] = -quat.Y
	m2[0][3] = -quat.X

	m2[1][0] = -quat.Z
	m2[1][1] = quat.W
	m2[1][2] = quat.X
	m2[1][3] = -quat.Y

	m2[2][0] = quat.Y
	m2[2][1] = -quat.X
	m2[2][2] = quat.W
	m2[2][3] = -quat.Z

	m2[3][0] = quat.X
	m2[3][1] = quat.Y
	m2[3][2] = quat.Z
	m2[3][3] = quat.W

	return m1.Mult(m2)
}

// func NewLookAtQuaternion(from, to, up vector.Vector) *Quaternion {
// 	// Cribbed from StackOverflow: https://stackoverflow.com/questions/12435671/quaternion-lookat-function

// 	forward := to.Sub(from).Unit()
// 	globalForward := vector.Z.Invert()

// 	dot := globalForward.Dot(forward)

// 	if math.Abs(dot+1.0) < 0.000001 {
// 		return NewQuaternion(vector.Y[0], vector.Y[1], vector.Y[2], math.Pi)
// 	}

// 	if math.Abs(dot-1.0) < 0.000001 {
// 		return NewQuaternion(0, 0, 0, 0)
// 	}

// 	rotAngle := math.Acos(dot)
// 	rotAxis, _ := globalForward.Cross(forward)
// 	vector.In(rotAxis).Unit()
// 	return NewQuaternionFromAxisAngle(rotAxis, rotAngle)
// }

// func NewQuaternionFromAxisAngle(axis vector.Vector, angle float64) *Quaternion {
// 	// Also cribbed from StackOverflow whoops
// 	halfAngle := angle * .5
// 	s := math.Sin(halfAngle)
// 	return NewQuaternion(
// 		axis[0]*s,
// 		axis[1]*s,
// 		axis[2]*s,
// 		math.Cos(halfAngle),
// 	)
// }

// func (quat *Quaternion) Add(other *Quaternion) *Quaternion {

// 	newQuat := quat.Clone()
// 	newQuat.X *= other.X
// 	newQuat.Y *= other.Y
// 	newQuat.Z *= other.Z
// 	newQuat.W *= other.W
// 	newQuat = newQuat.Normalized()
// 	return newQuat

// }

// func NewLookAtQuaternion(from, to, up vector.Vector) *Quaternion {
// 	fmt.Println("mat: ", NewLookAtMatrix(from, to, up))
// 	return NewLookAtMatrix(from, to, up).ToQuaternion()
// }

// func NewLookAtQuaternion(from, to, up vector.Vector) *Quaternion {

// 	forward := to.Sub(from).Unit()
// 	right, _ := forward.Cross(up)
// 	localUp, _ := right.Cross(forward)

// 	q := NewQuaternion(0, 0, 0, 0)

// 	trace := right[0] + localUp[1] + forward[2]

// 	if trace > 0 {

// 		s := 0.5 / math.Sqrt(trace+1.0)
// 		q.W = 0.25 / s
// 		q.X = (localUp[1] - forward[2]) * s
// 		q.Z = (forward[0] - right[1]) * s
// 		q.Y = (right[2] - localUp[0]) * s

// 	} else {

// 		if right[0] > localUp[2] && right[0] > forward[1] {
// 			s := 2.0 * math.Sqrt(1.0+right[0]-localUp[2]-forward[1])
// 			q.W = (localUp[1] - forward[2]) / s
// 			q.X = 0.25 * s
// 			q.Y = (localUp[0] + right[2]) / s
// 			q.Z = (forward[0] + right[1]) / s
// 		} else if localUp[2] > forward[1] {
// 			s := 2.0 * math.Sqrt(1.0+localUp[2]-right[0]-forward[1])
// 			q.W = (forward[0] - right[1]) / s
// 			q.X = (localUp[0] + right[2]) / s
// 			q.Y = 0.25 * s
// 			q.Z = (forward[2] + localUp[1]) / s
// 		} else {
// 			s := 2.0 * math.Sqrt(1.0+forward[1]-right[0]-localUp[2])
// 			q.W = (right[2] - localUp[0]) / s
// 			q.X = (forward[0] + right[1]) / s
// 			q.Y = (forward[2] + localUp[1]) / s
// 			q.Z = 0.25 * s
// 		}

// 	}

// 	fmt.Println(trace, q)

// 	return q

// 	// your code from before
// 	// F = normalize(target - camera);   // lookAt
// 	// R = normalize(cross(F, worldUp)); // sideaxis
// 	// U = cross(R, F);                  // rotatedup

// 	// // note that R needed to be re-normalized
// 	// // since F and worldUp are not necessary perpendicular
// 	// // so must remove the sin(angle) factor of the cross-product
// 	// // same not true for U because dot(R, F) = 0

// 	// // adapted source
// 	// Quaternion q;
// 	// double trace = R.x + U.y + F.z;
// 	// if (trace > 0.0) {
// 	//   double s = 0.5 / sqrt(trace + 1.0);
// 	//   q.w = 0.25 / s;
// 	//   q.x = (U.z - F.y) * s;
// 	//   q.y = (F.x - R.z) * s;
// 	//   q.z = (R.y - U.x) * s;
// 	// } else {
// 	//   if (R.x > U.y && R.x > F.z) {
// 	//     double s = 2.0 * sqrt(1.0 + R.x - U.y - F.z);
// 	//     q.w = (U.z - F.y) / s;
// 	//     q.x = 0.25 * s;
// 	//     q.y = (U.x + R.y) / s;
// 	//     q.z = (F.x + R.z) / s;
// 	//   } else if (U.y > F.z) {
// 	//     double s = 2.0 * sqrt(1.0 + U.y - R.x - F.z);
// 	//     q.w = (F.x - R.z) / s;
// 	//     q.x = (U.x + R.y) / s;
// 	//     q.y = 0.25 * s;
// 	//     q.z = (F.y + U.z) / s;
// 	//   } else {
// 	//     double s = 2.0 * sqrt(1.0 + F.z - R.x - U.y);
// 	//     q.w = (R.y - U.x) / s;
// 	//     q.x = (F.x + R.z) / s;
// 	//     q.y = (F.y + U.z) / s;
// 	//     q.z = 0.25 * s;
// 	//   }
// 	// }

// }

// func NewLookAtQuaternion(from, to, up vector.Vector) *Quaternion {
// 	// Cribbed from StackOverflow: https://stackoverflow.com/questions/12435671/quaternion-lookat-function
// 	diff := to.Sub(from).Unit()

// 	forward := vector.Z
// 	rotAxis, _ := forward.Cross(diff)

// 	vector.In(rotAxis).Unit()

// 	if rotAxis.Magnitude() == 0 {
// 		rotAxis = up
// 	}

// 	dot := forward.Dot(diff)
// 	angle := math.Acos(dot)

// 	return NewQuaternionFromAxisAngle(rotAxis, angle)

// }

// func NewQuaternionFromAxisAngle(axis vector.Vector, angle float64) *Quaternion {
// 	// Also cribbed from the same StackOverflow site whoops
// 	s := math.Sin(angle / 2)
// 	axis = axis.Unit()
// 	return NewQuaternion(axis[0]*s, axis[1]*s, axis[2]*s, math.Cos(angle/2))

// }

// Quaternion lookAt(const Vector3f& sourcePoint, const Vector3f& destPoint, const Vector3f& front, const Vector3f& up)
// {
//     Vector3f toVector = (destPoint - sourcePoint).normalized();

//     //compute rotation axis
//     Vector3f rotAxis = front.cross(toVector).normalized();
//     if (rotAxis.squaredNorm() == 0)
//         rotAxis = up;

//     //find the angle around rotation axis
//     float dot = VectorMath::front().dot(toVector);
//     float ang = std::acosf(dot);

//     //convert axis angle to quaternion
//     return Eigen::AngleAxisf(rotAxis, ang);
// }

// Bove uses popular Eigen library. If you don't want to use that then you might need following replacement for Eigen::AngleAxisf:

// //Angle-Axis to Quaternion
// Quaternionr angleAxisf(const Vector3r& axis, float angle) {
//     auto s = std::sinf(angle / 2);
//     auto u = axis.normalized();
//     return Quaternionr(std::cosf(angle / 2), u.x() * s, u.y() * s, u.z() * s);
// }
