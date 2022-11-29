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
	ColorBlendingFunc func(model *Model, meshPart *MeshPart) ebiten.ColorM // A user-customizeable blending function used to color the Model.
	BoundingSphere    *BoundingSphere

	DynamicBatchModels map[*MeshPart][]*Model // Models that are dynamically merged into this one.
	DynamicBatchOwner  *Model

	Skinned        bool  // If the model is skinned and this is enabled, the model will tranform its vertices to match the skinning armature (Model.SkinRoot).
	SkinRoot       INode // The root node of the armature skinning this Model.
	skinMatrix     Matrix4
	bones          [][]*Node // The bones (nodes) of the Model, assuming it has been skinned. A Mesh's bones slice will point to indices indicating bones in the Model.
	skinVectorPool *VectorPool

	// A LightGroup indicates if a Model should be lit by a specific group of Lights. This allows you to control the overall lighting of scenes more accurately.
	// If a Model has no LightGroup, the Model is lit by the lights present in the Scene.
	LightGroup *LightGroup

	// VertexTransformFunction is a function that runs on the world position of each vertex position rendered with the material.
	// It accepts the vertex position as an argument, along with the index of the vertex in the mesh.
	// One can use this to simply transform vertices of the mesh on CPU (note that this is, of course, not as performant as
	// a traditional GPU vertex shader, but is fine for simple / low-poly mesh transformations).
	// This function is run after skinning the vertex if the material belongs to a mesh that is skinned by an armature.
	// Note that the VertexTransformFunction must return the vector passed.
	VertexTransformFunction func(vertexPosition vector.Vector, vertexIndex int)

	// VertexClipFunction is a function that runs on the clipped result of each vertex position rendered with the material.
	// The function takes the vertex position along with the vertex index in the mesh.
	// This program runs after the vertex position is clipped to screen coordinates.
	// Note that the VertexClipFunction must return the vector passed.
	VertexClipFunction func(vertexPosition vector.Vector, vertexIndex int)
}

