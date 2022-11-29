package tetra3d

import (
	"fmt"
	"log"
	"math"

	"github.com/kvartborg/vector"
)

// Dimensions represents the minimum and maximum spatial dimensions of a Mesh arranged in a 2-space Vector slice.
type Dimensions []vector.Vector

func (dim Dimensions) Clone() Dimensions {
	return Dimensions{
		dim[0].Clone(),
		dim[1].Clone(),
	}
}

// MaxDimension returns the maximum value from all of the axes in the Dimensions. For example, if the Dimensions have a min of [-1, -2, -2],
// and a max of [6, 1.5, 1], Max() will return 7 for the X axis, as it's the largest distance between all axes.
func (dim Dimensions) MaxDimension() float64 {
	return math.Max(math.Max(dim.Width(), dim.Height()), dim.Depth())
}

// MaxSpan returns the maximum span between the corners of the dimension set.
func (dim Dimensions) MaxSpan() float64 {
	return dim[1].Sub(dim[0]).Magnitude()
}

// Center returns the center point inbetween the two corners of the dimension set.
func (dim Dimensions) Center() vector.Vector {
	return vector.Vector{
		(dim[1][0] + dim[0][0]) / 2,
		(dim[1][1] + dim[0][1]) / 2,
		(dim[1][2] + dim[0][2]) / 2,
	}
}

// Width returns the total difference between the minimum and maximum X values.
func (dim Dimensions) Width() float64 {
	return dim[1][0] - dim[0][0]
}

// Height returns the total difference between the minimum and maximum Y values.
func (dim Dimensions) Height() float64 {
	return dim[1][1] - dim[0][1]
}

// Depth returns the total difference between the minimum and maximum Z values.
func (dim Dimensions) Depth() float64 {
	return dim[1][2] - dim[0][2]
}

// Limit limits the provided position vector to be within the dimensions set.
func (dim Dimensions) Limit(position vector.Vector) vector.Vector {

	if position[0] < dim[0][0] {
		position[0] = dim[0][0]
	} else if position[0] > dim[1][0] {
		position[0] = dim[1][0]
	}

	if position[1] < dim[0][1] {
		position[1] = dim[0][1]
	} else if position[1] > dim[1][1] {
		position[1] = dim[1][1]
	}

	if position[2] < dim[0][2] {
		position[2] = dim[0][2]
	} else if position[2] > dim[1][2] {
		position[2] = dim[1][2]
	}

	return position
}

// Inside returns if a position is inside a set of dimensions.
func (dim Dimensions) Inside(position vector.Vector) bool {

	if position[0] < dim[0][0] ||
		position[0] > dim[1][0] ||
		position[1] < dim[0][1] ||
		position[1] > dim[1][1] ||
		position[2] < dim[0][2] ||
		position[2] > dim[1][2] {
		return false
	}

	return true
}

func (dim Dimensions) Size() vector.Vector {
	return vector.Vector{dim.Width(), dim.Height(), dim.Depth()}
}

func (dim Dimensions) reform() {

	for i := 0; i < 3; i++ {

		if dim[0][i] > dim[1][i] {
			swap := dim[1][i]
			dim[1][i] = dim[0][i]
			dim[0][i] = swap
		}

	}

}

// NewDimensionsFromPoints creates a new Dimensions struct from the given series of positions.
func NewDimensionsFromPoints(points ...vector.Vector) Dimensions {
	dim := Dimensions{
		{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64},
		{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64},
	}

	for _, point := range points {

		if dim[0][0] > point[0] {
			dim[0][0] = point[0]
		}

		if dim[0][1] > point[1] {
			dim[0][1] = point[1]
		}

		if dim[0][2] > point[2] {
			dim[0][2] = point[2]
		}

		if dim[1][0] < point[0] {
			dim[1][0] = point[0]
		}

		if dim[1][1] < point[1] {
			dim[1][1] = point[1]
		}

		if dim[1][2] < point[2] {
			dim[1][2] = point[2]
		}

	}

	return dim
}

