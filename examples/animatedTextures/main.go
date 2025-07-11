package main

import (
	"embed"

	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"github.com/solarlune/tetra3d/examples"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene

	Camera examples.BasicFreeCam
	System examples.BasicSystemHandler

	AnimatedTexture *tetra3d.TexturePlayer
}

//go:embed assets
var assets embed.FS

func NewGame() *Game {
	game := &Game{}

	game.Init()

	return game
}

func (g *Game) Init() {

	library, err := tetra3d.LoadGLTFFileSystem(assets, "assets/animatedTextures.glb", nil)
	if err != nil {
		panic(err)
	}

	g.Library = library

	// We clone the scene so we have an original to work from
	g.Scene = library.ExportedScene.Clone()

	// Turn off lighting
	g.Scene.World.LightingOn = false

	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.Camera.Move(0, 5, 0)
	g.System = examples.NewBasicSystemHandler(g)

	// Firstly, we create a TexturePlayer, which animates a collection of vertices' UV values to
	// animate a texture on them.

	mesh := library.MeshByName("Plane")

	// We can select all vertices...
	selection := tetra3d.NewVertexSelection().SelectMeshes(mesh)

	// ...And then create a TexturePlayer, which steps through all vertices contained in the passed vertex selection and assigns their UV values according
	// to the TexturePlayer's playing animation.
	g.AnimatedTexture = tetra3d.NewTexturePlayer(selection)

	// Next we create the animation using NewTextureAnimationPixels():
	bloopAnim := tetra3d.NewTextureAnimationPixels(15, mesh.MeshParts[0].Material.Texture,
		0, 0, // UV offset for frame 0,
		16, 0, // Frame 1,
		0, 16, // Frame 2,
		16, 16, // And frame 3.
	)

	// Note that we want to pass 2 values (x and y position) for each frame. Otherwise, NewTextureAnimationPixels will panic.

	// And finally, begin playing the animation. That's it!
	g.AnimatedTexture.Play(bloopAnim)

}

func (g *Game) Update() error {

	// Update the TexturePlayer with the time that's passed since the previous frame.
	g.AnimatedTexture.Update(1.0 / 60.0)

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.AnimatedTexture.Playing = !g.AnimatedTexture.Playing
	}

	g.Camera.Update()

	return g.System.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear, but with a color
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the logo first
	g.Camera.RenderScene(g.Scene)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	g.System.Draw(screen, g.Camera.Camera)

	if g.System.DrawDebugText {
		txt := `This demo shows how animated textures and billboarding work.
There are several lava planes, but they all share
the same mesh, which is animated by the
TexturePlayer.

The character faces the camera because his
material has its BillboardMode set (so that
it faces the camera, but doesn't tilt horizontally).
1 key: Toggle playback`
		g.Camera.DrawDebugText(screen, txt, 0, 220, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Animated Textures Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
