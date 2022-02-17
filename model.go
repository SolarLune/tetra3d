package tetra3d

import (
	"math"
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
	ColorBlendingFunc func(model *Model, meshPart *MeshPart) ebiten.ColorM
	BoundingSphere    *BoundingSphere
	BoundingAABB      *BoundingAABB

	Skinned    bool  // If the model is skinned and this is enabled, the model will tranform its vertices to match the skinning armature (Model.SkinRoot).
	SkinRoot   INode // The root node of the armature skinning this Model.
	skinMatrix Matrix4
	bones      [][]*Node
	vectorPool *VectorPool
}

var defaultColorBlendingFunc = func(model *Model, meshPart *MeshPart) ebiten.ColorM {
	colorM := ebiten.ColorM{}
	colorM.Scale(model.Color.ToFloat64s())

	if meshPart.Material != nil {
		colorM.Scale(meshPart.Material.Color.ToFloat64s())
	}

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
		model.vectorPool = NewVectorPool(mesh.TotalVertexCount())
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

	return newModel

}

func (model *Model) Transform() Matrix4 {

	model.BoundingSphere.SetLocalPosition(model.WorldPosition().Add(model.Mesh.Dimensions.Center()))
	model.BoundingSphere.Radius = model.Mesh.Dimensions.Max() / 2

	return model.Node.Transform()

}

// Merge merges the provided models into the calling Model. You can use this to merge several objects initially dynamically placed into the calling Model's mesh,
// thereby saving on draw calls. Note that models are merged into MeshParts (saving draw calls) based on maximum vertex count and shared materials (so to get any
// benefit from merging, ensure the merged models share materials; if they all have unique materials, they will be turned into individual MeshParts, thereby forcing multiple
// draw calls).
func (model *Model) Merge(models ...*Model) {

	for _, other := range models {

		p, s, r := model.Transform().Decompose()
		op, os, or := other.Transform().Decompose()

		inverted := NewMatrix4Scale(os[0], os[1], os[2])
		scaleMatrix := NewMatrix4Scale(s[0], s[1], s[2])
		inverted = inverted.Mult(scaleMatrix)

		inverted = inverted.Mult(r.Transposed().Mult(or))

		inverted = inverted.Mult(NewMatrix4Translate(op[0]-p[0], op[1]-p[1], op[2]-p[2]))

		for _, otherPart := range other.Mesh.MeshParts {

			// Here, we'll merge models into the calling Model, using its existing mesh parts if the materials match and if adding the vertices wouldn't exceed the maximum triangle count (21845 in a single draw call).

			var targetPart *MeshPart

			for _, mp := range model.Mesh.MeshParts {
				if mp.Material == otherPart.Material && len(mp.Triangles)+len(otherPart.Triangles) < ebiten.MaxIndicesNum/3 {
					targetPart = mp
					break
				}
			}

			if targetPart == nil {
				targetPart = model.Mesh.AddMeshPart(otherPart.Material)
			}

			verts := []*Vertex{}

			for _, tri := range otherPart.Triangles {

				v0 := tri.Vertices[0].Clone()
				v0.Position = inverted.MultVec(v0.Position)

				v1 := tri.Vertices[1].Clone()
				v1.Position = inverted.MultVec(v1.Position)

				v2 := tri.Vertices[2].Clone()
				v2.Position = inverted.MultVec(v2.Position)

				verts = append(verts, v0, v1, v2)

			}

			targetPart.AddTriangles(verts...)

		}

	}

	model.Mesh.UpdateBounds()

	model.BoundingSphere.SetLocalPosition(model.Mesh.Dimensions.Center())
	model.BoundingSphere.Radius = model.Mesh.Dimensions.Max() / 2

	model.vectorPool = NewVectorPool(model.Mesh.TotalVertexCount())

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

	bones := armatureRoot.ChildrenRecursive()

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

func (model *Model) TransformedVertices(vpMatrix Matrix4, camera *Camera, meshPart *MeshPart) []*Triangle {

	model.vectorPool.Reset()

	var transformFunc func(vector.Vector) vector.Vector

	if meshPart.Material != nil && meshPart.Material.VertexProgram != nil {
		transformFunc = meshPart.Material.VertexProgram
	}

	if model.Skinned {

		t := time.Now()

		// If we're skinning a model, it will automatically copy the armature's position, scale, and rotation by copying its bones
		for _, tri := range meshPart.Triangles {

			tri.visible = true

			tri.depth = math.MaxFloat64

			for _, vert := range tri.Vertices {

				v := model.skinVertex(vert)
				if transformFunc != nil {
					v = transformFunc(v)
					if v == nil {
						tri.visible = false
						break
					}
				}
				vert.transformed = model.vectorPool.MultVecW(vpMatrix, v)
				if vert.transformed[3] < tri.depth {
					tri.depth = vert.transformed[3]
				}
			}

		}

		camera.DebugInfo.animationTime += time.Since(t)

	} else {

		mvp := model.Transform().Mult(vpMatrix)

		for _, tri := range meshPart.Triangles {

			tri.visible = true

			// tri.depth = 0
			tri.depth = math.MaxFloat64

			for _, vert := range tri.Vertices {
				v := vert.Position
				if transformFunc != nil {
					v = transformFunc(v)
					if v == nil {
						tri.visible = false
						break
					}
				}
				vert.transformed = model.vectorPool.MultVecW(mvp, v)

				if vert.transformed[3] < tri.depth {
					tri.depth = vert.transformed[3]
				}

			}

		}

	}

	sortMode := TriangleSortBackToFront

	if meshPart.Material != nil {
		sortMode = meshPart.Material.TriangleSortMode
	}

	// Preliminary tests indicate sort.SliceStable is faster than sort.Slice for our purposes

	if sortMode == TriangleSortBackToFront {
		sort.SliceStable(meshPart.sortedTriangles, func(i, j int) bool {
			return meshPart.sortedTriangles[i].depth > meshPart.sortedTriangles[j].depth
		})
	} else if sortMode == TriangleSortFrontToBack {
		sort.SliceStable(meshPart.sortedTriangles, func(i, j int) bool {
			return meshPart.sortedTriangles[i].depth < meshPart.sortedTriangles[j].depth
		})
	}

	return meshPart.sortedTriangles

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

// Type returns the NodeType for this object.
func (model *Model) Type() NodeType {
	return NodeTypeModel
}

// isTransparent returns true if the provided MeshPart has a Material with TransparencyModeTransparent, or if it's
// TransparencyModeAuto with the model or material alpha color being under 0.99. This is a helper function for sorting
// MeshParts into either transparent or opaque buckets for rendering.
func (model *Model) isTransparent(meshPart *MeshPart) bool {
	mat := meshPart.Material
	return mat != nil && (mat.TransparencyMode == TransparencyModeTransparent || (mat.TransparencyMode == TransparencyModeAuto && (mat.Color.A < 0.99 || model.Color.A < 0.99)))
}
