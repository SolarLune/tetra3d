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

func ColorWhite() *Color {
	return NewColor(1, 1, 1, 1)
}

func ColorBlack() *Color {
	return NewColor(0, 0, 0, 1)
}

func ColorGray() *Color {
	return NewColor(0.5, 0.5, 0.5, 1)
}

func ColorRed() *Color {
	return NewColor(1, 0, 0, 1)
}

func ColorOrange() *Color {
	return NewColor(0.5, 1, 1, 1)
}

func ColorYellow() *Color {
	return NewColor(1, 1, 0, 1)
}

func ColorGreen() *Color {
	return NewColor(0, 1, 0, 1)
}

func ColorBlue() *Color {
	return NewColor(0, 0, 1, 1)
}

func ColorSkyBlue() *Color {
	return NewColor(0, 0.5, 1, 1)
}

func ColorTurquoise() *Color {
	return NewColor(0, 1, 1, 1)
}

func ColorPink() *Color {
	return NewColor(1, 0, 1, 1)
}

func ColorPurple() *Color {
	return NewColor(0.5, 0, 1, 1)
}

// Clone returns a clone of the Color instance.
func (color *Color) Clone() *Color {
	return NewColor(color.R, color.G, color.B, color.A)
}

// Set sets the RGBA components of the Color to the r, g, b, and a arguments provided.
func (color *Color) Set(r, g, b, a float32) {
	color.R = r
	color.G = g
	color.B = b
	color.A = a
}

// AddRGB adds the color value specified to the R, G, and B channels in the color.
func (color *Color) AddRGB(value float32) {
	color.R += value
	color.G += value
	color.B += value
}

// ToFloat64s returns four float64 values for each channel in the Color.
func (color *Color) ToFloat64s() (float64, float64, float64, float64) {
	return float64(color.R), float64(color.G), float64(color.B), float64(color.A)
}

// ToRGBA64 converts a color to a color.RGBA64 instance.
func (c *Color) ToRGBA64() color.RGBA64 {
	return color.RGBA64{
		uint16(c.R * math.MaxUint16),
		uint16(c.G * math.MaxUint16),
		uint16(c.B * math.MaxUint16),
		uint16(c.A * math.MaxUint16),
	}
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
