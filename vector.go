package tetra3d

import (
	"fmt"
	"math"
	"strings"
)

// Much of the harder code taken with HEAVY appreciation from quartercastle: https://github.com/quartercastle/vector

// WorldRight represents a unit vector in the global direction of WorldRight on the right-handed OpenGL / Tetra3D's coordinate system (+X).
var WorldRight = NewVector(1, 0, 0)

// WorldLeft represents a unit vector in the global direction of WorldLeft on the right-handed OpenGL / Tetra3D's coordinate system (-X).
var WorldLeft = WorldRight.Invert()

// WorldUp represents a unit vector in the global direction of WorldUp on the right-handed OpenGL / Tetra3D's coordinate system (+Y).
var WorldUp = NewVector(0, 1, 0)

// WorldDown represents a unit vector in the global direction of WorldDown on the right-handed OpenGL / Tetra3D's coordinate system (+Y).
var WorldDown = WorldUp.Invert()

// WorldBack represents a unit vector in the global direction of WorldBack on the right-handed OpenGL / Tetra3D's coordinate system (+Z).
var WorldBack = NewVector(0, 0, 1)

// WorldForward represents a unit vector in the global direction of WorldForward on the right-handed OpenGL / Tetra3D's coordinate system (-Z).
var WorldForward = WorldBack.Invert()

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

// NewVector creates a new Vector with the specified x, y, and z components. The W component is generally ignored for most purposes.
func NewVector(x, y, z float64) Vector {
	return Vector{X: x, Y: y, Z: z, W: 0}
}

// NewVector creates a new Vector with the specified x and y components. The W and Z components are set to 0.
func NewVector2d(x, y float64) Vector {
	return Vector{X: x, Y: y, Z: 0, W: 0}
}

// NewVectorZero creates a new "zero-ed out" Vector, with the values of 0, 0, 0, and 0 (for W).
func NewVectorZero() Vector {
	return Vector{}
}

// Modify returns a ModVector object (a pointer to the original vector).
func (vec *Vector) Modify() ModVector {
	ip := ModVector{Vector: vec}
	return ip
}

// Clone clones the provided Vector, returning a new copy of it. This isn't necessary for a value-based struct like this.
// func (vec Vector) Clone() Vector {
// 	return vec
// }

// String returns a string representation of the Vector, excluding its W component (which is primarily used for internal purposes).
func (vec Vector) String() string {
	return fmt.Sprintf("{%.2f, %.2f, %.2f}", vec.X, vec.Y, vec.Z)
}

// String returns a string representation of the Vector, excluding its W component (which is primarily used for internal purposes).
func (vec Vector) StringW() string {
	return fmt.Sprintf("{%.2f, %.2f, %.2f, %.2f}", vec.X, vec.Y, vec.Z, vec.W)
}

// Plus returns a copy of the calling vector, added together with the other Vector provided (ignoring the W component).
func (vec Vector) Add(other Vector) Vector {
	vec.X += other.X
	vec.Y += other.Y
	vec.Z += other.Z
	return vec
}

// Sub returns a copy of the calling Vector, with the other Vector subtracted from it (ignoring the W component).
func (vec Vector) Sub(other Vector) Vector {
	vec.X -= other.X
	vec.Y -= other.Y
	vec.Z -= other.Z
	return vec
}

