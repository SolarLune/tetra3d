package tetra3d

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/solarlune/tetra3d/math32"
)

// Much of the harder code taken with HEAVY appreciation from quartercastle: https://github.com/quartercastle/vector

// WorldRight represents a unit vector in the global direction of rightward on the right-handed OpenGL / Tetra3D's coordinate system (+X).
var WorldRight = NewVector3(1, 0, 0)

// WorldLeft represents a unit vector in the global direction of leftward on the right-handed OpenGL / Tetra3D's coordinate system (-X).
var WorldLeft = WorldRight.Invert()

// WorldUp represents a unit vector in the global direction of upward on the right-handed OpenGL / Tetra3D's coordinate system (+Y).
var WorldUp = NewVector3(0, 1, 0)

// WorldDown represents a unit vector in the global direction of downward on the right-handed OpenGL / Tetra3D's coordinate system (+Y).
var WorldDown = WorldUp.Invert()

// WorldBackward represents a unit vector in the global direction of backwards (towards the camera, assuming no rotations) on the right-handed OpenGL / Tetra3D's coordinate system (+Z).
var WorldBackward = NewVector3(0, 0, 1)

// WorldForward represents a unit vector in the global direction of forwards (away from the camera, assuming no rotations) on the right-handed OpenGL / Tetra3D's coordinate system (-Z).
var WorldForward = WorldBackward.Invert()

// Vector3 represents a 3D Vector3, which can be used for usual 3D applications (position, direction, velocity, etc).
// The fourth component, W, can be ignored and is used for internal Tetra3D usage.
// Any Vector3 functions that modify the calling Vector3 return copies of the modified Vector3, meaning you can do method-chaining easily.
// Vectors seem to be most efficient when copied (so try not to store pointers to them if possible, as dereferencing pointers
// can be more inefficient than directly acting on data, and storing pointers moves variables to heap).
type Vector3 struct {
	X float32 // The X (1st) component of the Vector
	Y float32 // The Y (2nd) component of the Vector
	Z float32 // The Z (3rd) component of the Vector
}

// NewVector3 creates a new Vector3 with the specified x, y, and z components. The W component is generally ignored for most purposes.
func NewVector3(x, y, z float32) Vector3 {
	return Vector3{X: x, Y: y, Z: z}
}

// NewVector3Random creates a new random vector with components ranging between the minimum and maximum values provided.
func NewVector3Random(minX, maxX, minY, maxY, minZ, maxZ float32) Vector3 {
	dx := maxX - minX
	dy := maxY - minY
	dz := maxZ - minZ
	return Vector3{
		X: minX + rand.Float32()*dx,
		Y: minY + rand.Float32()*dy,
		Z: minZ + rand.Float32()*dz,
	}
}

// Modify returns a ModVector3 object (a pointer to the original vector).
func (vec *Vector3) Modify() ModVector3 {
	return ModVector3{Vector3: vec}
}

// Clone clones the provided Vector, returning a new copy of it. This isn't necessary for a value-based struct like this.
// func (vec Vector) Clone() Vector3 {
// 	return vec
// }

// String returns a string representation of the Vector, excluding its W component (which is primarily used for internal purposes).
func (vec Vector3) String() string {
	return fmt.Sprintf("{%.2f, %.2f, %.2f}", vec.X, vec.Y, vec.Z)
}

// String returns a string representation of the Vector, excluding its W component (which is primarily used for internal purposes).
func (vec Vector4) String() string {
	return fmt.Sprintf("{%.2f, %.2f, %.2f, %.2f}", vec.X, vec.Y, vec.Z, vec.W)
}

// Add returns a copy of the calling vector, added together with the other Vector3 provided (ignoring the W component).
func (vec Vector3) Add(other Vector3) Vector3 {
	vec.X += other.X
	vec.Y += other.Y
	vec.Z += other.Z
	return vec
}

// AddX returns a copy of the calling vector, with the value provided added to the X component.
func (vec Vector3) AddX(value float32) Vector3 {
	vec.X += value
	return vec
}

// AddY returns a copy of the calling vector, with the value provided added to the Y component.
func (vec Vector3) AddY(value float32) Vector3 {
	vec.Y += value
	return vec
}

// AddZ returns a copy of the calling vector, with the value provided added to the Z component.
func (vec Vector3) AddZ(value float32) Vector3 {
	vec.Z += value
	return vec
}

// Sub returns a copy of the calling Vector, with the other Vector3 subtracted from it (ignoring the W component).
func (vec Vector3) Sub(other Vector3) Vector3 {
	vec.X -= other.X
	vec.Y -= other.Y
	vec.Z -= other.Z
	return vec
}

// SubX returns a copy of the calling vector, with the value provided subtracted from the X component.
func (vec Vector3) SubX(value float32) Vector3 {
	vec.X -= value
	return vec
}

// SubY returns a copy of the calling vector, with the value provided subtracted from the Y component.
func (vec Vector3) SubY(value float32) Vector3 {
	vec.Y -= value
	return vec
}

// SubZ returns a copy of the calling vector, with the value provided subtracted from the Z component.
func (vec Vector3) SubZ(value float32) Vector3 {
	vec.Z -= value
	return vec
}

// Expand expands the Vector3 by the margin specified, in absolute units, if each component is over the minimum argument.
// To illustrate: Given a Vector3 of {1, 0.1, -0.3}, Vector.Expand(0.5, 0.2) would give you a Vector3 of {1.5, 0.1, -0.8}.
// This function returns a copy of the Vector3 with the result.
func (vec Vector3) Expand(margin, min float32) Vector3 {

	if vec.X > min || vec.X < -min {
		vec.X += math32.Copysign(margin, vec.X)
	}
	if vec.Y > min || vec.Y < -min {
		vec.Y += math32.Copysign(margin, vec.Y)
	}
	if vec.Z > min || vec.Z < -min {
		vec.Z += math32.Copysign(margin, vec.Z)
	}
	return vec
}

