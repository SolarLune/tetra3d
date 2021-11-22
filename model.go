package tetra3d

import (
	"sort"

	"github.com/kvartborg/vector"
	"github.com/takeyourhatoff/bitset"
)

// Model represents a singular visual instantiation of a Mesh. A Mesh contains the vertex information (what to draw); a Model references the Mesh to draw it with a specific
// Position, Rotation, and/or Scale (where and how to draw).
type Model struct {
	*NodeBase
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
		NodeBase:            NewNodeBase(name),
		Mesh:                mesh,
		FrustumCulling:      true,
		BackfaceCulling:     true,
		Color:               NewColor(1, 1, 1, 1),
		SortTrisBackToFront: true,
	}

	dimensions := 0.0
	if mesh != nil {
		dimensions = mesh.Dimensions.Max()
	}
	model.BoundingSphere = NewBoundingSphere(model, vector.Vector{0, 0, 0}, dimensions)

	return model

}

// Clone creates a clone of the Model.
func (model *Model) Clone() Node {
	newModel := NewModel(model.Mesh, model.name)
	newModel.BoundingSphere = model.BoundingSphere.Clone()
	newModel.BoundingSphere.Node = newModel
	newModel.FrustumCulling = model.FrustumCulling
	newModel.BackfaceCulling = model.BackfaceCulling
	newModel.visible = model.visible
	newModel.Color = model.Color.Clone()
	newModel.SortTrisBackToFront = model.SortTrisBackToFront
	return newModel
}

// Merge merges the provided models into the calling Model. You can use this to merge several objects initially dynamically placed into the calling Model's mesh,
// thereby saving on draw calls. Note that the finished Mesh still only uses one Image for texturing, so this is best done between objects that share textures or
// tilemaps.
func (model *Model) Merge(models ...*Model) {

	for _, other := range models {

		p, s, r := model.Transform().Decompose()
		op, os, or := other.Transform().Decompose()

		inverted := NewMatrix4Scale(os[0], os[1], os[2])
		scaleMatrix := NewMatrix4Scale(s[0], s[1], s[2])
		inverted = inverted.Mult(scaleMatrix)

		inverted = inverted.Mult(r.Transposed().Mult(or))

		inverted = inverted.Mult(NewMatrix4Translate(op[0]-p[0], op[1]-p[1], op[2]-p[2]))

		verts := []*Vertex{}

		for i := 0; i < len(other.Mesh.Vertices); i += 3 {

			v0 := other.Mesh.Vertices[i].Clone()
			v0.Position = inverted.MultVec(v0.Position)

			v1 := other.Mesh.Vertices[i+1].Clone()
			v1.Position = inverted.MultVec(v1.Position)

			v2 := other.Mesh.Vertices[i+2].Clone()
			v2.Position = inverted.MultVec(v2.Position)

			verts = append(verts, v0, v1, v2)

		}

		model.Mesh.AddTriangles(verts...)

	}

	model.Mesh.UpdateBounds()
	model.BoundingSphere.LocalPosition = model.Mesh.Dimensions.Center()
	model.BoundingSphere.LocalRadius = model.Mesh.Dimensions.Max()

}

// TransformedVertices returns the vertices of the Model, as transformed by the (provided) Camera's view matrix, sorted by distance to the (provided) Camera's position.
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

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (model *Model) AddChildren(children ...Node) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	model.addChildren(model, children...)
}
