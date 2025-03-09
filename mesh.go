package tetra3d

import (
	"errors"
	"fmt"
	"log"
	"math"

	"github.com/solarlune/tetra3d/math32"
)

// Dimensions represents the minimum and maximum spatial dimensions of a Mesh arranged in a 2-space Vector slice.
type Dimensions struct {
	Min, Max Vector3
}

func NewEmptyDimensions() Dimensions {
	return Dimensions{
		Vector3{float32(math.MaxFloat32), float32(math.MaxFloat32), float32(math.MaxFloat32)},
		Vector3{-float32(math.MaxFloat32), -float32(math.MaxFloat32), -float32(math.MaxFloat32)},
	}
}

// MaxDimension returns the maximum value from all of the axes in the Dimensions. For example, if the Dimensions have a min of [-1, -2, -2],
// and a max of [6, 1.5, 1], Max() will return 7 for the X axis, as it's the largest distance between all axes.
func (dim Dimensions) MaxDimension() float32 {
	return math32.Max(math32.Max(dim.Width(), dim.Height()), dim.Depth())
}

// MaxSpan returns the maximum span between the corners of the dimension set.
func (dim Dimensions) MaxSpan() float32 {
	return dim.Max.Sub(dim.Min).Magnitude()
}

// Center returns the center point inbetween the two corners of the dimension set.
func (dim Dimensions) Center() Vector3 {
	return Vector3{
		(dim.Max.X + dim.Min.X) / 2,
		(dim.Max.Y + dim.Min.Y) / 2,
		(dim.Max.Z + dim.Min.Z) / 2,
	}
}

// Width returns the total difference between the minimum and maximum X values.
func (dim Dimensions) Width() float32 {
	return dim.Max.X - dim.Min.X
}

// Height returns the total difference between the minimum and maximum Y values.
func (dim Dimensions) Height() float32 {
	return dim.Max.Y - dim.Min.Y
}

// Depth returns the total difference between the minimum and maximum Z values.
func (dim Dimensions) Depth() float32 {
	return dim.Max.Z - dim.Min.Z
}

// Clamp limits the provided position vector to be within the dimensions set.
func (dim Dimensions) Clamp(position Vector3) Vector3 {

	if position.X < dim.Min.X {
		position.X = dim.Min.X
	} else if position.X > dim.Max.X {
		position.X = dim.Max.X
	}

	if position.Y < dim.Min.Y {
		position.Y = dim.Min.Y
	} else if position.Y > dim.Max.Y {
		position.Y = dim.Max.Y
	}

	if position.Z < dim.Min.Z {
		position.Z = dim.Min.Z
	} else if position.Z > dim.Max.Z {
		position.Z = dim.Max.Z
	}

	return position
}

// Inside returns if a position is inside a set of dimensions.
func (dim Dimensions) Inside(position Vector3) bool {

	if position.X < dim.Min.X ||
		position.X > dim.Max.X ||
		position.Y < dim.Min.Y ||
		position.Y > dim.Max.Y ||
		position.Z < dim.Min.Z ||
		position.Z > dim.Max.Z {
		return false
	}

	return true
}

func (dim Dimensions) Size() Vector3 {
	return Vector3{dim.Width(), dim.Height(), dim.Depth()}
}

// func (dim Dimensions) reform() {

// 	if dim.Min.X > dim.Max.X {
// 		swap := dim.Mi
// 	}

// 	for i := 0; i < 3; i++ {

// 		if dim.Min[i] > dim.Max[i] {
// 			swap := dim.Max[i]
// 			dim.Max[i] = dim.Min[i]
// 			dim.Min[i] = swap
// 		}

// 	}

// }

// NewDimensionsFromPoints creates a new Dimensions struct from the given series of positions.
func NewDimensionsFromPoints(points ...Vector3) Dimensions {

	if len(points) == 0 {
		panic("error: no points passed to NewDimensionsFromPoints()")
	}

	dim := NewEmptyDimensions()

	for _, point := range points {

		if dim.Min.X > point.X {
			dim.Min.X = point.X
		}

		if dim.Min.Y > point.Y {
			dim.Min.Y = point.Y
		}

		if dim.Min.Z > point.Z {
			dim.Min.Z = point.Z
		}

		if dim.Max.X < point.X {
			dim.Max.X = point.X
		}

		if dim.Max.Y < point.Y {
			dim.Max.Y = point.Y
		}

		if dim.Max.Z < point.Z {
			dim.Max.Z = point.Z
		}

	}

	return dim
}

type MeshUniqueType int

const (
	MeshUniqueFalse MeshUniqueType = iota
	MeshUniqueMesh
	MeshUniqueMeshAndMaterials
)

// VertexColorChannel represents the colors of vertices in a channel.
// Each value in the color channel is a color for the vertex in the associated index.
type VertexColorChannel []Color

// Mesh represents a mesh that can be represented visually in different locations via Models. By default, a new Mesh has no MeshParts (so you would need to add one
// manually if you want to construct a Mesh via code).
type Mesh struct {
	Name    string   // The name of the Mesh resource
	library *Library // A reference to the Library this Mesh came from.

	MeshParts []*MeshPart // The various mesh parts (collections of triangles, rendered with a single material).
	Triangles []*Triangle // The various triangles composing the Mesh.
	triIndex  int

	// Vertices are stored as a struct-of-arrays for simplified and faster rendering.
	// Each vertex property (position, normal, UV, colors, weights, bones, etc) is stored
	// here and indexed in order of vertex index.
	vertexTransforms         []Vector4
	VertexPositions          []Vector3
	VertexNormals            []Vector3
	vertexSkinnedNormals     []Vector3
	vertexSkinnedPositions   []Vector3
	vertexTransformedNormals []Vector3
	VertexUVs                []Vector2            // The UV values for each vertex
	VertexUVOriginalValues   []Vector2            // The original UV values for each vertex
	VertexColors             []VertexColorChannel // A slice of channels for the mesh
	VertexGroupNames         []string             // The names of the vertex groups applies to the Mesh; this is only populated if the Mesh is affected by an armature
	VertexWeights            [][]float32          // TODO: Replace this with [][8]float32 (or however many the maximum is for GLTF)
	VertexBones              [][]uint16           // TODO: Replace this with [][8]uint16 (or however many the maximum number of bones affecting a single vertex is for GLTF)
	visibleVertices          []bool
	maxTriangleSpan          float32
	VertexActiveColorChannel int // VertexActiveColorChannel is the active vertex color used for coloring the mesh

	vertexLights  []Color
	vertsAddStart int
	vertsAddEnd   int

	VertexColorChannelNames map[string]int // VertexColorChannelNames is a map allowing you to get the index of a mesh's vertex color channel by its name.
	Dimensions              Dimensions
	properties              Properties

	// If Unique is set to a value other than MeshUniqueNone, whenever a Mesh is used for a Model and the Model is cloned,
	// the Mesh or Mesh and Materials are cloned with it. Useful for things like sprites.
	Unique MeshUniqueType
}

// NewMesh takes a name and a slice of *Vertex instances, and returns a new Mesh. If you provide *Vertex instances, the number must be divisible by 3,
// or NewMesh will panic.
func NewMesh(name string, verts ...VertexInfo) *Mesh {

	mesh := &Mesh{
		Name:                    name,
		MeshParts:               []*MeshPart{},
		Dimensions:              Dimensions{Vector3{0, 0, 0}, Vector3{0, 0, 0}},
		VertexColorChannelNames: map[string]int{},
		properties:              NewProperties(),

		vertexTransforms:         []Vector4{},
		VertexPositions:          []Vector3{},
		visibleVertices:          []bool{},
		VertexNormals:            []Vector3{},
		vertexSkinnedNormals:     []Vector3{},
		vertexSkinnedPositions:   []Vector3{},
		vertexTransformedNormals: []Vector3{},
		vertexLights:             []Color{},
		VertexUVs:                []Vector2{},
		VertexColors:             []VertexColorChannel{},
		VertexBones:              [][]uint16{},
		VertexWeights:            [][]float32{},
		VertexActiveColorChannel: -1,
	}

	if len(verts) > 0 {
		mesh.AddVertices(verts...)
	}

	return mesh

}