// Mesh represents a mesh that can be represented visually in different locations via Models. By default, a new Mesh has no MeshParts (so you would need to add one
// manually if you want to construct a Mesh via code).
type Mesh struct {
	Name    string
	library *Library // A reference to the Library this Mesh came from.

	MeshParts []*MeshPart // The various mesh parts (collections of triangles, rendered with a single material).
	Triangles []*Triangle // The various triangles composing the Mesh.
	Indices   []int       // The indices of each triangle
	triIndex  int

	// Vertices are stored as a struct-of-arrays for simplified and faster rendering.
	// Each vertex property (position, normal, UV, colors, weights, bones, etc) is stored
	// here and indexed in order of index.
	vertexTransforms         []vector.Vector
	VertexPositions          []vector.Vector
	VertexNormals            []vector.Vector
	vertexSkinnedNormals     []vector.Vector
	vertexSkinnedPositions   []vector.Vector
	VertexUVs                []vector.Vector
	VertexColors             [][]*Color
	VertexActiveColorChannel []int
	VertexWeights            [][]float32 // TODO: Replace this with [][8]float32 (or however many the maximum is for GLTF)
	VertexBones              [][]uint16  // TODO: Replace this with [][8]uint16 (or however many the maximum number of bones affecting a single vertex is for GLTF)

	vertexLights  []*Color
	vertsAddStart int
	vertsAddEnd   int

	VertexColorChannelNames map[string]int
	Dimensions              Dimensions
	Properties              *Properties
}

// NewMesh takes a name and a slice of *Vertex instances, and returns a new Mesh. If you provide *Vertex instances, the number must be divisible by 3,
// or NewMesh will panic.
func NewMesh(name string, verts ...VertexInfo) *Mesh {

	mesh := &Mesh{
		Name:                    name,
		MeshParts:               []*MeshPart{},
		Dimensions:              Dimensions{{0, 0, 0}, {0, 0, 0}},
		VertexColorChannelNames: map[string]int{},
		Properties:              NewProperties(),

		vertexTransforms:         []vector.Vector{},
		VertexPositions:          []vector.Vector{},
		VertexNormals:            []vector.Vector{},
		vertexSkinnedNormals:     []vector.Vector{},
		vertexSkinnedPositions:   []vector.Vector{},
		vertexLights:             []*Color{},
		VertexUVs:                []vector.Vector{},
		VertexColors:             [][]*Color{},
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
		newMesh.VertexPositions[i] = mesh.VertexPositions[i].Clone()
	}

	for i := range mesh.VertexNormals {
		newMesh.VertexNormals[i] = mesh.VertexNormals[i].Clone()
	}

	for i := range mesh.vertexLights {
		newMesh.vertexLights[i] = mesh.vertexLights[i].Clone()
	}

	for i := range mesh.VertexUVs {
		newMesh.VertexUVs[i] = mesh.VertexUVs[i].Clone()
	}

	for i := range mesh.VertexColors {
		newMesh.VertexColors[i] = make([]*Color, len(mesh.VertexColors[i]))
		for channelIndex := range mesh.VertexColors[i] {
			newMesh.VertexColors[i][channelIndex] = mesh.VertexColors[i][channelIndex].Clone()
		}
	}

	for c := range mesh.VertexActiveColorChannel {
		newMesh.VertexActiveColorChannel[c] = mesh.VertexActiveColorChannel[c]
	}

	for c := range mesh.VertexBones {
		for v := range mesh.VertexBones[c] {
			newMesh.VertexBones[c][v] = mesh.VertexBones[c][v]
		}
	}

	for c := range mesh.VertexWeights {
		for v := range mesh.VertexWeights[c] {
			newMesh.VertexWeights[c][v] = mesh.VertexWeights[c][v]
		}
	}

	for v := range mesh.vertexTransforms {
		newMesh.vertexTransforms[v] = mesh.vertexTransforms[v].Clone()
	}

	for v := range mesh.vertexSkinnedNormals {
		newMesh.vertexSkinnedNormals[v] = mesh.vertexSkinnedNormals[v].Clone()
	}

	for v := range mesh.vertexSkinnedPositions {
		newMesh.vertexSkinnedPositions[v] = mesh.vertexSkinnedPositions[v].Clone()
	}

	newMesh.Triangles = make([]*Triangle, 0, len(mesh.Triangles))

	for _, part := range mesh.MeshParts {
		newPart := part.Clone()

		part.ForEachTri(
			func(tri *Triangle) {
				newTri := tri.Clone()
				newTri.MeshPart = newPart
				newMesh.Triangles = append(newMesh.Triangles, newTri)
			},
		)

		newMesh.MeshParts = append(newMesh.MeshParts, newPart)
		newPart.AssignToMesh(newMesh)
	}

	for channelName, index := range mesh.VertexColorChannelNames {
		newMesh.VertexColorChannelNames[channelName] = index
	}

	newMesh.Dimensions = mesh.Dimensions.Clone()

	return newMesh
}

