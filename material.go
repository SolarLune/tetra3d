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

const (
	TextureMapModeUV     = iota // Default texture mapping mode. The UV values on the model impact where the texture renders.
	TextureMapModeScreen        // Screen-space texture mapping mode. The texture renders plainly to the screen.
)

const (
	DepthModeDefault             = iota // Default custom depth mode; unaltered depth writing
	CustomDepthModeUnbillboarded        // Unbillboarded depth mode; depth is written as though the mesh were not transformed
)

type Material struct {
	library           *Library       // library is a reference to the Library that this Material came from.
	name              string         // Name is the name of the Material.
	Color             Color          // The overall color of the Material.
	Texture           *ebiten.Image  // The texture applied to the Material.
	UseTexture        bool           // Whether to use the texture while rendering or not.
	TexturePath       string         // The path to the texture, if it was not packed into the exporter.
	TextureFilterMode ebiten.Filter  // Texture filtering mode
	textureWrapMode   ebiten.Address // Texture wrapping mode; this is ignored currently, as all triangles render through shaders, where looping is enforced.

	TextureMapMode         int     // Texture mapping mode.
	TextureMapScreenSize   float32 // The size multiplier of the texture when mapping to the screen in pixels. 1.0 means a 32px texture will appear 32px onscreen.
	TextureMapScreenOffset Vector2 // Offset for screenspace texture mapping in pixels.

	properties       Properties   // Properties allows you to specify auxiliary data on the Material. This is loaded from GLTF files or Blender's Custom Properties if the setting is enabled on the export menu.
	BackfaceCulling  bool         // If backface culling is enabled (which it is by default), faces turned away from the camera aren't rendered.
	TriangleSortMode int          // TriangleSortMode influences how triangles with this Material are sorted.
	Shadeless        bool         // If the material should be shadeless (unlit) or not
	Fogless          bool         // If the material should be fogless or not
	Blend            ebiten.Blend // Blend mode to use when rendering the material (i.e. additive, multiplicative, etc)
	BillboardMode    int          // Billboard mode
	Visible          bool         // Whether the material is visible or not

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

	DepthMode    int // What depth mode to use for writing depth for this material
	LightingMode int // How materials are lit

	// CustomDepthFunction is a customizeable function that takes the depth value of each vertex of a rendered MeshPart and
	// optionally transforms it, returning a different value.
	// A good use for this would be to render sprites on billboarded planes with a higher or fixed depth, thereby fixing them
	// "cutting" into geometry that's further back.
	// model is a reference to the rendering model, camera the rendering camera, meshPart the rendering mesh part, vertIndex
	// the depth for the vertex being rendered of the given index, and originalDepth being the vertex's original transformed
	// depth. The function should return the ideal depth of the vertex from the camera in world units.
	// The default value for CustomDepthFunction is nil.
	CustomDepthFunction func(model *Model, camera *Camera, meshPart *MeshPart, vertIndex int, originalDepth float32) float32
}

// NewMaterial creates a new Material with the name given.
func NewMaterial(name string) *Material {
	return &Material{
		name:                  name,
		Color:                 NewColor(1, 1, 1, 1),
		properties:            NewProperties(),
		TextureFilterMode:     ebiten.FilterNearest,
		textureWrapMode:       ebiten.AddressRepeat,
		BackfaceCulling:       true,
		UseTexture:            true,
		TriangleSortMode:      TriangleSortModeBackToFront,
		TransparencyMode:      TransparencyModeAuto,
		FragmentShaderOptions: &ebiten.DrawTrianglesShaderOptions{},
		FragmentShaderOn:      true,
		Blend:                 ebiten.BlendSourceOver,
		Visible:               true,
		TextureMapScreenSize:  1,
	}
}

