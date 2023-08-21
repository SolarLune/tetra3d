package tetra3d

import (
	"fmt"
	"log"
	"math"
)

// Dimensions represents the minimum and maximum spatial dimensions of a Mesh arranged in a 2-space Vector slice.
type Dimensions struct {
	Min, Max Vector
}

func NewEmptyDimensions() Dimensions {
	return Dimensions{
		Vector{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64, 0},
		Vector{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64, 0},
	}
}

// MaxDimension returns the maximum value from all of the axes in the Dimensions. For example, if the Dimensions have a min of [-1, -2, -2],
// and a max of [6, 1.5, 1], Max() will return 7 for the X axis, as it's the largest distance between all axes.
func (dim Dimensions) MaxDimension() float64 {
	return math.Max(math.Max(dim.Width(), dim.Height()), dim.Depth())
}

// MaxSpan returns the maximum span between the corners of the dimension set.
func (dim Dimensions) MaxSpan() float64 {
	return dim.Max.Sub(dim.Min).Magnitude()
}

// Center returns the center point inbetween the two corners of the dimension set.
func (dim Dimensions) Center() Vector {
	return Vector{
		(dim.Max.X + dim.Min.X) / 2,
		(dim.Max.Y + dim.Min.Y) / 2,
		(dim.Max.Z + dim.Min.Z) / 2,
		0,
	}
}

// Width returns the total difference between the minimum and maximum X values.
func (dim Dimensions) Width() float64 {
	return dim.Max.X - dim.Min.X
}

// Height returns the total difference between the minimum and maximum Y values.
func (dim Dimensions) Height() float64 {
	return dim.Max.Y - dim.Min.Y
}

// Depth returns the total difference between the minimum and maximum Z values.
func (dim Dimensions) Depth() float64 {
	return dim.Max.Z - dim.Min.Z
}

