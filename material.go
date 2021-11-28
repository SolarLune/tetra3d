package tetra3d

import "github.com/hajimehoshi/ebiten/v2"

type Material struct {
	Name  string
	Image *ebiten.Image
}

func NewMaterial(name string) *Material {
	return &Material{
		Name: name,
	}
}
