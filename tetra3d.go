package tetra3d

// tetra3d is a 3D hybrid software/hardware renderer written for video games by usage of Ebitengine.

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var defaultImg = ebiten.NewImage(1, 1)

// const MaxTriangleCount = ebiten.MaxVertexCount / 3

const MaxTriangleCount = 21845

var colorVertexList = make([]ebiten.Vertex, MaxTriangleCount*3)
var normalVertexList = make([]ebiten.Vertex, MaxTriangleCount*3)
var depthVertexList = make([]ebiten.Vertex, MaxTriangleCount*3)
var indexList = make([]uint16, MaxTriangleCount*3)
var vertexListIndex = 0
var indexListIndex = 0
var indexListStart = 0

var globalVertexTransforms = make([]Vector4, MaxTriangleCount*3)
var globalVertexTransformedNormals = make([]Vector3, MaxTriangleCount*3)
var globalVertexDepthUnbillboarded = make([]float32, MaxTriangleCount*3)

func init() {
	defaultImg.Fill(color.White)
}
