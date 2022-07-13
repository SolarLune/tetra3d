package tetra3d

import (
	"image/color"
	"math"
)

// Color represents a color, containing R, G, B, and A components, each expected to range from 0 to 1.
type Color struct {
	R, G, B, A float32
}

// NewColor returns a new Color, with the provided R, G, B, and A components expected to range from 0 to 1.
func NewColor(r, g, b, a float32) *Color {
	return &Color{r, g, b, a}
}

// Clone returns a clone of the Color instance.
func (color *Color) Clone() *Color {
	return NewColor(color.R, color.G, color.B, color.A)
}

// Set sets the RGBA components of the Color to the r, g, b, and a arguments provided. The components are expected to range from 0 to 1.
func (color *Color) Set(r, g, b, a float32) {
	color.R = r
	color.G = g
	color.B = b
	color.A = a
}

// AddRGBA adds the provided R, G, B, and A values to the color as provided. The components are expected to range from 0 to 1.
func (color *Color) AddRGBA(r, g, b, a float32) {
	color.R += r
	color.G += g
	color.B += b
	color.A += a
}

// Add adds the provided Color to the existing Color.
func (color *Color) Add(other *Color) {
	color.AddRGBA(other.ToFloat32s())
}

// MultiplyRGBA multiplies the color's RGBA channels by the provided R, G, B, and A scalar values.
func (color *Color) MultiplyRGBA(scalarR, scalarG, scalarB, scalarA float32) {
	color.R *= scalarR
	color.G *= scalarG
	color.B *= scalarB
	color.A *= scalarA
}

// Multiply multiplies the existing Color by the provided Color.
func (color *Color) Multiply(other *Color) {
	color.MultiplyRGBA(other.ToFloat32s())
}

// ToFloat32s returns the Color as four float32 in the order R, G, B, and A.
func (color *Color) ToFloat32s() (float32, float32, float32, float32) {
	return color.R, color.G, color.B, color.A
}

// ToFloat64s returns four float64 values for each channel in the Color in the order R, G, B, and A.
func (color *Color) ToFloat64s() (float64, float64, float64, float64) {
	return float64(color.R), float64(color.G), float64(color.B), float64(color.A)
}

// ToRGBA64 converts a color to a color.RGBA64 instance.
func (c *Color) ToRGBA64() color.RGBA64 {
	return color.RGBA64{
		c.capRGBA64(c.R),
		c.capRGBA64(c.G),
		c.capRGBA64(c.B),
		c.capRGBA64(c.A),
	}
}

func (color *Color) capRGBA64(value float32) uint16 {
	if value > 1 {
		value = 1
	} else if value < 0 {
		value = 0
	}
	return uint16(value * math.MaxUint16)
}

// ConvertTosRGB() converts the color's R, G, and B components to the sRGB color space. This is used to convert
// colors from their values in GLTF to how they should appear on the screen. See: https://en.wikipedia.org/wiki/SRGB
func (color *Color) ConvertTosRGB() {

	if color.R <= 0.0031308 {
		color.R *= 12.92
	} else {
		color.R = float32(1.055*math.Pow(float64(color.R), 1/2.4) - 0.055)
	}

	if color.G <= 0.0031308 {
		color.G *= 12.92
	} else {
		color.G = float32(1.055*math.Pow(float64(color.G), 1/2.4) - 0.055)
	}

	if color.B <= 0.0031308 {
		color.B *= 12.92
	} else {
		color.B = float32(1.055*math.Pow(float64(color.B), 1/2.4) - 0.055)
	}

}

// NewColorFromHSV returns a new color, using hue, saturation, and value numbers, each ranging from 0 to 1. A hue of
// 0 is red, while 1 is also red, but on the other end of the spectrum.
// Cribbed from: https://github.com/lucasb-eyer/go-colorful/blob/master/colors.go
func NewColorFromHSV(h, s, v float64) *Color {

	for h > 1 {
		h--
	}
	for h < 0 {
		h++
	}

	h *= 360

	if s > 1 {
		s = 1
	} else if s < 0 {
		s = 0
	}

	if v > 1 {
		v = 1
	} else if v < 0 {
		v = 0
	}

	Hp := h / 60.0
	C := v * s
	X := C * (1.0 - math.Abs(math.Mod(Hp, 2.0)-1.0))

	m := v - C
	r, g, b := 0.0, 0.0, 0.0

	switch {
	case 0.0 <= Hp && Hp < 1.0:
		r = C
		g = X
	case 1.0 <= Hp && Hp < 2.0:
		r = X
		g = C
	case 2.0 <= Hp && Hp < 3.0:
		g = C
		b = X
	case 3.0 <= Hp && Hp < 4.0:
		g = X
		b = C
	case 4.0 <= Hp && Hp < 5.0:
		r = X
		b = C
	case 5.0 <= Hp && Hp < 6.0:
		r = C
		b = X
	}

	return &Color{float32(m + r), float32(m + g), float32(m + b), 1}
}

// HSV returns a color as a hue, saturation, and value (each ranging from 0 to 1).
// Also cribbed from: https://github.com/lucasb-eyer/go-colorful/blob/master/colors.go
func (color *Color) HSV() (float64, float64, float64) {

	r := float64(color.R)
	g := float64(color.G)
	b := float64(color.B)

	min := math.Min(math.Min(r, g), b)
	v := math.Max(math.Max(r, g), b)
	C := v - min

	s := 0.0
	if v != 0.0 {
		s = C / v
	}

	h := 0.0 // We use 0 instead of undefined as in wp.
	if min != v {
		if v == r {
			h = math.Mod((g-b)/C, 6.0)
		}
		if v == g {
			h = (b-r)/C + 2.0
		}
		if v == b {
			h = (r-g)/C + 4.0
		}
		h *= 60.0
		if h < 0.0 {
			h += 360.0
		}
	}
	return h / 360, s, v
}
