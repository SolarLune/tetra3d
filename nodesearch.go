package tetra3d

import (
	"strings"
)

type NodeCollectionSet struct {
	OrderedSet[INode]
}

func NewNodeCollection(nodes ...INode) *NodeCollectionSet {
	set := &NodeCollectionSet{
		OrderedSet: newOrderedSet[INode](),
	}
	set.Add(nodes...)
	return set
}

func (n *NodeCollectionSet) Clone() *NodeCollectionSet {
	newOrderedSet := NewNodeCollection()
	return newOrderedSet.Combine(n)
}

func (n *NodeCollectionSet) Combine(others ...*NodeCollectionSet) *NodeCollectionSet {
	for _, o := range others {
		n.OrderedSet.Combine(o.OrderedSet)
	}
	return n
}

func (n *NodeCollectionSet) First() INode {
	if len(n.OrderedSet) > 0 {
		return n.OrderedSet[0]
	}
	return nil
}

func (n *NodeCollectionSet) Last() INode {
	if len(n.OrderedSet) > 0 {
		return n.OrderedSet[len(n.OrderedSet)-1]
	}
	return nil
}

func (n *NodeCollectionSet) Get(index int) INode {
	if len(n.OrderedSet) > 0 {
		for index < 0 {
			index += len(n.OrderedSet)
		}
		for index >= len(n.OrderedSet) {
			index -= len(n.OrderedSet)
		}
		return n.OrderedSet[index]
	}
	return nil
}

func (n *NodeCollectionSet) Count() int {
	return len(n.OrderedSet)
}

func (n *NodeCollectionSet) ForEach(forEach func(node INode) bool) {
	for _, node := range n.OrderedSet {
		if !forEach(node) {
			break
		}
	}
}

func (n *NodeCollectionSet) ForEachLight(forEach func(light ILight) bool) {
	for _, node := range n.OrderedSet {
		if light, ok := node.(ILight); ok {
			if !forEach(light) {
				break
			}
		}
	}
}

func (n *NodeCollectionSet) ForEachModel(forEach func(model *Model) bool) {
	for _, node := range n.OrderedSet {
		if light, ok := node.(*Model); ok {
			if !forEach(light) {
				break
			}
		}
	}
}

func (n *NodeCollectionSet) ForEachCamera(forEach func(camera *Camera) bool) {
	for _, node := range n.OrderedSet {
		if camera, ok := node.(*Camera); ok {
			if !forEach(camera) {
				break
			}
		}
	}
}

func (n *NodeCollectionSet) ForEachBoundingObject(forEach func(boundingObject IBoundingObject) bool) {
	for _, node := range n.OrderedSet {
		if boundingObject, ok := node.(IBoundingObject); ok {
			if !forEach(boundingObject) {
				break
			}
		}
	}
}

func (n NodeCollectionSet) ToLights() []ILight {
	out := []ILight{}
	n.ForEachLight(func(light ILight) bool {
		out = append(out, light)
		return true
	})
	return out
}

func (n NodeCollectionSet) ToModels() []*Model {
	out := []*Model{}
	n.ForEachModel(func(model *Model) bool {
		out = append(out, model)
		return true
	})
	return out
}

func (n NodeCollectionSet) ToNodes() []INode {
	out := []INode{}
	n.ForEach(func(node INode) bool {
		out = append(out, node)
		return true
	})
	return out
}

func (n NodeCollectionSet) ToCameras() []*Camera {
	out := []*Camera{}
	n.ForEachCamera(func(camera *Camera) bool {
		out = append(out, camera)
		return true
	})
	return out
}

type SearchOptions struct {
	Names              []string
	PartialNames       []string
	PropNames          []string
	CustomFunc         func(node INode) bool
	WithoutNodes       *NodeCollectionSet
	PropValues         map[string]any
	NodeTypes          []NodeType
	ReturnChildren     bool // When enabled, if a Node fulfills the options set, the children are returned rather than the Node itself
	ForEach            func(node INode) bool
	NoReturnCollection bool // If NoReturnCollection is set, FindNodes returns an empty Collection Set.
	NoRoot             bool // If NoRoot is set, the starting node is not included in the search (but its children and descendants are still).
}

func (f SearchOptions) ByNames(names ...string) SearchOptions {
	f.Names = names
	return f
}

func (f SearchOptions) ByNamesPartial(names ...string) SearchOptions {
	f.PartialNames = names
	return f
}

func (f SearchOptions) ByProps(propNames ...string) SearchOptions {
	f.PropNames = propNames
	return f
}

// Syntactic sugar for SearchOptions.ByProp().WithReturnChildren(true)
func (f SearchOptions) ByParentProps(propNames ...string) SearchOptions {
	f.PropNames = propNames
	f.ReturnChildren = true
	return f
}

func (f SearchOptions) ByPropValues(propPairs map[string]any) SearchOptions {
	f.PropValues = propPairs
	return f
}

func (f SearchOptions) ByType(withNodeTypes ...NodeType) SearchOptions {
	f.NodeTypes = withNodeTypes
	return f
}

func (f SearchOptions) WithReturnChildren(withReturnChildren bool) SearchOptions {
	f.ReturnChildren = withReturnChildren
	return f
}

func (f SearchOptions) Not(nodes *NodeCollectionSet) SearchOptions {
	if f.WithoutNodes == nil {
		f.WithoutNodes = nodes
	}
	return f
}

// Searches through a Node's hierarchical tree for any nodes that fulfill the given
// search parameters (including the node's start).
func (node *Node) Search(options SearchOptions) *NodeCollectionSet {
	output := NewNodeCollection()

	check := func(node INode, index, size int) bool {

		add := false
		nothingSet := true

		if options.WithoutNodes != nil && options.WithoutNodes.Contains(node) {
			nothingSet = false
			add = false
		} else {

			if !add && options.Names != nil {
				nothingSet = false
				for _, n := range options.Names {
					if node.Name() == n {
						add = true
					}
				}
			}

			if !add && options.PartialNames != nil {
				nothingSet = false
				for _, n := range options.PartialNames {
					if strings.Contains(node.Name(), n) {
						add = true
					}
				}
			}

			if !add && options.PropNames != nil {
				nothingSet = false
				for _, n := range options.PropNames {
					if node.Properties().Has(n) {
						add = true
					}
				}
			}

			if !add && options.PropValues != nil {
				nothingSet = false
				for k, v := range options.PropValues {
					if res := node.Properties().Get(k); res != nil && res.Value == v {
						add = true
					}
				}
			}

			if !add && options.NodeTypes != nil {
				nothingSet = false
				for _, nt := range options.NodeTypes {
					if node.Type().Is(nt) {
						add = true
					}
				}
			}

			if !add && options.CustomFunc != nil {
				nothingSet = false
				if options.CustomFunc(node) {
					add = true
				}
			}

		}

		if nothingSet || add {

			if options.ForEach != nil {
				if !options.ForEach(node) {
					return false
				}
			}

			if options.NoReturnCollection {
				return true
			}

			if options.ReturnChildren {
				node.ForEachChild(false, func(child INode, index, size int) bool {
					output.Add(child)
					return true
				})
			} else {
				output.Add(node)
			}

		}

		return true

	}

	if !options.NoRoot {
		check(node, 0, 1)
	}

	node.ForEachChild(true, check)

	return output
}
