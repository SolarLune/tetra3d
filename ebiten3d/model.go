package ebiten3d

import "github.com/kvartborg/vector"

type Model struct {
	Mesh      *Mesh
	Transform Matrix4
	Scale     vector.Vector
	Position  vector.Vector
	Rotation  vector.Vector
}

func NewModel(mesh *Mesh) *Model {

	return &Model{
		Mesh:      mesh,
		Transform: NewMatrix4(),
		Position:  UnitVector(0),
		Scale:     UnitVector(1),
		Rotation:  vector.Vector{0, 1, 0, 0},
	}

}

func (model *Model) UpdateTransform() {

	// T * R * S * O

	model.Transform = NewMatrix4()
	model.Transform = model.Transform.Mult(Scale(model.Scale[0], model.Scale[1], model.Scale[2]))
	model.Transform = model.Transform.Mult(Rotate(model.Rotation[:3], model.Rotation[3]))
	model.Transform = model.Transform.Mult(Translate(model.Position[0], model.Position[1], model.Position[2]))

}

func (model *Model) TransformedVertices(mvpMatrix Matrix4) []vector.Vector {

	verts := []vector.Vector{}

	for _, vert := range model.Mesh.Vertices {
		// verts = append(verts, mvpMatrix.MultVec(vert))
		verts = append(verts, mvpMatrix.MultVecW(vert))
	}

	return verts

}