// Clamp limits the provided position vector to be within the dimensions set.
func (dim Dimensions) Clamp(position Vector) Vector {

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
func (dim Dimensions) Inside(position Vector) bool {

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

func (dim Dimensions) Size() Vector {
	return Vector{dim.Width(), dim.Height(), dim.Depth(), 0}
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
func NewDimensionsFromPoints(points ...Vector) Dimensions {

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

// Mesh represents a mesh that can be represented visually in different locations via Models. By default, a new Mesh has no MeshParts (so you would need to add one
// manually if you want to construct a Mesh via code).
type Mesh struct {
	Name    string   // The name of the Mesh resource
	Unique  bool     // If true, whenever a Mesh is used for a Model and the Model is cloned, the Mesh is cloned with it. Useful for things like sprites.
	library *Library // A reference to the Library this Mesh came from.

	MeshParts []*MeshPart // The various mesh parts (collections of triangles, rendered with a single material).
	Triangles []*Triangle // The various triangles composing the Mesh.
	triIndex  int

	// Vertices are stored as a struct-of-arrays for simplified and faster rendering.
	// Each vertex property (position, normal, UV, colors, weights, bones, etc) is stored
	// here and indexed in order of vertex index.
	vertexTransforms         []Vector
	VertexPositions          []Vector
	VertexNormals            []Vector
	vertexSkinnedNormals     []Vector
	vertexSkinnedPositions   []Vector
	vertexTransformedNormals []Vector
	VertexUVs                []Vector
	VertexColors             [][]Color
	VertexActiveColorChannel []int       // VertexActiveColorChannel is the active vertex color used for coloring the vertex in the index given.
	VertexWeights            [][]float32 // TODO: Replace this with [][8]float32 (or however many the maximum is for GLTF)
	VertexBones              [][]uint16  // TODO: Replace this with [][8]uint16 (or however many the maximum number of bones affecting a single vertex is for GLTF)
	visibleVertices          []bool

	vertexLights  []Color
	vertsAddStart int
	vertsAddEnd   int

	VertexColorChannelNames map[string]int // VertexColorChannelNames is a map allowing you to get the index of a mesh's vertex color channel by its name.
	Dimensions              Dimensions
	Properties              Properties
}

// NewMesh takes a name and a slice of *Vertex instances, and returns a new Mesh. If you provide *Vertex instances, the number must be divisible by 3,
// or NewMesh will panic.
func NewMesh(name string, verts ...VertexInfo) *Mesh {

	mesh := &Mesh{
		Name:                    name,
		MeshParts:               []*MeshPart{},
		Dimensions:              Dimensions{Vector{0, 0, 0, 0}, Vector{0, 0, 0, 0}},
		VertexColorChannelNames: map[string]int{},
		Properties:              NewProperties(),

		vertexTransforms:         []Vector{},
		VertexPositions:          []Vector{},
		visibleVertices:          []bool{},
		VertexNormals:            []Vector{},
		vertexSkinnedNormals:     []Vector{},
		vertexSkinnedPositions:   []Vector{},
		vertexTransformedNormals: []Vector{},
		vertexLights:             []Color{},
		VertexUVs:                []Vector{},
		VertexColors:             [][]Color{},
		VertexActiveColorChannel: []int{},
		VertexBones:              [][]uint16{},
		VertexWeights:            [][]float32{},
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
	newMesh.Properties = mesh.Properties.Clone()
	newMesh.triIndex = mesh.triIndex

	newMesh.allocateVertexBuffers(len(mesh.VertexPositions))

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
	}

	for i := range mesh.VertexColors {
		newMesh.VertexColors = append(newMesh.VertexColors, make([]Color, len(mesh.VertexColors[i])))
		for channelIndex := range mesh.VertexColors[i] {
			newMesh.VertexColors[i][channelIndex] = mesh.VertexColors[i][channelIndex]
		}
	}

	for c := range mesh.VertexActiveColorChannel {
		newMesh.VertexActiveColorChannel = append(newMesh.VertexActiveColorChannel, mesh.VertexActiveColorChannel[c])
	}

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

	return newMesh
}

// allocateVertexBuffers allows us to allocate the slices for vertex properties all at once rather than resizing multiple times as
// we append to a slice and have its backing buffer automatically expanded (which is slower).
func (mesh *Mesh) allocateVertexBuffers(vertexCount int) {

	if cap(mesh.VertexPositions) >= vertexCount {
		return
	}

	mesh.VertexPositions = append(make([]Vector, 0, vertexCount), mesh.VertexPositions...)
	if len(mesh.visibleVertices) < vertexCount {
		mesh.visibleVertices = make([]bool, vertexCount)
	}

	mesh.VertexNormals = append(make([]Vector, 0, vertexCount), mesh.VertexNormals...)

	mesh.vertexLights = append(make([]Color, 0, vertexCount), mesh.vertexLights...)

	mesh.VertexUVs = append(make([]Vector, 0, vertexCount), mesh.VertexUVs...)

	mesh.VertexColors = append(make([][]Color, 0, vertexCount), mesh.VertexColors...)

	mesh.VertexActiveColorChannel = append(make([]int, 0, vertexCount), mesh.VertexActiveColorChannel...)

	mesh.VertexBones = append(make([][]uint16, 0, vertexCount), mesh.VertexBones...)

	mesh.VertexWeights = append(make([][]float32, 0, vertexCount), mesh.VertexWeights...)

	mesh.vertexTransforms = append(make([]Vector, 0, vertexCount), mesh.vertexTransforms...)

	mesh.vertexSkinnedNormals = append(make([]Vector, 0, vertexCount), mesh.vertexSkinnedNormals...)

	mesh.vertexTransformedNormals = append(make([]Vector, 0, vertexCount), mesh.vertexTransformedNormals...)

	mesh.vertexSkinnedPositions = append(make([]Vector, 0, vertexCount), mesh.vertexSkinnedPositions...)

}

func (mesh *Mesh) ensureEnoughVertexColorChannels(channelIndex int) {

	for i := range mesh.VertexColors {
		for len(mesh.VertexColors[i]) <= channelIndex {
			mesh.VertexColors[i] = append(mesh.VertexColors[i], NewColor(1, 1, 1, 1))
		}
	}

}

// CombineVertexColors allows you to combine vertex color channels together. The targetChannel is the channel that will hold
// the result, and multiplicative controls whether the combination is multiplicative (true) or additive (false). The sourceChannels
// ...int is the vertex color channel indices to combine together.
func (mesh *Mesh) CombineVertexColors(targetChannel int, multiplicative bool, sourceChannels ...int) {

	if len(sourceChannels) == 0 {
		return
	}

	for i := range mesh.VertexColors {

		base := NewColor(0, 0, 0, 1)

		if multiplicative {
			base = NewColor(1, 1, 1, 1)
		}

		for _, c := range sourceChannels {

			if len(mesh.VertexColors[i]) <= c {
				continue
			}

			if multiplicative {
				base = base.Multiply(mesh.VertexColors[i][c])
			} else {
				base = base.Add(mesh.VertexColors[i][c])
			}

		}

		mesh.ensureEnoughVertexColorChannels(targetChannel)

		mesh.VertexColors[i][targetChannel] = base

	}

}

// SetVertexColor sets the specified vertex color for all vertices in the mesh for the target color channel.
func (mesh *Mesh) SetVertexColor(targetChannel int, color Color) {
	mesh.SelectVertices().SelectAll().SetColor(targetChannel, color)
}

// SetActiveColorChannel sets the active color channel for all vertices in the mesh to the specified channel index.
func (mesh *Mesh) SetActiveColorChannel(targetChannel int) {
	mesh.SelectVertices().SelectAll().SetActiveColorChannel(targetChannel)
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
		mesh.VertexPositions = append(mesh.VertexPositions, Vector{vertInfo.X, vertInfo.Y, vertInfo.Z, 0})
		mesh.VertexNormals = append(mesh.VertexNormals, Vector{vertInfo.NormalX, vertInfo.NormalY, vertInfo.NormalZ, 0})
		mesh.VertexUVs = append(mesh.VertexUVs, Vector{vertInfo.U, vertInfo.V, 0, 0})
		mesh.VertexColors = append(mesh.VertexColors, vertInfo.Colors)
		mesh.VertexActiveColorChannel = append(mesh.VertexActiveColorChannel, vertInfo.ActiveColorChannel)
		mesh.VertexBones = append(mesh.VertexBones, vertInfo.Bones)
		mesh.VertexWeights = append(mesh.VertexWeights, vertInfo.Weights)

		mesh.vertexLights = append(mesh.vertexLights, NewColor(0, 0, 0, 1))
		mesh.vertexTransforms = append(mesh.vertexTransforms, Vector{0, 0, 0, 0}) // x, y, z, w
		mesh.vertexSkinnedNormals = append(mesh.vertexSkinnedNormals, Vector{0, 0, 0, 0})
		mesh.vertexTransformedNormals = append(mesh.vertexTransformedNormals, Vector{0, 0, 0, 0})
		mesh.vertexSkinnedPositions = append(mesh.vertexSkinnedPositions, Vector{0, 0, 0, 0})

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
func (mesh *Mesh) SelectVertices() *VertexSelection {
	return &VertexSelection{Indices: map[int]bool{}, Mesh: mesh}
}

// VertexSelection represents a selection of vertices on a Mesh.
type VertexSelection struct {
	Indices map[int]bool
	Mesh    *Mesh
}

// SelectInChannel selects all vertices in the Mesh that have a non-pure black color in the color channel
// with the specified index. If the index lies outside of the bounds of currently existent color channels,
// the color channel will be created.
func (vs *VertexSelection) SelectInChannel(channelIndex int) *VertexSelection {

	vs.Mesh.ensureEnoughVertexColorChannels(channelIndex)

	for vertexIndex := range vs.Mesh.VertexColors {

		color := vs.Mesh.VertexColors[vertexIndex][channelIndex]

		if color.R > 0.01 || color.G > 0.01 || color.B > 0.01 {
			vs.Indices[vertexIndex] = true
		}

	}

	return vs

}

// SelectAll selects all vertices on the source Mesh.
func (vs *VertexSelection) SelectAll() *VertexSelection {

	for i := 0; i < len(vs.Mesh.VertexPositions); i++ {
		vs.Indices[i] = true
	}

	return vs
}

// SelectMeshPart selects all vertices in the Mesh belonging to the specified MeshPart.
func (vs *VertexSelection) SelectMeshPart(meshPart *MeshPart) *VertexSelection {

	meshPart.ForEachTri(

		func(tri *Triangle) {

			for _, index := range tri.VertexIndices {
				vs.Indices[int(index)] = true
			}

		},
	)

	return vs

}

// SetColor sets the color of the specified channel in all vertices contained within the VertexSelection to the provided Color.
func (vs *VertexSelection) SetColor(channelIndex int, color Color) {

	vs.Mesh.ensureEnoughVertexColorChannels(channelIndex)

	for i := range vs.Indices {
		vs.Mesh.VertexColors[i][channelIndex] = NewColor(color.ToFloat32s())
	}

}

// SetNormal sets the normal of all vertices contained within the VertexSelection to the provided normal vector.
func (vs *VertexSelection) SetNormal(normal Vector) {

	for i := range vs.Indices {
		vs.Mesh.VertexNormals[i].X = normal.X
		vs.Mesh.VertexNormals[i].Y = normal.Y
		vs.Mesh.VertexNormals[i].Z = normal.Z
	}

}

// MoveUVs moves the UV values by the values specified.
func (vs *VertexSelection) MoveUVs(dx, dy float64) {

	for index := range vs.Indices {
		vs.Mesh.VertexUVs[index].X += dx
		vs.Mesh.VertexUVs[index].Y += dy
	}

}

// MoveUVsVec moves the UV values by the Vector values specified.
func (vs *VertexSelection) MoveUVsVec(vec Vector) {

	for index := range vs.Indices {
		vs.Mesh.VertexUVs[index].X += vec.X
		vs.Mesh.VertexUVs[index].Y += vec.Y
	}

}

// RotateUVs rotates the UV values around the center of the UV values for the mesh in radians
func (vs *VertexSelection) RotateUVs(rotation float64) {
	center := Vector{}

	for index := range vs.Indices {
		center = center.Add(vs.Mesh.VertexUVs[index])
	}

	center = center.Divide(float64(len(vs.Indices)))

	for index := range vs.Indices {
		diff := vs.Mesh.VertexUVs[index].Sub(center)
		vs.Mesh.VertexUVs[index] = center.Add(diff.Rotate(0, 0, 1, rotation))
	}

}

// SetActiveColorChannel sets the active color channel in all vertices contained within the VertexSelection to the channel with the
// specified index.
func (vs *VertexSelection) SetActiveColorChannel(channelIndex int) {

	vs.Mesh.ensureEnoughVertexColorChannels(channelIndex)

	for i := range vs.Indices {
		vs.Mesh.VertexActiveColorChannel[i] = channelIndex
	}

}

// ApplyMatrix applies a Matrix4 to the position of all vertices contained within the VertexSelection.
func (vs *VertexSelection) ApplyMatrix(matrix Matrix4) {

	for index := range vs.Indices {
		vs.Mesh.VertexPositions[index] = matrix.MultVec(vs.Mesh.VertexPositions[index])
	}

}

// Move moves all vertices contained within the VertexSelection by the provided x, y, and z values.
func (vs *VertexSelection) Move(x, y, z float64) {

	for index := range vs.Indices {

		vs.Mesh.VertexPositions[index].X += x
		vs.Mesh.VertexPositions[index].Y += y
		vs.Mesh.VertexPositions[index].Z += z

	}

}

// Move moves all vertices contained within the VertexSelection by the provided 3D vector.
func (vs *VertexSelection) MoveVec(vec Vector) {

	for index := range vs.Indices {

		vs.Mesh.VertexPositions[index].X += vec.X
		vs.Mesh.VertexPositions[index].Y += vec.Y
		vs.Mesh.VertexPositions[index].Z += vec.Z

	}

}

// NewCubeMesh creates a new Cube Mesh and gives it a new material (suitably named "Cube").
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

	t := (1.0 + math.Sqrt(5)) / 2

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

		a := Vector{vertices[index0].X, vertices[index0].Y, vertices[index0].Z, 0}
		b := Vector{vertices[index1].X, vertices[index1].Y, vertices[index1].Z, 0}
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

		v := Vector{vertices[i].X, vertices[i].Y, vertices[i].Z, 0}.Unit() // Make it spherical
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

// NewPlaneMesh creates a new plane Mesh with a new material (suitably named "Plane").
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

			tx := float64(x) / float64(vertexCountX-1)
			tz := float64(z) / float64(vertexCountZ-1)
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
func NewCylinderMesh(sideCount int, radius float64, createCaps bool) *Mesh {

	if sideCount < 3 {
		sideCount = 3
	}

	mesh := NewMesh("Cylinder")

	verts := []VertexInfo{}

	for i := 0; i < sideCount; i++ {

		pos := NewVector(radius, 1, 0)
		pos = pos.Rotate(0, 1, 0, float64(i)/float64(sideCount)*math.Pi*2)
		verts = append(verts, NewVertex(pos.X, pos.Y, pos.Z, 0, 0))

	}

	for i := 0; i < sideCount; i++ {

		pos := NewVector(radius, -1, 0)
		pos = pos.Rotate(0, 1, 0, float64(i)/float64(sideCount)*math.Pi*2)
		verts = append(verts, NewVertex(pos.X, pos.Y, pos.Z, 0, 0))

	}

	if createCaps {
		verts = append(verts, NewVertex(0, 1, 0, 0, 0))
		verts = append(verts, NewVertex(0, -1, 0, 0, 0))
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

// sortingTriangle is used specifically for sorting triangles when rendering. Less data means more data fits in cache,
// which means sorting is faster.
type sortingTriangle struct {
	Triangle *Triangle
	depth    float32
}

// A Triangle represents the smallest renderable object in Tetra3D. A triangle contains very little data, and is mainly used to help identify triads of vertices.
type Triangle struct {
	ID            uint16
	VertexIndices []int     // Vertex indices that compose the triangle
	MaxSpan       float64   // The maximum span from corner to corner of the triangle's dimensions; this is used in intersection testing.
	Center        Vector    // The untransformed center of the Triangle.
	Normal        Vector    // The physical normal of the triangle (i.e. the direction the triangle is facing). This is different from the visual normals of a triangle's vertices (i.e. a selection of vertices can have inverted normals to be see through, for example).
	MeshPart      *MeshPart // The specific MeshPart this Triangle belongs to.
}

// NewTriangle creates a new Triangle, and requires a reference to its owning MeshPart, along with its id within that MeshPart.
func NewTriangle(meshPart *MeshPart, id uint16, indices ...int) *Triangle {
	tri := &Triangle{
		ID:            id,
		MeshPart:      meshPart,
		VertexIndices: []int{indices[0], indices[1], indices[2]},
		Center:        Vector{0, 0, 0, 0},
	}
	return tri
}

// Clone clones the Triangle, keeping a reference to the same Material.
func (tri *Triangle) Clone() *Triangle {
	newTri := NewTriangle(tri.MeshPart, tri.ID, tri.VertexIndices...)
	newTri.MeshPart = tri.MeshPart
	newTri.Center = tri.Center
	newTri.Normal = tri.Normal
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
		Min: Vector{0, 0, 0, 0},
		Max: Vector{0, 0, 0, 0},
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

func (tri *Triangle) SharesVertexPositions(other *Triangle) []int {

	triIndices := tri.VertexIndices
	otherIndices := other.VertexIndices

	mesh := tri.MeshPart.Mesh
	otherMesh := other.MeshPart.Mesh

	shareCount := []int{-1, -1, -1}

	for i, index := range triIndices {
		for j, otherIndex := range otherIndices {
			if mesh.VertexPositions[index].Equals(otherMesh.VertexPositions[otherIndex]) {
				shareCount[i] = j
				break
			}
		}
	}

	if shareCount[0] == -1 && shareCount[1] == -1 && shareCount[2] == -1 {
		return nil
	}

	return shareCount

}

func (tri *Triangle) ForEachVertexIndex(vertFunc func(vertexIndex int)) {
	for _, vi := range tri.VertexIndices {
		vertFunc(vi)
	}
}

func calculateNormal(p1, p2, p3 Vector) Vector {

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

	sortingTriangles []sortingTriangle
}

// NewMeshPart creates a new MeshPart that renders using the specified Material.
func NewMeshPart(mesh *Mesh, material *Material) *MeshPart {
	return &MeshPart{
		Mesh:             mesh,
		Material:         material,
		sortingTriangles: []sortingTriangle{},
		TriangleStart:    math.MaxInt,
		VertexIndexStart: mesh.vertsAddStart,
	}
}

// Clone clones the MeshPart, returning the copy.
func (part *MeshPart) Clone() *MeshPart {

	newMP := NewMeshPart(part.Mesh, part.Material)

	for i := 0; i < len(part.sortingTriangles); i++ {
		newMP.sortingTriangles = append(newMP.sortingTriangles, sortingTriangle{
			Triangle: part.sortingTriangles[i].Triangle,
		})
	}

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

	newSorts := make([]sortingTriangle, 0, len(part.sortingTriangles))
	for _, tri := range part.sortingTriangles {
		newSorts = append(newSorts, sortingTriangle{
			Triangle: part.Mesh.Triangles[tri.Triangle.ID],
		})
	}
	part.sortingTriangles = newSorts

	mesh.MeshParts = append(mesh.MeshParts, part)

	part.Mesh = mesh

}

// func (part *MeshPart) allocateSortingBuffer(size int) {
// 	part.sortingTriangles = make([]sortingTriangle, size)
// }

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

		newTri := NewTriangle(part, uint16(mesh.triIndex), indices[i]+part.VertexIndexStart, indices[i+1]+part.VertexIndexStart, indices[i+2]+part.VertexIndexStart)
		newTri.RecalculateCenter()
		newTri.RecalculateNormal()
		part.sortingTriangles = append(part.sortingTriangles, sortingTriangle{
			Triangle: newTri,
		})
		mesh.Triangles = append(mesh.Triangles, newTri)
		mesh.triIndex++

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
func (part *MeshPart) primaryDimensions() (float64, float64) {

	points := make([]Vector, 0, len(part.Mesh.VertexPositions))

	part.ForEachVertexIndex(func(vertIndex int) { points = append(points, part.Mesh.VertexPositions[vertIndex]) }, false)

	dimensions := NewDimensionsFromPoints(points...)

	x := dimensions.Width()
	y := dimensions.Height()
	z := dimensions.Depth()

	w, h := 0.0, 0.0

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

type VertexInfo struct {
	ID                        int
	X, Y, Z                   float64
	U, V                      float64
	NormalX, NormalY, NormalZ float64
	Weights                   []float32
	Colors                    []Color
	ActiveColorChannel        int
	Bones                     []uint16
}

// GetVertexInfo returns a VertexInfo struct containing the vertex information for the vertex with the provided index.
func (mesh *Mesh) GetVertexInfo(vertexIndex int) VertexInfo {

	v := VertexInfo{
		ID:                 vertexIndex,
		X:                  mesh.VertexPositions[vertexIndex].X,
		Y:                  mesh.VertexPositions[vertexIndex].Y,
		Z:                  mesh.VertexPositions[vertexIndex].Z,
		U:                  mesh.VertexUVs[vertexIndex].X,
		V:                  mesh.VertexUVs[vertexIndex].Y,
		NormalX:            mesh.VertexNormals[vertexIndex].X,
		NormalY:            mesh.VertexNormals[vertexIndex].Y,
		NormalZ:            mesh.VertexNormals[vertexIndex].Z,
		Colors:             mesh.VertexColors[vertexIndex],
		ActiveColorChannel: mesh.VertexActiveColorChannel[vertexIndex],
		Bones:              mesh.VertexBones[vertexIndex],
		Weights:            mesh.VertexWeights[vertexIndex],
	}

	return v

}

// NewVertex creates a new vertex information struct, which is used to create new Triangles. VertexInfo is purely for getting data into
// Meshes' vertex buffers, so after creating the Triangle, VertexInfos can be safely discarded (i.e. the VertexInfo doesn't hold "power"
// over the vertex it represents after creation).
func NewVertex(x, y, z, u, v float64) VertexInfo {
	return VertexInfo{
		X:                  x,
		Y:                  y,
		Z:                  z,
		U:                  u,
		V:                  v,
		Weights:            []float32{},
		Colors:             make([]Color, 0, 8),
		ActiveColorChannel: -1,
		Bones:              []uint16{},
	}
}