// NewModel creates a new Model (or instance) of the Mesh and Name provided. A Model represents a singular visual instantiation of a Mesh.
func NewModel(mesh *Mesh, name string) *Model {

	model := &Model{
		Node:               NewNode(name),
		Mesh:               mesh,
		FrustumCulling:     true,
		Color:              NewColor(1, 1, 1, 1),
		skinMatrix:         NewMatrix4(),
		DynamicBatchModels: map[*MeshPart][]*Model{},
	}

	model.Node.onTransformUpdate = model.TransformUpdate

	if mesh != nil {
		model.skinVectorPool = NewVectorPool(len(mesh.Triangles)*3*2, true) // Both position and normal
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

	for k := range model.DynamicBatchModels {
		newModel.DynamicBatchModels[k] = append([]*Model{}, model.DynamicBatchModels[k]...)
	}

	newModel.DynamicBatchOwner = model.DynamicBatchOwner

	newModel.Skinned = model.Skinned
	newModel.SkinRoot = model.SkinRoot
	for i := range model.bones {
		newModel.bones = append(newModel.bones, append([]*Node{}, model.bones[i]...))
	}

	newModel.Node = model.Node.Clone().(*Node)
	newModel.Node.onTransformUpdate = newModel.TransformUpdate
	for _, child := range newModel.children {
		child.setParent(newModel)
	}

	newModel.originalLocalPosition = model.originalLocalPosition.Clone()

	if model.LightGroup != nil {
		newModel.LightGroup = model.LightGroup.Clone()
	}

	newModel.VertexClipFunction = model.VertexClipFunction
	newModel.VertexTransformFunction = model.VertexTransformFunction

	return newModel

}

// Transform returns the global transform of the Model, taking into account any transforms its parents or grandparents have that
// would impact the Model.
func (model *Model) TransformUpdate() {

	transform := model.cachedTransform

	position, scale, rotation := transform.Decompose()

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

	center = rotation.MultVec(center)

	position[0] += center[0] * scale[0]
	position[1] += center[1] * scale[1]
	position[2] += center[2] * scale[2]
	model.BoundingSphere.SetLocalPositionVec(position)

	dim := model.Mesh.Dimensions.Clone()
	dim[0][0] *= scale[0]
	dim[0][1] *= scale[1]
	dim[0][2] *= scale[2]

	dim[1][0] *= scale[0]
	dim[1][1] *= scale[1]
	dim[1][2] *= scale[2]

	model.BoundingSphere.Radius = dim.MaxSpan() / 2

}

func (model *Model) modelAlreadyDynamicallyBatched(batchedModel *Model) bool {

	for _, modelSlice := range model.DynamicBatchModels {

		for _, model := range modelSlice {

			if model == batchedModel {
				return true
			}

		}

	}

	return false
}

// DynamicBatchAdd adds the provided models to the calling Model's dynamic batch, rendering with the specified meshpart (which should be part of the calling Model,
// of course). Note that unlike StaticMerge(), DynamicBatchAdd works by simply rendering the batched models using the calling Model's first MeshPart's material. By
// dynamically batching models together, this allows us to not flush between rendering multiple Models, saving a lot of render time, particularly if rendering many
// low-poly, individual models that have very little variance (i.e. if they all share a single texture).
// For more information, see this Wiki page on batching / merging: https://github.com/SolarLune/Tetra3d/wiki/Merging-and-Batching-Draw-Calls
func (model *Model) DynamicBatchAdd(meshPart *MeshPart, batchedModels ...*Model) error {

	for _, other := range batchedModels {

		if model == other || model.modelAlreadyDynamicallyBatched(other) {
			continue
		}

		triCount := model.DynamicBatchTriangleCount()

		if triCount+len(other.Mesh.Triangles) > MaxTriangleCount {
			return errors.New("too many triangles in dynamic merge")
		}

		if _, exists := model.DynamicBatchModels[meshPart]; !exists {
			model.DynamicBatchModels[meshPart] = []*Model{}
		}

		model.DynamicBatchModels[meshPart] = append(model.DynamicBatchModels[meshPart], other)
		other.DynamicBatchOwner = model

	}

	return nil

}

// DynamicBatchRemove removes the specified batched Models from the calling Model's dynamic batch slice.
func (model *Model) DynamicBatchRemove(batched ...*Model) {
	for _, m := range batched {
		for meshPartName, modelSlice := range model.DynamicBatchModels {
			for i, existing := range modelSlice {
				if existing == m {
					model.DynamicBatchModels[meshPartName][i] = nil
					model.DynamicBatchModels[meshPartName] = append(model.DynamicBatchModels[meshPartName][:i], model.DynamicBatchModels[meshPartName][i+1:]...)
					m.DynamicBatchOwner = nil
					break
				}
			}
		}
	}

	for mp := range model.DynamicBatchModels {
		if len(model.DynamicBatchModels[mp]) == 0 {
			delete(model.DynamicBatchModels, mp)
		}
	}
}

// DynamicBatchTriangleCount returns the total number of triangles of Models in the calling Model's dynamic batch.
func (model *Model) DynamicBatchTriangleCount() int {
	count := 0
	for _, models := range model.DynamicBatchModels {
		for _, child := range models {
			count += len(child.Mesh.Triangles)
		}
	}
	return count
}

// StaticMerge statically merges the provided models into the calling Model's mesh, such that their vertex properties (position, normal, UV, etc) are part of the calling Model's Mesh.
// You can use this to merge several objects initially dynamically placed into the calling Model's mesh, thereby pulling back to a single draw call. Note that models are merged into MeshParts
// (saving draw calls) based on maximum vertex count and shared materials (so to get any benefit from merging, ensure the merged models share materials; if they all have unique
// materials, they will be turned into individual MeshParts, thereby forcing multiple draw calls). Also note that as the name suggests, this is static merging, which means that
// after merging, the new vertices are static - part of the merging Model.
// For more information, see this Wiki page on batching / merging: https://github.com/SolarLune/Tetra3d/wiki/Merging-and-Batching-Draw-Calls
func (model *Model) StaticMerge(models ...*Model) {

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

	if int(model.Mesh.triIndex)+totalSize > len(model.Mesh.VertexPositions) {
		model.Mesh.allocateVertexBuffers(len(model.Mesh.VertexPositions) + totalSize)
	}

	vec := vector.Vector{0, 0, 0}

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
				if mp.Material == otherPart.Material && mp.TriangleCount()+otherPart.TriangleCount() < MaxTriangleCount {
					targetPart = mp
					break
				}
			}

			if targetPart == nil {
				targetPart = model.Mesh.AddMeshPart(otherPart.Material)
			}

			// Optimize these two
			verts := make([]VertexInfo, 0, otherPart.VertexIndexCount())
			indices := make([]int, 0, otherPart.TriangleCount()*3)

			otherPart.ForEachVertexIndex(func(vertIndex int) {

				vertInfo := otherPart.Mesh.GetVertexInfo(vertIndex)
				vec[0] = vertInfo.X
				vec[1] = vertInfo.Y
				vec[2] = vertInfo.Z
				x, y, z := fastMatrixMultVec(inverted, vec)
				vertInfo.X = x
				vertInfo.Y = y
				vertInfo.Z = z
				verts = append(verts, vertInfo)

			})

			model.Mesh.AddVertices(verts...)

			otherPart.ForEachTri(func(tri *Triangle) {
				for _, i := range tri.VertexIndices {
					indices = append(indices, i-otherPart.VertexIndexStart)
				}
			})

			targetPart.AddTriangles(indices...)

		}

	}

	model.Mesh.UpdateBounds()

	model.BoundingSphere.SetLocalPositionVec(model.Mesh.Dimensions.Center())
	model.BoundingSphere.Radius = model.Mesh.Dimensions.MaxSpan() / 2

	model.skinVectorPool = NewVectorPool(len(model.Mesh.Triangles)*3*2, true)

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

