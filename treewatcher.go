package tetra3d

// TreeWatcher is a utility struct used to watch a tree for hierarchy changes underneath a specific node. This is useful when, for example,
// adding game logic code for objects added into a scene.
type TreeWatcher struct {
	rootNode     INode
	elements     []INode
	prevElements []INode
	// WatchFilter is a function that is used to filter down which nodes to watch, and is called for each object in the scene tree.
	// If the function returns true, the Node is watched.
	WatchFilter func(node INode) bool
	OnChange    func(node INode) // OnChange is a function that is run for every altered node under a tree's root node whenever anything in that hierarchy is added or removed from the tree.
}

// NewTreeWatcher creates a new TreeWatcher, a utility that watches the scene tree underneath the rootNode for changes.
// onChange should be the function that is run for each node when it is added to, or removed from, the tree underneath the rootNode.
func NewTreeWatcher(rootNode INode, onChange func(node INode)) *TreeWatcher {
	treeWatcher := &TreeWatcher{
		rootNode: rootNode,
		OnChange: onChange,
	}
	return treeWatcher
}

// Update updates the TreeWatcher instance, and should be run once every game frame.
func (watch *TreeWatcher) Update() {

	if watch.rootNode != nil {

		watch.elements = make([]INode, 0, cap(watch.prevElements))

		watch.rootNode.SearchTree().ForEach(func(node INode) bool {
			if watch.WatchFilter == nil || watch.WatchFilter(node) {
				watch.elements = append(watch.elements, node)
			}
			return true
		})

		for _, e := range watch.elements {

			existedPreviously := false

			for _, p := range watch.prevElements {
				if e == p {
					existedPreviously = true
					break
				}
			}

			if !existedPreviously && watch.OnChange != nil {
				watch.OnChange(e)
			}

		}

		for _, e := range watch.prevElements {

			stillExists := false

			for _, p := range watch.elements {
				if e == p {
					stillExists = true
					break
				}
			}

			if !stillExists && watch.OnChange != nil {
				watch.OnChange(e)
			}

		}

		watch.prevElements = watch.elements

	}

}

// SetRoot sets the root Node to be watched for the TreeWatcher.
func (watch *TreeWatcher) SetRoot(rootNode INode) {
	watch.rootNode = rootNode
}
