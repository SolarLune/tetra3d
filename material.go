package tetra3d

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

const (
	TriangleSortModeBackToFront = iota // TriangleSortBackToFront sorts the triangles from back to front (naturally). This is the default.
	TriangleSortModeFrontToBack        // TriangleSortFrontToBack sorts the triangles in reverse order.
	TriangleSortModeNone               // TriangleSortNone doesn't sort the triangles at all; this is the fastest triangle sorting mode, while also being the most graphically inaccurate. Usable if triangles don't visually intersect.
)

const (
	// TransparencyModeAuto means it will be opaque if the object or material's alpha >= 1, and transparent otherwise.
	TransparencyModeAuto = iota

	// TransparencyModeOpaque means the triangles are rendered to the color and depth buffer as normal.
	TransparencyModeOpaque

	// TransparencyModeAlphaClip means the triangles are rendered to the color and depth buffer, using the alpha of the triangles' texture to "cut out" the triangles.
	TransparencyModeAlphaClip

	// TransparencyModeTransparent means the triangles are not rendered to the depth buffer, but are rendered in a second pass after opaque and alpha-clip triangles. They are automatically sorted from back-to-front.
	TransparencyModeTransparent
)

const (
	BillboardModeNone = iota
	BillboardModeXZ   // Billboards on just X and Z (so the tilt stays the same)
	BillboardModeAll  // Billboards on all axes
)

type Material struct {
	library           *Library             // library is a reference to the Library that this Material came from.
	Name              string               // Name is the name of the Material.
	Color             *Color               // The overall color of the Material.
	Texture           *ebiten.Image        // The texture applied to the Material.
	TexturePath       string               // The path to the texture, if it was not packed into the exporter.
	TextureFilterMode ebiten.Filter        // Texture filtering mode
	TextureWrapMode   ebiten.Address       // Texture wrapping mode
	Tags              *Tags                // Tags is a Tags object, allowing you to specify auxiliary data on the Material. This is loaded from GLTF files if / Blender's Custom Properties if the setting is enabled on the export menu.
	BackfaceCulling   bool                 // If backface culling is enabled (which it is by default), faces turned away from the camera aren't rendered.
	TriangleSortMode  int                  // TriangleSortMode influences how triangles with this Material are sorted.
	Shadeless         bool                 // If the material should be shadeless (unlit) or not
	CompositeMode     ebiten.CompositeMode // Blend mode to use when rendering the material (i.e. additive, multiplicative, etc)
	BillboardMode     int                  // Billboard mode

	// VertexTransformFunction is a function that runs on the world position of each vertex position rendered with the material.
	// It accepts the vertex position as an argument, along with the index of the vertex in the mesh.
	// One can use this to simply transform vertices of the mesh on CPU (note that this is, of course, not as performant as
	// a traditional GPU vertex shader, but is fine for simple / low-poly mesh transformations).
	// This function is run after skinning the vertex if the material belongs to a mesh that is skinned by an armature.
	// Note that the VertexTransformFunction must return the vector passed.
	VertexTransformFunction func(vertexPosition vector.Vector, vertexIndex int) vector.Vector

	// VertexClipFunction is a function that runs on the clipped result of each vertex position rendered with the material.
	// The function takes the vertex position along with the vertex index in the mesh.
	// This program runs after the vertex position is clipped to screen coordinates.
	// Note that the VertexClipFunction must return the vector passed.
	VertexClipFunction func(vertexPosition vector.Vector, vertexIndex int) vector.Vector

	// fragmentShader represents a shader used to render the material with. This shader is activated after rendering
	// to the depth texture, but before compositing the finished render to the screen after fog.
	fragmentShader *ebiten.Shader
	// FragmentShaderOn is an easy boolean toggle to control whether the shader is activated or not (it defaults to on).
	FragmentShaderOn bool
	// FragmentShaderOptions allows you to customize the custom fragment shader with uniforms or images. It does NOT take the
	// CompositeMode property from the Material's CompositeMode. By default, it's an empty DrawTrianglesShaderOptions struct.
	FragmentShaderOptions *ebiten.DrawTrianglesShaderOptions
	fragmentSrc           []byte

	// If a material is tagged as transparent, it's rendered in a separate render pass.
	// Objects with transparent materials don't render to the depth texture and are sorted and rendered back-to-front, AFTER
	// all non-transparent materials.
	TransparencyMode int
}