func (model *Model) skinVertex(vertID int) (vector.Vector, vector.Vector) {

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
			model.skinMatrix = model.skinMatrix.Add(bone.boneInfluence.ScaleByScalar(weightPerc)) // TODO: We should NOT be cloning a matrix for each vertex like this
		}

	}

	vertOut := model.skinVectorPool.MultVecW(model.skinMatrix, model.Mesh.VertexPositions[vertID])

	model.skinMatrix[3][0] = 0
	model.skinMatrix[3][1] = 0
	model.skinMatrix[3][2] = 0
	model.skinMatrix[3][3] = 1

	normal = model.skinVectorPool.MultVecW(model.skinMatrix, model.Mesh.VertexNormals[vertID])

	return vertOut, normal

}

var transformedVertexPositions = [3]vector.Vector{
	{0, 0},
	{0, 0},
	{0, 0},
}

// ProcessVertices processes the vertices a Model has in preparation for rendering, given a view-projection
// matrix, a camera, and the MeshPart being rendered.
func (model *Model) ProcessVertices(vpMatrix Matrix4, camera *Camera, meshPart *MeshPart, scene *Scene) []sortingTriangle {

	var transformFunc func(vertPos vector.Vector, index int)

	if model.VertexTransformFunction != nil {
		transformFunc = model.VertexTransformFunction
	}

	modelTransform := model.Transform()

	far := camera.Far

	sortingTriIndex := 0

	mat := meshPart.Material
	mesh := meshPart.Mesh
	base := modelTransform

	camPos := camera.WorldPosition()

	if mat != nil && mat.BillboardMode != BillboardModeNone {

		lookat := NewLookAtMatrix(model.WorldPosition(), camPos, vector.Y)

		if mat.BillboardMode == BillboardModeXZ {
			lookat.SetRow(1, vector.Vector{0, 1, 0, 0})
			x := lookat.Row(0)
			x[1] = 0
			lookat.SetRow(0, x.Unit())

			z := lookat.Row(2)
			z[1] = 0
			lookat.SetRow(2, z.Unit())
		}

		// This is the slowest part, for sure, but it's necessary to have a billboarded object still be accurate
		p, s, r := base.Decompose()
		base = r.Mult(lookat).Mult(NewMatrix4Scale(s[0], s[1], s[2]))
		base.SetRow(3, vector.Vector{p[0], p[1], p[2], 1})

	}

	mvp := fastMatrixMult(base, vpMatrix)

	// Reslice to fill the capacity
	meshPart.sortingTriangles = meshPart.sortingTriangles[:cap(meshPart.sortingTriangles)]

	if model.Skinned {
		model.skinVectorPool.Reset()
	}

	for ti := meshPart.TriangleStart; ti <= meshPart.TriangleEnd; ti++ {

		tri := mesh.Triangles[ti]

		meshPart.sortingTriangles[sortingTriIndex].Triangle = mesh.Triangles[ti]
		depth := math.MaxFloat64

		outOfBounds := true

		// If we're skinning a model, it will automatically copy the armature's position, scale, and rotation by copying its bones

		for i := 0; i < 3; i++ {

			if model.Skinned {

				t := time.Now()

				vertPos, vertNormal := model.skinVertex(tri.VertexIndices[i])
				if transformFunc != nil {
					transformFunc(vertPos, tri.VertexIndices[i])
				}
				if vertNormal != nil {
					mesh.vertexSkinnedNormals[tri.VertexIndices[i]] = vertNormal
					mesh.vertexSkinnedPositions[tri.VertexIndices[i]] = vertPos
				}
				transformed := mesh.vertexTransforms[tri.VertexIndices[i]]
				x, y, z, w := fastMatrixMultVecW(vpMatrix, vertPos)
				transformed[0] = x
				transformed[1] = y
				transformed[2] = z
				transformed[3] = w

				camera.DebugInfo.animationTime += time.Since(t)

			} else {

				v0 := mesh.VertexPositions[tri.VertexIndices[i]]

				if transformFunc != nil {
					v0 = v0.Clone()
					transformFunc(v0, tri.VertexIndices[i])
				}

				transformed := mesh.vertexTransforms[tri.VertexIndices[i]]
				transformed[0], transformed[1], transformed[2], transformed[3] = fastMatrixMultVecW(mvp, v0)

			}

			w := mesh.vertexTransforms[tri.VertexIndices[i]][3]

			if w < depth {
				depth = w
			}

			if !camera.Perspective {
				w = 1.0
			}

			// If the trangle is beyond the screen, we'll just pretend it's not and limit it to the closest possible value > 0
			// TODO: Replace this with triangle clipping or fix whatever graphical glitch seems to arise periodically
			if w < 0 {
				w = 0.000001
			}

			transformedVertexPositions[i][0] = mesh.vertexTransforms[tri.VertexIndices[i]][0] / w
			transformedVertexPositions[i][1] = mesh.vertexTransforms[tri.VertexIndices[i]][1] / w

			if mesh.vertexTransforms[tri.VertexIndices[i]][3] >= 0 && mesh.vertexTransforms[tri.VertexIndices[i]][2] < far {
				outOfBounds = false
			}

		}

		if outOfBounds {
			continue
		}

		// Backface culling

		if !outOfBounds && meshPart.Material != nil && meshPart.Material.BackfaceCulling {

			v0 := transformedVertexPositions[0]
			v1 := transformedVertexPositions[1]
			v2 := transformedVertexPositions[2]

			n0x := v0[0] - v1[0]
			n0y := v0[1] - v1[1]

			n1x := v1[0] - v2[0]
			n1y := v1[1] - v2[1]

			// Essentially calculating the cross product, but only for the important dimension (Z, which is "in / out" in this context)
			nor := (n0x * n1y) - (n1x * n0y)

			// We use this method of backface culling because it helps to ensure
			// there's fewer graphical glitches when looking from very near a surface outwards; this
			// doesn't help if a surface does not have backface culling, of course...
			if nor < 0 {
				outOfBounds = true
			}

			// Old methods of backface culling; performant, but causes graphical glitches.

			// if backfaceCulling {

			// 	camera.backfacePool.Reset()
			// 	n0 := camera.backfacePool.Sub(p0, p1)
			// 	n1 := camera.backfacePool.Sub(p1, p2)
			// 	nor := camera.backfacePool.Cross(n0, n1)

			// 	if nor[2] > 0 {
			// 		continue
			// 	}

			// }
			// if model.Skinned {
			// 	v0 := model.Mesh.vertexSkinnedNormals[tri.VertexIndices[0]]
			// 	v1 := model.Mesh.vertexSkinnedNormals[tri.VertexIndices[1]]
			// 	v2 := model.Mesh.vertexSkinnedNormals[tri.VertexIndices[2]]

			// 	backfaceTriNormal[0] = (v0[0] + v1[0] + v2[0]) / 3
			// 	backfaceTriNormal[1] = (v0[1] + v1[1] + v2[1]) / 3
			// 	backfaceTriNormal[2] = (v0[2] + v1[2] + v2[2]) / 3

			// 	interimVec[0] = camPos[0] - model.Mesh.vertexSkinnedPositions[tri.VertexIndices[0]][0]
			// 	interimVec[1] = camPos[1] - model.Mesh.vertexSkinnedPositions[tri.VertexIndices[0]][1]
			// 	interimVec[2] = camPos[2] - model.Mesh.vertexSkinnedPositions[tri.VertexIndices[0]][2]

			// 	if dot(backfaceTriNormal, interimVec) < 0 {
			// 		continue
			// 	}

			// } else {

			// 	interimVec[0] = invertedCamPos[0] - tri.Center[0]
			// 	interimVec[1] = invertedCamPos[1] - tri.Center[1]
			// 	interimVec[2] = invertedCamPos[2] - tri.Center[2]

			// 	if dot(tri.Normal, interimVec) < 0 {
			// 		continue
			// 	}

			// }

		}

		if !outOfBounds {

			// If all transformed vertices are wholly out of bounds to the right, left, top, or bottom of the screen, then we can assume
			// the triangle does not need to be rendered
			if (transformedVertexPositions[0][0] < -0.5 && transformedVertexPositions[1][0] < -0.5 && transformedVertexPositions[2][0] < -0.5) ||
				(transformedVertexPositions[0][0] > 0.5 && transformedVertexPositions[1][0] > 0.5 && transformedVertexPositions[2][0] > 0.5) ||
				(transformedVertexPositions[0][1] < -0.5 && transformedVertexPositions[1][1] < -0.5 && transformedVertexPositions[2][1] < -0.5) ||
				(transformedVertexPositions[0][1] > 0.5 && transformedVertexPositions[1][1] > 0.5 && transformedVertexPositions[2][1] > 0.5) {
				outOfBounds = true
			}

		}

		if outOfBounds {
			continue
		}

		meshPart.sortingTriangles[sortingTriIndex].depth = float32(depth)

		sortingTriIndex++

	}

	meshPart.sortingTriangles = meshPart.sortingTriangles[:sortingTriIndex]

	sortMode := TriangleSortModeBackToFront

	if meshPart.Material != nil {
		sortMode = meshPart.Material.TriangleSortMode
	}

	// Preliminary tests indicate sort.SliceStable is faster than sort.Slice for our purposes

	if sortMode == TriangleSortModeBackToFront {
		sort.Slice(meshPart.sortingTriangles, func(i, j int) bool {
			return meshPart.sortingTriangles[i].depth > meshPart.sortingTriangles[j].depth
		})
	} else if sortMode == TriangleSortModeFrontToBack {
		sort.Slice(meshPart.sortingTriangles, func(i, j int) bool {
			return meshPart.sortingTriangles[i].depth < meshPart.sortingTriangles[j].depth
		})
	}

	return meshPart.sortingTriangles

}

