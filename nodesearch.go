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

// Represents a set of options to control searching through a hierarchy of nodes recursively.
// Note that the options do stack (e.g. if you specify SearchOptions such that nodes must have a given name and a given property name,
// a node has to have both to pass).
type SearchOptions struct {
	Names       []string
	NamesParent []string

	PartialNames       []string
	PartialNamesParent []string

	PropNames       []string
	ParentPropNames []string

	PropValues       map[string]any
	ParentPropValues map[string]any

	NodeTypes       []NodeType
	NodeTypesParent []NodeType

	CustomFunc         func(node INode) bool // If the function is set, each node is run through the custom function; if it returns true, the node is added to the collection set
	WithoutNodes       *NodeCollectionSet
	ForEach            func(node INode) bool // If set, for each Node that passes the function set, ForEach is run on it. If it returns false, the loop is broken out of early
	NoReturnCollection bool                  // If NoReturnCollection is set, FindNodes returns an empty Collection Set (can be used in connection with ForEach to instead loop through all passing Nodes).
	NoRoot             bool                  // If NoRoot is set, the starting node is not included in the search (but its children and descendants can still be).
}

// Filters out the search options to find Nodes with any of the names specified.
func (f SearchOptions) ByNames(names ...string) SearchOptions {
	f.Names = names
	return f
}

// Filters out the search options to find children of Nodes with any of the names specified.
func (f SearchOptions) ByNamesParent(names ...string) SearchOptions {
	f.NamesParent = names
	return f
}

// Filters out the search options to find Nodes with a part of any of the names specified.
func (f SearchOptions) ByNamesPartial(partialNames ...string) SearchOptions {
	f.PartialNames = partialNames
	return f
}

// Filters out the search options to find children of Nodes with a part of any of the names specified.
func (f SearchOptions) ByNamesPartialParent(partialNames ...string) SearchOptions {
	f.PartialNamesParent = partialNames
	return f
}

// Filters out the search options to find Nodes with properties by any of the names specified.
func (f SearchOptions) ByPropNames(propNames ...string) SearchOptions {
	f.PropNames = propNames
	return f
}

// Filters out the search options to find children of Nodes with properties by any of the names specified.
func (f SearchOptions) ByPropNamesParent(propNames ...string) SearchOptions {
	f.ParentPropNames = propNames
	return f
}

// Filters out the search options to find Nodes with a property with the name specified set to the specified value.
func (f SearchOptions) ByPropValuePair(propName string, value any) SearchOptions {
	f.PropValues = map[string]any{propName: value}
	return f
}

// Filters out the search options to find children of Nodes with a property with the name specified set to the specified value.
func (f SearchOptions) ByPropValuePairParent(propName string, value any) SearchOptions {
	f.ParentPropValues = map[string]any{propName: value}
	return f
}

// Filters out the search options to find Nodes with properties with the names and values specified as keys and values in the given propPairs map.
func (f SearchOptions) ByPropValuePairs(propPairs map[string]any) SearchOptions {
	f.PropValues = propPairs
	return f
}

// Filters out the search options to find children of Nodes with properties with the names and values specified as keys and values in the given propPairs map.
func (f SearchOptions) ByPropValuePairsParent(propPairs map[string]any) SearchOptions {
	f.ParentPropValues = propPairs
	return f
}

// Filters out the search options to find Nodes of any of the given types.
func (f SearchOptions) ByType(withNodeTypes ...NodeType) SearchOptions {
	f.NodeTypes = withNodeTypes
	return f
}

// Filters out the search options to find Nodes of any of the given types.
func (f SearchOptions) ByTypeParent(withNodeTypes ...NodeType) SearchOptions {
	f.NodeTypesParent = withNodeTypes
	return f
}

// Sets the search options to run a custom function on each Node - if it passes, then the Node is added to the resulting collection set.
func (f SearchOptions) ByCustomFunc(customFunc func(node INode) bool) SearchOptions {
	f.CustomFunc = customFunc
	return f
}

// Removes nodes from consideration from the Search.
func (f SearchOptions) Not(nodes *NodeCollectionSet) SearchOptions {
	f.WithoutNodes = nodes
	return f
}

// Searches through a Node's hierarchical tree for any nodes that fulfill the given
// search parameters (including the node's start).
func (node *Node) Search(options SearchOptions) *NodeCollectionSet {
	output := NewNodeCollection()

	check := func(node INode, index, size int) bool {

		add := true

		if options.WithoutNodes != nil && options.WithoutNodes.Contains(node) {
			add = false
		} else {

			if options.Names != nil {
				result := false
				for _, n := range options.Names {
					if node.Name() == n {
						result = true
						break
					}
				}
				add = add && result
			}

			if options.NamesParent != nil {
				result := false
				if parent := node.Parent(); parent != nil {

					for _, n := range options.NamesParent {
						if parent.Name() == n {
							result = true
							break
						}
					}

				}
				add = add && result

			}

			if options.PartialNames != nil {
				result := false
				for _, n := range options.PartialNames {
					if strings.Contains(node.Name(), n) {
						result = true
						break
					}
				}
				add = add && result
			}

			if options.PartialNamesParent != nil {
				result := false
				if parent := node.Parent(); parent != nil {

					for _, n := range options.PartialNamesParent {
						if strings.Contains(parent.Name(), n) {
							result = true
							break
						}
					}

				}
				add = add && result
			}

			if options.PropNames != nil {
				result := false
				for _, n := range options.PropNames {
					if node.Properties().Has(n) {
						result = true
						break
					}
				}
				add = add && result
			}

			if options.ParentPropNames != nil {
				result := false
				if parent := node.Parent(); parent != nil {

					for _, n := range options.ParentPropNames {
						if parent.Properties().Has(n) {
							result = true
							break
						}
					}

				}
				add = add && result
			}

			if options.PropValues != nil {
				result := false
				for k, v := range options.PropValues {
					if res := node.Properties().Get(k); res != nil && res.Value == v {
						result = true
						break
					}
				}
				add = add && result
			}

			if options.ParentPropValues != nil {
				result := false
				if parent := node.Parent(); parent != nil {
					for k, v := range options.ParentPropValues {

						if res := parent.Properties().Get(k); res != nil && res.Value == v {
							result = true
							break
						}

					}
				}
				add = add && result
			}

			if options.NodeTypes != nil {
				result := false
				for _, nt := range options.NodeTypes {
					if node.Type().Is(nt) {
						result = true
						break
					}
				}
				add = add && result
			}

			if options.NodeTypesParent != nil {
				result := false
				if parent := node.Parent(); parent != nil {

					for _, nt := range options.NodeTypesParent {
						if parent.Type().Is(nt) {
							result = true
							break
						}
					}

				}
				add = add && result
			}

			if options.CustomFunc != nil {
				result := false
				if options.CustomFunc(node) {
					result = true
				}
				add = add && result
			}

		}

		// If no filters are set, then the function just returns each Node's children recursively.
		if add {

			if options.ForEach != nil {
				if !options.ForEach(node) {
					return false
				}
			}

			if options.NoReturnCollection {
				return true
			}

			output.Add(node)

		}

		return true

	}

	if !options.NoRoot {
		check(node, 0, 1)
	}

	node.ForEachChild(true, check)

	return output
}