// Expand expands the Vector by the margin specified, in absolute units, if each component is over the minimum argument.
// To illustrate: Given a Vector of {1, 0.1, -0.3}, Vector.Expand(0.5, 0.2) would give you a Vector of {1.5, 0.1, -0.8}.
// This function returns a copy of the Vector with the result.
func (vec Vector) Expand(margin, min float64) Vector {
	if vec.X > min || vec.X < -min {
		vec.X += math.Copysign(margin, vec.X)
	}
	if vec.Y > min || vec.Y < -min {
		vec.Y += math.Copysign(margin, vec.Y)
	}
	if vec.Z > min || vec.Z < -min {
		vec.Z += math.Copysign(margin, vec.Z)
	}
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

// Invert returns a copy of the Vector with all components inverted (ignoring the Vector's W component).
func (vec Vector) Invert() Vector {
	vec.X = -vec.X
	vec.Y = -vec.Y
	vec.Z = -vec.Z
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

// ClampMagnitude clamps the overall magnitude of the Vector to the maximum magnitude specified, returning a copy with the result.
// If the magnitude is less than that maximum magnitude, the vector is unmodified.
func (vec Vector) ClampMagnitude(maxMag float64) Vector {
	if vec.Magnitude() > maxMag {
		vec = vec.Unit().Scale(maxMag)
	}
	return vec
}

// SubMagnitude subtacts the given magnitude from the Vector's. If the vector's magnitude is less than the given magnitude to subtract,
// a zero-length Vector will be returned.
func (vec Vector) SubMagnitude(mag float64) Vector {
	if vec.Magnitude() > mag {
		return vec.Sub(vec.Unit().Scale(mag))
	}
	return Vector{0, 0, 0, vec.W}

}

// Distance returns the distance from the calling Vector to the other Vector provided.
func (vec Vector) Distance(other Vector) float64 {
	return vec.Sub(other).Magnitude()
}

// Distance returns the squared distance from the calling Vector to the other Vector provided. This is faster than Distance(), as it avoids using math.Sqrt().
func (vec Vector) DistanceSquared(other Vector) float64 {
	return vec.Sub(other).MagnitudeSquared()
}

// Mult performs Hadamard (component-wise) multiplication on the calling Vector with the other Vector provided, returning a copy with the result (and ignoring the Vector's W component).
func (vec Vector) Mult(other Vector) Vector {
	vec.X *= other.X
	vec.Y *= other.Y
	vec.Z *= other.Z
	return vec
}

// Unit returns a copy of the Vector, normalized (set to be of unit length).
// It does not alter the W component of the Vector.
func (vec Vector) Unit() Vector {
	l := vec.Magnitude()
	if l < 1e-8 || l == 1 {
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

	eps := 1e-4

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

	eps := 1e-4

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
func (vec Vector) RotateVec(axis Vector, angle float64) Vector {
	return NewQuaternionFromAxisAngle(axis, angle).RotateVec(vec)
}

// Rotate returns a copy of the Vector, rotated around an axis Vector with the x, y, and z components provided, by the angle
// provided (in radians), counter-clockwise.
// The function is most efficient if passed an orthogonal, normalized axis (i.e. the X, Y, or Z constants).
// Note that this function ignores the W component of both Vectors.
func (vec Vector) Rotate(x, y, z, angle float64) Vector {
	return NewQuaternionFromAxisAngle(Vector{X: x, Y: y, Z: z}, angle).RotateVec(vec)
}

// Angle returns the angle between the calling Vector and the provided other Vector (ignoring the W component).
func (vec Vector) Angle(other Vector) float64 {
	d := vec.Unit().Dot(other.Unit())
	d = clamp(d, -1, 1) // Acos returns NaN if value < -1 or > 1
	return math.Acos(float64(d))
}

// Scale scales a Vector by the given scalar (ignoring the W component), returning a copy with the result.
func (vec Vector) Scale(scalar float64) Vector {
	vec.X *= scalar
	vec.Y *= scalar
	vec.Z *= scalar
	return vec
}

// Divide divides a Vector by the given scalar (ignoring the W component), returning a copy with the result.
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

// Snap snaps the Vector's components to the given space in world units, returning a clone (e.g. Vector{0.1, 1.27, 3.33}.Snap(0.25) will return Vector{0, 1.25, 3.25}).
func (vec Vector) Snap(snapToUnits float64) Vector {
	vec.X = round(vec.X/snapToUnits) * snapToUnits
	vec.Y = round(vec.Y/snapToUnits) * snapToUnits
	vec.Z = round(vec.Z/snapToUnits) * snapToUnits
	return vec
}

// ModVector represents a reference to a Vector, made to facilitate easy method-chaining and modifications on that Vector (as you
// don't need to re-assign the results of a chain of operations to the original variable to "save" the results).
// Note that a ModVector is not meant to be used to chain methods on a vector to pass directly into a function; you can just
// use the normal vector functions for that purpose. ModVectors are pointers, which are allocated to the heap. This being the case,
// they should be slower relative to normal Vectors, so use them only in non-performance-critical parts of your application.
type ModVector struct {
	*Vector
}

// Add adds the other Vector provided to the ModVector.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Add(other Vector) ModVector {
	ip.X += other.X
	ip.Y += other.Y
	ip.Z += other.Z
	return ip
}

// Sub subtracts the other Vector from the calling ModVector.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Sub(other Vector) ModVector {
	ip.X -= other.X
	ip.Y -= other.Y
	ip.Z -= other.Z
	return ip
}

func (ip ModVector) SetZero() ModVector {
	ip.X = 0
	ip.Y = 0
	ip.Z = 0
	return ip
}

// Scale scales the Vector by the scalar provided.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Scale(scalar float64) ModVector {
	ip.X *= scalar
	ip.Y *= scalar
	ip.Z *= scalar
	return ip
}

// Divide divides a Vector by the given scalar (ignoring the W component).
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Divide(scalar float64) ModVector {
	ip.X /= scalar
	ip.Y /= scalar
	ip.Z /= scalar
	return ip
}

// Expand expands the ModVector by the margin specified, in absolute units, if each component is over the minimum argument.
// To illustrate: Given a ModVector of {1, 0.1, -0.3}, ModVector.Expand(0.5, 0.2) would give you a ModVector of {1.5, 0.1, -0.8}.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Expand(margin, min float64) ModVector {
	exp := ip.Vector.Expand(margin, min)
	ip.X = exp.X
	ip.Y = exp.Y
	ip.Z = exp.Z
	return ip
}

