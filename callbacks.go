package tetra3d

// When set to false, callbacks don't run; this is to ensure callbacks are run at the right time if multiple callbacks may run at a time.
var runCallbacks = true

// NodeCallbacks represents a set of callbacks to be called when a Node is reparented, cloned, etc.
type NodeCallbacks struct {
	OnReparent func(node, oldParent, newParent INode) // A callback to be called whenever a Node is reparented.
	OnClone    func(newNode INode)                    // A callback to be called whenever a Node is cloned (including when its owning Scene is cloned).
	// TODO?: Maybe add OnSceneTreeChange(), where a function is called for each child in a tree when an ancestor is reparented?
	// I'm hesitant to add this without running into the use-case myself, though.
}

type SceneCallbacks struct {
	OnClone func(newScene *Scene) // A callback to be called whenever a Scene is cloned, after all Nodes have been cloned and reparented.
}
