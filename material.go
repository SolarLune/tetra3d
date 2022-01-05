package tetra3d

import "github.com/hajimehoshi/ebiten/v2"

const (
	TriangleSortBackToFront = iota
	TriangleSortFrontToBack
	TriangleSortNone
)

type Material struct {
	Name             string
	Image            *ebiten.Image
	Tags             *Tags
	BackfaceCulling  bool
	TriangleSortMode int
	EnableLighting   bool
}

func NewMaterial(name string) *Material {
	return &Material{
		Name:             name,
		Tags:             NewTags(),
		BackfaceCulling:  true,
		TriangleSortMode: TriangleSortBackToFront,
		EnableLighting:   true,
	}
}

func (material *Material) Clone() *Material {
	newMat := NewMaterial(material.Name)
	newMat.Image = material.Image
	newMat.Tags = material.Tags.Clone()
	newMat.BackfaceCulling = material.BackfaceCulling
	newMat.TriangleSortMode = material.TriangleSortMode
	newMat.EnableLighting = material.EnableLighting
	return newMat
}
