package tetra3d

import (
	"errors"
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
	FrustumCulling    bool                                                 // Whether the Model is culled when it leaves the frustum.
	Color             *Color                                               // The overall multiplicative color of the Model.
	ColorBlendingFunc func(model *Model, meshPart *MeshPart) ebiten.ColorM // The blending function used to color the Model; by default, it basically modulates the model by the color.
	BoundingSphere    *BoundingSphere

	DynamicBatchModels []*Model // Models that are dynamically merged into this one.
	DynamicBatchOwner  *Model

	Skinned        bool  // If the model is skinned and this is enabled, the model will tranform its vertices to match the skinning armature (Model.SkinRoot).
	SkinRoot       INode // The root node of the armature skinning this Model.
	skinMatrix     Matrix4
	bones          [][]*Node // The bones (nodes) of the Model, assuming it has been skinned. A Mesh's bones slice will point to indices indicating bones in the Model.
	skinVectorPool *VectorPool
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
		Node:               NewNode(name),
		Mesh:               mesh,
		FrustumCulling:     true,
		Color:              NewColor(1, 1, 1, 1),
		ColorBlendingFunc:  defaultColorBlendingFunc,
		skinMatrix:         NewMatrix4(),
		DynamicBatchModels: []*Model{},
	}

	if mesh != nil {
		model.skinVectorPool = NewVectorPool(mesh.VertexCount * 2) // Both position and normal
	}

	radius := 0.0
	if mesh != nil {
		radius = mesh.Dimensions.MaxSpan() / 2
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
	newModel.DynamicBatchModels = append(newModel.DynamicBatchModels, model.DynamicBatchModels...)
	newModel.DynamicBatchOwner = model.DynamicBatchOwner

	newModel.Skinned = model.Skinned
	newModel.SkinRoot = model.SkinRoot
	for i := range model.bones {
		newModel.bones = append(newModel.bones, append([]*Node{}, model.bones[i]...))
	}

	newModel.Node = model.Node.Clone().(*Node)
	for _, child := range newModel.children {
		child.setParent(newModel)
	}

	newModel.originalLocalPosition = model.originalLocalPosition.Clone()

	return newModel

}

// Transform returns the global transform of the Model, taking into account any transforms its parents or grandparents have that
// would impact the Model.
func (model *Model) Transform() Matrix4 {

	if model.Mesh == nil {
		return NewEmptyMatrix4()
	}

	if model.isTransformDirty {

		wp := model.WorldPosition()

		// Skinned models have their positions at 0, 0, 0, and vertices offset according to wherever they were when exported.
		// To combat this, we save the original local positions of the mesh on export to position the bounding sphere in the
		// correct location.

		var center vector.Vector

		// We do this because if a model is skinned and we've parented the model to the armature, then the center is
		// now from origin relative to the base of the armature on scene export.
		if model.SkinRoot != nil && model.Skinned && model.parent == model.SkinRoot {
			parent := model.parent.(*Node)
			center = model.Mesh.Dimensions.Center().Sub(parent.originalLocalPosition)
		} else {
			center = model.Mesh.Dimensions.Center()
		}

		wp[0] += center[0]
		wp[1] += center[1]
		wp[2] += center[2]

		model.BoundingSphere.SetLocalPosition(wp)

		dim := model.Mesh.Dimensions.Clone()
		scale := model.WorldScale()
		dim[0][0] *= scale[0]
		dim[0][1] *= scale[1]
		dim[0][2] *= scale[2]

		dim[1][0] *= scale[0]
		dim[1][1] *= scale[1]
		dim[1][2] *= scale[2]

		model.BoundingSphere.Radius = dim.MaxSpan() / 2

	}

	return model.Node.Transform()

}

func (model *Model) modelAlreadyDynamicallyBatched(batchedModel *Model) bool {
	for _, m := range model.DynamicBatchModels {
		if m == batchedModel {
			return true
		}
	}
	return false
}

// DynamicBatchAdd adds the provided models to the calling Model's dynamic batch. Note that unlike StaticMerge(), DynamicBatchAdd works by simply
// rendering the batched models using the calling Model's first MeshPart's material. By dynamically batching models together, this allows us to
// not flush between rendering multiple Models, saving a lot of render time, particularly if rendering many low-poly, individual models that have
// very little variance (i.e. if they all share a single texture).
// For more information, see this Wiki page on batching / merging: https://github.com/SolarLune/Tetra3d/wiki/Merging-and-Batching-Draw-Calls
func (model *Model) DynamicBatchAdd(batchedModels ...*Model) error {

	for _, other := range batchedModels {

		if model == other || model.modelAlreadyDynamicallyBatched(other) {
			continue
		}

		triCount := model.DynamicBatchTriangleCount()

		if triCount+len(other.Mesh.Triangles) > maxTriangleCount {
			return errors.New("too many triangles in dynamic merge")
		}

		// for _, otherPart := range other.Mesh.MeshParts {

		// 	var targetPart *MeshPart

		// 	for _, mp := range model.Mesh.MeshParts {
		// 		if mp.Material == otherPart.Material && mp.TriangleCount()+otherPart.TriangleCount() < maxTriangleCount {
		// 			targetPart = mp
		// 			break
		// 		}
		// 	}

		// 	if targetPart == nil {
		// 		targetPart = model.Mesh.AddMeshPart(otherPart.Material)
		// 	}

		// 	// Here, we'll batch meshparts together, using its existing mesh parts if the materials match
		// 	// and if adding in the triangles wouldn't exceed the maximum triangle count (21845 in a single draw call).

		// }

		model.DynamicBatchModels = append(model.DynamicBatchModels, other)
		other.DynamicBatchOwner = model

	}

	return nil

}

