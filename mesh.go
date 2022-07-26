package tetra3d

import (
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
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

func (dim Dimensions) Size() vector.Vector {
	return vector.Vector{dim.Width(), dim.Height(), dim.Depth()}
}

// Mesh represents a mesh that can be represented visually in different locations via Models. By default, a new Mesh has no MeshParts (so you would need to add one
// manually if you want to construct a Mesh via code).
type Mesh struct {
	Name    string
	library *Library // A reference to the Library this Mesh came from.

	MeshParts []*MeshPart // The various mesh parts (collections of triangles, rendered with a single material).
	Triangles []*Triangle // The various triangles composing the Mesh.

	// Vertices are stored as a struct-of-arrays for simplified and faster rendering.
	// Each vertex property (position, normal, UV, colors, weights, bones, etc) is stored
	// here and indexed in order of triangle ID * 3 + vertex (so the first triangle, 0, has
	// the vertices 0, 1, and 2, while the 10th triangle would have the vertices 30, 31, and 32).
	vertexTransforms         []vector.Vector
	VertexPositions          []vector.Vector
	VertexNormals            []vector.Vector
	vertexSkinnedNormals     []vector.Vector
	vertexSkinnedPositions   []vector.Vector
	VertexUVs                []vector.Vector
	VertexColors             [][]*Color
	VertexActiveColorChannel []int
	VertexWeights            [][]float32
	VertexBones              [][]uint16
	VertexCount              int
	VertexMax                int

	VertexColorChannelNames map[string]int
	Dimensions              Dimensions
	triIndex                int
	Tags                    *Tags
}

// NewMesh takes a name and a slice of *Vertex instances, and returns a new Mesh. If you provide *Vertex instances, the number must be divisible by 3,
// or NewMesh will panic.
func NewMesh(name string) *Mesh {

	mesh := &Mesh{
		Name:                    name,
		MeshParts:               []*MeshPart{},
		Dimensions:              Dimensions{{0, 0, 0}, {0, 0, 0}},
		VertexColorChannelNames: map[string]int{},
		triIndex:                0,
		Tags:                    NewTags(),

		vertexTransforms:         []vector.Vector{},
		VertexPositions:          []vector.Vector{},
		VertexNormals:            []vector.Vector{},
		vertexSkinnedNormals:     []vector.Vector{},
		vertexSkinnedPositions:   []vector.Vector{},
		VertexUVs:                []vector.Vector{},
		VertexColors:             [][]*Color{},
		VertexActiveColorChannel: []int{},
		VertexBones:              [][]uint16{},
		VertexWeights:            [][]float32{},
	}

	return mesh

}

// Clone clones the Mesh, creating a new Mesh that has cloned MeshParts.
func (mesh *Mesh) Clone() *Mesh {
	newMesh := NewMesh(mesh.Name)
	newMesh.library = mesh.library
	newMesh.Tags = mesh.Tags.Clone()
	newMesh.triIndex = mesh.triIndex

	newMesh.allocateVertexBuffers(mesh.VertexMax)
	copy(newMesh.VertexPositions, mesh.VertexPositions)
	copy(newMesh.VertexNormals, mesh.VertexNormals)
	copy(newMesh.VertexUVs, mesh.VertexUVs)
	copy(newMesh.VertexColors, mesh.VertexColors)
	copy(newMesh.VertexActiveColorChannel, mesh.VertexActiveColorChannel)
	copy(newMesh.VertexBones, mesh.VertexBones)
	copy(newMesh.VertexWeights, mesh.VertexWeights)

	newMesh.VertexCount = mesh.VertexCount
	newMesh.VertexMax = mesh.VertexMax

	for _, part := range mesh.MeshParts {
		newPart := part.Clone()
		newMesh.MeshParts = append(newMesh.MeshParts, newPart)
		newPart.Mesh = mesh
	}

	for channelName, index := range mesh.VertexColorChannelNames {
		newMesh.VertexColorChannelNames[channelName] = index
	}

	return newMesh
}

// allocateVertexBuffers allows us to allocate the slices for vertex properties all at once rather than resizing multiple times as
// we append to a slice and have its backing buffer automatically expanded (which is slower).
func (mesh *Mesh) allocateVertexBuffers(size int) {

	newVP := make([]vector.Vector, size)
	copy(newVP, mesh.VertexPositions)
	mesh.VertexPositions = newVP

	newVN := make([]vector.Vector, size)
	copy(newVN, mesh.VertexNormals)
	mesh.VertexNormals = newVN

	newVUVs := make([]vector.Vector, size)
	copy(newVUVs, mesh.VertexUVs)
	mesh.VertexUVs = newVUVs

	newVC := make([][]*Color, size)
	copy(newVC, mesh.VertexColors)
	mesh.VertexColors = newVC

	newActiveChannel := make([]int, size)
	copy(newActiveChannel, mesh.VertexActiveColorChannel)
	mesh.VertexActiveColorChannel = newActiveChannel

	newBones := make([][]uint16, size)
	copy(newBones, mesh.VertexBones)
	mesh.VertexBones = newBones

	newWeights := make([][]float32, size)
	copy(newWeights, mesh.VertexWeights)
	mesh.VertexWeights = newWeights

	newTransforms := make([]vector.Vector, size)
	copy(newTransforms, mesh.vertexTransforms)
	mesh.vertexTransforms = newTransforms

	newNormals := make([]vector.Vector, size)
	copy(newNormals, mesh.vertexSkinnedNormals)
	mesh.vertexSkinnedNormals = newNormals

	newPositions := make([]vector.Vector, size)
	copy(newPositions, mesh.vertexSkinnedPositions)
	mesh.vertexSkinnedPositions = newPositions

	mesh.VertexMax = size

}

// AddMeshPart allows you to add a new MeshPart to the Mesh with the given Material (with a nil Material reference also being valid).
func (mesh *Mesh) AddMeshPart(material *Material) *MeshPart {
	mp := NewMeshPart(mesh, material)
	mesh.MeshParts = append(mesh.MeshParts, mp)
	return mp
}

// FindMeshPart allows you to retrieve a MeshPart by its material's name. If no material with the provided name is given, the function returns nil.
func (mesh *Mesh) FindMeshPart(materialName string) *MeshPart {
	for _, mp := range mesh.MeshParts {
		if mp.Material.Name == materialName {
			return mp
		}
	}
	return nil
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

// SelectColorInChannel selects all vertices in the Mesh that have a non-pure black color in a color channel
// with the specified name.
func (vs *VertexSelection) SelectColorInChannel(channelName string) *VertexSelection {

	colorChannel, exists := vs.Mesh.VertexColorChannelNames[channelName]
	if !exists {
		return vs
	}

	for vertexIndex := range vs.Mesh.VertexColors {

		// if colorChannel < len(vs.Mesh.VertexColors[vertexIndex])-1 {
		// 	return false
		// }

		color := vs.Mesh.VertexColors[vertexIndex][colorChannel]

		if color.R > 0.01 || color.G > 0.01 || color.B > 0.01 {
			vs.Indices[vertexIndex] = true
		}

	}

	return vs

}

// SelectAll selects all vertices on the source Mesh.
func (vs *VertexSelection) SelectAll() *VertexSelection {

	for i := 0; i < vs.Mesh.VertexMax; i++ {
		vs.Indices[i] = true
	}

	return vs
}

// SelectMeshPart selects all vertices in the Mesh belonging to the specified MeshPart.
func (vs *VertexSelection) SelectMeshPart(meshPart *MeshPart) *VertexSelection {

	for i := meshPart.TriangleStart * 3; i < meshPart.TriangleEnd*3; i++ {
		vs.Indices[i] = true
	}

	return vs

}

// SetColor sets the color of the specified channel in all vertices contained within the VertexSelection to the provided Color.
func (vs *VertexSelection) SetColor(channelName string, color *Color) {

	colorChannel, exists := vs.Mesh.VertexColorChannelNames[channelName]
	if !exists {
		return
	}

	for i := range vs.Indices {
		vs.Mesh.VertexColors[i][colorChannel].Set(color.ToFloat32s())
	}

}

// SetActiveColorChannel sets the active color channel in all vertices contained within the VertexSelection to the named channel.
func (vs *VertexSelection) SetActiveColorChannel(channelName string) {

	colorChannel, exists := vs.Mesh.VertexColorChannelNames[channelName]
	if !exists {
		return
	}

	for i := range vs.Indices {
		vs.Mesh.VertexActiveColorChannel[i] = colorChannel
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

	mesh := NewMesh("Cube")
	mp := mesh.AddMeshPart(NewMaterial("Cube"))
	mp.AddTriangles(
		// Top

		NewVertex(-1, 1, -1, 0, 0),
		NewVertex(1, 1, 1, 1, 1),
		NewVertex(1, 1, -1, 1, 0),

		NewVertex(-1, 1, 1, 0, 1),
		NewVertex(1, 1, 1, 1, 1),
		NewVertex(-1, 1, -1, 0, 0),

		// Bottom

		NewVertex(1, -1, -1, 1, 0),
		NewVertex(1, -1, 1, 1, 1),
		NewVertex(-1, -1, -1, 0, 0),

		NewVertex(-1, -1, -1, 0, 0),
		NewVertex(1, -1, 1, 1, 1),
		NewVertex(-1, -1, 1, 0, 1),

		// Front

		NewVertex(-1, 1, 1, 0, 0),
		NewVertex(1, -1, 1, 1, 1),
		NewVertex(1, 1, 1, 1, 0),

		NewVertex(-1, -1, 1, 0, 1),
		NewVertex(1, -1, 1, 1, 1),
		NewVertex(-1, 1, 1, 0, 0),

		// Back

		NewVertex(1, 1, -1, 1, 0),
		NewVertex(1, -1, -1, 1, 1),
		NewVertex(-1, 1, -1, 0, 0),

		NewVertex(-1, 1, -1, 0, 0),
		NewVertex(1, -1, -1, 1, 1),
		NewVertex(-1, -1, -1, 0, 1),

		// Right

		NewVertex(1, 1, -1, 1, 0),
		NewVertex(1, 1, 1, 1, 1),
		NewVertex(1, -1, -1, 0, 0),

		NewVertex(1, -1, -1, 0, 0),
		NewVertex(1, 1, 1, 1, 1),
		NewVertex(1, -1, 1, 0, 1),

		// Left

		NewVertex(-1, -1, -1, 0, 0),
		NewVertex(-1, 1, 1, 1, 1),
		NewVertex(-1, 1, -1, 1, 0),

		NewVertex(-1, -1, 1, 0, 1),
		NewVertex(-1, 1, 1, 1, 1),
		NewVertex(-1, -1, -1, 0, 0),
	)

	mesh.UpdateBounds()

	return mesh

}

// NewPlane creates a new plane Mesh and gives it a new material (suitably named "Plane").
func NewPlane() *Mesh {

	mesh := NewMesh("Plane")
	part := mesh.AddMeshPart(NewMaterial("Plane"))
	part.AddTriangles(
		NewVertex(1, 0, -1, 1, 0),
		NewVertex(-1, 0, -1, 0, 0),
		NewVertex(1, 0, 1, 1, 1),

		NewVertex(-1, 0, -1, 0, 0),
		NewVertex(-1, 0, 1, 0, 1),
		NewVertex(1, 0, 1, 1, 1),
	)

	mesh.UpdateBounds()

	return mesh

}

func NewWeirdDebuggingStatueThing() *Mesh {

	mesh := NewMesh("Weird Statue")

	part := mesh.AddMeshPart(NewMaterial("Weird Statue"))

	part.AddTriangles(

		NewVertex(1, 0, -1, 1, 0),
		NewVertex(-1, 0, -1, 0, 0),
		NewVertex(1, 0, 1, 1, 1),

		NewVertex(-1, 0, -1, 0, 0),
		NewVertex(-1, 0, 1, 0, 1),
		NewVertex(1, 0, 1, 1, 1),

		NewVertex(-1, 2, -1, 0, 0),
		NewVertex(1, 2, 1, 1, 1),
		NewVertex(1, 0, -1, 1, 0),

		NewVertex(-1, 0, 1, 0, 1),
		NewVertex(1, 2, 1, 1, 1),
		NewVertex(-1, 2, -1, 0, 0),
	)

	mesh.UpdateBounds()

	return mesh

}

// sortingTriangle is used specifically for sorting triangles when rendering. Less data means more data fits in cache,
// which means sorting is faster.
type sortingTriangle struct {
	ID       int
	depth    float32
	rendered bool
}

// A Triangle represents the smallest renderable object in Tetra3D. A triangle contains very little data, and is mainly used to help identify triads of vertices.
type Triangle struct {
	ID int // Unique identifier number (index) in the Mesh. You can use the ID to find a triangle's vertices
	// using the formula: Mesh.VertexPositions[TriangleIndex*3+i], with i being the index of the vertex in the triangle
	// (so either 0, 1, or 2).
	MaxSpan  float64       // The maximum span from corner to corner of the triangle's dimensions; this is used in intersection testing.
	Center   vector.Vector // The untransformed center of the Triangle.
	Normal   vector.Vector // The physical normal of the triangle (i.e. the direction the triangle is facing). This is different from the visual normals of a triangle's vertices (i.e. a selection of vertices can have inverted normals to be see through, for example).
	MeshPart *MeshPart     // The specific MeshPart this Triangle belongs to.
}

// NewTriangle creates a new Triangle, and requires a reference to its owning MeshPart, along with its id within that MeshPart.
func NewTriangle(meshPart *MeshPart, id int) *Triangle {
	tri := &Triangle{
		MeshPart: meshPart,
		ID:       id,
		Center:   vector.Vector{0, 0, 0},
	}
	return tri
}

// Clone clones the Triangle, keeping a reference to the same Material.
func (tri *Triangle) Clone() *Triangle {
	newTri := NewTriangle(tri.MeshPart, tri.ID)
	newTri.MeshPart = tri.MeshPart
	newTri.Center = tri.Center.Clone()
	newTri.Normal = tri.Normal.Clone()
	return newTri
}

// RecalculateCenter recalculates the center for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position.
func (tri *Triangle) RecalculateCenter() {

	verts := tri.MeshPart.Mesh.VertexPositions

	tri.Center[0] = (verts[tri.ID*3][0] + verts[tri.ID*3+1][0] + verts[tri.ID*3+2][0]) / 3
	tri.Center[1] = (verts[tri.ID*3][1] + verts[tri.ID*3+1][1] + verts[tri.ID*3+2][1]) / 3
	tri.Center[2] = (verts[tri.ID*3][2] + verts[tri.ID*3+1][2] + verts[tri.ID*3+2][2]) / 3

	// Determine the maximum span of the triangle; this is done for bounds checking, since we can reject triangles early if we
	// can easily tell we're too far away from them.
	dim := Dimensions{{0, 0, 0}, {0, 0, 0}}

	for i := 0; i < 3; i++ {

		if dim[0][0] > verts[tri.ID*3+i][0] {
			dim[0][0] = verts[tri.ID*3+i][0]
		}

		if dim[0][1] > verts[tri.ID*3+i][1] {
			dim[0][1] = verts[tri.ID*3+i][1]
		}

		if dim[0][2] > verts[tri.ID*3+i][2] {
			dim[0][2] = verts[tri.ID*3+i][2]
		}

		if dim[1][0] < verts[tri.ID*3+i][0] {
			dim[1][0] = verts[tri.ID*3+i][0]
		}

		if dim[1][1] < verts[tri.ID*3+i][1] {
			dim[1][1] = verts[tri.ID*3+i][1]
		}

		if dim[1][2] < verts[tri.ID*3+i][2] {
			dim[1][2] = verts[tri.ID*3+i][2]
		}

	}

	tri.MaxSpan = dim.MaxSpan()

}

// RecalculateNormal recalculates the physical normal for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position. Also note that vertex normals (visual normals) are automatically set when loading Meshes from model files.
func (tri *Triangle) RecalculateNormal() {
	verts := tri.MeshPart.Mesh.VertexPositions
	tri.Normal = calculateNormal(verts[tri.ID*3], verts[tri.ID*3+1], verts[tri.ID*3+2])
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
	TriangleStart    int
	TriangleEnd      int
	sortingTriangles []sortingTriangle
}

// NewMeshPart creates a new MeshPart that renders using the specified Material.
func NewMeshPart(mesh *Mesh, material *Material) *MeshPart {
	return &MeshPart{
		Mesh:             mesh,
		Material:         material,
		TriangleStart:    -1,
		TriangleEnd:      -1,
		sortingTriangles: []sortingTriangle{},
	}
}

// Clone clones the MeshPart, returning the copy.
func (part *MeshPart) Clone() *MeshPart {
	newMP := &MeshPart{
		Mesh:          part.Mesh,
		Material:      part.Material,
		TriangleStart: part.TriangleStart,
		TriangleEnd:   part.TriangleEnd,
	}
	newMP.Material = part.Material
	return newMP
}

// func (part *MeshPart) allocateSortingBuffer(size int) {
// 	part.sortingTriangles = make([]sortingTriangle, size)
// }

// AddTriangles adds triangles to the MeshPart using the provided VertexInfo slice. Note that
func (part *MeshPart) AddTriangles(verts ...VertexInfo) {

	mesh := part.Mesh

	if part.TriangleEnd > -1 && part.TriangleEnd < mesh.triIndex {
		panic("Error: Cannot add triangles to MeshPart non-contiguously (i.e. partA.AddTriangles(), partB.AddTriangles(), partA.AddTriangles() ).")
	}

	if len(verts) == 0 || len(verts)%3 > 0 {
		panic("Error: MeshPart.AddTriangles() not given enough vertices to construct complete triangles (i.e. multiples of 3 vertices).")
	}

	if part.TriangleStart < 0 {
		part.TriangleStart = mesh.triIndex
	}

	if mesh.triIndex*3+len(verts) > part.Mesh.VertexMax {
		part.Mesh.allocateVertexBuffers(part.Mesh.VertexMax + len(verts))
	}

	for i := 0; i < len(verts); i += 3 {

		for j := 0; j < 3; j++ {
			vertInfo := verts[i+j]
			index := (mesh.triIndex * 3) + j
			mesh.VertexPositions[index] = vector.Vector{vertInfo.X, vertInfo.Y, vertInfo.Z}
			mesh.VertexNormals[index] = vector.Vector{vertInfo.NormalX, vertInfo.NormalY, vertInfo.NormalZ}
			mesh.VertexUVs[index] = vector.Vector{vertInfo.U, vertInfo.V}
			mesh.VertexColors[index] = vertInfo.Colors
			mesh.VertexActiveColorChannel[index] = vertInfo.ActiveColorChannel
			mesh.VertexBones[index] = vertInfo.Bones
			mesh.VertexWeights[index] = vertInfo.Weights

			mesh.vertexTransforms[index] = vector.Vector{0, 0, 0, 0}
			mesh.vertexSkinnedNormals[index] = vector.Vector{0, 0, 0}
			mesh.vertexSkinnedPositions[index] = vector.Vector{0, 0, 0}
		}

		newTri := NewTriangle(part, mesh.triIndex)
		newTri.RecalculateCenter()
		newTri.RecalculateNormal()
		part.sortingTriangles = append(part.sortingTriangles, sortingTriangle{
			ID: mesh.triIndex,
		})
		mesh.Triangles = append(mesh.Triangles, newTri)
		mesh.triIndex++
		mesh.VertexCount += 3

	}

	if part.TriangleCount() >= ebiten.MaxIndicesNum/3 {
		matName := "nil"
		if part.Material != nil {
			matName = part.Material.Name
		}
		log.Println("warning: mesh [" + part.Mesh.Name + "] has part with material named [" + matName + "], which has " + fmt.Sprintf("%d", part.TriangleCount()) + " triangles. This exceeds the renderable maximum of 21845 triangles total for one MeshPart; please break up the mesh into multiple MeshParts using materials, or split it up into multiple models. Otherwise, the game will crash if it renders over the maximum number of triangles.")
	}

	part.TriangleEnd = mesh.triIndex

}

// TriangleCount returns the total number of triangles in the MeshPart, specifically.
func (part *MeshPart) TriangleCount() int {
	return part.TriangleEnd - part.TriangleStart + 1
}

// func (part *MeshPart) ApplyMatrix(matrix Matrix4) {
// 	mesh := part.Mesh
// 	for triIndex := part.TriangleStart; triIndex < part.TriangleEnd; triIndex++ {
// 		for i := 0; i < 3; i++ {
// 			x, y, z := fastMatrixMultVec(matrix, mesh.VertexPositions[triIndex*3+i])
// 			vert := mesh.VertexPositions[triIndex*3+i]
// 			vert[0] = x
// 			vert[1] = y
// 			vert[2] = z
// 		}
// 	}
// }

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
