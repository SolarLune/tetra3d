package tetra3d

// Callbacks represents a set of callbacks to be called when a Node is reparented, cloned, etc.
type Callbacks struct {
	OnReparent func(node, oldParent, newParent INode) // A callback to be called whenever a Node is reparented.
	OnClone    func(newNode INode)                    // A callback to be called whenever a Node is cloned (including when its owning Scene is cloned).
	// TODO?: Maybe add OnSceneTreeChange(), where a function is called for each child in a tree when an ancestor is reparented?
	// I'm hesitant to add this without running into the use-case myself, though.
}
