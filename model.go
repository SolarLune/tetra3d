package tetra3d

import (
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

// Model represents a singular visual instantiation of a Mesh. A Mesh contains the vertex information (what to draw); a Model references the Mesh to draw it with a specific
// Position, Rotation, and/or Scale (where and how to draw).
type Model struct {
	*Node
	Mesh              *Mesh
	FrustumCulling    bool   // Whether the Model is culled when it leaves the frustum.
	Color             *Color // The overall color of the Model.
	ColorBlendingFunc func(model *Model) ebiten.ColorM
	BoundingSphere    *BoundingSphere
	BoundingAABB      *BoundingAABB

	Skinned    bool  // If the model is skinned and this is enabled, the model will tranform its vertices to match the skinning armature (Model.SkinRoot).
	SkinRoot   INode // The root node of the armature skinning this Model.
	skinMatrix Matrix4
	bones      [][]*Node
	vectorPool *VectorPool
}

var defaultColorBlendingFunc = func(model *Model) ebiten.ColorM {
	colorM := ebiten.ColorM{}
	colorM.Scale(model.Color.RGBA64())
	return colorM
}

// NewModel creates a new Model (or instance) of the Mesh and Name provided. A Model represents a singular visual instantiation of a Mesh.
func NewModel(mesh *Mesh, name string) *Model {

	model := &Model{
		Node:              NewNode(name),
		Mesh:              mesh,
		FrustumCulling:    true,
		Color:             NewColor(1, 1, 1, 1),
		ColorBlendingFunc: defaultColorBlendingFunc,
		skinMatrix:        NewMatrix4(),
	}

	if mesh != nil {
		model.vectorPool = NewVectorPool(len(mesh.Vertices))
	}

	model.onTransformUpdate = func() {
		model.BoundingSphere.SetLocalPosition(model.WorldPosition().Add(model.Mesh.Dimensions.Center()))
		model.BoundingSphere.Radius = model.Mesh.Dimensions.Max() / 2
	}

	radius := 0.0
	if mesh != nil {
		radius = mesh.Dimensions.Max() / 2
	}
	model.BoundingSphere = NewBoundingSphere("bounding sphere", radius)

	return model

}

// Clone creates a clone of the Model.
func (model *Model) Clone() INode {
	newModel := NewModel(model.Mesh, model.name)
	newModel.BoundingSphere = model.BoundingSphere.Clone().(*BoundingSphere)
	newModel.FrustumCulling = model.FrustumCulling
	newModel.visible = model.visible
	newModel.Color = model.Color.Clone()

	newModel.Skinned = model.Skinned
	newModel.SkinRoot = model.SkinRoot
	for i := range model.bones {
		newModel.bones = append(newModel.bones, append([]*Node{}, model.bones[i]...))
	}

	newModel.Node = model.Node.Clone().(*Node)
	for _, child := range newModel.children {
		child.setParent(newModel)
	}

	newModel.onTransformUpdate = func() {
		newModel.BoundingSphere.SetLocalPosition(newModel.WorldPosition().Add(model.Mesh.Dimensions.Center()))
		newModel.BoundingSphere.Radius = model.Mesh.Dimensions.Max() / 2
	}

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

	model.BoundingSphere.SetLocalPosition(model.Mesh.Dimensions.Center())
	model.BoundingSphere.Radius = model.Mesh.Dimensions.Max() / 2

	model.vectorPool = NewVectorPool(len(model.Mesh.Vertices))

}

// ReassignBones reassigns the model to point to a different armature. armatureNode should be a pointer to the starting object Node of the
// armature (not any of its bones).
func (model *Model) ReassignBones(armatureRoot INode) {

	if len(model.bones) == 0 {
		return
	}

	if armatureRoot.IsBone() {
		panic(`Error: Cannot reassign skinned Model [` + model.Path() + `] to armature bone [` + armatureRoot.Path() + `]. ReassignBones() should be called with the desired armature's root node.`)
	}

	bones := armatureRoot.ChildrenRecursive(false)

	boneMap := map[string]*Node{}

	for _, b := range bones {
		if b.IsBone() {
			boneMap[b.Name()] = b.(*Node)
		}
	}

	model.SkinRoot = armatureRoot

	for vertexIndex := range model.bones {

		for i := range model.bones[vertexIndex] {
			model.bones[vertexIndex][i] = boneMap[model.bones[vertexIndex][i].name]
		}

	}

}

func (model *Model) skinVertex(vertex *Vertex) vector.Vector {

	// Avoid reallocating a new matrix for every vertex; that's wasteful
	model.skinMatrix.Clear()

	for boneIndex, bone := range model.bones[vertex.ID] {

		weightPerc := float64(vertex.Weights[boneIndex])

		// The correct way to do this according to GLTF is to multiply the bone by the inverse of the model matrix, but
		// we don't have to because we simply don't multiply the vertex by the model matrix in the first place if it's skinned.
		influence := fastMatrixMult(bone.inverseBindMatrix, bone.Transform())

		if weightPerc == 1 {
			model.skinMatrix = influence
		} else {
			model.skinMatrix = model.skinMatrix.Add(influence.ScaleByScalar(weightPerc))
		}

	}

	vertOut := model.skinMatrix.MultVecW(vertex.Position)

	return vertOut

}

func (model *Model) TransformedVertices(vpMatrix Matrix4, camera *Camera) []*Triangle {

	model.vectorPool.Reset()

	if model.Skinned {

		t := time.Now()

		// If we're skinning a model, it will automatically copy the armature's position, scale, and rotation by copying its bones
		for _, tri := range model.Mesh.Triangles {
			tri.closestDepth = 0
			for _, vert := range tri.Vertices {
				vert.transformed = model.vectorPool.MultVecW(vpMatrix, model.skinVertex(vert))
				tri.closestDepth += vert.transformed[2]
			}
			tri.closestDepth /= 3
		}

		camera.DebugInfo.animationTime += time.Since(t)

	} else {

		mvp := model.Transform().Mult(vpMatrix)

		for _, tri := range model.Mesh.Triangles {
			tri.closestDepth = 0
			for _, vert := range tri.Vertices {
				vert.transformed = model.vectorPool.MultVecW(mvp, vert.Position)
				tri.closestDepth += vert.transformed[2]
			}
			tri.closestDepth /= 3
		}

	}

	sortMode := TriangleSortBackToFront

	if model.Mesh.Material != nil {
		sortMode = model.Mesh.Material.TriangleSortMode
	}

	if sortMode == TriangleSortBackToFront {
		sort.SliceStable(model.Mesh.sortedTriangles, func(i, j int) bool {
			return model.Mesh.sortedTriangles[i].closestDepth > model.Mesh.sortedTriangles[j].closestDepth
		})
	} else if sortMode == TriangleSortFrontToBack {
		sort.SliceStable(model.Mesh.sortedTriangles, func(i, j int) bool {
			return model.Mesh.sortedTriangles[i].closestDepth < model.Mesh.sortedTriangles[j].closestDepth
		})
	}

	return model.Mesh.sortedTriangles

}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (model *Model) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	model.addChildren(model, children...)
}

// Unparent unparents the Model from its parent, removing it from the scenegraph.
func (model *Model) Unparent() {
	if model.parent != nil {
		model.parent.RemoveChildren(model)
	}
}
