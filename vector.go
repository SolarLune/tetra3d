package tetra3d

import (
	"math"
	"strings"
)

// Much of the harder code taken with HEAVY appreciation from quartercastle: https://github.com/quartercastle/vector

// VecX represents a unit vector in the global direction of VecX on the right-handed OpenGL / Tetra3D's coordinate system (right).
var VecX = NewVector(1, 0, 0)

// VecY represents a unit vector in the global direction of VecY on the right-handed OpenGL / Tetra3D's coordinate system (upwards).
var VecY = NewVector(0, 1, 0)

// VecZ represents a unit vector in the global direction of VecZ on the right-handed OpenGL / Tetra3D's coordinate system (backwards, towards you).
var VecZ = NewVector(0, 0, 1)

// Vector represents a 3D Vector, which can be used for usual 3D applications (position, direction, velocity, etc).
// The fourth component, W, can be ignored and is used for internal Tetra3D usage.
// Any Vector functions that modify the calling Vector return copies of the modified Vector, meaning you can do method-chaining easily.
// Vectors seem to be most efficient when copied (so try not to store pointers to them if possible, as dereferencing pointers
// can be more inefficient than directly acting on data, and storing pointers moves variables to heap).
type Vector struct {
	X float64 // The X (1st) component of the Vector
	Y float64 // The Y (2nd) component of the Vector
	Z float64 // The Z (3rd) component of the Vector
	W float64 // The w (4th) component of the Vector; not used for most Vector functions
}

// NewVector creates a new Vector with the specified x, y, z, and w components. The W component is generally ignored for most purposes.
func NewVector(x, y, z float64) Vector {
	return Vector{X: x, Y: y, Z: z, W: 0}
}

// NewVectorZero creates a new "zero-ed out" Vector, with the values of 0, 0, 0, and 0 (for W).
func NewVectorZero() Vector {
	return Vector{}
}

// Clone clones the provided Vector, returning a new copy of it. This isn't necessary for a value-based struct like this.
// func (vec Vector) Clone() Vector {
// 	return vec
// }

// Add returns a copy of the calling vector, added together with the other Vector provided (ignoring the W component).
func (vs Vector) Add(other Vector) Vector {
	vs.X += other.X
	vs.Y += other.Y
	vs.Z += other.Z
	return vs
}

// Sub returns a copy of the calling Vector, with the other Vector subtracted from it (ignoring the W component).
func (vec Vector) Sub(other Vector) Vector {
	vec.X -= other.X
	vec.Y -= other.Y
	vec.Z -= other.Z
	return vec
}

// Cross returns a new Vector, indicating the cross product of the calling Vector and the provided Other Vector.
// This function ignores the W component of both Vectors.
func (vec Vector) Cross(other Vector) Vector {

	ogVecY := vec.Y
	ogVecZ := vec.Z

	vec.Z = vec.X*other.Y - other.X*vec.Y
	vec.Y = ogVecZ*other.X - other.Z*vec.X
	vec.X = ogVecY*other.Z - other.Y*ogVecZ

	// It's possible Cross can fail
	// if vec.LengthSquared() < 0.0001 {
	// 	return vec.Cross(Y)
	// }

	return vec

}

func (vec Vector) Invert() Vector {
	vec.X = -vec.X
	vec.Y = -vec.Y
	vec.Z = -vec.Z
	vec.W = -vec.W
	return vec
}

// Magnitude returns the length of the Vector (ignoring the Vector's W component).
func (vec Vector) Magnitude() float64 {
	return math.Sqrt(vec.X*vec.X + vec.Y*vec.Y + vec.Z*vec.Z)
}

// MagnitudeSquared returns the squared length of the Vector (ignoring the Vector's W component); this is faster than Length() as it avoids using math.Sqrt().
func (vec Vector) MagnitudeSquared() float64 {
	return vec.X*vec.X + vec.Y*vec.Y + vec.Z*vec.Z
}

func (vec Vector) Distance(other Vector) float64 {
	return vec.Sub(other).Magnitude()
}

func (vec Vector) DistanceSquared(other Vector) float64 {
	return vec.Sub(other).MagnitudeSquared()
}

