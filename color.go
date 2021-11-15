package tetra3d

// A Color represents a color, containing R, G, B, and A components, each expected to range from 0 to 1.
type Color struct {
	R, G, B, A float32
}

// NewColor returns a new Color.
func NewColor(r, g, b, a float32) Color {
	return Color{r, g, b, a}
}

func (color Color) Clone() Color {
	return NewColor(color.R, color.G, color.B, color.A)
}

func (color *Color) SetRGB(r, g, b float32) {
	color.R = r
	color.G = g
	color.B = b
}

func (color *Color) AddRGB(value float32) {
	color.R += value
	color.G += value
	color.B += value
}

func (color Color) RGBA64() (float64, float64, float64, float64) {
	return float64(color.R), float64(color.G), float64(color.B), float64(color.A)
}
