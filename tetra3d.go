package tetra3d

// tetra3d is a 3D software renderer written for video games by usage of Ebiten. It's kinda jank, but it's pretty fun. Check it out!

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var defaultImg = ebiten.NewImage(1, 1)

var colorVertexList = make([]ebiten.Vertex, ebiten.MaxIndicesCount)
var depthVertexList = make([]ebiten.Vertex, ebiten.MaxIndicesCount)
var indexList = make([]uint16, ebiten.MaxIndicesCount)
var vertexListIndex = 0

const maxTriangleCount = 21845

func init() {
	defaultImg.Fill(color.White)
}