// Clone clones the Mesh, creating a new Mesh that has cloned MeshParts.
func (mesh *Mesh) Clone() *Mesh {
	newMesh := NewMesh(mesh.Name)
	newMesh.library = mesh.library
	newMesh.properties = mesh.properties.Clone()
	newMesh.triIndex = mesh.triIndex

	newMesh.allocateVertexBuffers(len(mesh.VertexPositions))

	// TODO: Replace these with copy() calls?

	for i := range mesh.VertexPositions {
		newMesh.VertexPositions = append(newMesh.VertexPositions, mesh.VertexPositions[i])
		newMesh.visibleVertices = append(newMesh.visibleVertices, false)
	}

	for i := range mesh.VertexNormals {
		newMesh.VertexNormals = append(newMesh.VertexNormals, mesh.VertexNormals[i])
	}

	for i := range mesh.vertexLights {
		newMesh.vertexLights = append(newMesh.vertexLights, mesh.vertexLights[i])
	}

	for i := range mesh.VertexUVs {
		newMesh.VertexUVs = append(newMesh.VertexUVs, mesh.VertexUVs[i])
		newMesh.VertexUVOriginalValues = append(newMesh.VertexUVOriginalValues, mesh.VertexUVs[i])
	}

	newMesh.VertexColors = append(newMesh.VertexColors, make(VertexColorChannel, len(mesh.VertexColors)))
	for channelIndex, channel := range mesh.VertexColors {
		for vertIndex := range channel {
			channel = append(channel, mesh.VertexColors[channelIndex][vertIndex])
		}
	}

	newMesh.VertexActiveColorChannel = mesh.VertexActiveColorChannel

	for c := range mesh.VertexBones {
		newMesh.VertexBones = append(newMesh.VertexBones, []uint16{})
		for v := range mesh.VertexBones[c] {
			newMesh.VertexBones[c] = append(newMesh.VertexBones[c], mesh.VertexBones[c][v])
		}
	}

	for c := range mesh.VertexWeights {
		newMesh.VertexWeights = append(newMesh.VertexWeights, []float32{})
		for v := range mesh.VertexWeights[c] {
			newMesh.VertexWeights[c] = append(newMesh.VertexWeights[c], mesh.VertexWeights[c][v])
		}
	}

	for v := range mesh.vertexTransforms {
		newMesh.vertexTransforms = append(newMesh.vertexTransforms, mesh.vertexTransforms[v])
	}

	for v := range mesh.vertexSkinnedNormals {
		newMesh.vertexSkinnedNormals = append(newMesh.vertexSkinnedNormals, mesh.vertexSkinnedNormals[v])
	}

	for v := range mesh.vertexTransformedNormals {
		newMesh.vertexTransformedNormals = append(newMesh.vertexTransformedNormals, mesh.vertexTransformedNormals[v])
	}

	for v := range mesh.vertexSkinnedPositions {
		newMesh.vertexSkinnedPositions = append(newMesh.vertexSkinnedPositions, mesh.vertexSkinnedPositions[v])
	}

	newMesh.Triangles = make([]*Triangle, 0, len(mesh.Triangles))

	for _, part := range mesh.MeshParts {
		newPart := part.Clone()

		newPart.ForEachTri(
			func(tri *Triangle) {
				newTri := tri.Clone()
				newTri.MeshPart = newPart
				newMesh.Triangles = append(newMesh.Triangles, newTri)
			},
		)

		newPart.AssignToMesh(newMesh)
	}

	newMesh.vertsAddEnd = mesh.vertsAddEnd
	newMesh.vertsAddStart = mesh.vertsAddStart

	for channelName, index := range mesh.VertexColorChannelNames {
		newMesh.VertexColorChannelNames[channelName] = index
	}

	newMesh.Dimensions = mesh.Dimensions

	newMesh.Unique = mesh.Unique

	if newMesh.Unique == MeshUniqueMeshAndMaterials {
		for _, meshPart := range newMesh.MeshParts {
			meshPart.Material = meshPart.Material.Clone()
		}
	}

	newMesh.maxTriangleSpan = mesh.maxTriangleSpan

	newMesh.VertexGroupNames = append(newMesh.VertexGroupNames, mesh.VertexGroupNames...)

	return newMesh
}

// allocateVertexBuffers allows us to allocate the slices for vertex properties all at once rather than resizing multiple times as
// we append to a slice and have its backing buffer automatically expanded (which is slower).
func (mesh *Mesh) allocateVertexBuffers(vertexCount int) {

	if cap(mesh.VertexPositions) >= vertexCount {
		return
	}

	mesh.VertexPositions = append(make([]Vector3, 0, vertexCount), mesh.VertexPositions...)
	if len(mesh.visibleVertices) < vertexCount {
		mesh.visibleVertices = make([]bool, vertexCount)
	}

	mesh.VertexNormals = append(make([]Vector3, 0, vertexCount), mesh.VertexNormals...)

	mesh.vertexLights = append(make([]Color, 0, vertexCount), mesh.vertexLights...)

	mesh.VertexUVs = append(make([]Vector2, 0, vertexCount), mesh.VertexUVs...)
	mesh.VertexUVOriginalValues = append(make([]Vector2, 0, vertexCount), mesh.VertexUVs...)

	for ci := range mesh.VertexColors {
		mesh.VertexColors[ci] = append(make(VertexColorChannel, 0, vertexCount), mesh.VertexColors[ci]...)
	}

	mesh.VertexBones = append(make([][]uint16, 0, vertexCount), mesh.VertexBones...)

	mesh.VertexWeights = append(make([][]float32, 0, vertexCount), mesh.VertexWeights...)

	mesh.vertexTransforms = append(make([]Vector4, 0, vertexCount), mesh.vertexTransforms...)

	mesh.vertexSkinnedNormals = append(make([]Vector3, 0, vertexCount), mesh.vertexSkinnedNormals...)

	mesh.vertexTransformedNormals = append(make([]Vector3, 0, vertexCount), mesh.vertexTransformedNormals...)

	mesh.vertexSkinnedPositions = append(make([]Vector3, 0, vertexCount), mesh.vertexSkinnedPositions...)

}

func (mesh *Mesh) ensureEnoughVertexColorChannels(channelIndex int) {

	for len(mesh.VertexColors) <= channelIndex+1 {
		mesh.VertexColors = append(mesh.VertexColors, VertexColorChannel{})
	}

	for ci := range mesh.VertexColors {
		for len(mesh.VertexColors[ci]) < len(mesh.VertexPositions) {
			mesh.VertexColors[ci] = append(mesh.VertexColors[ci], Color{1, 1, 1, 1})
		}
	}

}

// CombineVertexColors allows you to combine vertex color channels together. The targetChannel is the channel that will hold
// the result, and multiplicative controls whether the combination is multiplicative (true) or additive (false). The sourceChannels
// ...int is the vertex color channel indices to combine together.
// If the channel indices provided in the sourceChannels ...int are too high for the number of channels on each vertex in the mesh,
// then those indices will be skipped.
func (mesh *Mesh) CombineVertexColors(targetChannel int, multiplicative bool, sourceChannels ...int) {

	mesh.ensureEnoughVertexColorChannels(targetChannel)

	base := NewColor(1, 1, 1, 1)

	for vertexIndex := 0; vertexIndex < len(mesh.VertexColors[targetChannel]); vertexIndex++ {

		base.R = 0
		base.G = 0
		base.B = 0

		if multiplicative {
			base.R = 1
			base.G = 1
			base.B = 1
		}

		for _, channelIndex := range sourceChannels {

			if multiplicative {
				base.R *= mesh.VertexColors[channelIndex][vertexIndex].R
				base.G *= mesh.VertexColors[channelIndex][vertexIndex].G
				base.B *= mesh.VertexColors[channelIndex][vertexIndex].B
			} else {
				base.R += mesh.VertexColors[channelIndex][vertexIndex].R
				base.G += mesh.VertexColors[channelIndex][vertexIndex].G
				base.B += mesh.VertexColors[channelIndex][vertexIndex].B
			}

		}

		mesh.VertexColors[targetChannel][vertexIndex] = base

	}

}

