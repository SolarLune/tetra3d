package tetra3d

import (
	"errors"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/solarlune/tetra3d/math32"
)

const (
	AutoBatchNone    = iota // No automatic batching
	AutoBatchDynamic        // Dynamically batch
	AutoBatchStatic         // Statically merge
)

// Model represents a singular visual instantiation of a Mesh. A Mesh contains the vertex information (what to draw); a Model references the Mesh to draw it with a specific
// Position, Rotation, and/or Scale (where and how to draw).
type Model struct {
	*Node
	Mesh                 *Mesh
	FrustumCulling       bool            // Whether the Model is culled when it leaves the frustum.
	frustumCullingSphere *BoundingSphere // Used for frustum culling
	updateFrustumSphere  bool            // True usually, but false for particles because we calculate it ourselves
	Color                Color           // The overall multiplicative color of the Model.
	Shadeless            bool            // Indicates if a Model is shadeless.

	DynamicBatchModels map[*MeshPart][]*Model // Models that are dynamically merged into this one.
	DynamicBatchOwner  *Model

	skinned    bool  // If the model is skinned and this is enabled, the model will tranform its vertices to match the skinning armature (Model.SkinRoot).
	SkinRoot   INode // The root node of the armature skinning this Model.
	skinMatrix Matrix4
	bones      [][]*Node // The bones (nodes) of the Model, assuming it has been skinned. A Mesh's bones slice will point to indices indicating bones in the Model.

	// A LightGroup indicates if a Model should be lit by a specific group of Lights. This allows you to control the overall lighting of scenes more accurately.
	// If a Model has no LightGroup, the Model is lit by the lights present in the Scene.
	LightGroup *LightGroup

	// VertexTransformFunction is a function that runs on the world position of each vertex position rendered with the material.
	// It accepts the vertex position as an argument, along with the index of the vertex in the mesh.
	// One can use this to simply transform vertices of the mesh on CPU (note that this is, of course, not as performant as
	// a traditional GPU vertex shader, but is fine for simple / low-poly mesh transformations).
	// This function is run after skinning the vertex if the material belongs to a mesh that is skinned by an armature.
	// Note that the VertexTransformFunction must return the vector passed.
	VertexTransformFunction func(vertexPosition *Vector3, vertexIndex int)

	// VertexClipFunction is a function that runs on the clipped result of each vertex position rendered with the material.
	// The function takes the vertex position along with the vertex index in the mesh.
	// This program runs after the vertex position is clipped to screen coordinates.
	// Note that the VertexClipFunction must return the vector passed.
	VertexClipFunction func(vertexPosition *Vector4, vertexIndex int)

	// Automatic batching mode; when set and a Model changes parenting, it will be automatically batched as necessary according to
	// the AutoBatchMode set.
	AutoBatchMode int
	autoBatched   bool

	sector *Sector // Sector is a reference to the Sector object that the Model stands in for, if sector-based rendering is enabled.
}

// NewModel creates a new Model (or instance) of the Mesh and Name provided. A Model represents a singular visual instantiation of a Mesh.
func NewModel(name string, mesh *Mesh) *Model {

	model := &Model{
		Node:                NewNode(name),
		Mesh:                mesh,
		FrustumCulling:      true,
		updateFrustumSphere: true,
		Color:               NewColor(1, 1, 1, 1),
		skinMatrix:          NewMatrix4(),
		DynamicBatchModels:  map[*MeshPart][]*Model{},
	}

	model.owner = model
	model.Node.onTransformUpdate = model.onTransformUpdate

	radius := float32(0)
	if mesh != nil {
		radius = mesh.Dimensions.MaxSpan() / 2
	}
	model.frustumCullingSphere = NewBoundingSphere("bounding sphere", radius)

	return model

}