// Cross returns a new Vector, indicating the cross product of the calling Vector3 and the provided Other Vector.
// This function ignores the W component of both Vectors.
func (vec Vector3) Cross(other Vector3) Vector3 {

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

// Invert returns a copy of the Vector3 with all components inverted (ignoring the Vector's W component).
func (vec Vector3) Invert() Vector3 {
	vec.X = -vec.X
	vec.Y = -vec.Y
	vec.Z = -vec.Z
	return vec
}

// Magnitude returns the length of the Vector3 (ignoring the Vector's W component).
func (vec Vector3) Magnitude() float32 {
	return math32.Sqrt(vec.X*vec.X + vec.Y*vec.Y + vec.Z*vec.Z)
}

// MagnitudeSquared returns the squared length of the Vector3 (ignoring the Vector's W component); this is faster than Length() as it avoids using math32.Sqrt().
func (vec Vector3) MagnitudeSquared() float32 {
	return vec.X*vec.X + vec.Y*vec.Y + vec.Z*vec.Z
}

// ClampMagnitude clamps the overall magnitude of the Vector3 to the maximum magnitude specified, returning a copy with the result.
// If the magnitude is less than that maximum magnitude, the vector is unmodified.
func (vec Vector3) ClampMagnitude(maxMag float32) Vector3 {
	if vec.Magnitude() > maxMag {
		vec = vec.Unit().Scale(maxMag)
	}
	return vec
}

// SubMagnitude returns a copy of the Vector3 with the given magnitude subtracted from it. If the vector's magnitude is less than the given magnitude to subtract,
// a zero-length Vector3 will be returned.
func (vec Vector3) SubMagnitude(mag float32) Vector3 {
	if vec.Magnitude() > mag {
		return vec.Sub(vec.Unit().Scale(mag))
	}
	return Vector3{0, 0, 0}

}

// MoveTowards moves a Vector3 towards another Vector3 given a specific magnitude. If the distance is less than that magnitude, it returns the target vector.
func (vec Vector3) MoveTowards(target Vector3, magnitude float32) Vector3 {
	diff := target.Sub(vec)
	if diff.Magnitude() > magnitude {
		return vec.Add(diff.Unit().Scale(magnitude))
	}
	return target
}

// DistanceTo returns the distance from the calling Vector3 to the other Vector3 provided.
func (vec Vector3) DistanceTo(other Vector3) float32 {
	// return vec.Sub(other).Magnitude()
	vec.X -= other.X
	vec.Y -= other.Y
	vec.Z -= other.Z
	return math32.Sqrt(vec.X*vec.X + vec.Y*vec.Y + vec.Z*vec.Z)
}

// DistanceSquaredTo returns the squared distance from the calling Vector3 to the other Vector3 provided. This is faster than Distance(), as it avoids using math32.Sqrt().
func (vec Vector3) DistanceSquaredTo(other Vector3) float32 {
	vec.X -= other.X
	vec.Y -= other.Y
	vec.Z -= other.Z
	return vec.X*vec.X + vec.Y*vec.Y + vec.Z*vec.Z
}

// Mult performs Hadamard (component-wise) multiplication on the calling Vector3 with the other Vector3 provided, returning a copy with the result (and ignoring the Vector's W component).
func (vec Vector3) Mult(other Vector3) Vector3 {
	vec.X *= other.X
	vec.Y *= other.Y
	vec.Z *= other.Z
	return vec
}

// Unit returns a copy of the Vector, normalized (set to be of unit length).
// It does not alter the W component of the Vector.
func (vec Vector3) Unit() Vector3 {
	l := vec.Magnitude()
	if l < 1e-8 || l == 1 {
		// If it's 0, then don't modify the vector
		return vec
	}
	vec.X, vec.Y, vec.Z = vec.X/l, vec.Y/l, vec.Z/l
	return vec
}

// Swizzle swizzles the Vector3 using the string provided, returning the swizzled copy.
// The string can be of length 3 ("xyz") or 4 ("xyzw").
// The string should be composed of the axes of a vector, i.e. 'x', 'y', 'z', or 'w'.
// Example: `vec := Vector{1, 2, 3}.Swizzle("zxy") // Returns a Vector3 of {3, 1, 2}.`
func (vec Vector3) Swizzle(swizzleString string) Vector3 {

	if l := len(swizzleString); l < 3 || l > 4 {
		panic("Error: Can't call Vec.Swizzle() with less than 3, or greater than 4 values")
	}

	swizzleString = strings.ToLower(swizzleString)

	ogX := vec.X
	ogY := vec.Y
	ogZ := vec.Z
	var targetValue float32

	for i, v := range swizzleString {

		switch v {
		case 'x':
			targetValue = ogX
		case 'y':
			targetValue = ogY
		case 'z':
			targetValue = ogZ
		}

		switch i {
		case 0:
			vec.X = targetValue
		case 1:
			vec.Y = targetValue
		case 2:
			vec.Z = targetValue
		}

	}

	return vec

}

// SetX sets the X component in the vector to the value provided.
func (vec Vector3) SetX(x float32) Vector3 {
	vec.X = x
	return vec
}

// SetY sets the Y component in the vector to the value provided.
func (vec Vector3) SetY(y float32) Vector3 {
	vec.Y = y
	return vec
}

// SetZ sets the Z component in the vector to the value provided.
func (vec Vector3) SetZ(z float32) Vector3 {
	vec.Z = z
	return vec
}

// SetXYZ sets the values in the Vector3 to the x, y, and z values provided.
func (vec Vector3) SetXYZ(x, y, z float32) Vector3 {
	vec.X = x
	vec.Y = y
	vec.Z = z
	return vec
}

// Set sets the values in the Vector3 to the values in the other Vector3.
func (vec *Vector3) Set(other Vector3) {
	vec.X = other.X
	vec.Y = other.Y
	vec.Z = other.Z
}

// Floats returns a [4]float32 array consisting of the Vector's contents.
func (vec Vector3) Floats() [3]float32 {
	return [3]float32{vec.X, vec.Y, vec.Z}
}

// Equals returns true if the two Vectors are close enough in all values (excluding W).
func (vec Vector3) Equals(other Vector3) bool {

	eps := float32(1e-4)

	if math32.Abs(float32(vec.X-other.X)) > eps || math32.Abs(float32(vec.Y-other.Y)) > eps || math32.Abs(float32(vec.Z-other.Z)) > eps {
		return false
	}

	// if !onlyXYZ && math32.Abs(vec.W-other.W) > eps {
	// 	return false
	// }

	return true

}

// IsZero returns true if all of the values in the Vector3 are extremely close to 0 (excluding W).
func (vec Vector3) IsZero() bool {

	eps := float32(1e-4)

	if math32.Abs(float32(vec.X)) > eps || math32.Abs(float32(vec.Y)) > eps || math32.Abs(float32(vec.Z)) > eps {
		return false
	}

	// if !onlyXYZ && math32.Abs(vec.W-other.W) > eps {
	// 	return false
	// }

	return true

}

// IsNaN returns if any of the values are NaN (not a number).
func (vec Vector3) IsNaN() bool {
	return math32.IsNaN(vec.X) || math32.IsNaN(vec.Y) || math32.IsNaN(vec.Z)
}

// IsInf returns if any of the values are infinite.
func (vec Vector3) IsInf() bool {
	return math32.IsInf(vec.X, 0) || math32.IsInf(vec.Y, 0) || math32.IsInf(vec.Z, 0)
}

// Rotate returns a copy of the Vector, rotated around the Vector3 axis provided by the angle provided (in radians).
// The function is most efficient if passed an orthogonal, normalized axis (i.e. the X, Y, or Z constants).
// Note that this function ignores the W component of both Vectors.
func (vec Vector3) RotateVec(axis Vector3, angle float32) Vector3 {
	return NewQuaternionFromAxisAngle(axis, angle).RotateVec(vec)
}

// Rotate returns a copy of the Vector, rotated around an axis Vector3 with the x, y, and z components provided, by the angle
// provided (in radians), counter-clockwise.
// The function is most efficient if passed an orthogonal, normalized axis (i.e. the X, Y, or Z constants).
// Note that this function ignores the W component of both Vectors.
func (vec Vector3) Rotate(x, y, z, angle float32) Vector3 {
	return NewQuaternionFromAxisAngle(Vector3{X: x, Y: y, Z: z}, angle).RotateVec(vec)
}

// Angle returns the angle in radians between the calling Vector3 and the provided other Vector3 (ignoring the W component).
func (vec Vector3) Angle(other Vector3) float32 {
	// d := vec.Unit().Dot(other.Unit())
	d := vec.Dot(other)
	d /= vec.MagnitudeSquared() * other.MagnitudeSquared() // REVIEW
	d = math32.Clamp(d, -1, 1)                             // Acos returns NaN if value < -1 or > 1
	return math32.Acos(float32(d))
}

// AngleSigned returns the signed angle between the calling Vector and the other Vector, with planeNormal indicating the plane that both vectors share.
// In other words, if you want the signed angle between Vector3 A and Vector3 B, and the vectors differ only on two axes (say, X and Z),
// then the planeNormal Vector3 would be the cross product of these two Vectors (a Vector3 pointing "up", or Y+).
func (vec Vector3) AngleSigned(other, planeNormal Vector3) float32 {
	return math32.Atan2(other.Cross(vec).Dot(planeNormal.Unit()), vec.Dot(other))
}

// ClampAngle clamps the Vector3 such that it doesn't exceed the angle specified (in radians)
// between it and the baseline Vector.
func (vec Vector3) ClampAngle(baselineVec Vector3, maxAngle float32) Vector3 {

	mag := vec.Magnitude()

	angle := vec.Angle(baselineVec)

	if angle > maxAngle {
		vec = baselineVec.Slerp(vec, maxAngle/angle).Unit()
	}

	return vec.Scale(mag)

}

// Scale scales a Vector3 by the given scalar (ignoring the W component), returning a copy with the result.
func (vec Vector3) Scale(scalar float32) Vector3 {
	vec.X *= scalar
	vec.Y *= scalar
	vec.Z *= scalar
	return vec
}

// Divide divides a Vector3 by the given scalar (ignoring the W component), returning a copy with the result.
func (vec Vector3) Divide(scalar float32) Vector3 {
	vec.X /= scalar
	vec.Y /= scalar
	vec.Z /= scalar
	return vec
}

// Dot returns the dot product of a Vector3 and another Vector3 (ignoring the W component).
func (vec Vector3) Dot(other Vector3) float32 {
	return vec.X*other.X + vec.Y*other.Y + vec.Z*other.Z
}

// Round rounds the Vector's components off to the given increments, returning a new Vector.
// For example, Vector{0.1, 1.47, 3.33}.Round(0.25) will return Vector{0, 1.5, 3.25}.
func (vec Vector3) Round(roundToUnits float32) Vector3 {
	vec.X = math32.Round(vec.X/roundToUnits) * roundToUnits
	vec.Y = math32.Round(vec.Y/roundToUnits) * roundToUnits
	vec.Z = math32.Round(vec.Z/roundToUnits) * roundToUnits
	return vec
}

// Floor floors the Vector's components off, returning a new Vector.
// For example, Vector{0.1, 1.87, 3.33}.Floor() will return Vector{0, 1, 3}.
func (vec Vector3) Floor() Vector3 {
	vec.X = math32.Floor(vec.X)
	vec.Y = math32.Floor(vec.Y)
	vec.Z = math32.Floor(vec.Z)
	return vec
}

// Ceil ceils the Vector's components off, returning a new Vector.
// For example, Vector{0.1, 1.27, 3.33}.Ceil() will return Vector{1, 2, 4}.
func (vec Vector3) Ceil() Vector3 {
	vec.X = math32.Ceil(vec.X)
	vec.Y = math32.Ceil(vec.Y)
	vec.Z = math32.Ceil(vec.Z)
	return vec
}

// Lerp performs a linear interpolation between the starting Vector3 and the provided
// other Vector, to the given percentage (ranging from 0 to 1).
func (vec Vector3) Lerp(other Vector3, percentage float32) Vector3 {
	percentage = math32.Clamp(percentage, 0, 1)
	vec.X = vec.X + ((other.X - vec.X) * percentage)
	vec.Y = vec.Y + ((other.Y - vec.Y) * percentage)
	vec.Z = vec.Z + ((other.Z - vec.Z) * percentage)
	return vec
}

// Slerp performs a spherical linear interpolation between the starting Vector3 and the provided
// ending Vector, to the given percentage (ranging from 0 to 1).
// A slerp rotates a Vector3 around to a desired end Vector, rather than linearly interpolating to it.
// This normalizes the resulting Vector.
func (vec Vector3) Slerp(end Vector3, percentage float32) Vector3 {

	vec = vec.Unit()
	end = end.Unit()

	// Thank you StackOverflow, once again! : https://stackoverflow.com/questions/67919193/how-does-unity-implements-vector3-slerp-exactly
	percentage = math32.Clamp(percentage, 0, 1)

	dot := vec.Dot(end)

	dot = math32.Clamp(dot, -1, 1)

	theta := math32.Acos(dot) * percentage
	relative := end.Sub(vec.Scale(dot)).Unit()

	return (vec.Scale(math32.Cos(theta)).Add(relative.Scale(math32.Sin(theta)))).Unit()

}

// Clamp clamps the Vector3 to the maximum values provided.
func (vec Vector3) Clamp(x, y, z float32) Vector3 {
	vec.X = math32.Clamp(vec.X, -x, x)
	vec.Y = math32.Clamp(vec.Y, -y, y)
	vec.Z = math32.Clamp(vec.Z, -z, z)
	return vec
}

// ClampVec clamps the Vector3 to the maximum values in the Vector3 provided.
func (vec Vector3) ClampVec(extents Vector3) Vector3 {
	vec.X = math32.Clamp(vec.X, -extents.X, extents.X)
	vec.Y = math32.Clamp(vec.Y, -extents.Y, extents.Y)
	vec.Z = math32.Clamp(vec.Z, -extents.Z, extents.Z)
	return vec
}

// ModVector3 represents a reference to a Vector, made to facilitate easy method-chaining and modifications on that Vector3 (as you
// don't need to re-assign the results of a chain of operations to the original variable to "save" the results).
// Note that a ModVector3 is not meant to be used to chain methods on a vector to pass directly into a function; you can just
// use the normal vector functions for that purpose. ModVectors are pointers, which are allocated to the heap. This being the case,
// they should be slower relative to normal Vectors, so use them only in non-performance-critical parts of your application.
type ModVector3 struct {
	*Vector3
}

// NewModVector3 creates a new ModVector3 instance.
func NewModVector3(x, y, z float32) *ModVector3 {
	return &ModVector3{
		Vector3: &Vector3{X: x, Y: y, Z: z},
	}
}

// SetXYZ sets the values in the Vector to the x, y, and z values provided.
func (ip *ModVector3) SetXYZ(x, y, z float32) *ModVector3 {
	ip.Vector3.X = x
	ip.Vector3.Y = y
	ip.Vector3.Z = z
	return ip
}

// Set sets the values in the Vector3 to the values in the other Vector2.
func (vec *ModVector3) Set(other Vector3) {
	vec.Vector3.X = other.X
	vec.Vector3.Y = other.Y
	vec.Vector3.Z = other.Z
}

// Add adds the other Vector3 provided to the ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Add(other Vector3) *ModVector3 {
	ip.X += other.X
	ip.Y += other.Y
	ip.Z += other.Z
	return ip
}

