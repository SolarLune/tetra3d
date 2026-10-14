package tetra3d

import (
	"math"
	"reflect"
	"slices"
	"sort"
	"testing"
)

// uniformBits converts each uniform value to its dwords in the way that
// Ebitengine's ui.Shader.AppendUniforms does, for the kinds that the colour
// and depth passes use.
func uniformBits(t *testing.T, m map[string]any) map[string][]uint32 {
	t.Helper()
	out := map[string][]uint32{}
	for name, value := range m {
		v := reflect.ValueOf(value)
		one := func(e reflect.Value) uint32 {
			switch e.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return uint32(e.Int())
			case reflect.Float32, reflect.Float64:
				return math.Float32bits(float32(e.Float()))
			}
			t.Fatalf("%s: unexpected kind %s", name, e.Kind())
			return 0
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			bits := make([]uint32, v.Len())
			for i := range bits {
				bits[i] = one(v.Index(i))
			}
			out[name] = bits
		} else {
			out[name] = []uint32{one(v)}
		}
	}
	return out
}

// literalColorUniforms builds the colour-pass map as Camera.Render did before
// it reused one map, as the reference for the reused maps.
func literalColorUniforms(world *World, perspectiveCorrection, textureFilterMode, textureMapMode int, w, h, fogless float32, normals bool, extra map[string]any) map[string]any {
	var m map[string]any
	if world != nil {
		m = map[string]any{
			"Fog":                             world.fogAsFloatSlice(),
			"FogRange":                        world.FogRange,
			"DitherSize":                      world.DitheredFogSize,
			"FogCurve":                        float32(world.FogCurve),
			"BayerMatrix":                     bayerMatrix,
			"PerspectiveCorrection":           perspectiveCorrection,
			"TextureFilterMode":               textureFilterMode,
			"TextureMapMode":                  textureMapMode,
			"TextureMapScreenSizeMultiplierW": w,
			"TextureMapScreenSizeMultiplierH": h,
		}
	} else {
		m = map[string]any{
			"Fog":                   []float32{0, 0, 0, 0},
			"FogRange":              []float32{0, 1},
			"PerspectiveCorrection": perspectiveCorrection,
		}
	}
	if world == nil || !world.FogOn {
		fogless = 1
	}
	if normals {
		m["Fogless"] = 1
	} else {
		m["Fogless"] = fogless
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

// TestReusedUniformsMatchLiteralMaps checks that the reused uniform maps give
// Ebitengine the same dwords as a new map for each part, over a sequence of
// parts in which a material's own uniforms must not stay for the next part.
func TestReusedUniformsMatchLiteralMaps(t *testing.T) {
	world := NewWorld("world")
	world.FogColor = NewColor4(0.3, 0.55, 0.9, 1)
	world.FogMode = FogBlendMode(2)
	world.DitheredFogSize = 0.25
	world.FogCurve = FogCurve(1)
	world.FogRange = []float32{0.1875, 1}
	noFog := NewWorld("no fog")
	noFog.FogOn = false
	noFog.FogRange = []float32{0.5, 0.75}
	paint := map[string]any{"Paint": []float32{0.8, 0.1, 0.1, 1}, "Fog": []float32{1, 2, 3, 4}}

	type part struct {
		world         *World
		pc, tf, tm    int
		w, h, fogless float32
		normals       bool
		extra         map[string]any
	}
	parts := []part{
		{world: world, tf: 1, w: 1, h: 1},
		{world: world, pc: 1, tf: 0, tm: 2, w: 0.5, h: 2, fogless: 1, extra: paint},
		{world: world, tf: 1, w: 1, h: 1},
		{world: noFog, tm: 1, w: 3, h: 0.25, fogless: 1},
		{world: nil, pc: 1, w: 1, h: 1, fogless: 1, extra: paint},
		{world: nil, pc: 0, w: 1, h: 1},
		{world: world, tf: 2, w: 1, h: 1, normals: true},
		{world: world, tf: 2, w: 1, h: 1, fogless: 1},
	}

	s := newDrawScratch()
	for i, p := range parts {
		s.setPartUniforms(p.pc, p.tf, p.tm, p.w, p.h)
		got := s.colorUniforms(p.world, p.fogless, p.normals)
		if p.extra != nil {
			got = s.withFragmentUniforms(got, p.extra)
		}
		want := literalColorUniforms(p.world, p.pc, p.tf, p.tm, p.w, p.h, p.fogless, p.normals, p.extra)
		if g, w := uniformBits(t, got), uniformBits(t, want); !reflect.DeepEqual(g, w) {
			t.Fatalf("part %d: uniforms differ:\n got %v\nwant %v", i, g, w)
		}

		clip := map[string]any{
			"PerspectiveCorrection":           p.pc,
			"TextureMapMode":                  p.tm,
			"TextureMapScreenSizeMultiplierW": p.w,
			"TextureMapScreenSizeMultiplierH": p.h,
			"TextureFilterMode":               p.tf,
		}
		if g, w := uniformBits(t, s.clipOptions.Uniforms), uniformBits(t, clip); !reflect.DeepEqual(g, w) {
			t.Fatalf("part %d: clip uniforms differ:\n got %v\nwant %v", i, g, w)
		}
	}
}

func TestReusedUniformsDoNotAllocate(t *testing.T) {
	world := NewWorld("world")
	paint := map[string]any{"Paint": []float32{0.8, 0.1, 0.1, 1}}
	s := newDrawScratch()
	allocs := testing.AllocsPerRun(100, func() {
		s.setPartUniforms(1, 1, 0, 1, 1)
		base := s.colorUniforms(world, 1, false)
		s.withFragmentUniforms(base, paint)
		s.colorUniforms(nil, 0, true)
	})
	if allocs != 0 {
		t.Fatalf("%v allocations for each part, want 0", allocs)
	}
}

// TestFogOffSkipsFog checks that a World with its fog off sets Fogless, and
// gives the pixels of the fog work that it skips: with the fog on and a fog
// mode that no branch of the shader matches, which is the state that fog off
// gave before.
func TestFogOffSkipsFog(t *testing.T) {
	scene := NewScene("fog")
	scene.World.LightingOn = false
	scene.World.FogColor = NewColor4(0.3, 0.55, 0.9, 1)
	scene.World.DitheredFogSize = 0.5
	mesh := NewCubeMesh(2, 2, 2)
	mesh.MeshParts[0].Material.Color = NewColor4(0.8, 0.4, 0.2, 1)
	mesh.MeshParts[0].Material.Shadeless = true
	model := NewModel("cube", mesh)
	model.SetLocalPosition(0, 0, -6)
	cam := NewCamera("camera", 64, 64)
	scene.Root.AddChildren(cam, model)
	read := func(fogOn bool, mode FogBlendMode) []byte {
		scene.World.FogOn, scene.World.FogMode = fogOn, mode
		pix := make([]byte, 4*64*64)
		inFrame(t, func() {
			cam.Clear()
			cam.RenderScene(scene)
			cam.ColorTexture().ReadPixels(pix)
		})
		return pix
	}
	before := read(true, -1)
	if cam.draw.fogless[0] != 0 {
		t.Fatalf("fog on: Fogless %v, want 0", cam.draw.fogless[0])
	}
	after := read(false, FogAdd)
	if cam.draw.fogless[0] != 1 {
		t.Fatalf("fog off: Fogless %v, want 1", cam.draw.fogless[0])
	}
	if !slices.Equal(after, before) {
		t.Error("fog off changes the pixels")
	}
	covered := 0
	for i := 0; i < len(after); i += 4 {
		if !slices.Equal(after[i:i+4], after[:4]) {
			covered++
		}
	}
	if covered < 300 {
		t.Fatalf("the cube covers %d pixels, want at least 300", covered)
	}
}

// TestCompareTransparentsMatchesLess checks that slices.SortStableFunc with
// compareTransparents gives the order of the sort.SliceStable less function
// that it replaces, with ties in order and depth, and a NaN depth.
func TestCompareTransparentsMatchesLess(t *testing.T) {
	var pairs []renderPair
	for i := range 300 {
		depth := float32((i * 7919) % 13)
		if i%37 == 0 {
			depth = float32(math.NaN())
		}
		pairs = append(pairs, renderPair{order: (i * 31) % 4, depth: depth, MeshPart: &MeshPart{TriangleStart: i}})
	}
	want := slices.Clone(pairs)
	less := func(i, j int) bool {
		if want[i].order == want[j].order {
			return want[i].depth > want[j].depth
		}
		return want[i].order < want[j].order
	}
	sort.SliceStable(want, less)
	got := slices.Clone(pairs)
	slices.SortStableFunc(got, compareTransparents)
	for i := range got {
		if got[i].MeshPart != want[i].MeshPart {
			t.Fatalf("position %d: got part %d, want part %d", i, got[i].MeshPart.TriangleStart, want[i].MeshPart.TriangleStart)
		}
	}
}