// SetVertexColor sets the specified vertex color for all vertices in the mesh for the target color channel.
func (mesh *Mesh) SetVertexColor(targetChannel int, color Color) {
	NewVertexSelection().SelectMeshes(mesh).SetColor(targetChannel, color)
}

// SetActiveColorChannel sets the active color channel for all vertices in the mesh to the specified channel index.
func (mesh *Mesh) SetActiveColorChannel(targetChannel int) {
	NewVertexSelection().SelectMeshes(mesh).SetActiveColorChannel(targetChannel)
}

// Select allows you to easily select the vertices associated with the Mesh.
// Select is just syntactic sugar for tetra3d.NewVertexSelection().SelectMeshes(mesh).
func (mesh *Mesh) Select() VertexSelection {
	return NewVertexSelection().SelectMeshes(mesh)
}

// Materials returns a slice of the materials present in the Mesh's MeshParts.
func (mesh *Mesh) Materials() []*Material {
	mats := []*Material{}
	for _, mp := range mesh.MeshParts {
		if mp.Material != nil {
			mats = append(mats, mp.Material)
		}
	}
	return mats
}

// AddMeshPart allows you to add a new MeshPart to the Mesh with the given Material (with a nil Material reference also being valid).
func (mesh *Mesh) AddMeshPart(material *Material, indices ...int) *MeshPart {
	mp := NewMeshPart(mesh, material)
	mesh.MeshParts = append(mesh.MeshParts, mp)
	if len(indices) > 0 {
		mp.AddTriangles(indices...)
	}
	return mp
}

// FindMeshPart allows you to retrieve a MeshPart by its material's name. If no material with the provided name is given, the function returns nil.
func (mesh *Mesh) FindMeshPart(materialName string) *MeshPart {
	for _, mp := range mesh.MeshParts {
		if mp.Material != nil && mp.Material.Name == materialName {
			return mp
		}
	}
	return nil
}

func (mesh *Mesh) AddVertices(verts ...VertexInfo) {

	mesh.vertsAddStart = len(mesh.VertexPositions)
	mesh.vertsAddEnd = mesh.vertsAddStart + len(verts)

	if len(verts) == 0 {
		panic("Error: Mesh.AddVertices() given 0 vertices.")
	}

	mesh.allocateVertexBuffers(len(mesh.VertexPositions) + len(verts))

	for i := 0; i < len(verts); i++ {

		vertInfo := verts[i]
		mesh.VertexPositions = append(mesh.VertexPositions, Vector3{vertInfo.X, vertInfo.Y, vertInfo.Z})
		mesh.VertexNormals = append(mesh.VertexNormals, Vector3{vertInfo.NormalX, vertInfo.NormalY, vertInfo.NormalZ})
		mesh.VertexUVs = append(mesh.VertexUVs, Vector2{vertInfo.U, vertInfo.V})
		mesh.VertexUVOriginalValues = append(mesh.VertexUVOriginalValues, Vector2{vertInfo.U, vertInfo.V})

		mesh.ensureEnoughVertexColorChannels(len(vertInfo.Colors) - 1)

		for channelIndex := 0; channelIndex < len(vertInfo.Colors); channelIndex++ {
			mesh.VertexColors[channelIndex][mesh.vertsAddStart+i] = vertInfo.Colors[channelIndex]
		}

		mesh.VertexBones = append(mesh.VertexBones, vertInfo.Bones)
		mesh.VertexWeights = append(mesh.VertexWeights, vertInfo.Weights)

		mesh.vertexLights = append(mesh.vertexLights, NewColor(0, 0, 0, 1))
		mesh.vertexTransforms = append(mesh.vertexTransforms, Vector4{}) // x, y, z, w
		mesh.vertexSkinnedNormals = append(mesh.vertexSkinnedNormals, Vector3{})
		mesh.vertexTransformedNormals = append(mesh.vertexTransformedNormals, Vector3{})
		mesh.vertexSkinnedPositions = append(mesh.vertexSkinnedPositions, Vector3{})

	}

}

// Library returns the Library from which this Mesh was loaded. If it was created through code, this function will return nil.
func (mesh *Mesh) Library() *Library {
	return mesh.library
}

// UpdateBounds updates the mesh's dimensions; call this after manually changing vertex positions.
func (mesh *Mesh) UpdateBounds() {

	mesh.Dimensions = NewEmptyDimensions()

	for _, position := range mesh.VertexPositions {

		if mesh.Dimensions.Min.X > position.X {
			mesh.Dimensions.Min.X = position.X
		}

		if mesh.Dimensions.Min.Y > position.Y {
			mesh.Dimensions.Min.Y = position.Y
		}

		if mesh.Dimensions.Min.Z > position.Z {
			mesh.Dimensions.Min.Z = position.Z
		}

		if mesh.Dimensions.Max.X < position.X {
			mesh.Dimensions.Max.X = position.X
		}

		if mesh.Dimensions.Max.Y < position.Y {
			mesh.Dimensions.Max.Y = position.Y
		}

		if mesh.Dimensions.Max.Z < position.Z {
			mesh.Dimensions.Max.Z = position.Z
		}

	}

}

// AutoNormal automatically recalculates the normals for the triangles contained within the Mesh and sets the vertex normals for
// all triangles to the triangles' surface normal.
func (mesh *Mesh) AutoNormal() {

	for _, tri := range mesh.Triangles {
		tri.RecalculateNormal()
		for i := 0; i < 3; i++ {
			mesh.VertexNormals[tri.VertexIndices[i]] = tri.Normal
		}
	}

}

// SelectVertices generates a new vertex selection for the current Mesh.
// This selection should generally be retained to operate on sequentially.
// func (mesh *Mesh) SelectVertices() VertexSelection {
// 	return VertexSelection{Indices: newSet[int](), Mesh: mesh}
// }

// Properties returns this Mesh object's game Properties struct.
func (mesh *Mesh) Properties() Properties {
	return mesh.properties
}

// VertexSelectionSet represents a selection set of indices for a given Mesh.
type VertexSelectionSet struct {
	Indices   Set[int]
	SelectAll bool
}

// VertexSelection represents a selection of vertices on a Mesh.
type VertexSelection struct {
	SelectionSet map[*Mesh]*VertexSelectionSet
}

// NewVertexSelection selects all
func NewVertexSelection() VertexSelection {
	return VertexSelection{
		SelectionSet: map[*Mesh]*VertexSelectionSet{},
	}
}

const ErrorVertexChannelOutsideRange = "error: vertex color channel not found by given name"

func (vs VertexSelection) ensureSelectionSetExists(mesh *Mesh) {
	if _, ok := vs.SelectionSet[mesh]; !ok {
		vs.SelectionSet[mesh] = &VertexSelectionSet{
			Indices: newSet[int](),
		}
	}
}

// SelectInVertexColorChannel selects all vertices in the Mesh that have a non-pure black color in the vertex color channel
// with the specified index. If the index is over the number of vertex colors currently on the Mesh, then the function
// will not alter the VertexSelection and will return an error.
func (vs VertexSelection) SelectInVertexColorChannel(mesh *Mesh, channelNames ...string) (VertexSelection, error) {

	vs.ensureSelectionSetExists(mesh)

	var err error

	for _, groupName := range channelNames {

		if channelIndex, ok := mesh.VertexColorChannelNames[groupName]; ok {

			for vertexIndex := range mesh.VertexColors[channelIndex] {

				color := mesh.VertexColors[channelIndex][vertexIndex]

				if color.R > 0.01 || color.G > 0.01 || color.B > 0.01 {
					vs.SelectionSet[mesh].Indices.Add(vertexIndex)
				}

			}

		} else {
			err = errors.New(ErrorVertexChannelOutsideRange)
			continue
		}

	}

	return vs, err

}

