package colors

// package colors contains functions to quickly and easily generate tetra3d.Color instances by name (i.e. "White()", "Blue()", "Green()", etc).

import "github.com/solarlune/tetra3d"

// Transparent generates a tetra3d.Color instance of the provided name.
func Transparent() tetra3d.Color {
	return tetra3d.NewColor(0, 0, 0, 0)
}

// White generates a tetra3d.Color instance of the provided name.
func White() tetra3d.Color {
	return tetra3d.NewColor(1, 1, 1, 1)
}

// Black generates a tetra3d.Color instance of the provided name.
func Black() tetra3d.Color {
	return tetra3d.NewColor(0, 0, 0, 1)
}

// Gray generates a tetra3d.Color instance of the provided name.
func Gray() tetra3d.Color {
	return tetra3d.NewColor(0.5, 0.5, 0.5, 1)
}

// LightGray generates a tetra3d.Color instance of the provided name.
func LightGray() tetra3d.Color {
	return tetra3d.NewColor(0.8, 0.8, 0.8, 1)
}

// DarkGray generates a tetra3d.Color instance of the provided name.
func DarkGray() tetra3d.Color {
	return tetra3d.NewColor(0.2, 0.2, 0.2, 1)
}

// DarkGray generates a tetra3d.Color instance of the provided name.
func DarkestGray() tetra3d.Color {
	return tetra3d.NewColor(0.05, 0.05, 0.05, 1)
}

// Red generates a tetra3d.Color instance of the provided name.
func Red() tetra3d.Color {
	return tetra3d.NewColor(1, 0, 0, 1)
}

// PaleRed generates a tetra3d.Color instance of the provided name.
func PaleRed() tetra3d.Color {
	return tetra3d.NewColor(0.678, 0.172, 0.384, 1)
}

// Orange generates a tetra3d.Color instance of the provided name.
func Orange() tetra3d.Color {
	return tetra3d.NewColor(1, 0.5, 0, 1)
}

// Yellow generates a tetra3d.Color instance of the provided name.
func Yellow() tetra3d.Color {
	return tetra3d.NewColor(1, 1, 0, 1)
}

// Green generates a tetra3d.Color instance of the provided name.
func Green() tetra3d.Color {
	return tetra3d.NewColor(0, 1, 0, 1)
}

// SkyBlue generates a tetra3d.Color instance of the provided name.
func SkyBlue() tetra3d.Color {
	return tetra3d.NewColor(0, 0.5, 1, 1)
}

// Turquoise generates a tetra3d.Color instance of the provided name.
func Turquoise() tetra3d.Color {
	return tetra3d.NewColor(0, 1, 1, 1)
}

// Blue generates a tetra3d.Color instance of the provided name.
func Blue() tetra3d.Color {
	return tetra3d.NewColor(0, 0, 1, 1)
}

// Pink generates a tetra3d.Color instance of the provided name.
func Pink() tetra3d.Color {
	return tetra3d.NewColor(1, 0, 1, 1)
}

// Purple generates a tetra3d.Color instance of the provided name.
func Purple() tetra3d.Color {
	return tetra3d.NewColor(0.5, 0, 1, 1)
}
