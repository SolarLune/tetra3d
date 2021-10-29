package ebiten3d

import (
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

var defaultImg *ebiten.Image

var vertexList = make([]ebiten.Vertex, 0, ebiten.MaxIndicesNum)
var indexList = make([]uint16, 0, ebiten.MaxIndicesNum)

func init() {

	defaultImg = ebiten.NewImage(4, 4)
	defaultImg.Fill(color.White)

	for i := 0; i < ebiten.MaxIndicesNum; i++ {
		// vertexList = append(vertexList, ebiten.Vertex{ColorR: 1, ColorG: 0, ColorB: 0, ColorA: 1})
		vertexList = append(vertexList, ebiten.Vertex{ColorR: rand.Float32(), ColorG: rand.Float32(), ColorB: rand.Float32(), ColorA: 1})
		indexList = append(indexList, 0)
	}

}

func UnitVector(value float64) vector.Vector {
	return vector.Vector{value, value, value}
}