func (vec Vector) MultComp(other Vector) Vector {
	vec.X *= other.X
	vec.Y *= other.Y
	vec.Z *= other.Z
	return vec
}

// Unit returns a copy of the Vector, normalized (set to be of unit length).
// It does not alter the W component of the Vector.
func (vec Vector) Unit() Vector {
	l := vec.Magnitude()
	if l < 1e-8 {
		// If it's 0, then don't modify the vector
		return vec
	}
	vec.X, vec.Y, vec.Z = vec.X/l, vec.Y/l, vec.Z/l
	return vec
}

// Swizzle swizzles the Vector using the string provided, returning the swizzled copy.
// The string can be of length 3 ("xyz") or 4 ("xyzw").
// The string should be composed of the axes of a vector, i.e. 'x', 'y', 'z', or 'w'.
// Example: `vec := Vector{1, 2, 3}.Swizzle("zxy") // Returns a Vector of {3, 1, 2}.`
func (vec Vector) Swizzle(swizzleString string) Vector {

	if l := len(swizzleString); l < 3 || l > 4 {
		panic("Error: Can't call Vec.Swizzle() with less than 3, or greater than 4 values")
	}

	swizzleString = strings.ToLower(swizzleString)

	ogX := vec.X
	ogY := vec.Y
	ogZ := vec.Z
	ogW := vec.W
	var targetValue float64

	for i, v := range swizzleString {

		switch v {
		case 'x':
			targetValue = ogX
		case 'y':
			targetValue = ogY
		case 'z':
			targetValue = ogZ
		case 'w':
			targetValue = ogW
		}

		switch i {
		case 0:
			vec.X = targetValue
		case 1:
			vec.Y = targetValue
		case 2:
			vec.Z = targetValue
		case 3:
			vec.W = targetValue
		}

	}

	return vec

}

// SetX sets the X component in the vector to the value provided.
func (vec Vector) SetX(x float64) Vector {
	vec.X = x
	return vec
}

// SetY sets the Y component in the vector to the value provided.
func (vec Vector) SetY(y float64) Vector {
	vec.Y = y
	return vec
}

// SetZ sets the Z component in the vector to the value provided.
func (vec Vector) SetZ(z float64) Vector {
	vec.Z = z
	return vec
}

// Set sets the values in the Vector to the x, y, and z values provided.
func (vec Vector) Set(x, y, z float64) Vector {
	vec.X = x
	vec.Y = y
	vec.Z = z
	return vec
}

// Floats returns a [4]float64 array consisting of the Vector's contents.
func (vec Vector) Floats() [4]float64 {
	return [4]float64{vec.X, vec.Y, vec.Z, vec.W}
}

// Equals returns true if the two Vectors are close enough in all values (excluding W).
func (vec Vector) Equals(other Vector) bool {

	eps := 1e-8

	if math.Abs(float64(vec.X-other.X)) > eps || math.Abs(float64(vec.Y-other.Y)) > eps || math.Abs(float64(vec.Z-other.Z)) > eps {
		return false
	}

	// if !onlyXYZ && math.Abs(vec.W-other.W) > eps {
	// 	return false
	// }

	return true

}

// IsZero returns true if the values in the Vector are extremely close to 0 (excluding W).
func (vec Vector) IsZero() bool {

	eps := 1e-8

	if math.Abs(float64(vec.X)) > eps || math.Abs(float64(vec.Y)) > eps || math.Abs(float64(vec.Z)) > eps {
		return false
	}

	// if !onlyXYZ && math.Abs(vec.W-other.W) > eps {
	// 	return false
	// }

	return true

}

