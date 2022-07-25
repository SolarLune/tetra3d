package tetra3d

const (
	FogOff       = iota // No fog
	FogAdd              // Additive blended fog
	FogMultiply         // Multiplicative blended fog
	FogOverwrite        // Color overwriting fog (mixing base with fog color over depth distance)
)

type FogMode int

// World represents a collection of settings that one uses to control lighting and ambience. This includes the screen clear color, fog color,
// mode, and range, whether lighting is globally enabled or not, and finally the ambient lighting level (using the World's AmbientLight).
type World struct {
	Name       string
	ClearColor *Color // The clear color of the screen; note that this doesn't clear the color of the camera buffer or screen automatically;
	// this is just what the color is if the scene was exported using the Tetra3D addon from Blender. It's up to you as to how you'd like to
	// use it.
	FogColor *Color  // The Color of any fog present in the Scene.
	FogMode  FogMode // The FogMode, indicating how the fog color is blended if it's on (not FogOff).
	// FogRange is the depth range at which the fog is active. FogRange consists of two numbers,
	// ranging from 0 to 1. The first indicates the start of the fog, and the second the end, in
	// terms of total depth of the near / far clipping plane. The default is [0, 1].
	FogRange     []float32
	LightingOn   bool          // If lighting is enabled when rendering the scene.
	AmbientLight *AmbientLight // Ambient lighting for this world
}

// NewWorld creates a new World with the specified name and default values for fog, lighting, etc).
func NewWorld(name string) *World {

	return &World{
		Name:         name,
		FogColor:     NewColor(0, 0, 0, 0),
		FogRange:     []float32{0, 1},
		LightingOn:   true,
		ClearColor:   NewColor(0.08, 0.09, 0.1, 1),
		AmbientLight: NewAmbientLight("ambient light", 1, 1, 1, 0),
	}

}

// Clone returns a new World with the same properties as the existing World.
func (world *World) Clone() *World {

	newWorld := NewWorld(world.Name)

	newWorld.ClearColor = world.ClearColor.Clone()
	newWorld.FogColor = world.FogColor.Clone()
	newWorld.FogMode = world.FogMode
	newWorld.FogRange[0] = world.FogRange[0]
	newWorld.FogRange[1] = world.FogRange[1]
	newWorld.LightingOn = world.LightingOn
	newWorld.AmbientLight = world.AmbientLight.Clone().(*AmbientLight)

	return newWorld

}

func (world *World) fogAsFloatSlice() []float32 {

	fog := []float32{
		float32(world.FogColor.R),
		float32(world.FogColor.G),
		float32(world.FogColor.B),
		float32(world.FogMode),
	}

	if world.FogMode == FogMultiply {
		fog[0] = 1 - fog[0]
		fog[1] = 1 - fog[1]
		fog[2] = 1 - fog[2]
	}

	return fog
}