// AddX adds the value provided to the ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) AddX(x float32) *ModVector3 {
	ip.X += x
	return ip
}

// AddY adds the value provided to the ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) AddY(x float32) *ModVector3 {
	ip.Y += x
	return ip
}

// AddZ adds the value provided to the ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) AddZ(x float32) *ModVector3 {
	ip.Z += x
	return ip
}

// Sub subtracts the other Vector3 from the calling ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Sub(other Vector3) *ModVector3 {
	ip.X -= other.X
	ip.Y -= other.Y
	ip.Z -= other.Z
	return ip
}

// SubX subs the value provided to the ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) SubX(x float32) *ModVector3 {
	ip.X -= x
	return ip
}

// SubY subs the value provided to the ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) SubY(x float32) *ModVector3 {
	ip.Y -= x
	return ip
}

// SubZ subs the value provided to the ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) SubZ(x float32) *ModVector3 {
	ip.Z -= x
	return ip
}

// MoveTowards moves a Vector3 towards another Vector3 given a specific magnitude. If the distance is less than that magnitude, it returns the target vector.
func (ip *ModVector3) MoveTowards(target Vector3, magnitude float32) *ModVector3 {
	modified := (ip.Vector3).MoveTowards(target, magnitude)
	ip.X = modified.X
	ip.Y = modified.Y
	ip.Z = modified.Z
	return ip
}