const ErrorVertexGroupNotFound = "error: vertex channel name not found"

// SelectInVertexGroup selects all vertices in the Mesh that are assigned to the specifiefd vertex groups.
// If any of the vertex groups are not found, the function will return an error.
func (vs VertexSelection) SelectInVertexGroup(mesh *Mesh, vertexGroupNames ...string) (VertexSelection, error) {

	vs.ensureSelectionSetExists(mesh)

	var err error

	for _, groupName := range vertexGroupNames {

		vertexGroupIndex := -1

		for i, g := range mesh.VertexGroupNames {
			if g == groupName {
				vertexGroupIndex = i
			}
		}

		if vertexGroupIndex < 0 {
			// return vs, errors.New(ErrorVertexGroupNotFound)
			err = errors.New(ErrorVertexGroupNotFound)
			continue
		}

		for vertexIndex := range mesh.VertexBones {

			boneSet := mesh.VertexBones[vertexIndex]
			for _, b := range boneSet {
				if b == uint16(vertexGroupIndex) {
					vs.SelectionSet[mesh].Indices.Add(vertexIndex)
				}
			}

		}

	}

	return vs, err

}

// SelectMeshes selects all vertices on the target meshes.
func (vs VertexSelection) SelectMeshes(meshes ...*Mesh) VertexSelection {

	for _, mesh := range meshes {

		vs.ensureSelectionSetExists(mesh)

		vs.SelectionSet[mesh].Indices.Clear()
		vs.SelectionSet[mesh].SelectAll = true

	}

	return vs
}

func (vs VertexSelection) Clear() VertexSelection {
	for m := range vs.SelectionSet {
		delete(vs.SelectionSet, m)
	}
	return vs
}

func (vs VertexSelection) IsEmpty() bool {
	return len(vs.SelectionSet) == 0
}

// SelectMeshPart selects all vertices in the Mesh belonging to any of the specified MeshParts.
func (vs VertexSelection) SelectMeshPart(meshParts ...*MeshPart) VertexSelection {

	for _, meshPart := range meshParts {

		meshPart.ForEachTri(

			func(tri *Triangle) {

				vs.ensureSelectionSetExists(meshPart.Mesh)

				for _, index := range tri.VertexIndices {
					vs.SelectionSet[meshPart.Mesh].Indices.Add(index)
				}

			},
		)

	}

	return vs

}

// SelectMeshPartByIndex selects all vertices in the Mesh belonging to the specified MeshPart by
// index.
// If the MeshPart doesn't exist, this function will panic.
func (vs VertexSelection) SelectMeshPartByIndex(mesh *Mesh, indexNumber int) VertexSelection {
	vs.ensureSelectionSetExists(mesh)
	vs.SelectMeshPart(mesh.MeshParts[indexNumber])
	return vs

}

// SelectMeshPartByName selects all vertices in the Mesh belonging to materials with the specified
// name.
func (vs VertexSelection) SelectMeshPartByName(mesh *Mesh, materialNames ...string) VertexSelection {

	vs.ensureSelectionSetExists(mesh)
	for _, matName := range materialNames {
		if mp := mesh.FindMeshPart(matName); mp != nil {
			vs.SelectMeshPart(mp)
		}
	}

	return vs

}

// SelectIndices selects the passed vertex indices in the Mesh.
// This is syntactic sugar for VertexSelection.Indices.Add(indices...)
func (vs VertexSelection) SelectIndices(mesh *Mesh, indices ...int) VertexSelection {
	vs.ensureSelectionSetExists(mesh)
	for _, i := range indices {
		vs.SelectionSet[mesh].Indices.Add(i)
	}
	return vs
}

// SelectTriangles selects the vertex indices composing the triangles passed.
func (vs VertexSelection) SelectTriangles(mesh *Mesh, triangles ...*Triangle) VertexSelection {
	vs.ensureSelectionSetExists(mesh)
	for _, t := range triangles {
		for _, i := range t.VertexIndices {
			vs.SelectionSet[mesh].Indices.Add(i)
		}
	}
	return vs
}

// SelectTriangles selects vertices that share positions with already-selected vertices.
func (vs VertexSelection) SelectSharedVertices() VertexSelection {

	for mesh, set := range vs.SelectionSet {

		for index := range set.Indices {

			for i, vp := range mesh.VertexPositions {

				if index == i {
					continue
				}

				if vp.Equals(mesh.VertexPositions[index]) {
					set.Indices.Add(i)
				}

			}

		}

	}

	return vs

}

// SetColor sets the color of the specified channel in all vertices contained within the VertexSelection to the provided Color.
// If the channelIndex provided is greater than the number of channels in the Mesh minus one, vertex color channels will be created for all vertices
// up to the index provided (e.g. VertexSelection.SetColor(2, colors.White()) will make it so that the mesh has at least three color channels - 0, 1, and 2).
func (vs VertexSelection) SetColor(channelIndex int, color Color) {

	for mesh := range vs.SelectionSet {
		mesh.ensureEnoughVertexColorChannels(channelIndex)
	}

	vs.ForEachIndex(func(mesh *Mesh, index int) {
		mesh.VertexColors[channelIndex][index].R = color.R
		mesh.VertexColors[channelIndex][index].G = color.G
		mesh.VertexColors[channelIndex][index].B = color.B
		mesh.VertexColors[channelIndex][index].A = color.A
	})

}

// SetNormal sets the normal of all vertices contained within the VertexSelection to the provided normal vector.
func (vs VertexSelection) SetNormal(normal Vector3) {

	vs.ForEachIndex(func(mesh *Mesh, index int) {
		mesh.VertexNormals[index].X = normal.X
		mesh.VertexNormals[index].Y = normal.Y
		mesh.VertexNormals[index].Z = normal.Z
	})

}

// MoveUVs moves the UV values by the values specified.
func (vs VertexSelection) MoveUVs(dx, dy float32) {

	vs.ForEachIndex(func(mesh *Mesh, index int) {
		mesh.VertexUVs[index].X += dx
		mesh.VertexUVs[index].Y += dy
	})

}

// ScaleUVs scales the UV values by the percentages specified.
func (vs VertexSelection) ScaleUVs(px, py float32) {

	vs.ForEachIndex(func(mesh *Mesh, index int) {
		mesh.VertexUVs[index].X *= px
		mesh.VertexUVs[index].Y *= py
	})

}

// MoveUVsVec moves the UV values by the Vector values specified.
func (vs *VertexSelection) MoveUVsVec(vec Vector3) {
	vs.MoveUVs(vec.X, vec.Y)
}

// SetUVOffset moves all UV values for vertices selected to be offset by the values specified, with [0, 0] being their original locations.
// Note that for this to work, you would need to store and work with the same vertex selection over multiple frames.
func (vs VertexSelection) SetUVOffset(x, y float32) {

	vs.ForEachIndex(func(mesh *Mesh, index int) {

		mesh.VertexUVs[index].X = x + mesh.VertexUVOriginalValues[index].X
		mesh.VertexUVs[index].Y = y + mesh.VertexUVOriginalValues[index].Y

	})

}

// RotateUVs rotates the UV values around the center of the UV values for the mesh in radians
func (vs VertexSelection) RotateUVs(rotation float32) {
	center := Vector2{}

	vs.ForEachIndex(func(mesh *Mesh, index int) {
		center = center.Add(mesh.VertexUVs[index])
	})

	center = center.Divide(float32(vs.Count()))

	vs.ForEachIndex(func(mesh *Mesh, index int) {
		diff := mesh.VertexUVs[index].Sub(center)
		mesh.VertexUVs[index] = center.Add(diff.Rotate(rotation))
	})

}