// Rotate returns a copy of the Vector, rotated around the Vector axis provided by the angle provided (in radians).
// The function is most efficient if passed an orthogonal, normalized axis (i.e. the X, Y, or Z constants).
// Note that this function ignores the W component of both Vectors.
func (vec Vector) Rotate(axis Vector, angle float64) Vector {

	cos, sin := math.Cos(float64(angle)), math.Sin(float64(angle))

	if axis.Equals(VecX) {
		ay, az := float64(vec.Y), float64(vec.Z)
		vec.Y = float64(ay*cos - az*sin)
		vec.Z = float64(ay*sin + az*cos)
		return vec
	}

	if axis.Equals(VecY) {
		ax, az := float64(vec.X), float64(vec.Z)
		vec.X = float64(ax*cos + az*sin)
		vec.Z = float64(-ax*sin + az*cos)
		return vec
	}

	if axis.Equals(VecZ) {
		ax, ay := float64(vec.X), float64(vec.Y)
		vec.X = float64(ax*cos - ay*sin)
		vec.Y = float64(ax*sin + ay*cos)
		return vec
	}

	u := axis.Unit()

	x := u.Cross(vec)

	d := u.Dot(vec)

	vec = vec.Add(vec.Scale(float64(cos)))
	vec = vec.Add(x.Scale(float64(sin)))
	vec = vec.Add(u.Scale(d).Scale(float64(1 - cos)))

	// u := unit(clone(axis))

	// x, _ := cross(u, a)
	// d := dot(u, a)

	// add(a, scale(a, cos))
	// add(a, scale(x, sin))
	// add(a, scale(scale(u, d), 1-cos))

	return vec

}

// Angle returns the angle between the calling Vector and the provided other Vector (ignoring the W component).
func (vec Vector) Angle(other Vector) float64 {
	angle := math.Acos(float64(vec.Unit().Dot(other.Unit())))
	return float64(angle)
}

// Scale scales a Vector by the given scalar (ignoring the W component).
func (vec Vector) Scale(scalar float64) Vector {
	vec.X *= scalar
	vec.Y *= scalar
	vec.Z *= scalar
	return vec
}

// Divide dividese a Vector by the given scalar (ignoring the W component).
func (vec Vector) Divide(scalar float64) Vector {
	vec.X /= scalar
	vec.Y /= scalar
	vec.Z /= scalar
	return vec
}

// Dot returns the dot product of a Vector and another Vector (ignoring the W component).
func (vec Vector) Dot(other Vector) float64 {
	return vec.X*other.X + vec.Y*other.Y + vec.Z*other.Z
}

// The previous iteration of a Vector implementation was this VectorFloat, which was more performant than the previously used quartercastle Vector,
// but more awkward, since we have to pass pointers into functions to modify them, but avoid actually storing these pointers to keep vectors from being allocated to heap.
// It also meant you couldn't effectively method chain (as this allocated to heap as well).
// That created an awkward system like so:

// ```
// result := vectorA.Add(&vectorB)
// result.Sub(&vectorC) // We pass pointers to arrays to avoid data being put on heap.
// ```

// It seems to be faster to simply use structs that are copied into functions. This allows us to be more efficient and do method chaining comfortably:
// `result := vectorA.Add(vectorB).Sub(vectorC) // All vectors are structs, not pointers to structs, arrays, or slices`

// The previous iteration of Vectors (VectorFloat) is presented below:

// VectorFloat represents a vector in space, as stored in a [4]float64. It has 3D vector functions like Length and LengthSquared(), but is fundamentally a 4D VectorFloat.
// type VectorFloat [3]float64

// // NewVectorFloat creates a new Vector with the specified x, y, z, and w components.
// func NewVectorFloat(x, y, z float64) VectorFloat {
// 	return VectorFloat{x, y, z}
// }

// // NewVectorFloatZero creates a new "zero-ed out" Vector, with the values of 0, 0, 0, and 1 (for W).
// func NewVectorFloatZero() VectorFloat {
// 	return NewVectorFloat(0, 0, 0)
// }

// // Clone clones the provided Vector, returning a new copy of it.
// func (vec VectorFloat) Clone() VectorFloat {
// 	return NewVectorFloat(vec[0], vec[1], vec[2])
// }

// // Add adds the provided other Vector into the existing one.
// func (vec VectorFloat) Add(other VectorFloat) {
// 	vec[0] += other[0]
// 	vec[1] += other[1]
// 	vec[2] += other[2]
// }

// // Sub subtracts the provided other Vector into the existing one.
// func (vec VectorFloat) Sub(other VectorFloat) VectorFloat {
// 	vec[0] -= other[0]
// 	vec[1] -= other[1]
// 	vec[2] -= other[2]
// 	return vec
// }