// SetZero sets the Vector3 to zero.
func (ip *ModVector3) SetZero() *ModVector3 {
	ip.X = 0
	ip.Y = 0
	ip.Z = 0
	return ip
}

// Scale scales the Vector3 by the scalar provided.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Scale(scalar float32) *ModVector3 {
	ip.X *= scalar
	ip.Y *= scalar
	ip.Z *= scalar
	return ip
}

// Divide divides a Vector3 by the given scalar (ignoring the W component).
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Divide(scalar float32) *ModVector3 {
	ip.X /= scalar
	ip.Y /= scalar
	ip.Z /= scalar
	return ip
}

// Expand expands the ModVector3 by the margin specified, in absolute units, if each component is over the minimum argument.
// To illustrate: Given a ModVector3 of {1, 0.1, -0.3}, ModVector.Expand(0.5, 0.2) would give you a ModVector3 of {1.5, 0.1, -0.8}.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Expand(margin, min float32) *ModVector3 {
	exp := ip.Vector3.Expand(margin, min)
	ip.X = exp.X
	ip.Y = exp.Y
	ip.Z = exp.Z
	return ip
}

// Mult performs Hadamard (component-wise) multiplication with the Vector3 on the other Vector3 provided.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Mult(other Vector3) *ModVector3 {
	ip.X *= other.X
	ip.Y *= other.Y
	ip.Z *= other.Z
	return ip
}