// allocateVertexBuffers allows us to allocate the slices for vertex properties all at once rather than resizing multiple times as
// we append to a slice and have its backing buffer automatically expanded (which is slower).
func (mesh *Mesh) allocateVertexBuffers(vertexCount int) {

	if cap(mesh.VertexPositions) >= vertexCount {
		return
	}

	mesh.VertexPositions = append(make([]vector.Vector, 0, vertexCount), mesh.VertexPositions...)

	mesh.VertexNormals = append(make([]vector.Vector, 0, vertexCount), mesh.VertexNormals...)

	mesh.vertexLights = append(make([]*Color, 0, vertexCount), mesh.vertexLights...)

	mesh.VertexUVs = append(make([]vector.Vector, 0, vertexCount), mesh.VertexUVs...)

	mesh.VertexColors = append(make([][]*Color, 0, vertexCount), mesh.VertexColors...)

	mesh.VertexActiveColorChannel = append(make([]int, 0, vertexCount), mesh.VertexActiveColorChannel...)

	mesh.VertexBones = append(make([][]uint16, 0, vertexCount), mesh.VertexBones...)

	mesh.VertexWeights = append(make([][]float32, 0, vertexCount), mesh.VertexWeights...)

	mesh.vertexTransforms = append(make([]vector.Vector, 0, vertexCount), mesh.vertexTransforms...)

	mesh.vertexSkinnedNormals = append(make([]vector.Vector, 0, vertexCount), mesh.vertexSkinnedNormals...)

	mesh.vertexSkinnedPositions = append(make([]vector.Vector, 0, vertexCount), mesh.vertexSkinnedPositions...)

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

		var base *Color

		if multiplicative {
			base = NewColor(1, 1, 1, 1)
		} else {
			base = NewColor(0, 0, 0, 1)
		}

		for _, c := range sourceChannels {

			if len(mesh.VertexColors[i]) <= c {
				continue
			}

			if multiplicative {
				base.Multiply(mesh.VertexColors[i][c])
			} else {
				base.Add(mesh.VertexColors[i][c])
			}

		}

		mesh.ensureEnoughVertexColorChannels(targetChannel)

		mesh.VertexColors[i][targetChannel] = base

	}

}

