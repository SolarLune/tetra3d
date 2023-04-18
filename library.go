package tetra3d

// Library represents a collection of Scenes, Meshes, Animations, etc., as loaded from an intermediary file format (.dae or .gltf / .glb).
type Library struct {
	Scenes        []*Scene              // A slice of Scenes
	ExportedScene *Scene                // The scene that was open when the library was exported from the modeler
	Meshes        map[string]*Mesh      // A Map of Meshes to their names
	Animations    map[string]*Animation // A Map of Animations to their names
	Materials     map[string]*Material  // A Map of Materials to their names
	Worlds        map[string]*World     // A Map of Worlds to their names
	Categories    []string
}

// NewLibrary creates a new Library.
func NewLibrary() *Library {
	return &Library{
		Scenes:     []*Scene{},
		Meshes:     map[string]*Mesh{},
		Animations: map[string]*Animation{},
		Materials:  map[string]*Material{},
		Worlds:     map[string]*World{},
		Categories: []string{},
	}
}

// FindScene searches all scenes in a Library to find the one with the provided name. If a scene with the given name isn't found,
// FindScene will return nil.
func (lib *Library) FindScene(name string) *Scene {
	for _, scene := range lib.Scenes {
		if scene.Name == name {
			return scene
		}
	}
	return nil
}

func (lib *Library) AddScene(sceneName string) *Scene {
	newScene := NewScene(sceneName)
	newScene.library = lib
	lib.Scenes = append(lib.Scenes, newScene)
	return newScene
}

// FindNode allows you to find a node by name by searching through each of a Library's scenes. If the Node with the given name isn't found,
// FindNode will return nil.
func (lib *Library) FindNode(objectName string) INode {
	for _, scene := range lib.Scenes {
		if n := scene.Root.SearchTree().ByName(objectName).First(); n != nil {
			return n
		}
	}
	return nil
}