// Unit normalizes the ModVector3 (sets it to be of unit length).
// It does not alter the W component of the Vector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Unit() *ModVector3 {
	l := ip.Magnitude()
	if l < 1e-8 || l == 1 {
		// If it's 0, then don't modify the vector
		return ip
	}
	ip.X, ip.Y, ip.Z = ip.X/l, ip.Y/l, ip.Z/l
	return ip
}

// Swizzle swizzles the ModVector3 using the string provided.
// The string can be of length 3 ("xyz") or 4 ("xyzw").
// The string should be composed of the axes of a vector, i.e. 'x', 'y', 'z', or 'w'.
// Example: `vec := Vector{1, 2, 3}.Swizzle("zxy") // Returns a Vector3 of {3, 1, 2}.`
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Swizzle(swizzleString string) *ModVector3 {
	vec := ip.Vector3.Swizzle(swizzleString)
	ip.X = vec.X
	ip.Y = vec.Y
	ip.Z = vec.Z
	return ip
}

// Cross performs a cross-product multiplication on the ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Cross(other Vector3) *ModVector3 {
	cross := ip.Vector3.Cross(other)
	ip.X = cross.X
	ip.Y = cross.Y
	ip.Z = cross.Z
	return ip
}

// RotateVec rotates the calling ModVector3 by the axis Vector3 and angle provided (in radians).
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) RotateVec(axis Vector3, angle float32) *ModVector3 {
	rot := ip.Vector3.RotateVec(axis, angle)
	ip.X = rot.X
	ip.Y = rot.Y
	ip.Z = rot.Z
	return ip
}

// RotateVec rotates the calling ModVector3 by an axis Vector3 composed of the x, y, and z components provided,
// by the angle provided (in radians).
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Rotate(x, y, z, angle float32) *ModVector3 {
	rot := ip.Vector3.Rotate(x, y, z, angle)
	ip.X = rot.X
	ip.Y = rot.Y
	ip.Z = rot.Z
	return ip
}

// Invert inverts all components of the calling ModVector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Invert() *ModVector3 {
	ip.X = -ip.X
	ip.Y = -ip.Y
	ip.Z = -ip.Z
	return ip
}

// Round rounds off the ModVector's components to the given space in world units.
// For example, Vector{0.1, 1.27, 3.33}.Modify().Round(0.25) will return Vector{0, 1.25, 3.25}.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Round(roundToUnits float32) *ModVector3 {
	snapped := ip.Vector3.Round(roundToUnits)
	ip.X = snapped.X
	ip.Y = snapped.Y
	ip.Z = snapped.Z
	return ip
}

// Floor floors the ModVector's components off, returning a new Vector.
// For example, Vector{0.1, 1.27, 3.33}.Modify().Floor() will return Vector{0, 1, 3}.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Floor() *ModVector3 {
	snapped := ip.Vector3.Floor()
	ip.X = snapped.X
	ip.Y = snapped.Y
	ip.Z = snapped.Z
	return ip
}

// Ceil ceils the ModVector's components off, returning a new Vector.
// For example, Vector{0.1, 1.27, 3.33}.Modify().Ceil() will return Vector{1, 2, 4}.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Ceil() *ModVector3 {
	snapped := ip.Vector3.Ceil()
	ip.X = snapped.X
	ip.Y = snapped.Y
	ip.Z = snapped.Z
	return ip
}

