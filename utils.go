package tetra3d

import (
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"

	_ "embed"
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

func min[V float64 | int](a, b V) V {
	if a < b {
		return a
	}
	return b
}

func max[V float64 | int](a, b V) V {
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

//go:embed shaders/base3d.kage
var base3DShaderText []byte

// ExtendBase3DShader allows you to make a custom fragment shader that extends the base 3D shader, allowing you
// to make a shader that has fog and depth testing built-in, as well as access to the combined painted vertex
// colors and vertex-based lighting.
// To do this, make a Kage shader, but rename the fragment entry point from "Fragment()" to "CustomFragment()".
// Otherwise, the arguments are the same - dstPos, srcPos, and color, with color containing both vertex color
// and lighting data. The return vec4 is also the same - the value that is returned from CustomFragment will be
// used for fog.
// To turn off lighting or fog individually, you would simply turn on shadelessness and foglessness in
// your object's Material (or shadelessness in your Model itself).
func ExtendBase3DShader(customFragment string) (*ebiten.Shader, error) {

	shaderText := string(base3DShaderText)

	if len(customFragment) > 0 {

		shaderTextSplit := strings.Split(shaderText, "\n")
		customTextSplit := strings.Split(customFragment, "\n")

		uniformLocationDest := -1
		customFragmentCallLocation := -1
		customFragmentDefinitionLocation := -1

		for i, line := range shaderTextSplit {
			if strings.Contains(line, "// tetra3d Custom Uniform Location //") {
				uniformLocationDest = i
			}
			if strings.Contains(line, "// tetra3d Custom Fragment Call Location //") {
				customFragmentCallLocation = i + 1
			}
			if strings.Contains(line, "// tetra3d Custom Fragment Definition Location //") {
				customFragmentDefinitionLocation = i
			}
		}

		customShaderStart := -1
		fragFunctionStart := -1

		for i, line := range customTextSplit {
			if strings.Contains(line, "package main") {
				customShaderStart = i + 1
			}
			if strings.Contains(line, "func Fragment(") {
				fragFunctionStart = i
			}
		}

		out := ""

		for i, line := range shaderTextSplit {
			if i == uniformLocationDest {
				out += strings.Join(customTextSplit[customShaderStart:fragFunctionStart], "\n") + "\n"
				out += line + "\n"
			} else if i == customFragmentCallLocation {
				// Replace the line with a new ColorTex definition
				out += "colorTex := CustomFragment(dstPos, srcPos, color)\n\n"
			} else if i == customFragmentDefinitionLocation {
				out += strings.Join(customTextSplit[fragFunctionStart:], "\n") + "\n"
				out += line + "\n"
			} else {
				out += line + "\n"
			}
		}

		shaderText = out

	}

	return ebiten.NewShader([]byte(shaderText))

}