// DynamicBatchRemove removes the specified batched Models from the calling Model's dynamic batch slice.
func (model *Model) DynamicBatchRemove(batched ...*Model) {
	for _, m := range batched {
		for i, existing := range model.DynamicBatchModels {
			if existing == m {
				model.DynamicBatchModels[i] = nil
				model.DynamicBatchModels = append(model.DynamicBatchModels[:i], model.DynamicBatchModels[i+1:]...)
				m.DynamicBatchOwner = nil
				break
			}
		}
	}
}

// DynamicBatchTriangleCount returns the total number of triangles of Models in the calling Model's dynamic batch.
func (model *Model) DynamicBatchTriangleCount() int {
	count := 0
	for _, child := range model.DynamicBatchModels {
		count += len(child.Mesh.Triangles)
	}
	return count
}

// Merge statically merges the provided models into the calling Model's mesh, such that their vertex properties (position, normal, UV, etc) are part of the calling Model's Mesh.
// You can use this to merge several objects initially dynamically placed into the calling Model's mesh, thereby pulling back to a single draw call. Note that models are merged into MeshParts
// (saving draw calls) based on maximum vertex count and shared materials (so to get any benefit from merging, ensure the merged models share materials; if they all have unique
// materials, they will be turned into individual MeshParts, thereby forcing multiple draw calls). Also note that as the name suggests, this is static merging, which means that
// after merging, the new vertices are static - part of the merging Model.
// For more information, see this Wiki page on batching / merging: https://github.com/SolarLune/Tetra3d/wiki/Merging-and-Batching-Draw-Calls
func (model *Model) Merge(models ...*Model) {

	totalSize := 0
	for _, other := range models {
		if model == other {
			continue
		}
		totalSize += len(other.Mesh.VertexPositions)
	}

	if totalSize == 0 {
		return
	}

	if model.Mesh.triIndex*3+totalSize > model.Mesh.VertexMax {
		model.Mesh.allocateVertexBuffers(model.Mesh.VertexMax + totalSize)
	}

	for _, other := range models {

		if model == other {
			continue
		}

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
				if mp.Material == otherPart.Material && mp.TriangleCount()+otherPart.TriangleCount() < maxTriangleCount {
					targetPart = mp
					break
				}
			}

			if targetPart == nil {
				targetPart = model.Mesh.AddMeshPart(otherPart.Material)
			}

			verts := []VertexInfo{}

			for triIndex := otherPart.TriangleStart; triIndex < otherPart.TriangleEnd; triIndex++ {
				for i := 0; i < 3; i++ {
					vertInfo := otherPart.Mesh.GetVertexInfo(triIndex*3 + i)
					vec := vector.Vector{vertInfo.X, vertInfo.Y, vertInfo.Z}
					x, y, z := fastMatrixMultVec(inverted, vec)
					vertInfo.X = x
					vertInfo.Y = y
					vertInfo.Z = z
					verts = append(verts, vertInfo)
				}
			}

			targetPart.AddTriangles(verts...)

		}

	}

	model.Mesh.UpdateBounds()

	model.BoundingSphere.SetLocalPosition(model.Mesh.Dimensions.Center())
	model.BoundingSphere.Radius = model.Mesh.Dimensions.MaxSpan() / 2

	model.skinVectorPool = NewVectorPool(len(model.Mesh.VertexPositions))

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

func (model *Model) skinVertex(vertID int, transformNormal bool) (vector.Vector, vector.Vector) {

	// Avoid reallocating a new matrix for every vertex; that's wasteful
	model.skinMatrix.Clear()

	var normal vector.Vector

	for boneIndex, bone := range model.bones[vertID] {

		weightPerc := float64(model.Mesh.VertexWeights[vertID][boneIndex])

		if weightPerc == 0 {
			continue
		}

		// We don't actually have to calculate the bone influence; it's automatically
		// cached in the bone (Node) when the transform changes.
		bone.Transform()

		if weightPerc == 1 {
			model.skinMatrix = bone.boneInfluence
			break // I think we can end here if the weight percentage is 100%, right?
		} else {
			model.skinMatrix = model.skinMatrix.Add(bone.boneInfluence.ScaleByScalar(weightPerc))
		}

	}

	vertOut := model.skinVectorPool.MultVecW(model.skinMatrix, model.Mesh.VertexPositions[vertID])

	if transformNormal {
		model.skinMatrix[3][0] = 0
		model.skinMatrix[3][1] = 0
		model.skinMatrix[3][2] = 0
		model.skinMatrix[3][3] = 1

		normal = model.skinVectorPool.MultVecW(model.skinMatrix, model.Mesh.VertexNormals[vertID])
	}

	return vertOut, normal

}

