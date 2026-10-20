package tetra3d

import (
	"math"
	"testing"
)

// gridModel returns a model of a bumpy grid of side times side vertices in
// front of a camera. Part of the grid is outside the view, behind the camera,
// and beyond the far plane, so every clip code occurs.
func gridModel(side int) (*Model, *Camera) {
	mesh := NewMesh("grid")
	part := mesh.AddMeshPart(NewMaterial("grid"))
	verts := make([]VertexInfo, 0, side*side)
	for j := range side {
		for i := range side {
			x := float32(i)/float32(side-1)*200 - 100
			z := float32(j)/float32(side-1)*160 - 130
			y := float32(8 * math.Sin(float64(x)*0.3) * math.Cos(float64(z)*0.2))
			verts = append(verts, NewVertex(x, y, z, 0, 0))
		}
	}
	mesh.AddVertices(verts...)
	indices := make([]int, 0, (side-1)*(side-1)*6)
	for j := range side - 1 {
		for i := range side - 1 {
			a := j*side + i
			indices = append(indices, a, a+side, a+1, a+1, a+side, a+side+1)
		}
	}
	part.AddTriangles(indices...)

	camera := NewCamera("camera", 1280, 800)
	camera.SetLocalPosition(0, 4, 0)
	return NewModel("grid", mesh), camera
}

// TestTransformVerticesMatchesGeneralPass checks that the vertex pass of the
// common path gives the same bits as the general vertex pass.
func TestTransformVerticesMatchesGeneralPass(t *testing.T) {
	model, camera := gridModel(60)
	model.SetLocalPosition(1.5, -0.25, 3)
	model.SetLocalScale(1.1, 0.9, 1)
	vp := camera.ViewMatrix().Mult(camera.Projection())
	part := model.mesh.MeshParts[0]
	n := len(model.mesh.VertexPositions)

	// Stored positions for the lights take the general pass.
	vertexListIndex = 0
	model.processVertices(vp, camera, part, true, nil, true)
	clip := append([]Vector4(nil), globalVertexTransforms[:n]...)
	screen := append([]Vector2(nil), globalVertexScreen[:n]...)
	codes := append([]uint8(nil), globalVertexClipCodes[:n]...)
	general := len(globalSortingTriangleBucket.sorted)

	clear(globalVertexTransforms[:n])
	clear(globalVertexScreen[:n])
	clear(globalVertexClipCodes[:n])
	vertexListIndex = 0
	model.processVertices(vp, camera, part, true, nil, false)

	seen := uint8(0)
	for i := range n {
		a, b := clip[i], globalVertexTransforms[i]
		if math.Float32bits(a.X) != math.Float32bits(b.X) || math.Float32bits(a.Y) != math.Float32bits(b.Y) ||
			math.Float32bits(a.Z) != math.Float32bits(b.Z) || math.Float32bits(a.W) != math.Float32bits(b.W) {
			t.Fatalf("vertex %d: clip position %v, want %v", i, b, a)
		}
		if s := globalVertexScreen[i]; math.Float32bits(s.X) != math.Float32bits(screen[i].X) || math.Float32bits(s.Y) != math.Float32bits(screen[i].Y) {
			t.Fatalf("vertex %d: screen position %v, want %v", i, s, screen[i])
		}
		if globalVertexClipCodes[i] != codes[i] {
			t.Fatalf("vertex %d: clip code %b, want %b", i, globalVertexClipCodes[i], codes[i])
		}
		seen |= codes[i]
	}
	if seen != clipLeft|clipRight|clipBottom|clipTop|clipDepth {
		t.Errorf("clip codes seen %b, want all of them", seen)
	}
	if got := len(globalSortingTriangleBucket.sorted); got != general || got == 0 {
		t.Errorf("%d visible triangles, want %d and more than 0", got, general)
	}
}

// BenchmarkProcessVertices measures the vertex and triangle passes of one
// mesh part of about 30,000 vertices.
func BenchmarkProcessVertices(b *testing.B) {
	model, camera := gridModel(174)
	vp := camera.ViewMatrix().Mult(camera.Projection())
	part := model.mesh.MeshParts[0]
	b.ResetTimer()
	for b.Loop() {
		vertexListIndex = 0
		model.processVertices(vp, camera, part, true, nil, false)
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(model.mesh.VertexPositions)), "ns/vertex")
}

// BenchmarkTransformVertices measures the vertex pass of the common path
// alone over about 30,000 vertices.
func BenchmarkTransformVertices(b *testing.B) {
	model, camera := gridModel(174)
	mvp := camera.ViewMatrix().Mult(camera.Projection())
	pos := model.mesh.VertexPositions
	for len(pos) >= len(colorVertexList) {
		growDisplayLists()
	}
	n := len(pos)
	for b.Loop() {
		transformVertices(&mvp, camera.near, camera.far, pos, globalVertexTransforms[:n], globalVertexScreen[:n], globalVertexClipCodes[:n])
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(n), "ns/vertex")
}
