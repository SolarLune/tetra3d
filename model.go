package jank3d

import (
	"sort"

	"github.com/kvartborg/vector"
	"github.com/takeyourhatoff/bitset"
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
	BoundingSphere  *BoundingSphere
}

// NewModel creates a new Model (or instance) of the Mesh provided.
func NewModel(mesh *Mesh) *Model {

	model := &Model{
		Mesh:            mesh,
		Position:        UnitVector(0),
		Rotation:        NewQuaternion(vector.Vector{0, 1, 0}, 0),
		Scale:           UnitVector(1),
		Visible:         true,
		FrustumCulling:  true,
		BackfaceCulling: true,
		Color:           NewColor(1, 1, 1, 1),
	}

	model.BoundingSphere = NewBoundingSphere(model, UnitVector(0), model.Mesh.Dimensions.Max())

	return model

}

func (model *Model) Transform() Matrix4 {

	// T * R * S * O

	transform := NewMatrix4()
	transform = transform.Mult(Scale(model.Scale[0], model.Scale[1], model.Scale[2]))
	transform = transform.Mult(Rotate(model.Rotation.Axis[0], model.Rotation.Axis[1], model.Rotation.Axis[2], model.Rotation.Angle))
	transform = transform.Mult(Translate(model.Position[0], model.Position[1], model.Position[2]))
	return transform

}

// TransformedVertices returns the vertices of the Model, as transformed by the Camera's view matrix, sorted by distance to the Camera's position.
func (model *Model) TransformedVertices(vpMatrix Matrix4, viewPos vector.Vector) []*Triangle {

	mvp := model.Transform().Mult(vpMatrix)

	for _, vert := range model.Mesh.sortedVertices {
		vert.transformed = mvp.MultVecW(vert.Position)
	}

	model.closestTris = model.closestTris[:0]

	sort.SliceStable(model.Mesh.sortedVertices, func(i, j int) bool {
		return fastSub(model.Mesh.sortedVertices[i].transformed, viewPos).Magnitude() > fastSub(model.Mesh.sortedVertices[j].transformed, viewPos).Magnitude()
	})

	// By using a bitset.Set, we can avoid putting triangles in the closestTris slice multiple times and avoid the time cost by looping over the slice / checking
	// a map to see if the triangle's been added.
	set := bitset.Set{}

	for _, v := range model.Mesh.sortedVertices {

		if !set.Test(v.triangle.ID) {
			model.closestTris = append(model.closestTris, v.triangle)
			set.Add(v.triangle.ID)
		}

	}

	return model.closestTris

}
