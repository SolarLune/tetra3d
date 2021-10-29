package ebiten3d

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/kvartborg/vector"
)

type Mesh struct {
	Vertices        []vector.Vector
	Triangles       []*Triangle
	BackfaceCulling bool
}

func NewMesh() *Mesh {

	mesh := &Mesh{
		Vertices:        []vector.Vector{},
		Triangles:       []*Triangle{},
		BackfaceCulling: true,
	}
	return mesh

}

func (mesh *Mesh) Clone() *Mesh {
	newMesh := NewMesh()
	newMesh.Vertices = append(newMesh.Vertices, mesh.Vertices...)
	newMesh.Triangles = append(newMesh.Triangles, mesh.Triangles...)
	for _, t := range newMesh.Triangles {
		t.Mesh = mesh
	}
	return newMesh
}

// func (model *Model) WorldSpaceToClipSpace(matrix Matrix4) []Vec {

// 	verts := []Vec{}

// 	for _, vert := range model.Vertices {

// 		newV := vert.Clone()

// 		verts = append(verts, newV)

// 	}

// 	return verts

// }

func LoadMeshFromOBJFile(filepath string) *Mesh {

	loadedFile, err := os.ReadFile(filepath)

	if err != nil {
		log.Println(err.Error())
	} else {
		return LoadMeshFromOBJData(loadedFile)
	}

	return nil

}

func LoadMeshFromOBJData(objData []byte) *Mesh {

	model := NewMesh()

	for _, line := range strings.Split(string(objData), "\n") {

		elements := strings.Split(line, " ")

		switch elements[0] {
		case "v":
			floatValueX, _ := strconv.ParseFloat(strings.TrimSpace(elements[1]), 64)
			floatValueY, _ := strconv.ParseFloat(strings.TrimSpace(elements[2]), 64)
			floatValueZ, _ := strconv.ParseFloat(strings.TrimSpace(elements[3]), 64)

			model.Vertices = append(model.Vertices, vector.Vector{floatValueX, floatValueY, floatValueZ})
		case "f":
			newFace := NewTriangle(model)
			index0, _ := strconv.ParseInt(strings.TrimSpace(elements[1]), 0, 64)
			index1, _ := strconv.ParseInt(strings.TrimSpace(elements[2]), 0, 64)
			index2, _ := strconv.ParseInt(strings.TrimSpace(elements[3]), 0, 64)

			// Interestingly, the indices seem to be 1-based, not 0-based
			newFace.Indices = append(newFace.Indices, uint16(index0-1), uint16(index1-1), uint16(index2-1))
			model.Triangles = append(model.Triangles, newFace)
		}

	}

	return model

}

func NewCube() *Mesh {

	model := NewMesh()

	verts := []vector.Vector{
		{1, 1, 1},   // RFU
		{1, -1, 1},  // RBU
		{-1, -1, 1}, // LBU
		{-1, 1, 1},  // LFU

		{1, 1, -1},   // LFD
		{1, -1, -1},  // LFD
		{-1, -1, -1}, // LFD
		{-1, 1, -1},  // LFD
	}

	indices := []uint16{

		1, 2, 3,
		4, 7, 6,

		4, 5, 1,
		1, 5, 6,

		6, 7, 3,
		4, 0, 3,

		0, 1, 3,
		5, 4, 6,

		0, 4, 1,
		2, 1, 6,

		2, 6, 3,
		7, 4, 3,
	}

	model.Vertices = append(model.Vertices, verts...)

	for i := 0; i < len(indices); i += 3 {
		newTri := NewTriangle(model)
		newTri.Indices = []uint16{indices[i], indices[i+1], indices[i+2]}
		newTri.RecalculateNormal()
		model.Triangles = append(model.Triangles, newTri)
	}

	return model

}

func NewPlane() *Mesh {

	model := NewMesh()

	verts := []vector.Vector{
		{1, 0, 1},   // RF
		{1, 0, -1},  // RB
		{-1, 0, -1}, // LB
		{-1, 0, 1},  // LF
	}

	indices := []uint16{
		0, 1, 2,
		2, 3, 0,
	}

	model.Vertices = append(model.Vertices, verts...)

	for i := 0; i < len(indices); i += 3 {
		newTri := NewTriangle(model)
		newTri.Indices = []uint16{indices[i], indices[i+1], indices[i+2]}
		newTri.RecalculateNormal()
		model.Triangles = append(model.Triangles, newTri)
	}

	return model

}

type Triangle struct {
	Indices []uint16
	Normal  vector.Vector
	Mesh    *Mesh
}

func NewTriangle(mesh *Mesh) *Triangle {
	return &Triangle{
		Indices: []uint16{},
		Mesh:    mesh,
	}
}

func (tri *Triangle) Clone() *Triangle {
	newTri := NewTriangle(tri.Mesh)
	newTri.Indices = append(newTri.Indices, tri.Indices...)
	newTri.RecalculateNormal()
	return newTri
}

func (tri *Triangle) RecalculateNormal() {

	tri.Normal = calculateNormal(
		tri.Mesh.Vertices[tri.Indices[0]],
		tri.Mesh.Vertices[tri.Indices[1]],
		tri.Mesh.Vertices[tri.Indices[2]],
	)

}

func (tri *Triangle) Vertices() []vector.Vector {
	verts := []vector.Vector{}
	for _, index := range tri.Indices {
		verts = append(verts, tri.Mesh.Vertices[index])
	}
	return verts
}

func calculateNormal(p1, p2, p3 vector.Vector) vector.Vector {

	v0 := p2.Sub(p1)
	v1 := p3.Sub(p2)

	cross, _ := v0.Cross(v1)
	cross = cross.Unit()
	return cross

}