// Clone creates a clone of the Model.
func (model *Model) Clone() INode {

	mesh := model.Mesh
	if mesh != nil && mesh.Unique != MeshUniqueFalse {
		mesh = mesh.Clone()
	}

	newModel := NewModel(model.name, mesh)
	newModel.frustumCullingSphere = model.frustumCullingSphere.Clone().(*BoundingSphere)
	newModel.FrustumCulling = model.FrustumCulling
	newModel.updateFrustumSphere = model.updateFrustumSphere
	newModel.visible = model.visible
	newModel.Color = model.Color
	newModel.Shadeless = model.Shadeless
	newModel.AutoBatchMode = model.AutoBatchMode

	for k := range model.DynamicBatchModels {
		newModel.DynamicBatchModels[k] = append([]*Model{}, model.DynamicBatchModels[k]...)
	}

	newModel.DynamicBatchOwner = model.DynamicBatchOwner

	newModel.skinned = model.skinned
	newModel.SkinRoot = model.SkinRoot
	for i := range model.bones {
		newModel.bones = append(newModel.bones, append([]*Node{}, model.bones[i]...))
	}

	newModel.Node = model.Node.clone(newModel).(*Node)
	newModel.Node.onTransformUpdate = newModel.onTransformUpdate

	if model.LightGroup != nil {
		newModel.LightGroup = model.LightGroup.Clone()
	}

	newModel.VertexClipFunction = model.VertexClipFunction
	newModel.VertexTransformFunction = model.VertexTransformFunction

	if model.sector != nil {
		newModel.sector = model.sector.Clone()
		newModel.sector.Model = newModel
	}

	if runCallbacks && newModel.Callbacks().OnClone != nil {
		newModel.Callbacks().OnClone(newModel)
	}

	return newModel

}

