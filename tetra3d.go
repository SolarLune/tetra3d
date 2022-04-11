package tetra3d

// tetra3d is a 3D software renderer written for video games by usage of Ebiten. It's kinda jank, but it's pretty fun. Check it out!

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var defaultImg *ebiten.Image

var colorVertexList = make([]ebiten.Vertex, 0, ebiten.MaxIndicesNum)
var depthVertexList = make([]ebiten.Vertex, 0, ebiten.MaxIndicesNum)
var indexList = make([]uint16, 0, ebiten.MaxIndicesNum)
var triList = make([]*Triangle, 0, ebiten.MaxIndicesNum/3)

func init() {

	defaultImg = ebiten.NewImage(4, 4)
	defaultImg.Fill(color.White)

	for i := 0; i < cap(colorVertexList); i++ {
		colorVertexList = append(colorVertexList, ebiten.Vertex{})
		depthVertexList = append(depthVertexList, ebiten.Vertex{})
		indexList = append(indexList, 0)
	}

	for i := 0; i < cap(triList); i++ {
		triList = append(triList, nil)
	}

}
