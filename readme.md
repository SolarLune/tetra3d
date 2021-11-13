# Tetra3D

[Tetra3D Docs](HERE)

## What is Tetra3D?

Tetra3D is a 3D software renderer written in Go with [Ebiten](https://ebiten.org/), primarily for video games. It's relatively slow and buggy, but _it's also janky_, and I love it for that.

It evokes a similar feeling to primitive 3D game consoles like the PS1, N64, or DS. Being that a software renderer is not _nearly_ fast enough for big, modern 3D titles, the best you're going to get out of Tetra is drawing some 3D elements for your primarily 2D Ebiten game, or a relatively rudimentary fully 3D game (_maybe_ something on the level of a PS1 or N64 game would be possible). That said, limitation breeds creativity, and I am intrigued at the thought of what people could make with Tetra.

## Why did you make it?

Because there's not really too much of an ability to do 3D for gamedev in Go apart from [g3n](http://g3n.rocks), [go-gl](https://github.com/go-gl/gl) and [Raylib-go](https://github.com/gen2brain/raylib-go). I like Go, I like janky 3D, and so, here we are. 

It's also interesting to have the ability to spontaneously do things in 3D sometimes. For example, if you were making a 2D game with Ebiten but wanted to display something in 3D, Tetra3D should work well for you.

Finally, while a software renderer is not by any means fast, it doesn't require anything more than Ebiten requires. So, any platforms that Ebiten supports should also work for Tetra3D automatically (hopefully!).

## Why Tetra3D? Why is it named that?

Because it's like a tetrahedron, a relatively primitive (but visually interesting) 3D shape made of 4 triangles. 

Otherwise, I had other names, but I didn't really like them very much. "Jank3D" was one, haha.

## What's missing?

The following is a rough to-do list (tasks with checks have been implemented):

- [x] 3D rendering
- [x] -- Perspective projection
- [ ] -- Orthographic projection
- [ ] -- Billboards
- [ ] -- Some additional way to draw 2D stuff with no perspective changes (if desired) in 3D space
- [x] Culling
- [x] -- Backface culling
- [x] -- Frustum culling
- [ ] -- Near / Far culling
- [ ] -- Triangle clipping to view (this isn't implemented, but not having it doesn't seem to be too much of a problem for now)
- [x] Basic depth sorting (sorting vertices in a model according to distance, sorting models according to distance)
- [x] -- A depth buffer and [depth testing](https://learnopengl.com/Advanced-OpenGL/Depth-testing) - This is now implemented by means of a depth texture and [Kage shader](https://ebiten.org/documents/shader.html#Shading_language_Kage), though the downside is that it requires rendering and compositing the scene into textures _twice_. Also, it doesn't work on triangles from the same object (as we can't render to the depth texture while reading it for existing depth).
- [x] Debug
- [x] -- Debug text: overall render time, FPS, render call count, vertex count, triangle count, skipped triangle count
- [x] -- Wireframe debug rendering
- [x] -- Normal debug rendering
- [x] Basic Single Image Texturing
- [ ] -- Multitexturing?
- [ ] -- Perspective-corrected texturing (currently it's affine, see [Wikipedia](https://en.wikipedia.org/wiki/Texture_mapping#Affine_texture_mapping))
- [x] DAE model loading
- [x] -- Vertex colors loading
- [x] -- UV map loading
- [x] -- Normal loading
- [ ] -- Transform / full scene loading
- [ ] -- Bones / Armatures / Animations
- [ ] A scenegraph for parenting / relative object positioning (not sure if I'll implement this, but I could definitely see the utility)
- [ ] Scenes
- [x] -- Fog
- [ ] -- Ambient vertex coloring
- [ ] Lighting?
- [ ] Shaders
- [ ] -- Normal rendering (useful for, say, screen-space reflection shaders)
- [ ] Basic Collisions
- [ ] -- AABB collision testing / sphere collision testing?
- [ ] -- Raycasting
- [ ] Multithreading (particularly for vertex transformations)
- [ ] Model Batching (Ebiten should batch render calls together automatically, at least partially, as long as we adhere to the [efficiency guidelines](https://ebiten.org/documents/performancetips.html#Make_similar_draw_function_calls_successive))
- [ ] [Prefer Discrete GPU](https://github.com/silbinarywolf/preferdiscretegpu) for computers with both discrete and integrated graphics cards

Again, it's incomplete and jank. However, it's also pretty cool!

## How do I use it?

Make a camera, load a mesh, place them somewhere, render your thing. A simple renderer gets a simple API.

Here's an example:

```go

package main

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"os"
	"runtime/pprof"
	"time"

	_ "image/png"

	"github.com/solarlune/tetra3d"

	"github.com/hajimehoshi/ebiten/v2"
)

const ScreenWidth = 320
const ScreenHeight = 180

type Game struct {
	Scene        *tetra3d.Scene
	Camera        *tetra3d.Camera
}

func NewGame() *Game {

	g := &Game{
		Models: []*tetra3d.Model{},
	}

	// Load a scene from a .dae file - returns a *Scene and error, if the call was unsuccessful. The scene will contain all 
	// objects exported along with their meshes, handily converted to *tetra3d.Model and *tetra3d.Mesh instances.
	scene, err := tetra3d.LoadDAEFile("examples.dae") 

	// Something went wrong with the loading process.
	if err != nil {
		panic(err)
	}

	g.Scene = scene

	// If you need to rotate the model because the 3D modeler you're using doesn't use the same axes as Tetra (like Blender),
	// you can call Mesh.ApplyMatrix() to apply a rotation matrix (or any other kind) to the vertices, thereby rotating 
	// them and their triangles' normals. Tetra uses OpenGL's coordinate system (+X = Right, +Y = Up, +Z = Back), in
	// comparison to Blender's coordinate system (+X = Right, +Y = Forward, +Z = Up). 

	// By default (if you're using Blender), this conversion is done for you; you can change this by passing a different
	// DaeLoadOptions parameter when calling tetra3d.LoadDAEFile().

	// Create a new Camera. We pass the size of the screen so it can
	// create its own backing texture. Internally, this is an *ebiten.Image.
	g.Camera = tetra3d.NewCamera(360, 180)

	// Place it using the Position property (which is a 3D Vector).
	// Cameras look forward down the -Z axis.
	g.Camera.Position[2] = 12

	return game
}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {

	// Call Camera.Clear() to clear its internal backing texture. This should be called once before drawing your set of objects.
	g.Camera.Clear()

	// Render your scene from the camera. The Camera's ColorTexture will then 
	// hold the result. If you call Render multiple times, the models will draw on top
	// of each other; pass them all in at the same time to get actual depth.
	g.Camera.Render(g.Scene) 

	// Before drawing the result, clear the screen first.
	screen.Fill(color.RGBA{20, 30, 40, 255})

	// Draw the resulting texture to the screen, and you're done! You can also visualize
	// the depth texture with g.Camera.DepthTexture.
	screen.DrawImage(g.Camera.ColorTexture, nil) 

}

func (g *Game) Layout(w, h int) (int, int) {
	// This is the size of the window; note that a larger (or smaller) 
	// layout doesn't seem to impact performance very much.
	return 360, 180
}

func main() {

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}


```

## Shout-out time~

Huge shout-out to the open-source community (i.e. StackOverflow, [fauxgl](https://github.com/fogleman/fauxgl), [tinyrenderer](https://github.com/ssloy/tinyrenderer), [learnopengl.com](https://learnopengl.com/Getting-started/Coordinate-Systems), etc) at large for sharing the information and code to make this possible; I would definitely have never made this happen otherwise.

Tetra depends on kvartborg's [Vector](https://github.com/kvartborg/vector) package, takeyourhatoff's [bitset](https://github.com/takeyourhatoff/bitset) package, and [Ebiten](https://ebiten.org/) itself for rendering. It also requires Go v1.16 or above.

## Support

If you want to support development, feel free to check out my [itch.io](https://solarlune.itch.io/masterplan) / [Steam](https://store.steampowered.com/app/1269310/MasterPlan/) / [Patreon](https://www.patreon.com/SolarLune), haha. Thanks~!