package tetra3d

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

const (
	TextAlignLeft   = iota // Left aligned text. The text hugs the left side of the texture.
	TextAlignCenter        // Center aligned text. All text lines are centered horizontally in the texture's width.
	TextAlignRight         // Right aligned text. The text hugs the right side of the texture.
)

type TextStyle struct {
	Font                 font.Face // The font to use for rendering the text.
	BGColor              *Color    // The Background color for the text. Defaults to black.
	FGColor              *Color    // The foreground color for the text. Defaults to white.
	Cursor               string    // A cursor string sequence is drawn at the end while typewriter-ing
	LineHeightMultiplier float64   // The multiplier for line height changes
	HorizontalAlignment  int       // How the text should be horizontally aligned in the Text texture
}

func NewDefaultTextStyle() TextStyle {
	return TextStyle{
		Font:                 basicfont.Face7x13,
		LineHeightMultiplier: 1,
		BGColor:              NewColor(0, 0, 0, 1),
		FGColor:              NewColor(1, 1, 1, 1),
	}
}

// Text represents a helper object that writes text for display as a texture on a Model's MeshPart.
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

// NewText creates a new Text rendering surface for typing out text and assigns the MeshPart provided
// to use that surface as a texture, on a brand new material.
// textureWidth is how wide (in pixels) the backing texture should be for displaying the text; the height is determined by the
// aspect ratio of the dimensions of the meshpart given. Note that for this to work ideally, the mesh cannot be rotated (i.e. the
// mesh's faces should not be at an angle).
// All changes to the Text require that the Text object updates its texture, which can be costly as this means redrawing the text as
// necessary; this is handled automatically.
func NewText(meshPart *MeshPart, textureWidth int) *Text {

	text := &Text{
		meshPart:        meshPart,
		textureSize:     textureWidth,
		typewriterIndex: 0,
		style:           NewDefaultTextStyle(),
	}

	// Calculate the width and height of the dimensions based off of the
	// meshpart's vertex positions; this determines our texture's aspect ratio.
	w, h := meshPart.primaryDimensions()

	asr := float64(h) / float64(w)

	text.Texture = ebiten.NewImage(textureWidth, int(float64(textureWidth)*asr))

	meshPart.Material = NewMaterial("text material")
	meshPart.Material.TransparencyMode = TransparencyModeTransparent
	meshPart.Material.Texture = text.Texture
	meshPart.Material.BackfaceCulling = true
	meshPart.Material.Shadeless = true

	return text
}

// NewTextAutoSize creates a new Text rendering surface for typing out text,
// with the backing texture's size calculated from an orthographic Camera's render scale.
// This is generally useful for UI Text objects.
// All changes to the Text require that the Text object updates its texture, which can be costly as this means redrawing the text as
// necessary; this is handled automatically.
func NewTextAutoSize(meshPart *MeshPart, camera *Camera) *Text {
	w, _ := camera.Size()

	meshPartDimWidth, _ := meshPart.primaryDimensions()

	texWidth := meshPartDimWidth / camera.OrthoScale() * float64(w)
	return NewText(meshPart, int(texWidth))
}

// Clone clones the Text object.
func (text *Text) Clone() *Text {

	newText := NewText(text.meshPart, text.textureSize)
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
// Setting the text to be blank effectively clears the text, though
// Text.ClearText() also exists, and is just syntactic sugar for this purpose.
func (textObj *Text) SetText(txt string) *Text {

	if textObj.setText != txt {

		textObj.setText = txt

		textureWidth := textObj.Texture.Bounds().Dx()

		// If a word gets too close to the texture's right side, we loop
		safetyMargin := int(float64(textureWidth) * 0.1)

		parsedText := []string{}

		for _, line := range strings.Split(txt, "\n") {

			split := splitWithSeparator(line, " -")

			runningMeasure := 0
			wordIndex := 0

			// Some fonts have space characters that are basically empty somehow...?
			spaceAdd := 0
			if text.BoundString(textObj.style.Font, " ").Dx() <= 0 {
				spaceAdd = text.BoundString(textObj.style.Font, "M").Dx()
			}

			for i, word := range split {
				ws := text.BoundString(textObj.style.Font, word)
				// wordSpace := text.BoundString(textObj.font, word+".").Dx()
				wordSpace := ws.Dx()

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

	textLineMargin := 2
	lineHeight := int(float64(textObj.style.Font.Metrics().Height.Ceil()+textLineMargin) * textObj.style.LineHeightMultiplier)
	dip := textObj.style.Font.Metrics().Ascent.Ceil()

	typing := true

	if textObj.style.BGColor != nil {
		textObj.Texture.Fill(textObj.style.BGColor.ToRGBA64())
	} else {
		textObj.Texture.Clear()
	}

	textureWidth := textObj.Texture.Bounds().Dx()

	for lineIndex, line := range textObj.parsedText {

		measure := text.BoundString(textObj.style.Font, line)

		if textObj.typewriterOn && typewriterIndex >= 0 {

			if !typing {
				break
			}

			if typewriterIndex > len(line) {
				typewriterIndex -= len(line)
			} else if typing {
				line = line[:typewriterIndex] + textObj.style.Cursor
				typing = false
			}

		}

		x := -measure.Min.X

		d := -measure.Min.Y
		if dip > d {
			d = dip
		}

		y := d + (lineIndex * lineHeight)

		if textObj.style.HorizontalAlignment == TextAlignCenter {
			x = textureWidth/2 - measure.Dx()/2
		} else if textObj.style.HorizontalAlignment == TextAlignRight {
			x = textureWidth - measure.Dx()
		}

		text.Draw(textObj.Texture, line, textObj.style.Font, x, y, textObj.style.FGColor.ToRGBA64())

	}

}

func (text *Text) CurrentStyle() TextStyle {
	return text.style
}

func (text *Text) ApplyStyle(style TextStyle) {
	if text.style != style {
		text.style = style
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
	oldTypewriterOn := text.typewriterOn

	text.typewriterIndex = typewriterIndex
	text.typewriterOn = true

	if text.typewriterIndex >= len(text.setText) {
		text.typewriterIndex = len(text.setText)
	}
	if text.typewriterIndex < 0 {
		text.typewriterIndex = 0
	}

	if oldTypewriterOn != text.typewriterOn || oldIndex != text.typewriterIndex {
		text.UpdateTexture()
	}
}

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
// 	lineHeight      float64
// }

// func NewText(name string, lineHeight float64) *Text {
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

// 	asr := float64(measure.Dx()) / float64(measure.Dy())

// 	if textObj.texture == nil || measure.Dx() > textObj.texture.Bounds().Dx() || measure.Dy() > textObj.texture.Bounds().Dy() {

// 		if textObj.texture != nil {
// 			textObj.texture.Dispose()
// 		}

// 		newWidth := int(closestPowerOfTwo(float64(measure.Dx())) * 1.5)
// 		newHeight := int(closestPowerOfTwo(float64(measure.Dy())) * 1.5)

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

// 	lineCount := float64(strings.Count(textObj.Text(), "\n")) + 1

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
