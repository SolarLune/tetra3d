package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"runtime/pprof"
	"time"

	_ "embed"

	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

//go:embed testimage.png
var testImage []byte

//go:embed character.png
var character []byte

type Game struct {
	Width, Height int
	Scene         *tetra3d.Scene

	Camera            *tetra3d.Camera
	CameraTilt        float64
	CameraRotate      float64
	PrevMousePosition vector.Vector

	DrawDebugText      bool
	DrawDebugDepth     bool
	DrawDebugWireframe bool
	DrawDebugNormals   bool

	Cubes []*tetra3d.Model

	Time float64
}

func NewGame() *Game {

	game := &Game{
		Width:             796,
		Height:            448,
		PrevMousePosition: vector.Vector{},
	}

	game.Init()

	return game
}

func (g *Game) Init() {

	// OK, so in this stress test, we're rendering a lot of individual, low-poly objects (cubes).

	// To make this run faster, we can batch objects together dynamically - unlike static batching,
	// this will allow them to move and act individually while also allowing us to render them more
	// efficiently as they will render in a single batch.
	// However, dynamic batching has some limitations.

	// 1) Dynamically batched Models can't have individual material / object properties (like object
	// color, texture, material blend mode, or texture filtering mode). Vertex colors still
	// work for fading individual triangles / a dynamically batched Model, though.

	// 2) Dynamically batched objects all count up to a single vertex count, so we can't have
	// more than the max number of vertices after batching all objects together into a single
	// Model (which at this stage is 65535 vertices, or 21845 triangles).

	// 3) Dynamically batched objects render using the batching object's first meshpart's material
	// for rendering. This is to make it predictable as to how all batched objects appear at all times.

	// 4) Because we're batching together dynamically batched objects, they can't write to the depth
	// texture individually. This means dynamically batched objects cannot visually intersect with one
	// another - they simply draw behind or in front of one another, and so are sorted according to their
	// objects' depths in comparison to the camera.

	// While there are some restrictions, the advantages are weighty enough to make it worth it - this
	// functionality is very useful for things like drawing a lot of small, simple models, like NPCs,
	// particles, or map icons.

	// Create the scene (we're doing this ourselves in this case).
	g.Scene = tetra3d.NewScene("Test Scene")

	// Load the test cube texture.
	img, _, err := image.Decode(bytes.NewReader(testImage))
	if err != nil {
		panic(err)
	}

	// We can reuse the same mesh for all of the models.
	cubeMesh := tetra3d.NewCube()

	// Set up how the cube should appear.
	mat := cubeMesh.MeshParts[0].Material
	mat.Shadeless = true
	mat.Texture = ebiten.NewImageFromImage(img)

	// The batched model will hold all of our batching results. The batched objects keep their mesh-level
	// details (vertex positions, UV, normals, etc), but will mimic the batching object's material-level
	// and object-level properties (texture, material / object color, texture filtering, etc).

	// When a model is dynamically batching other models, the batching model doesn't render.

	// In this example, you will see it as a plane, floating above all the other Cubes before batching.

	planeMesh := tetra3d.NewPlane()

	// Load the character texture.
	img, _, err = image.Decode(bytes.NewReader(character))
	if err != nil {
		panic(err)
	}

	mat = planeMesh.MeshParts[0].Material
	mat.Shadeless = true
	mat.Texture = ebiten.NewImageFromImage(img)

	batched := tetra3d.NewModel(planeMesh, "DynamicBatching")
	batched.Move(0, 4, 0)
	batched.Rotate(1, 0, 0, tetra3d.ToRadians(90))
	batched.Rotate(0, 1, 0, tetra3d.ToRadians(180))

	g.Cubes = []*tetra3d.Model{}

	for i := 0; i < 21; i++ {
		for j := 0; j < 21; j++ {
			// Create a new Cube, position it, add it to the scene, and add it to the cubes slice.
			cube := tetra3d.NewModel(cubeMesh, "Cube")
			cube.SetLocalPosition(vector.Vector{float64(i) * 1.5, 0, float64(-j * 3)})
			g.Scene.Root.AddChildren(cube)
			g.Cubes = append(g.Cubes, cube)
		}
	}

	g.Scene.Root.AddChildren(batched)

	g.Camera = tetra3d.NewCamera(g.Width, g.Height)
	g.Camera.Far = 120
	g.Camera.SetLocalPosition(vector.Vector{0, 0, 15})

	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

}

