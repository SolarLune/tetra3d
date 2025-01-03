package tetra3d

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/solarlune/tetra3d/math32"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"

	_ "embed"
)

type TextHorizontalAlignment int
type TextVerticalAlignment int

const (
	TextAlignHorizontalLeft   TextHorizontalAlignment = iota // Left aligned text. The text hugs the left side of the texture.
	TextAlignHorizontalCenter                                // Center aligned text. All text lines are centered horizontally in the texture's width.
	TextAlignHorizontalRight                                 // Right aligned text. The text hugs the right side of the texture.

	TextAlignVerticalTop    TextVerticalAlignment = iota // Top aligned text. The text hugs the top side of the texture.
	TextAlignVerticalCenter                              // Center aligned text. All text lines are centered horizontally in the texture's width.
	TextAlignVerticalBottom                              // Bottom aligned text. The text hugs the bottom side of the texture.
)

type TextStyle struct {
	Font                 font.Face               // The font face to use for rendering the text. The size is customizeable, but the DPI should be 72.
	Cursor               string                  // A cursor string sequence is drawn at the end while typewriter-ing; defaults to a blank string ("").
	LineHeightMultiplier float32                 // The multiplier for line height changes.
	AlignmentHorizontal  TextHorizontalAlignment // How the text should be horizontally aligned in the Text texture.
	AlignmentVertical    TextVerticalAlignment

	BGColor Color // The Background color for the text. Defaults to black (0, 0, 0, 1).
	FGColor Color // The Foreground color for the text. Defaults to white (1, 1, 1, 1).

	ShadowDirection Vector3 // A vector indicating direction of the shadow's heading. Defaults to down-right ( {1, 1, 0}, normalized ).
	ShadowLength    int     // The length of the shadow in pixels. Defaults to 0 (no shadow).
	ShadowColorNear Color   // The color of the shadow near the letters. Defaults to black (0, 0, 0, 1).
	ShadowColorFar  Color   // The color of the shadow towards the end of the letters. Defaults to black (0, 0, 0, 1).

	OutlineThickness int   // Overall thickness of the outline in pixels. Defaults to 0 (no outline).
	OutlineRounded   bool  // If the outline is rounded or not. Defaults to false (square outlines).
	OutlineColor     Color // Color of the outline. Defaults to black (0, 0, 0, 1).

	// Margin (in pixels) of space to leave around the frame of the texture (left or right, depending on HorizontalAlignment, and from the top). Defaults to 0.
	MarginHorizontal, MarginVertical int

	// Manual offsets to positioning
	OffsetX, OffsetY int
}

func NewDefaultTextStyle() TextStyle {
	return TextStyle{
		Font:                 basicfont.Face7x13,
		LineHeightMultiplier: 1,
		BGColor:              NewColor(0, 0, 0, 1),
		FGColor:              NewColor(1, 1, 1, 1),

		OutlineColor: NewColor(0, 0, 0, 1),

		ShadowDirection: Vector3{1, 1, 0}.Unit(),
		ShadowColorNear: NewColor(0, 0, 0, 1),
		ShadowColorFar:  NewColor(0, 0, 0, 1),
	}
}

// Text represents a helper object that writes text for display as a texture on a Model's MeshPart.
// Text objects use a pre-made shader to render.
type Text struct {
	meshPart *MeshPart     // The MeshPart that the Text is operating on
	Texture  *ebiten.Image // The texture used to render text

	style TextStyle

	setText         string
	parsedText      []string
	typewriterIndex int
	typewriterOn    bool
	textureSize     int
}

//go:embed shaders/text.kage
var textShaderSrc []byte