// ClampMagnitude clamps the overall magnitude of the Vector3 to the maximum magnitude specified.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) ClampMagnitude(maxMag float32) *ModVector3 {
	clamped := ip.Vector3.ClampMagnitude(maxMag)
	ip.X = clamped.X
	ip.Y = clamped.Y
	ip.Z = clamped.Z
	return ip
}

// SubMagnitude subtacts the given magnitude from the Vector's. If the vector's magnitude is less than the given magnitude to subtract,
// a zero-length Vector3 will be returned.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) SubMagnitude(mag float32) *ModVector3 {
	if ip.Magnitude() > mag {
		ip.Sub(ip.ToVector().Unit().Scale(mag))
	} else {
		ip.X = 0
		ip.Y = 0
		ip.Z = 0
	}
	return ip

}

// Lerp performs a linear interpolation between the starting ModVector3 and the provided
// other Vector, to the given percentage.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Lerp(other Vector3, percentage float32) *ModVector3 {
	lerped := ip.Vector3.Lerp(other, percentage)
	ip.X = lerped.X
	ip.Y = lerped.Y
	ip.Z = lerped.Z
	return ip
}

// Slerp performs a spherical linear interpolation between the starting ModVector3 and the provided
// other Vector, to the given percentage.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Slerp(other Vector3, percentage float32) *ModVector3 {
	slerped := ip.Vector3.Slerp(other, percentage)
	ip.X = slerped.X
	ip.Y = slerped.Y
	ip.Z = slerped.Z
	return ip
}

// Clamp clamps the Vector3 to the maximum values provided.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Clamp(x, y, z float32) *ModVector3 {
	clamped := (ip.Vector3).Clamp(x, y, z)
	ip.X = clamped.X
	ip.Y = clamped.Y
	ip.Z = clamped.Z
	return ip
}

// ClampVec clamps the Vector3 to the maximum values in the Vector3 provided.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) ClampVec(extents Vector3) *ModVector3 {
	clamped := ip.Vector3.ClampVec(extents)
	ip.X = clamped.X
	ip.Y = clamped.Y
	ip.Z = clamped.Z
	return ip
}

// ClampAngle clamps the Vector3 such that it doesn't exceed the angle specified (in radians).
// This function returns a normalized (unit) ModVector3 for method chaining.
func (ip *ModVector3) ClampAngle(baselineVec Vector3, maxAngle float32) *ModVector3 {
	clamped := ip.Vector3.ClampAngle(baselineVec, maxAngle)
	ip.X = clamped.X
	ip.Y = clamped.Y
	ip.Z = clamped.Z
	return ip
}

// String converts the ModVector3 to a string. Because it's a ModVector, it's represented with a *.
func (ip *ModVector3) String() string {
	return fmt.Sprintf("*{%.2f, %.2f, %.2f}", ip.X, ip.Y, ip.Z)
}

// Clone returns a ModVector3 of a clone of its backing Vector.
// This function returns the calling ModVector3 for method chaining.
func (ip *ModVector3) Clone() *ModVector3 {
	v := *ip
	return &v
}

// ToVector3 returns a copy of the Vector3 that the ModVector3 is modifying.
func (ip *ModVector3) ToVector() Vector3 {
	return *ip.Vector3
}

// //////

// Vector2 represents a 2D Vector2, which can be used for usual 2D applications (position, direction, velocity, etc).
// Any Vector2 functions that modify the calling Vector2 return copies of the modified Vector2, meaning you can do method-chaining easily.
type Vector2 struct {
	X float32 // The X (1st) component of the Vector
	Y float32 // The Y (2nd) component of the Vector
}

// NewVector2 creates a new Vector with the specified x, y, and z components. The W component is generally ignored for most purposes.
func NewVector2(x, y float32) Vector2 {
	return Vector2{X: x, Y: y}
}

// String returns a string representation of the Vector, excluding its W component (which is primarily used for internal purposes).
func (vec Vector2) String() string {
	return fmt.Sprintf("{%.2f, %.2f}", vec.X, vec.Y)
}

// Add returns a copy of the calling vector, added together with the other Vector provided (ignoring the W component).
func (vec Vector2) Add(other Vector2) Vector2 {
	vec.X += other.X
	vec.Y += other.Y
	return vec
}

// AddX returns a copy of the calling vector with an added value to the X axis.
func (vec Vector2) AddX(x float32) Vector2 {
	vec.X += x
	return vec
}

// AddY returns a copy of the calling vector with an added value to the Y axis.
func (vec Vector2) AddY(y float32) Vector2 {
	vec.Y += y
	return vec
}

// Sub returns a copy of the calling Vector, with the other Vector subtracted from it (ignoring the W component).
func (vec Vector2) Sub(other Vector2) Vector2 {
	vec.X -= other.X
	vec.Y -= other.Y
	return vec
}

// SubX returns a copy of the calling vector with an added value to the X axis.
func (vec Vector2) SubX(x float32) Vector2 {
	vec.X -= x
	return vec
}

// SubY returns a copy of the calling vector with an added value to the Y axis.
func (vec Vector2) SubY(y float32) Vector2 {
	vec.Y -= y
	return vec
}

// Expand expands the Vector by the margin specified, in absolute units, if each component is over the minimum argument.
// To illustrate: Given a Vector of {1, 0.1, -0.3}, Vector.Expand(0.5, 0.2) would give you a Vector of {1.5, 0.1, -0.8}.
// This function returns a copy of the Vector with the result.
func (vec Vector2) Expand(margin, min float32) Vector2 {
	if vec.X > min || vec.X < -min {
		vec.X += math32.Copysign(margin, vec.X)
	}
	if vec.Y > min || vec.Y < -min {
		vec.Y += math32.Copysign(margin, vec.Y)
	}
	return vec
}