type AOBakeOptions struct {
	TargetChannel  int     // The target vertex color channel to bake the ambient occlusion to.
	OcclusionAngle float64 // How severe the angle must be (in radians) for the occlusion effect to show up.
	Color          *Color  // The color for the ambient occlusion.

	// A slice indicating other models that influence AO when baking. If this is empty, the AO will
	// just take effect for triangles within the Model, rather than also taking effect for objects that
	// are too close to the baking Model.
	OtherModels        []*Model
	InterModelDistance float64 // How far the other models in OtherModels must be to influence the baking AO.
}

// NewDefaultAOBakeOptions creates a new AOBakeOptions struct with default settings.
func NewDefaultAOBakeOptions() *AOBakeOptions {

	return &AOBakeOptions{
		TargetChannel:      0,
		OcclusionAngle:     ToRadians(60),
		Color:              NewColor(0.4, 0.4, 0.4, 1),
		InterModelDistance: 1,
		OtherModels:        []*Model{},
	}

}

// BakeAO bakes the ambient occlusion for a model to its vertex colors, using the baking options set in the provided AOBakeOptions
// struct. If a slice of models is passed in the OtherModels slice, then inter-object AO will also be baked.
// If nil is passed instead of bake options, a default AOBakeOptions struct will be created and used.
// The resulting vertex color will be mixed between whatever was originally there in that channel and the AO color where the color
// takes effect.
func (model *Model) BakeAO(bakeOptions *AOBakeOptions) {

	if bakeOptions == nil {
		bakeOptions = NewDefaultAOBakeOptions()
	}

	if model.Mesh == nil || bakeOptions.TargetChannel < 0 {
		return
	}

	model.Mesh.ensureEnoughVertexColorChannels(bakeOptions.TargetChannel)

	// Same model AO first

	for _, tri := range model.Mesh.Triangles {

		ao := [3]float32{0, 0, 0}

		verts := tri.VertexIndices

		for _, other := range model.Mesh.Triangles {

			if tri == other || vectorsEqual(tri.Normal, other.Normal) {
				continue
			}

			span := tri.MaxSpan
			if other.MaxSpan > span {
				span = other.MaxSpan
			}

			span *= 0.66

			if fastVectorDistanceSquared(tri.Center, other.Center) > span*span {
				continue
			}

			angle := tri.Normal.Angle(other.Normal)
			if angle < bakeOptions.OcclusionAngle {
				continue
			}

			if shared := tri.SharesVertexPositions(other); shared != nil {

				if shared[0] >= 0 {
					ao[0] = 1
				}
				if shared[1] >= 0 {
					ao[1] = 1
				}
				if shared[2] >= 0 {
					ao[2] = 1
				}

			}

		}

		for i := 0; i < 3; i++ {
			model.Mesh.VertexColors[verts[i]][bakeOptions.TargetChannel].Mix(bakeOptions.Color, ao[i])
		}

	}

	// Inter-object AO next; this is kinda slow and janky, but it does work OK, I think

	transform := model.Transform()

	distanceSquared := bakeOptions.InterModelDistance * bakeOptions.InterModelDistance

	for _, other := range bakeOptions.OtherModels {

		rad := model.BoundingSphere.WorldRadius()
		if or := other.BoundingSphere.WorldRadius(); or > rad {
			rad = or
		}
		if model == other || fastVectorDistanceSquared(model.WorldPosition(), other.WorldPosition()) > rad*rad {
			continue
		}

		otherTransform := other.Transform()

		for _, tri := range model.Mesh.Triangles {

			ao := [3]float32{0, 0, 0}

			verts := tri.VertexIndices

			transformedTriVerts := [3]vector.Vector{
				transform.MultVec(model.Mesh.VertexPositions[verts[0]]),
				transform.MultVec(model.Mesh.VertexPositions[verts[1]]),
				transform.MultVec(model.Mesh.VertexPositions[verts[2]]),
			}

			for _, otherTri := range other.Mesh.Triangles {

				otherVerts := otherTri.VertexIndices

				span := tri.MaxSpan
				if otherTri.MaxSpan > span {
					span = otherTri.MaxSpan
				}

				span *= 0.66

				if fastVectorDistanceSquared(transform.MultVec(tri.Center), otherTransform.MultVec(otherTri.Center)) > span {
					continue
				}

				transformedOtherVerts := [3]vector.Vector{
					otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[0]]),
					otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[1]]),
					otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[2]]),
				}

				for i := 0; i < 3; i++ {
					for j := 0; j < 3; j++ {
						if fastVectorDistanceSquared(transformedTriVerts[i], transformedOtherVerts[j]) <= distanceSquared {
							ao[i] = 1
							break
						}
					}
				}

			}

			for i := 0; i < 3; i++ {
				model.Mesh.VertexColors[verts[i]][bakeOptions.TargetChannel].Mix(bakeOptions.Color, ao[i])
			}

		}

	}

}