// When updating a Model's transform, we have to also update its bounding sphere for frustum culling.
func (model *Model) onTransformUpdate() {

	if !model.updateFrustumSphere {
		return
	}

	transform := model.cachedTransform

	position, scale, rotation := transform.Decompose()

	// Skinned models have their positions at 0, 0, 0, and vertices offset according to wherever they were when exported.
	// To combat this, we save the original local positions of the mesh on export to position the bounding sphere in the
	// correct location.

	var center Vector3

	// We do this because if a model is skinned and we've parented the model to the armature, then the center is
	// now from origin relative to the base of the armature on scene export. Otherwise, it's the center of the mesh.
	if !model.skinned {
		center = model.Mesh.Dimensions.Center()
	}

	center = rotation.MultVec(center)

	position.X += center.X * scale.X
	position.Y += center.Y * scale.Y
	position.Z += center.Z * scale.Z
	model.frustumCullingSphere.SetLocalPositionVec(position)

	dim := model.Mesh.Dimensions
	dim.Min.X *= scale.X
	dim.Min.Y *= scale.Y
	dim.Min.Z *= scale.Z

	dim.Max.X *= scale.X
	dim.Max.Y *= scale.Y
	dim.Max.Z *= scale.Z

	model.frustumCullingSphere.Radius = dim.MaxSpan() / 2

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

/*
DynamicBatchAdd adds the provided models to the calling Model's dynamic batch, rendering with the specified meshpart (which should be part of the calling Model,
of course). Note that unlike StaticMerge(), DynamicBatchAdd works by simply rendering the batched models using the calling Model's first MeshPart's material. By
dynamically batching models together, this allows us to not flush between rendering multiple Models, saving a lot of render time, particularly if rendering many
low-poly, individual models that have very little variance (i.e. if they all share a single texture).
Calling this turns the model into a dynamic batching owner, meaning that it will no longer render its own mesh (for simplicity).
For more information, see this Wiki page on batching / merging: https://github.com/SolarLune/Tetra3d/wiki/Merging-and-Batching-Draw-Calls
*/
func (model *Model) DynamicBatchAdd(meshPart *MeshPart, batchedModels ...*Model) error {

	for _, other := range batchedModels {

		if model == other || model.modelAlreadyDynamicallyBatched(other) {
			continue
		}

		// It must be owned by another batch
		if other.DynamicBatchOwner != nil {
			other.DynamicBatchOwner.DynamicBatchRemove(other)
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

func (model *Model) DynamicBatcher() bool {
	return len(model.DynamicBatchModels) > 0
}

// DynamicBatchClear clears the model's dynamic batch map from any models associated with it.
func (model *Model) DynamicBatchClear() {

	for _, modelSlice := range model.DynamicBatchModels {
		for _, existing := range modelSlice {
			existing.DynamicBatchOwner = nil
		}
	}

	for mp := range model.DynamicBatchModels {
		delete(model.DynamicBatchModels, mp)
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

	vec := Vector3{}

	for _, other := range models {

		if model == other {
			continue
		}

		p, s, r := model.Transform().Decompose()
		op, os, or := other.Transform().Decompose()

		inverted := NewMatrix4Scale(os.X, os.Y, os.Z)
		scaleMatrix := NewMatrix4Scale(s.X, s.Y, s.Z)
		inverted = inverted.Mult(scaleMatrix)

		inverted = inverted.Mult(r.Transposed().Mult(or))

		inverted = inverted.Mult(NewMatrix4Translate(op.X-p.X, op.Y-p.Y, op.Z-p.Z))

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

				vec.X = vertInfo.X
				vec.Y = vertInfo.Y
				vec.Z = vertInfo.Z

				vec = inverted.MultVec(vec)

				vertInfo.X = vec.X
				vertInfo.Y = vec.Y
				vertInfo.Z = vec.Z

				verts = append(verts, vertInfo)

			}, false)

			model.Mesh.AddVertices(verts...)

			otherPart.ForEachTri(func(tri *Triangle) {
				for _, i := range tri.VertexIndices {
					indices = append(indices, i-otherPart.VertexIndexStart+model.Mesh.vertsAddStart)
				}
			})

			targetPart.AddTriangles(indices...)

		}

	}

	model.Mesh.UpdateBounds()

	model.frustumCullingSphere.SetLocalPositionVec(model.Mesh.Dimensions.Center())
	model.frustumCullingSphere.Radius = model.Mesh.Dimensions.MaxSpan() / 2

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

	bones := armatureRoot.SearchTree().INodes()

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

func (model *Model) skinVertex(vertID int) (Vector3, Vector3) {

	// Avoid reallocating a new matrix for every vertex; that's wasteful
	model.skinMatrix.Clear()

	for boneIndex, bone := range model.bones[vertID] {

		weightPerc := float32(model.Mesh.VertexWeights[vertID][boneIndex])

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
			model.skinMatrix = model.skinMatrix.Add(bone.boneInfluence.Scale(weightPerc))
		}

	}

	vertOut := model.skinMatrix.MultVec(model.Mesh.VertexPositions[vertID])

	model.skinMatrix[3][0] = 0
	model.skinMatrix[3][1] = 0
	model.skinMatrix[3][2] = 0
	model.skinMatrix[3][3] = 1

	normal := model.skinMatrix.MultVec(model.Mesh.VertexNormals[vertID])

	return vertOut, normal

}

func (model *Model) refreshVertexVisibility() {
	for i := range model.Mesh.visibleVertices {
		model.Mesh.visibleVertices[i] = false
	}
}

// ProcessVertices processes the vertices a Model has in preparation for rendering, given a view-projection
// matrix, a camera, and the MeshPart being rendered.
func (model *Model) ProcessVertices(vpMatrix Matrix4, camera *Camera, meshPart *MeshPart, processOnlyVisible bool) {

	globalSortingTriangleBucket.Clear()

	if processOnlyVisible && ((model.Color.A == 0 && model.isTransparent(meshPart)) || !model.visible) {
		return
	}

	var transformFunc func(vertPos *Vector3, index int)

	if model.VertexTransformFunction != nil {
		transformFunc = model.VertexTransformFunction
	}

	modelTransform := model.Transform()

	sortingTriIndex := 0

	mat := meshPart.Material
	mesh := meshPart.Mesh
	base := modelTransform

	camPos := camera.WorldPosition()

	// invertedCamPos := modelTransform.Inverted().MultVec(camPos)

	// TODO: Review this, as it still seems problematic when it comes to distance checks for pre-emptive culling?
	p, s, r := modelTransform.Inverted().Decompose()
	invertedCamPos := r.MultVec(camPos).Add(p.Mult(Vector3{1 / s.X, 1 / s.Y, 1 / s.Z}))

	// invertedCamPos := camPos

	if mat != nil && mat.BillboardMode != BillboardModeNone {

		var lookat Matrix4

		if mat.BillboardMode == BillboardModeFixedVertical {

			out := camera.cameraForward.Invert()
			lookat = NewLookAtMatrix(Vector3{}, out, camera.cameraUp)

		} else if mat.BillboardMode != BillboardModeNone {

			lookat = NewLookAtMatrix(model.WorldPosition(), camPos, WorldUp)

			if mat.BillboardMode == BillboardModeHorizontal {
				lookat.SetRow(1, Vector4{0, 1, 0, 0})
				x := lookat.Row(0)
				x.Y = 0
				lookat.SetRow(0, x.Unit())

				z := lookat.Row(2)
				z.Y = 0
				lookat.SetRow(2, z.Unit())
			}

			// This is the slowest part, for sure, but it's necessary to have a billboarded object still be accurate

		}

		p, s, r := base.Decompose()
		base = r.Mult(NewMatrix4Scale(s.X, s.Y, s.Z)).Mult(lookat)
		base.SetRow(3, Vector4{p.X, p.Y, p.Z, 1})

	}

	mvp := base.Mult(vpMatrix)

	var unalteredMVP Matrix4
	unbillboarded := false

	if mat != nil && mat.DepthMode == CustomDepthModeUnbillboarded {
		unbillboarded = true
		unalteredMVP = modelTransform.Mult(vpMatrix)
	}

	var mvJustRForNormals Matrix4
	if camera.RenderNormals {
		_, _, mvJustRForNormals = modelTransform.Mult(camera.ViewMatrix()).Decompose()
	}

	transformFuncExists := transformFunc != nil

	minDepth := float32(math.MaxFloat32)
	maxDepth := -float32(math.MaxFloat32)

	camNear := camera.near
	camFar := camera.far
	modelSkinned := model.skinned
	vertexSnappingOn := camera.VertexSnapping > 0
	renderNormals := camera.RenderNormals

	var vert Vector3
	var normal Vector3

	var t time.Time
	if modelSkinned {
		t = time.Now()
	}

	vertexPositions := mesh.VertexPositions
	vertexTransforms := mesh.vertexTransforms

	for vertexIndex := meshPart.VertexIndexStart; vertexIndex < meshPart.VertexIndexEnd; vertexIndex++ {

		if modelSkinned {

			vert, normal = model.skinVertex(vertexIndex)

			if transformFuncExists {
				transformFunc(&vert, vertexIndex)
			}

			mesh.vertexSkinnedNormals[vertexIndex].X = normal.X
			mesh.vertexSkinnedNormals[vertexIndex].Y = normal.Y
			mesh.vertexSkinnedNormals[vertexIndex].Z = normal.Z

			mesh.vertexSkinnedPositions[vertexIndex].X = vert.X
			mesh.vertexSkinnedPositions[vertexIndex].Y = vert.Y
			mesh.vertexSkinnedPositions[vertexIndex].Z = vert.Z

			// MultVecW() matrix multiplication, but faster to do it right here rather than using functions and/or pointers
			mesh.vertexTransforms[vertexIndex].X = vpMatrix[0][0]*vert.X + vpMatrix[1][0]*vert.Y + vpMatrix[2][0]*vert.Z + vpMatrix[3][0]
			mesh.vertexTransforms[vertexIndex].Y = vpMatrix[0][1]*vert.X + vpMatrix[1][1]*vert.Y + vpMatrix[2][1]*vert.Z + vpMatrix[3][1]
			mesh.vertexTransforms[vertexIndex].Z = vpMatrix[0][2]*vert.X + vpMatrix[1][2]*vert.Y + vpMatrix[2][2]*vert.Z + vpMatrix[3][2]
			mesh.vertexTransforms[vertexIndex].W = vpMatrix[0][3]*vert.X + vpMatrix[1][3]*vert.Y + vpMatrix[2][3]*vert.Z + vpMatrix[3][3]

		} else {

			// It is, of course, faster to set the values in a vertex than to allocate memory for a new one
			vert.X = vertexPositions[vertexIndex].X
			vert.Y = vertexPositions[vertexIndex].Y
			vert.Z = vertexPositions[vertexIndex].Z

			if transformFunc != nil {
				transformFunc(&vert, vertexIndex)
			}

			vertexTransforms[vertexIndex].X = mvp[0][0]*vert.X + mvp[1][0]*vert.Y + mvp[2][0]*vert.Z + mvp[3][0]
			vertexTransforms[vertexIndex].Y = mvp[0][1]*vert.X + mvp[1][1]*vert.Y + mvp[2][1]*vert.Z + mvp[3][1]

			if unbillboarded {
				vertexTransforms[vertexIndex].Z = unalteredMVP[0][2]*vert.X + unalteredMVP[1][2]*vert.Y + unalteredMVP[2][2]*vert.Z + unalteredMVP[3][2]
				vertexTransforms[vertexIndex].W = unalteredMVP[0][3]*vert.X + unalteredMVP[1][3]*vert.Y + unalteredMVP[2][3]*vert.Z + unalteredMVP[3][3]
			} else {
				vertexTransforms[vertexIndex].Z = mvp[0][2]*vert.X + mvp[1][2]*vert.Y + mvp[2][2]*vert.Z + mvp[3][2]
				vertexTransforms[vertexIndex].W = mvp[0][3]*vert.X + mvp[1][3]*vert.Y + mvp[2][3]*vert.Z + mvp[3][3]
			}

		}

		if vertexSnappingOn {
			mesh.vertexTransforms[vertexIndex] = mesh.vertexTransforms[vertexIndex].Round(camera.VertexSnapping)
		}

		if renderNormals {
			mesh.vertexTransformedNormals[vertexIndex] = mvJustRForNormals.MultVec(mesh.VertexNormals[vertexIndex])
		}

	}

	if modelSkinned {
		camera.DebugInfo.currentAnimationTime += time.Since(t)
	}

	var skinnedTriCenter Vector3
	var transformedVertexPositions = [3]Vector3{}

	for ti := meshPart.TriangleStart; ti <= meshPart.TriangleEnd; ti++ {

		tri := mesh.Triangles[ti]

		vertIndices := tri.VertexIndices

		// Backface culling
		// if meshPart.Material != nil && meshPart.Material.BackfaceCulling {

		// 	// For a perspective camera, we can use the inverted camera position to do a normal test against the triangle's center.
		// 	// For an orthographic camera, we have to check the normal against the direction the camera's facing (inverted by the model's rotation).
		// 	if (camera.perspective && tri.Normal.Dot(invertedCamPos.Sub(tri.Center)) < 0) || (!camera.perspective && tri.Normal.Dot(invertedCamForward) > 0) {
		// 		continue
		// 	}

		// }

		// If we're skinning a model, it will automatically copy the armature's position, scale, and rotation by copying its bones

		// Skinned models transform vertices according to animations (and so the triangle center),
		// so we have to keep track of the skinned center (which is done by averaging the vertex positions for the triangle),
		// and then we can divide that by 3 to sort the triangles.
		if modelSkinned {
			skinnedTriCenter.X = 0
			skinnedTriCenter.Y = 0
			skinnedTriCenter.Z = 0
		}

		// This was causing problems, so I axed it
		// invertedCamDist := invertedCamPos.DistanceSquared(tri.Center)

		// if !model.skinned && invertedCamDist > farSquared {
		// 	continue
		// }

		// View clipping

		outOfBounds := true

		for i := 0; i < 3; i++ {

			// It's faster to store the indices of the triangles in a variable than constantly dereference a pointer
			if modelSkinned {
				skinnedTriCenter = skinnedTriCenter.Add(mesh.vertexSkinnedPositions[vertIndices[i]])
			}

			w := mesh.vertexTransforms[vertIndices[i]].W

			// If the trangle is beyond the screen, we'll just pretend it's not and limit it to the closest possible value > 0
			// TODO: Replace this with triangle clipping or fix whatever graphical glitch seems to arise periodically
			if w < 0 {
				w = 0.000001
			}

			transformedVertexPositions[i].X = mesh.vertexTransforms[vertIndices[i]].X / w
			transformedVertexPositions[i].Y = mesh.vertexTransforms[vertIndices[i]].Y / w

			if mesh.vertexTransforms[vertIndices[i]].Z+1 >= camNear && mesh.vertexTransforms[vertIndices[i]].Z < camFar {
				outOfBounds = false
			}

		}

		if outOfBounds {
			continue
		}

		// If all transformed vertices are wholly out of bounds to the right, left, top, or bottom of the screen, then we can assume
		// the triangle does not need to be rendered
		if (transformedVertexPositions[0].X < -0.5 && transformedVertexPositions[1].X < -0.5 && transformedVertexPositions[2].X < -0.5) ||
			(transformedVertexPositions[0].X > 0.5 && transformedVertexPositions[1].X > 0.5 && transformedVertexPositions[2].X > 0.5) ||
			(transformedVertexPositions[0].Y < -0.5 && transformedVertexPositions[1].Y < -0.5 && transformedVertexPositions[2].Y < -0.5) ||
			(transformedVertexPositions[0].Y > 0.5 && transformedVertexPositions[1].Y > 0.5 && transformedVertexPositions[2].Y > 0.5) {
			continue
		}

		// Going back to using the transformed vertex positions for backface culling as it works better when the camera is super close to the
		// triangles.
		if meshPart.Material != nil && meshPart.Material.BackfaceCulling {

			v0 := transformedVertexPositions[0]
			v1 := transformedVertexPositions[1]
			v2 := transformedVertexPositions[2]

			n0x := v0.X - v1.X
			n0y := v0.Y - v1.Y

			n1x := v1.X - v2.X
			n1y := v1.Y - v2.Y

			// Essentially calculating the cross product, but only for the important dimension (Z, which is "in / out" in this context)
			nor := (n0x * n1y) - (n1x * n0y)

			// We use this method of backface culling because it helps to ensure
			// there's fewer graphical glitches when looking from very near a surface outwards; this
			// doesn't help if a surface does not have backface culling, of course...
			if nor < 0 {
				continue
			}

		}

		// Previously, depth was compared using the lowest W value of all vertices in the triangle; after that, I tried
		// averaging them out. Neither of these was completely satisfactory, and in addition, there was no depth sorting
		// for orthographic triangles that overlapped using this method.

		depth := float32(0)

		// if modelSkinned {
		// 	depth = float32(camPos.DistanceSquared(skinnedTriCenter.Divide(3)))
		// } else {
		// 	depth = float32(invertedCamPos.DistanceSquared(tri.Center))
		// }

		if modelSkinned {

			dx := camPos.X - (skinnedTriCenter.X / 3)
			dy := camPos.Y - (skinnedTriCenter.Y / 3)
			dz := camPos.Z - (skinnedTriCenter.Z / 3)
			depth = float32(dx*dx + dy*dy + dz*dz)

		} else {

			dx := invertedCamPos.X - tri.Center.X
			dy := invertedCamPos.Y - tri.Center.Y
			dz := invertedCamPos.Z - tri.Center.Z
			depth = float32(dx*dx + dy*dy + dz*dz)

		}

		if depth < minDepth {
			minDepth = depth
		}
		if depth > maxDepth {
			maxDepth = depth
		}

		globalSortingTriangleBucket.AddTriangle(ti, depth, vertIndices)

		// I could substitute depth for W, but sorting by distance to the triangle center directly gives a better result overall, it seems.
		// if sortMode != TriangleSortModeNone {
		// 	if model.skinned {
		// 		meshPart.sortingTriangles[sortingTriIndex].depth = float32(camPos.DistanceSquared(skinnedTriCenter.Divide(3)))
		// 	} else {
		// 		// meshPart.sortingTriangles[sortingTriIndex].depth = float32(invertedCamDist)
		// 		meshPart.sortingTriangles[sortingTriIndex].depth = float32(invertedCamPos.DistanceSquared(tri.Center))
		// 	}
		// }

		mesh.visibleVertices[vertIndices[0]] = true
		mesh.visibleVertices[vertIndices[1]] = true
		mesh.visibleVertices[vertIndices[2]] = true

		sortingTriIndex++

	}

	globalSortingTriangleBucket.Sort(minDepth, maxDepth)

}

// Dimensions returns the transformed dimensions of the Model's mesh.
func (model *Model) Dimensions() Dimensions {
	dim := model.Mesh.Dimensions
	transform := model.Transform()
	dim.Min = transform.MultVec(dim.Min)
	dim.Max = transform.MultVec(dim.Max)
	return dim
}

type AOBakeOptions struct {
	TargetMeshParts []*MeshPart // The target meshparts / materials to use for baking AO values to. If not set, then all meshparts will be used.
	SourceVertices  VertexSelection
	// The target vertex color channel to bake the ambient occlusion to.
	// If the Model doesn't have enough vertex color channels to bake to this channel index, the BakeAO() function will
	// create vertex color channels to fill in the values up to the target channel index.
	TargetChannel  int
	OcclusionAngle float32 // How severe the angle must be (in radians) for the occlusion effect to show up.
	OcclusionColor Color   // The color for the ambient occlusion.

	// A filter indicating other models that influence ambient occlusion when baking. If this is not set, the AO will
	// just take effect for triangles within the Model, rather than also taking effect for objects that
	// are close to the baking Model.
	OtherModels        NodeFilter
	InterModelDistance float32 // How far the other models in OtherModels must be to influence the baking AO.
}

// NewDefaultAOBakeOptions creates a new AOBakeOptions struct with default settings.
func NewDefaultAOBakeOptions() *AOBakeOptions {

	return &AOBakeOptions{
		TargetChannel:      0,
		OcclusionAngle:     math32.ToRadians(60),
		OcclusionColor:     NewColor(0.4, 0.4, 0.4, 1),
		InterModelDistance: 1,
	}

}

// BakeAO bakes the ambient occlusion for a model to its vertex colors, using the baking options set in the provided AOBakeOptions
// struct.
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

		if len(bakeOptions.TargetMeshParts) > 0 {
			include := false
			for _, m := range bakeOptions.TargetMeshParts {
				if m == tri.MeshPart {
					include = true
					break
				}
			}
			if !include {
				continue
			}
		}

		ao := [3]float32{0, 0, 0}

		verts := tri.VertexIndices

		for _, other := range model.Mesh.Triangles {

			if tri == other {
				continue
			}

			span := math32.Max(tri.MaxSpan, other.MaxSpan) * 0.66

			if tri.Center.DistanceSquaredTo(other.Center) > span*span {
				continue
			}

			angle := tri.Normal.Angle(other.Normal)
			if angle < bakeOptions.OcclusionAngle {
				continue
			}

			if sharedA, sharedB, sharedC, count := tri.SharesVertexPositions(other); count > 0 {

				if sharedA >= 0 {
					ao[0] = 1
				}
				if sharedB >= 0 {
					ao[1] = 1
				}
				if sharedC >= 0 {
					ao[2] = 1
				}

			}

			if ao[0] == 1 && ao[1] == 1 && ao[2] == 1 {
				break
			}

		}

		for i := 0; i < 3; i++ {
			p := math32.Clamp(ao[i], 0, 1)
			model.Mesh.VertexColors[bakeOptions.TargetChannel][verts[i]].R += (bakeOptions.OcclusionColor.R - model.Mesh.VertexColors[bakeOptions.TargetChannel][verts[i]].R) * p
			model.Mesh.VertexColors[bakeOptions.TargetChannel][verts[i]].G += (bakeOptions.OcclusionColor.G - model.Mesh.VertexColors[bakeOptions.TargetChannel][verts[i]].G) * p
			model.Mesh.VertexColors[bakeOptions.TargetChannel][verts[i]].B += (bakeOptions.OcclusionColor.B - model.Mesh.VertexColors[bakeOptions.TargetChannel][verts[i]].B) * p
			model.Mesh.VertexColors[bakeOptions.TargetChannel][verts[i]].A += (bakeOptions.OcclusionColor.A - model.Mesh.VertexColors[bakeOptions.TargetChannel][verts[i]].A) * p
		}

	}

	// Inter-object AO next; this is kinda slow and janky, but it does work OK, I think

	if !bakeOptions.OtherModels.IsZero() {

		transform := model.Transform()

		distanceSquared := bakeOptions.InterModelDistance * bakeOptions.InterModelDistance

		bakeOptions.OtherModels.ForEach(func(node INode) bool {

			other := node.(*Model)

			rad := model.frustumCullingSphere.WorldRadius()
			if or := other.frustumCullingSphere.WorldRadius(); or > rad {
				rad = or
			}
			if model == other || model.WorldPosition().DistanceSquaredTo(other.WorldPosition()) > rad*rad {
				return true
			}

			otherTransform := other.Transform()

			for _, tri := range model.Mesh.Triangles {

				ao := [3]float32{0, 0, 0}

				verts := tri.VertexIndices

				transformedTriVerts := [3]Vector3{
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

					if transform.MultVec(tri.Center).DistanceSquaredTo(otherTransform.MultVec(otherTri.Center)) > span {
						continue
					}

					transformedOtherVerts := [3]Vector3{
						otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[0]]),
						otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[1]]),
						otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[2]]),
					}

					for i := 0; i < 3; i++ {
						for j := 0; j < 3; j++ {
							if transformedTriVerts[i].DistanceSquaredTo(transformedOtherVerts[j]) <= distanceSquared {
								ao[i] = 1
								break
							}
						}
					}

				}

				for i := 0; i < 3; i++ {
					color := model.Mesh.VertexColors[verts[i]][bakeOptions.TargetChannel]
					model.Mesh.VertexColors[verts[i]][bakeOptions.TargetChannel] = color.Mix(bakeOptions.OcclusionColor, ao[i])
				}

			}

			return true

		})

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

	for _, mp := range model.Mesh.MeshParts {

		if mp.Material != nil && mp.Material.Shadeless {

			mp.ForEachVertexIndex(func(vertIndex int) {
				model.Mesh.VertexColors[targetChannel][vertIndex].R = 1
				model.Mesh.VertexColors[targetChannel][vertIndex].G = 1
				model.Mesh.VertexColors[targetChannel][vertIndex].B = 1
			}, false)
			continue
		} else {

			mp.ForEachVertexIndex(func(vertIndex int) {
				model.Mesh.VertexColors[targetChannel][vertIndex].R = 0
				model.Mesh.VertexColors[targetChannel][vertIndex].G = 0
				model.Mesh.VertexColors[targetChannel][vertIndex].B = 0
			}, false)

			for _, light := range allLights {

				if light.IsOn() {
					light.Light(mp, model, model.Mesh.VertexColors[targetChannel], false)
				}

			}

		}

	}

}

