# Jank

## What is Jank?

Jank is a 3D software renderer written in Go with [Ebiten](https://ebiten.org/), primarily for video games. It's relatively slow and buggy, but it's also janky, and I love it for that.

It evokes a similar feeling to primitive 3D game consoles like the PS1, N64, or DS. Being that a software renderer is not _nearly_ fast enough for big, modern 3D titles, the best you're going to get out of Jank is drawing some 3D elements for your primarily 2D game, or a relatively rudimentary fully 3D game (_maybe_ something on the level of a PS1 or N64 game would be possible). That said, limitation breeds creativity, and I am intrigued at the thought of what people could make with Jank.

## Why did you make it?

Because there's not really too much of an ability to do 3D for gamedev in Go apart from [g3n](http://g3n.rocks) or [go-gl](https://github.com/go-gl/gl). I like Go, and so, here we are. 

It's also interesting to have the ability to spontaneously do things in 3D sometimes. For example, if you were making a 2D game in Ebiten but wanted to display something in 3D, Jank should work well for you.

Finally, while a software renderer is not by any means fast, it doesn't require any special drivers or OpenGL support, so any platforms that Ebiten supports should also work for Jank automatically.

## Why Jank? Why is it named that?

Because a homemade 3D software renderer's kinda janky. For those who are unfamiliar with the slang, Jank is pronounced the same as "rank" or "bank" and is the noun form of "janky", which basically means "rough".

## What's missing?

The following is a rough to-do list (tasks with checks have been implented):

- [x] 3D rendering
- [x] -- Perspective projection
- [ ] -- Orthographic projection
- [x] Culling
- [ ] -- Better backface culling (the current existing implementation is very inaccurate)
- [ ] -- Near / Far culling
- [ ] -- Better frustum culling (currently objects are culled from just their center)
- [ ] -- Triangle clipping to the window (without clipping triangles, there can be massive slowdown and memory usage if a triangle draws at a large scale beyond the screen)
- [x] Texturing
- [ ] -- Multitexturing?
- [x] DAE model loading
- [x] -- Vertex colors loading
- [x] -- UV map loading
- [ ] -- Normal loading
- [ ] -- Transform loading (for loading what are essentially full scenes)
- [ ] -- Bones / Armatures / Animations
- [ ] Billboarding
- [ ] -- Some way to draw 2D stuff with no perspective changes (if desired) in 3D space
- [ ] A scenegraph for parenting / relative object positioning (not sure if I'll implement this, but I could definitely see the utility)
- [ ] Lighting?
- [ ] Shaders
- [x] Basic depth sorting
- [ ] -- More advanced depth sorting (of each triangle of all models rendered at a given time; this would be more accurate, while probably being far more inefficient)
- [ ] A depth buffer and [depth testing](https://learnopengl.com/Advanced-OpenGL/Depth-testing) (currently triangles are just rendered in order of their distance to the camera, and objects are similarly sorted)
- [ ] Raycasting
- [ ] Multithreading (particularly on vertex transformations)
- [ ] Render batching (Ebiten should do this partially already as long as we adhere to the [efficiency guidelines](https://ebiten.org/documents/performancetips.html#Make_similar_draw_function_calls_successive))
- [ ] AABB collision testing / sphere collision testing?
- [ ] [Prefer Discrete GPU](https://github.com/silbinarywolf/preferdiscretegpu) for computers with both discrete and integrated graphics cards

Again, it's incomplete and jank. However, it's also pretty cool!

## How do I use it?

Make a camera, load a mesh, place them somewhere, render it. A simple renderer gets a simple API.

Here's an example that uses Ebiten:

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

	"github.com/solarlune/jank"

	"github.com/hajimehoshi/ebiten/v2"
)

const ScreenWidth = 320
const ScreenHeight = 180

type Game struct {
	Models        []*jank.Model
	Camera        *jank.Camera
}

func NewGame() *Game {

	game := &Game{
		Models: []*jank.Model{},
	}

	// Load meshes from a .dae file. The file can contain multiple meshes, so the LoadMeshes functions return maps of mesh name to *jank.Mesh.
	meshes, _ := jank.LoadMeshesFromDAEFile("examples.dae") 

	// Get the mesh by its name in the DAE file, and make a Model (an individual instance) of it.
	sphere := jank.NewModel(meshes["Sphere"]) 

	// Add the model to the models slice.
	g.Models = append(g.Models, sphere)

	// Create a new Camera. We pass the size of the screen so it can
	// create its own backing texture. Internally, this is an *ebiten.Image.
	g.Camera = jank.NewCamera(360, 180)

	// Place it using the Position property (which is a 3D Vector).
	// Jank uses OpenGL's coordinate system. (+X = Right, +Y = Up, +Z = Back)
	// Cameras look forward down the -Z axis.
	g.Camera.Position[2] = 12

	return game
}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {

	// Clear the screen first.
	screen.Fill(color.RGBA{20, 30, 40, 255})

	// Call Camera.Begin() to clear its internal backing texture...
	g.Camera.Begin()

	// Render your models list from the camera. The Camera's ColorTexture will then 
	// hold the result. If you call Render multiple times, the models will draw on top
	// of each other; pass them all in at the same time to get some actual depth.
	g.Camera.Render(g.Models...) 

	// Draw the resulting texture to the screen, and you're done!
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

Jank depends on kvartborg's [Vector](https://github.com/kvartborg/vector) package and [Ebiten](https://ebiten.org/) itself for rendering.