// SetActiveColorChannel sets the active color channel in all vertices contained within the VertexSelection to the channel with the
// specified index.
// If the channelIndex provided is greater than the number of vertex color channels in the Mesh minus one, vertex color channels will be created for all vertices
// up to the index provided (e.g. VertexSelection.SetActiveColorChannel(3) will make it so that the mesh has at least four color channels - 0, 1, 2, and 3).
func (vs VertexSelection) SetActiveColorChannel(channelIndex int) {

	for mesh := range vs.SelectionSet {
		mesh.ensureEnoughVertexColorChannels(channelIndex)
		mesh.VertexActiveColorChannel = channelIndex
	}

}

// ApplyMatrix applies a Matrix4 to the position of all vertices contained within the VertexSelection.
func (vs VertexSelection) ApplyMatrix(matrix Matrix4) {

	vs.ForEachIndex(func(mesh *Mesh, index int) {
		mesh.VertexPositions[index] = matrix.MultVec(mesh.VertexPositions[index])
	})

}

// Move moves all vertices contained within the VertexSelection by the provided x, y, and z values.
func (vs VertexSelection) Move(x, y, z float32) {

	vs.ForEachIndex(func(mesh *Mesh, index int) {

		mesh.VertexPositions[index].X += x
		mesh.VertexPositions[index].Y += y
		mesh.VertexPositions[index].Z += z

	})

}

// Move moves all vertices contained within the VertexSelection by the provided 3D vector.
func (vs VertexSelection) MoveVec(vec Vector3) {

	vs.ForEachIndex(func(mesh *Mesh, index int) {

		mesh.VertexPositions[index].X += vec.X
		mesh.VertexPositions[index].Y += vec.Y
		mesh.VertexPositions[index].Z += vec.Z

	})

}

// ForEachIndex calls the provided function for each index selected in the VertexSelection.
func (vs VertexSelection) ForEachIndex(forEach func(mesh *Mesh, index int)) {
	for mesh, set := range vs.SelectionSet {
		if set.SelectAll {
			for index := range mesh.VertexPositions {
				forEach(mesh, index)
			}
		} else {
			for index := range set.Indices {
				forEach(mesh, index)
			}
		}
	}
}

// Count returns the number of indices selected in the VertexSelection.
func (vs VertexSelection) Count() int {
	count := 0
	for _, set := range vs.SelectionSet {
		count += len(set.Indices)
	}
	return count
}

// NewCubeMesh creates a new 2x2x2 Cube Mesh and gives it a new material (suitably named "Cube").
func NewCubeMesh() *Mesh {

	mesh := NewMesh("Cube",

		// Top

		NewVertex(-1, 1, -1, 0, 0),
		NewVertex(1, 1, 1, 1, 1),
		NewVertex(1, 1, -1, 1, 0),
		NewVertex(-1, 1, 1, 0, 1),

		// Bottom

		NewVertex(1, -1, -1, 1, 0),
		NewVertex(1, -1, 1, 1, 1),
		NewVertex(-1, -1, -1, 0, 0),
		NewVertex(-1, -1, 1, 0, 1),

		// Front

		NewVertex(-1, 1, 1, 0, 0),
		NewVertex(1, -1, 1, 1, 1),
		NewVertex(1, 1, 1, 1, 0),
		NewVertex(-1, -1, 1, 0, 1),

		// Back

		NewVertex(1, 1, -1, 1, 0),
		NewVertex(1, -1, -1, 1, 1),
		NewVertex(-1, 1, -1, 0, 0),
		NewVertex(-1, -1, -1, 0, 1),

		// Right

		NewVertex(1, 1, -1, 1, 0),
		NewVertex(1, 1, 1, 1, 1),
		NewVertex(1, -1, -1, 0, 0),
		NewVertex(1, -1, 1, 0, 1),

		// Left

		NewVertex(-1, -1, -1, 0, 0),
		NewVertex(-1, 1, 1, 1, 1),
		NewVertex(-1, 1, -1, 1, 0),
		NewVertex(-1, -1, 1, 0, 1),
	)

	mesh.AddMeshPart(

		NewMaterial("Cube"),

		// Top
		0, 1, 2,
		3, 1, 0,

		// Bottom
		4, 5, 6,
		6, 5, 7,

		// Front

		8, 9, 10,
		11, 9, 8,

		// Back

		12, 13, 14,
		14, 13, 15,

		// Right

		16, 17, 18,
		18, 17, 19,

		// Left

		20, 21, 22,
		23, 21, 20,
	)

	mesh.UpdateBounds()
	mesh.AutoNormal()

	return mesh

}

// NewPrismMesh creates a new prism Mesh and gives it a new material (suitably named "Prism").
func NewPrismMesh() *Mesh {

	mesh := NewMesh("Prism",

		// Top

		NewVertex(0, 1, 0, 0, 0),

		// Middle

		NewVertex(-1, 0, 1, 0, 0),
		NewVertex(1, 0, 1, 1, 1),
		NewVertex(1, 0, -1, 1, 0),
		NewVertex(-1, 0, -1, 0, 1),

		// Bottom

		NewVertex(0, -1, 0, 1, 0),
	)

	mesh.AddMeshPart(

		NewMaterial("Prism"),

		// Upper half
		0, 1, 2,
		0, 2, 3,
		0, 3, 4,
		0, 4, 1,

		// // Bottom half
		1, 5, 2,
		2, 5, 3,
		3, 5, 4,
		4, 5, 1,
	)

	mesh.UpdateBounds()
	mesh.AutoNormal()

	return mesh

}

// NewIcosphereMesh creates a new icosphere Mesh of the specified detail level. Note that the UVs are left at {0,0} because I'm lazy.
func NewIcosphereMesh(detailLevel int) *Mesh {

	// Code cribbed from http://blog.andreaskahler.com/2009/06/creating-icosphere-mesh-in-code.html, thank you very much Andreas!

	mesh := NewMesh("Icosphere")

	t := (1.0 + math32.Sqrt(5)) / 2

	v0 := NewVertex(-1, t, 0, 0, 0)
	v1 := NewVertex(1, t, 0, 0, 0)
	v2 := NewVertex(-1, -t, 0, 0, 0)
	v3 := NewVertex(1, -t, 0, 0, 0)

	v4 := NewVertex(0, -1, t, 0, 0)
	v5 := NewVertex(0, 1, t, 0, 0)
	v6 := NewVertex(0, -1, -t, 0, 0)
	v7 := NewVertex(0, 1, -t, 0, 0)

	v8 := NewVertex(t, 0, -1, 0, 0)
	v9 := NewVertex(t, 0, 1, 0, 0)
	v10 := NewVertex(-t, 0, -1, 0, 0)
	v11 := NewVertex(-t, 0, 1, 0, 0)

	indices := []int{
		0, 11, 5,
		0, 5, 1,
		0, 1, 7,
		0, 7, 10,
		0, 10, 11,

		1, 5, 9,
		5, 11, 4,
		11, 10, 2,
		10, 7, 6,
		7, 1, 8,

		3, 9, 4,
		3, 4, 2,
		3, 2, 6,
		3, 6, 8,
		3, 8, 9,

		4, 9, 5,
		2, 4, 11,
		6, 2, 10,
		8, 6, 7,
		9, 8, 1,
	}

	vertices := []VertexInfo{
		v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11,
	}

	type cacheStorage struct {
		v0, v1 int
	}

	cache := map[cacheStorage]int{}

	getMiddlePoint := func(index0, index1 int) int {

		if v, exists := cache[cacheStorage{index0, index1}]; exists {
			return v
		}

		a := Vector3{vertices[index0].X, vertices[index0].Y, vertices[index0].Z}
		b := Vector3{vertices[index1].X, vertices[index1].Y, vertices[index1].Z}
		midA := a.Add(b).Divide(2)
		// midA := a.Add(b.Sub(a).Scale(0.5))
		vertices = append(vertices, NewVertex(midA.X, midA.Y, midA.Z, 0, 0))

		cache[cacheStorage{index0, index1}] = len(vertices) - 1

		return len(vertices) - 1

	}

	if detailLevel > 0 {

		for i := 0; i < detailLevel+1; i++ {

			newIndices := make([]int, 0, len(indices)*4)

			for i := 0; i < len(indices); i += 3 {

				midAB := getMiddlePoint(indices[i], indices[i+1])
				midBC := getMiddlePoint(indices[i+1], indices[i+2])
				midCA := getMiddlePoint(indices[i+2], indices[i])

				newIndices = append(newIndices, indices[i], midAB, midCA)
				newIndices = append(newIndices, indices[i+1], midBC, midAB)
				newIndices = append(newIndices, indices[i+2], midCA, midBC)
				newIndices = append(newIndices, midAB, midBC, midCA)
				// newIndices = append(newIndices, indices[i], indices[i+1], indices[i+2])

			}

			indices = newIndices

		}

	}

	for i := range vertices {

		v := Vector3{vertices[i].X, vertices[i].Y, vertices[i].Z}.Unit() // Make it spherical
		vertices[i].X = v.X
		vertices[i].Y = v.Y
		vertices[i].Z = v.Z

		vertices[i].U = 0
		vertices[i].V = 0
		if i%3 == 1 {
			vertices[i].U = 1
		} else if i%3 == 2 {
			vertices[i].V = 1
		}

	}

	mesh.AddVertices(vertices...)

	mesh.AddMeshPart(NewMaterial("Icosphere"), indices...)

	mesh.AutoNormal()

	mesh.UpdateBounds()

	return mesh
}