// Invert returns a copy of the Vector with all components inverted.
func (vec Vector2) Invert() Vector2 {
	vec.X = -vec.X
	vec.Y = -vec.Y
	return vec
}

// Magnitude returns the length of the Vector.
func (vec Vector2) Magnitude() float32 {
	return math32.Sqrt(vec.X*vec.X + vec.Y*vec.Y)
}

// MagnitudeSquared returns the squared length of the Vector; this is faster than Length() as it avoids using math.math32.Sqrt().
func (vec Vector2) MagnitudeSquared() float32 {
	return vec.X*vec.X + vec.Y*vec.Y
}

// ClampMagnitude clamps the overall magnitude of the Vector to the maximum magnitude specified, returning a copy with the result.
func (vec Vector2) ClampMagnitude(maxMag float32) Vector2 {
	if vec.Magnitude() > maxMag {
		vec = vec.Unit().Scale(maxMag)
	}
	return vec
}

// AddMagnitude adds magnitude to the Vector in the direction it's already pointing.
func (vec Vector2) AddMagnitude(mag float32) Vector2 {
	return vec.Add(vec.Unit().Scale(mag))
}

// SubMagnitude subtracts the given magnitude from the Vector's existing magnitude.
// If the vector's magnitude is less than the given magnitude to subtract, a zero-length Vector will be returned.
func (vec Vector2) SubMagnitude(mag float32) Vector2 {
	if vec.Magnitude() > mag {
		return vec.Sub(vec.Unit().Scale(mag))
	}
	return Vector2{0, 0}

}

// Distance returns the distance from the calling Vector to the other Vector provided.
func (vec Vector2) Distance(other Vector2) float32 {
	vec.X -= other.X
	vec.Y -= other.Y
	return math32.Sqrt(vec.X*vec.X + vec.Y*vec.Y)
}

// Distance returns the squared distance from the calling Vector to the other Vector provided. This is faster than Distance(), as it avoids using math.math32.Sqrt().
func (vec Vector2) DistanceSquared(other Vector2) float32 {
	vec.X -= other.X
	vec.Y -= other.Y
	return vec.X*vec.X + vec.Y*vec.Y
}

// Mult performs Hadamard (component-wise) multiplication on the calling Vector with the other Vector provided, returning a copy with the result (and ignoring the Vector's W component).
func (vec Vector2) Mult(other Vector2) Vector2 {
	vec.X *= other.X
	vec.Y *= other.Y
	return vec
}

// Unit returns a copy of the Vector, normalized (set to be of unit length).
// It does not alter the W component of the Vector.
func (vec Vector2) Unit() Vector2 {
	l := vec.Magnitude()
	if l < 1e-8 || l == 1 {
		// If it's 0, then don't modify the vector
		return vec
	}
	vec.X, vec.Y = vec.X/l, vec.Y/l
	return vec
}

// SetX sets the X component in the vector to the value provided.
func (vec Vector2) SetX(x float32) Vector2 {
	vec.X = x
	return vec
}

// SetY sets the Y component in the vector to the value provided.
func (vec Vector2) SetY(y float32) Vector2 {
	vec.Y = y
	return vec
}

// SetXY sets the values in the Vector to the x, y, and z values provided.
func (vec Vector2) SetXY(x, y float32) Vector2 {
	vec.X = x
	vec.Y = y
	return vec
}

// Set sets the values in the Vector3 to the values in the other Vector2.
func (vec *Vector2) Set(other Vector2) {
	vec.X = other.X
	vec.Y = other.Y
}

// Reflect reflects the vector against the given surface normal.
func (vec Vector2) Reflect(normal Vector2) Vector2 {
	n := normal.Unit()
	return vec.Sub(n.Scale(2 * n.Dot(vec)))
}

// Perp returns the right-handed perpendicular of the vector (i.e. the vector rotated 90 degrees to the right, clockwise).
func (vec Vector2) Perp() Vector2 {
	return Vector2{-vec.Y, vec.X}
}

// Floats returns a [2]float32 array consisting of the Vector's contents.
func (vec Vector2) Floats() [2]float32 {
	return [2]float32{vec.X, vec.Y}
}

// Equals returns true if the two Vectors are close enough in all values (excluding W).
func (vec Vector2) Equals(other Vector2) bool {

	eps := float32(1e-4)

	if math32.Abs(float32(vec.X-other.X)) > eps || math32.Abs(float32(vec.Y-other.Y)) > eps {
		return false
	}

	return true

}

// IsZero returns true if the values in the Vector are extremely close to 0 (excluding W).
func (vec Vector2) IsZero() bool {

	eps := float32(1e-4)

	if math32.Abs(float32(vec.X)) > eps || math32.Abs(float32(vec.Y)) > eps {
		return false
	}

	// if !onlyXYZ && math.math32.Abs(vec.W-other.W) > eps {
	// 	return false
	// }

	return true

}

// Rotate returns a copy of the Vector, rotated around an axis Vector with the x, y, and z components provided, by the angle
// provided (in radians), counter-clockwise.
// The function is most efficient if passed an orthogonal, normalized axis (i.e. the X, Y, or Z constants).
// Note that this function ignores the W component of both Vectors.
func (vec Vector2) Rotate(angle float32) Vector2 {
	x := vec.X
	y := vec.Y
	vec.X = x*math32.Cos(angle) - y*math32.Sin(angle)
	vec.Y = x*math32.Sin(angle) + y*math32.Cos(angle)
	return vec
}

