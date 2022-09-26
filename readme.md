# Tetra3D

[Ebitengine Discord](https://discord.gg/fXM7VYASTu)

![Tetra3D Logo](https://thumbs.gfycat.com/DifferentZealousFowl-size_restricted.gif)

![It Breeds Fear - Construction Worker](https://thumbs.gfycat.com/ThoughtfulChubbyBunny-size_restricted.gif)

![Dark exploration](https://thumbs.gfycat.com/ScalySlimCrayfish-size_restricted.gif)

[Tetra3D Docs](https://pkg.go.dev/github.com/solarlune/tetra3d) / [Tetra3D Wiki](https://github.com/SolarLune/Tetra3d/wiki)

[Quickstart Project Repo](https://github.com/SolarLune/tetra3d-quickstart)

## Support

If you want to support development, feel free to check out my [itch.io](https://solarlune.itch.io/masterplan) / [Steam](https://store.steampowered.com/app/1269310/MasterPlan/) / [Patreon](https://www.patreon.com/SolarLune). I also have a [Discord server here](https://discord.gg/cepcpfV). Thanks~!

## What is Tetra3D?

Tetra3D is a 3D hybrid software / hardware renderer written in Go by means of [Ebitengine](https://ebiten.org/), primarily for video games. Compared to a professional 3D rendering system like OpenGL or Vulkan, it's slow and buggy, but _it's also janky_, and I love it for that. Tetra3D is largely implemented in software, but uses the GPU a bit for rendering triangles and for depth testing (by use of shaders to compare and write depth and composite the result onto the finished texture). Depth testing can be turned off for a slight performance increase in exchange for no visual inter-object intersection.

Tetra3D's rendering evokes a similar feeling to primitive 3D game consoles like the PS1, N64, or DS. Being that a largely-software renderer is not _nearly_ fast enough for big, modern 3D titles, the best you're going to get out of Tetra is drawing some 3D elements for your primarily 2D Ebitengine game, or a relatively simple fully 3D game (i.e. something on the level of a PS1, or N64 game). That said, limitation breeds creativity, and I am intrigued at the thought of what people could make with Tetra.

In general, Tetra3D's just a renderer, so you can target higher resolutions (like 1080p or 4K) _or_ lower resolutions. Anything's fine as long as the target GPU can handle generating the color and depth textures at your desired resolution (assuming you have depth texture rendering on).

Tetra3D also gives you a Blender add-on to make the Blender > Tetra3D development process flow a bit smoother. See the Releases section for the add-on, and [this wiki page](https://github.com/SolarLune/Tetra3d/wiki/Blender-Addon) for more information.

## Why did I make it?

Because there's not really too much of an ability to do 3D for gamedev in Go apart from [g3n](http://g3n.rocks), [go-gl](https://github.com/go-gl/gl) and [Raylib-go](https://github.com/gen2brain/raylib-go). I like Go, I like janky 3D, and so, here we are.

It's also interesting to have the ability to spontaneously do things in 3D sometimes. For example, if you were making a 2D game with Ebitengine but wanted to display just a few GUI elements or objects in 3D, Tetra3D should work well for you.

Finally, while this hybrid renderer is not by any means fast, it is relatively simple and easy to use. Any platforms that Ebiten supports _should_ also work for Tetra3D automatically. Basing a 3D framework off of an existing 2D framework also means any improvements or refinements to Ebitengine may be of help to Tetra3D, and it keeps the codebase small and unified between platforms.

## Why Tetra3D? Why is it named that?

Because it's like a [tetrahedron](https://en.wikipedia.org/wiki/Tetrahedron), a relatively primitive (but visually interesting) 3D shape made of 4 triangles. Otherwise, I had other names, but I didn't really like them very much. "Jank3D" was the second-best one, haha.

## How do you get it?

`go get github.com/solarlune/tetra3d`

Tetra depends on kvartborg's [vector](https://github.com/kvartborg/vector) package, and [Ebitengine](https://ebiten.org/) itself for rendering. Tetra3D requires Go v1.16 or above. This minimum required version is somewhat arbitrary, as it could run on an older Go version if a couple of functions (primarily the ones that loads data from a file directly) were changed.

There is an optional Blender add-on as well (`tetra3d.py`) that can be downloaded from the releases page or from the repo directly (i.e. click on the file and download it). The add-on provides some useful helper functionality that makes using Tetra3D simpler - for more information, check the [Wiki](https://github.com/SolarLune/Tetra3d/wiki/Blender-Addon).

## How do you use it?

Load a scene, render it. A simple 3D framework means a simple 3D API.

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

const ScreenWidth = 786
const ScreenHeight = 448

type Game struct {
	GameScene    *tetra3d.Scene
	Camera       *tetra3d.Camera
}

func NewGame() *Game {

	g := &Game{}

	// First, we load a scene from a .gltf or .glb file. LoadGLTFFile takes a filepath and
	// any loading options (nil is taken as a default), and returns a *Library 
	// and an error if it was unsuccessful. We can also use tetra3d.LoadGLTFData() if we don't
	// have access to the host OS's filesystem (like on web, or if the assets are embedded).

	options := tetra3d.DefaultGLTFLoadOptions()
	options.CameraWidth = ScreenWidth
	options.CameraHeight = ScreenHeight

	library, err := tetra3d.LoadGLTFFile("example.gltf", options) 

	if err != nil {
		panic(err)
	}

	// A Library is essentially everything that got exported from your 3D modeler - 
	// all of the scenes, meshes, materials, and animations.

	// The ExportedScene of a Library is the scene that was active when the file was exported.

	// We'll clone the ExportedScene so we don't change it irreversibly; making a clone
	// of a Tetra3D resource makes a deep copy of it.
	g.GameScene = library.ExportedScene.Clone()

	// Tetra3D uses OpenGL's coordinate system (+X = Right, +Y = Up, +Z = Forward), 
	// in comparison to Blender's coordinate system (+X = Right, +Y = Forward, 
	// +Z = Up). Note that when loading models in via GLTF or DAE, models are
	// converted automatically (so up is +Z in Blender and +Y in Tetra3D automatically).

	// We could create a new Camera as below - we would pass the size of the screen to the 
	// Camera so it can create its own buffer textures (which are *ebiten.Images).

	// g.Camera = tetra3d.NewCamera(ScreenWidth, ScreenHeight)

	// However, we can also just grab an existing camera from the scene if it 
	// were exported from the GLTF file. The loading options struct speciies the camera's
	// backing texture size (defaulting to 1920x1080 if either
	// no loading options are passed, or the default loading options struct is used
	// unaltered).

	g.Camera = g.GameScene.Root.Get("Camera").(*tetra3d.Camera)

	// Camera implements the tetra3d.INode interface, which means it can be placed
	// in 3D space and can be parented to another Node somewhere in the scene tree.
	// Models, Lights, and Nodes (which are essentially "empties" one can
	// use for positioning and parenting) can, as well.

	// We can place Models, Cameras, and other Nodes with node.SetWorldPositionVec() or 
	// node.SetLocalPositionVec(). Both functions take a 3D vector.Vector from kvartborg's 
	// vector package, and there's an additional variant that just takes the components directly,
	// for convenience.

	// The *World variants position Nodes in absolute space; the Local variants
	// position Nodes relative to their parents' positioning and transforms (and is
	// more performant.)
	// You can also move Nodes using Node.Move(x, y, z) / Node.MoveVec(vector).

	// Each Scene has a tree that starts with the Root Node. To add Nodes to the Scene, 
	// parent them to the Scene's base, like so:

	// g.GameScene.Root.AddChildren(object)

	// To remove them, either use Node.RemoveChildren() or Node.Unparent().

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
	// how Models draw (fog, for example), and 2) we may not want to render all Models. 

	// Camera.RenderNodes() renders all Nodes in a tree, starting with the 
	// Node specified. You can also use Camera.Render() to simply render a selection of
	// individual Models.
	g.Camera.RenderNodes(g.GameScene, g.GameScene.Root) 

	// Before drawing the result, clear the screen first; in this case, with a color.
	screen.Fill(color.RGBA{20, 30, 40, 255})

	// Draw the resulting texture to the screen, and you're done! You can 
	// also visualize the depth texture with g.Camera.DepthTexture.
	screen.DrawImage(g.Camera.ColorTexture, nil) 

}

func (g *Game) Layout(w, h int) (int, int) {
	// This is the size of the window; note that we set it
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

You can also do collision testing between BoundingObjects, a category of nodes designed for this purpose. As a simplified example:

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

	// Move it over on the X axis by 4 units.
	g.Cube.Move(4, 0, 0)

	return g

}

func (g *Game) Update() {

	// Move the capsule 0.2 units to the right every frame.
	g.Capsule.Move(0.2, 0, 0)

	// Will print the result of the Collision, or nil, if there was no intersection.
	fmt.Println(g.Capsule.Collision(g.Cube))

}

```

That's basically it.

Note that Tetra3D is, indeed, a work-in-progress and so will require time to get to a good state. But I feel like it works pretty well as is. Feel free to examine the examples folder for some examples showing how Tetra3D works. Calling `go run .` from within their directories should work.

There's a quick start project repo available [here](https://github.com/SolarLune/tetra3d-quickstart), as well to help with getting started.

For more information, check out the [Wiki](https://github.com/SolarLune/Tetra3d/wiki) for tips and tricks.

## What's missing?

The following is a rough to-do list (tasks with checks have been implemented):

- [X] **3D rendering**
- [X] -- Perspective projection
- [X] -- Orthographic projection (it's kinda jank, but it works)
- [x] -- Automatic billboarding
- [ ] -- Sprites (a way to draw 2D images with no perspective changes (if desired), but within 3D space) (not sure?)
- [X] -- Basic depth sorting (sorting vertices in a model according to distance, sorting models according to distance)
- [X] -- A depth buffer and [depth testing](https://learnopengl.com/Advanced-OpenGL/Depth-testing) - This is now implemented by means of a depth texture and [Kage shader](https://ebiten.org/documents/shader.html#Shading_language_Kage), though the downside is that it requires rendering and compositing the scene into textures _twice_. Also, it doesn't work on triangles from the same object (as we can't render to the depth texture while reading it for existing depth).
- [X] -- A more advanced / accurate depth buffer
- [ ] -- Writing depth through some other means than vertex colors for precision
- [ ] -- Depth testing within the same object - I'm unsure if I will be able to implement this.
- [X] -- Offscreen Rendering
- [X] -- Mesh merging - Meshes can be merged together to lessen individual object draw calls.
- [x] -- Render batching - We can avoid calling Image.DrawTriangles between objects if they share properties (blend mode, material, etc) and it's not too many triangles to push before flushing to the GPU. Perhaps these Materials can have a flag that you can toggle to enable this behavior? (EDIT: This has been partially added by dynamic batching of Models.)
- [ ] -- Texture wrapping (will require rendering with shaders) - This is kind of implemented, but I don't believe it's been implemented for alpha clip materials.
- [ ] -- Draw triangle in 3D space through a function (could be useful for 3D lines, for example)
- [ ] -- Easy dynamic 3D Text (to make this simple, it might be best to allow the user to render the text as he wishes, and then make a function to map it (or any other *Image) to a plane of variable size).
- [ ] -- Lighting Probes - general idea is to be able to specify a space that has basic (optionally continuously updated) AO and lighting information, so standing a character in this spot makes him greener, that spot redder, that spot darker because he's in the shadows, etc.
- [X] **Culling**
- [X] -- Backface culling
- [X] -- Frustum culling
- [X] -- Far triangle culling
- [ ] -- Triangle clipping to view (this isn't implemented, but not having it doesn't seem to be too much of a problem for now)
- [X] **Debug**
- [X] -- Debug text: overall render time, FPS, render call count, vertex count, triangle count, skipped triangle count
- [X] -- Wireframe debug rendering
- [X] -- Normal debug rendering
- [X] **Materials**
- [X] -- Basic Texturing
- [X] -- Multitexturing / Per-triangle Materials
- [ ] -- Perspective-corrected texturing (currently it's affine, see [Wikipedia](https://en.wikipedia.org/wiki/Texture_mapping#Affine_texture_mapping))
- [X] **Animations**
- [X] -- Armature-based animations
- [X] -- Object transform-based animations
- [X] -- Blending between animations
- [X] -- Linear keyframe interpolation
- [X] -- Constant keyframe interpolation
- [ ] -- Bezier keyframe interpolation
- [ ] -- Morph (mesh-based) animations
- [X] **Scenes**
- [X] -- Fog
- [X] -- A node or scenegraph for parenting and simple visibility culling
- [ ] -- Ambient vertex coloring?
- [ ] -- Multiple vertex color channels
- [X] **GLTF / GLB model loading**
- [X] -- Vertex colors loading
- [X] -- UV map loading
- [X] -- Normal loading
- [X] -- Transform / full scene loading
- [X] -- Animation loading
- [X] -- Camera loading
- [X] -- Loading world color in as ambient lighting
- [ ] -- Separate .bin loading
- [x] -- Support for multiple scenes in a single Blend file (was broken due to GLTF exporter changes; working again in Blender 3.3)
- [X] **Blender Add-on**
- [X] -- Export GLTF on save / on command via button
- [X] -- Bounds node creation
- [X] -- Game property export (less clunky version of Blender's vanilla custom properties)
- [X] -- Collection / group substitution
- [ ] -- -- Overwriting properties through collection instance
- [ ] -- Optional camera size export
- [X] -- Linking collections from external files
- [X] -- Material data export
- [X] -- Option to pack textures or leave them as a path
- [X] -- Path / 3D Curve support
- [X] -- Grid support (for pathfinding / linking 3D points together)
- [X] **DAE model loading**
- [X] -- Vertex colors loading
- [X] -- UV map loading
- [X] -- Normal loading
- [X] -- Transform / full scene loading
- [X] **Lighting**
- [X] -- Smooth shading
- [X] -- Ambient lights
- [X] -- Point lights
- [X] -- Directional lights
- [X] -- Cube (AABB volume) lights
- [X] -- Lighting Groups
- [X] -- Ability to bake lighting to vertex colors
- [X] -- Ability to bake ambient occlusion to vertex colors
- [ ] -- Take into account view normal (seems most useful for seeing a dark side if looking at a non-backface-culled triangle that is lit) - This is now done for point lights, but not sun lights
- [ ] -- Per-fragment lighting (by pushing it to the GPU, it would be more efficient and look better, of course)
- [X] **Shaders**
- [X] -- Custom fragment shaders
- [ ] -- Normal rendering (useful for, say, screen-space shaders)
- [X] **Collision Testing**
- [X] -- Normal reporting
- [X] -- Slope reporting
- [X] -- Contact point reporting
- [X] -- Varying collision shapes
- [X] -- Checking multiple collisions at the same time
- [X] -- Bounding / Broadphase collision checking


| Collision Type | Sphere | AABB       | Triangle   | Capsule | Ray (not implemented yet) |
| ---------------- | -------- | ------------ | ------------ | --------- | --------------------------- |
| Sphere         | ✅     | ✅         | ✅         | ✅      | ❌                        |
| AABB           | ✅     | ✅         | ⛔ (buggy) | ✅      | ❌                        |
| Triangle       | ✅     | ⛔ (buggy) | ⛔ (buggy) | ✅      | ❌                        |
| Capsule        | ✅     | ✅         | ✅         | ✅      | ❌                        |
| Ray            | ❌     | ❌         | ❌         | ❌      | ❌                        |

- [ ] **3D Sound** (adjusting panning of sound sources based on 3D location)
- [ ] **Optimization**
- [ ] -- Multithreading (particularly for vertex transformations)
- [X] -- Armature animation improvements?
- [ ] -- Replace vector.Vector usage with struct-based custom vectors (that aren't allocated to the heap or reallocated unnecessarily, ideally)?
- [X] -- Vector pools
- [ ] -- Matrix pools?
- [ ] -- Instead of doing collision testing using triangles directly, we can test against planes / faces if possible to reduce checks?
- [ ] -- Lighting speed improvements
- [ ] -- [Prefer Discrete GPU](https://github.com/silbinarywolf/preferdiscretegpu) for computers with both discrete and integrated graphics cards

Again, it's incomplete and jank. However, it's also pretty cool!

## Shout-out time~

Huge shout-out to the open-source community:

- StackOverflow, in general
- [fauxgl](https://github.com/fogleman/fauxgl)
- [tinyrenderer](https://github.com/ssloy/tinyrenderer)
- [learnopengl.com](https://learnopengl.com/Getting-started/Coordinate-Systems)
- [3DCollisions](https://gdbooks.gitbooks.io/3dcollisions/content/)
- [Simon Rodriguez's "Writing a Small Software Renderer" article](http://blog.simonrodriguez.fr/articles/2017/02/writing_a_small_software_renderer.html)
- And many other articles that I've forgotten to note down

... For sharing the information and code to make this possible; I would definitely have never been able to create this otherwise.