// NewText creates a new Text rendering surface for typing out text and assigns the MeshPart provided to use that surface as a texture.
// If the MeshPart has no Material, then a new one will be created with sane default settings.
// NewText sets the transparency mode of the material to be transparent, as clip alpha doesn't work properly.
// textureWidth is how wide (in pixels) the backing texture should be for displaying the text; the height is determined by the
// aspect ratio of the dimensions of the meshpart given.
// Text objects update the mesh's material to point to an internal texture, and use a shader as well. If you want to tweak the rendering further,
// do so on the provided MeshPart after calling NewText().
// All changes to the Text require that the Text object updates its texture, which can be costly as this means redrawing the text as
// necessary; this is handled automatically.
// Note that for this to work ideally, the mesh cannot be rotated (i.e. the mesh's faces should not be at an angle).
// The function will return an error if the UV values for the vertices don't cover a large enough range (i.e. if the plane doesn't cover
// the entire text texture, from 0,0 to 1,1). This isn't necessarily a problem, but can indicate an issue that would make text not render.
func NewText(meshPart *MeshPart, textureWidth int) (*Text, error) {

	text := &Text{
		meshPart:    meshPart,
		textureSize: textureWidth,
	}

	// Calculate the width and height of the dimensions based off of the
	// meshpart's vertex positions; this determines our texture's aspect ratio.
	w, h := meshPart.primaryDimensions()

	asr := float32(h) / float32(w)

	text.Texture = ebiten.NewImage(textureWidth, int(float32(textureWidth)*asr))

	if meshPart.Material == nil {
		// If no material is present, then we can create a new one with sane defaults
		meshPart.Material = NewMaterial("text material")
		meshPart.Material.BackfaceCulling = true
		meshPart.Material.Shadeless = true
	} else {
		// We have to clone the material to ensure that unique objects that share the same material can both
		// have their own Text textures.
		meshPart.Material = meshPart.Material.Clone()
	}

	meshPart.Material.FragmentShaderOn = true
	meshPart.Material.Texture = text.Texture

	// We set this because Alpha Clip doesn't work with shadows / outlines, as just the text itself writes depth values
	meshPart.Material.TransparencyMode = TransparencyModeTransparent

	// shader, err := ebiten.NewShader(textShaderSrc)
	shader, err := ExtendBase3DShader(string(textShaderSrc))
	if err != nil {
		panic(err)
	}

	meshPart.Material.SetShader(shader)

	// We set the default text here so that something appears, and we
	// apply a style using the function because otherwise, the text would be invisible.
	text.setText = "Default text"
	text.parsedText = []string{"Default text"}

	text.SetStyle(NewDefaultTextStyle()) // The texture will update when we apply the style.

	uvMin := Vector3{math.MaxFloat32, math.MaxFloat32, 0}
	uvMax := Vector3{-math.MaxFloat32, -math.MaxFloat32, 0}

	meshPart.ForEachVertexIndex(func(vertIndex int) {
		uv := meshPart.Mesh.VertexUVs[vertIndex]

		if uv.X < uvMin.X {
			uvMin.X = uv.X
		}
		if uv.Y < uvMin.Y {
			uvMin.Y = uv.Y
		}

		if uv.X > uvMax.X {
			uvMax.X = uv.X
		}
		if uv.Y > uvMax.Y {
			uvMax.Y = uv.Y
		}

	}, false)

	err = nil

	if uvMax.X-uvMin.X < 0.99 || uvMax.Y-uvMin.Y < 0.99 {
		err = fmt.Errorf("warning: uv values for text mesh part bounds have a small or unusual bounds of: %s : %s", uvMin, uvMax)
	}

	return text, err
}

// NewTextAutoSize creates a new Text rendering surface for typing out text,
// with the backing texture's size calculated from an orthographic Camera's render scale.
// This is generally useful for UI Text objects.
// Note that for this to work ideally, the mesh cannot be rotated (i.e. the mesh's faces should not be at an angle).
// All changes to the Text require that the Text object updates its texture, which can be costly as this means redrawing the text as
// necessary; this is handled automatically.
// The function will return an error if the UV values for the vertices don't cover a large enough range (i.e. if the plane doesn't cover
// the entire text texture, from 0,0 to 1,1). This isn't necessarily a problem, but can indicate an issue that would make text not render.
func NewTextAutoSize(meshPart *MeshPart, camera *Camera) (*Text, error) {
	w, _ := camera.Size()

	meshPartDimWidth, _ := meshPart.primaryDimensions()

	texWidth := math32.Round(meshPartDimWidth / camera.OrthoScale() * float32(w))

	return NewText(meshPart, int(texWidth))
}

