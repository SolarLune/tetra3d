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

// func (dim Dimensions) reform() {

// 	for i := 0; i < 3; i++ {

// 		if dim[0][i] > dim[1][i] {
// 			swap := dim[1][i]
// 			dim[1][i] = dim[0][i]
// 			dim[0][i] = swap
// 		}

// 	}

// }

type Stride struct {
	data     *MeshBufferData
	Start    int
	Capacity int
}

func newStride(data *MeshBufferData, cap int) *Stride {

	stride := &Stride{
		data:     data,
		Start:    data.lastStart,
		Capacity: cap,
	}
	data.lastStart += cap
	data.totalStrideLength += cap
	return stride

}

func (stride *Stride) UV(vertexIndex int) (u, v float32) {
	total := stride.data.totalStrideLength
	u = stride.data.Data[(total*vertexIndex)+stride.Start]
	v = stride.data.Data[(total*vertexIndex)+stride.Start+1]
	return
}

func (stride *Stride) XYZ(vertexIndex int) (x, y, z float32) {
	total := stride.data.totalStrideLength
	x = stride.data.Data[(total*vertexIndex)+stride.Start]
	y = stride.data.Data[(total*vertexIndex)+stride.Start+1]
	z = stride.data.Data[(total*vertexIndex)+stride.Start+2]
	return
}

func (stride *Stride) XYZW(vertexIndex int) (x, y, z, w float32) {
	total := stride.data.totalStrideLength
	x = stride.data.Data[(total*vertexIndex)+stride.Start]
	y = stride.data.Data[(total*vertexIndex)+stride.Start+1]
	z = stride.data.Data[(total*vertexIndex)+stride.Start+2]
	w = stride.data.Data[(total*vertexIndex)+stride.Start+3]
	return
}

func (stride *Stride) RGBA(vertexIndex, channelIndex int) (r, g, b, a float32) {
	total := stride.data.totalStrideLength
	r = stride.data.Data[(total*vertexIndex)+stride.Start+(channelIndex*4)]
	g = stride.data.Data[(total*vertexIndex)+stride.Start+(channelIndex*4)+1]
	b = stride.data.Data[(total*vertexIndex)+stride.Start+(channelIndex*4)+2]
	a = stride.data.Data[(total*vertexIndex)+stride.Start+(channelIndex*4)+3]
	return
}

func (stride *Stride) Set(vertexIndex int, values ...float32) {
	total := stride.data.totalStrideLength
	for i := 0; i < len(values); i++ {
		stride.data.Data[(total*vertexIndex)+stride.Start+i] = values[i]
	}
}

func (stride *Stride) SetRGBA(vertexIndex, channelIndex int, values ...float32) {
	total := stride.data.totalStrideLength
	for i := 0; i < len(values); i++ {
		stride.data.Data[(total*vertexIndex)+stride.Start+(channelIndex*4)+i] = values[i]
	}
}

type MeshBufferData struct {
	Mesh                     *Mesh
	Data                     []float32
	BoneData                 [][]uint16
	VertexActiveColorChannel []int

	Position        *Stride
	Transform       *Stride
	Normal          *Stride
	UV              *Stride
	VertexColor     *Stride
	SkinnedPosition *Stride
	SkinnedNormal   *Stride
	Weights         *Stride

	totalStrideLength int
	lastStart         int
}

func NewMeshBufferData(mesh *Mesh) *MeshBufferData {

	mbd := &MeshBufferData{
		Mesh:     mesh,
		Data:     []float32{},
		BoneData: [][]uint16{},
	}

	mbd.Position = newStride(mbd, 3)
	mbd.Transform = newStride(mbd, 4)
	mbd.Normal = newStride(mbd, 3)
	mbd.UV = newStride(mbd, 2)
	mbd.VertexColor = newStride(mbd, 4*4)

	return mbd

}

