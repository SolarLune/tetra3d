package tetra3d

import "fmt"

// There is no zero ID; this is to make it so that systems that reference scenes by id can use 0 as an invalid reference.
var sceneID uint32 = 1

// Scene represents a world of sorts, and can contain a variety of Meshes and Nodes, which organize the scene into a
// graph of parents and children.
type Scene struct {
	id      uint32   // Unique ID for this scene
	name    string   // The name of the Scene. Set automatically to the scene name in your 3D modeler if the DAE file exports it.
	library *Library // The library from which this Scene was created. If the Scene was instantiated through code, this will be nil.
	// Root indicates the root node for the scene hierarchy. For visual Models to be displayed, they must be added to the
	// scene graph by simply adding them into the tree via parenting anywhere under the Root. For them to be removed from rendering,
	// they simply need to be removed from the tree.
	// See this page for more information on how a scene graph works: https://webglfundamentals.org/webgl/lessons/webgl-scene-graph.html
	Root          *Node
	World         *World
	props         Properties
	data          any
	View3DCameras []*Camera // Any 3D view cameras that were exported from Blender

	updateAutobatch     bool
	autobatchDynamicMap map[*Material]*Model
	autobatchStaticMap  map[*Material]*Model

	callbacks *SceneCallbacks
}

// NewScene creates a new Scene by the name given.
func NewScene(name string) *Scene {

	scene := &Scene{
		id:                  sceneID,
		name:                name,
		Root:                NewNode("Root"),
		World:               NewWorld("World"),
		props:               NewProperties(),
		autobatchDynamicMap: map[*Material]*Model{},
		autobatchStaticMap:  map[*Material]*Model{},
		callbacks:           &SceneCallbacks{},
	}

	sceneID++

	scene.Root.scene = scene
	scene.Root.cachedSceneRootNode = scene.Root

	return scene
}

// Clone clones the Scene, returning a copy. Meshes are shared between them.
func (scene *Scene) Clone() *Scene {

	newScene := NewScene(scene.name)
	newScene.library = scene.library

	// Just for when you clone
	scene.Root.scene = newScene

	runCallbacks = false

	newScene.Root = scene.Root.Clone().(*Node)

	runCallbacks = true

	scene.Root.scene = scene

	newScene.Root.scene = newScene
	newScene.Root.cachedSceneRootNode = newScene.Root

	newScene.World = scene.World // Here, we simply reference the same world; we don't clone it, since a single world can be shared across multiple Scenes
	newScene.props = scene.props.Clone()

	newScene.updateAutobatch = true

	// Update sectors after cloning the scene
	models := newScene.Root.SearchTree().bySectors().Models()

	for _, n := range models {
		n.sector.Neighbors.Clear()
	}

	for _, n := range models {
		n.sector.UpdateNeighbors(models...)
	}

	for _, cam := range scene.View3DCameras {
		newScene.View3DCameras = append(newScene.View3DCameras, cam.Clone().(*Camera))
	}

	newScene.data = scene.data

	// Make a copy of the callbacks object
	cb := *scene.callbacks
	newScene.callbacks = &cb

	if runCallbacks {

		// We wait to call the onclone callbacks manually here because we want, for example, an "OnClone()" callback that triggers for a scene duplication
		// to trigger once all nodes have been cloned.
		if newScene.Callbacks().OnClone != nil {
			newScene.Callbacks().OnClone(newScene)
		}

		// Loop through and call OnClone for each Node that has it set in the tree.
		newScene.Root.SearchTree().ForEach(func(node INode) bool {
			if node.Callbacks().OnClone != nil {
				node.Callbacks().OnClone(node)
			}
			return true
		})

	}

	return newScene

}

// ID returns the unique ID for this specific scene. Each scene has its own ID.
func (s *Scene) ID() uint32 {
	return s.id
}

// SetName sets the name of the scene to the specified desired value.
func (s *Scene) SetName(name string) {
	s.name = name
}

// Name returns the name of the Scene.
func (s *Scene) Name() string {
	return s.name
}

