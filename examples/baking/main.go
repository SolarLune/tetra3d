package main

import (
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"runtime/pprof"
	"time"

	_ "embed"

	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed baking.gltf
var gltfData []byte

type Game struct {
	Width, Height int
	Library       *tetra3d.Library
	Scene         *tetra3d.Scene

	Camera       *tetra3d.Camera
	CameraTilt   float64
	CameraRotate float64

	DrawDebugText     bool
	DrawDebugDepth    bool
	PrevMousePosition Vector

	Time float64
}

func NewGame() *Game {
	game := &Game{
		Width:             796,
		Height:            448,
		DrawDebugText:     true,
		PrevMousePosition: Vector{},
	}

	game.Init()

	return game
}

const (
	ChannelLight    = iota // The light color channel
	ChannelAO              // The AO color channel
	ChannelCombined        // The combined color channel
	ChannelNone     = -1   // No color channel
)

func (g *Game) Init() {

	opt := tetra3d.DefaultGLTFLoadOptions()
	opt.CameraWidth = g.Width
	opt.CameraHeight = g.Height
	library, err := tetra3d.LoadGLTFData(gltfData, opt)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = library.Scenes[0]

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.SetLocalPositionVec(Vector{0, 2, 15})
	g.Scene.Root.AddChildren(g.Camera)

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	// OK, so in this example we're going to bake lighting and AO.
	// We will combine them into different vertex color channels to
	// easily preview the differences, as well.

	// We start by collecting all the lights we'll be baking. NodeFilter.Lights() will automatically give us just the lights out of our selection and discard any other INodes.
	lights := g.Scene.Root.ChildrenRecursive().Lights()

	// // Don't forget the world's ambient light!
	lights = append(lights, g.Scene.World.AmbientLight)

	// Let's get all the solid, occluding models here.
	models := g.Scene.Root.ChildrenRecursive().ByTags("ao").Models()

	// The idea is that we'll bake the lighting and AO of each ao-applicable Model.

	// We'll bake the lighting to vertex channel #0, AO to channel #1, and combine both
	// in vertex channel #2. We will use named consts to make referencing these channels simple.

	for _, model := range models {

		// First, we'll bake the lighting into the lighting vertex color channel.
		model.BakeLighting(ChannelLight, lights...)

		// Next, we'll bake the AO into the AO vertex color channel.

		// The AOBakeOptions struct controls how our AO should be baked. For readability, we create it
		// here, though we should, of course, create it outside our model loop if the options don't differ
		// for each Model.
		bakeOptions := tetra3d.NewDefaultAOBakeOptions()
		bakeOptions.OcclusionAngle = tetra3d.ToRadians(60)
		bakeOptions.TargetChannel = ChannelAO // Set the target baking channel (as it's not 0)
		// bakeOptions.OtherModels = models      // Specify what other objects should influence the AO (note that this is optional)
		model.Mesh.SetVertexColor(ChannelAO, colors.White())
		model.BakeAO(bakeOptions) // And bake the AO.

		// We'll combine the channels together multiplicatively here.
		model.Mesh.CombineVertexColors(ChannelCombined, true, ChannelLight, ChannelAO)

		for _, mat := range model.Mesh.Materials() {
			mat.Shadeless = true // We don't need lighting anymore.
		}

		// Finally, we'll set the models' active color channel here. By default, it's -1, indicating no vertex colors are active (unless
		// the mesh was exported from Blender with an active vertex color channel).
		model.Mesh.SetActiveColorChannel(ChannelCombined)

	}

	// We can turn off the lights now, as we won't need them anymore.
	for _, light := range lights {
		light.SetOn(false)
	}

}

func (g *Game) Update() error {

	// lights := g.Scene.Root.ChildrenRecursive().AsLights()

	// Bake the lighting real quick
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		for _, model := range g.Scene.Root.ChildrenRecursive().Models() {
			model.Mesh.SelectVertices().SelectAll().SetActiveColorChannel(ChannelNone) // Switch to unlit
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		for _, model := range g.Scene.Root.ChildrenRecursive().Models() {
			model.Mesh.SelectVertices().SelectAll().SetActiveColorChannel(ChannelAO) // Switch to AO only
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		for _, model := range g.Scene.Root.ChildrenRecursive().Models() {
			model.Mesh.SelectVertices().SelectAll().SetActiveColorChannel(ChannelLight) // Switch to Lighting
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		for _, model := range g.Scene.Root.ChildrenRecursive().Models() {
			model.Mesh.SelectVertices().SelectAll().SetActiveColorChannel(ChannelCombined) // Switch to lighting+AO
		}
	}

	g.Time += 1.0 / 60.0

	var err error

	moveSpd := 0.05

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Moving the Camera

	// We use Camera.Rotation.Forward().Invert() because the camera looks down -Z (so its forward vector is inverted)
	forward := g.Camera.LocalRotation().Forward().Invert()
	right := g.Camera.LocalRotation().Right()

	pos := g.Camera.LocalPosition()

	// g.Scene.Root.Get("point light").(*tetra3d.PointLight).Distance = 10 + (math.Sin(g.Time*math.Pi) * 5)

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		pos = pos.Add(forward.Scale(moveSpd))
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		pos = pos.Add(right.Scale(moveSpd))
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		pos = pos.Add(forward.Scale(-moveSpd))
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		pos = pos.Add(right.Scale(-moveSpd))
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		pos[1] += moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		pos[1] -= moveSpd
	}

	g.Camera.SetLocalPositionVec(pos)

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// Rotating the camera with the mouse

	// Rotate and tilt the camera according to mouse movements
	mx, my := ebiten.CursorPosition()

	mv := Vector{float64(mx), float64(my)}

	diff := mv.Sub(g.PrevMousePosition)

	g.CameraTilt -= diff[1] * 0.005
	g.CameraRotate -= diff[0] * 0.005

	g.CameraTilt = math.Max(math.Min(g.CameraTilt, math.Pi/2-0.1), -math.Pi/2+0.1)

	tilt := tetra3d.NewMatrix4Rotate(1, 0, 0, g.CameraTilt)
	rotate := tetra3d.NewMatrix4Rotate(0, 1, 0, g.CameraRotate)

	// Order of this is important - tilt * rotate works, rotate * tilt does not, lol
	g.Camera.SetLocalRotation(tilt.Mult(rotate))

	g.PrevMousePosition = mv.Clone()

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		f, err := os.Create("screenshot" + time.Now().Format("2006-01-02 15:04:05") + ".png")
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		png.Encode(f, g.Camera.ColorTexture())
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.Init()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.StartProfiling()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.DrawDebugText = !g.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color - we can use the world lighting color for this.
	screen.Fill(g.Scene.World.ClearColor.ToRGBA64())

	// Clear the Camera
	g.Camera.Clear()

	// Render the scene
	g.Camera.RenderNodes(g.Scene, g.Scene.Root)

	// We rescale the depth or color textures here just in case we render at a different resolution than the window's; this isn't necessary,
	// we could just draw the images straight.
	opt := &ebiten.DrawImageOptions{}
	w, h := g.Camera.ColorTexture().Size()
	opt.GeoM.Scale(float64(g.Width)/float64(w), float64(g.Height)/float64(h))
	if g.DrawDebugDepth {
		screen.DrawImage(g.Camera.DepthTexture(), opt)
	} else {
		screen.DrawImage(g.Camera.ColorTexture(), opt)
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nThis example shows how lighting and primitive\nambient occlusion can be baked into vertex colors.\n1 Key: Switch to unlit channel\n2 Key: Switch to only AO\n3 key: Switch to only lighting\n4 Key: Switch to lighting+AO\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		g.Camera.DebugDrawText(screen, txt, 0, 150, 1, colors.LightGray())
	}
}

func (g *Game) StartProfiling() {
	outFile, err := os.Create("./cpu.pprof")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Beginning CPU profiling...")
	pprof.StartCPUProfile(outFile)
	go func() {
		time.Sleep(2 * time.Second)
		pprof.StopCPUProfile()
		fmt.Println("CPU profiling finished.")
	}()
}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {
	ebiten.SetWindowTitle("Tetra3d - Baked Lighting Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
