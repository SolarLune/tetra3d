package tetra3d

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
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
	BillboardModeNone          = iota // No billboarding
	BillboardModeFixedVertical        // Billboard to face forward relative to the camera / screen under all circumstances; up is screen up / up relative to the camera, locally (local +Y)
	BillboardModeHorizontal           // Billboard to face towards the camera, but skews as you go above / below the object)
	BillboardModeAll                  // Billboard on all axes
)

const (
	LightingModeDefault      = iota // Default lighting mode
	LightingModeFixedNormals        // Lighting applies as though faces always point towards light sources; good for 2D sprites
	LightingModeDoubleSided         // Lighting applies for double-sided faces
)

type Material struct {
	library           *Library             // library is a reference to the Library that this Material came from.
	Name              string               // Name is the name of the Material.
	Color             Color                // The overall color of the Material.
	Texture           *ebiten.Image        // The texture applied to the Material.
	TexturePath       string               // The path to the texture, if it was not packed into the exporter.
	TextureFilterMode ebiten.Filter        // Texture filtering mode
	textureWrapMode   ebiten.Address       // Texture wrapping mode; this is ignored currently, as all triangles render through shaders, where looping is enforced.
	properties        Properties           // Properties allows you to specify auxiliary data on the Material. This is loaded from GLTF files or Blender's Custom Properties if the setting is enabled on the export menu.
	BackfaceCulling   bool                 // If backface culling is enabled (which it is by default), faces turned away from the camera aren't rendered.
	TriangleSortMode  int                  // TriangleSortMode influences how triangles with this Material are sorted.
	Shadeless         bool                 // If the material should be shadeless (unlit) or not
	Fogless           bool                 // If the material should be fogless or not
	CompositeMode     ebiten.CompositeMode // Blend mode to use when rendering the material (i.e. additive, multiplicative, etc)
	BillboardMode     int                  // Billboard mode

	// fragmentShader represents a shader used to render the material with. This shader is activated after rendering
	// to the depth texture, but before compositing the finished render to the screen after fog.
	fragmentShader *ebiten.Shader
	// FragmentShaderOn is an easy boolean toggle to control whether the shader is activated or not (it defaults to on).
	FragmentShaderOn bool
	// FragmentShaderOptions allows you to customize the custom fragment shader with uniforms or images.
	// By default, it's an empty DrawTrianglesShaderOptions struct.
	// Note that the first image slot is reserved for the color texture associated with the Material.
	// The second slot is reserved for a depth texture (primarily the intermediate texture used to "cut" a
	// rendered model).
	// If you want a custom fragment shader that already has fog and depth-testing, use Extend3DBaseShader() to
	// extend your custom fragment shader from Tetra3D's base 3D shader.
	FragmentShaderOptions *ebiten.DrawTrianglesShaderOptions
	fragmentSrc           []byte

	// If a material is tagged as transparent, it's rendered in a separate render pass.
	// Objects with transparent materials don't render to the depth texture and are sorted and rendered back-to-front, AFTER
	// all non-transparent materials.
	TransparencyMode int

	CustomDepthOffsetOn    bool    // Whether custom depth offset is on or not.
	CustomDepthOffsetValue float64 // How many world units to offset the depth of the material by.
	LightingMode           int     // How materials are lit

	// CustomDepthFunction is a customizeable function that takes the depth value of each vertex of a rendered MeshPart and
	// transforms it, returning a different value.
	// A good use for this would be to render sprites on billboarded planes with a higher depth, thereby fixing them
	// "cutting" into geometry that's further back.
	// The default value for CustomDepthFunction is nil.
	CustomDepthFunction func(originalDepth float64) float64
}

// NewMaterial creates a new Material with the name given.
func NewMaterial(name string) *Material {
	return &Material{
		Name:                  name,
		Color:                 NewColor(1, 1, 1, 1),
		properties:            NewProperties(),
		TextureFilterMode:     ebiten.FilterNearest,
		textureWrapMode:       ebiten.AddressRepeat,
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
	newMat.Color = material.Color
	newMat.Texture = material.Texture
	newMat.properties = material.properties.Clone()
	newMat.BackfaceCulling = material.BackfaceCulling
	newMat.TriangleSortMode = material.TriangleSortMode
	newMat.Shadeless = material.Shadeless
	newMat.Fogless = material.Fogless
	newMat.TransparencyMode = material.TransparencyMode
	newMat.TextureFilterMode = material.TextureFilterMode
	newMat.textureWrapMode = material.textureWrapMode
	newMat.CompositeMode = material.CompositeMode

	newMat.BillboardMode = material.BillboardMode
	newMat.SetShaderText(material.fragmentSrc)
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

// SetShaderText creates a new custom Kage fragment shader for the Material if provided the shader's source code as a []byte.
// This custom shader would be used to render the mesh utilizing the material after rendering to the depth texture, but before
// compositing the finished render to the screen after fog. If the shader is nil, the Material will render using the default Tetra3D
// render setup (e.g. texture, UV values, vertex colors, and vertex lighting).
// SetShader will return the Shader, and an error if the Shader failed to compile.
// Note that custom shaders require usage of pixel-unit Kage shaders.
func (material *Material) SetShaderText(src []byte) (*ebiten.Shader, error) {

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

// SetShader sets an already-compiled custom Kage shader to the Material.
// By default, a custom shader will render on top of everything and with no fog.
// Lighting will also be missing, but that's included in the Model's vertex color.
// If you want to extend the base 3D shader, use tetra3d.ExtendBase3DShader().
// Note that custom shaders require usage of pixel-unit Kage shaders.
func (material *Material) SetShader(shader *ebiten.Shader) {
	if material.fragmentShader != shader {
		material.fragmentShader = shader
		material.fragmentSrc = nil
	}
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

func (material *Material) String() string {
	if ReadableReferences {
		return "<" + material.Name + ">"
	} else {
		return fmt.Sprintf("%p", material)
	}
}

// Properties returns this Material's game Properties struct.
func (material *Material) Properties() Properties {
	return material.properties
}