// String returns the scene as a string in a human-readable format if ReadableReferences is on.
// Otherwise, it returns the pointer value, as usual.
func (scene *Scene) String() string {
	if ReadableReferences {
		return "Scene { Name: " + scene.name + ", ID: " + fmt.Sprintf("%d", scene.id) + " }"
	} else {
		return fmt.Sprintf("%p", scene)
	}
}

// Data returns the Scene's user-customizeable data.
func (scene *Scene) Data() any {
	return scene.data
}

// SetData sets the Scene's user-customizeable data pointer to whatever you specify (i.e. a backing "Level" instance or something, for example).
func (scene *Scene) SetData(data any) {
	scene.data = data
}

// Library returns the Library from which this Scene was loaded. If it was created through code and not associated with a Library, this function will return nil.
func (scene *Scene) Library() *Library {
	return scene.library
}

func (scene *Scene) Properties() Properties {
	return scene.props
}

var autobatchBlankMat = NewMaterial("autobatch null material")

func (scene *Scene) HandleAutobatch() {

	if scene.updateAutobatch {

		for _, node := range scene.Root.SearchTree().INodes() {

			if model, ok := node.(*Model); ok {

				if !model.autoBatched {

					mat := autobatchBlankMat

					if mats := model.Mesh.Materials(); len(mats) > 0 {
						mat = mats[0]
					}

					if model.AutoBatchMode == AutoBatchDynamic {

						if _, exists := scene.autobatchDynamicMap[mat]; !exists {
							mesh := NewMesh("auto dynamic batch")
							mesh.AddMeshPart(mat)
							m := NewModel("auto dynamic batch", mesh)
							m.FrustumCulling = false
							m.sectorType = SectorTypeStandalone
							scene.autobatchDynamicMap[mat] = m
							scene.Root.AddChildren(m)
						}
						scene.autobatchDynamicMap[mat].DynamicBatchAdd(scene.autobatchDynamicMap[mat].Mesh.MeshParts[0], model)

					} else if model.AutoBatchMode == AutoBatchStatic {

						if _, exists := scene.autobatchStaticMap[mat]; !exists {
							m := NewModel("auto static merge", NewMesh("auto static merge"))
							m.sectorType = SectorTypeStandalone
							scene.autobatchStaticMap[mat] = m
							scene.Root.AddChildren(scene.autobatchStaticMap[mat])
						}
						scene.autobatchStaticMap[mat].StaticMerge(model)
						if len(scene.autobatchStaticMap[mat].Mesh.VertexColors) > 0 {
							scene.autobatchStaticMap[mat].Mesh.VertexActiveColorChannel = 0
						}

					}

					model.autoBatched = true

				}

			}

		}

		for _, dyn := range scene.autobatchDynamicMap {

			for _, models := range dyn.DynamicBatchModels {

				modelList := append(make([]*Model, 0, len(models)), models...)

				for _, model := range modelList {

					if model.Root() == nil {
						dyn.DynamicBatchRemove(model)
					}

				}

			}

		}

		scene.updateAutobatch = false

	}

}

// Get searches a node's hierarchy using a string to find a specified node. The path is in the format of names of nodes, separated by forward
// slashes ('/'), and is relative to the node you use to call Get. As an example of Get, if you had a cup parented to a desk, which was
// parented to a room, that was finally parented to the root of the scene, it would be found at "Room/Desk/Cup". Note also that you can use "../" to
// "go up one" in the hierarchy (so cup.Get("../") would return the Desk node).
// Since Get uses forward slashes as path separation, it would be good to avoid using forward slashes in your Node names. Also note that Get()
// trims the extra spaces from the beginning and end of Node Names, so avoid using spaces at the beginning or end of your Nodes' names.
// Syntactic sugar for Scene.Root.Get().
func (scene *Scene) Get(nodePath string) INode {
	return scene.Root.Get(nodePath)
}

// FindNode searches through a Node's tree for the node by name exactly. This is mostly syntactic sugar for
// Node.SearchTree().ByName(nodeName).First().
func (scene *Scene) FindNode(nodeName string) INode {
	return scene.Root.SearchTree().ByName(nodeName).First()
}

func (scene *Scene) Callbacks() *SceneCallbacks {
	return scene.callbacks
}
