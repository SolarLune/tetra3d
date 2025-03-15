// math32 is a stand-in for the built-in math package, but the functions take at least float32s (if not all comparable numbers) instead of float64s.
// This is to make it easier to work with Tetra3D, since internal datatypes for e.g. vectors, quaternions, and matrices moved from float64 to float32 instead.
package math32

import (
	"math"

	_ "embed"
)

const MaxFloat32 = float32(math.MaxFloat32)

// ToRadians is a helper function to easily convert degrees to radians (which is what the rotation-oriented functions in Tetra3D use).
func ToRadians(degrees float32) float32 {
	return math.Pi * degrees / 180
}

// ToDegrees is a helper function to easily convert radians to degrees for human readability.
func ToDegrees(radians float32) float32 {
	return radians / math.Pi * 180
}

// Min returns the minimum value out of two provided values.
func Min[number float32 | float64 | int | int32 | int64](x, y number) number {
	return number(math.Min(float64(x), float64(y)))
}

// Max returns the maximum value out of two provided values.
func Max[number float32 | float64 | int | int32 | int64](x, y number) number {
	return number(math.Max(float64(x), float64(y)))
}

// Clamp clamps a value to the minimum and maximum values provided.
func Clamp[number float32 | float64 | int | int32 | int64](value, min, max number) number {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

// MoveTowardsZero subtracts a value from the given float and gives that result, unless it is less than
// that value, in which case 0 is returned.
func MoveTowardsZero(f, delta float32) float32 {
	if f > delta {
		return f - delta
	} else if f < -delta {
		return f + delta
	}
	return 0
}

// Sign returns the sign of the value given. If it's greater than 0, it returns 1. If less than 0, it returns -1. Otherwise, it returns 0.
func Sign(f float32) float32 {
	if f > 0 {
		return 1
	} else if f < 0 {
		return -1
	}
	return 0
}

// IsNaN returns if the provided float32 is a NaN.
func IsNaN(x float32) bool {
	return math.IsNaN(float64(x))
}

// IsInf returns if the provided float32 (x) is Inf in the direction of the sign provided.
func IsInf(x float32, sign int) bool {
	return math.IsInf(float64(x), sign)
}

// Pow returns x**y, the base-x exponential of y.
//
// Special cases are (in order):
//
// Pow(x, ±0) = 1 for any x
// Pow(1, y) = 1 for any y
// Pow(x, 1) = x for any x
// Pow(NaN, y) = NaN
// Pow(x, NaN) = NaN
// Pow(±0, y) = ±Inf for y an odd integer < 0
// Pow(±0, -Inf) = +Inf
// Pow(±0, +Inf) = +0
// Pow(±0, y) = +Inf for finite y < 0 and not an odd integer
// Pow(±0, y) = ±0 for y an odd integer > 0
// Pow(±0, y) = +0 for finite y > 0 and not an odd integer
// Pow(-1, ±Inf) = 1
// Pow(x, +Inf) = +Inf for |x| > 1
// Pow(x, -Inf) = +0 for |x| > 1
// Pow(x, +Inf) = +0 for |x| < 1
// Pow(x, -Inf) = +Inf for |x| < 1
// Pow(+Inf, y) = +Inf for y > 0
// Pow(+Inf, y) = +0 for y < 0
// Pow(-Inf, y) = Pow(-0, -y)
// Pow(x, y) = NaN for finite x < 0 and finite non-integer y
func Pow(x, y float32) float32 {
	return float32(math.Pow(float64(x), float64(y)))
}

// Round returns the nearest integer, rounding half away from zero.
//
// Special cases are:
//
// Round(±0) = ±0
// Round(±Inf) = ±Inf
// Round(NaN) = NaN
func Round(x float32) float32 {
	return float32(math.Round(float64(x)))
}

// Sqrt returns the square root of x.
//
// Special cases are:
//
//	Sqrt(+Inf) = +Inf
//	Sqrt(±0) = ±0
//	Sqrt(x < 0) = NaN
//	Sqrt(NaN) = NaN
func Sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// Sin returns the sine of the radian argument x.
//
// Special cases are:
//
//	Sin(±0) = ±0
//	Sin(±Inf) = NaN
//	Sin(NaN) = NaN
func Sin(x float32) float32 {
	return float32(math.Sin(float64(x)))
}

// Cos returns the cosine of the radian argument x.
//
// Special cases are:
//
//	Cos(±Inf) = NaN
//	Cos(NaN) = NaN
func Cos(x float32) float32 {
	return float32(math.Cos(float64(x)))
}

// Acos returns the arccosine, in radians, of x.
//
// Special case is:
//
//	Acos(x) = NaN if x < -1 or x > 1
func Acos(x float32) float32 {
	return float32(math.Acos(float64(x)))
}

// Acosh returns the inverse hyperbolic cosine of x.
//
// Special cases are:
//
//	Acosh(+Inf) = +Inf
//	Acosh(x) = NaN if x < 1
//	Acosh(NaN) = NaN
func Acosh(x float32) float32 {
	return float32(math.Acosh(float64(x)))
}

// Atan returns the arctangent, in radians, of x.
//
// Special cases are:
//
//	Atan(±0) = ±0
//	Atan(±Inf) = ±Pi/2
func Atan(x float32) float32 {
	return float32(math.Atan(float64(x)))
}

// Atan2 returns the arc tangent of y/x, using
// the signs of the two to determine the quadrant
// of the return value.
//
// Special cases are (in order):
//
//	Atan2(y, NaN) = NaN
//	Atan2(NaN, x) = NaN
//	Atan2(+0, x>=0) = +0
//	Atan2(-0, x>=0) = -0
//	Atan2(+0, x<=-0) = +Pi
//	Atan2(-0, x<=-0) = -Pi
//	Atan2(y>0, 0) = +Pi/2
//	Atan2(y<0, 0) = -Pi/2
//	Atan2(+Inf, +Inf) = +Pi/4
//	Atan2(-Inf, +Inf) = -Pi/4
//	Atan2(+Inf, -Inf) = 3Pi/4
//	Atan2(-Inf, -Inf) = -3Pi/4
//	Atan2(y, +Inf) = 0
//	Atan2(y>0, -Inf) = +Pi
//	Atan2(y<0, -Inf) = -Pi
//	Atan2(+Inf, x) = +Pi/2
//	Atan2(-Inf, x) = -Pi/2
func Atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

// Atanh returns the inverse hyperbolic tangent of x.
//
// Special cases are:
//
//	Atanh(1) = +Inf
//	Atanh(±0) = ±0
//	Atanh(-1) = -Inf
//	Atanh(x) = NaN if x < -1 or x > 1
//	Atanh(NaN) = NaN
func Atanh(x float32) float32 {
	return float32(math.Atanh(float64(x)))
}

// Abs returns the absolute value of x.
//
// Special cases are:
//
//	Abs(±Inf) = +Inf
//	Abs(NaN) = NaN
func Abs(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

// Ceil returns the least integer value greater than or equal to x.
//
// Special cases are:
//
//	Ceil(±0) = ±0
//	Ceil(±Inf) = ±Inf
//	Ceil(NaN) = NaN
func Ceil(x float32) float32 {
	return float32(math.Ceil(float64(x)))
}

// Floor returns the greatest integer value less than or equal to x.
//
// Special cases are:
//
//	Floor(±0) = ±0
//	Floor(±Inf) = ±Inf
//	Floor(NaN) = NaN
func Floor(x float32) float32 {
	return float32(math.Floor(float64(x)))
}

// Copysign returns a value with the magnitude of f
// and the sign of sign.
func Copysign(f, sign float32) float32 {
	return float32(math.Copysign(float64(f), float64(sign)))
}

// Signbit reports whether x is negative or negative zero.
func Signbit(x float32) bool {
	return math.Signbit(float64(x))
}

// Asin returns the arcsine, in radians, of x.
//
// Special cases are:
//
//	Asin(±0) = ±0
//	Asin(x) = NaN if x < -1 or x > 1
func Asin(x float32) float32 {
	return float32(math.Asin(float64(x)))
}

// Asinh returns the inverse hyperbolic sine of x.
//
// Special cases are:
//
//	Asinh(±0) = ±0
//	Asinh(±Inf) = ±Inf
//	Asinh(NaN) = NaN
func Asinh(x float32) float32 {
	return float32(math.Asinh(float64(x)))
}

// Cosh returns the hyperbolic cosine of x.
//
// Special cases are:
//
//	Cosh(±0) = 1
//	Cosh(±Inf) = +Inf
//	Cosh(NaN) = NaN
func Cosh(x float32) float32 {
	return float32(math.Cosh(float64(x)))
}

// Exp returns e**x, the base-e exponential of x.
//
// Special cases are:
//
//	Exp(+Inf) = +Inf
//	Exp(NaN) = NaN
//
// Very large values overflow to 0 or +Inf.
// Very small values underflow to 1.
func Exp(x float32) float32 {
	return float32(math.Exp(float64(x)))
}

// Exp2 returns 2**x, the base-2 exponential of x.
//
// Special cases are the same as [Exp].
func Exp2(x float32) float32 {
	return float32(math.Exp2(float64(x)))
}

// Expm1 returns e**x - 1, the base-e exponential of x minus 1.
// It is more accurate than [Exp](x) - 1 when x is near zero.
//
// Special cases are:
//
//	Expm1(+Inf) = +Inf
//	Expm1(-Inf) = -1
//	Expm1(NaN) = NaN
//
// Very large values overflow to -1 or +Inf.
func Expm1(x float32) float32 {
	return float32(math.Expm1(float64(x)))
}

// FMA returns x * y + z, computed with only one rounding.
// (That is, FMA returns the fused multiply-add of x, y, and z.)
func FMA(x, y, z float32) float32 {
	return float32(math.FMA(float64(x), float64(y), float64(z)))
}

// Log returns the natural logarithm of x.
//
// Special cases are:
//
//	Log(+Inf) = +Inf
//	Log(0) = -Inf
//	Log(x < 0) = NaN
//	Log(NaN) = NaN
func Log(x float32) float32 {
	return float32(math.Log(float64(x)))
}

// Log10 returns the decimal logarithm of x.
// The special cases are the same as for [Log].
func Log10(x float32) float32 {
	return float32(math.Log10(float64(x)))
}

// Log1p returns the natural logarithm of 1 plus its argument x.
// It is more accurate than [Log](1 + x) when x is near zero.
//
// Special cases are:
//
//	Log1p(+Inf) = +Inf
//	Log1p(±0) = ±0
//	Log1p(-1) = -Inf
//	Log1p(x < -1) = NaN
//	Log1p(NaN) = NaN
func Log1p(x float32) float32 {
	return float32(math.Log1p(float64(x)))
}

// Log2 returns the binary logarithm of x.
// The special cases are the same as for [Log].
func Log2(x float32) float32 {
	return float32(math.Log2(float64(x)))
}

// Logb returns the binary exponent of x.
//
// Special cases are:
//
//	Logb(±Inf) = +Inf
//	Logb(0) = -Inf
//	Logb(NaN) = NaN
func Logb(x float32) float32 {
	return float32(math.Logb(float64(x)))
}

// Mod returns the floating-point remainder of x/y.
// The magnitude of the result is less than y and its
// sign agrees with that of x.
//
// Special cases are:
//
//	Mod(±Inf, y) = NaN
//	Mod(NaN, y) = NaN
//	Mod(x, 0) = NaN
//	Mod(x, ±Inf) = x
//	Mod(x, NaN) = NaN
func Mod(x, y float32) float32 {
	return float32(math.Mod(float64(x), float64(y)))
}

// Modf returns integer and fractional floating-point numbers
// that sum to f. Both values have the same sign as f.
//
// Special cases are:
//
//	Modf(±Inf) = ±Inf, NaN
//	Modf(NaN) = NaN, NaN
func Modf(f float32) (float32, float32) {
	integer, frac := math.Modf(float64(f))
	return float32(integer), float32(frac)
}

// Pow10 returns 10**n, the base-10 exponential of n.
//
// Special cases are:
//
//	Pow10(n) =    0 for n < -323
//	Pow10(n) = +Inf for n > 308
func Pow10(n int) float32 {
	return float32(math.Pow10(n))
}

// Remainder returns the IEEE 754 floating-point remainder of x/y.
//
// Special cases are:
//
//	Remainder(±Inf, y) = NaN
//	Remainder(NaN, y) = NaN
//	Remainder(x, 0) = NaN
//	Remainder(x, ±Inf) = x
//	Remainder(x, NaN) = NaN
func Remainder(x, y float32) float32 {
	return float32(math.Remainder(float64(x), float64(y)))
}

// RoundToEven returns the nearest integer, rounding ties to even.
//
// Special cases are:
//
//	RoundToEven(±0) = ±0
//	RoundToEven(±Inf) = ±Inf
//	RoundToEven(NaN) = NaN
func RoundToEven(x float32) float32 {
	return float32(math.RoundToEven(float64(x)))
}

// Sincos returns Sin(x), Cos(x).
//
// Special cases are:
//
//	Sincos(±0) = ±0, 1
//	Sincos(±Inf) = NaN, NaN
//	Sincos(NaN) = NaN, NaN
func Sincos(x float32) (float32, float32) {
	sin, cos := math.Sincos(float64(x))
	return float32(sin), float32(cos)
}

// Sinh returns the hyperbolic sine of x.
//
// Special cases are:
//
//	Sinh(±0) = ±0
//	Sinh(±Inf) = ±Inf
//	Sinh(NaN) = NaN
func Sinh(x float32) float32 {
	return float32(math.Sinh(float64(x)))
}

// Tan returns the tangent of the radian argument x.
//
// Special cases are:
//
//	Tan(±0) = ±0
//	Tan(±Inf) = NaN
//	Tan(NaN) = NaN
func Tan(x float32) float32 {
	return float32(math.Tan(float64(x)))
}

// Tanh returns the hyperbolic tangent of x.
//
// Special cases are:
//
//	Tanh(±0) = ±0
//	Tanh(±Inf) = ±1
//	Tanh(NaN) = NaN
func Tanh(x float32) float32 {
	return float32(math.Tanh(float64(x)))
}

// Trunc returns the integer value of x.
//
// Special cases are:
//
//	Trunc(±0) = ±0
//	Trunc(±Inf) = ±Inf
//	Trunc(NaN) = NaN
func Trunc(x float32) float32 {
	return float32(math.Trunc(float64(x)))
}

const Pi = math.Pi