// Clone creates a clone of the specified Material. Note that Clone() cannot clone the Material's fragment shader or shader options.
func (m *Material) Clone() *Material {
	newMat := NewMaterial(m.name)
	newMat.library = m.library
	newMat.Color = m.Color

	newMat.Texture = m.Texture
	newMat.UseTexture = m.UseTexture
	newMat.TexturePath = m.TexturePath
	newMat.TextureFilterMode = m.TextureFilterMode
	newMat.textureWrapMode = m.textureWrapMode
	newMat.TextureMapMode = m.TextureMapMode
	newMat.TextureMapScreenSize = m.TextureMapScreenSize
	newMat.TextureMapScreenOffset = m.TextureMapScreenOffset

	newMat.properties = m.properties.Clone()
	newMat.BackfaceCulling = m.BackfaceCulling
	newMat.TriangleSortMode = m.TriangleSortMode
	newMat.Shadeless = m.Shadeless
	newMat.Fogless = m.Fogless
	newMat.Blend = m.Blend
	newMat.BillboardMode = m.BillboardMode
	newMat.Visible = m.Visible

	newMat.SetShaderText(m.fragmentSrc)
	newMat.FragmentShaderOn = m.FragmentShaderOn

	f := *m.FragmentShaderOptions
	newMat.FragmentShaderOptions = &f
	for i := range m.FragmentShaderOptions.Images {
		newMat.FragmentShaderOptions.Images[i] = m.FragmentShaderOptions.Images[i]
	}
	for k, v := range newMat.FragmentShaderOptions.Uniforms {
		newMat.FragmentShaderOptions.Uniforms[k] = v
	}

	newMat.TransparencyMode = m.TransparencyMode
	newMat.DepthMode = m.DepthMode
	newMat.LightingMode = m.LightingMode

	newMat.CustomDepthFunction = m.CustomDepthFunction

	return newMat
}

// Name returns the name of the material.
func (m *Material) Name() string {
	return m.name
}

// SetName sets the name of the material to the specified value.
func (m *Material) SetName(name string) {
	m.name = name
}

// SetShaderText creates a new custom Kage fragment shader for the Material if provided the shader's source code as a []byte.
// This custom shader would be used to render the mesh utilizing the material after rendering to the depth texture, but before
// compositing the finished render to the screen after fog. If the shader is nil, the Material will render using the default Tetra3D
// render setup (e.g. texture, UV values, vertex colors, and vertex lighting).
// SetShader will return the Shader, and an error if the Shader failed to compile.
// Note that custom shaders require usage of pixel-unit Kage shaders.
func (m *Material) SetShaderText(src []byte) (*ebiten.Shader, error) {

	if m.fragmentShader != nil {
		m.fragmentShader.Dispose()
		m.fragmentShader = nil
	}

	if src == nil {
		m.fragmentShader = nil
		m.fragmentSrc = nil
		return nil, nil
	}

	newShader, err := ebiten.NewShader(src)
	if err != nil {
		return nil, err
	}

	m.fragmentShader = newShader
	m.fragmentSrc = src

	return m.fragmentShader, nil

}

// SetShader sets an already-compiled custom Kage shader to the Material.
// By default, a custom shader will render on top of everything and with no fog.
// Lighting will also be missing, but that's included in the Model's vertex color.
// If you want to extend the base 3D shader, use tetra3d.ExtendBase3DShader().
// Note that custom shaders require usage of pixel-unit Kage shaders.
func (m *Material) SetShader(shader *ebiten.Shader) {
	if m.fragmentShader != shader {
		m.fragmentShader = shader
		m.fragmentSrc = nil
	}
}

// Shader returns the custom Kage fragment shader for the Material.
func (m *Material) Shader() *ebiten.Shader {
	return m.fragmentShader
}

// DisposeShader disposes the custom fragment Shader for the Material (assuming it has one). If it does not have a Shader, nothing happens.
func (m *Material) DisposeShader() {
	if m.fragmentShader != nil {
		m.fragmentShader.Dispose()
	}
	m.fragmentSrc = nil
	m.fragmentShader = nil
}

// Library returns the Library from which this Material was loaded. If it was created through code, this function will return nil.
func (m *Material) Library() *Library {
	return m.library
}

func (m *Material) String() string {
	if ReadableReferences {
		return "<" + m.name + ">"
	} else {
		return fmt.Sprintf("%p", m)
	}
}

// Properties returns this Material's game Properties struct.
func (m *Material) Properties() Properties {
	return m.properties
}
