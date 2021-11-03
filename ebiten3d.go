package jank

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

var defaultImg *ebiten.Image

var vertexList = make([]ebiten.Vertex, 0, ebiten.MaxIndicesNum)
var indexList = make([]uint16, 0, ebiten.MaxIndicesNum)
var triList = make([]*Triangle, 0, ebiten.MaxIndicesNum/3)

func init() {

	defaultImg = ebiten.NewImage(4, 4)
	defaultImg.Fill(color.White)

	for i := 0; i < ebiten.MaxIndicesNum; i++ {
		vertexList = append(vertexList, ebiten.Vertex{})
		indexList = append(indexList, 0)
	}

	for i := 0; i < cap(triList); i++ {
		triList = append(triList, nil)
	}

}

func UnitVector(value float64) vector.Vector {
	return vector.Vector{value, value, value}
}