func (md *MeshBufferData) Clone() *MeshBufferData {
	newMD := &MeshBufferData{}
	newMD.Data = make([]float32, len(md.Data))
	copy(newMD.Data, md.Data)

	newMD.BoneData = make([][]uint16, len(md.BoneData))
	for i := range md.BoneData {
		newMD.BoneData[i] = make([]uint16, len(md.BoneData[i]))
		copy(newMD.BoneData[i], md.BoneData[i])
	}

	newMD.VertexActiveColorChannel = make([]int, len(md.VertexActiveColorChannel))

	copy(newMD.VertexActiveColorChannel, md.VertexActiveColorChannel)

	return newMD
}

func (md *MeshBufferData) AddVertex(vertexInfo VertexInfo) {

	md.Data = append(md.Data,
		float32(vertexInfo.X), float32(vertexInfo.Y), float32(vertexInfo.Z),
		0, 0, 0, 0, // Transform
		float32(vertexInfo.NormalX), float32(vertexInfo.NormalY), float32(vertexInfo.NormalZ),
		float32(vertexInfo.U), float32(vertexInfo.V),
	)
	for _, color := range vertexInfo.Colors {
		md.Data = append(md.Data, color.R, color.G, color.B, color.A)
	}

	md.VertexActiveColorChannel = append(md.VertexActiveColorChannel, vertexInfo.ActiveColorChannel)

	if len(vertexInfo.Weights) > 0 {

		if md.Weights == nil {
			md.SkinnedPosition = newStride(md, 3)
			md.SkinnedNormal = newStride(md, 3)
			md.Weights = newStride(md, 4)
		}

		md.Data = append(md.Data, 0, 0, 0, 0, 0, 0) // Skinned position / normal
		for i := 0; i < 4; i++ {
			md.Data = append(md.Data, vertexInfo.Weights[i]) // weights
		}

		md.BoneData = append(md.BoneData, append(make([]uint16, len(vertexInfo.Bones)), vertexInfo.Bones...))

	}

}

func (md *MeshBufferData) Allocate(vertexCount int) {
	newData := make([]float32, 0, vertexCount*md.totalStrideLength)
	copy(newData, md.Data)
	md.Data = newData

	md.Mesh.VertexMax = vertexCount
}

func (md *MeshBufferData) Positions() []float64 {
	positions := make([]float64, md.Mesh.VertexCount*3)
	for i := 0; i < md.Mesh.VertexCount; i++ {
		x, y, z := md.Position.XYZ(i)
		positions[i] = float64(x)
		positions[i+1] = float64(y)
		positions[i+2] = float64(z)
	}
	return positions
}

// Mesh represents a mesh that can be represented visually in different locations via Models. By default, a new Mesh has no MeshParts (so you would need to add one
// manually if you want to construct a Mesh via code).
type Mesh struct {
	Name    string
	library *Library // A reference to the Library this Mesh came from.

	MeshParts []*MeshPart // The various mesh parts (collections of triangles, rendered with a single material).
	Triangles []*Triangle // The various triangles composing the Mesh.

	MeshBuffer *MeshBufferData

	VertexCount int
	VertexMax   int

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
	}

	mesh.MeshBuffer = NewMeshBufferData(mesh)

	return mesh

}

// Clone clones the Mesh, creating a new Mesh that has cloned MeshParts.
func (mesh *Mesh) Clone() *Mesh {
	newMesh := NewMesh(mesh.Name)
	newMesh.library = mesh.library
	newMesh.Tags = mesh.Tags.Clone()
	newMesh.triIndex = mesh.triIndex

	newMesh.MeshBuffer = mesh.MeshBuffer.Clone()

	newMesh.VertexCount = mesh.VertexCount
	newMesh.VertexMax = mesh.VertexMax

	newMesh.Triangles = make([]*Triangle, 0, len(mesh.Triangles))

	for _, part := range mesh.MeshParts {
		newPart := part.Clone()

		for i := newPart.TriangleStart; i < newPart.TriangleEnd; i++ {
			newTri := mesh.Triangles[i].Clone()
			newTri.MeshPart = newPart
			newMesh.Triangles = append(newMesh.Triangles, newTri)
		}

		newMesh.MeshParts = append(newMesh.MeshParts, newPart)
		newPart.Mesh = newMesh
	}

	for channelName, index := range mesh.VertexColorChannelNames {
		newMesh.VertexColorChannelNames[channelName] = index
	}

	newMesh.Dimensions = mesh.Dimensions.Clone()

	return newMesh
}

