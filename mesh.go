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

// Max returns the maximum value from all of the axes in the Dimensions. For example, if the Dimensions have a min of [-1, -2, -2],
// and a max of [6, 1.5, 1], Max() will return 7, as it's the largest distance between all axes.
func (dim Dimensions) Max() float64 {
	return math.Max(math.Max(dim.Width(), dim.Height()), dim.Depth())
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

// Mesh represents a mesh that can be represented visually in different locations via Models. By default, a new Mesh has no MeshParts (so you would need to add one
// manually if you want to construct a Mesh via code).
type Mesh struct {
	Name       string
	library    *Library    // A reference to the Library this Mesh came from.
	MeshParts  []*MeshPart // A slice of mesh parts
	Dimensions Dimensions
	triIndex   int
	Tags       *Tags
}

// NewMesh takes a name and a slice of *Vertex instances, and returns a new Mesh. If you provide *Vertex instances, the number must be divisible by 3,
// or NewMesh will panic.
func NewMesh(name string) *Mesh {

	mesh := &Mesh{
		Name:      name,
		MeshParts: []*MeshPart{},
		// Triangles:  []*Triangle{},
		Dimensions: Dimensions{{0, 0, 0}, {0, 0, 0}},
		triIndex:   1,
		Tags:       NewTags(),
	}

	return mesh

}

// TotalVertexCount returns the total number of vertices in the Mesh by adding up the number of vertices of each MeshPart.
func (mesh *Mesh) TotalVertexCount() int {
	total := 0
	for _, part := range mesh.MeshParts {
		total += len(part.Vertices)
	}
	return total
}

// TotalTriangleCount returns the total number of triangles in the Mesh by adding up the number of triangles of each MeshPart.
func (mesh *Mesh) TotalTriangleCount() int {
	total := 0
	for _, part := range mesh.MeshParts {
		total += len(part.Triangles)
	}
	return total
}

// AddMeshPart allows you to add a new MeshPart to the Mesh with the given Material (with a nil Material reference also being valid).
func (mesh *Mesh) AddMeshPart(material *Material) *MeshPart {
	mp := NewMeshPart(mesh, material)
	mesh.MeshParts = append(mesh.MeshParts, mp)
	return mp
}

// GetMeshPart allows you to retrieve a MeshPart by its material's name. If no material with the provided name is given, the function returns nil.
func (mesh *Mesh) GetMeshPart(materialName string) *MeshPart {
	for _, mp := range mesh.MeshParts {
		if mp.Material.Name == materialName {
			return mp
		}
	}
	return nil
}

// Clone clones the Mesh, creating a new Mesh that has cloned MeshParts.
func (mesh *Mesh) Clone() *Mesh {
	newMesh := NewMesh(mesh.Name)
	newMesh.library = mesh.library
	newMesh.Tags = mesh.Tags.Clone()
	for _, part := range mesh.MeshParts {
		newPart := part.Clone()
		newMesh.MeshParts = append(newMesh.MeshParts, newPart)
		newPart.Mesh = mesh
	}
	return newMesh
}

// Library returns the Library from which this Mesh was loaded. If it was created through code, this function will return nil.
func (mesh *Mesh) Library() *Library {
	return mesh.library
}

// UpdateBounds updates the mesh's dimensions; call this after manually changing vertex positions.
func (mesh *Mesh) UpdateBounds() {

	mesh.Dimensions[1] = vector.Vector{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64}
	mesh.Dimensions[0] = vector.Vector{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}

	for _, part := range mesh.MeshParts {

		for _, v := range part.Vertices {

			if mesh.Dimensions[0][0] > v.Position[0] {
				mesh.Dimensions[0][0] = v.Position[0]
			}

			if mesh.Dimensions[0][1] > v.Position[1] {
				mesh.Dimensions[0][1] = v.Position[1]
			}

			if mesh.Dimensions[0][2] > v.Position[2] {
				mesh.Dimensions[0][2] = v.Position[2]
			}

			if mesh.Dimensions[1][0] < v.Position[0] {
				mesh.Dimensions[1][0] = v.Position[0]
			}

			if mesh.Dimensions[1][1] < v.Position[1] {
				mesh.Dimensions[1][1] = v.Position[1]
			}

			if mesh.Dimensions[1][2] < v.Position[2] {
				mesh.Dimensions[1][2] = v.Position[2]
			}

		}

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

// func NewWeirdDebuggingStatueThing() *Mesh {

// 	mesh := NewMesh()

// 	type v = vector.Vector

// 	mesh.AddTriangles(

// 		NewVertex(1, 0, -1, 1, 0),
// 		NewVertex(1, 0, 1, 1, 1),
// 		NewVertex(-1, 0, -1, 0, 0),

// 		NewVertex(-1, 0, -1, 0, 0),
// 		NewVertex(1, 0, 1, 1, 1),
// 		NewVertex(-1, 0, 1, 0, 1),

// 		NewVertex(-1, 2, -1, 0, 0),
// 		NewVertex(1, 2, 1, 1, 1),
// 		NewVertex(1, 0, -1, 1, 0),

// 		NewVertex(-1, 0, 1, 0, 1),
// 		NewVertex(1, 2, 1, 1, 1),
// 		NewVertex(-1, 2, -1, 0, 0),
// 	)

// 	return mesh

// }

// A Triangle represents the smallest renderable object in Tetra3D.
type Triangle struct {
	Vertices []*Vertex
	MaxSpan  float64
	Normal   vector.Vector
	MeshPart *MeshPart
	Center   vector.Vector
	visible  bool
	depth    float64
	ID       int // Unique identifier number (index) in the Mesh. Each Triangle has a unique ID to assist with the triangle sorting process (see Model.TransformedVertices()).
}

// NewTriangle creates a new Triangle, and requires a reference to its owning Mesh.
func NewTriangle(meshPart *MeshPart) *Triangle {
	tri := &Triangle{
		Vertices: []*Vertex{},
		MeshPart: meshPart,
		Center:   vector.Vector{0, 0, 0},
	}
	return tri
}

// SetVertices sets the vertices on the Triangle given; if the Triangle receives fewer than 3 vertices, it will panic.
func (tri *Triangle) SetVertices(verts ...*Vertex) {

	if len(verts) < 3 {
		panic("Error: Triangle.AddVertices() received less than 3 vertices.")
	}

	tri.Vertices = verts
	for i, v := range verts {
		v.triangle = tri
		v.ID = ((tri.ID - 1) * 3) + i
	}

	tri.RecalculateCenter()
	tri.RecalculateNormal()

}

// Clone clones the Triangle, keeping a reference to the same Material.
func (tri *Triangle) Clone() *Triangle {
	newTri := NewTriangle(tri.MeshPart)
	newVerts := make([]*Vertex, len(tri.Vertices))
	for i, vertex := range tri.Vertices {
		newVerts[i] = vertex.Clone()
	}
	newTri.SetVertices(newVerts...)
	newTri.MeshPart = tri.MeshPart
	newTri.RecalculateCenter()
	return newTri
}

// RecalculateCenter recalculates the center for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position. Otherwise, it's called automatically when setting the vertices for a Triangle.
func (tri *Triangle) RecalculateCenter() {

	tri.Center[0] = (tri.Vertices[0].Position[0] + tri.Vertices[1].Position[0] + tri.Vertices[2].Position[0]) / 3
	tri.Center[1] = (tri.Vertices[0].Position[1] + tri.Vertices[1].Position[1] + tri.Vertices[2].Position[1]) / 3
	tri.Center[2] = (tri.Vertices[0].Position[2] + tri.Vertices[1].Position[2] + tri.Vertices[2].Position[2]) / 3

	// Determine the maximum span of the triangle; this is done for bounds checking, since we can reject triangles early if we
	// can easily tell we're too far away from them.
	dim := Dimensions{{0, 0, 0}, {0, 0, 0}}

	for _, v := range tri.Vertices {

		if dim[0][0] > v.Position[0] {
			dim[0][0] = v.Position[0]
		}

		if dim[0][1] > v.Position[1] {
			dim[0][1] = v.Position[1]
		}

		if dim[0][2] > v.Position[2] {
			dim[0][2] = v.Position[2]
		}

		if dim[1][0] < v.Position[0] {
			dim[1][0] = v.Position[0]
		}

		if dim[1][1] < v.Position[1] {
			dim[1][1] = v.Position[1]
		}

		if dim[1][2] < v.Position[2] {
			dim[1][2] = v.Position[2]
		}

	}

	tri.MaxSpan = dim.Max()

}

// RecalculateNormal recalculates the normal for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position. Otherwise, it's called automatically when setting the vertices for a Triangle. Also note that normals are
// automatically set when loading Meshes from model files; if you call RecalculateNormal(), you'll lose any custom-defined normals
// that you defined in your modeler in favor of the default.
func (tri *Triangle) RecalculateNormal() {
	tri.Normal = calculateNormal(tri.Vertices[0].Position, tri.Vertices[1].Position, tri.Vertices[2].Position)
}

// Vertex represents a vertex. Vertices are not shared between Triangles.
type Vertex struct {
	Position    vector.Vector
	Color       *Color
	UV          vector.Vector
	transformed vector.Vector
	Normal      vector.Vector
	triangle    *Triangle
	Weights     []float32
	ID          int
}

// NewVertex creates a new Vertex with the provided position (x, y, z) and UV values (u, v).
func NewVertex(x, y, z, u, v float64) *Vertex {
	return &Vertex{
		Position:    vector.Vector{x, y, z},
		Color:       NewColor(1, 1, 1, 1),
		UV:          vector.Vector{u, v},
		transformed: vector.Vector{0, 0, 0},
	}
}

// Clone clones the Vertex.
func (vertex *Vertex) Clone() *Vertex {
	newVert := &Vertex{
		Position:    vector.Vector{vertex.Position[0], vertex.Position[1], vertex.Position[2]},
		Color:       vertex.Color.Clone(),
		UV:          vector.Vector{vertex.UV[0], vertex.UV[1]},
		Weights:     make([]float32, len(vertex.Weights)),
		transformed: vector.Vector{0, 0, 0},
	}

	copy(newVert.Weights, vertex.Weights)

	return newVert
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
	Mesh            *Mesh
	Triangles       []*Triangle
	sortedTriangles []*Triangle
	Vertices        []*Vertex
	Material        *Material
}

func NewMeshPart(mesh *Mesh, material *Material) *MeshPart {
	return &MeshPart{
		Mesh:      mesh,
		Triangles: []*Triangle{},
		Vertices:  []*Vertex{},
		Material:  material,
	}
}

func (part *MeshPart) Clone() *MeshPart {
	newMP := &MeshPart{
		Mesh:            part.Mesh,
		Triangles:       make([]*Triangle, len(part.Triangles)),
		sortedTriangles: make([]*Triangle, len(part.Triangles)),
		Material:        part.Material,
		Vertices:        make([]*Vertex, 0, len(part.Triangles)*3),
	}
	for i, tri := range part.Triangles {
		newTri := tri.Clone()
		newMP.Triangles[i] = newTri
		newMP.sortedTriangles[i] = newTri
		newMP.Vertices = append(newMP.Vertices, tri.Vertices...)
	}
	newMP.Material = part.Material
	return newMP
}

// SetVertexColor sets the vertex color of all vertices in the Mesh to the color values provided.
func (part *MeshPart) SetVertexColor(color Color) {
	for _, v := range part.Vertices {
		v.Color.R = color.R
		v.Color.G = color.G
		v.Color.B = color.B
		v.Color.A = color.A
	}
}

// ApplyMatrix applies the Matrix provided to all vertices on the Mesh. You can use this to, for example, translate (move) all vertices
// of a Mesh to the right by 5 units ( mesh.ApplyMatrix(tetra3d.Translate(5, 0, 0)) ), or rotate all vertices around the center by
// 90 degrees on the Y axis ( mesh.ApplyMatrix(tetra3d.Rotate(0, 1, 0, math.Pi/2) ) ).
// Note that ApplyMatrix does NOT update the bounds on the Mesh after matrix application.
func (part *MeshPart) ApplyMatrix(matrix Matrix4) {

	for _, tri := range part.Triangles {
		for _, vert := range tri.Vertices {
			vert.Position = matrix.MultVec(vert.Position)
		}
		tri.RecalculateCenter()
	}

}

// AddTriangles adds triangles consisting of vertices to the Mesh. You must provide a number of *Vertex instances divisible by 3.
func (part *MeshPart) AddTriangles(verts ...*Vertex) {

	if len(verts) == 0 || len(verts)%3 != 0 {
		panic("Error: AddTriangles() has not been given a correct number of vertices to constitute triangles (it needs to be greater than 0 and divisible by 3).")
	}
	part.Vertices = append(part.Vertices, verts...)
	for i := 0; i < len(verts); i += 3 {

		tri := NewTriangle(part)
		tri.ID = part.Mesh.triIndex
		part.Triangles = append(part.Triangles, tri)
		part.sortedTriangles = append(part.sortedTriangles, tri)
		tri.SetVertices(verts[i], verts[i+1], verts[i+2])
		part.Mesh.triIndex++

	}

	if len(part.Triangles) >= ebiten.MaxIndicesNum/3 {
		matName := "nil"
		if part.Material != nil {
			matName = part.Material.Name
		}
		log.Println("warning: mesh [" + part.Mesh.Name + "] has part with material named [" + matName + "], which has " + fmt.Sprintf("%d", len(part.Triangles)) + " triangles. This exceeds the renderable maximum of 21845 triangles total for one MeshPart; please break up the mesh into multiple MeshParts using materials, or split it up into models. Otherwise, the game will crash if it renders over the maximum number of triangles.")
	}

}
