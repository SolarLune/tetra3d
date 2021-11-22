# Tetra3D

![Tetra3D Logo](https://thumbs.gfycat.com/DifferentZealousFowl-size_restricted.gif)
![Dark exploration](https://thumbs.gfycat.com/ScalySlimCrayfish-size_restricted.gif)

[Tetra3D Docs](https://pkg.go.dev/github.com/solarlune/tetra3d)

## Support

If you want to support development, feel free to check out my [itch.io](https://solarlune.itch.io/masterplan) / [Steam](https://store.steampowered.com/app/1269310/MasterPlan/) / [Patreon](https://www.patreon.com/SolarLune). I also have a [Discord server here](https://discord.gg/cepcpfV). Thanks~!
## What is Tetra3D?

Tetra3D is a 3D software renderer written in Go by means of [Ebiten](https://ebiten.org/), primarily for video games. Compared to a professional 3D rendering systems like OpenGL or Vulkan, it's slow and buggy, but _it's also janky_, and I love it for that.

It evokes a similar feeling to primitive 3D game consoles like the PS1, N64, or DS. Being that a software renderer is not _nearly_ fast enough for big, modern 3D titles, the best you're going to get out of Tetra is drawing some 3D elements for your primarily 2D Ebiten game, or a relatively rudimentary fully 3D game (_maybe_ something on the level of a PS1 or N64 game would be possible). That said, limitation breeds creativity, and I am intrigued at the thought of what people could make with Tetra.

## Why did I make it?

Because there's not really too much of an ability to do 3D for gamedev in Go apart from [g3n](http://g3n.rocks), [go-gl](https://github.com/go-gl/gl) and [Raylib-go](https://github.com/gen2brain/raylib-go). I like Go, I like janky 3D, and so, here we are. 

It's also interesting to have the ability to spontaneously do things in 3D sometimes. For example, if you were making a 2D game with Ebiten but wanted to display just a few things in 3D, Tetra3D should work well for you.

Finally, while a software renderer is not by any means fast, it is relatively simple and easy to use. Any platforms that Ebiten supports should also work for Tetra3D automatically (hopefully!).

## Why Tetra3D? Why is it named that?

Because it's like a [tetrahedron](https://en.wikipedia.org/wiki/Tetrahedron), a relatively primitive (but visually interesting) 3D shape made of 4 triangles. Otherwise, I had other names, but I didn't really like them very much. "Jank3D" was the second-best one, haha.

## How do you get it?

`go get github.com/solarlune/tetra3d`

Tetra depends on kvartborg's [vector](https://github.com/kvartborg/vector) package, takeyourhatoff's [bitset](https://github.com/takeyourhatoff/bitset) package, and [Ebiten](https://ebiten.org/) itself for rendering. Tetra3D requires Go v1.16 or above.

## How do you use it?

Make a camera, load a scene, render it. A simple 3D framework means a simple 3D API.

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
	"github.com/kvartborg/vector"
	"github.com/hajimehoshi/ebiten/v2"
)

const ScreenWidth = 398
const ScreenHeight = 224

type Game struct {
	Scene        *tetra3d.Scene
	Camera       *tetra3d.Camera
}

func NewGame() *Game {

	g := &Game{}

	// First, we load a scene from a .dae file. LoadDAEFile takes a filepath and
	// any loading options (nil is taken as a default), and returns a *Scene 
	// and an error if it was unsuccessful. 
	scene, err := tetra3d.LoadDAEFile("example.dae", nil) 

	if err != nil {
		panic(err)
	}

	g.Scene = scene

	// NOTE: If you need to rotate the model because the 3D modeler you're 
	// using doesn't use the same axes as Tetra (like Blender),
	// you can call Mesh.ApplyMatrix() to apply a rotation matrix 
	// (or any other kind of matrix) to the vertices, thereby rotating 
	// them and their triangles' normals. 

	// With Blender and the default DAE exporter, this conversion is 
	// handled for you; you can change this by passing a different
	// DaeLoadOptions parameter when calling tetra3d.LoadDAEFile().

	// Tetra uses OpenGL's coordinate system (+X = Right, +Y = Up, +Z = Back), 
	// in comparison to Blender's coordinate system (+X = Right, +Y = Forward, 
	// +Z = Up). 

	// Create a new Camera. We pass the size of the screen to the Camera so
	// it can create its own buffer textures (which are *ebiten.Images).

	g.Camera = tetra3d.NewCamera(ScreenWidth, ScreenHeight)

	// A Camera implements the tetra3d.Node interface, which means it can be placed
	// in 3D space and can be parented to another Node-implementing object somewhere
	// in the scene tree. Models and NodeBases (which are essentially "empties" one can
	// use for positioning and parenting) can, as well.

	// Each Scene has a tree that starts with the root Node. To add Nodes to the Scene, 
	// parent them to the Scene's base.
	g.Scene.Root.AddChildren(g.Camera)

	// For Cameras, we don't actually need to have them in the scene to view it, since
	// the presence of the Camera in the Scene node tree doesn't impact what it would see.

	// Place Models or Cameras with SetWorldPosition() or SetLocalPosition(). 
	// Both functions take a 3D vector.Vector from kvartborg's vector package. 
	
	// The *World variants position Nodes in absolute space; the Local variants
	// position Nodes relative to their parents' positioning and transforms.
	
	// Cameras look forward down the -Z axis, so we'll move the Camera back 12 units to look
	// towards [0, 0, 0].
	g.Camera.SetLocalPosition(vector.Vector{0, 0, 12})

	return game
}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {

	// Call Camera.Clear() to clear its internal backing texture. This
	// should be called once per frame before drawing your Scene.
	g.Camera.Clear()

	// Now we'll render the Scene from the camera. The Camera's ColorTexture will then 
	// hold the result. 
	
	// We pass both the Scene and the Models because 1) the Scene influences
	// how Models draw (fog, for example), and 2) we may not want to render
	// all Models. Camera.RenderNodes() renders all Nodes in a tree, starting with the 
	// Node specified. You can also use Camera.Render() to simply render a selection of
	// Models.
	g.Camera.RenderNodes(g.Scene, g.Scene.Root) 

	// Before drawing the result, clear the screen first.
	screen.Fill(color.RGBA{20, 30, 40, 255})

	// Draw the resulting texture to the screen, and you're done! You can 
	// also visualize the depth texture with g.Camera.DepthTexture.
	screen.DrawImage(g.Camera.ColorTexture, nil) 

}

func (g *Game) Layout(w, h int) (int, int) {
	// This is the size of the window; note that a larger (or smaller) 
	// layout doesn't seem to impact performance very much.
	return ScreenWidth, ScreenHeight
}

func main() {

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}


```

That's basically it. Note that Tetra3D is, indeed, a work-in-progress and so will require time to get to a good state. But I feel like it works pretty well and feels pretty solid to work with as is. Feel free to examine the examples folder for a couple of examples showing how Tetra3D works - calling `go run .` from within their directories should work.

## What's missing?

The following is a rough to-do list (tasks with checks have been implemented):

- [x] 3D rendering
- [x] -- Perspective projection
- [ ] -- Orthographic projection
- [ ] -- Billboarding
- [ ] -- Some additional way to draw 2D stuff with no perspective changes (if desired) in 3D space
- [x] -- Basic depth sorting (sorting vertices in a model according to distance, sorting models according to distance)
- [x] -- A depth buffer and [depth testing](https://learnopengl.com/Advanced-OpenGL/Depth-testing) - This is now implemented by means of a depth texture and [Kage shader](https://ebiten.org/documents/shader.html#Shading_language_Kage), though the downside is that it requires rendering and compositing the scene into textures _twice_. Also, it doesn't work on triangles from the same object (as we can't render to the depth texture while reading it for existing depth).
- [ ] -- A more advanced depth buffer - currently, the depth is written using vertex colors.
- [x] -- Offscreen Rendering
- [x] -- Mesh merging - Meshes can be merged together to lessen individual object draw calls.
- [ ] -- Render batching - We can avoid calling Image.DrawTriangles between objects if they share properties (image, mainly).
- [x] Culling
- [x] -- Backface culling
- [x] -- Frustum culling
- [x] -- Far triangle culling
- [ ] -- Triangle clipping to view (this isn't implemented, but not having it doesn't seem to be too much of a problem for now)
- [x] Debug
- [x] -- Debug text: overall render time, FPS, render call count, vertex count, triangle count, skipped triangle count
- [x] -- Wireframe debug rendering
- [x] -- Normal debug rendering
- [x] Basic Single Image Texturing
- [ ] -- Multitexturing?
- [ ] -- Per-triangle images
- [ ] -- Perspective-corrected texturing (currently it's affine, see [Wikipedia](https://en.wikipedia.org/wiki/Texture_mapping#Affine_texture_mapping))
- [ ] Animations
- [ ] -- Armature-based animations
- [ ] -- Object transform-based animations
- [x] Scenes
- [x] -- Fog
- [x] -- A node or scenegraph for parenting and simple visibility culling
- [ ] -- Ambient vertex coloring
- [x] DAE model loading
- [x] -- Vertex colors loading
- [x] -- UV map loading
- [x] -- Normal loading
- [x] -- Transform / full scene loading
- [ ] Lighting?
- [ ] Shaders
- [ ] -- Normal rendering (useful for, say, screen-space shaders)
- [ ] Basic Collisions
- [ ] -- AABB collision testing / sphere collision testing?
- [ ] -- Raycasting
- [ ] 3D Sound (just adjusting panning of sound sources based on 3D location, or something like that)
- [ ] Multithreading (particularly for vertex transformations)
- [ ] [Prefer Discrete GPU](https://github.com/silbinarywolf/preferdiscretegpu) for computers with both discrete and integrated graphics cards

Again, it's incomplete and jank. However, it's also pretty cool!

## Shout-out time~

Huge shout-out to the open-source community (i.e. StackOverflow, [fauxgl](https://github.com/fogleman/fauxgl), [tinyrenderer](https://github.com/ssloy/tinyrenderer), [learnopengl.com](https://learnopengl.com/Getting-started/Coordinate-Systems), etc) at large for sharing the information and code to make this possible; I would definitely have never made this happen otherwise.
