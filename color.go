package tetra3d

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/solarlune/tetra3d/math32"
	"github.com/tanema/gween/ease"
)

// Color represents a color, containing R, G, B, and A components, each expected to range from 0 to 1.
type Color struct {
	R, G, B, A float32
}

// NewColor returns a new Color, with the provided R, G, B, and A components expected to range from 0 to 1.
func NewColor(r, g, b, a float32) Color {
	return Color{r, g, b, a}
}

// NewColorFromColor returns a new Color based on a provided color.Color.
func NewColorFromColor(color color.Color) Color {

	r, g, b, a := color.RGBA()

	return NewColor(
		float32(r)/65535,
		float32(g)/65535,
		float32(b)/65535,
		float32(a)/65535,
	)

}

// NewColorRandom creates a randomized color, with each component lying between the minimum and maximum values.
func NewColorRandom(min, max float32, grayscale bool) Color {
	color := NewColor(1, 1, 1, 1)
	diff := max - min
	if grayscale {
		r := min + (diff * rand.Float32())
		color.R = r
		color.G = r
		color.B = r
	} else {
		color.R = min + (diff * rand.Float32())
		color.G = min + (diff * rand.Float32())
		color.B = min + (diff * rand.Float32())
	}
	return color
}

// AddRGBA adds the provided R, G, B, and A values to the color as provided. The components are expected to range from 0 to 1.
func (color Color) AddRGBA(r, g, b, a float32) Color {
	color.R += r
	color.G += g
	color.B += b
	color.A += a
	return color
}

// Add adds the provided Color to the existing Color.
func (color Color) Add(other Color) Color {
	return color.AddRGBA(other.ToFloat32s())
}

// MultiplyRGBA multiplies the color's RGBA channels by the provided R, G, B, and A scalar values.
func (color Color) MultiplyRGBA(scalarR, scalarG, scalarB, scalarA float32) Color {
	color.R *= scalarR
	color.G *= scalarG
	color.B *= scalarB
	color.A *= scalarA
	return color
}

func (color Color) MultiplyScalarRGB(scalar float32) Color {
	return color.MultiplyRGBA(scalar, scalar, scalar, 1)
}

func (color Color) MultiplyScalarRGBA(scalar float32) Color {
	return color.MultiplyRGBA(scalar, scalar, scalar, scalar)
}

// Multiply multiplies the existing Color by the provided Color.
func (color Color) Multiply(other Color) Color {
	return color.MultiplyRGBA(other.ToFloat32s())
}

// Sub subtracts the other Color from the calling Color instance.
func (color Color) SubRGBA(r, g, b, a float32) Color {
	color.R -= r
	color.G -= g
	color.B -= b
	color.A -= a
	return color
}

// Sub subtracts the other Color from the calling Color instance.
func (color Color) Sub(other Color) Color {
	return color.SubRGBA(other.ToFloat32s())
}

// Mix mixes the calling Color with the other Color, mixed to the percentage given (ranging from 0 - 1).
func (color Color) Mix(other Color, percentage float32) Color {

	// percentage = math32.Clamp(percentage, 0, 1)
	color.R += (other.R - color.R) * percentage
	color.G += (other.G - color.G) * percentage
	color.B += (other.B - color.B) * percentage
	color.A += (other.A - color.A) * percentage
	return color

}

// AddAlpha adds the provided alpha amount to the Color
func (c Color) AddAlpha(alpha float32) Color {
	c.A += alpha
	return c
}

// SubAlpha adds the provided alpha amount to the Color
func (c Color) SubAlpha(alpha float32) Color {
	c.A -= alpha
	return c
}

// SetAlpha returns a copy of the the Color with the alpha set to the provided alpha value.
func (color Color) SetAlpha(alpha float32) Color {
	color.A = alpha
	return color
}

// ToFloat32s returns the Color as four float32 in the order R, G, B, and A.
func (color Color) ToFloat32s() (float32, float32, float32, float32) {
	return color.R, color.G, color.B, color.A
}

// Tofloat32s returns four float32 values for each channel in the Color in the order R, G, B, and A.
func (color Color) Tofloat32s() (float32, float32, float32, float32) {
	return float32(color.R), float32(color.G), float32(color.B), float32(color.A)
}

// ToFloat32Array returns a [4]float32 array for each channel in the Color in the order of R, G, B, and A.
func (color Color) ToFloat32Array() [4]float32 {
	return [4]float32{float32(color.R), float32(color.G), float32(color.B), float32(color.A)}
}

