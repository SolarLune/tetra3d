package jank3d

import (
	"image/color"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
)

var defaultImg *ebiten.Image

var vertexList = make([]ebiten.Vertex, 0, ebiten.MaxIndicesNum)
var indexList = make([]uint16, 0, ebiten.MaxIndicesNum)
var triList = make([]*Triangle, 0, ebiten.MaxIndicesNum/3)

var debugFontFace font.Face

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

	tt, err := truetype.Parse(gomono.TTF)
	if err != nil {
		panic(err)
	}

	debugFontFace = truetype.NewFace(tt, &truetype.Options{Size: 16, DPI: 72, Hinting: font.HintingFull})

}

func UnitVector(value float64) vector.Vector {
	return vector.Vector{value, value, value}
}
