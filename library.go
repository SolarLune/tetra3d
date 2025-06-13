package tetra3d

// Library represents a collection of Scenes, Meshes, Animations, etc., as loaded from an intermediary file format (.dae or .gltf / .glb).
type Library struct {
	Scenes        []*Scene     // A slice of Scenes
	ExportedScene *Scene       // The scene that was open when the library was exported from the modeler
	Animations    []*Animation // A Map of Animations to their names
	Meshes        []*Mesh      // A selection of Meshes (because they can be cloned while having the same names)
	Materials     []*Material  // A selection of Materials (because they can be cloned while having the same names)
	Worlds        []*World     // A Map of Worlds to their names
}

// NewLibrary creates a new Library.
func NewLibrary() *Library {
	return &Library{
		Scenes:     []*Scene{},
		Meshes:     []*Mesh{},
		Animations: []*Animation{},
		Materials:  []*Material{},
		Worlds:     []*World{},
	}
}

// SceneByName searches all scenes in a Library to find the one with the provided name. If a scene with the given name isn't found,
// SceneByName will return nil.
func (lib *Library) SceneByName(name string) *Scene {
	for _, scene := range lib.Scenes {
		if scene.name == name {
			return scene
		}
	}
	return nil
}

func (lib *Library) SceneByID(id uint32) *Scene {
	for _, scene := range lib.Scenes {
		if scene.id == id {
			return scene
		}
	}
	return nil
}

func (lib *Library) AnimationByName(name string) *Animation {
	for _, anim := range lib.Animations {
		if anim.Name == name {
			return anim
		}
	}
	return nil
}

func (lib *Library) AnimationByID(id uint32) *Animation {
	for _, anim := range lib.Animations {
		if anim.id == id {
			return anim
		}
	}
	return nil
}

func (lib *Library) WorldByName(name string) *World {
	for _, world := range lib.Worlds {
		if world.Name == name {
			return world
		}
	}
	return nil
}

func (lib *Library) WorldByID(id uint32) *World {
	for _, world := range lib.Worlds {
		if world.id == id {
			return world
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

func (lib *Library) NodeByID(id uint32) INode {
	for _, scene := range lib.Scenes {
		if n := scene.Root.SearchTree().ByID(id).First(); n != nil {
			return n
		}
	}
	return nil
}

// NodeByName allows you to find a node by name by searching through each of a Library's scenes. If the Node with the given name isn't found,
// NodeByName will return nil.
func (lib *Library) NodeByName(objectName string) INode {
	for _, scene := range lib.Scenes {
		if n := scene.Root.SearchTree().ByName(objectName).First(); n != nil {
			return n
		}
	}
	return nil
}

func (lib *Library) MeshByID(id uint32) *Mesh {
	for _, mesh := range lib.Meshes {
		if mesh.ID == id {
			return mesh
		}
	}
	return nil
}

func (lib *Library) MeshByName(name string) *Mesh {
	for _, mesh := range lib.Meshes {
		if mesh.Name == name {
			return mesh
		}
	}
	return nil
}

func (lib *Library) MaterialByID(id uint32) *Material {
	for _, mat := range lib.Materials {
		if mat.ID() == id {
			return mat
		}
	}
	return nil
}

func (lib *Library) MaterialByName(name string) *Material {
	for _, mat := range lib.Materials {
		if mat.Name() == name {
			return mat
		}
	}
	return nil
}
