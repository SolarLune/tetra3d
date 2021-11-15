package tetra3d

const (
	FogOff       = iota // No fog
	FogAdd              // Additive blended fog
	FogMultiply         // Multiplicative blended fog
	FogOverwrite        // Color overwriting fog (mixing base with fog color over depth distance)
)

type FogMode int

// Scene represents a world of sorts, and can contain a variety of Models and Meshes.
type Scene struct {
	Name     string
	Models   []*Model
	Meshes   map[string]*Mesh
	FogColor Color   // The Color of any fog present in the Scene.
	FogMode  FogMode // The FogMode, indicating how the fog color is blended if it's on (not FogOff).
	// FogRange is the depth range at which the fog is active. FogRange consists of two numbers,
	// the first indicating the start of the fog, and the second the end, in terms of total depth
	// of the near / far clipping plane.
	FogRange []float32
}

// NewScene creates a new Scene by the name given.
func NewScene(name string) *Scene {
	scene := &Scene{
		Name:     name,
		Models:   []*Model{},
		Meshes:   map[string]*Mesh{},
		FogColor: NewColor(0, 0, 0, 0),
		FogRange: []float32{0, 1},
	}

	return scene
}

// Clone clones the Scene, returning a copy. Models and Meshes are shared between them.
func (scene *Scene) Clone() *Scene {

	newScene := NewScene(scene.Name)

	for name, mesh := range scene.Meshes {
		newScene.Meshes[name] = mesh
	}

	newScene.Models = append(newScene.Models, scene.Models...)

	newScene.FogColor = scene.FogColor.Clone()
	newScene.FogMode = scene.FogMode
	newScene.FogRange[0] = scene.FogRange[0]
	newScene.FogRange[1] = scene.FogRange[1]
	return newScene

}

// AddModels adds the specified models to the Scene's models list.
func (scene *Scene) AddModels(models ...*Model) {
	scene.Models = append(scene.Models, models...)
}

// RemoveModels removes the specified models from the Scene's models list.
func (scene *Scene) RemoveModels(models ...*Model) {

	for _, model := range models {

		for i := 0; i < len(scene.Models); i++ {
			if scene.Models[i] == model {
				scene.Models[i] = nil
				scene.Models = append(scene.Models[:i], scene.Models[i+1:]...)
				break
			}
		}
	}

}

// FindModel allows you to search for a Model by the provided name. If a Model
// with the name provided isn't found,
func (scene *Scene) FindModel(modelName string) *Model {
	for _, m := range scene.Models {
		if m.Name == modelName {
			return m
		}
	}
	return nil
}

// FilterModels filters out the Scene's Models list to return just the Models
// that satisfy the function passed. You can use this to, for example, find
// Models that have a specific name, or render a Scene in stages.
func (scene *Scene) FilterModels(filterFunc func(model *Model) bool) []*Model {
	newMS := []*Model{}
	for _, m := range scene.Models {
		if filterFunc(m) {
			newMS = append(newMS, m)
		}
	}
	return newMS
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