// CombineVertexColors allows you to combine vertex color channels together. The targetChannel is the channel that will hold
// the result, and multiplicative controls whether the combination is multiplicative (true) or additive (false). The sourceChannels
// ...int is the vertex color channel indices to combine together.
func (mesh *Mesh) CombineVertexColors(targetChannel int, multiplicative bool, sourceChannels ...int) {

	if len(sourceChannels) == 0 {
		return
	}

	for i := 0; i < mesh.VertexCount; i++ {

		var base *Color

		if multiplicative {
			base = NewColor(1, 1, 1, 1)
		} else {
			base = NewColor(0, 0, 0, 1)
		}

		for _, c := range sourceChannels {

			r, g, b, a := mesh.MeshBuffer.VertexColor.RGBA(i, c)
			color := NewColor(r, g, b, a)

			if multiplicative {
				base.Multiply(color)
			} else {
				base.Add(color)
			}

		}

		mesh.MeshBuffer.VertexColor.SetRGBA(i, targetChannel, base.R, base.G, base.B, base.A)

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
func (mesh *Mesh) AddMeshPart(material *Material) *MeshPart {
	mp := NewMeshPart(mesh, material)
	mesh.MeshParts = append(mesh.MeshParts, mp)
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

// Library returns the Library from which this Mesh was loaded. If it was created through code, this function will return nil.
func (mesh *Mesh) Library() *Library {
	return mesh.library
}

// UpdateBounds updates the mesh's dimensions; call this after manually changing vertex positions.
func (mesh *Mesh) UpdateBounds() {

	mesh.Dimensions[1] = vector.Vector{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64}
	mesh.Dimensions[0] = vector.Vector{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}

	positions := mesh.MeshBuffer.Positions()
	for i := 0; i < len(positions); i += 3 {

		if mesh.Dimensions[0][0] > positions[i] {
			mesh.Dimensions[0][0] = positions[i]
		}

		if mesh.Dimensions[0][1] > positions[i+1] {
			mesh.Dimensions[0][1] = positions[i+1]
		}

		if mesh.Dimensions[0][2] > positions[i+2] {
			mesh.Dimensions[0][2] = positions[i+2]
		}

		if mesh.Dimensions[1][0] < positions[i] {
			mesh.Dimensions[1][0] = positions[i]
		}

		if mesh.Dimensions[1][1] < positions[i+1] {
			mesh.Dimensions[1][1] = positions[i+1]
		}

		if mesh.Dimensions[1][2] < positions[i+2] {
			mesh.Dimensions[1][2] = positions[i+2]
		}

	}

}

// GetVertexInfo returns a VertexInfo struct containing the vertex information for the vertex with the provided index.
func (mesh *Mesh) GetVertexInfo(vertexIndex int) VertexInfo {

	x, y, z := mesh.MeshBuffer.Position.XYZ(vertexIndex)
	u, v := mesh.MeshBuffer.Position.UV(vertexIndex)
	nx, ny, nz := mesh.MeshBuffer.Normal.XYZ(vertexIndex)
	
	r0,g0,b0,a0 := mesh.MeshBuffer.VertexColor.RGBA(vertexIndex, 0)
	r1,g1,b1,a1 := mesh.MeshBuffer.VertexColor.RGBA(vertexIndex, 1)
	r2,g2,b2,a2 := mesh.MeshBuffer.VertexColor.RGBA(vertexIndex, 2)
	r3,g3,b3,a3 := mesh.MeshBuffer.VertexColor.RGBA(vertexIndex, 3)


	vertInfo := VertexInfo{
		ID:                 vertexIndex,
		X:                  float64(x),
		Y:                  float64(y),
		Z:                  float64(z),
		U:                  float64(u),
		V:                  float64(v),
		NormalX:            float64(nx),
		NormalY:            float64(ny),
		NormalZ:            float64(nz),
		Colors:             []*Color{
			NewColor(r0,g0,b0,a0),
			NewColor(r1,g1,b1,a1),
			NewColor(r2,g2,b2,a2),
			NewColor(r3,g3,b3,a3),
		},
		ActiveColorChannel: mesh.MeshBuffer.VertexActiveColorChannel[vertexIndex],
		Bones:              mesh.MeshBuffer.BoneData[vertexIndex],
		Weights:            mesh.MeshBuffer.Weights.,
	}

	return vertInfo

}

// AutoNormal automatically recalculates the normals for the triangles contained within the Mesh and sets the vertex normals for
// all triangles to the triangles' surface normal.
func (mesh *Mesh) AutoNormal() {

	for _, tri := range mesh.Triangles {
		tri.RecalculateNormal()
		mesh.MeshBuffer.Normal.Set(tri.ID*3, float32(tri.Normal[0]), float32(tri.Normal[1]), float32(tri.Normal[2]))
		mesh.MeshBuffer.Normal.Set(tri.ID*3+1, float32(tri.Normal[0]), float32(tri.Normal[1]), float32(tri.Normal[2]))
		mesh.MeshBuffer.Normal.Set(tri.ID*3+2, float32(tri.Normal[0]), float32(tri.Normal[1]), float32(tri.Normal[2]))
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

	for vertexIndex := 0; vertexIndex < vs.Mesh.VertexCount; vertexIndex++ {

		r, g, b, _ := vs.Mesh.MeshBuffer.VertexColor.RGBA(vertexIndex, channelIndex)

		if r > 0.01 || g > 0.01 || b > 0.01 {
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
func (vs *VertexSelection) SetColor(channelIndex int, color *Color) {

	for i := range vs.Indices {
		vs.Mesh.MeshBuffer.VertexColor.SetRGBA(i, channelIndex, color.R, color.G, color.B, color.A)
	}

}

// SetActiveColorChannel sets the active color channel in all vertices contained within the VertexSelection to the channel with the
// specified index.
func (vs *VertexSelection) SetActiveColorChannel(channelIndex int) {

	for i := range vs.Indices {
		vs.Mesh.MeshBuffer.VertexActiveColorChannel[i] = channelIndex
	}

}

// ApplyMatrix applies a Matrix4 to the position of all vertices contained within the VertexSelection.
func (vs *VertexSelection) ApplyMatrix(matrix Matrix4) {

	vec := vector.Vector{}

	for index := range vs.Indices {

		x, y, z := vs.Mesh.MeshBuffer.Position.XYZ(index)
		setVector(vec, x, y, z)

		nx, ny, nz := fastMatrixMultVec(matrix, vec)

		vs.Mesh.MeshBuffer.Position.Set(index, float32(nx), float32(ny), float32(nz))

	}

}

// Move moves all vertices contained within the VertexSelection by the provided x, y, and z values.
func (vs *VertexSelection) Move(dx, dy, dz float64) {

	for index := range vs.Indices {

		x, y, z := vs.Mesh.MeshBuffer.Position.XYZ(index)
		vs.Mesh.MeshBuffer.Position.Set(index, x+float32(dx), y+float32(dy), z+float32(dz))

	}

}

// Move moves all vertices contained within the VertexSelection by the provided 3D vector.
func (vs *VertexSelection) MoveVec(vec vector.Vector) {

	for index := range vs.Indices {

		x, y, z := vs.Mesh.MeshBuffer.Position.XYZ(index)
		vs.Mesh.MeshBuffer.Position.Set(index, x+float32(vec[0]), y+float32(vec[1]), z+float32(vec[2]))

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
	mesh.AutoNormal()

	return mesh

}

// NewIcosphere creates a new icosphere Mesh of the specified detail level. Note that the UVs are left at {0,0} because I'm lazy.
func NewIcosphere(detailLevel int) *Mesh {

	// Code cribbed from http://blog.andreaskahler.com/2009/06/creating-icosphere-mesh-in-code.html, thank you very much Andreas!

	mesh := NewMesh("Icosphere")
	part := mesh.AddMeshPart(NewMaterial("Icosphere"))

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

	triangles := []VertexInfo{

		v0, v11, v5,
		v0, v5, v1,
		v0, v1, v7,
		v0, v7, v10,
		v0, v10, v11,

		v1, v5, v9,
		v5, v11, v4,
		v11, v10, v2,
		v10, v7, v6,
		v7, v1, v8,

		v3, v9, v4,
		v3, v4, v2,
		v3, v2, v6,
		v3, v6, v8,
		v3, v8, v9,

		v4, v9, v5,
		v2, v4, v11,
		v6, v2, v10,
		v8, v6, v7,
		v9, v8, v1,
	}

	if detailLevel > 0 {

		for i := 0; i < detailLevel+1; i++ {

			newFaces := make([]VertexInfo, 0, len(triangles)*4)

			for triIndex := 0; triIndex < len(triangles); triIndex += 3 {

				a := vector.Vector{triangles[triIndex].X, triangles[triIndex].Y, triangles[triIndex].Z}
				b := vector.Vector{triangles[triIndex+1].X, triangles[triIndex+1].Y, triangles[triIndex+1].Z}
				c := vector.Vector{triangles[triIndex+2].X, triangles[triIndex+2].Y, triangles[triIndex+2].Z}

				midA := a.Add(b.Sub(a).Scale(0.5))
				midB := b.Add(c.Sub(b).Scale(0.5))
				midC := c.Add(a.Sub(c).Scale(0.5))

				midAVert := NewVertex(midA[0], midA[1], midA[2], 0, 0)
				midBVert := NewVertex(midB[0], midB[1], midB[2], 0, 0)
				midCVert := NewVertex(midC[0], midC[1], midC[2], 0, 0)

				newFaces = append(newFaces,
					triangles[triIndex], midAVert, midCVert,
					triangles[triIndex+1], midBVert, midAVert,
					triangles[triIndex+2], midCVert, midBVert,
					midAVert, midBVert, midCVert,
					// triangles[triIndex], triangles[triIndex+1], triangles[triIndex+2],
				)

			}

			triangles = newFaces

		}

	}

	for i := range triangles {

		v := vector.Vector{triangles[i].X, triangles[i].Y, triangles[i].Z}.Unit() // Make it spherical
		triangles[i].X = v[0]
		triangles[i].Y = v[1]
		triangles[i].Z = v[2]

		triangles[i].U = 0
		triangles[i].V = 0
		if i%3 == 1 {
			triangles[i].U = 1
		} else if i%3 == 2 {
			triangles[i].V = 1
		}

	}

	part.AddTriangles(triangles...)

	mesh.AutoNormal()

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
	mesh.AutoNormal()

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

	verts := []vector.Vector{}

	x, y, z := tri.MeshPart.Mesh.MeshBuffer.Position.XYZ(tri.ID * 3)
	verts = append(verts, vector.Vector{float64(x), float64(y), float64(z)})

	x, y, z = tri.MeshPart.Mesh.MeshBuffer.Position.XYZ(tri.ID*3 + 1)
	verts = append(verts, vector.Vector{float64(x), float64(y), float64(z)})

	x, y, z = tri.MeshPart.Mesh.MeshBuffer.Position.XYZ(tri.ID*3 + 2)
	verts = append(verts, vector.Vector{float64(x), float64(y), float64(z)})

	tri.Center[0] = (verts[0][0] + verts[1][0] + verts[2][0]) / 3
	tri.Center[1] = (verts[0][1] + verts[1][1] + verts[2][1]) / 3
	tri.Center[2] = (verts[0][2] + verts[1][2] + verts[2][2]) / 3

	// Determine the maximum span of the triangle; this is done for bounds checking, since we can reject triangles early if we
	// can easily tell we're too far away from them.
	dim := Dimensions{{0, 0, 0}, {0, 0, 0}}

	for i := 0; i < 3; i++ {

		if dim[0][0] > verts[i][0] {
			dim[0][0] = verts[i][0]
		}

		if dim[0][1] > verts[i][1] {
			dim[0][1] = verts[i][1]
		}

		if dim[0][2] > verts[i][2] {
			dim[0][2] = verts[i][2]
		}

		if dim[1][0] < verts[i][0] {
			dim[1][0] = verts[i][0]
		}

		if dim[1][1] < verts[i][1] {
			dim[1][1] = verts[i][1]
		}

		if dim[1][2] < verts[i][2] {
			dim[1][2] = verts[i][2]
		}

	}

	tri.MaxSpan = dim.MaxSpan()

}

// RecalculateNormal recalculates the physical normal for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position. Also note that vertex normals (visual normals) are automatically set when loading Meshes from model files.
func (tri *Triangle) RecalculateNormal() {
	x, y, z := tri.MeshPart.Mesh.MeshBuffer.Position.XYZ(tri.ID * 3)
	v0 := vector.Vector{float64(x), float64(y), float64(z)}

	x, y, z = tri.MeshPart.Mesh.MeshBuffer.Position.XYZ(tri.ID*3 + 1)
	v1 := vector.Vector{float64(x), float64(y), float64(z)}

	x, y, z = tri.MeshPart.Mesh.MeshBuffer.Position.XYZ(tri.ID*3 + 2)
	v2 := vector.Vector{float64(x), float64(y), float64(z)}

	tri.Normal = calculateNormal(v0, v1, v2)
}

func (tri *Triangle) VertexIndices() [3]int {
	return [3]int{tri.ID * 3, tri.ID*3 + 1, tri.ID*3 + 2}
}

func (tri *Triangle) SharesVertexPositions(other *Triangle) []int {

	triIndices := tri.VertexIndices()
	otherIndices := other.VertexIndices()

	mesh := tri.MeshPart.Mesh
	otherMesh := other.MeshPart.Mesh

	shareCount := []int{-1, -1, -1}

	for i, index := range triIndices {
		for j, otherIndex := range otherIndices {
			x1, y1, z1 := mesh.MeshBuffer.Position.XYZ(index)
			x2, y2, z2 := otherMesh.MeshBuffer.Position.XYZ(otherIndex)

			if vectorsEqual(x1, y1, z1, x2, y2, z2) {
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

	for i := 0; i < len(part.sortingTriangles); i++ {
		newMP.sortingTriangles = append(newMP.sortingTriangles, sortingTriangle{
			ID: part.sortingTriangles[i].ID,
		})
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
		part.Mesh.MeshBuffer.Allocate(part.Mesh.VertexMax + len(verts))
	}

	for i := 0; i < len(verts); i += 3 {

		for j := 0; j < 3; j++ {
			vertInfo := verts[i+j]
			mesh.MeshBuffer.AddVertex(vertInfo)
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

	if part.TriangleCount() >= ebiten.MaxIndicesCount/3 {
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

// ApplyMatrix applies a transformation matrix to the vertices referenced by the MeshPart.
func (part *MeshPart) ApplyMatrix(matrix Matrix4) {
	mesh := part.Mesh
	vec := vector.Vector{0, 0, 0}
	for triIndex := part.TriangleStart; triIndex < part.TriangleEnd; triIndex++ {
		for i := 0; i < 3; i++ {
			vx, vy, vz := mesh.MeshBuffer.Position.XYZ(triIndex*3 + i)
			setVector(vec, vx, vy, vz)
			x, y, z := fastMatrixMultVec(matrix, vec)
			mesh.MeshBuffer.Position.Set(triIndex*3+i, float32(x), float32(y), float32(z))
		}
	}
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
		X:       x,
		Y:       y,
		Z:       z,
		U:       u,
		V:       v,
		Weights: []float32{},
		Colors: []*Color{
			NewColor(1, 1, 1, 1),
			NewColor(1, 1, 1, 1),
			NewColor(1, 1, 1, 1),
			NewColor(1, 1, 1, 1),
		},
		ActiveColorChannel: -1,
		Bones:              []uint16{},
	}
}