// Clone clones the Text object.
func (text *Text) Clone() *Text {

	newText, _ := NewText(text.meshPart, text.textureSize)
	newText.typewriterIndex = text.typewriterIndex
	newText.typewriterOn = text.typewriterOn
	newText.setText = text.setText
	newText.parsedText = append([]string{}, text.parsedText...)
	newText.Texture = ebiten.NewImageFromImage(text.Texture)
	newText.textureSize = text.textureSize
	newText.style = text.style
	return newText

}

// Text returns the current text that is being displayed for the Text object.
func (text *Text) Text() string {
	return text.setText
}

func splitWithSeparator(str string, seps string) []string {

	output := []string{}
	start := 0

	index := strings.IndexAny(str, seps)

	if index < 0 {
		return []string{str}
	}

	for index >= 0 {
		end := start + index + 1
		if end > len(str) {
			end = len(str)
		}
		output = append(output, str[start:end])
		start += index + 1
		index = strings.IndexAny(str[start:], seps)
		if index < 0 {
			output = append(output, str[start:])
		}
	}

	return output

}

// SetText sets the text to be displayed for the Text object.
// arguments can be variables to be displayed in the text string, formatted using fmt.Sprintf()'s formatting rules.
// Text objects handle automatically splitting newlines based on length to the owning plane mesh's size.
// Setting the text to be blank clears the text, though Text.ClearText() also exists, and is just syntactic sugar for this purpose.
// SetText accounts for the margin set in the Text object's active TextStyle, but if it is applied prior to calling SetText().
func (textObj *Text) SetText(txt string, arguments ...any) *Text {

	if len(arguments) > 0 {
		txt = fmt.Sprintf(txt, arguments...)
	}

	if textObj.setText != txt {

		textObj.setText = txt

		textureWidth := textObj.Texture.Bounds().Dx()

		// If a word gets too close to the texture's right side, we loop
		safetyMargin := int(float32(textureWidth)*0.1) + textObj.style.MarginHorizontal

		parsedText := []string{}

		for _, line := range strings.Split(txt, "\n") {

			split := splitWithSeparator(line, " -")

			runningMeasure := 0
			wordIndex := 0

			// Some fonts have space characters that are basically empty somehow...?
			spaceAdd := 0
			if measureText(" ", textObj.style.Font).Dx() <= 0 {
				spaceAdd = measureText("M", textObj.style.Font).Dx()
			}

			for i, word := range split {
				wordSpace := measureText(word, textObj.style.Font).Dx()
				runningMeasure += wordSpace + spaceAdd

				if runningMeasure >= textureWidth-safetyMargin {
					t := strings.Join(split[wordIndex:i], "")
					parsedText = append(parsedText, t)

					runningMeasure = wordSpace
					wordIndex = i

					// if i == len(split)-1 {
					// 	parsedText = append(parsedText, strings.Join(split[wordIndex:], ""))
					// }

				}

			}

			t := strings.Join(split[wordIndex:], "")
			parsedText = append(parsedText, t)

		}

		textObj.parsedText = parsedText

		if textObj.typewriterIndex >= 0 {
			textObj.typewriterIndex = 0
		}

		textObj.UpdateTexture()

	}

	return textObj

}

// ClearText clears the text displaying in the Text Object.
func (textObj *Text) ClearText() *Text {
	return textObj.SetText("")
}

