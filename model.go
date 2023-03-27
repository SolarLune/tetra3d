package tetra3d

import (
	"errors"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
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
	Mesh              *Mesh
	FrustumCulling    bool                                                 // Whether the Model is culled when it leaves the frustum.
	Color             *Color                                               // The overall multiplicative color of the Model.
	ColorBlendingFunc func(model *Model, meshPart *MeshPart) ebiten.ColorM // A user-customizeable blending function used to color the Model.
	BoundingSphere    *BoundingSphere

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
	VertexTransformFunction func(vertexPosition Vector, vertexIndex int) Vector

	// VertexClipFunction is a function that runs on the clipped result of each vertex position rendered with the material.
	// The function takes the vertex position along with the vertex index in the mesh.
	// This program runs after the vertex position is clipped to screen coordinates.
	// Note that the VertexClipFunction must return the vector passed.
	VertexClipFunction func(vertexPosition Vector, vertexIndex int) Vector

	// Automatic batching mode; when set and a Model changes parenting, it will be automatically batched as necessary according to
	// the AutoBatchMode set.
	AutoBatchMode  int
	autoBatched    bool
	dynamicBatcher bool

	sector *Sector // Sector is a reference to the Sector object that the Model stands in for, if sector-based rendering is enabled.
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

	model.Node.onTransformUpdate = model.onTransformUpdate

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

	newModel.Node = model.Node.Clone().(*Node)
	newModel.Node.onTransformUpdate = newModel.onTransformUpdate
	for _, child := range newModel.children {
		child.setParent(newModel)
	}

	if model.LightGroup != nil {
		newModel.LightGroup = model.LightGroup.Clone()
	}

	newModel.VertexClipFunction = model.VertexClipFunction
	newModel.VertexTransformFunction = model.VertexTransformFunction
	newModel.dynamicBatcher = model.dynamicBatcher

	if model.sector != nil {
		newModel.sector = model.sector.Clone()
		newModel.sector.Model = newModel
	}

	return newModel

}

// When updating a Model's transform, we have to also update its bounding sphere for frustum culling.
func (model *Model) onTransformUpdate() {

	transform := model.cachedTransform

	position, scale, rotation := transform.Decompose()

	// Skinned models have their positions at 0, 0, 0, and vertices offset according to wherever they were when exported.
	// To combat this, we save the original local positions of the mesh on export to position the bounding sphere in the
	// correct location.

	var center Vector

	// We do this because if a model is skinned and we've parented the model to the armature, then the center is
	// now from origin relative to the base of the armature on scene export. Otherwise, it's the center of the mesh.
	if !model.skinned {
		center = model.Mesh.Dimensions.Center()
	}

	center = rotation.MultVec(center)

	position.X += center.X * scale.X
	position.Y += center.Y * scale.Y
	position.Z += center.Z * scale.Z
	model.BoundingSphere.SetLocalPositionVec(position)

	dim := model.Mesh.Dimensions
	dim.Min.X *= scale.X
	dim.Min.Y *= scale.Y
	dim.Min.Z *= scale.Z

	dim.Max.X *= scale.X
	dim.Max.Y *= scale.Y
	dim.Max.Z *= scale.Z

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

/*
DynamicBatchAdd adds the provided models to the calling Model's dynamic batch, rendering with the specified meshpart (which should be part of the calling Model,
of course). Note that unlike StaticMerge(), DynamicBatchAdd works by simply rendering the batched models using the calling Model's first MeshPart's material. By
dynamically batching models together, this allows us to not flush between rendering multiple Models, saving a lot of render time, particularly if rendering many
low-poly, individual models that have very little variance (i.e. if they all share a single texture).
Calling this turns the model into a dynamic batching owner, meaning that it will no longer render its own mesh (for simplicity).
For more information, see this Wiki page on batching / merging: https://github.com/SolarLune/Tetra3d/wiki/Merging-and-Batching-Draw-Calls
*/
func (model *Model) DynamicBatchAdd(meshPart *MeshPart, batchedModels ...*Model) error {

	model.dynamicBatcher = true

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

	vec := NewVectorZero()

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

			})

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

	model.BoundingSphere.SetLocalPositionVec(model.Mesh.Dimensions.Center())
	model.BoundingSphere.Radius = model.Mesh.Dimensions.MaxSpan() / 2

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

