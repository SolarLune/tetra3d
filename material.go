package tetra3d

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

const (
	TriangleSortBackToFront = iota // TriangleSortBackToFront sorts the triangles from back to front (naturally). This is the default.
	TriangleSortFrontToBack        // TriangleSortFrontToBack sorts the triangles in reverse order.
	TriangleSortNone               // TriangleSortNone doesn't sort the triangles at all; this is the fastest triangle sorting mode.
)

const (
	// TransparencyModeOpaque means the triangles are rendered to the color and depth buffer as normal.
	TransparencyModeOpaque = iota

	// TransparencyModeAlphaClip means the triangles are rendered to the color and depth buffer, using the alpha of the triangles' texture to "cut out" the triangles.
	TransparencyModeAlphaClip

	// TransparencyModeTransparent means the triangles are not rendered to the depth buffer, but are rendered in a second pass after opaque and alpha-clip triangles. They are automatically sorted from back-to-front.
	TransparencyModeTransparent
)

type Material struct {
	library          *Library      // library is a reference to the Library that this Material came from.
	Name             string        // Name is the name of the Material.
	Color            *Color        // The overall color of the Material.
	Image            *ebiten.Image // The texture applied to the Material.
	Tags             *Tags         // Tags is a Tags object, allowing you to specify auxiliary data on the Material. This is loaded from GLTF files if / Blender's Custom Properties if the setting is enabled on the export menu.
	BackfaceCulling  bool          // If backface culling is enabled (which it is by default), faces turned away from the camera aren't rendered.
	TriangleSortMode int           // TriangleSortMode influences how triangles with this Material are sorted.
	EnableLighting   bool

	// VertexProgram is a function that runs on the world position of each vertex position rendered with the material.
	// One can use this to simply transform the vertices of the mesh (note that this is, of course, not as performant as
	// a traditional vertex shader, but is fine for simple / low-poly mesh transformations). This function is run after
	// skinning the vertex (if the material belongs to a mesh that is skinned by an armature).
	VertexProgram func(vector.Vector) vector.Vector

	// ClipProgram is a function that runs on the clipped result of each vertex position rendered with the material.
	ClipProgram func(vector.Vector) vector.Vector
	// If a material is tagged as transparent, it's rendered in a separate render pass.
	// Objects with transparent materials don't render to the depth texture and are sorted and rendered back-to-front, AFTER
	// all non-transparent materials.
	TransparencyMode int
}

// NewMaterial creates a new Material with the name given.
func NewMaterial(name string) *Material {
	return &Material{
		Name:             name,
		Color:            NewColor(1, 1, 1, 1),
		Tags:             NewTags(),
		BackfaceCulling:  true,
		TriangleSortMode: TriangleSortBackToFront,
		EnableLighting:   true,
		TransparencyMode: TransparencyModeOpaque,
	}
}

// Clone creates a clone of the specified Material.
func (material *Material) Clone() *Material {
	newMat := NewMaterial(material.Name)
	newMat.library = material.library
	newMat.Color = material.Color.Clone()
	newMat.Image = material.Image
	newMat.Tags = material.Tags.Clone()
	newMat.BackfaceCulling = material.BackfaceCulling
	newMat.TriangleSortMode = material.TriangleSortMode
	newMat.EnableLighting = material.EnableLighting
	newMat.TransparencyMode = material.TransparencyMode
	return newMat
}

// Library returns the Library from which this Material was loaded. If it was created through code, this function will return nil.
func (material *Material) Library() *Library {
	return material.library
}