// ProcessVertices processes the vertices a Model has in preparation for rendering, given a view-projection
// matrix, a camera, and the MeshPart being rendered.
func (model *Model) ProcessVertices(vpMatrix Matrix4, camera *Camera, meshPart *MeshPart, scene *Scene) {

	var transformFunc func(vertPos vector.Vector, index int) vector.Vector

	if meshPart.Material != nil && meshPart.Material.VertexTransformFunction != nil {
		transformFunc = meshPart.Material.VertexTransformFunction
	}

	if model.Skinned {

		lightingOn := false
		if scene != nil {
			lightingOn = scene.LightingOn && (meshPart.Material == nil || !meshPart.Material.Shadeless)
		}

		model.skinVectorPool.Reset()

		t := time.Now()

		// If we're skinning a model, it will automatically copy the armature's position, scale, and rotation by copying its bones
		for i := 0; i < len(meshPart.sortingTriangles); i++ {

			tri := meshPart.sortingTriangles[i]

			depth := math.MaxFloat32

			for v := 0; v < 3; v++ {

				vertPos, vertNormal := model.skinVertex(tri.ID*3+v, lightingOn)
				if transformFunc != nil {
					vertPos = transformFunc(vertPos, tri.ID*3+v)
				}
				if vertNormal != nil {
					model.Mesh.vertexSkinnedNormals[tri.ID*3+v] = vertNormal
					model.Mesh.vertexSkinnedPositions[tri.ID*3+v] = vertPos
				}
				transformed := model.Mesh.vertexTransforms[tri.ID*3+v]
				x, y, z, w := fastMatrixMultVecW(vpMatrix, vertPos)
				transformed[0] = x
				transformed[1] = y
				transformed[2] = z
				transformed[3] = w

				if w < depth {
					depth = w
				}
			}

			meshPart.sortingTriangles[i].depth = float32(depth)

		}

		camera.DebugInfo.animationTime += time.Since(t)

	} else {

		mat := meshPart.Material

		var base Matrix4
		if mat == nil || mat.BillboardMode == BillboardModeNone {
			base = model.Transform()
		} else if mat.BillboardMode == BillboardModeXZ {
			base = NewLookAtMatrix(camera.WorldPosition(), model.WorldPosition(), vector.Y)
			base = base.SetRow(1, vector.Vector{0, 1, 0, 0})
			base = base.Mult(model.Transform())
		} else if mat.BillboardMode == BillboardModeAll {
			base = NewLookAtMatrix(camera.WorldPosition(), model.WorldPosition(), vector.Y).Mult(model.Transform())
		}

		mvp := fastMatrixMult(base, vpMatrix)

		for i := 0; i < len(meshPart.sortingTriangles); i++ {

			tri := meshPart.sortingTriangles[i]
			depth := math.MaxFloat64

			for i := 0; i < 3; i++ {
				v0 := model.Mesh.VertexPositions[tri.ID*3+i]

				if transformFunc != nil {
					v0 = transformFunc(v0.Clone(), tri.ID*3+i)
				}

				t0 := model.Mesh.vertexTransforms[tri.ID*3+i]
				x, y, z, w := fastMatrixMultVecW(mvp, v0)
				t0[0] = x
				t0[1] = y
				t0[2] = z
				t0[3] = w

				if w < depth {
					depth = w
				}
			}

			meshPart.sortingTriangles[i].depth = float32(depth)

		}

	}

	sortMode := TriangleSortModeBackToFront

	if meshPart.Material != nil {
		sortMode = meshPart.Material.TriangleSortMode
	}

	// Preliminary tests indicate sort.SliceStable is faster than sort.Slice for our purposes

	if sortMode == TriangleSortModeBackToFront {
		sort.SliceStable(meshPart.sortingTriangles, func(i, j int) bool {
			return meshPart.sortingTriangles[i].depth > meshPart.sortingTriangles[j].depth
		})
	} else if sortMode == TriangleSortModeFrontToBack {
		sort.SliceStable(meshPart.sortingTriangles, func(i, j int) bool {
			return meshPart.sortingTriangles[i].depth < meshPart.sortingTriangles[j].depth
		})
	}

}

// isTransparent returns true if the provided MeshPart has a Material with TransparencyModeTransparent, or if it's
// TransparencyModeAuto with the model or material alpha color being under 0.99. This is a helper function for sorting
// MeshParts into either transparent or opaque buckets for rendering.
func (model *Model) isTransparent(meshPart *MeshPart) bool {
	mat := meshPart.Material
	return mat != nil && (mat.TransparencyMode == TransparencyModeTransparent || mat.CompositeMode != ebiten.CompositeModeSourceOver || (mat.TransparencyMode == TransparencyModeAuto && (mat.Color.A < 0.99 || model.Color.A < 0.99)))
}

////////

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