// UpdateTexture will update the Text's backing texture, clearing and/or redrawing the texture as necessary.
// This won't do anything if the texture is nil (has been disposed).
func (textObj *Text) UpdateTexture() {

	if textObj.Texture == nil {
		return
	}

	typewriterIndex := textObj.typewriterIndex
	if !textObj.typewriterOn {
		typewriterIndex = len(textObj.parsedText) - 1
	}

	textLineMargin := 2
	lineHeight := int(float32(textObj.style.Font.Metrics().Height.Ceil() + textLineMargin))
	multipliedLineHeight := int(float32(lineHeight) * textObj.style.LineHeightMultiplier)
	ascent := textObj.style.Font.Metrics().Ascent.Ceil()

	typing := true

	// if textObj.style.BGColor != nil {
	// 	textObj.Texture.Fill(textObj.style.BGColor.ToRGBA64())
	// } else {
	textObj.Texture.Clear()
	// }

	textureWidth := textObj.Texture.Bounds().Dx()
	textureHeight := textObj.Texture.Bounds().Dy()

	blockHeight := math32.Max(len(textObj.parsedText)*multipliedLineHeight, lineHeight)

	for lineIndex, line := range textObj.parsedText {

		measure := measureText(line, textObj.style.Font)

		if textObj.typewriterOn && typewriterIndex >= 0 {

			if !typing {
				break
			}

			if typewriterIndex > len(line) {
				typewriterIndex -= len(line)
			} else if typing {
				line = line[:typewriterIndex]
				typing = false
			}

		}

		x := -measure.Min.X

		y := -measure.Min.Y + (lineIndex * multipliedLineHeight)

		if textObj.style.AlignmentHorizontal == TextAlignHorizontalCenter {
			x = textureWidth/2 - measure.Dx()/2
		} else if textObj.style.AlignmentHorizontal == TextAlignHorizontalRight {
			x = textureWidth - measure.Dx()
			x -= textObj.style.MarginHorizontal
		} else {
			x += textObj.style.MarginHorizontal
		}

		if textObj.style.AlignmentVertical == TextAlignVerticalCenter {
			// We add the minimum because the height shouldn't probably include parts of text
			// that drop below the baseline (e.g. measure.Dy())
			y += textureHeight/2 - blockHeight/2 + ascent/2
		} else if textObj.style.AlignmentVertical == TextAlignVerticalBottom {
			y += textureHeight - blockHeight
			y -= textObj.style.MarginVertical
		} else {
			y += textObj.style.MarginVertical
		}

		x += textObj.style.OffsetX
		y += textObj.style.OffsetY

		if textObj.typewriterOn && (len(line) < len(textObj.parsedText[lineIndex]) || lineIndex == len(textObj.parsedText)-1) {
			line += textObj.style.Cursor
		}

		text.Draw(textObj.Texture, line, textObj.style.Font, x, y, color.RGBA{255, 255, 255, 255})
		// text.Draw(textObj.Texture, line, textObj.style.Font, x, y, textObj.style.FGColor.ToRGBA64())

	}

}

func (text *Text) Style() TextStyle {
	return text.style
}

func (text *Text) SetStyle(style TextStyle) {
	if text.style != style {

		oldStyle := text.style
		text.style = style

		rounded := float32(0)
		if style.OutlineRounded {
			rounded = 1
		}

		shadowVec := style.ShadowDirection.Unit().Invert()

		uniformMap := map[string]any{
			"OutlineThickness": float32(style.OutlineThickness),
			"OutlineRounded":   rounded,
			"ShadowVector":     [2]float32{float32(shadowVec.X), float32(shadowVec.Y)},
			"ShadowLength":     float32(style.ShadowLength),
			"BGColor":          style.BGColor.ToFloat32Array(),
			"FGColor":          style.FGColor.ToFloat32Array(),
			"OutlineColor":     style.OutlineColor.ToFloat32Array(),
			"ShadowColorNear":  style.ShadowColorNear.ToFloat32Array(),
			"ShadowColorFar":   style.ShadowColorFar.ToFloat32Array(),
		}

		if !style.ShadowColorFar.IsZero() {
			uniformMap["ShadowColorFarSet"] = 1.0
		}

		text.meshPart.Material.FragmentShaderOptions = &ebiten.DrawTrianglesShaderOptions{
			Images: [4]*ebiten.Image{
				text.Texture,
			},
			Uniforms: uniformMap,
		}

		// If the font changes, we have to set the text again to ensure the text wraps properly.
		if style.Font != oldStyle.Font {
			setText := text.setText
			text.setText = ""
			text.SetText(setText)
		}

		text.UpdateTexture()

	}
}

