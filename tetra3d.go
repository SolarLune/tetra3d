package tetra3d

// tetra3d is a 3D hybrid software/hardware renderer written for video games by usage of Ebitengine.

import (
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

var defaultImg = ebiten.NewImage(1, 1)

// The hardware-limited maximum number of triangles displayable at any given time.
const MaxTriangleCount = ebiten.MaxVertexCount / 3

// const MaxTriangleCount = 21845

const startingDisplayListSize = 30_000 // 30000 / 3 = 10000 tris

var colorVertexList = make([]ebiten.Vertex, startingDisplayListSize) // Max triangle count is 1431655765, but the display lists can have more sensible starting capacities.
var normalVertexList = make([]ebiten.Vertex, startingDisplayListSize)
var depthVertexList = make([]ebiten.Vertex, startingDisplayListSize)
var indexList = make([]uint16, startingDisplayListSize)

var vertexListIndex = 0

var indexListStart = 0

var globalVertexTransforms = make([]Vector4, startingDisplayListSize)
var globalVertexTransformedNormals = make([]Vector3, startingDisplayListSize)
var globalVertexDepthUnbillboarded = make([]float32, startingDisplayListSize)

func init() {
	defaultImg.Fill(color.White)
}

func growDisplayLists() {
	colorVertexList = slices.Grow(colorVertexList, cap(colorVertexList)*2)
	normalVertexList = slices.Grow(normalVertexList, cap(normalVertexList)*2)
	depthVertexList = slices.Grow(depthVertexList, cap(depthVertexList)*2)
	indexList = slices.Grow(indexList, cap(indexList)*2)
	globalVertexTransforms = slices.Grow(globalVertexTransforms, cap(globalVertexTransforms)*2)
	globalVertexTransformedNormals = slices.Grow(globalVertexTransformedNormals, cap(globalVertexTransformedNormals)*2)
	globalVertexDepthUnbillboarded = slices.Grow(globalVertexDepthUnbillboarded, cap(globalVertexDepthUnbillboarded)*2)

	for i := len(colorVertexList); i < cap(colorVertexList); i++ {
		colorVertexList = append(colorVertexList, ebiten.Vertex{})
	}

	for i := len(depthVertexList); i < cap(depthVertexList); i++ {
		depthVertexList = append(depthVertexList, ebiten.Vertex{})
	}

	for i := len(normalVertexList); i < cap(normalVertexList); i++ {
		normalVertexList = append(normalVertexList, ebiten.Vertex{})
	}

	for i := len(indexList); i < cap(indexList); i++ {
		indexList = append(indexList, 0)
	}

	for i := len(globalVertexTransforms); i < cap(globalVertexTransforms); i++ {
		globalVertexTransforms = append(globalVertexTransforms, Vector4{})
	}

	for i := len(globalVertexTransformedNormals); i < cap(globalVertexTransformedNormals); i++ {
		globalVertexTransformedNormals = append(globalVertexTransformedNormals, Vector3{})
	}

	for i := len(globalVertexDepthUnbillboarded); i < cap(globalVertexDepthUnbillboarded); i++ {
		globalVertexDepthUnbillboarded = append(globalVertexDepthUnbillboarded, 0)
	}

	globalSortingTriangleBucket.resizeTriangleCount(len(globalSortingTriangleBucket.unsetTris) * 2)

}