// Mult performs Hadamard (component-wise) multiplication with the Vector on the other Vector provided.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Mult(other Vector) ModVector {
	ip.X *= other.X
	ip.Y *= other.Y
	ip.Z *= other.Z
	return ip
}

// Unit normalizes the ModVector (sets it to be of unit length).
// It does not alter the W component of the Vector.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Unit() ModVector {
	l := ip.Magnitude()
	if l < 1e-8 || l == 1 {
		// If it's 0, then don't modify the vector
		return ip
	}
	ip.X, ip.Y, ip.Z = ip.X/l, ip.Y/l, ip.Z/l
	return ip
}

// Swizzle swizzles the ModVector using the string provided.
// The string can be of length 3 ("xyz") or 4 ("xyzw").
// The string should be composed of the axes of a vector, i.e. 'x', 'y', 'z', or 'w'.
// Example: `vec := Vector{1, 2, 3}.Swizzle("zxy") // Returns a Vector of {3, 1, 2}.`
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Swizzle(swizzleString string) ModVector {
	vec := ip.Vector.Swizzle(swizzleString)
	ip.X = vec.X
	ip.Y = vec.Y
	ip.Z = vec.Z
	return ip
}

// Cross performs a cross-product multiplication on the ModVector.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Cross(other Vector) ModVector {
	cross := (*ip.Vector).Cross(other)
	ip.X = cross.X
	ip.Y = cross.Y
	ip.Z = cross.Z
	return ip
}

// RotateVec rotates the calling Vector by the axis Vector and angle provided (in radians).
// This function returns the calling ModVector for method chaining.
func (ip ModVector) RotateVec(axis Vector, angle float64) ModVector {
	rot := (*ip.Vector).RotateVec(axis, angle)
	ip.X = rot.X
	ip.Y = rot.Y
	ip.Z = rot.Z
	return ip
}

// RotateVec rotates the calling Vector by an axis Vector composed of the x, y, and z components provided,
// by the angle provided (in radians).
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Rotate(x, y, z, angle float64) ModVector {
	rot := (*ip.Vector).Rotate(x, y, z, angle)
	ip.X = rot.X
	ip.Y = rot.Y
	ip.Z = rot.Z
	return ip
}

// Invert inverts all components of the calling Vector.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Invert() ModVector {
	ip.X = -ip.X
	ip.Y = -ip.Y
	ip.Z = -ip.Z
	return ip
}

// Snap snaps the Vector's components to the given space in world units, returning a clone (e.g. Vector{0.1, 1.27, 3.33}.Snap(0.25) will return Vector{0, 1.25, 3.25}).
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Snap(snapToUnits float64) ModVector {
	snapped := (*ip.Vector).Snap(snapToUnits)
	ip.X = snapped.X
	ip.Y = snapped.Y
	ip.Z = snapped.Z
	return ip
}

// ClampMagnitude clamps the overall magnitude of the Vector to the maximum magnitude specified.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) ClampMagnitude(maxMag float64) ModVector {
	clamped := (*ip.Vector).ClampMagnitude(maxMag)
	ip.X = clamped.X
	ip.Y = clamped.Y
	ip.Z = clamped.Z
	return ip
}

// SubMagnitude subtacts the given magnitude from the Vector's. If the vector's magnitude is less than the given magnitude to subtract,
// a zero-length Vector will be returned.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) SubMagnitude(mag float64) ModVector {
	if ip.Magnitude() > mag {
		ip.Sub(ip.Vector.Unit().Scale(mag))
	} else {
		ip.X = 0
		ip.Y = 0
		ip.Z = 0
	}
	return ip

}

// String converts the ModVector to a string. Because it's a ModVector, it's represented with a *.
func (ip ModVector) String() string {
	return fmt.Sprintf("*{%.2f, %.2f, %.2f}", ip.X, ip.Y, ip.Z)
}

// Clone returns a ModVector of a clone of its backing Vector.
// This function returns the calling ModVector for method chaining.
func (ip ModVector) Clone() ModVector {
	v := *ip.Vector
	return v.Modify()
}

func (ip ModVector) ToVector() Vector {
	return *ip.Vector
}

// The previous iteration of a Vector implementation was this VectorFloat, which was more performant than the previously used quartercastle's Vector package,
// but also more awkward, since we have to pass pointers into functions to modify them, but avoid actually storing these pointers to keep vectors from being allocated to heap.
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
