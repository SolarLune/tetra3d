package tetra3d

import (
	"sort"

	"github.com/kvartborg/vector"
	"github.com/takeyourhatoff/bitset"
)

// Model represents a singular visual instantiation of a Mesh. A Mesh contains the vertex information (what to draw); a Model references the Mesh to draw it with a specific
// Position, Rotation, and/or Scale (where and how to draw).
type Model struct {
	*Node
	Mesh                *Mesh
	FrustumCulling      bool // Whether the Model is culled when it leaves the frustum.
	BackfaceCulling     bool // Whether the Model's backfaces are culled.
	closestTris         []*Triangle
	Color               Color // The overall color of the Model.
	BoundingSphere      *BoundingSphere
	SortTrisBackToFront bool // Whether the Model's triangles are sorted back-to-front or not.
}

// NewModel creates a new Model (or instance) of the Mesh and Name provided. A Model represents a singular visual instantiation of a Mesh.
func NewModel(mesh *Mesh, name string) *Model {

	model := &Model{
		Node:                NewNode(name),
		Mesh:                mesh,
		FrustumCulling:      true,
		BackfaceCulling:     true,
		Color:               NewColor(1, 1, 1, 1),
		SortTrisBackToFront: true,
	}

	model.Node.Owner = model

	dimensions := 0.0
	if mesh != nil {
		dimensions = mesh.Dimensions.Max()
	}
	model.BoundingSphere = NewBoundingSphere(model, vector.Vector{0, 0, 0}, dimensions)

	return model

}

func (model *Model) Clone() *Model {
	newModel := NewModel(model.Mesh, model.Name)
	newModel.BoundingSphere = model.BoundingSphere.Clone()
	newModel.BoundingSphere.Model = newModel
	newModel.FrustumCulling = model.FrustumCulling
	newModel.BackfaceCulling = model.BackfaceCulling
	newModel.Visible = model.Visible
	newModel.Color = model.Color.Clone()
	newModel.SortTrisBackToFront = model.SortTrisBackToFront
	return newModel
}

// TransformedVertices returns the vertices of the Model, as transformed by the Camera's view matrix, sorted by distance to the Camera's position.
func (model *Model) TransformedVertices(vpMatrix Matrix4, viewPos vector.Vector) []*Triangle {

	mvp := model.Transform().Mult(vpMatrix)

	for _, vert := range model.Mesh.sortedVertices {
		vert.transformed = mvp.MultVecW(vert.Position)
	}

	model.closestTris = model.closestTris[:0]

	if model.SortTrisBackToFront {
		sort.SliceStable(model.Mesh.sortedVertices, func(i, j int) bool {
			return fastSub(model.Mesh.sortedVertices[i].transformed, viewPos).Magnitude() > fastSub(model.Mesh.sortedVertices[j].transformed, viewPos).Magnitude()
		})
	} else {
		sort.SliceStable(model.Mesh.sortedVertices, func(i, j int) bool {
			return fastSub(model.Mesh.sortedVertices[i].transformed, viewPos).Magnitude() < fastSub(model.Mesh.sortedVertices[j].transformed, viewPos).Magnitude()
		})
	}

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

func (model *Model) getNode() *Node {
	return model.Node
}