// isTransparent returns true if the provided MeshPart has a Material with TransparencyModeTransparent, or if it's
// TransparencyModeAuto with the model or material alpha color being under 0.99. This is a helper function for sorting
// MeshParts into either transparent or opaque buckets for rendering.
// Note that this function doesn't work with transparent vertex colors.
func (model *Model) isTransparent(meshPart *MeshPart) bool {
	mat := meshPart.Material
	if mat != nil {
		matTransparent := mat.TransparencyMode == TransparencyModeTransparent || mat.Blend != ebiten.BlendSourceOver || (mat.TransparencyMode == TransparencyModeAuto && mat.Color.A < 0.999)
		modelTransparent := mat.TransparencyMode != TransparencyModeOpaque && model.Color.A < 0.999
		return matTransparent || modelTransparent
	}
	return model.Color.A < 0.999
}

////////

func (model *Model) setParent(parent INode) {

	prevScene := model.Scene()

	model.Node.setParent(parent)

	batched := model.AutoBatchMode != AutoBatchNone

	if !batched {

		for _, child := range model.SearchTree().INodes() {

			if child, ok := child.(*Model); ok && child.AutoBatchMode != AutoBatchNone {

				batched = true
				break

			}

		}

	} else {

		if prevScene != nil {
			prevScene.updateAutobatch = true
		}

		if newScene := model.Scene(); newScene != nil {
			newScene.updateAutobatch = true
		}

	}

}

// Type returns the NodeType for this object.
func (model *Model) Type() NodeType {
	return NodeTypeModel
}

func (model *Model) Sector() *Sector {
	if model.sector != nil {
		return model.sector
	}
	return model.Node.Sector()
}
