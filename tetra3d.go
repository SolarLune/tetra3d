package tetra3d

// tetra3d is a 3D software renderer written for video games by usage of Ebiten. It's kinda jank, but it's pretty fun. Check it out!

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var defaultImg = ebiten.NewImage(1, 1)

var colorVertexList = make([]ebiten.Vertex, ebiten.MaxIndicesNum)
var depthVertexList = make([]ebiten.Vertex, ebiten.MaxIndicesNum)
var indexList = make([]uint16, ebiten.MaxIndicesNum)

func init() {
	defaultImg.Fill(color.White)
}