// SetVertexColor sets the specified vertex color for all vertices in the mesh for the target color channel.
func (mesh *Mesh) SetVertexColor(targetChannel int, color *Color) {
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
		mesh.VertexPositions = append(mesh.VertexPositions, vector.Vector{vertInfo.X, vertInfo.Y, vertInfo.Z})
		mesh.VertexNormals = append(mesh.VertexNormals, vector.Vector{vertInfo.NormalX, vertInfo.NormalY, vertInfo.NormalZ})
		mesh.VertexUVs = append(mesh.VertexUVs, vector.Vector{vertInfo.U, vertInfo.V})
		mesh.VertexColors = append(mesh.VertexColors, vertInfo.Colors)
		mesh.VertexActiveColorChannel = append(mesh.VertexActiveColorChannel, vertInfo.ActiveColorChannel)
		mesh.VertexBones = append(mesh.VertexBones, vertInfo.Bones)
		mesh.VertexWeights = append(mesh.VertexWeights, vertInfo.Weights)

		mesh.vertexLights = append(mesh.vertexLights, NewColor(0, 0, 0, 1))
		mesh.vertexTransforms = append(mesh.vertexTransforms, vector.Vector{0, 0, 0, 0}) // x, y, z, w
		mesh.vertexSkinnedNormals = append(mesh.vertexSkinnedNormals, vector.Vector{0, 0, 0})
		mesh.vertexSkinnedPositions = append(mesh.vertexSkinnedPositions, vector.Vector{0, 0, 0})

	}

}

// Library returns the Library from which this Mesh was loaded. If it was created through code, this function will return nil.
func (mesh *Mesh) Library() *Library {
	return mesh.library
}

// UpdateBounds updates the mesh's dimensions; call this after manually changing vertex positions.
func (mesh *Mesh) UpdateBounds() {

	mesh.Dimensions[1] = vector.Vector{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64}
	mesh.Dimensions[0] = vector.Vector{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}

	for _, position := range mesh.VertexPositions {

		if mesh.Dimensions[0][0] > position[0] {
			mesh.Dimensions[0][0] = position[0]
		}

		if mesh.Dimensions[0][1] > position[1] {
			mesh.Dimensions[0][1] = position[1]
		}

		if mesh.Dimensions[0][2] > position[2] {
			mesh.Dimensions[0][2] = position[2]
		}

		if mesh.Dimensions[1][0] < position[0] {
			mesh.Dimensions[1][0] = position[0]
		}

		if mesh.Dimensions[1][1] < position[1] {
			mesh.Dimensions[1][1] = position[1]
		}

		if mesh.Dimensions[1][2] < position[2] {
			mesh.Dimensions[1][2] = position[2]
		}

	}

}

// GetVertexInfo returns a VertexInfo struct containing the vertex information for the vertex with the provided index.
func (mesh *Mesh) GetVertexInfo(vertexIndex int) VertexInfo {

	v := VertexInfo{
		ID:                 vertexIndex,
		X:                  mesh.VertexPositions[vertexIndex][0],
		Y:                  mesh.VertexPositions[vertexIndex][1],
		Z:                  mesh.VertexPositions[vertexIndex][2],
		U:                  mesh.VertexUVs[vertexIndex][0],
		V:                  mesh.VertexUVs[vertexIndex][1],
		NormalX:            mesh.VertexNormals[vertexIndex][0],
		NormalY:            mesh.VertexNormals[vertexIndex][1],
		NormalZ:            mesh.VertexNormals[vertexIndex][2],
		Colors:             mesh.VertexColors[vertexIndex],
		ActiveColorChannel: mesh.VertexActiveColorChannel[vertexIndex],
		Bones:              mesh.VertexBones[vertexIndex],
		Weights:            mesh.VertexWeights[vertexIndex],
	}

	return v

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
	return NewVertexSelection(mesh)
}

// VertexSelection represents a selection of vertices on a Mesh.
type VertexSelection struct {
	Indices map[int]bool
	Mesh    *Mesh
}

