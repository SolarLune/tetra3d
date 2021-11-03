package ebiten3d

type Color struct {
	R, G, B, A float32
}

func NewColor(r, g, b, a float32) Color {
	return Color{r, g, b, a}
}

func (color Color) Clone() Color {
	return NewColor(color.R, color.G, color.B, color.A)
}

func (color Color) AddRGB(value float32) Color {
	c := color
	c.R += value
	c.G += value
	c.B += value
	// c.A += value
	return c
}

func (color Color) RGBA64() (float64, float64, float64, float64) {
	return float64(color.R), float64(color.G), float64(color.B), float64(color.A)
}