func (g *Game) Update() error {

	var err error

	moveSpd := 0.1

	tps := 1.0 / 60.0 // 60 ticks per second, regardless of display FPS

	g.Time += tps

	for cubeIndex, cube := range g.Cubes {
		wp := cube.WorldPosition()

		wp[1] = math.Sin(g.Time*math.Pi + float64(cubeIndex))

		cube.SetWorldPosition(wp)
		cube.Color.R = float32(cubeIndex) / 100
		cube.Color.G = float32(cubeIndex) / 10
	}

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		dyn := g.Scene.Root.Get("DynamicBatching").(*tetra3d.Model)

		if len(dyn.DynamicBatchModels) == 0 {
			dyn.DynamicBatchAdd(dyn.Mesh.MeshParts[0], g.Cubes...) // Note that Model.DynamicBatchAdd() can return an error if batching the specified objects would push it over the vertex limit.
		} else {
			dyn.DynamicBatchRemove(g.Cubes...)
		}

	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Moving the Camera

	// We use Camera.Rotation.Forward().Invert() because the camera looks down -Z (so its forward vector is inverted)
	forward := g.Camera.LocalRotation().Forward().Invert()
	right := g.Camera.LocalRotation().Right()

	pos := g.Camera.LocalPosition()

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

	g.Camera.SetLocalPosition(pos)

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// Rotating the camera with the mouse

	// Rotate and tilt the camera according to mouse movements
	mx, my := ebiten.CursorPosition()

	mv := vector.Vector{float64(mx), float64(my)}

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

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.Init()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.StartProfiling()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.DrawDebugText = !g.DrawDebugText
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.DrawDebugWireframe = !g.DrawDebugWireframe
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.DrawDebugNormals = !g.DrawDebugNormals
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.DrawDebugDepth = !g.DrawDebugDepth
	}

	return err

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
		time.Sleep(10 * time.Second)
		pprof.StopCPUProfile()
		fmt.Println("CPU profiling finished.")
	}()

}

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear, but with a color
	screen.Fill(color.RGBA{60, 70, 80, 255})

	// Clear the Camera
	g.Camera.Clear()

	// Render the non-screen Models
	// g.Camera.Render(g.Scene, g.Scene.Models...)

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

	if g.DrawDebugWireframe {
		g.Camera.DrawDebugWireframe(screen, g.Scene.Root, colors.Red())
	}

	if g.DrawDebugNormals {
		g.Camera.DrawDebugNormals(screen, g.Scene.Root, 0.25, colors.Blue())
	}

	if g.DrawDebugText {
		g.Camera.DrawDebugRenderInfo(screen, 1, colors.White())
		txt := "F1 to toggle this text\nWASD: Move, Mouse: Look\nStress Test 2 - Here, cubes are moving. We can render them\nefficiently by dynamically batching them, though they\nwill mimic the batching object (the character plane) visually - \nthey no longer have their own texture,\nblend mode, or texture filtering (as they all\ntake these properties from the floating character plane).\nThey also can no longer intersect; rather, they\nwill just draw in front of or behind each other.\n1 Key: Toggle batching cubes together\nF5: Toggle depth debug view\nF4: Toggle fullscreen\nESC: Quit"
		text.Draw(screen, txt, basicfont.Face7x13, 0, 140, color.RGBA{200, 200, 200, 255})
	}

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Width, g.Height
}

func main() {

	ebiten.SetWindowTitle("Tetra3d Test - Stress Test 2")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}
