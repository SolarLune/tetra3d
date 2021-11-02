package ebiten3d

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

type Mesh struct {
	Triangles       []*Triangle
	BackfaceCulling bool
	Image           *ebiten.Image
}

func NewMesh(verts ...*Vertex) *Mesh {

	mesh := &Mesh{
		Triangles:       []*Triangle{},
		BackfaceCulling: true,
	}
	for i := 0; i < len(verts); i += 3 {
		if len(verts) < i {
			panic("error: NewMesh() did not receive enough vertices")
		}
		mesh.AddTriangles(verts[i], verts[i+1], verts[i+2])
	}
	return mesh

}

func (mesh *Mesh) Clone() *Mesh {
	newMesh := NewMesh()
	for _, t := range mesh.Triangles {
		newTri := t.Clone()
		newMesh.Triangles = append(newMesh.Triangles, newTri)
		newTri.Mesh = mesh
	}
	return newMesh
}

func (mesh *Mesh) AddTriangles(verts ...*Vertex) {
	for i := 0; i < len(verts); i += 3 {
		tri := NewTriangle(mesh)
		mesh.Triangles = append(mesh.Triangles, tri)
		tri.SetVertices(verts[i], verts[i+1], verts[i+2])
	}
}

// func (model *Model) WorldSpaceToClipSpace(matrix Matrix4) []Vec {

// 	verts := []Vec{}

// 	for _, vert := range model.Vertices {

// 		newV := vert.Clone()

// 		verts = append(verts, newV)

// 	}

// 	return verts

// }

// func LoadMeshFromOBJFile(filepath string) *Mesh {

// 	loadedFile, err := os.ReadFile(filepath)

// 	if err != nil {
// 		log.Println(err.Error())
// 	} else {
// 		return LoadMeshFromOBJData(loadedFile)
// 	}

// 	return nil

// }

// func LoadMeshFromOBJData(objData []byte) *Mesh {

// 	mesh := NewMesh()

// 	verts := []*Vertex{}

// 	for _, line := range strings.Split(string(objData), "\n") {

// 		elements := strings.Split(line, " ")

// 		switch elements[0] {
// 		case "v":
// 			floatValueX, _ := strconv.ParseFloat(strings.TrimSpace(elements[1]), 64)
// 			floatValueY, _ := strconv.ParseFloat(strings.TrimSpace(elements[2]), 64)
// 			floatValueZ, _ := strconv.ParseFloat(strings.TrimSpace(elements[3]), 64)

// 			// mesh.Vertices = append(mesh.Vertices, vector.Vector{floatValueX, floatValueY, floatValueZ})
// 			verts = append(verts, NewVertex(vector.Vector{floatValueX, floatValueY, floatValueZ}))

// 		}

// 	}

// 	for _, line := range strings.Split(string(objData), "\n") {

// 		elements := strings.Split(line, " ")

// 		switch elements[0] {

// 		case "f":

// 			mesh.AddTriangle()
// 			index0, _ := strconv.ParseInt(strings.TrimSpace(elements[1]), 0, 64)
// 			index1, _ := strconv.ParseInt(strings.TrimSpace(elements[2]), 0, 64)
// 			index2, _ := strconv.ParseInt(strings.TrimSpace(elements[3]), 0, 64)

// 			// Interestingly, the indices seem to be 1-based, not 0-based
// 			// newFace.Indices = append(newFace.Indices, uint16(index0-1), uint16(index1-1), uint16(index2-1))
// 			// mesh.Triangles = append(mesh.Triangles, newFace)

// 		}

// 	}

// 	return mesh

// }

func NewCube() *Mesh {

	mesh := NewMesh()

	mesh.AddTriangles(

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

	mesh := NewMesh()

	type v = vector.Vector

	mesh.AddTriangles(

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
}

func NewTriangle(mesh *Mesh) *Triangle {
	tri := &Triangle{
		Vertices: []*Vertex{},
		Mesh:     mesh,
	}
	return tri
}

func (tri *Triangle) SetVertices(verts ...*Vertex) {

	if len(verts) < 3 {
		panic("Error: Triangle.AddVertices() received less than 3 vertices.")
	}

	tri.Vertices = verts

	tri.RecalculateNormal()

}

func (tri *Triangle) Clone() *Triangle {
	newTri := NewTriangle(tri.Mesh)
	for _, vertex := range tri.Vertices {
		newTri.SetVertices(vertex.Clone())
	}
	newTri.RecalculateNormal()
	return newTri
}

func (tri *Triangle) RecalculateNormal() {

	tri.Normal = calculateNormal(
		tri.Vertices[0].Position,
		tri.Vertices[1].Position,
		tri.Vertices[2].Position,
	)

}

func (tri *Triangle) Center() vector.Vector {

	v := vector.Vector{0, 0, 0}
	for _, vert := range tri.Vertices {
		v = v.Add(vert.Position)
	}
	v = v.Scale(1.0 / float64(len(tri.Vertices)))
	return v

}

func (tri *Triangle) NearestPoint(from vector.Vector) vector.Vector {

	var closest vector.Vector
	for _, vert := range tri.Vertices {
		if closest == nil || from.Sub(vert.Position).Magnitude() < from.Sub(closest).Magnitude() {
			closest = vert.Position
		}
	}
	return closest

}

type Vertex struct {
	Position vector.Vector
	Color    vector.Vector
	UV       vector.Vector
}

func NewVertex(x, y, z, u, v float64) *Vertex {
	return &Vertex{
		Position: vector.Vector{x, y, z},
		Color:    vector.Vector{1, 1, 1, 1},
		UV:       vector.Vector{u, v},
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