// ToRGBA64 converts a color to a color.RGBA64 instance.
func (c Color) ToRGBA64() color.RGBA64 {
	return color.RGBA64{
		c.capRGBA64(c.R),
		c.capRGBA64(c.G),
		c.capRGBA64(c.B),
		c.capRGBA64(c.A),
	}
}

// ToNRGBA64 converts a color to a color.NRGBA64 (non-alpha color multiplied) color instance.
func (c Color) ToNRGBA64() color.NRGBA64 {
	return color.NRGBA64{
		c.capRGBA64(c.R),
		c.capRGBA64(c.G),
		c.capRGBA64(c.B),
		c.capRGBA64(c.A),
	}
}

func (color Color) capRGBA64(value float32) uint16 {
	if value > 1 {
		value = 1
	} else if value < 0 {
		value = 0
	}
	return uint16(value * math.MaxUint16)
}

// ConvertTosRGB() converts the color's R, G, and B components to the sRGB color space. This is used to convert
// colors from their values in GLTF to how they should appear on the screen. See: https://en.wikipedia.org/wiki/SRGB
func (color Color) ConvertTosRGB() Color {

	if color.R <= 0.0031308 {
		color.R *= 12.92
	} else {
		color.R = float32(1.055*math32.Pow(float32(color.R), 1/2.4) - 0.055)
	}

	if color.G <= 0.0031308 {
		color.G *= 12.92
	} else {
		color.G = float32(1.055*math32.Pow(float32(color.G), 1/2.4) - 0.055)
	}

	if color.B <= 0.0031308 {
		color.B *= 12.92
	} else {
		color.B = float32(1.055*math32.Pow(float32(color.B), 1/2.4) - 0.055)
	}

	return color

}

// Lerp linearly interpolates the color from the starting color to the target by the percentage given.
func (c Color) Lerp(other Color, percentage float32) Color {

	percentage = math32.Clamp(percentage, 0, 1)
	c.R += (other.R - c.R) * float32(percentage)
	c.G += (other.G - c.G) * float32(percentage)
	c.B += (other.B - c.B) * float32(percentage)
	c.A += (other.A - c.A) * float32(percentage)

	return c
}

// Lerp linearly interpolates the color from the starting color to the target by the percentage given.
func (c Color) LerpRGBA(r, g, b, a, percentage float32) Color {

	percentage = math32.Clamp(percentage, 0, 1)
	c.R += (float32(r) - c.R) * float32(percentage)
	c.G += (float32(g) - c.G) * float32(percentage)
	c.B += (float32(b) - c.B) * float32(percentage)
	c.A += (float32(a) - c.A) * float32(percentage)

	return c
}

func (color Color) String() string {
	return fmt.Sprintf("<%0.2f, %0.2f, %0.2f, %0.2f>", color.R, color.G, color.B, color.A)
}

// NewColorFromHSV returns a new color, using hue, saturation, and value numbers, each ranging from 0 to 1. A hue of
// 0 is red, while 1 is also red, but on the other end of the spectrum.
// Cribbed from: https://github.com/lucasb-eyer/go-colorful/blob/master/colors.go
func NewColorFromHSV(h, s, v float32) Color {

	for h >= 1 {
		h--
	}
	for h < 0 {
		h++
	}

	h *= 360

	if s > 1 {
		s = 1
	} else if s < 0 {
		s = 0
	}

	if v > 1 {
		v = 1
	} else if v < 0 {
		v = 0
	}

	Hp := float32(h / 60.0)
	C := v * s
	X := C * (1.0 - math32.Abs(math32.Mod(Hp, 2.0)-1.0))

	m := v - C
	r, g, b := float32(0), float32(0), float32(0)

	switch {
	case 0.0 <= Hp && Hp < 1.0:
		r = C
		g = X
	case 1.0 <= Hp && Hp < 2.0:
		r = X
		g = C
	case 2.0 <= Hp && Hp < 3.0:
		g = C
		b = X
	case 3.0 <= Hp && Hp < 4.0:
		g = X
		b = C
	case 4.0 <= Hp && Hp < 5.0:
		r = X
		b = C
	case 5.0 <= Hp && Hp < 6.0:
		r = C
		b = X
	}

	return Color{float32(m + r), float32(m + g), float32(m + b), 1}
}