// Angle returns the signed angle in radians between the calling Vector and the provided other Vector (ignoring the W component).
func (vec Vector2) Angle(other Vector2) float32 {
	angle := math32.Atan2(other.Y, other.X) - math32.Atan2(vec.Y, vec.X)
	if angle > math.Pi {
		angle -= 2 * math.Pi
	} else if angle <= -math.Pi {
		angle += 2 * math.Pi
	}
	return angle
}

var worldRight2D = Vector2{1, 0}

// AngleRotation returns the angle in radians between this Vector and world right (1, 0).
func (vec Vector2) AngleRotation() float32 {
	return vec.Angle(worldRight2D)
}

// Scale scales a Vector by the given scalar (ignoring the W component), returning a copy with the result.
func (vec Vector2) Scale(scalar float32) Vector2 {
	vec.X *= scalar
	vec.Y *= scalar
	return vec
}

// Divide divides a Vector by the given scalar (ignoring the W component), returning a copy with the result.
func (vec Vector2) Divide(scalar float32) Vector2 {
	vec.X /= scalar
	vec.Y /= scalar
	return vec
}

// Dot returns the dot product of a Vector and another Vector (ignoring the W component).
func (vec Vector2) Dot(other Vector2) float32 {
	return vec.X*other.X + vec.Y*other.Y
}

// Round rounds off the Vector's components to the given space in world unit increments, returning a clone
// (e.g. Vector{0.1, 1.27, 3.33}.Snap(0.25) will return Vector{0, 1.25, 3.25}).
func (vec Vector2) Round(snapToUnits float32) Vector2 {
	vec.X = math32.Round(vec.X/snapToUnits) * snapToUnits
	vec.Y = math32.Round(vec.Y/snapToUnits) * snapToUnits
	return vec
}

// ClampAngle clamps the Vector such that it doesn't exceed the angle specified (in radians).
// This function returns a normalized (unit) Vector.
func (vec Vector2) ClampAngle(baselineVec Vector2, maxAngle float32) Vector2 {

	mag := vec.Magnitude()

	angle := vec.Angle(baselineVec)

	if angle > maxAngle {
		vec = baselineVec.Slerp(vec, maxAngle/angle).Unit()
	}

	return vec.Scale(mag)

}

// Lerp performs a linear interpolation between the starting Vector and the provided
// other Vector, to the given percentage (ranging from 0 to 1).
func (vec Vector2) Lerp(other Vector2, percentage float32) Vector2 {
	percentage = math32.Clamp(percentage, 0, 1)
	vec.X = vec.X + ((other.X - vec.X) * percentage)
	vec.Y = vec.Y + ((other.Y - vec.Y) * percentage)
	return vec
}

// Slerp performs a spherical linear interpolation between the starting Vector and the provided
// ending Vector, to the given percentage (ranging from 0 to 1).
// This should be done with directions, usually, rather than positions.
// This being the case, this normalizes both Vectors.
func (vec Vector2) Slerp(targetDirection Vector2, percentage float32) Vector2 {

	vec = vec.Unit()
	targetDirection = targetDirection.Unit()

	// Thank you StackOverflow, once again! : https://stackoverflow.com/questions/67919193/how-does-unity-implements-vector3-slerp-exactly
	percentage = math32.Clamp(percentage, 0, 1)

	dot := vec.Dot(targetDirection)

	dot = math32.Clamp(dot, -1, 1)

	theta := math32.Acos(dot) * percentage
	relative := targetDirection.Sub(vec.Scale(dot)).Unit()

	return (vec.Scale(math32.Cos(theta)).Add(relative.Scale(math32.Sin(theta)))).Unit()

}

type Vector4 struct {
	X float32 // The X (1st) component of the Vector
	Y float32 // The Y (2nd) component of the Vector
	Z float32 // The Z (3rd) component of the Vector
	W float32 // The w (4th) component of the Vector; not used for most Vector3 functions
}

// Unit returns a copy of the Vector, normalized (set to be of unit length).
// It does not alter the W component of the Vector.
func (vec Vector4) Unit() Vector4 {
	l := vec.Magnitude()
	if l < 1e-8 || l == 1 {
		// If it's 0, then don't modify the vector
		return vec
	}
	vec.X, vec.Y, vec.Z = vec.X/l, vec.Y/l, vec.Z/l
	return vec
}

// Magnitude returns the length of the Vector3 (ignoring the Vector's W component).
func (vec Vector4) Magnitude() float32 {
	return math32.Sqrt(vec.X*vec.X + vec.Y*vec.Y + vec.Z*vec.Z)
}

// Invert returns a copy of the Vector3 with all components inverted (ignoring the Vector's W component).
func (vec Vector4) Invert() Vector4 {
	vec.X = -vec.X
	vec.Y = -vec.Y
	vec.Z = -vec.Z
	return vec
}

func (vec Vector3) To4D() Vector4 {
	return Vector4{vec.X, vec.Y, vec.Z, 0}
}

func (vec Vector4) To3D() Vector3 {
	return Vector3{vec.X, vec.Y, vec.Z}
}

// Round rounds the Vector's components off to the given increments, returning a new Vector.
// For example, Vector{0.1, 1.47, 3.33}.Round(0.25) will return Vector{0, 1.5, 3.25}.
func (vec Vector4) Round(roundToUnits float32) Vector4 {
	vec.X = math32.Round(vec.X/roundToUnits) * roundToUnits
	vec.Y = math32.Round(vec.Y/roundToUnits) * roundToUnits
	vec.Z = math32.Round(vec.Z/roundToUnits) * roundToUnits
	return vec
}