// NewVertexSelection creates a new VertexSelection instance for the specified Mesh.
func NewVertexSelection(mesh *Mesh) *VertexSelection {
	return &VertexSelection{Indices: map[int]bool{}, Mesh: mesh}
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
func (vs *VertexSelection) SetColor(channelIndex int, color *Color) {

	vs.Mesh.ensureEnoughVertexColorChannels(channelIndex)

	for i := range vs.Indices {
		vs.Mesh.VertexColors[i][channelIndex].Set(color.ToFloat32s())
	}

}

// SetNormal sets the normal of all vertices contained within the VertexSelection to the provided normal vector.
func (vs *VertexSelection) SetNormal(normal vector.Vector) {

	for i := range vs.Indices {
		vs.Mesh.VertexNormals[i][0] = normal[0]
		vs.Mesh.VertexNormals[i][1] = normal[1]
		vs.Mesh.VertexNormals[i][2] = normal[2]
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

		x, y, z := fastMatrixMultVec(matrix, vs.Mesh.VertexPositions[index])
		vs.Mesh.VertexPositions[index][0] = x
		vs.Mesh.VertexPositions[index][1] = y
		vs.Mesh.VertexPositions[index][2] = z

	}

}

// Move moves all vertices contained within the VertexSelection by the provided x, y, and z values.
func (vs *VertexSelection) Move(x, y, z float64) {

	for index := range vs.Indices {

		vs.Mesh.VertexPositions[index][0] += x
		vs.Mesh.VertexPositions[index][1] += y
		vs.Mesh.VertexPositions[index][2] += z

	}

}

// Move moves all vertices contained within the VertexSelection by the provided 3D vector.
func (vs *VertexSelection) MoveVec(vec vector.Vector) {

	for index := range vs.Indices {

		vs.Mesh.VertexPositions[index][0] += vec[0]
		vs.Mesh.VertexPositions[index][1] += vec[1]
		vs.Mesh.VertexPositions[index][2] += vec[2]

	}

}

// NewCube creates a new Cube Mesh and gives it a new material (suitably named "Cube").
func NewCube() *Mesh {

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

// NewIcosphere creates a new icosphere Mesh of the specified detail level. Note that the UVs are left at {0,0} because I'm lazy.
func NewIcosphere(detailLevel int) *Mesh {

	// Code cribbed from http://blog.andreaskahler.com/2009/06/creating-icosphere-mesh-in-code.html, thank you very much Andreas!

	mesh := NewMesh("Icosphere")
	// part := mesh.AddMeshPart(NewMaterial("Icosphere"))

	// t := (1.0 + math.Sqrt(5)) / 2

	// v0 := NewVertex(-1, t, 0, 0, 0)
	// v1 := NewVertex(1, t, 0, 0, 0)
	// v2 := NewVertex(-1, -t, 0, 0, 0)
	// v3 := NewVertex(1, -t, 0, 0, 0)

	// v4 := NewVertex(0, -1, t, 0, 0)
	// v5 := NewVertex(0, 1, t, 0, 0)
	// v6 := NewVertex(0, -1, -t, 0, 0)
	// v7 := NewVertex(0, 1, -t, 0, 0)

	// v8 := NewVertex(t, 0, -1, 0, 0)
	// v9 := NewVertex(t, 0, 1, 0, 0)
	// v10 := NewVertex(-t, 0, -1, 0, 0)
	// v11 := NewVertex(-t, 0, 1, 0, 0)

	// indices := []uint32{
	// 	0, 11, 5,
	// 	0, 5, 1,
	// 	0, 1, 7,
	// 	0, 7, 10,
	// 	0, 10, 11,

	// 	1, 5, 9,
	// 	5, 11, 4,
	// 	11, 10, 2,
	// 	10, 7, 6,
	// 	7, 1, 8,

	// 	3, 9, 4,
	// 	3, 4, 2,
	// 	3, 2, 6,
	// 	3, 6, 8,
	// 	3, 8, 9,

	// 	4, 9, 5,
	// 	2, 4, 11,
	// 	6, 2, 10,
	// 	8, 6, 7,
	// 	9, 8, 1,
	// }

	// vertices := []VertexInfo{
	// 	v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11,
	// }

	// type cacheStorage struct {
	// 	v0, v1 int
	// }

	// cache := map[cacheStorage]int{}

	// getMiddlePoint := func(index0, index1 int) int {

	// 	if v, exists := cache[cacheStorage{index0, index1}]; exists {
	// 		return v
	// 	}

	// 	a := vector.Vector{vertices[index0].X, vertices[index0].Y, vertices[index0].Z}
	// 	b := vector.Vector{vertices[index1].X, vertices[index1].Y, vertices[index1].Z}
	// 	midA := a.Add(b.Sub(a).Scale(0.5))
	// 	midAVert := NewVertex(midA[0], midA[1], midA[2], 0, 0)
	// 	vertices = append(vertices, midAVert)

	// 	cache[cacheStorage{index0, index1}] = len(vertices)
	// 	return len(vertices)

	// }

	// if detailLevel > 0 {

	// 	for i := 0; i < detailLevel+1; i++ {

	// 		newFaces := make([]VertexInfo, 0, len(vertices)*4)
	// 		newIndices := make([]int, 0, len(indices)*4)

	// 		for triIndex := 0; triIndex < len(vertices); triIndex += 3 {

	// 			a := vector.Vector{vertices[triIndex].X, vertices[triIndex].Y, vertices[triIndex].Z}
	// 			b := vector.Vector{vertices[triIndex+1].X, vertices[triIndex+1].Y, vertices[triIndex+1].Z}
	// 			c := vector.Vector{vertices[triIndex+2].X, vertices[triIndex+2].Y, vertices[triIndex+2].Z}

	// 			midA := a.Add(b.Sub(a).Scale(0.5))
	// 			midB := b.Add(c.Sub(b).Scale(0.5))
	// 			midC := c.Add(a.Sub(c).Scale(0.5))

	// 			midAVert := NewVertex(midA[0], midA[1], midA[2], 0, 0)
	// 			midBVert := NewVertex(midB[0], midB[1], midB[2], 0, 0)
	// 			midCVert := NewVertex(midC[0], midC[1], midC[2], 0, 0)

	// 			newFaces = append(newFaces,
	// 				vertices[triIndex], midAVert, midCVert,
	// 				vertices[triIndex+1], midBVert, midAVert,
	// 				vertices[triIndex+2], midCVert, midBVert,
	// 				midAVert, midBVert, midCVert,
	// 			)
	// 			newIndices = append(newIndices)

	// 		}

	// 		vertices = newFaces

	// 	}

	// }

	// for i := range vertices {

	// 	v := vector.Vector{vertices[i].X, vertices[i].Y, vertices[i].Z}.Unit() // Make it spherical
	// 	vertices[i].X = v[0]
	// 	vertices[i].Y = v[1]
	// 	vertices[i].Z = v[2]

	// 	vertices[i].U = 0
	// 	vertices[i].V = 0
	// 	if i%3 == 1 {
	// 		vertices[i].U = 1
	// 	} else if i%3 == 2 {
	// 		vertices[i].V = 1
	// 	}

	// }

	// part.AddTriangles(
	// 	indices,
	// 	vertices...)

	mesh.AutoNormal()

	mesh.UpdateBounds()

	return mesh
}

// NewPlane creates a new plane Mesh and gives it a new material (suitably named "Plane").
func NewPlane() *Mesh {

	mesh := NewMesh("Plane",
		NewVertex(1, 0, -1, 1, 0),
		NewVertex(-1, 0, -1, 0, 0),
		NewVertex(1, 0, 1, 1, 1),
		NewVertex(-1, 0, 1, 0, 1),
	)

	mesh.AddMeshPart(NewMaterial("Plane"),
		0, 1, 2,
		1, 3, 2,
	)

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
	VertexIndices []int         // Vertex indices that compose the triangle
	MaxSpan       float64       // The maximum span from corner to corner of the triangle's dimensions; this is used in intersection testing.
	Center        vector.Vector // The untransformed center of the Triangle.
	Normal        vector.Vector // The physical normal of the triangle (i.e. the direction the triangle is facing). This is different from the visual normals of a triangle's vertices (i.e. a selection of vertices can have inverted normals to be see through, for example).
	MeshPart      *MeshPart     // The specific MeshPart this Triangle belongs to.
}

// NewTriangle creates a new Triangle, and requires a reference to its owning MeshPart, along with its id within that MeshPart.
func NewTriangle(meshPart *MeshPart, id uint16, indices ...int) *Triangle {
	tri := &Triangle{
		ID:            id,
		MeshPart:      meshPart,
		VertexIndices: []int{indices[0], indices[1], indices[2]},
		Center:        vector.Vector{0, 0, 0},
	}
	return tri
}

// Clone clones the Triangle, keeping a reference to the same Material.
func (tri *Triangle) Clone() *Triangle {
	newTri := NewTriangle(tri.MeshPart, tri.ID, tri.VertexIndices...)
	newTri.MeshPart = tri.MeshPart
	newTri.Center = tri.Center.Clone()
	newTri.Normal = tri.Normal.Clone()
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

	tri.Center[0] = (verts[indices[0]][0] + verts[indices[1]][0] + verts[indices[2]][0]) / 3
	tri.Center[1] = (verts[indices[0]][1] + verts[indices[1]][1] + verts[indices[2]][1]) / 3
	tri.Center[2] = (verts[indices[0]][2] + verts[indices[1]][2] + verts[indices[2]][2]) / 3

	// Determine the maximum span of the triangle; this is done for bounds checking, since we can reject triangles early if we
	// can easily tell we're too far away from them.
	dim := Dimensions{{0, 0, 0}, {0, 0, 0}}

	for i := 0; i < 3; i++ {

		if dim[0][0] > verts[indices[i]][0] {
			dim[0][0] = verts[indices[i]][0]
		}

		if dim[0][1] > verts[indices[i]][1] {
			dim[0][1] = verts[indices[i]][1]
		}

		if dim[0][2] > verts[indices[i]][2] {
			dim[0][2] = verts[indices[i]][2]
		}

		if dim[1][0] < verts[indices[i]][0] {
			dim[1][0] = verts[indices[i]][0]
		}

		if dim[1][1] < verts[indices[i]][1] {
			dim[1][1] = verts[indices[i]][1]
		}

		if dim[1][2] < verts[indices[i]][2] {
			dim[1][2] = verts[indices[i]][2]
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
			if vectorsEqual(mesh.VertexPositions[index], otherMesh.VertexPositions[otherIndex]) {
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

func calculateNormal(p1, p2, p3 vector.Vector) vector.Vector {

	v0 := p2.Sub(p1)
	v1 := p3.Sub(p2)

	cross, _ := v0.Cross(v1)
	return cross.Unit()

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

}

// func (part *MeshPart) allocateSortingBuffer(size int) {
// 	part.sortingTriangles = make([]sortingTriangle, size)
// }

func (part *MeshPart) ForEachTri(triFunc func(tri *Triangle)) {
	for i := part.TriangleStart; i <= part.TriangleEnd; i++ {
		triFunc(part.Mesh.Triangles[i])
	}
}

func (part *MeshPart) ForEachVertexIndex(vertFunc func(vertIndex int)) {
	for i := part.VertexIndexStart; i < part.VertexIndexEnd; i++ {
		vertFunc(i)
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
				vert := mesh.VertexPositions[index]
				x, y, z := fastMatrixMultVec(matrix, vert)
				vert[0] = x
				vert[1] = y
				vert[2] = z
			}

		},
	)

}

type VertexInfo struct {
	ID                        int
	X, Y, Z                   float64
	U, V                      float64
	NormalX, NormalY, NormalZ float64
	Weights                   []float32
	Colors                    []*Color
	ActiveColorChannel        int
	Bones                     []uint16
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
		Colors:             make([]*Color, 0, 8),
		ActiveColorChannel: -1,
		Bones:              []uint16{},
	}
}
