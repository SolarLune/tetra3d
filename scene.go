package tetra3d

const (
	FogOff = iota
	FogAdd
	FogMultiply
	FogOverwrite
)

type FogMode int

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

func (scene *Scene) FindModel(modelName string) *Model {
	for _, model := range scene.Models {
		if model.Name == modelName {
			return model
		}
	}
	return nil
}

func (scene *Scene) AddModels(models ...*Model) {
	scene.Models = append(scene.Models, models...)
}

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