func NewColorFromHexString(hex string) Color {

	c := NewColor(0, 0, 0, 1)

	hex = strings.TrimPrefix(hex, "#")

	if len(hex) >= 2 {
		v, _ := strconv.ParseInt(hex[:2], 16, 32)
		c.R = float32(v) / 256.0

		if len(hex) >= 4 {
			v, _ := strconv.ParseInt(hex[2:4], 16, 32)
			c.G = float32(v) / 256.0
		} else {
			c.G = 1
		}

		if len(hex) >= 6 {
			v, _ := strconv.ParseInt(hex[4:6], 16, 32)
			c.B = float32(v) / 256.0
		} else {
			c.B = 1
		}

		if len(hex) >= 8 {
			v, _ := strconv.ParseInt(hex[6:8], 16, 32)
			c.A = float32(v) / 256.0
		} else {
			c.A = 1
		}

	}

	return c

}

// Hue returns the hue of the color as a value ranging from 0 to 1.
func (color Color) Hue() float32 {
	// Function cribbed from: https://github.com/lucasb-eyer/go-colorful/blob/master/colors.go

	r := float32(color.R)
	g := float32(color.G)
	b := float32(color.B)

	min := math32.Min(math32.Min(r, g), b)
	v := math32.Max(math32.Max(r, g), b)
	C := v - min

	h := float32(0)
	if min != v {
		if v == r {
			h = math32.Mod((g-b)/C, 6.0)
		}
		if v == g {
			h = (b-r)/C + 2.0
		}
		if v == b {
			h = (r-g)/C + 4.0
		}
		h *= 60.0
		if h < 0.0 {
			h += 360.0
		}
	}
	return h / 360
}

// Saturation returns the saturation of the color as a value ranging from 0 to 1.
func (color Color) Saturation() float32 {

	r := float32(color.R)
	g := float32(color.G)
	b := float32(color.B)

	min := math32.Min(math32.Min(r, g), b)
	v := math32.Max(math32.Max(r, g), b)
	C := v - min

	s := float32(0)
	if v != 0.0 {
		s = C / v
	}

	return s
}

// Value returns the value of the color as a value, ranging from 0 to 1.
func (color Color) Value() float32 {

	r := float32(color.R)
	g := float32(color.G)
	b := float32(color.B)

	return math32.Max(math32.Max(r, g), b)
}

// func (color Color) HSV() (float32, float32, float32) {

// 	r := float32(color.R)
// 	g := float32(color.G)
// 	b := float32(color.B)

// 	min := math32.Min(math32.Min(r, g), b)
// 	v := math32.Max(math32.Max(r, g), b)
// 	C := v - min

// 	s := 0.0
// 	if v != 0.0 {
// 		s = C / v
// 	}

// 	h := 0.0
// 	if min != v {
// 		if v == r {
// 			h = math32.Mod((g-b)/C, 6.0)
// 		}
// 		if v == g {
// 			h = (b-r)/C + 2.0
// 		}
// 		if v == b {
// 			h = (r-g)/C + 4.0
// 		}
// 		h *= 60.0
// 		if h < 0.0 {
// 			h += 360.0
// 		}
// 	}
// 	return h / 360, s, v
// }

// SetHue returns a copy of the color with the hue set to the specified value.
// Hue goes through the rainbow, starting with red, and ranges from 0 to 1.
func (color Color) SetHue(h float32) Color {
	return NewColorFromHSV(h, color.Saturation(), color.Value())
}

// SetSaturation returns a copy of the color with the saturation of the color set to the specified value.
// Saturation ranges from 0 to 1.
func (color Color) SetSaturation(s float32) Color {
	return NewColorFromHSV(color.Hue(), s, color.Value())
}

// SetValue returns a copy of the color with the value of the color set to the specified value.
// Value ranges from 0 to 1.
func (color Color) SetValue(v float32) Color {
	return NewColorFromHSV(color.Hue(), color.Saturation(), v)
}

func (c Color) IsZero() bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0
}

// ColorCurvePoint indicates an individual color point in a color curve.
type ColorCurvePoint struct {
	Color      Color
	Percentage float32
}

// ColorCurve represents a range of colors that a value of 0 to 1 can interpolate between.
type ColorCurve struct {
	Points         []ColorCurvePoint
	EasingFunction ease.TweenFunc
}

// NewColorCurve creats a new ColorCurve composed of the colors given, evenly spaced throughout the curve.
// If no colors are given, the ColorCurve is still valid - it's just empty.
func NewColorCurve(colors ...Color) ColorCurve {
	cc := ColorCurve{
		Points:         []ColorCurvePoint{},
		EasingFunction: ease.Linear,
	}

	if len(colors) == 1 {
		cc.Add(colors[0], 0)
	} else if len(colors) > 1 {
		for i, color := range colors {
			cc.Add(color, float32(i)/float32(len(colors)-1))
		}
	}

	return cc
}

