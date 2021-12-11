package tetra3d

import "math"

// A Color represents a color, containing R, G, B, and A components, each expected to range from 0 to 1.
type Color struct {
	R, G, B, A float32
}

// NewColor returns a new Color, with the provided R, G, B, and A components expected to range from 0 to 1.
func NewColor(r, g, b, a float32) *Color {
	return &Color{r, g, b, a}
}

func (color *Color) Clone() *Color {
	return NewColor(color.R, color.G, color.B, color.A)
}

func (color *Color) SetRGBA(r, g, b, a float32) {
	color.R = r
	color.G = g
	color.B = b
	color.A = a
}

func (color *Color) AddRGB(value float32) {
	color.R += value
	color.G += value
	color.B += value
}

func (color Color) RGBA64() (float64, float64, float64, float64) {
	return float64(color.R), float64(color.G), float64(color.B), float64(color.A)
}

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