// NewPlaneMesh creates a new 2x2 plane Mesh with a new material (suitably named "Plane").
// vertexCountX and vertexCountY indicate how many vertices should be on the plane in the X
// and Z direction, respectively. The minimum number of vertices for either argument is 2.
// Code for this is taken from https://answers.unity.com/questions/1850185/mesh-triangles-not-filling-whole-space-2.html.
func NewPlaneMesh(vertexCountX, vertexCountZ int) *Mesh {

	if vertexCountX < 2 {
		vertexCountX = 2
	}

	if vertexCountZ < 2 {
		vertexCountZ = 2
	}

	verts := []VertexInfo{}

	for z := 0; z < vertexCountZ; z++ {

		for x := 0; x < vertexCountX; x++ {

			tx := float32(x) / float32(vertexCountX-1) * 2
			tz := float32(z) / float32(vertexCountZ-1) * 2
			verts = append(verts, NewVertex(-0.5+tx, 0, -0.5+tz, tx, -tz))

		}

	}

	mesh := NewMesh("Plane", verts...)

	indices := []int{}

	for z := 0; z < vertexCountZ-1; z++ {

		for x := 0; x < vertexCountX-1; x++ {

			quad := z*vertexCountX + x

			indices = append(indices,
				quad+1, quad+vertexCountX, quad+vertexCountX+1,
				quad, quad+vertexCountX, quad+1,
			)

		}

	}

	mesh.AddMeshPart(NewMaterial("Plane"), indices...)

	mesh.UpdateBounds()
	mesh.AutoNormal()

	return mesh

}

// NewCylinderMesh creates a new cylinder Mesh and gives it a new material (suitably named "Cylinder").
// sideCount is how many sides the cylinder should have, while radius is the radius of the cylinder in world units.
// if createCaps is true, then the cylinder will have triangle caps.
func NewCylinderMesh(sideCount int, radius, height float32, createCaps bool) *Mesh {

	if sideCount < 3 {
		sideCount = 3
	}

	mesh := NewMesh("Cylinder")

	verts := []VertexInfo{}

	for i := 0; i < sideCount; i++ {

		pos := NewVector3(radius, height/2, 0)
		pos = pos.Rotate(0, 1, 0, float32(i)/float32(sideCount)*math.Pi*2)
		verts = append(verts, NewVertex(pos.X, pos.Y, pos.Z, 0, 0))

	}

	for i := 0; i < sideCount; i++ {

		pos := NewVector3(radius, -height/2, 0)
		pos = pos.Rotate(0, 1, 0, float32(i)/float32(sideCount)*math.Pi*2)
		verts = append(verts, NewVertex(pos.X, pos.Y, pos.Z, 0, 0))

	}

	if createCaps {
		verts = append(verts, NewVertex(0, height/2, 0, 0, 0))
		verts = append(verts, NewVertex(0, -height/2, 0, 0, 0))
	}

	mesh.AddVertices(verts...)

	indices := []int{}
	for i := 0; i < sideCount; i++ {
		if i < sideCount-1 {
			indices = append(indices, i, i+sideCount, i+1)
			indices = append(indices, i+1, i+sideCount, i+sideCount+1)
		} else {
			indices = append(indices, i, i+sideCount, 0)
			indices = append(indices, 0, i+sideCount, sideCount)
		}
	}

	if createCaps {
		for i := 0; i < sideCount; i++ {
			topCenter := len(verts) - 2
			bottomCenter := len(verts) - 1
			if i < sideCount-1 {
				indices = append(indices, i, i+1, topCenter)
				indices = append(indices, i+sideCount+1, i+sideCount, bottomCenter)
			} else {
				indices = append(indices, i, 0, topCenter)
				indices = append(indices, sideCount, i+sideCount, bottomCenter)
			}
		}
	}

	mesh.AddMeshPart(NewMaterial("Cylinder"), indices...)

	mesh.UpdateBounds()
	mesh.AutoNormal()

	return mesh

}

// func NewWeirdDebuggingStatueThing() *Mesh {

// 	mesh := NewMesh("Weird Statue")

// 	part := mesh.AddMeshPart(NewMaterial("Weird Statue"))

// 	part.AddTriangles(

// 		NewVertex(1, 0, -1, 1, 0),
// 		NewVertex(-1, 0, -1, 0, 0),
// 		NewVertex(1, 0, 1, 1, 1),

// 		NewVertex(-1, 0, -1, 0, 0),
// 		NewVertex(-1, 0, 1, 0, 1),
// 		NewVertex(1, 0, 1, 1, 1),

// 		NewVertex(-1, 2, -1, 0, 0),
// 		NewVertex(1, 2, 1, 1, 1),
// 		NewVertex(1, 0, -1, 1, 0),

// 		NewVertex(-1, 0, 1, 0, 1),
// 		NewVertex(1, 2, 1, 1, 1),
// 		NewVertex(-1, 2, -1, 0, 0),
// 	)

// 	mesh.UpdateBounds()

// 	return mesh

// }

// A Triangle represents the smallest renderable object in Tetra3D. A triangle contains very little data, and is mainly used to help identify triads of vertices.
type Triangle struct {
	VertexIndices []int     // Vertex indices that compose the triangle
	MaxSpan       float32   // The maximum span from corner to corner of the triangle's dimensions; this is used in intersection testing.
	Center        Vector3   // The untransformed center of the Triangle.
	Normal        Vector3   // The physical normal of the triangle (i.e. the direction the triangle is facing). This is different from the visual normals of a triangle's vertices (i.e. a selection of vertices can have inverted normals to be see through, for example).
	MeshPart      *MeshPart // The specific MeshPart this Triangle belongs to.
}

// NewTriangle creates a new Triangle, and requires a reference to its owning MeshPart.
func NewTriangle(meshPart *MeshPart, indices ...int) *Triangle {
	tri := &Triangle{
		MeshPart:      meshPart,
		VertexIndices: []int{indices[0], indices[1], indices[2]},
		Center:        Vector3{0, 0, 0},
	}
	return tri
}

// Clone clones the Triangle, keeping a reference to the same Material.
func (tri *Triangle) Clone() *Triangle {
	newTri := NewTriangle(tri.MeshPart, tri.VertexIndices...)
	newTri.MaxSpan = tri.MaxSpan
	newTri.Center = tri.Center
	newTri.Normal = tri.Normal
	newTri.MeshPart = tri.MeshPart
	return newTri
}

