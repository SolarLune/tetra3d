package tetra3d

import (
	"image"
	"math"
	"strings"

	_ "embed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/solarlune/tetra3d/math32"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

type AnimationLoopMode int

const (
	AnimationLoopModeLoop     AnimationLoopMode = iota // Loop on animation completion
	AnimationLoopModePingPong                          // Reverse on animation completion; if this is the case, the OnFinish() callback is called after two loops (one reversal)
	AnimationLoopModeOneshot                           // Stop on animation completion
)

// NearestPointOnLine returns the closest point along a line spanning from start to end.
func NearestPointOnLine(start, end, point Vector3) Vector3 {

	ab := end.Sub(start)
	t := point.Sub(start).Dot(ab) / ab.Dot(ab)
	return start.Add(ab.Scale(math32.Clamp(t, 0, 1)))

}

// type TextMeasure struct {
// 	X, Y int
// }

func measureText(text string, fontFace font.Face) image.Rectangle {
	splitByNewlines := strings.Split(text, "\n")
	var newBounds image.Rectangle
	if len(splitByNewlines) > 0 {
		for _, line := range splitByNewlines {
			bounds, _ := font.BoundString(fontFace, line)
			lineBounds := image.Rect(int(bounds.Min.X)>>6, int(bounds.Min.Y)>>6, int(bounds.Max.X)>>6, int(bounds.Max.Y)>>6)

			newBounds.Max.X += lineBounds.Dx()
			newBounds.Max.Y += lineBounds.Dy()
		}
	} else {
		bounds, _ := font.BoundString(fontFace, text)
		newBounds = image.Rect(int(bounds.Min.X)>>6, int(bounds.Min.Y)>>6, int(bounds.Max.X)>>6, int(bounds.Max.Y)>>6)
	}

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
			if strings.Contains(line, "func CustomFragment(") {
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
				out += "colorTex = CustomFragment(dstPos, tx + srcOrigin, color)\n\n"
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

// Set represents a Set of elements.
type Set[E comparable] map[E]struct{}

// newSet creates a new set.
func newSet[E comparable]() Set[E] {
	return Set[E]{}
}

func (s Set[E]) Clone() Set[E] {
	newSet := newSet[E]()
	newSet.Combine(s)
	return newSet
}

// Add adds the given elements to a set.
func (s Set[E]) Add(element E) {
	s[element] = struct{}{}
}

// Combine combines the given other elements to the set.
func (s Set[E]) Combine(otherSet Set[E]) {
	for element := range otherSet {
		s.Add(element)
	}
}

// Contains returns if the set contains the given element.
func (s Set[E]) Contains(element E) bool {
	_, ok := s[element]
	return ok
}

// Remove removes the given element from the set.
func (s Set[E]) Remove(element E) {
	delete(s, element)
}

// Clear clears the set.
func (s Set[E]) Clear() {
	clear(s)
	// for v := range s {
	// 	delete(s, v)
	// }
}

// ForEach runs the provided function for each element in the set.
func (s Set[E]) ForEach(f func(element E) bool) {
	for element := range s {
		if !f(element) {
			break
		}
	}

}

type orderedSet[E comparable] []E

// newOrderedSet creates a new OrderedSet.
func newOrderedSet[E comparable]() orderedSet[E] {
	return orderedSet[E]{}
}

func (s orderedSet[E]) Clone() orderedSet[E] {
	newOrderedSet := newOrderedSet[E]()
	newOrderedSet.Combine(s)
	return newOrderedSet
}

// Add adds the given elements to a OrderedSet.
func (s *orderedSet[E]) Add(elements ...E) {
	for _, element := range elements {
		if !s.Contains(element) {
			(*s) = append((*s), element)
		}
	}
}

// Combine combines the given other elements to the OrderedSet.
func (s *orderedSet[E]) Combine(otherOrderedSet orderedSet[E]) {
	for _, element := range otherOrderedSet {
		s.Add(element)
	}
}

// Contains returns if the OrderedSet contains the given element.
func (s *orderedSet[E]) Contains(element E) bool {
	for _, e := range *s {
		if e == element {
			return true
		}
	}
	return false
}

// Remove removes the given element from the OrderedSet.
func (s *orderedSet[E]) Remove(element E) {
	for i, e := range *s {
		if e == element {
			(*s) = append((*s)[:i], (*s)[i+1:]...)
		}
	}
}

// Clear clears the OrderedSet.
func (s *orderedSet[E]) Clear() {
	(*s) = (*s)[:0]
}

// ForEach runs the provided function for each element in the OrderedSet.
func (s orderedSet[E]) ForEach(f func(element E) bool) {
	for _, element := range s {
		if !f(element) {
			return
		}
	}

}

func BlendModeDefault() ebiten.Blend {
	return ebiten.BlendSourceOver
}
func BlendModeAdditive() ebiten.Blend {
	return ebiten.BlendLighter
}
func BlendModeMultiply() ebiten.Blend {
	return ebiten.Blend{
		BlendFactorSourceRGB:        ebiten.BlendFactorDestinationColor,
		BlendFactorSourceAlpha:      ebiten.BlendFactorDestinationColor,
		BlendFactorDestinationRGB:   ebiten.BlendFactorZero,
		BlendFactorDestinationAlpha: ebiten.BlendFactorZero,
		BlendOperationRGB:           ebiten.BlendOperationAdd,
		BlendOperationAlpha:         ebiten.BlendOperationAdd,
	}
}
func BlendModeClear() ebiten.Blend {
	return ebiten.BlendDestinationOut
}

/////

var renderUsingDepthShader *ebiten.Shader

//go:embed shaders/blendWithDepthTexture.kage
var blendWithDepthTextureSrc []byte

func init() {

	ds, err := ebiten.NewShader(blendWithDepthTextureSrc)

	if err != nil {
		panic(err)
	}

	renderUsingDepthShader = ds

}

// Draws src to dst blended with the given depth texture.
// All of the images provided should be the same size.
// The depth texture should be a packed texture, like the one generated by Tetra3d.
// rangeStart is the minimum allowed depth percentage, ranging from 0 to 1. 0 would be the sensible default.
// rangeEnd is the maximum allowed depth percentage, ranging from 0 to 1. 1 would be the sensible default.
// bayerMatrixDitherSize is the size of the dithering size. 0 disables dithering.
// transparencyCurve is the curve used to control fading.
func DrawSrcToDestBlendedWithDepthTexture(dst, src, depth *ebiten.Image, rangeStart, rangeEnd float32, bayerMatrixDitherSize float32, transparencyCurve FogCurve) {

	dst.DrawRectShader(src.Bounds().Dx(), src.Bounds().Dy(), renderUsingDepthShader, &ebiten.DrawRectShaderOptions{
		Images: [4]*ebiten.Image{
			src,
			depth,
		},
		Uniforms: map[string]any{
			"TransparencyRange": []float32{rangeStart, rangeEnd},
			"DitherSize":        bayerMatrixDitherSize,
			"TransparencyCurve": transparencyCurve,
			"BayerMatrix":       bayerMatrix,
		},
	})

}

// func alignmentCheck(s any) {

// 	structType := reflect.TypeOf(s)

// 	runningOffset := 0
// 	for i := 0; i < structType.NumField(); i++ {
// 		f := structType.Field(i)

// 		if runningOffset != int(f.Offset) {
// 			fmt.Println("<<PADDING>>")
// 			runningOffset = int(f.Offset)
// 		}

// 		fmt.Println("Name:", f.Name)
// 		fmt.Println("Offset:", f.Offset)
// 		fmt.Println("Size:", f.Type.Size())
// 		fmt.Println("Type:", f.Type)
// 		fmt.Println("Ideal alignment:", unsafe.Alignof(f))
// 		fmt.Println()
// 		runningOffset += int(f.Type.Size())
// 	}

// }

// Min returns the minimum value out of two provided values.
func Min[number float32 | float64 | int | int32 | int64](x, y number) number {
	return number(math.Min(float64(x), float64(y)))
}

// Max returns the maximum value out of two provided values.
func Max[number float32 | float64 | int | int32 | int64](x, y number) number {
	return number(math.Max(float64(x), float64(y)))
}

// clamp clamps a value to the minimum and maximum values provided.
func clamp[number float32 | float64 | int | int32 | int64](value, min, max number) number {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

//////

func DrawDebugText(screen *ebiten.Image, txtStr string, posX, posY, textScale float32, color Color4) {

	size := measureText(txtStr, basicfont.Face7x13).Size()

	if debugTextTexture == nil {
		debugTextTexture = ebiten.NewImage(size.X, size.Y+13)
	} else if size.X > debugTextTexture.Bounds().Dx() || size.Y > debugTextTexture.Bounds().Dy() {
		debugTextTexture = ebiten.NewImage(max(size.X, debugTextTexture.Bounds().Dx()), max(size.Y, debugTextTexture.Bounds().Dy())+13)
	}

	debugTextTexture.Clear()

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(0, 13)
	text.DrawWithOptions(debugTextTexture, txtStr, basicfont.Face7x13, opt)

	// TODO: This is slow, this could be way faster by drawing once to an image and then drawing that result

	dr := &ebiten.DrawImageOptions{}
	dr.ColorScale.Scale(0, 0, 0, 1)

	periodOffset := float32(8.0)

	for y := -1; y < 2; y++ {

		for x := -1; x < 2; x++ {

			dr.GeoM.Reset()
			dr.GeoM.Translate(float64(posX+4+float32(x)), float64(posY+4+float32(y)+periodOffset))
			dr.GeoM.Scale(float64(textScale), float64(textScale))

			screen.DrawImage(debugTextTexture, dr)
		}

	}

	dr.ColorScale.Reset()
	dr.ColorScale.ScaleWithColor(color.ToNRGBA64())

	dr.GeoM.Reset()
	dr.GeoM.Translate(float64(posX+4), float64(posY+4+periodOffset))
	dr.GeoM.Scale(float64(textScale), float64(textScale))

	screen.DrawImage(debugTextTexture, dr)

}
