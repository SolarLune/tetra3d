package main

import (
	_ "embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

//go:embed text.gltf
var sceneData []byte

//go:embed excel.ttf
var excelTTF []byte

type Game struct {
	Scene *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	Text *tetra3d.Text

	Time int
}

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	// Load the GLTF file and turn it into a Library, which is a collection of scenes and data shared between them (like meshes or animations).

	library, err := tetra3d.LoadGLTFData(sceneData, nil)
	if err != nil {
		panic(err)
	}

	g.Scene = library.FindScene("Scene")

	g.Camera = examples.NewBasicFreeCam(g.Scene)

	// Load the font data
	fontData, err := opentype.Parse(excelTTF)
	if err != nil {
		panic(err)
	}

	targetSize := 7.5

	newFace, err := opentype.NewFace(fontData, &opentype.FaceOptions{
		Size:    float64(targetSize),
		DPI:     96, // Pixel art fonts tend to be made for 96 PPI, apparently?
		Hinting: font.HintingFull,
	})

	if err != nil {
		panic(err)
	}

	font := text.FaceWithLineHeight(newFace, float64(targetSize))

	textPlane := g.Scene.Root.Get("Screen").(*tetra3d.Model)

	// Create a new Text object to handle text display; we specify what Meshpart it should
	// target, and how large to make the text rendering surface
	g.Text = tetra3d.NewText(textPlane.Mesh.FindMeshPart("Text"), 256)

	g.Text.ApplyStyle(
		tetra3d.TextStyle{
			Font:                 font,
			FGColor:              colors.SkyBlue(),
			BGColor:              colors.Blue().SubRGBA(0, 0, 0.8, 0),
			Cursor:               "I",
			HorizontalAlignment:  tetra3d.TextAlignCenter,
			LineHeightMultiplier: 1,
		},
	)

	g.Text.SetText("This screen will slowly display text; you can press the Q key to reset the typewriter effect. Got it? I think you do.\n\n-------\n\nManual newlines also work here.")

	// g.Text.HorizontalAlignment = tetra3d.TextAlignCenter

	g.System = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	g.Camera.Update()

	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		g.Text.SetTypewriterIndex(0)
	}

	if g.Time%4 == 0 {
		g.Text.AdvanceTypewriterIndex(1)
	}

	g.Time++

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(g.Scene.World.FogColor.ToRGBA64())

	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This example shows how Text can be displayed
dynamically in Tetra3D. By using the Text type,
you can dynamically write text and scroll it.
Note that Text objects work best when the displaying faces are not
rotated and face a world axis directly (though the object can
be rotated just fine).`
		g.Camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Text Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