// TypewriterIndex returns the typewriter index of the Text object.
func (text *Text) TypewriterIndex() int {
	return text.typewriterIndex
}

// SetTypewriterIndex sets the typewriter scroll of the text to the value given.
func (text *Text) SetTypewriterIndex(typewriterIndex int) {

	oldIndex := text.typewriterIndex

	text.typewriterIndex = typewriterIndex

	if text.typewriterIndex >= len(text.setText) {
		text.typewriterIndex = len(text.setText)
	}
	if text.typewriterIndex < 0 {
		text.typewriterIndex = 0
	}

	if text.typewriterOn && oldIndex != text.typewriterIndex {
		text.UpdateTexture()
	}

}

// FinishTypewriter finishes the typewriter effect, so that the entire message is visible.
func (text *Text) FinishTypewriter() {
	text.SetTypewriterIndex(len(text.setText))
}

// AdvanceTypewriterIndex advances the scroll of the text by the number of characters given.
// AdvanceTypewriterIndex will return a boolean value indicating if the Text advanced to the end
// or not.
func (text *Text) AdvanceTypewriterIndex(advanceBy int) bool {
	oldIndex := text.typewriterIndex
	adv := text.typewriterIndex + advanceBy
	if text.typewriterIndex == math.MaxInt {
		adv = 0
	}
	text.SetTypewriterIndex(adv)
	if advanceBy > 0 {
		return oldIndex >= len(text.setText)
	} else if advanceBy < 0 {
		return oldIndex <= 0
	}
	return false
}

// TypewriterFinished returns if the typewriter effect is finished.
func (text *Text) TypewriterFinished() bool {
	return text.typewriterIndex >= len(text.setText)
}

// SetTypewriterOn sets the typewriter effect on the Text object.
func (text *Text) SetTypewriterOn(on bool) {
	if text.typewriterOn != on {
		text.UpdateTexture()
	}
	text.typewriterOn = on
}

// TypewriterOn returns if the typewriter effect is enabled on the Text object.
func (text *Text) TypewriterOn() bool {
	return text.typewriterOn
}

// Dispose disposes of the text object's backing texture; this needs to be called to free VRAM, and should be called
// whenever the owning Model and Mesh are no longer is going to be used.
// This also will set the texture of the MeshPart this Text object is tied to, to nil.
func (text *Text) Dispose() {
	if text.Texture != nil {
		text.Texture.Dispose()
		text.Texture = nil
		text.meshPart.Material.Texture = nil
	}
}

// type Text struct {
// 	*Node

// 	textModel       *Model
// 	font            font.Face
// 	setText         string
// 	typewriterIndex int
// 	texture         *ebiten.Image
// 	lineHeight      float32
// }

// func NewText(name string, lineHeight float32) *Text {
// 	text := &Text{
// 		Node:            NewNode(name),
// 		textModel:       NewModel(NewSubdividedPlaneMesh(4, 4), "text model"),
// 		font:            basicfont.Face7x13,
// 		typewriterIndex: -1,
// 		lineHeight:      lineHeight,
// 	}

// 	// text.textModel.Mesh.SelectVertices().SelectAll().ApplyMatrix(NewMatrix4Scale(0.5, 0.5, 0.5))

// 	mat := text.textModel.Mesh.Materials()[0]
// 	mat.BackfaceCulling = true
// 	mat.TransparencyMode = TransparencyModeTransparent

// 	text.AddChildren(text.textModel)
// 	return text
// }

// func (text *Text) Clone() INode {

// 	newText := NewText(text.name, text.lineHeight)
// 	newText.textModel = text.textModel.Clone().(*Model)
// 	newText.typewriterIndex = text.typewriterIndex
// 	newText.setText = text.setText
// 	newText.font = text.font
// 	return newText
// }

// func (text *Text) Font() font.Face {
// 	return text.font
// }

// func (text *Text) SetFont(font font.Face) {
// 	if text.font != font {
// 		text.font = font
// 		text.updateTexture()
// 	}
// }

