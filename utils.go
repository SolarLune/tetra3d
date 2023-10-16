package tetra3d

import (
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
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

	shaderText := `
//kage:unit pixels

package main

var Fog vec4
var FogRange [2]float
var DitherSize float
var FogCurve float
var Fogless float

var BayerMatrix [16]float

func decodeDepth(rgba vec4) float {
	return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
}

func OutCirc(v float) float {
	return sqrt(1 - pow(v - 1, 2))
}

func InCirc(v float) float {
	return 1 - sqrt(1 - pow(v, 2))
}

func dstPosToSrcPos(dstPos vec2) vec2 {
	return dstPos.xy - imageDstOrigin() + imageSrc0Origin()
}

// tetra3d Custom Uniform Location //

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {

	depth := imageSrc1UnsafeAt(dstPosToSrcPos(dstPos.xy))
	
	if depth.a > 0 {

		srcOrigin := imageSrc0Origin()
		srcSize := imageSrc0Size()

		// There's atlassing going on behind the scenes here, so:
		// Subtract the source position by the src texture's origin on the atlas.
		// This gives us the actual pixel coordinates.
		tx := srcPos - srcOrigin

		// Divide by the source image size to get the UV coordinates.
		tx /= srcSize

		// Apply fract() to loop the UV coords around [0-1].
		tx = fract(tx)

		// Multiply by the size to get the pixel coordinates again.
		tx *= srcSize

		// tetra3d Custom Fragment Call Location //
		colorTex := imageSrc0UnsafeAt(tx + srcOrigin) * color

		if Fogless == 0 {

			var d float
		
			if FogCurve == 0 {
				d = smoothstep(FogRange[0], FogRange[1], decodeDepth(depth))
			} else if FogCurve == 1 {
				d = smoothstep(FogRange[0], FogRange[1], OutCirc(decodeDepth(depth)))
			} else if FogCurve == 2 {
				d = smoothstep(FogRange[0], FogRange[1], InCirc(decodeDepth(depth)))
			}

			if DitherSize > 0 {

				yc := int(dstPos.y / DitherSize)%4
				xc := int(dstPos.x / DitherSize)%4

				fogMult := step(0, d - BayerMatrix[(yc*4) + xc])

				// Fog mode is 4th channel in Fog vector ([4]float32)
				if Fog.a == 0 {
					colorTex.rgb += Fog.rgb * fogMult * colorTex.a
				} else if Fog.a == 1 {
					colorTex.rgb -= Fog.rgb * fogMult * colorTex.a
				} else if Fog.a == 2 {
					colorTex.rgb = mix(colorTex.rgb, Fog.rgb, fogMult) * colorTex.a
				} else if Fog.a == 3 {
					colorTex.a *= abs(1-d) * step(0, abs(1-d) - BayerMatrix[(yc*4) + xc])
					colorTex.rgb = mix(vec3(0, 0, 0), colorTex.rgb, colorTex.a)
				}

			} else {

				if Fog.a == 0 {
					colorTex.rgb += Fog.rgb * d * colorTex.a
				} else if Fog.a == 1 {
					colorTex.rgb -= Fog.rgb * d * colorTex.a
				} else if Fog.a == 2 {
					colorTex.rgb = mix(colorTex.rgb, Fog.rgb, d) * colorTex.a
				} else if Fog.a == 3 {
					colorTex.a *= abs(1-d)
					colorTex.rgb = mix(vec3(0, 0, 0), colorTex.rgb, colorTex.a)
				}

			}

		}

		return colorTex

	}

	discard()

}

// tetra3d Custom Fragment Definition Location //

`

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
				// Replace teh line with a new ColorTex definition
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
