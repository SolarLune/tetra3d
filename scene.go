package tetra3d

const (
	FogOff       = iota // No fog
	FogAdd              // Additive blended fog
	FogMultiply         // Multiplicative blended fog
	FogOverwrite        // Color overwriting fog (mixing base with fog color over depth distance)
)

type FogMode int

// Scene represents a world of sorts, and can contain a variety of Meshes and Nodes, which organize the scene into a
// graph of parents and children. Models (visual instances of Meshes), Cameras, and "empty" NodeBases all are kinds of Nodes.
type Scene struct {
	Name    string   // The name of the Scene. Set automatically to the scene name in your 3D modeler if the DAE file exports it.
	library *Library // The library from which this Scene was created. If the Scene was instantiated through code, this will be nil.
	// Root indicates the root node for the scene hierarchy. For visual Models to be displayed, they must be added to the
	// scene graph by simply adding them into the tree via parenting anywhere under the Root. For them to be removed from rendering,
	// they simply need to be removed from the tree.
	// See this page for more information on how a scene graph works: https://webglfundamentals.org/webgl/lessons/webgl-scene-graph.html
	Root     INode
	FogColor *Color  // The Color of any fog present in the Scene.
	FogMode  FogMode // The FogMode, indicating how the fog color is blended if it's on (not FogOff).
	// FogRange is the depth range at which the fog is active. FogRange consists of two numbers,
	// ranging from 0 to 1. The first indicates the start of the fog, and the second the end, in
	// terms of total depth of the near / far clipping plane. The default is [0, 1].
	FogRange   []float32
	LightingOn bool // If lighting is enabled when rendering the scene.
}

// NewScene creates a new Scene by the name given.
func NewScene(name string) *Scene {

	scene := &Scene{
		Name:       name,
		Root:       NewNode("Root"),
		FogColor:   NewColor(0, 0, 0, 0),
		FogRange:   []float32{0, 1},
		LightingOn: true,
	}

	scene.Root.(*Node).scene = scene

	return scene
}

// Clone clones the Scene, returning a copy. Models and Meshes are shared between them.
func (scene *Scene) Clone() *Scene {

	newScene := NewScene(scene.Name)
	newScene.library = scene.library
	newScene.LightingOn = scene.LightingOn

	// newScene.Models = append(newScene.Models, scene.Models...)
	newScene.Root = scene.Root.Clone()
	newScene.Root.(*Node).scene = newScene

	newScene.FogColor = scene.FogColor.Clone()
	newScene.FogMode = scene.FogMode
	newScene.FogRange[0] = scene.FogRange[0]
	newScene.FogRange[1] = scene.FogRange[1]
	return newScene

}

func (scene *Scene) fogAsFloatSlice() []float32 {

	fog := []float32{
		float32(scene.FogColor.R),
		float32(scene.FogColor.G),
		float32(scene.FogColor.B),
		float32(scene.FogMode),
	}

	if scene.FogMode == FogMultiply {
		fog[0] = 1 - fog[0]
		fog[1] = 1 - fog[1]
		fog[2] = 1 - fog[2]
	}

	return fog
}

// Library returns the Library from which this Scene was loaded. If it was created through code and not associated with a Library, this function will return nil.
func (scene *Scene) Library() *Library {
	return scene.library
}
