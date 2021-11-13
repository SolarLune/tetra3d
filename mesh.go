package tetra3d

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

// Dimensions represents the spatial dimensions of a Mesh (i.e. the minimum and maximum position of all vertices in the Mesh).
type Dimensions []vector.Vector

// Max returns the maximum number from the Dimensions (i.e. if the dimensions have a min of [-3, -1.5, -1], and a max of [3, 1.5, 1], Max()
// will return 6, as it's the largest distance.
func (dim Dimensions) Max() float64 {
	max := 0.0

	if d := dim[1][0] - dim[0][0]; d > max {
		max = d
	}

	if d := dim[1][1] - dim[0][1]; d > max {
		max = d
	}

	if d := dim[1][2] - dim[0][2]; d > max {
		max = d
	}

	return math.Abs(max)
}

type Mesh struct {
	Name           string
	Vertices       []*Vertex
	sortedVertices []*Vertex
	Triangles      []*Triangle
	Image          *ebiten.Image // What Image to use when rendering the triangles that define this Mesh
	MaterialName   string        // The name of any material assigned to the Mesh from the 3D modeler
	FilterMode     ebiten.Filter
	Dimensions     Dimensions
}

func NewMesh(name string, verts ...*Vertex) *Mesh {

	mesh := &Mesh{
		Name:           name,
		Vertices:       []*Vertex{},
		sortedVertices: []*Vertex{},
		Triangles:      []*Triangle{},
		FilterMode:     ebiten.FilterNearest,
		Dimensions:     Dimensions{{0, 0, 0}, {0, 0, 0}},
	}

	if len(verts) == 0 || len(verts)%3 != 0 {
		panic("Error: NewMesh() has not been given a correct number of vertices to constitute triangles (it needs to be greater than 0 and divisible by 3).")
	}

	mesh.AddTriangles(verts...)
	mesh.UpdateBounds()
	return mesh

}

func (mesh *Mesh) Clone() *Mesh {
	newMesh := NewMesh(mesh.Name)
	for _, t := range mesh.Triangles {
		newTri := t.Clone()
		newMesh.Triangles = append(newMesh.Triangles, newTri)
		newTri.Mesh = mesh
	}
	return newMesh
}

func (mesh *Mesh) AddTriangles(verts ...*Vertex) {
	index := 1

	for i := 0; i < len(verts); i += 3 {

		tri := NewTriangle(mesh)
		tri.ID = index
		mesh.Triangles = append(mesh.Triangles, tri)
		mesh.Vertices = append(mesh.Vertices, verts[i], verts[i+1], verts[i+2])
		mesh.sortedVertices = append(mesh.sortedVertices, verts[i], verts[i+1], verts[i+2])
		tri.SetVertices(verts[i], verts[i+1], verts[i+2])
		index++
	}

}

func (mesh *Mesh) SetVertexColor(r, g, b, a float32) {
	for _, t := range mesh.Triangles {
		for _, v := range t.Vertices {
			v.Color.R = r
			v.Color.G = g
			v.Color.B = b
			v.Color.A = a
		}
	}
}

// Repositions all vertices to take effect of the given Matrix. You can use this to, for example, translate (move) all vertices
// of a Mesh to the right by 5 units ( mesh.ApplyMatrix(tetra3d.Translate(5, 0, 0)) ), or rotate all vertices around the center by
// 90 degrees on the Y axis ( mesh.ApplyMatrix(tetra3d.Rotate(0, 1, 0, math.Pi/2) ) ) .
func (mesh *Mesh) ApplyMatrix(matrix Matrix4) {

	for _, tri := range mesh.Triangles {
		for _, vert := range tri.Vertices {
			vert.Position = matrix.MultVec(vert.Position)
		}
		tri.RecalculateCenter()
	}

	mesh.UpdateBounds()

}

// UpdateBounds updates the mesh's bounding sphere, which is used for frustum culling checks; call this after manually changing vertex positions.
func (mesh *Mesh) UpdateBounds() {

	for _, v := range mesh.Vertices {

		if v.Position[0] < mesh.Dimensions[0][0] {
			mesh.Dimensions[0][0] = v.Position[0]
		}
		if v.Position[0] > mesh.Dimensions[1][0] {
			mesh.Dimensions[1][0] = v.Position[0]
		}

		if v.Position[1] < mesh.Dimensions[0][1] {
			mesh.Dimensions[0][1] = v.Position[1]
		}
		if v.Position[1] > mesh.Dimensions[1][1] {
			mesh.Dimensions[1][1] = v.Position[1]
		}

		if v.Position[2] < mesh.Dimensions[0][2] {
			mesh.Dimensions[0][2] = v.Position[2]
		}
		if v.Position[2] > mesh.Dimensions[1][2] {
			mesh.Dimensions[1][2] = v.Position[2]
		}

	}

}

