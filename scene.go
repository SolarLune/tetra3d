package tetra3d

import "github.com/kvartborg/vector"

const (
	FogOff = iota
	FogAdd
	FogMultiply
)

type FogMode int

type Scene struct {
	Name   string
	Models []*Model
	Meshes map[string]*Mesh
	Fog    vector.Vector
}

func NewScene(name string) *Scene {
	scene := &Scene{
		Name:   name,
		Models: []*Model{},
		Meshes: map[string]*Mesh{},
		Fog:    nil,
	}
	scene.SetFog(0, 0, 0, FogOff)
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

func (scene *Scene) SetFog(r, g, b float64, fogMode FogMode) {
	a := float64(fogMode)
	scene.Fog = vector.Vector{
		r,
		g,
		b,
		a, // There's no alpha for fog, since it depends on depth
	}
}

func (scene *Scene) fogAsFloatSlice() []float32 {

	fog := []float32{
		float32(scene.Fog[0]),
		float32(scene.Fog[1]),
		float32(scene.Fog[2]),
		float32(scene.Fog[3]),
	}

	if scene.Fog[3] == FogMultiply {
		fog[0] = 1 - fog[0]
		fog[1] = 1 - fog[1]
		fog[2] = 1 - fog[2]
	}

	return fog
}