// RecalculateCenter recalculates the center for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position.
func (tri *Triangle) RecalculateCenter() {

	verts := tri.MeshPart.Mesh.VertexPositions

	indices := tri.VertexIndices

	// tri.Center[0] = (verts[indices[0]] + verts[indices[0]] + verts[tri.ID*3+2][0]) / 3
	// tri.Center[1] = (verts[indices[1]] + verts[indices[0]] + verts[tri.ID*3+2][1]) / 3
	// tri.Center[2] = (verts[indices[2]] + verts[indices[0]] + verts[tri.ID*3+2][2]) / 3

	tri.Center.X = (verts[indices[0]].X + verts[indices[1]].X + verts[indices[2]].X) / 3
	tri.Center.Y = (verts[indices[0]].Y + verts[indices[1]].Y + verts[indices[2]].Y) / 3
	tri.Center.Z = (verts[indices[0]].Z + verts[indices[1]].Z + verts[indices[2]].Z) / 3

	// Determine the maximum span of the triangle; this is done for bounds checking, since we can reject triangles early if we
	// can easily tell we're too far away from them.
	dim := Dimensions{
		Min: Vector3{0, 0, 0},
		Max: Vector3{0, 0, 0},
	}

	for i := 0; i < 3; i++ {

		if dim.Min.X > verts[indices[i]].X {
			dim.Min.X = verts[indices[i]].X
		}

		if dim.Min.Y > verts[indices[i]].Y {
			dim.Min.Y = verts[indices[i]].Y
		}

		if dim.Min.Z > verts[indices[i]].Z {
			dim.Min.Z = verts[indices[i]].Z
		}

		if dim.Max.X < verts[indices[i]].X {
			dim.Max.X = verts[indices[i]].X
		}

		if dim.Max.Y < verts[indices[i]].Y {
			dim.Max.Y = verts[indices[i]].Y
		}

		if dim.Max.Z < verts[indices[i]].Z {
			dim.Max.Z = verts[indices[i]].Z
		}

	}

	tri.MaxSpan = dim.MaxSpan()

}

// RecalculateNormal recalculates the physical normal for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position. Also note that vertex normals (visual normals) are automatically set when loading Meshes from model files.
func (tri *Triangle) RecalculateNormal() {
	verts := tri.MeshPart.Mesh.VertexPositions
	tri.Normal = calculateNormal(verts[tri.VertexIndices[0]], verts[tri.VertexIndices[1]], verts[tri.VertexIndices[2]])
}

// ResetVertexNormals sets all vertex normals associated with the triangle to the triangle's normal.
func (tri *Triangle) ResetVertexNormals() {
	mesh := tri.MeshPart.Mesh
	mesh.VertexNormals[tri.VertexIndices[0]] = tri.Normal
	mesh.VertexNormals[tri.VertexIndices[1]] = tri.Normal
	mesh.VertexNormals[tri.VertexIndices[2]] = tri.Normal
}

// SharesVertexPosition returns the indices of the vertices of the other Triangle that share indices with the calling Triangle.
// The last value returned is the count, the number of vertices shared (from 0 to 3).
// If a vertex is not shared, the value returned for shareA, shareB, or shareC is -1.
func (tri *Triangle) SharesVertexPositions(other *Triangle) (shareA, shareB, shareC, count int) {

	triIndices := tri.VertexIndices
	otherIndices := other.VertexIndices

	mesh := tri.MeshPart.Mesh
	otherMesh := other.MeshPart.Mesh

	shareA = -1
	shareB = -1
	shareC = -1
	count = 0

	for i, index := range triIndices {
		for _, otherIndex := range otherIndices {
			if mesh.VertexPositions[index].Equals(otherMesh.VertexPositions[otherIndex]) {
				switch i {
				case 0:
					shareA = otherIndex
					count++
				case 1:
					shareB = otherIndex
					count++
				case 2:
					shareC = otherIndex
					count++
				}
				break
			}
		}
	}

	return shareA, shareB, shareC, count

}

// type sharedTriangleSet struct {
// 	Triangle               *Triangle
// 	Index0, Index1, Index2 int
// }

// var sharedTriangles = []sharedTriangleSet{}

// func (tri *Triangle) ForEachSharedTriangle(forEach func(sharedTriangle *Triangle, sharedVertexIndexA, sharedVertexIndexB, sharedVertexIndexC int)) {

// 	sharedTriangles = sharedTriangles[:0]

// 	for _, mp := range tri.MeshPart.Mesh.MeshParts {

// 		mp.ForEachTri(func(t *Triangle) {

// 			if t == tri {
// 				return
// 			}

// 			v1, v2, v3 := tri.SharesVertexPositions(t)
// 			if v1 >= 0 || v2 >= 0 || v3 >= 0 {
// 				sharedTriangles = append(sharedTriangles, sharedTriangleSet{
// 					Triangle: t,
// 					Index0:   v1,
// 					Index1:   v2,
// 					Index2:   v3,
// 				})
// 			}

// 		})

// 	}

// 	for _, set := range sharedTriangles {
// 		forEach(set.Triangle, set.Index0, set.Index1, set.Index2)
// 	}

// }

// var sharedVerticesSet Set[int] = Set[int]{}

// func (tri *Triangle) ForEachSharedVertex(includeOwnVertices bool, forEach func(sharedVertexIndex int)) {

// 	sharedVerticesSet.Clear()

// 	if includeOwnVertices {
// 		sharedVerticesSet.Add(tri.VertexIndices[0])
// 		sharedVerticesSet.Add(tri.VertexIndices[1])
// 		sharedVerticesSet.Add(tri.VertexIndices[2])
// 	}

// 	for _, mp := range tri.MeshPart.Mesh.MeshParts {

// 		mp.ForEachTri(func(t *Triangle) {

// 			if t == tri {
// 				return
// 			}

// 			v1, v2, v3 := tri.SharesVertexPositions(t)
// 			if v1 >= 0 {
// 				sharedVerticesSet.Add(v1)
// 			}
// 			if v2 >= 0 {
// 				sharedVerticesSet.Add(v2)
// 			}
// 			if v3 >= 0 {
// 				sharedVerticesSet.Add(v3)
// 			}

// 		})

// 	}

// 	for index := range sharedVerticesSet {
// 		forEach(index)
// 	}

// }

func (tri *Triangle) ForEachVertexIndex(vertFunc func(vertexIndex int)) {
	for _, vi := range tri.VertexIndices {
		vertFunc(vi)
	}
}

func calculateNormal(p1, p2, p3 Vector3) Vector3 {

	v0 := p2.Sub(p1)
	v1 := p3.Sub(p2)

	return v0.Cross(v1).Unit()

}

// MeshPart represents a collection of vertices and triangles, which are all rendered at once, as a single part, with a single material.
// Depth testing is done between mesh parts or objects, so splitting an object up into different materials can be effective to help with depth sorting.
type MeshPart struct {
	Mesh             *Mesh
	Material         *Material
	VertexIndexStart int
	VertexIndexEnd   int
	TriangleStart    int
	TriangleEnd      int
}

// NewMeshPart creates a new MeshPart that renders using the specified Material.
func NewMeshPart(mesh *Mesh, material *Material) *MeshPart {
	mp := &MeshPart{
		Mesh:             mesh,
		Material:         material,
		TriangleStart:    math.MaxInt,
		VertexIndexStart: mesh.vertsAddStart,
	}

	return mp
}

// Clone clones the MeshPart, returning the copy.
func (part *MeshPart) Clone() *MeshPart {

	newMP := NewMeshPart(part.Mesh, part.Material)

	newMP.VertexIndexStart = part.VertexIndexStart
	newMP.VertexIndexEnd = part.VertexIndexEnd
	newMP.TriangleStart = part.TriangleStart
	newMP.TriangleEnd = part.TriangleEnd

	newMP.Material = part.Material
	return newMP
}