// // Length returns the length of the Vector.
// func (vec VectorFloat) Length() float64 {
// 	return float64(math.Sqrt(float64(vec[0]*vec[0] + vec[1]*vec[1] + vec[2]*vec[2])))
// }

// // LengthSquared returns the squared length of the Vector; this is faster than Length().
// func (vec VectorFloat) LengthSquared() float64 {
// 	return vec[0]*vec[0] + vec[1]*vec[1] + vec[2]*vec[2]
// }

// // Normalize normalizes the Vector (sets it to be of unit length).
// func (vec VectorFloat) Normalize() VectorFloat {
// 	l := vec.Length()
// 	if l < 1e-8 {
// 		// If it's 0, then remove it
// 		return vec
// 	}
// 	vec[0], vec[1], vec[2] = vec[0]/l, vec[1]/l, vec[2]/l
// 	return vec
// }

// // Swizzle swizzles the Vector
// func (vec VectorFloat) Swizzle(indices ...int) VectorFloat {

// 	if len(indices) < 3 {
// 		panic("Error: Can't call Vec.Swizzle() with less than 3 values")
// 	}

// 	nv := NewVectorFloatZero()

// 	for _, i := range indices {
// 		nv[i] = vec[i]
// 	}

// 	return nv

// }

// // Set3d sets the values in the Vector to the x, y, z, and w values provided. If you intend to use this Vector as a 3D Vector, then you
// // can just ignore the w value.
// func (vec VectorFloat) Set3d(x, y, z float64) {
// 	vec[0] = x
// 	vec[1] = y
// 	vec[2] = z
// }

// // Set4d sets the values in the Vector to the x, y, z, and w values provided. If you intend to use this Vector as a 3D Vector, then you
// // can just ignore the w value.
// func (vec VectorFloat) Set4d(x, y, z, w float64) {
// 	vec[0] = x
// 	vec[1] = y
// 	vec[2] = z
// }

// // SetVec sets the values in the Vector to the ones in the other Vector provided.
// func (vec VectorFloat) SetVec(other VectorFloat) {
// 	for i := range vec {
// 		vec[i] = other[i]
// 	}
// }

// // Cross works for the X, Y, and Z values only of a Vector
// func (vec VectorFloat) Cross(other VectorFloat) {

// 	ogVec1 := vec[1]
// 	ogVec2 := vec[2]

// 	vec[2] = vec[0]*other[1] - other[0]*vec[1]
// 	vec[1] = ogVec2*other[0] - other[2]*vec[0]
// 	vec[0] = ogVec1*other[2] - other[1]*ogVec2

// }

// func (vec VectorFloat) Floats() [3]float64 {
// 	return [3]float64(*vec)
// }

// func (vec VectorFloat) Rotate(axis VectorFloat, angle float64) {

// 	u := axis.Clone()
// 	u.Normalize()

// 	x := u.Clone()
// 	x.Cross(&vec)

// 	d := u.Dot(vec)

// 	c, s := math.Cos(float64(angle)), math.Sin(float64(angle))
// 	cos := float64(c)
// 	sin := float64(s)

// 	vec.Scale(cos)

// 	x.Scale(sin)
// 	vec.Add(&x)

// 	ud := u.Clone()
// 	ud.Scale(d)

// 	// vec.Add()

// 	// add(a, scale(a, cos))
// 	// add(a, scale(x, sin))
// 	// add(a, scale(scale(u, d), 1-cos))

// }

// func (vec VectorFloat) Scale(scalar float64) VectorFloat {
// 	vec[0] *= scalar
// 	vec[1] *= scalar
// 	vec[2] *= scalar
// 	return *vec
// }

// func (vec VectorFloat) Dot(other VectorFloat) float64 {
// 	return vec[0]*other[0] + vec[1]*other[1] + vec[2]*other[2]
// }

// // func BenchmarkAllocateArrays(b *testing.B) {

// // 	testMax := 100_000
// // 	vecs := make([][3]float64, 0, testMax)
// // 	for i := 0; i < testMax; i++ {
// // 		vecs = append(vecs, [3]float64{0, 0, 0})
// // 	}

// // }
