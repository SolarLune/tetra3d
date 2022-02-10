# Tetra3D

[Ebiten Discord](https://discord.gg/fXM7VYASTu)

![Tetra3D Logo](https://thumbs.gfycat.com/DifferentZealousFowl-size_restricted.gif)
![Dark exploration](https://thumbs.gfycat.com/ScalySlimCrayfish-size_restricted.gif)
![Transparency](https://i.imgur.com/4DhJomC.jpg)

[Vertex Lighting GIF (6mb GIF)](https://gfycat.com/deliriousfemalejunebug)

[Tetra3D Docs](https://pkg.go.dev/github.com/solarlune/tetra3d) / [Tetra3D Wiki](https://github.com/SolarLune/Tetra3d/wiki)

## Support

If you want to support development, feel free to check out my [itch.io](https://solarlune.itch.io/masterplan) / [Steam](https://store.steampowered.com/app/1269310/MasterPlan/) / [Patreon](https://www.patreon.com/SolarLune). I also have a [Discord server here](https://discord.gg/cepcpfV). Thanks~!
## What is Tetra3D?

Tetra3D is a 3D hybrid software / hardware renderer written in Go by means of [Ebiten](https://ebiten.org/), primarily for video games. Compared to a professional 3D rendering system like OpenGL or Vulkan, it's slow and buggy, but _it's also janky_, and I love it for that. Tetra3D is largely implemented in software, but uses the GPU a bit for rendering triangles and for depth testing (by use of shaders to compare and write depth and composite the result onto the finished texture). Depth testing can be turned off for a performance increase in exchange for no inter-object intersection.

Tetra's rendering evokes a similar feeling to primitive 3D game consoles like the PS1, N64, or DS. Being that a largely-software renderer is not _nearly_ fast enough for big, modern 3D titles, the best you're going to get out of Tetra is drawing some 3D elements for your primarily 2D Ebiten game, or a relatively simple fully 3D game (i.e. something on the level of a PS1 or N64 game). That said, limitation breeds creativity, and I am intrigued at the thought of what people could make with Tetra.

In general, Tetra3D's just a renderer, so you can target higher resolutions (like 1080p or 4K) _or_ lower resolutions. Anything's fine as long as the target GPU can handle generating the color and depth textures at your desired resolution (assuming you have depth texture rendering on).

## Why did I make it?

Because there's not really too much of an ability to do 3D for gamedev in Go apart from [g3n](http://g3n.rocks), [go-gl](https://github.com/go-gl/gl) and [Raylib-go](https://github.com/gen2brain/raylib-go). I like Go, I like janky 3D, and so, here we are. 

It's also interesting to have the ability to spontaneously do things in 3D sometimes. For example, if you were making a 2D game with Ebiten but wanted to display just a few GUI elements or objects in 3D, Tetra3D should work well for you.

Finally, while a software renderer is not by any means fast, it is relatively simple and easy to use. Any platforms that Ebiten supports should also work for Tetra3D automatically (hopefully!).

## Why Tetra3D? Why is it named that?

Because it's like a [tetrahedron](https://en.wikipedia.org/wiki/Tetrahedron), a relatively primitive (but visually interesting) 3D shape made of 4 triangles. Otherwise, I had other names, but I didn't really like them very much. "Jank3D" was the second-best one, haha.

## How do you get it?

`go get github.com/solarlune/tetra3d`

Tetra depends on kvartborg's [vector](https://github.com/kvartborg/vector) package, and [Ebiten](https://ebiten.org/) itself for rendering. Tetra3D requires Go v1.16 or above. This version is somewhat arbitrary, as it could run on an older Go version if a couple of functions were changed.

## How do you use it?

Make a camera, load a scene, render it. A simple 3D framework means a simple 3D API.

Here's an example:

```go

package main

import (
	"errors"
	"fmt"
	"image/color"

	"github.com/solarlune/tetra3d"
	"github.com/hajimehoshi/ebiten/v2"
)

const ScreenWidth = 398
const ScreenHeight = 224

type Game struct {
	GameScene    *tetra3d.Scene
	Camera       *tetra3d.Camera
}

func NewGame() *Game {

	g := &Game{}

	// First, we load a scene from a .gltf or .glb file. LoadGLTFFile takes a filepath and
	// any loading options (nil is taken as a default), and returns a *Library 
	// and an error if it was unsuccessful. 

	library, err := tetra3d.LoadGLTFFile("example.gltf", nil) 

	if err != nil {
		panic(err)
	}

	// A Library is essentially everything that got exported from your 3D modeler - 
	// all of the scenes, meshes, materials, and animations.
	g.GameScene = library.FindScene("Game")

	// NOTE: If you need to rotate any meshes after importing,
	// you can call Mesh.ApplyMatrix() to apply a rotation matrix 
	// (or any other kind of matrix) to all of its vertices.

	// Tetra uses OpenGL's coordinate system (+X = Right, +Y = Up, +Z = Back), 
	// in comparison to Blender's coordinate system (+X = Right, +Y = Forward, 
	// +Z = Up). Note that when loading models in via GLTF or DAE, models are
	// converted automatically (so up is +Z in Blender and +Y in Tetra3D automatically).

	// Here, we'll create a new Camera. We pass the size of the screen to the 
	// Camera so it can create its own buffer textures (which are *ebiten.Images).

	// If we were to load a camera in from the GLTF file, the loading options struct would 
	// specify the camera's backing texture size (defaulting to 1920x1080).

	g.Camera = tetra3d.NewCamera(ScreenWidth, ScreenHeight)

	// A Camera implements the tetra3d.INode interface, which means it can be placed
	// in 3D space and can be parented to another Node somewhere in the scene tree.
	// Models and Nodes (which are essentially "empties" one can
	// use for positioning and parenting) can, as well.

	// We can place Models, Cameras, and other Nodes with Node.SetWorldPosition() or 
	// Node.SetLocalPosition(). Both functions take a 3D vector.Vector from kvartborg's 
	// vector package.
	
	// The *World variants position Nodes in absolute space; the Local variants
	// position Nodes relative to their parents' positioning and transforms.
	// You can also move Nodes using Node.Move(x, y, z).
	
	// Cameras look forward down the local -Z axis, so we'll move the Camera back 
	// 12 units to look towards [0, 0, 0].
	g.Camera.Move(0, 0, 12)

	// Each Scene has a tree that starts with the Root Node. To add Nodes to the Scene, 
	// parent them to the Scene's base.
	g.GameScene.Root.AddChildren(g.Camera)

	// For Cameras, we don't actually need to have them in the scene to view it, since
	// the presence of the Camera in the Scene node tree doesn't impact what it would see.

	// We can see the tree "visually" by printing out the hierarchy:
	fmt.Println(g.GameScene.Root.HierarchyAsString())

	return g
}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {

	// Call Camera.Clear() to clear its internal backing texture. This
	// should be called once per frame before drawing your Scene.
	g.Camera.Clear()

	// Now we'll render the Scene from the camera. The Camera's ColorTexture will then 
	// hold the result. 
	
	// Below, we'll pass both the Scene and the scene root because 1) the Scene influences
	// how Models draw (fog, for example), and 2) we may not want to render
	// all Models. 
	
	// Camera.RenderNodes() renders all Nodes in a tree, starting with the 
	// Node specified. You can also use Camera.Render() to simply render a selection of
	// Models.
	g.Camera.RenderNodes(g.GameScene, g.GameScene.Root) 

	// Before drawing the result, clear the screen first.
	screen.Fill(color.RGBA{20, 30, 40, 255})

	// Draw the resulting texture to the screen, and you're done! You can 
	// also visualize the depth texture with g.Camera.DepthTexture.
	screen.DrawImage(g.Camera.ColorTexture, nil) 

}

func (g *Game) Layout(w, h int) (int, int) {
	// This is the size of the window; note that we are generally setting it
	// to be the same as the size of the backing camera texture. However,
	// you could use a much larger backing texture size, thereby reducing 
	// certain visual glitches from triangles not drawing tightly enough.
	return ScreenWidth, ScreenHeight
}

func main() {

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}


```

You can also do intersection testing between BoundingObjects, a category of nodes designed for collision / intersection testing. As a simplified example:

```go

type Game struct {

	Cube *tetra3d.BoundingAABB
	Capsule *tetra3d.BoundingCapsule

}

func NewGame() *Game {

	g := &Game{}

	// Create a new BoundingCapsule, 1 unit tall with a 0.25 unit radius for the caps at the ends.
	g.Capsule = tetra3d.NewBoundingCapsule("player", 1, 0.25)

	// Create a new BoundingAABB, of 0.5 width, height, and depth (in that order).
	g.Cube = tetra3d.NewBoundingAABB("block", 0.5, 0.5, 0.5)
	g.Cube.Move(4, 0, 0)

	return g

}

func (g *Game) Update() {

	g.Capsule.Move(0.2, 0, 0)

	// Will print the result of the intersection (an IntersectionResult), or nil, if there was no intersection.
	fmt.Println(g.Capsule.Intersection(g.Cube))

}

```

That's basically it.

Note that Tetra3D is, indeed, a work-in-progress and so will require time to get to a good state. But I feel like it works pretty well as is. Feel free to examine the examples folder for some examples showing how Tetra3D works. Calling `go run .` from within their directories should work.

## What's missing?

The following is a rough to-do list (tasks with checks have been implemented):

- [x] 3D rendering
- [x] -- Perspective projection
- [x] -- Orthographic projection (it's kinda jank, but it works)
- [ ] -- Automatic billboarding (not sure?)
- [ ] -- Sprites (a way to draw 2D images with no perspective changes (if desired), but within 3D space) (not sure?)
- [x] -- Basic depth sorting (sorting vertices in a model according to distance, sorting models according to distance)
- [x] -- A depth buffer and [depth testing](https://learnopengl.com/Advanced-OpenGL/Depth-testing) - This is now implemented by means of a depth texture and [Kage shader](https://ebiten.org/documents/shader.html#Shading_language_Kage), though the downside is that it requires rendering and compositing the scene into textures _twice_. Also, it doesn't work on triangles from the same object (as we can't render to the depth texture while reading it for existing depth).
- [x] -- A more advanced / accurate depth buffer
- [ ] -- Writing depth through some other means than vertex colors for precision
- [ ] -- Depth testing within the same object - I'm unsure if I will be able to implement this.
- [x] -- Offscreen Rendering
- [x] -- Mesh merging - Meshes can be merged together to lessen individual object draw calls.
- [ ] -- Render batching - We can avoid calling Image.DrawTriangles between objects if they share properties (blend mode and texture, for example).
- [ ] -- Texture wrapping (unsure if I can do much about this; I believe Ebiten has its limits in this regard)
- [x] Culling
- [x] -- Backface culling
- [ ] -- Frustum culling (this is implemented, but just with sphere-checks instead of actual frustum code)
- [x] -- Far triangle culling
- [ ] -- Triangle clipping to view (this isn't implemented, but not having it doesn't seem to be too much of a problem for now)
- [x] Debug
- [x] -- Debug text: overall render time, FPS, render call count, vertex count, triangle count, skipped triangle count
- [x] -- Wireframe debug rendering
- [x] -- Normal debug rendering
- [x] Materials
- [x] -- Basic Texturing
- [x] -- Multitexturing / Per-triangle Materials
- [ ] -- Perspective-corrected texturing (currently it's affine, see [Wikipedia](https://en.wikipedia.org/wiki/Texture_mapping#Affine_texture_mapping))
- [ ] Easy dynamic 3D Text (the current idea is to render the text to texture from a font, and then map it to a plane)
- [x] Animations
- [x] -- Armature-based animations
- [x] -- Object transform-based animations
- [x] -- Blending between animations
- [x] -- Linear keyframe interpolation
- [x] -- Constant keyframe interpolation
- [ ] -- Bezier keyframe interpolation
- [ ] -- Morph (mesh-based) animations
- [x] Scenes
- [x] -- Fog
- [x] -- A node or scenegraph for parenting and simple visibility culling
- [x] - Vertex Coloring
- [ ] -- Ambient vertex coloring?
- [ ] -- Multiple vertex color channels
- [x] GLTF / GLB model loading
- [x] -- Vertex colors loading
- [x] -- UV map loading
- [x] -- Normal loading
- [x] -- Transform / full scene loading
- [x] -- Animation loading
- [x] -- Camera loading
- [ ] -- Separate .bin loading
- [x] DAE model loading
- [x] -- Vertex colors loading
- [x] -- UV map loading
- [x] -- Normal loading
- [x] -- Transform / full scene loading
- [x] Lighting
- [x] -- Ambient lights
- [x] -- Point lights
- [x] -- Directional lights
- [x] -- Smooth shading
- [x] -- Take into account view normal (seems most useful for seeing a dark side if looking at a non-backface-culled triangle that is lit) - This is now done for point lights, but not sun lights
- [ ] -- Per-fragment lighting (by pushing it to the GPU, it would be more efficient and look better, of course)
- [x] Shaders
- [x] -- Custom fragment shaders
- [ ] -- Normal rendering (useful for, say, screen-space shaders)
- [x] Intersection Testing:
- [ ] -- Normal reporting
- [x] -- Varying collision shapes

| Collision Type | Sphere | AABB         | Triangle   | Capsule  | Ray (not implemented yet) |
| ------         | ----   | ----         | --------   | -------- | ---- |
| Sphere         |  ✅    | ✅           |    ✅      |  ✅      |   ❌  |
| AABB           | ✅     |  ✅          | ⛔ (buggy) |  ✅      |   ❌  |
| Triangle       | ✅     | ⛔ (buggy)   | ⛔ (buggy) | ✅       |   ❌  |
| Capsule        | ✅     | ✅           | ✅         | ✅       |   ❌  |
| Ray            | ❌     | ❌           | ❌         | ❌       |   ❌  |

- [ ] 3D Sound (just adjusting panning of sound sources based on 3D location, or something like that)
- [ ] Optimization
- [ ] -- Multithreading (particularly for vertex transformations)
- [ ] -- Replace vector.Vector usage with struct-based custom vectors (that aren't allocated to the heap or reallocated unnecessarily, ideally)
- [x] -- Vector pools
- [ ] -- Matrix pools
- [ ] -- Lighting speed improvements 
- [ ] [Prefer Discrete GPU](https://github.com/silbinarywolf/preferdiscretegpu) for computers with both discrete and integrated graphics cards

Again, it's incomplete and jank. However, it's also pretty cool!

## Shout-out time~

Huge shout-out to the open-source community (i.e. StackOverflow, [fauxgl](https://github.com/fogleman/fauxgl), [tinyrenderer](https://github.com/ssloy/tinyrenderer), [learnopengl.com](https://learnopengl.com/Getting-started/Coordinate-Systems), etc) at large for sharing the information and code to make this possible; I would definitely have never made this happen otherwise.
