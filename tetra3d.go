package tetra3d

// tetra3d is a 3D hybrid software/hardware renderer written for video games by usage of Ebitengine.

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var defaultImg = ebiten.NewImage(1, 1)

var colorVertexList = make([]ebiten.Vertex, ebiten.MaxIndicesCount)
var depthVertexList = make([]ebiten.Vertex, ebiten.MaxIndicesCount)
var indexList = make([]uint16, ebiten.MaxIndicesCount)
var vertexListIndex = 0
var indexListIndex = 0
var indexListStart = 0

const MaxTriangleCount = 21845

func init() {
	defaultImg.Fill(color.White)
}
