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
	// If a material is tagged as transparent, it's rendered in a separate render pass.
	// Objects with transparent materials don't render to the depth texture and are sorted and rendered back-to-front, AFTER
	// all non-transparent materials.
	Transparent bool
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
	newMat.Transparent = true
	return newMat
}
