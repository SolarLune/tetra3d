package tetra3d

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDepthPassRect(t *testing.T) {
	nan := float32(math.NaN())
	tests := []struct {
		name    string
		verts   []ebiten.Vertex
		rect    image.Rectangle
		partial bool
	}{
		{"inside", []ebiten.Vertex{{DstX: 10.5, DstY: 20.2}, {DstX: 30.7, DstY: 5.9}}, image.Rect(9, 4, 32, 22), true},
		{"clipped", []ebiten.Vertex{{DstX: -5000, DstY: 700}, {DstX: 50, DstY: 9000}}, image.Rect(0, 699, 52, 800), true},
		{"full", []ebiten.Vertex{{DstX: -1, DstY: -1}, {DstX: 2000, DstY: 2000}}, image.Rect(0, 0, 1280, 800), false},
		{"off screen", []ebiten.Vertex{{DstX: 1500, DstY: 10}, {DstX: 1600, DstY: 20}}, image.Rect(1280, 9, 1280, 22), true},
		{"not a number", []ebiten.Vertex{{DstX: 10, DstY: 10}, {DstX: nan, DstY: 20}}, image.Rectangle{}, false},
		{"empty", nil, image.Rectangle{}, false},
	}
	for _, tc := range tests {
		rect, partial := depthPassRect(tc.verts, 1280, 800)
		if rect != tc.rect || partial != tc.partial {
			t.Errorf("%s: got %v %v, want %v %v", tc.name, rect, partial, tc.rect, tc.partial)
		}
	}
}