// BakeLighting bakes the colors for the provided lights into a Model's Mesh's vertex colors. Note that the baked lighting overwrites whatever vertex colors
// previously existed in the target channel (as otherwise, the colors could only get brighter with additive mixing, or only get darker with multiplicative mixing).
func (model *Model) BakeLighting(targetChannel int, lights ...ILight) {

	if model.Mesh == nil || targetChannel < 0 {
		return
	}

	model.Mesh.ensureEnoughVertexColorChannels(targetChannel)

	allLights := append([]ILight{}, lights...)

	if model.Scene() != nil {
		allLights = append(allLights, model.Scene().World.AmbientLight)
	}

	for _, light := range allLights {

		if light.IsOn() {

			light.beginRender()
			light.beginModel(model)

		}

	}

	// TODO: Switch Mesh.VertexColors around so instead of [vertex index][channel] it's [channel][vertex index]; this way we could directly pass
	// a vertex channel into Light.Light().

	targetColors := make([]*Color, len(model.Mesh.VertexColors))

	for i := 0; i < len(targetColors); i++ {
		targetColors[i] = NewColor(0, 0, 0, 1)
	}

	for _, light := range allLights {

		if light.IsOn() {
			for _, mp := range model.Mesh.MeshParts {
				light.Light(mp, model, targetColors)
			}
		}

	}

	for i := 0; i < len(model.Mesh.VertexColors); i++ {
		model.Mesh.VertexColors[i][targetChannel].Set(targetColors[i].ToFloat32s())
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
