package tetra3d

import (
	"image"
	"math"

	"golang.org/x/image/font"
)

type FinishMode int

const (
	FinishModeLoop     FinishMode = iota // Loop on animation completion
	FinishModePingPong                   // Reverse on animation completion; if this is the case, the OnFinish() callback is called after two loops (one reversal)
	FinishModeStop                       // Stop on animation completion
)

// ToRadians is a helper function to easily convert degrees to radians (which is what the rotation-oriented functions in Tetra3D use).
func ToRadians(degrees float64) float64 {
	return math.Pi * degrees / 180
}

// ToDegrees is a helper function to easily convert radians to degrees for human readability.
func ToDegrees(radians float64) float64 {
	return radians / math.Pi * 180
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func clamp[V float64 | int](value, min, max V) V {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

func pow(value float64, power int) float64 {
	x := value
	for i := 0; i < power; i++ {
		x += x
	}
	return x
}

func round(value float64) float64 {

	iv := float64(int(value))

	if value > iv+0.5 {
		return iv + 1
	} else if value < iv-0.5 {
		return iv - 1
	}

	return iv

}

// type TextMeasure struct {
// 	X, Y int
// }

func measureText(text string, fontFace font.Face) image.Rectangle {
	bounds, _ := font.BoundString(fontFace, text)
	// advance := font.MeasureString(fontFace, text)
	newBounds := image.Rect(int(bounds.Min.X)>>6, int(bounds.Min.Y)>>6, int(bounds.Max.X)>>6, int(bounds.Max.Y)>>6)
	// newBounds := TextMeasure{int((bounds.Max.X - bounds.Min.X) >> 6), int((bounds.Max.Y - bounds.Min.Y) >> 6)}
	return newBounds
}
