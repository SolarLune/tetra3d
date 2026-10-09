package tetra3d

import (
	"maps"

	"github.com/hajimehoshi/ebiten/v2"
)

// drawScratch holds the values that Camera.Render reuses from one draw call
// and one frame to the next, so that the draw calls make no allocations for
// their uniforms, options, and render lists.
//
// Each uniform map gets its keys and values once, when it is made. Every value
// is a slice, and the draw calls write the new values into the slices, so the
// maps and their interface values never change. Ebitengine reads a slice
// whose length equals the dword count of the uniform in the same way as a
// scalar, so the uniform data is the same as with a new map of scalars.
type drawScratch struct {
	solids, transparents []renderPair

	colorShaderOptions ebiten.DrawTrianglesShaderOptions
	colorOptions       ebiten.DrawTrianglesOptions
	depthOptions       ebiten.DrawTrianglesShaderOptions
	clipOptions        ebiten.DrawTrianglesShaderOptions

	// worldUniforms is for a scene with a World, plainUniforms for a scene
	// without one, clipUniforms for the alpha-clip depth pass, and
	// fragmentUniforms for the colour pass of a material with its own
	// fragment shader uniforms.
	worldUniforms, plainUniforms, clipUniforms, fragmentUniforms map[string]any

	fog, fogRange, ditherSize, fogCurve, fogless []float32
	textureMapScreenSizeW, textureMapScreenSizeH []float32

	perspectiveCorrection, textureFilterMode, textureMapMode []int

	// foglessValue holds fogless. foglessNormal is the value of Fogless in a
	// normal render, an int like the literal 1 that it stands for.
	foglessValue, foglessNormal any
}

func newDrawScratch() *drawScratch {
	s := &drawScratch{
		fog:                   make([]float32, 4),
		fogRange:              make([]float32, 2),
		ditherSize:            make([]float32, 1),
		fogCurve:              make([]float32, 1),
		fogless:               make([]float32, 1),
		textureMapScreenSizeW: make([]float32, 1),
		textureMapScreenSizeH: make([]float32, 1),
		perspectiveCorrection: make([]int, 1),
		textureFilterMode:     make([]int, 1),
		textureMapMode:        make([]int, 1),
		fragmentUniforms:      map[string]any{},
	}
	s.foglessValue = s.fogless
	s.foglessNormal = []int{1}
	s.worldUniforms = map[string]any{
		"Fog":                             s.fog,
		"FogRange":                        s.fogRange,
		"DitherSize":                      s.ditherSize,
		"FogCurve":                        s.fogCurve,
		"BayerMatrix":                     bayerMatrix,
		"PerspectiveCorrection":           s.perspectiveCorrection,
		"TextureFilterMode":               s.textureFilterMode,
		"TextureMapMode":                  s.textureMapMode,
		"TextureMapScreenSizeMultiplierW": s.textureMapScreenSizeW,
		"TextureMapScreenSizeMultiplierH": s.textureMapScreenSizeH,
		"Fogless":                         s.foglessValue,
	}
	s.plainUniforms = map[string]any{
		"Fog":                   []float32{0, 0, 0, 0},
		"FogRange":              []float32{0, 1},
		"PerspectiveCorrection": s.perspectiveCorrection,
		"Fogless":               s.foglessValue,
	}
	s.clipUniforms = map[string]any{
		"PerspectiveCorrection":           s.perspectiveCorrection,
		"TextureMapMode":                  s.textureMapMode,
		"TextureMapScreenSizeMultiplierW": s.textureMapScreenSizeW,
		"TextureMapScreenSizeMultiplierH": s.textureMapScreenSizeH,
		"TextureFilterMode":               s.textureFilterMode,
	}
	s.clipOptions.Uniforms = s.clipUniforms
	return s
}

// setPartUniforms writes the uniform values of one mesh part, which the
// alpha-clip depth pass and the colour pass share.
func (s *drawScratch) setPartUniforms(perspectiveCorrection, textureFilterMode, textureMapMode int, textureMapScreenSizeW, textureMapScreenSizeH float32) {
	s.perspectiveCorrection[0] = perspectiveCorrection
	s.textureFilterMode[0] = textureFilterMode
	s.textureMapMode[0] = textureMapMode
	s.textureMapScreenSizeW[0] = textureMapScreenSizeW
	s.textureMapScreenSizeH[0] = textureMapScreenSizeH
}

// colorUniforms returns the uniform map of the colour pass for world, which
// can be nil, with Fogless set to fogless, or to the int 1 in a normal render.
func (s *drawScratch) colorUniforms(world *World, fogless float32, normals bool) map[string]any {
	m := s.plainUniforms
	if world != nil {
		m = s.worldUniforms
		s.fog[0] = float32(world.FogColor.R)
		s.fog[1] = float32(world.FogColor.G)
		s.fog[2] = float32(world.FogColor.B)
		s.fog[3] = float32(world.FogMode)
		if !world.FogOn {
			s.fog[3] = -1
		}
		// FogRange goes to Ebitengine as the World's own slice would, so a
		// slice of another length still fails in the same way.
		if len(world.FogRange) != len(s.fogRange) {
			s.fogRange = make([]float32, len(world.FogRange))
			m["FogRange"] = s.fogRange
		}
		copy(s.fogRange, world.FogRange)
		s.ditherSize[0] = world.DitheredFogSize
		s.fogCurve[0] = float32(world.FogCurve)
	}
	s.fogless[0] = fogless
	if normals {
		m["Fogless"] = s.foglessNormal
	} else {
		m["Fogless"] = s.foglessValue
	}
	return m
}

// withFragmentUniforms returns a map with the entries of base and then of
// extra, the uniforms of a material's fragment shader. It reuses one map, so
// that the keys of one material do not stay in base for the next part.
func (s *drawScratch) withFragmentUniforms(base, extra map[string]any) map[string]any {
	clear(s.fragmentUniforms)
	maps.Copy(s.fragmentUniforms, base)
	maps.Copy(s.fragmentUniforms, extra)
	return s.fragmentUniforms
}