// Clone creates a duplicate ColorCurve.
func (cc ColorCurve) Clone() ColorCurve {
	ncc := NewColorCurve()
	ncc.Points = append(ncc.Points, cc.Points...)
	return ncc
}

// Add adds a color point to the ColorCurve with the color and percentage provided (from 0-1).
func (cc *ColorCurve) Add(color Color, percentage float32) {
	cc.AddRGBA(color.R, color.G, color.B, color.A, percentage)
}

// AddRGBA adds a point to the ColorCurve, with r, g, b, and a being the color and the percentage being a number between 0 and 1 indicating the .
func (cc *ColorCurve) AddRGBA(r, g, b, a float32, percentage float32) {

	if percentage > 1 {
		percentage = 1
	} else if percentage < 0 {
		percentage = 0
	}

	cc.Points = append(cc.Points, ColorCurvePoint{
		Color:      NewColor(r, g, b, a),
		Percentage: percentage,
	})

	sort.Slice(cc.Points, func(i, j int) bool { return cc.Points[i].Percentage < cc.Points[j].Percentage })
}

// Color returns the Color for the given percentage in the color curve. For example, if you have a curve composed of
// the colors {0, 0, 0, 0} at 0 and {1, 1, 1, 1} at 1, then calling Curve.Color(0.5) would return {0.5, 0.5, 0.5, 0.5}.
// If the curve doesn't have any color curve points, Color will return transparent.
func (cc ColorCurve) Color(perc float32) Color {

	var c Color

	for i := 0; i < len(cc.Points); i++ {

		c = cc.Points[i].Color

		if i >= len(cc.Points)-1 || cc.Points[i].Percentage >= perc {
			break
		}

		if cc.Points[i].Percentage <= perc && cc.Points[i+1].Percentage >= perc {
			// c = cc.Points[i].Color.Mix(cc.Points[i+1].Color, float32((perc-cc.Points[i].Percentage)/(cc.Points[i+1].Percentage-cc.Points[i].Percentage)))
			pp := (perc - cc.Points[i].Percentage)

			c = cc.Points[i].Color.Mix(cc.Points[i+1].Color, cc.EasingFunction(float32(pp), 0, 1, float32(cc.Points[i+1].Percentage-cc.Points[i].Percentage)))
			break
		}

	}

	return c

}

// ColorTween represents an object that tweens across a ColorCurve.
type ColorTween struct {
	ColorCurve ColorCurve // ColorCurve is the curve to use while tweening
	Percent    float32    // Percent is the percentage through the tween the ColorTween is
	Speed      float32    // Speed is the speed of the tween; if you want a
	Duration   float32    // Duration is the duration of the tween in seconds
	playing    bool
}

// ColorTween creates a new ColorTween object with the specified duration and curve.
func NewColorTween(duration float32, curve ColorCurve) ColorTween {
	ct := ColorTween{
		ColorCurve: curve,
		Speed:      1,
		Duration:   duration,
	}
	return ct
}

// Update updates the ColorTween using the delta value provided in seconds and returns if the tween is finished or not.
func (c *ColorTween) Update(dt float32) bool {
	if c.playing {
		c.Percent += dt / c.Duration * c.Speed
		if (c.Percent >= 1 && c.Speed > 0) || (c.Percent <= 0 && c.Speed < 0) {
			c.playing = false
			c.Percent = math32.Clamp(c.Percent, 0, 1)
			return true
		}
	}
	return false
}

// Play starts playing the ColorTween back.
func (c *ColorTween) Play() {
	c.playing = true
}

// Stop stops the ColorTween.
func (c *ColorTween) Stop() {
	c.playing = false
}

// IsPlaying returns if the ColorTween is playing.
func (c *ColorTween) IsPlaying() bool {
	return c.playing
}

// Reset resets the ColorTween.
func (c *ColorTween) Reset() {
	c.Percent = 0
}

// Color returns the current color from the internal ColorCurve.
func (c *ColorTween) Color() Color {
	return c.ColorCurve.Color(c.Percent)
}

// SetEasingFunction sets the easing function to be used by the tween and is present on the ColorCurve.
func (c *ColorTween) SetEasingFunction(easingFunction ease.TweenFunc) {
	c.ColorCurve.EasingFunction = easingFunction
}