// func (text *Text) Text() string {
// 	return text.setText
// }

// func (text *Text) SetText(txt string) {
// 	if text.setText != txt {
// 		text.setText = txt
// 		text.updateTexture()
// 	}
// }

// func (textObj *Text) updateTexture() {

// 	if textObj.setText == "" {
// 		return
// 	}

// 	measure := text.BoundString(textObj.font, textObj.Text())

// 	asr := float32(measure.Dx()) / float32(measure.Dy())

// 	if textObj.texture == nil || measure.Dx() > textObj.texture.Bounds().Dx() || measure.Dy() > textObj.texture.Bounds().Dy() {

// 		if textObj.texture != nil {
// 			textObj.texture.Dispose()
// 		}

// 		newWidth := int(closestPowerOfTwo(float32(measure.Dx())) * 1.5)
// 		newHeight := int(closestPowerOfTwo(float32(measure.Dy())) * 1.5)

// 		textObj.texture = ebiten.NewImage(newWidth, newHeight)

// 		textObj.textModel.Mesh.Materials()[0].Texture = textObj.texture

// 	}

// 	textObj.texture.Clear()

// 	txt := ""
// 	if textObj.typewriterIndex >= 0 {
// 		txt = textObj.Text()[:textObj.typewriterIndex]
// 	} else {
// 		txt = textObj.Text()
// 	}

// 	lineCount := float32(strings.Count(textObj.Text(), "\n")) + 1

// 	text.Draw(textObj.texture, txt, textObj.font, -measure.Min.X, -measure.Min.Y, color.White)

// 	fmt.Println("text:", txt)

// 	// targetDimensions := textObj.dimensions.Mult(textObj.WorldScale())

// 	// textObj.textModel.SetLocalScaleVec(targetDimensions)

// 	// textObj.textModel.SetLocalScale(asr, 1, 1)

// 	lh := textObj.lineHeight * lineCount
// 	fmt.Println(textObj.lineHeight, lh, lineCount)
// 	textObj.textModel.SetLocalScale(lh*asr, lh, 1)

// 	fmt.Println("text set?")

// 	// Left align
// 	textObj.textModel.SetLocalPosition(textObj.textModel.LocalScale().X/2, 0, 0)

// }

// func (text *Text) TypewriterIndex() int {
// 	return text.typewriterIndex
// }

// // SetTypewriterIndex advances the scroll of the text by the number of characters given.
// // SetTypewriterIndex will return a boolean value indicating if the Text is at the end
// // of the scroll or not.
// func (text *Text) SetTypewriterIndex(typewriterIndex int) bool {

// 	oldIndex := text.typewriterIndex
// 	text.typewriterIndex += typewriterIndex

// 	if oldIndex != text.typewriterIndex {
// 		text.updateTexture()
// 	}

// 	if text.typewriterIndex >= len(text.setText) {
// 		text.typewriterIndex = len(text.setText)
// 	}
// 	return text.typewriterIndex >= len(text.setText)
// }

// // AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// // hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
// func (text *Text) AddChildren(children ...INode) {
// 	text.addChildren(text, children...)
// }

// // Unparent unparents the AmbientLight from its parent, removing it from the scenegraph.
// func (text *Text) Unparent() {
// 	if text.parent != nil {
// 		text.parent.RemoveChildren(text)
// 	}
// }

// // Type returns the NodeType for this object.
// func (text *Text) Type() NodeType {
// 	return NodeTypeText
// }

// // Index returns the index of the Node in its parent's children list.
// // If the node doesn't have a parent, its index will be -1.
// func (text *Text) Index() int {
// 	if text.parent != nil {
// 		for i, c := range text.parent.Children() {
// 			if c == text {
// 				return i
// 			}
// 		}
// 	}
// 	return -1
// }

// func (text *Text) String() string {
// 	if ReadableReferences {
// 		return "<" + text.Path() + "> : " + text.Text()
// 	} else {
// 		return fmt.Sprintf("%p", text)
// 	}
// }
