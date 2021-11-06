package jank3d

import (
	"sort"

	"github.com/kvartborg/vector"
)

// Model represents a singular visual instantiation of a Mesh. A Mesh contains the vertex information (what to draw); a Model references the Mesh to draw it with a specific
// Position, Rotation, and/or Scale (where and how to draw).
type Model struct {
	Mesh            *Mesh
	FrustumCulling  bool // Whether the model is culled when it leaves the frustum.
	BackfaceCulling bool // Whether the model's backfaces are culled.
	Position        vector.Vector
	Scale           vector.Vector
	Rotation        Quaternion
	closestTris     []*Triangle
	Visible         bool
	Color           Color
}

// NewModel creates a new Model (or instance) of the Mesh provided.
func NewModel(mesh *Mesh) *Model {

	return &Model{
		Mesh:            mesh,
		Position:        UnitVector(0),
		Rotation:        NewQuaternion(vector.Vector{0, 1, 0}, 0),
		Scale:           UnitVector(1),
		Visible:         true,
		FrustumCulling:  true,
		BackfaceCulling: true,
		Color:           NewColor(1, 1, 1, 1),
	}

}

func (model *Model) Transform() Matrix4 {

	// T * R * S * O

	transform := NewMatrix4()
	transform = transform.Mult(Scale(model.Scale[0], model.Scale[1], model.Scale[2]))
	transform = transform.Mult(Rotate(model.Rotation.Axis[0], model.Rotation.Axis[1], model.Rotation.Axis[2], model.Rotation.Angle))
	transform = transform.Mult(Translate(model.Position[0], model.Position[1], model.Position[2]))
	return transform

}

func (model *Model) triangleInList(tri *Triangle, triList []*Triangle) bool {

	for _, t := range triList {
		if tri == t {
			return true
		}
	}
	return false

}

// TransformedVertices returns the vertices of the Model, as transformed by the Camera's view matrix, sorted by distance to the Camera's position.
func (model *Model) TransformedVertices(viewMatrix Matrix4, cameraPosition vector.Vector) []*Triangle {

	mvp := model.Transform().Mult(viewMatrix)

	for _, vert := range model.Mesh.sortedVertices {
		vert.transformed = mvp.MultVecW(vert.Position)
	}

	model.closestTris = model.closestTris[:0]

	sort.SliceStable(model.Mesh.sortedVertices, func(i, j int) bool {
		return fastSub(cameraPosition, model.Mesh.sortedVertices[i].transformed).Magnitude() > fastSub(cameraPosition, model.Mesh.sortedVertices[j].transformed).Magnitude()
	})

	for _, v := range model.Mesh.sortedVertices {
		if !model.triangleInList(v.triangle, model.closestTris) {
			model.closestTris = append(model.closestTris, v.triangle)
		}
	}

	return model.closestTris

}
