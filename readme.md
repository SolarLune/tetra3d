# Tetra3D

[View the Examples Online](https://solarlune.github.io/tetra3d.site/)

[Ebitengine Discord](https://discord.gg/fXM7VYASTu)

[SolarLune's Discord](https://discord.gg/cepcpfV)

[TetraTerm, an easy-to-use tool to visualize your game scene](https://github.com/SolarLune/tetraterm)

![Tetra3D Logo](https://user-images.githubusercontent.com/4733521/207243838-b3ece6c4-965a-4cb5-aa81-b6b61f34d4d4.gif)

[Tetra3D Docs](https://pkg.go.dev/github.com/solarlune/tetra3d) / [Tetra3D Wiki](https://github.com/SolarLune/Tetra3d/wiki)

[Quickstart Project Repo](https://github.com/SolarLune/tetra3d-quickstart)

## Support

If you want to support development, feel free to check out my [itch.io](https://solarlune.itch.io/masterplan) / [Steam](https://store.steampowered.com/app/1269310/MasterPlan/) / [Patreon](https://www.patreon.com/SolarLune). I also have a [Discord server here](https://discord.gg/cepcpfV). Thanks~!

## What is Tetra3D?

Tetra3D is a 3D hybrid software / hardware renderer written in Go by means of [Ebitengine](https://ebiten.org/), primarily for video games. Compared to a professional 3D rendering system like OpenGL or Vulkan, it's slow(er), but _it's also janky_, and I love it for that. Tetra3D is largely implemented in software, but uses the GPU a bit for rendering the triangles and for performing inter-object depth testing (by use of Kage shaders to compare and write depth and composite the result onto the finished texture). Depth testing can be turned off for a slight performance increase in exchange for no visual inter-object intersection.

Tetra3D's rendering evokes a similar feeling to primitive 3D game consoles like the PS1, N64, or DS. Being that a largely-software renderer is not _nearly_ fast enough for big, modern 3D titles, the best you're going to get out of Tetra is drawing some 3D elements for your primarily 2D Ebitengine game, or a relatively simple fully 3D game (i.e. something on the level of a PS1, or N64 game). That said, limitation breeds creativity, and I am intrigued at the thought of what people could make with Tetra.

Tetra3D also gives you a Blender add-on to make the Blender > Tetra3D development process flow a bit smoother. See the Releases section for the add-on, and [this wiki page](https://github.com/SolarLune/Tetra3d/wiki/Blender-Addon) for more information.

## Why did I make it?

Because there's not really too much of an ability to do 3D for gamedev in Go apart from [g3n](http://g3n.rocks), [go-gl](https://github.com/go-gl/gl) and [Raylib-go](https://github.com/gen2brain/raylib-go). I like Go, I like janky 3D, and so, here we are.

It's also interesting to have the ability to spontaneously do things in 3D sometimes. For example, if you were making a 2D game with Ebitengine but wanted to display just a few GUI elements or objects in 3D, Tetra3D should work well for you.

Finally, while this hybrid renderer is not by any means fast, it is relatively simple and easy to use. Any platforms that Ebiten supports _should_ also work for Tetra3D automatically. Basing a 3D framework off of an existing 2D framework also means any improvements or refinements to Ebitengine may be of help to Tetra3D, and it keeps the codebase small and unified between platforms.

## Why Tetra3D? Why is it named that?

Because it's like a [tetrahedron](https://en.wikipedia.org/wiki/Tetrahedron), a relatively primitive (but visually interesting) 3D shape made of 4 triangles. Otherwise, I had other names, but I didn't really like them very much. "Jank3D" was the second-best one, haha.

## How do you get it?

`go get github.com/solarlune/tetra3d`

Tetra depends on [Ebitengine](https://ebiten.org/) itself for rendering. Tetra3D requires Go v1.18 or above. This minimum required version is somewhat arbitrary, as it could run on an older Go version if a couple of functions (primarily the ones that loads data from a file directly) were changed.

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

type Game struct {
	GameScene    *tetra3d.Scene
	Camera       *tetra3d.Camera
}

func NewGame() *Game {

	g := &Game{}

	// First, we load a scene from a .gltf or .glb file. LoadGLTFFile takes a filepath and
	// any loading options (nil can be taken as a valid default set of loading options), and
	// returns a *tetra3d.Library and an error if it was unsuccessful. We can also use
	// tetra3d.LoadGLTFData() if we don't have access to the host OS's filesystem (if the
	// assets are embedded, for example).

	library, err := tetra3d.LoadGLTFFile("example.gltf", nil)

	if err != nil {
		panic(err)
	}

	// A Library is essentially everything that got exported from your 3D modeler -
	// all of the scenes, meshes, materials, and animations. The ExportedScene of a Library
	// is the scene that was active when the file was exported.

	// We'll clone the ExportedScene so we don't change it irreversibly; making a clone
	// of a Tetra3D resource (Scene, Node, Material, Mesh, Camera, whatever) makes a deep
	// copy of it.
	g.GameScene = library.ExportedScene.Clone()

	// Tetra3D uses OpenGL's coordinate system (+X = Right, +Y = Up, +Z = Backward [towards the camera]),
	// in comparison to Blender's coordinate system (+X = Right, +Y = Forward,
	// +Z = Up). Note that when loading models in via GLTF or DAE, models are
	// converted automatically (so up is +Z in Blender and +Y in Tetra3D automatically).

	// We could create a new Camera as below - we would pass the size of the screen to the
	// Camera so it can create its own buffer textures (which are *ebiten.Images).

	// g.Camera = tetra3d.NewCamera(ScreenWidth, ScreenHeight)

	// However, we can also just grab an existing camera from the scene if it
	// were exported from the GLTF file - if exported through Blender's Tetra3D add-on,
	// then the camera size can be easily set from within Blender.

	g.Camera = g.GameScene.Root.Get("Camera").(*tetra3d.Camera)

	// Camera implements the tetra3d.INode interface, which means it can be placed
	// in 3D space and can be parented to another Node somewhere in the scene tree.
	// Models, Lights, and Nodes (which are essentially "empties" one can
	// use for positioning and parenting) can, as well.

	// We can place Models, Cameras, and other Nodes with node.SetWorldPosition() or
	// node.SetLocalPosition(). There are also variants that take a 3D Vector.

	// The *World variants of positioning functions takes into account absolute space;
	// the Local variants position Nodes relative to their parents' positioning and
	// transforms (and is more performant.)
	// You can also move Nodes using Node.Move(x, y, z) / Node.MoveVec(vector).

	// Each Scene has a tree that starts with the Root Node. To add Nodes to the Scene,
	// parent them to the Scene's base, like so:

	// scene.Root.AddChildren(object)

	// To remove them, you can unparent them from either the parent (Node.RemoveChildren())
	// or the child (Node.Unparent()). When a Node is unparented, it is removed from the scene
	// tree; if you want to destroy the Node, then dropping any references to this Node
	// at this point would be sufficient.

	// For Cameras, we don't actually need to place them in a scene to view the Scene, since
	// the presence of the Camera in the Scene node tree doesn't impact what it would see.

	// We can see the tree "visually" by printing out the hierarchy:
	fmt.Println(g.GameScene.Root.HierarchyAsString())

	// You can also visualize the scene hierarchy using TetraTerm:
	// https://github.com/SolarLune/tetraterm

	return g
}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {

	// Here, we'll call Camera.Clear() to clear its internal backing texture. This
	// should be called once per frame before drawing your Scene.
	g.Camera.Clear()

	// Now we'll render the Scene from the camera. The Camera's ColorTexture will then
	// hold the result.

	// Camera.RenderScene() renders all Nodes in a scene, starting with the
	// scene's root. You can also use Camera.Render() to simply render a selection of
	// individual Models, or Camera.RenderNodes() to render a subset of a scene tree.
	g.Camera.RenderScene(g.GameScene)

	// To see the result, we draw the Camera's ColorTexture to the screen.
	// Before doing so, we'll clear the screen first. In this case, we'll do this
	// with a color, though we can also go with screen.Clear().
	screen.Fill(color.RGBA{20, 30, 40, 255})

	// Draw the resulting texture to the screen, and you're done! You can
	// also visualize the depth texture with g.Camera.DepthTexture().
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// Note that the resulting texture is indeed just an ordinary *ebiten.Image, so
	// you can also use this as a texture for a Model's Material, as an example.

}

func (g *Game) Layout(w, h int) (int, int) {

	// Here, by simply returning the camera's size, we are essentially
	// scaling the camera's output to the window size and letterboxing as necessary.

	// If you wanted to extend the camera according to window size, you would
	// have to resize the camera using the window's new width and height.
	return g.Camera.Size()

}

func main() {

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}

}


```

You can also do collision testing between BoundingObjects, a category of nodes designed for checking against collisions. As a simplified example:

```go

type Game struct {

	Cube *tetra3d.BoundingAABB
	Capsule *tetra3d.BoundingCapsule

}

func NewGame() *Game {

	g := &Game{}

	// Create a new BoundingCapsule named "player", 1 unit tall with a
	// 0.25 unit radius for the caps at the ends.
	g.Capsule = tetra3d.NewBoundingCapsule("player", 1, 0.25)

	// Create a new BoundingAABB named "block", of 0.5 width, height,
	// and depth (in that order).
	g.Cube = tetra3d.NewBoundingAABB("block", 0.5, 0.5, 0.5)

	// Move the cube over on the X axis by 4 units.
	g.Cube.Move(-4, 0, 0)

	return g

}

func (g *Game) Update() {

	// Move the capsule 0.2 units to the right every frame.
	g.Capsule.Move(0.2, 0, 0)

	// Will print the result of the Collision, or nil, if there was no intersection.
	fmt.Println(g.Capsule.Collision(g.Cube))

}

```

If you wanted a deeper collision test with multiple objects, you can do so using `IBoundingObject.CollisionTest()`. Take a look at the [Wiki](https://github.com/SolarLune/tetra3d/wiki/Collision-Testing) and the `bounds` example for more info.

That's basically it. Feel free to examine all of the examples in the `examples` folder. Calling `go run .` from within their directories will run them - the mouse usually controls the view, and clicking locks and unlocks the view. You can also view the examples online [here](https://solarlune.github.io/tetra3d.site/).

There's a quick start project repo available [here](https://github.com/SolarLune/tetra3d-quickstart), as well to help with getting started.

For more information, check out the [Wiki](https://github.com/SolarLune/Tetra3d/wiki) for tips and tricks.

## What's missing?

The following is a rough to-do list (tasks with checks have been implemented):

-   [x] **3D rendering**
-   [x] -- Perspective projection
-   [x] -- Orthographic projection (it's kinda jank, but it works)
-   [x] -- Automatic billboarding
-   [x] -- Sprites (a way to draw 2D images with no perspective changes (if desired), but within 3D space) (not sure?)
-   [x] -- Basic depth sorting (sorting vertices in a model according to distance, sorting models according to distance)
-   [x] -- A depth buffer and [depth testing](https://learnopengl.com/Advanced-OpenGL/Depth-testing) - This is now implemented by means of a depth texture and [Kage shader](https://ebiten.org/documents/shader.html#Shading_language_Kage), though the downside is that it requires rendering and compositing the scene into textures _twice_. Also, it doesn't work on triangles from the same object (as we can't render to the depth texture while reading it for existing depth).
-   [x] -- A more advanced / accurate depth buffer
-   [x] -- ~~Writing depth through some other means than vertex colors for precision~~ _This is fine for now, I think._
-   [ ] -- Depth testing within the same object - I'm unsure if I will be able to implement this.
-   [x] -- Offscreen Rendering
-   [x] -- Mesh merging - Meshes can be merged together to lessen individual object draw calls.
-   [x] -- Render batching - We can avoid calling Image.DrawTriangles between objects if they share properties (blend mode, material, etc) and it's not too many triangles to push before flushing to the GPU. Perhaps these Materials can have a flag that you can toggle to enable this behavior? (EDIT: This has been partially added by dynamic batching of Models.)
-   [ ] -- Texture wrapping (will require rendering with shaders) - This is kind of implemented, but I don't believe it's been implemented for alpha clip materials.
-   [ ] -- Draw ~~triangles~~ shapes in 3D space through a function (could be useful for 3D lines, for example)
-   [x] -- 3D Text (2D text, rendered on an appropriately-sized 3D plane)
-   [x] -- -- Typewriter effect
-   [x] -- -- Customizeable cursor
-   [x] -- -- Horizontal alignment
-   [ ] -- -- Vertical alignment
-   [ ] -- -- Vertical Scrolling
-   [ ] -- -- Replace style setting system with dedicated Style object, with a function to flush various style changes to batch and update the Text texture all at once?
-   [x] -- -- Outlines
-   [x] -- -- Shadows
-   [ ] -- -- Gradients
-   [ ] -- -- -- Per-letter gradient
-   [ ] -- -- -- Per-sentence gradient?
-   [ ] -- -- -- Per-block gradient?
-   [ ] -- -- -- Other patterns?
-   [ ] -- -- Parsing text for per-letter effects (this would probably require rendering the glyphs from a font to individual images to render; could also involve shaders?)
-   [ ] -- -- -- Per-letter colors
-   [ ] -- -- -- Bold
-   [ ] -- -- -- Italics
-   [ ] -- -- -- Strikethrough
-   [ ] -- -- -- Letters fading in or out, flickering?
-   [ ] -- -- -- Letters changing to other glyphs randomly
-   [ ] -- -- Additional effects? (Wavy text, shaky text, etc.)
-   [x] -- Multitexturing / Per-triangle Materials
-   [x] -- Perspective-corrected texturing (currently it's affine, see [Wikipedia](https://en.wikipedia.org/wiki/Texture_mapping#Affine_texture_mapping))
-   [ ] -- Automatic triangle / mesh subdivision depending on distance
-   [ ] -- Automatic level of detail
-   [ ] -- Manual level of detail (ability to render a model using various meshes in stages); note that these stages should be accessible at runtime to allow cloning meshes, for example
-   [ ] -- Possibly some UI utilities?
-   [x] **Culling**
-   [x] -- Backface culling
-   [x] -- Frustum culling
-   [x] -- Far triangle culling
-   [ ] -- Triangle clipping to view (this isn't implemented, but not having it doesn't seem to be too much of a problem for now)
-   [x] -- Sectors - The general idea is that the camera can be set up to only render sectors that it's in / neighboring (up to a customizeable depth)
-   [ ] -- -- Some method to have objects appear in multiple Sectors, but not others?
-   [ ] -- Occlusion culling - this should be possible using octrees to determine if an object is visible before rendering it; see: https://www.gamedeveloper.com/programming/occlusion-culling-algorithms
-   [ ] -- -- Something to reduce overdraw
-   [x] **Debug**
-   [x] -- Debug text: overall render time, FPS, render call count, vertex count, triangle count, skipped triangle count
-   [x] -- Wireframe debug rendering
-   [ ] -- TODO: Utilize depth testing and backface culling for wireframe rendering. Also replace ebitenutil.DrawLine() with the non-deprecated vector package usage.
-   [x] -- Normal debug rendering
-   [x] -- Bounding object debug rendering
-   [x] **Materials**
-   [x] -- Basic Texturing
-   [ ] -- Ability to use screen coordinates instead of just UV texturing (useful for repeating patterns)
-   [ ] -- Replace opaque transparency mode with just automatic transparency mode? I feel like there might be a reason to have opaque separate, but I can't imagine a normal situation where you'd want it when you could just go with auto + setting alpha to 1
-   [x] **Animations**
-   [x] -- Armature-based animations
-   [x] -- Object transform-based animations
-   [x] -- Blending between animations
-   [x] -- Linear keyframe interpolation
-   [x] -- Constant keyframe interpolation
-   [ ] -- Bezier keyframe interpolation
-   [ ] -- Morph (mesh-based / shape key) animations (See: https://github.com/KhronosGroup/glTF-Tutorials/blob/master/gltfTutorial/gltfTutorial_017_SimpleMorphTarget.md)
-   [ ] -- Mesh swapping during animations, primarily for 2D skeletal animation? (Can be worked around using bones.)
-   [x] **Scenes**
-   [x] -- Fog
-   [x] -- A node or scenegraph for parenting and simple visibility culling
-   [x] -- Ambient vertex coloring
-   [x] **GLTF / GLB model loading**
-   [x] -- Vertex colors loading
-   [x] -- Multiple vertex color channels
-   [x] -- UV map loading
-   [x] -- Normal loading
-   [x] -- Transform / full scene loading
-   [x] -- Animation loading
-   [x] -- Camera loading
-   [x] -- Loading world color in as ambient lighting
-   [ ] -- Separate .bin loading
-   [x] -- Support for multiple scenes in a single Blend file (was broken due to GLTF exporter changes; working again in Blender 3.3)
-   [x] **Blender Add-on**
-   [x] -- Export 3D view camera to Scenes for quick iteration
-   [ ] -- Object-level color option
-   [ ] -- Object-level shadeless checkbox
-   [ ] -- Custom mesh attribute to assign values to vertices, allowing you to, say, "mark" vertices (SolarLune 3/3/24: Doesn't seem to be possible outside of vertex colors)
-   [ ] -- Exporting animations and auto-subdivided mesh data doesn't work properly when the scene is not focused in Blender
-   [ ] -- Sharing materials and textures doesn't work properly
-   [x] -- Export GLTF on save / on command via button
-   [x] -- Bounds node creation
-   [x] -- Game property export (less clunky version of Blender's vanilla custom properties)
-   [x] -- Collection / group substitution
-   [x] -- -- Overwriting properties through collection instance objects (it would be nice to do this cleanly with a nice UI, but just hamfisting it is fine for now)
-   [x] -- -- Collection instances instantiate their objects in the same location in the tree
-   [x] -- Optional camera size export
-   [x] -- Linking collections from external files
-   [x] -- Material data export
-   [x] -- Option to pack textures or leave them as a path
-   [x] -- Path / 3D Curve support
-   [x] -- Grid support (for pathfinding / linking 3D points together)
-   [ ] -- -- Adding costs to pathfinding (should be as simple as adding a cost and currentcost to each GridPoint, then sorting the points to check by cost when pathfinding, then reduce all costs greater than 1 by 1 ) (7/5/23, SolarLune: This works currently, but the pathfinding is still a bit wonky, so it should be looked at again)
-   [ ] -- Toggleable option for drawing game property status to screen for each object using the gpu and blf modules
-   [ ] -- Game properties should be an ordered slice, rather than a map of property name to property values. (5/22/23, SolarLune: should it be?)
-   [ ] -- Consistency between Tetra3D material settings and Blender viewport (so modifying the options in the Tetra3D material panel alters the relevant options in a default material to not mess with it; maybe the material settings should even be wholly disabled for this purpose? It would be great if the models looked the way you'd expect)
-   [ ] -- Components (This would also require meta-programming; it'd be nice if components could have elements that were adjustable in Blender. Maybe a "game object" can have a dynamically-written "Components" struct, with space for one of each kind of component, for simplicity (i.e. one physics controller, one gameplay controller, one animation component, etc). There doesn't need to be ways to add or remove components from an object, and components can have any of OnInit, OnAdd, OnRemove, or Update functions to be considered components).
-   [x] **DAE model loading**
-   [x] -- Vertex colors loading
-   [x] -- UV map loading
-   [x] -- Normal loading
-   [x] -- Transform / full scene loading
-   [x] **Lighting**
-   [x] -- Smooth shading
-   [x] -- Ambient lights
-   [x] -- Point lights
-   [x] -- Directional lights
-   [x] -- Cube (AABB volume) lights
-   [x] -- Lighting Groups
-   [x] -- Ability to bake lighting to vertex colors
-   [x] -- Ability to bake ambient occlusion to vertex colors
-   [ ] -- Specular lighting (shininess)
-   [ ] -- Lighting Probes - general idea is to be able to specify a space that has basic (optionally continuously updated) AO and lighting information, so standing a character in this spot makes him greener, that spot redder, that spot darker because he's in the shadows, etc.
-   [ ] -- Lightmaps - might be possible with being able to use multiple textures at the same time now?
-   [ ] -- Baking AO and lighting into vertex colors? from Blender? It's possible to do already using Cycles, but not very easy or simple.
-   [ ] -- Take into account view normal (seems most useful for seeing a dark side if looking at a non-backface-culled triangle that is lit) - This is now done for point lights, but not sun lights
-   [ ] -- Per-fragment lighting (by pushing it to the GPU, it would be more efficient and look better, of course)
-   [x] **Particles**
-   [x] -- Basic particle system support
-   [ ] -- Fix layering issue when rendering a particle system underneath another one (visible in the Particles example)
-   [ ] -- Implement soft particles (see [this article](https://dev.to/keaukraine/implementing-soft-particles-in-webgl-and-opengl-es-3l6e))
-   [x] **Shaders**
-   [x] -- Custom fragment shaders
-   [x] -- Normal rendering (useful for, say, screen-space shaders)
-   [x] **Collision Testing**
-   [x] -- Normal reporting
-   [x] -- Slope reporting
-   [x] -- Contact point reporting
-   [x] -- Varying collision shapes
-   [x] -- Checking multiple collisions at the same time
-   [x] -- Composing collision shapes out of multiple sub-shapes (this can be done by simply creating them, parenting them to some node, and then testing against that node)
-   [x] -- Bounding / Broadphase collision checking

-   -- Supported collision types:

| Collision Type | Sphere | AABB | Triangle | Capsule |
| -------------- | ------ | ---- | -------- | ------- |
| Sphere         | ✅     | ✅   | ✅       | ✅      |
| AABB           | ✅     | ✅   | ⛔       | ✅      |
| Triangle       | ✅     | ⛔   | ⛔       | ✅      |
| Capsule        | ✅     | ✅   | ✅       | ⛔      |
| Ray            | ✅     | ✅   | ✅       | ✅      |

✅ = Fully supported, should be working without issues
⛔ = Partially supported; somewhat buggy

-   [ ] -- An actual collision system, rather than simply doing basic collision checks?
-   [ ] -- Scaling triangle mesh objects won't work (particularly the broadphase object)

-   [ ] **3D Sound** (adjusting panning of sound sources based on 3D location?)
-   [ ] **Optimization / Bugfixes**
-   [ ] -- It's currently problematic to set world scale and rotation (and possible position as well) at the same time.
-   [ ] -- Try using [avo](https://github.com/mmcloughlin/avo) to generate assembly for SIMD-ing, particularly Matrix x Vector multiplication
-   [ ] -- Try using ebiten.Vertex directly, rather than storing data in Mesh.VertexTransforms and then copying that to the vertex list. Maybe the W component could be stored in the current Custom0 attribute and perspective divide could be done in the Kage shader.
-   [ ] -- It might be possible to not have to write depth manually (5/22/23, SolarLune: Not sure what past me meant by this)
-   [ ] -- Minimize texture-swapping - should be possible to do now that Kage shaders can handle images of multiple sizes.
-   [x] -- Make NodeFilters work lazily, rather than gathering all nodes in the filter at once
        -- [x] -- Optimize NodeFilters to avoid creating slices unnecessarily by avoiding using Node.Children() and instead use callbacks
        -- [ ] -- Allow NodeFilters to append filtered nodes to a shared slice to have sorting available while still not allocating new memory unless the internal slice grows
-   [x] -- Reusing vertex indices for adjacent triangles
-   [ ] -- Multithreading (particularly for vertex transformations)
-   [x] -- Armature animation improvements?
-   [x] -- Custom Vectors
-   [ ] -- -- Move Vector and Matrix to external package for simplicity / resuability / create separate Vector types for 2D / 4D (?) Vectors. Maybe Colors as well?
-   [ ] -- Investigate replacing vector multiplaction with in-place vector multiplication
-   [ ] -- -- Vector pools again?
-   [ ] -- -- Move over to float32 for mathematics - seems like it should be faster? Should be possible with math32 : https://github.com/chewxy/math32
-   [ ] -- -- Maybe make Vectors generic, so you could have float32 / int Vectors? Or something like that?
-   [ ] -- Matrix pools?
-   [ ] -- Raytest optimization
-   [ ] -- -- Reduce memory allocation by not using functions that return new Vectors, but rather simply perform math within the Vectors themselves
-   [ ] -- -- Sphere?
-   [ ] -- -- AABB?
-   [ ] -- -- Capsule?
-   [ ] -- -- Triangles
-   [ ] -- -- -- Maybe we can combine contiguous triangles into faces and just check faces?
-   [ ] -- -- -- We could use the broadphase system to find triangles that are in the raycast's general area, specifically
-   [ ] -- Instead of doing collision testing using triangles directly, we can test against planes / faces if possible to reduce checks?
-   [ ] -- Lighting speed improvements
-   [ ] -- Resource tracking system to ease cloning elements (i.e. rather than live-cloning Meshes, it would be faster to re-use "dead" Meshes)
-   [ ] -- -- Model
-   [ ] -- -- Mesh
-   [ ] -- [Prefer Discrete GPU](https://github.com/silbinarywolf/preferdiscretegpu) for computers with both discrete and integrated graphics cards
-   [x] -- Replace \*Color with just the plain Color struct (this would be a breaking change)
-   [ ] -- Replace color usage with HTML or W3C colors? : https://www.computerhope.com/htmcolor.htm#gray / https://www.computerhope.com/jargon/w/w3c-color-names.htm
-   [ ] -- Update to use Generics where possible; we're already on Go 1.18.
-   [ ] -- Move utility objects (quaternion, vector, color, text, matrix, treewatcher, etc) to utility package.
-   [ ] -- Optimize getting an object by path; maybe this could be done with some kind of string serialization, rather than text parsing?
-   [ ] -- Replace panics / log.prints with error returns (e.g. gltf.go)

Again, it's incomplete and jank. However, it's also pretty cool!

## Shout-out time~

Huge shout-out to the open-source community:

-   StackOverflow, in general, _FOR REAL_ - this would've been impossible otherwise
-   [quartercastle's vector package](https://github.com/quartercastle/vector)
-   [fauxgl](https://github.com/fogleman/fauxgl)
-   [tinyrenderer](https://github.com/ssloy/tinyrenderer)
-   [learnopengl.com](https://learnopengl.com/Getting-started/Coordinate-Systems)
-   [3DCollisions](https://gdbooks.gitbooks.io/3dcollisions/content/)
-   [Simon Rodriguez's "Writing a Small Software Renderer" article](http://blog.simonrodriguez.fr/articles/2017/02/writing_a_small_software_renderer.html)
-   And many other articles that I've forgotten to note down

... For sharing the information and code to make this possible; I would definitely have never been able to create this otherwise.
