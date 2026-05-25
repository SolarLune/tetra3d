package main

import (
	"embed"
	"strings"

	"github.com/solarlune/tetra3d"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/solarlune/tetra3d/colors"
	examples "github.com/solarlune/tetra3d/examples"
)

type Game struct {
	Scene         *tetra3d.Scene
	SystemHandler examples.BasicSystemHandler
	Camera        examples.BasicFreeCam
}

//go:embed assets/*.glb
var assets embed.FS

// To load an asset, normally, we would just call one of the tetra3d.LoadGLTF_____() functions,
// but because we're working with data exported from other blend files, we have to set up dependent
// library resolution. It's pretty simple, actually.
func loadAsset(assetName string) *tetra3d.Library {

	loadOptions := tetra3d.DefaultGLTFLoadOptions()

	// If a dependent library is found, then in the process of loading, Tetra3D will
	// call the given dependent library resolver function to attempt to retrieve that data as well.
	// Here, we simply call loadAsset() again, substituting `.blend` in the filename with `.glb`.
	//
	// In practice, this should store the result somewhere and then return that
	// if possible if dependent libraries are used more than once.
	loadOptions.DependentLibraryResolver = func(blendPath string) *tetra3d.Library {
		path := "assets/" + strings.Split(blendPath, ".blend")[0] + ".glb"
		return loadAsset(path)
	}

	library, err := tetra3d.LoadGLTFFileSystem(assets, assetName, loadOptions)

	if err != nil {
		panic(err)
	}

	return library
}

func NewGame() *Game {

	game := &Game{}
	game.Init()
	return game

}

func (g *Game) Init() {

	library := loadAsset("assets/collections.glb")

	g.Scene = library.ExportedScene.Clone()
	g.Camera = examples.NewBasicFreeCam(g.Scene)
	g.SystemHandler = examples.NewBasicSystemHandler(g)

}

func (g *Game) Update() error {

	// Let the camera move around, handle locking and unlocking, etc.
	g.Camera.Update()

	// Let the system handler handle quitting, fullscreening, restarting, etc.
	return g.SystemHandler.Update()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear with a color:
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera:
	g.Camera.Clear()

	// Render the logos first:
	g.Camera.RenderScene(g.Scene)

	// And then just draw the color texture output:
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// Finally, do any debug rendering that might be necessary. Because BasicFreeCam embeds *tetra3d.Camera, we pass
	// its embedded Camera instance.
	g.SystemHandler.Draw(screen, g.Camera.Camera)

	if g.SystemHandler.DrawDebugText {
		txt := `This demo shows how external linking works.
The cone is linked from another blend file (cone.blend) to the
main one (collections.blend). They both are exported to GLTF files,
and when the main scene is loaded in Tetra3D, the dependent library is
also loaded.`
		tetra3d.DrawDebugText(screen, txt, 0, 230, 1, colors.LightGray())
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Collections Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
