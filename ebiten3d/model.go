package ebiten3d

import (
	"sort"

	"github.com/kvartborg/vector"
)

type Model struct {
	Mesh        *Mesh
	Transform   Matrix4
	Scale       vector.Vector
	Position    vector.Vector
	Rotation    vector.Vector // Rotation is effectively a Quaternion (a 4D vector; the first three arguments control the axis, and the fourth controls the angle).
	closestTris []*Triangle
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

func (model *Model) TransformedVertices(mvpMatrix Matrix4, camera *Camera) []vector.Vector {

	verts := []vector.Vector{}

	model.closestTris = append([]*Triangle{}, model.Mesh.Triangles...)

	sort.SliceStable(model.closestTris, func(i, j int) bool {
		a := mvpMatrix.MultVecW(model.closestTris[i].Center())
		b := mvpMatrix.MultVecW(model.closestTris[j].Center())
		return camera.Position.Sub(a).Magnitude() > camera.Position.Sub(b).Magnitude()
	})

	for _, tri := range model.closestTris {
		for _, vert := range tri.Vertices {
			verts = append(verts, mvpMatrix.MultVecW(vert.Position))
		}
	}

	return verts

}