// NewMaterial creates a new Material with the name given.
func NewMaterial(name string) *Material {
	return &Material{
		Name:                  name,
		Color:                 NewColor(1, 1, 1, 1),
		Tags:                  NewTags(),
		TextureFilterMode:     ebiten.FilterNearest,
		TextureWrapMode:       ebiten.AddressRepeat,
		BackfaceCulling:       true,
		TriangleSortMode:      TriangleSortModeBackToFront,
		TransparencyMode:      TransparencyModeAuto,
		FragmentShaderOptions: &ebiten.DrawTrianglesShaderOptions{},
		FragmentShaderOn:      true,
		CompositeMode:         ebiten.CompositeModeSourceOver,
	}
}

// Clone creates a clone of the specified Material. Note that Clone() cannot clone the Material's fragment shader or shader options.
func (material *Material) Clone() *Material {
	newMat := NewMaterial(material.Name)
	newMat.library = material.library
	newMat.Color = material.Color.Clone()
	newMat.Texture = material.Texture
	newMat.Tags = material.Tags.Clone()
	newMat.BackfaceCulling = material.BackfaceCulling
	newMat.TriangleSortMode = material.TriangleSortMode
	newMat.Shadeless = material.Shadeless
	newMat.TransparencyMode = material.TransparencyMode
	newMat.TextureFilterMode = material.TextureFilterMode
	newMat.TextureWrapMode = material.TextureWrapMode
	newMat.CompositeMode = material.CompositeMode

	newMat.BillboardMode = material.BillboardMode
	newMat.VertexTransformFunction = material.VertexTransformFunction
	newMat.VertexClipFunction = material.VertexClipFunction
	newMat.SetShader(material.fragmentSrc)
	newMat.FragmentShaderOn = material.FragmentShaderOn

	newMat.FragmentShaderOptions.CompositeMode = material.FragmentShaderOptions.CompositeMode
	newMat.FragmentShaderOptions.FillRule = material.FragmentShaderOptions.FillRule
	for i := range material.FragmentShaderOptions.Images {
		newMat.FragmentShaderOptions.Images[i] = material.FragmentShaderOptions.Images[i]
	}
	for k, v := range newMat.FragmentShaderOptions.Uniforms {
		newMat.FragmentShaderOptions.Uniforms[k] = v
	}

	return newMat
}

// SetShader creates a new custom Kage fragment shader for the Material if provided the shader's source code, provided as a []byte.
// This custom shader would be used to render the mesh utilizing the material after rendering to the depth texture, but before
// compositing the finished render to the screen after fog. If the shader is nil, the Material will render using the default Tetra3D
// render setup (e.g. texture, UV values, vertex colors, and vertex lighting).
// SetShader will return the Shader, and an error if the Shader failed to compile.
func (material *Material) SetShader(src []byte) (*ebiten.Shader, error) {

	if src == nil {
		material.fragmentShader = nil
		material.fragmentSrc = nil
		return nil, nil
	}

	newShader, err := ebiten.NewShader(src)
	if err != nil {
		return nil, err
	}

	material.fragmentShader = newShader
	material.fragmentSrc = src

	return material.fragmentShader, nil

}

// Shader returns the custom Kage fragment shader for the Material.
func (material *Material) Shader() *ebiten.Shader {
	return material.fragmentShader
}

// DisposeShader disposes the custom fragment Shader for the Material (assuming it has one). If it does not have a Shader, nothing happens.
func (material *Material) DisposeShader() {
	if material.fragmentShader != nil {
		material.fragmentShader.Dispose()
	}
	material.fragmentSrc = nil
	material.fragmentShader = nil
}

// Library returns the Library from which this Material was loaded. If it was created through code, this function will return nil.
func (material *Material) Library() *Library {
	return material.library
}
