package main

import (
	_ "embed"
	"image/color"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

// Special notes:

// - This was, naturally, made using the Tetra3D addon; if you want to inspect how it's made, please ensure you've installed the add-on before continuing.
// - I set the Units used in the Scene tab in Blender to None, rather than Metric; this way, I can customize the grid divisions in the 3D view's viewport overlay
//   settings (the button at the top of the 3D view that looks like two overlapping circles, over to the right of the snapping options). I set it to 16 to match my pixel density.
//   This makes it easier to position the limbs on the skeleton to move in whole pixel increments when setting up.
// - Textures are embedded in the GLB file. If they weren't embedded, then the stick figure would just appear gray, as its material would just have a path to its texture.
//   If this were the case, I'd have to loop through all materials loaded in the library and manually load and assign the materials their textures.
// - The material for the stick figure has no lighting (Shadeless is toggled on), and is set to Alpha Blend mode, so that depth sorting works for the limbs
//   as they are partially transparent. These settings are available in the Material tab.
// - I chose to do a simple facial animation (blinking) by making two head sprites, assigning them to two different bones, and then scaling the bones with constant
//   animation keys (so they're either normally scaled, or scaled to 0). If I wanted a more complex animation, the best approach might be to make it in something like Aseprite,
//   and then use a library like goaseprite to load and play back the animation or play back the frames manually, and finally animate the target limb's UV values using the goaseprite
//   UV values. The concept sounds complex and requires some setting up, but once it's done, it should be relatively easy to manage and duplicate to other models.

//go:embed skeletal2d.glb
var sceneData []byte

type Game struct {
	Scene         *tetra3d.Scene
	Width, Height int

	System examples.BasicSystemHandler // This is just for the example, of course
}

func NewGame() *Game {
	game := &Game{
		Width:  640,
		Height: 360,
	}

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

	g.System = examples.NewBasicSystemHandler(g)
	g.System.DrawDebugText = false
	g.System.UsingBasicFreeCam = false

}

func (g *Game) Update() error {

	armature := g.Scene.Root.Get("Armature")
	anim := armature.AnimationPlayer()
	anim.Play("Run")

	anim.Update(1.0 / 60.0)

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{20, 30, 40, 255})

	camera := g.Scene.Root.Get("Camera").(*tetra3d.Camera)

	camera.Clear()
	camera.RenderScene(g.Scene)

	opt := &ebiten.DrawImageOptions{}
	w, h := camera.Size()
	opt.GeoM.Translate(float64(g.Width/2)-float64(w/2), float64(g.Height/2)-float64(h/2))
	screen.DrawImage(camera.ColorTexture(), opt)

	g.System.Draw(screen, camera)

	if g.System.DrawDebugText {
		txt := `This example just shows how you could use
Tetra3D for 2D skeletal animations as well as 3D.`
		camera.DebugDrawText(screen, txt, 0, 200, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - 2D Skeletal Animation Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
