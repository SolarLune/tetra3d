package tetra3d

// Scene represents a world of sorts, and can contain a variety of Meshes and Nodes, which organize the scene into a
// graph of parents and children. Models (visual instances of Meshes), Cameras, and "empty" NodeBases all are kinds of Nodes.
type Scene struct {
	Name    string   // The name of the Scene. Set automatically to the scene name in your 3D modeler if the DAE file exports it.
	library *Library // The library from which this Scene was created. If the Scene was instantiated through code, this will be nil.
	// Root indicates the root node for the scene hierarchy. For visual Models to be displayed, they must be added to the
	// scene graph by simply adding them into the tree via parenting anywhere under the Root. For them to be removed from rendering,
	// they simply need to be removed from the tree.
	// See this page for more information on how a scene graph works: https://webglfundamentals.org/webgl/lessons/webgl-scene-graph.html
	Root  INode
	World *World
}

// NewScene creates a new Scene by the name given.
func NewScene(name string) *Scene {

	scene := &Scene{
		Name:  name,
		Root:  NewNode("Root"),
		World: NewWorld("World"),
	}

	scene.Root.(*Node).scene = scene

	return scene
}

// Clone clones the Scene, returning a copy. Models and Meshes are shared between them.
func (scene *Scene) Clone() *Scene {

	newScene := NewScene(scene.Name)
	newScene.library = scene.library

	// newScene.Models = append(newScene.Models, scene.Models...)
	newScene.Root = scene.Root.Clone()
	newScene.Root.(*Node).scene = newScene

	newScene.World = scene.World // Here, we simply reference the same world; we don't clone it, since a single world can be shared across multiple Scenes

	return newScene

}

// Library returns the Library from which this Scene was loaded. If it was created through code and not associated with a Library, this function will return nil.
func (scene *Scene) Library() *Library {
	return scene.library
}