func NewCube() *Mesh {

	mesh := NewMesh("Cube",

		// Top

		NewVertex(1, 1, -1, 1, 0),
		NewVertex(1, 1, 1, 1, 1),
		NewVertex(-1, 1, -1, 0, 0),

		NewVertex(-1, 1, -1, 0, 0),
		NewVertex(1, 1, 1, 1, 1),
		NewVertex(-1, 1, 1, 0, 1),

		// Bottom

		NewVertex(-1, -1, -1, 0, 0),
		NewVertex(1, -1, 1, 1, 1),
		NewVertex(1, -1, -1, 1, 0),

		NewVertex(-1, -1, 1, 0, 1),
		NewVertex(1, -1, 1, 1, 1),
		NewVertex(-1, -1, -1, 0, 0),

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

	return mesh

}

func NewPlane() *Mesh {

	mesh := NewMesh("Plane",
		NewVertex(1, 0, -1, 1, 0),
		NewVertex(1, 0, 1, 1, 1),
		NewVertex(-1, 0, -1, 0, 0),

		NewVertex(-1, 0, -1, 0, 0),
		NewVertex(1, 0, 1, 1, 1),
		NewVertex(-1, 0, 1, 0, 1),
	)

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

type Triangle struct {
	Vertices []*Vertex
	Normal   vector.Vector
	Mesh     *Mesh
	Center   vector.Vector
	ID       int // Unique identifier number (index) in the Mesh. Each Triangle has a unique ID to assist with the triangle sorting process (see Model.TransformedVertices()).
}

func NewTriangle(mesh *Mesh) *Triangle {
	tri := &Triangle{
		Vertices: []*Vertex{},
		Mesh:     mesh,
		Center:   vector.Vector{0, 0, 0},
	}
	return tri
}

func (tri *Triangle) SetVertices(verts ...*Vertex) {

	if len(verts) < 3 {
		panic("Error: Triangle.AddVertices() received less than 3 vertices.")
	}

	tri.Vertices = verts
	for _, v := range verts {
		v.triangle = tri
	}

	tri.RecalculateCenter()
	tri.RecalculateNormal()

}

func (tri *Triangle) Clone() *Triangle {
	newTri := NewTriangle(tri.Mesh)
	for _, vertex := range tri.Vertices {
		newTri.SetVertices(vertex.Clone())
	}
	newTri.RecalculateCenter()
	return newTri
}

// RecalculateCenter recalculates the center for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position. Otherwise, it's called automatically when setting the vertices for a Triangle.
func (tri *Triangle) RecalculateCenter() {

	tri.Center[0] = (tri.Vertices[0].Position[0] + tri.Vertices[1].Position[0] + tri.Vertices[2].Position[0]) / 3
	tri.Center[1] = (tri.Vertices[0].Position[1] + tri.Vertices[1].Position[1] + tri.Vertices[2].Position[1]) / 3
	tri.Center[2] = (tri.Vertices[0].Position[2] + tri.Vertices[1].Position[2] + tri.Vertices[2].Position[2]) / 3

}

// RecalculateNormal recalculates the normal for the Triangle. Note that this should only be called if you manually change a vertex's
// individual position. Otherwise, it's called automatically when setting the vertices for a Triangle. Also note that normals are
// automatically set when loading Meshes from model files; if you call RecalculateNormal(), you'll lose any custom-defined normals
// that you defined in your modeler in favor of the default.
func (tri *Triangle) RecalculateNormal() {
	tri.Normal = calculateNormal(tri.Vertices[0].Position, tri.Vertices[1].Position, tri.Vertices[2].Position)
}

type Vertex struct {
	Position    vector.Vector
	Color       Color
	UV          vector.Vector
	transformed vector.Vector
	triangle    *Triangle
}

func NewVertex(x, y, z, u, v float64) *Vertex {
	return &Vertex{
		Position:    vector.Vector{x, y, z},
		Color:       NewColor(1, 1, 1, 1),
		UV:          vector.Vector{u, v},
		transformed: vector.Vector{0, 0, 0},
	}
}

func (vertex *Vertex) Clone() *Vertex {
	newVert := NewVertex(vertex.Position[0], vertex.Position[1], vertex.Position[2], vertex.UV[0], vertex.UV[1])
	newVert.Color = vertex.Color.Clone()
	return newVert
}

func calculateNormal(p1, p2, p3 vector.Vector) vector.Vector {

	v0 := p2.Sub(p1)
	v1 := p3.Sub(p2)

	cross, _ := v0.Cross(v1)
	return cross.Unit()

}
