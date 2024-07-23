package main

import (
	"bytes"
	_ "embed"
	"image/color"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
)

// Special notes:

// - This is a simple example to show how you could use Tetra3d to act as a skeleton animation system for 2D sprites as well as 3D ones.
//
// - I set the Units used in the Scene tab in Blender to None, rather than Metric; this way, I can customize the grid divisions in the 3D view's viewport overlay
//   settings (the button at the top of the 3D view that looks like two overlapping circles, over to the right of the snapping options). I set it to 16 to match my pixel density.
//   This makes it easier to position the limbs on the skeleton to move in whole pixel increments when setting up to set the limbs' base positions (though this is not strictly
//   necessary, since you'll always be playing back some animation, so you could just set the default positions in each animation manually).
//
// - Textures are embedded in the GLB file. If they weren't embedded, then the stick figure would just appear as composed of gray squares, as its material would just have a path
//   to its texture. If this were the case, I'd have to manually load and assign the material its texture (an *ebiten.Image).
//
// - The material for the stick figure has no lighting (Shadeless is toggled on), and is set to Alpha Blend mode, so that depth sorting works for the limbs
//   as they are partially transparent. These settings are available in the Material tab.
//
// - The character's material has a slight tweak to properly alpha-blend by connecting the texture's alpha component to the BSDF Alpha component (this is what makes it transparent
//   in the viewport in Blender, but has no other effect on the example).
//
// - I chose to do a simple facial animation (blinking) by making two head sprite planes, assigning them to two different bones, and then scaling the bones with constant
//   animation keys (so they're either normally scaled, or scaled to 0). If I wanted a more complex animation, the best approach might be to make it in something like Aseprite,
//   and then use a library like goaseprite to load and play back the animation or play back the frames manually, and finally animate the target limb's UV values using the goaseprite
//   UV values. The concept sounds complex and requires some setting up, but once it's done, it should be relatively easy to manage and duplicate to other models.

//go:embed skeletal2d.glb
var sceneData []byte

type Game struct {
	Scene         *tetra3d.Scene
	Width, Height int

	System examples.BasicSystemHandler
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
	library, err := tetra3d.LoadGLTFData(bytes.NewReader(sceneData), nil)
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
	anim.PlayByName("Run")

	anim.Update(1.0 / 60.0)

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{20, 30, 40, 255})

	camera := g.Scene.Root.Get("Camera").(*tetra3d.Camera)

	// Clear the camera's back buffer and render the scene. The camera size is set in the Blender file.
	camera.Clear()
	camera.RenderScene(g.Scene)

	// Draw the result to the screen, centered.
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
