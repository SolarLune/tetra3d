package tetra3d

import (
	"image/color"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
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

func (color *Color) RandomizeRGB(min, max float32, grayscale bool) *Color {
	diff := max - min
	if grayscale {
		r := min + (diff * rand.Float32())
		color.R = r
		color.G = r
		color.B = r
	} else {
		color.R = min + (diff * rand.Float32())
		color.G = min + (diff * rand.Float32())
		color.B = min + (diff * rand.Float32())
	}
	return color
}

// Set sets the RGBA components of the Color to the r, g, b, and a arguments provided. The components are expected to range from 0 to 1.
func (color *Color) Set(r, g, b, a float32) *Color {
	color.R = r
	color.G = g
	color.B = b
	color.A = a
	return color
}

// AddRGBA adds the provided R, G, B, and A values to the color as provided. The components are expected to range from 0 to 1.
func (color *Color) AddRGBA(r, g, b, a float32) *Color {
	color.R += r
	color.G += g
	color.B += b
	color.A += a
	return color
}

// Add adds the provided Color to the existing Color.
func (color *Color) Add(other *Color) *Color {
	return color.AddRGBA(other.ToFloat32s())
}

// MultiplyRGBA multiplies the color's RGBA channels by the provided R, G, B, and A scalar values.
func (color *Color) MultiplyRGBA(scalarR, scalarG, scalarB, scalarA float32) *Color {
	color.R *= scalarR
	color.G *= scalarG
	color.B *= scalarB
	color.A *= scalarA
	return color
}

// Multiply multiplies the existing Color by the provided Color.
func (color *Color) Multiply(other *Color) *Color {
	return color.MultiplyRGBA(other.ToFloat32s())
}

// Sub subtracts the other Color from the calling Color instance.
func (color *Color) SubRGBA(r, g, b, a float32) *Color {
	color.R -= r
	color.G -= g
	color.B -= b
	color.A -= a
	return color
}

// Sub subtracts the other Color from the calling Color instance.
func (color *Color) Sub(other *Color) *Color {
	return color.SubRGBA(other.ToFloat32s())
}

// Mix mixes the calling Color with the other Color, mixed to the percentage given (ranging from 0 - 1).
func (color *Color) Mix(other *Color, percentage float32) *Color {

	color.R += (other.R - color.R) * percentage
	color.G += (other.G - color.G) * percentage
	color.B += (other.B - color.B) * percentage
	color.A += (other.A - color.A) * percentage
	return color

}

// SetAlpha sets the Color's alpha to the provided alpha value.
func (color *Color) SetAlpha(alpha float32) *Color {
	color.A = alpha
	return color
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

func NewColorFromHexString(hex string) *Color {

	c := NewColor(0, 0, 0, 1)

	hex = strings.TrimPrefix(hex, "#")

	if len(hex) >= 2 {
		v, _ := strconv.ParseInt(hex[:2], 16, 32)
		c.R = float32(v) / 256.0

		if len(hex) >= 4 {
			v, _ := strconv.ParseInt(hex[2:4], 16, 32)
			c.G = float32(v) / 256.0
		} else {
			c.G = 1
		}

		if len(hex) >= 6 {
			v, _ := strconv.ParseInt(hex[4:6], 16, 32)
			c.B = float32(v) / 256.0
		} else {
			c.B = 1
		}

		if len(hex) >= 8 {
			v, _ := strconv.ParseInt(hex[6:8], 16, 32)
			c.A = float32(v) / 256.0
		} else {
			c.A = 1
		}

	}

	return c

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

// ColorCurvePoint indicates an individual color point in a color curve.
type ColorCurvePoint struct {
	Color      *Color
	Percentage float64
}

// Clone creates a duplicated the ColorcurvePoint.
func (point *ColorCurvePoint) Clone() *ColorCurvePoint {
	return &ColorCurvePoint{
		Color:      point.Color,
		Percentage: point.Percentage,
	}
}

// ColorCurve represents a range of colors that a value of 0 to 1 can interpolate between.
type ColorCurve struct {
	Points []*ColorCurvePoint
}

// NewColorCurve creats a new ColorCurve.
func NewColorCurve() *ColorCurve {
	return &ColorCurve{
		Points: []*ColorCurvePoint{},
	}
}

// Clone creates a duplicate ColorCurve.
func (cc *ColorCurve) Clone() *ColorCurve {
	ncc := NewColorCurve()
	for _, point := range cc.Points {
		ncc.Points = append(ncc.Points, point.Clone())
	}
	return ncc
}

// Add adds a color point to the ColorCurve with the color and percentage provided (from 0-1).
func (cc *ColorCurve) Add(color *Color, percentage float64) {
	cc.AddRGBA(color.R, color.G, color.B, color.A, percentage)
}

// AddRGBA adds a point to the ColorCurve, with r, g, b, and a being the color and the percentage being a number between 0 and 1 indicating the .
func (cc *ColorCurve) AddRGBA(r, g, b, a float32, percentage float64) {

	if percentage > 1 {
		percentage = 1
	} else if percentage < 0 {
		percentage = 0
	}

	cc.Points = append(cc.Points, &ColorCurvePoint{
		Color:      NewColor(r, g, b, a),
		Percentage: percentage,
	})

	sort.Slice(cc.Points, func(i, j int) bool { return cc.Points[i].Percentage < cc.Points[j].Percentage })
}

// Color returns the Color for the given percentage in the color curve. For example, if you have a curve composed of
// the colors {0, 0, 0, 0} at 0 and {1, 1, 1, 1} at 1, then calling Curve.Color(0.5) would return {0.5, 0.5, 0.5, 0.5}.
func (cc *ColorCurve) Color(perc float64) *Color {

	if perc > 1 {
		perc = 1
	} else if perc < 0 {
		perc = 0
	}

	var c *Color

	for i := 0; i < len(cc.Points); i++ {

		c = cc.Points[i].Color

		if i >= len(cc.Points)-1 || cc.Points[i].Percentage >= perc {
			break
		}

		if cc.Points[i].Percentage <= perc && cc.Points[i+1].Percentage >= perc {
			c = cc.Points[i].Color.Clone()
			c.Mix(cc.Points[i+1].Color, float32((perc-cc.Points[i].Percentage)/(cc.Points[i+1].Percentage-cc.Points[i].Percentage)))
			break
		}

	}

	return c

}