func (part *MeshPart) AssignToMesh(mesh *Mesh) {

	for _, p := range mesh.MeshParts {
		if p == part {
			return
		}
	}

	mesh.MeshParts = append(mesh.MeshParts, part)

	part.Mesh = mesh

}

func (part *MeshPart) ForEachTri(triFunc func(tri *Triangle)) {
	for i := part.TriangleStart; i <= part.TriangleEnd; i++ {
		triFunc(part.Mesh.Triangles[i])
	}
}

// ForEachVertexIndex calls the provided function for each vertex index that the MeshPart uses.
// If onlyVisible is true, then only the visible vertices (vertices that are rendered; that aren't
// backface culled or offscreen) will be used with the provided function.
func (part *MeshPart) ForEachVertexIndex(vertFunc func(vertIndex int), onlyVisible bool) {

	for i := part.VertexIndexStart; i < part.VertexIndexEnd; i++ {
		if !onlyVisible || part.Mesh.visibleVertices[i] {
			vertFunc(i)
		}
	}

}

func (part *MeshPart) VertexIndexCount() int {
	return part.VertexIndexEnd - part.VertexIndexStart
}

// AddTriangles adds triangles to the MeshPart using the provided VertexInfo slice.
func (part *MeshPart) AddTriangles(indices ...int) {

	mesh := part.Mesh

	if len(indices) == 0 || len(indices)%3 > 0 {
		panic("Error: MeshPart.AddTriangles() not given enough vertex indices to construct complete triangles (i.e. multiples of 3 vertices).")
	}

	// for _, i := range indices {
	// 	if int(i) > len(mesh.VertexPositions) {
	// 		panic("Error: Cannot add triangles to MeshPart with indices > length of vertices passed. Errorful index: " + strconv.Itoa(int(i)))
	// 	}
	// }

	part.VertexIndexEnd = part.Mesh.vertsAddEnd

	// for _, index := range indices {

	// 	// index += uint32(part.VertexIndexEnd)

	// 	if index < uint32(part.VertexIndexStart) {
	// 		part.VertexIndexStart = int(index)
	// 	}
	// 	if index > uint32(part.VertexIndexEnd) {
	// 		part.VertexIndexEnd = int(index)
	// 	}
	// }

	for i := 0; i < len(indices); i += 3 {

		if mesh.triIndex < part.TriangleStart {
			part.TriangleStart = mesh.triIndex
		}
		if mesh.triIndex > part.TriangleEnd {
			part.TriangleEnd = mesh.triIndex
		}

		newTri := NewTriangle(part, indices[i]+part.VertexIndexStart, indices[i+1]+part.VertexIndexStart, indices[i+2]+part.VertexIndexStart)

		// Slight UV adjustment to not have invisibility in-between seams
		uvcenter := mesh.VertexUVs[indices[i]].Add(mesh.VertexUVs[indices[i+1]]).Add(mesh.VertexUVs[indices[i+2]]).Divide(3)

		mesh.VertexUVs[indices[i]].X += (uvcenter.X - mesh.VertexUVs[indices[i]].X) * 0.0001
		mesh.VertexUVs[indices[i]].Y += (uvcenter.Y - mesh.VertexUVs[indices[i]].Y) * 0.0001
		mesh.VertexUVs[indices[i+1]].X += (uvcenter.X - mesh.VertexUVs[indices[i+1]].X) * 0.0001
		mesh.VertexUVs[indices[i+1]].Y += (uvcenter.Y - mesh.VertexUVs[indices[i+1]].Y) * 0.0001
		mesh.VertexUVs[indices[i+2]].X += (uvcenter.X - mesh.VertexUVs[indices[i+2]].X) * 0.0001
		mesh.VertexUVs[indices[i+2]].Y += (uvcenter.Y - mesh.VertexUVs[indices[i+2]].Y) * 0.0001

		newTri.RecalculateCenter()
		newTri.RecalculateNormal()
		mesh.Triangles = append(mesh.Triangles, newTri)
		mesh.triIndex++
		if mesh.maxTriangleSpan < newTri.MaxSpan {
			mesh.maxTriangleSpan = newTri.MaxSpan
		}
	}

	if part.TriangleCount() >= MaxTriangleCount {
		matName := "nil"
		if part.Material != nil {
			matName = part.Material.Name
		}
		log.Println("warning: mesh [" + part.Mesh.Name + "] has part with material named [" + matName + "], which has " + fmt.Sprintf("%d", part.TriangleCount()) + " triangles. This exceeds the renderable maximum of 21845 triangles total for one MeshPart; please break up the mesh into multiple MeshParts using materials, or split it up into multiple models. Otherwise, the game will crash if it renders over the maximum number of triangles.")
	}

}

// TriangleCount returns the total number of triangles in the MeshPart, specifically.
func (part *MeshPart) TriangleCount() int {
	return part.TriangleEnd - part.TriangleStart + 1
}

// ApplyMatrix applies a transformation matrix to the vertices referenced by the MeshPart.
func (part *MeshPart) ApplyMatrix(matrix Matrix4) {
	mesh := part.Mesh

	part.ForEachTri(

		func(tri *Triangle) {

			for _, index := range tri.VertexIndices {
				mesh.VertexPositions[index] = matrix.MultVec(mesh.VertexPositions[index])
			}

		},
	)

}

// This is supposed to return the primary two dimensions of the points constructing the MeshPart.
// TODO: Make this work regardless of orientation of the vertices?
func (part *MeshPart) primaryDimensions() (float32, float32) {

	points := make([]Vector3, 0, len(part.Mesh.VertexPositions))

	part.ForEachVertexIndex(func(vertIndex int) { points = append(points, part.Mesh.VertexPositions[vertIndex]) }, false)

	dimensions := NewDimensionsFromPoints(points...)

	x := dimensions.Width()
	y := dimensions.Height()
	z := dimensions.Depth()

	w, h := float32(0), float32(0)

	if z < y && z < x {
		w = x
		h = y
	} else if y < x && y < z {
		w = x
		h = z
	} else {
		w = z
		h = y
	}

	return w, h

}

func (p *MeshPart) isVisible() bool {
	if p.Material != nil {
		return p.Material.Visible
	}
	return true
}

type VertexInfo struct {
	ID                        int
	X, Y, Z                   float32
	U, V                      float32
	NormalX, NormalY, NormalZ float32
	Weights                   []float32
	Colors                    []Color
	Bones                     []uint16
}

// GetVertexInfo returns a VertexInfo struct containing the vertex information for the vertex with the provided index.
func (mesh *Mesh) GetVertexInfo(vertexIndex int) VertexInfo {

	v := VertexInfo{
		ID:      vertexIndex,
		X:       mesh.VertexPositions[vertexIndex].X,
		Y:       mesh.VertexPositions[vertexIndex].Y,
		Z:       mesh.VertexPositions[vertexIndex].Z,
		U:       mesh.VertexUVs[vertexIndex].X,
		V:       mesh.VertexUVs[vertexIndex].Y,
		NormalX: mesh.VertexNormals[vertexIndex].X,
		NormalY: mesh.VertexNormals[vertexIndex].Y,
		NormalZ: mesh.VertexNormals[vertexIndex].Z,
		Bones:   mesh.VertexBones[vertexIndex],
		Weights: mesh.VertexWeights[vertexIndex],
	}

	colors := []Color{}

	for _, channel := range mesh.VertexColors {
		colors = append(colors, channel[vertexIndex])
	}

	v.Colors = colors

	return v

}

// NewVertex creates a new vertex information struct, which is used to create new Triangles. VertexInfo is purely for getting data into
// Meshes' vertex buffers, so after creating the Triangle, VertexInfos can be safely discarded (i.e. the VertexInfo doesn't hold "power"
// over the vertex it represents after creation).
func NewVertex(x, y, z, u, v float32) VertexInfo {
	return VertexInfo{
		X:       x,
		Y:       y,
		Z:       z,
		U:       u,
		V:       v,
		Weights: []float32{},
		Colors:  make([]Color, 0, 8),
		Bones:   []uint16{},
	}
}