func (model *Model) skinVertex(vertID int) (Vector, Vector) {

	// Avoid reallocating a new matrix for every vertex; that's wasteful
	model.skinMatrix.Clear()

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

	vertOut := model.skinMatrix.MultVecW(model.Mesh.VertexPositions[vertID])

	model.skinMatrix[3][0] = 0
	model.skinMatrix[3][1] = 0
	model.skinMatrix[3][2] = 0
	model.skinMatrix[3][3] = 1

	normal := model.skinMatrix.MultVecW(model.Mesh.VertexNormals[vertID])

	return vertOut, normal

}

var transformedVertexPositions = [3]Vector{
	{0, 0, 0, 0},
	{0, 0, 0, 0},
	{0, 0, 0, 0},
}

func (model *Model) refreshVertexVisibility() {
	for i := range model.Mesh.visibleVertices {
		model.Mesh.visibleVertices[i] = false
	}
}

// ProcessVertices processes the vertices a Model has in preparation for rendering, given a view-projection
// matrix, a camera, and the MeshPart being rendered.
func (model *Model) ProcessVertices(vpMatrix Matrix4, camera *Camera, meshPart *MeshPart, scene *Scene) []sortingTriangle {

	var transformFunc func(vertPos Vector, index int) Vector

	if model.VertexTransformFunction != nil {
		transformFunc = model.VertexTransformFunction
	}

	modelTransform := model.Transform()

	far := camera.far

	sortingTriIndex := 0

	mat := meshPart.Material
	mesh := meshPart.Mesh
	base := modelTransform

	sortMode := TriangleSortModeBackToFront

	if meshPart.Material != nil {
		sortMode = meshPart.Material.TriangleSortMode
	}

	camPos := camera.WorldPosition()
	invertedCamPos := modelTransform.Inverted().MultVec(camPos)

	if mat != nil && mat.BillboardMode != BillboardModeNone {

		lookat := NewLookAtMatrix(model.WorldPosition(), camPos, WorldUp)

		if mat.BillboardMode == BillboardModeXZ {
			lookat.SetRow(1, Vector{0, 1, 0, 0})
			x := lookat.Row(0)
			x.Y = 0
			lookat.SetRow(0, x.Unit())

			z := lookat.Row(2)
			z.Y = 0
			lookat.SetRow(2, z.Unit())
		}

		// This is the slowest part, for sure, but it's necessary to have a billboarded object still be accurate
		p, s, r := base.Decompose()
		base = r.Mult(lookat).Mult(NewMatrix4Scale(s.X, s.Y, s.Z))
		base.SetRow(3, Vector{p.X, p.Y, p.Z, 1})

	}

	mvp := base.Mult(vpMatrix)

	// Reslice to fill the capacity
	meshPart.sortingTriangles = meshPart.sortingTriangles[:cap(meshPart.sortingTriangles)]

	var mvJustRForNormals Matrix4
	if camera.RenderNormals {
		_, _, mvJustRForNormals = modelTransform.Mult(camera.ViewMatrix()).Decompose()
	}

	for ti := meshPart.TriangleStart; ti <= meshPart.TriangleEnd; ti++ {

		tri := mesh.Triangles[ti]

		meshPart.sortingTriangles[sortingTriIndex].Triangle = mesh.Triangles[ti]

		// if invertedCamPos.DistanceSquared(tri.Center) > pow(camera.far+tri.MaxSpan, 2) {
		// 	continue
		// }

		outOfBounds := true

		// If we're skinning a model, it will automatically copy the armature's position, scale, and rotation by copying its bones

		for i := 0; i < 3; i++ {

			if model.skinned {

				t := time.Now()

				vertPos, vertNormal := model.skinVertex(tri.VertexIndices[i])

				if transformFunc != nil {
					vertPos = transformFunc(vertPos, tri.VertexIndices[i])
				}
				mesh.vertexSkinnedNormals[tri.VertexIndices[i]] = vertNormal
				mesh.vertexSkinnedPositions[tri.VertexIndices[i]] = vertPos

				mesh.vertexTransforms[tri.VertexIndices[i]] = vpMatrix.MultVecW(vertPos)

				camera.DebugInfo.animationTime += time.Since(t)

			} else {

				v0 := mesh.VertexPositions[tri.VertexIndices[i]]

				if transformFunc != nil {
					v0 = transformFunc(v0, tri.VertexIndices[i])
				}

				mesh.vertexTransforms[tri.VertexIndices[i]] = mvp.MultVecW(v0)

			}

			if camera.VertexSnapping > 0 {
				mesh.vertexTransforms[tri.VertexIndices[i]] = mesh.vertexTransforms[tri.VertexIndices[i]].Snap(camera.VertexSnapping)
			}

			if camera.RenderNormals {
				mesh.vertexTransformedNormals[tri.VertexIndices[i]] = mvJustRForNormals.MultVecW(mesh.VertexNormals[tri.VertexIndices[i]])
			}

			w := mesh.vertexTransforms[tri.VertexIndices[i]].W

			// If the trangle is beyond the screen, we'll just pretend it's not and limit it to the closest possible value > 0
			// TODO: Replace this with triangle clipping or fix whatever graphical glitch seems to arise periodically
			if w < 0 {
				w = 0.000001
			}

			transformedVertexPositions[i].X = mesh.vertexTransforms[tri.VertexIndices[i]].X / w
			transformedVertexPositions[i].Y = mesh.vertexTransforms[tri.VertexIndices[i]].Y / w

			if mesh.vertexTransforms[tri.VertexIndices[i]].W >= 0 && mesh.vertexTransforms[tri.VertexIndices[i]].Z < far {
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

			// Old methods of backface culling; performant, but causes graphical glitches.

			// if backfaceCulling {

			// 	camera.backfacePool.Reset()
			// 	n0 := camera.backfacePool.Sub(p0, p1)
			// 	n1 := camera.backfacePool.Sub(p1, p2)
			// 	nor := camera.backfacePool.Cross(n0, n1)

			// 	if nor.Z > 0 {
			// 		continue
			// 	}

			// }
			// if model.skinned {
			// 	v0 := model.Mesh.vertexSkinnedNormals[tri.VertexIndices.X]
			// 	v1 := model.Mesh.vertexSkinnedNormals[tri.VertexIndices.Y]
			// 	v2 := model.Mesh.vertexSkinnedNormals[tri.VertexIndices.Z]

			// 	backfaceTriNormal.X = (v0.X + v1.X + v2.X) / 3
			// 	backfaceTriNormal.Y = (v0.Y + v1.Y + v2.Y) / 3
			// 	backfaceTriNormal.Z = (v0.Z + v1.Z + v2.Z) / 3

			// 	interimVec.X = camPos.X - model.Mesh.vertexSkinnedPositions[tri.VertexIndices.X].X
			// 	interimVec.Y = camPos.Y - model.Mesh.vertexSkinnedPositions[tri.VertexIndices.X].Y
			// 	interimVec.Z = camPos.Z - model.Mesh.vertexSkinnedPositions[tri.VertexIndices.X].Z

			// 	if dot(backfaceTriNormal, interimVec) < 0 {
			// 		continue
			// 	}

			// } else {

			// 	interimVec.X = invertedCamPos.X - tri.Center.X
			// 	interimVec.Y = invertedCamPos.Y - tri.Center.Y
			// 	interimVec.Z = invertedCamPos.Z - tri.Center.Z

			// 	if dot(tri.Normal, interimVec) < 0 {
			// 		continue
			// 	}

			// }

		}

		// If all transformed vertices are wholly out of bounds to the right, left, top, or bottom of the screen, then we can assume
		// the triangle does not need to be rendered
		if (transformedVertexPositions[0].X < -0.5 && transformedVertexPositions[1].X < -0.5 && transformedVertexPositions[2].X < -0.5) ||
			(transformedVertexPositions[0].X > 0.5 && transformedVertexPositions[1].X > 0.5 && transformedVertexPositions[2].X > 0.5) ||
			(transformedVertexPositions[0].Y < -0.5 && transformedVertexPositions[1].Y < -0.5 && transformedVertexPositions[2].Y < -0.5) ||
			(transformedVertexPositions[0].Y > 0.5 && transformedVertexPositions[1].Y > 0.5 && transformedVertexPositions[2].Y > 0.5) {
			continue
		}

		// Previously, depth was compared using the lowest W value of all vertices in the triangle; after that, I tried
		// averaging them out. Neither of these was completely satisfactory, and in addition, there was no depth sorting
		// for orthographic triangles that overlapped using this method.

		// I could substitute Z for W, but sorting by distance to the triangle center directly gives a better result overall, it seems.
		if sortMode != TriangleSortModeNone {
			meshPart.sortingTriangles[sortingTriIndex].depth = float32(invertedCamPos.DistanceSquared(tri.Center))
		}

		mesh.visibleVertices[tri.VertexIndices[0]] = true
		mesh.visibleVertices[tri.VertexIndices[1]] = true
		mesh.visibleVertices[tri.VertexIndices[2]] = true

		sortingTriIndex++

	}

	meshPart.sortingTriangles = meshPart.sortingTriangles[:sortingTriIndex]

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

			if tri == other || tri.Normal.Equals(other.Normal) {
				continue
			}

			span := tri.MaxSpan
			if other.MaxSpan > span {
				span = other.MaxSpan
			}

			span *= 0.66

			if tri.Center.DistanceSquared(other.Center) > span*span {
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
		if model == other || model.WorldPosition().DistanceSquared(other.WorldPosition()) > rad*rad {
			continue
		}

		otherTransform := other.Transform()

		for _, tri := range model.Mesh.Triangles {

			ao := [3]float32{0, 0, 0}

			verts := tri.VertexIndices

			transformedTriVerts := [3]Vector{
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

				if transform.MultVec(tri.Center).DistanceSquared(otherTransform.MultVec(otherTri.Center)) > span {
					continue
				}

				transformedOtherVerts := [3]Vector{
					otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[0]]),
					otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[1]]),
					otherTransform.MultVec(other.Mesh.VertexPositions[otherVerts[2]]),
				}

				for i := 0; i < 3; i++ {
					for j := 0; j < 3; j++ {
						if transformedTriVerts[i].DistanceSquared(transformedOtherVerts[j]) <= distanceSquared {
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
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.Node.
	model.addChildren(model, children...)
}

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

	}

	if batched {

		if prevScene != nil {
			prevScene.updateAutobatch = true
		}

		if newScene := model.Scene(); newScene != nil {
			newScene.updateAutobatch = true
		}

	}

}

// Unparent unparents the Model from its parent, removing it from the scenegraph.
func (model *Model) Unparent() {
	if model.parent != nil {
		model.parent.RemoveChildren(model)
	}
	model.autoBatched = false
}

// Type returns the NodeType for this object.
func (model *Model) Type() NodeType {
	return NodeTypeModel
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (model *Model) Index() int {
	if model.parent != nil {
		for i, c := range model.parent.Children() {
			if c == model {
				return i
			}
		}
	}
	return -1
}

func (model *Model) Sector() *Sector {
	if model.sector != nil {
		return model.sector
	}
	return model.Node.Sector()
}
