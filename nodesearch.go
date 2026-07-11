package tetra3d

import (
	"strings"
)

type NodeIterator interface {
	ForEach(forEach func(node INode, index int) bool)
}

type NodeList []INode

func (n NodeList) ForEach(forEach func(node INode, index int) bool) {
	for i, node := range n {
		if !forEach(node, i) {
			return
		}
	}
}

func (n *NodeList) Combine(other NodeIterator) {
	other.ForEach(func(node INode, index int) bool {
		n.Add(node)
		return true
	})
}

func (n NodeList) Count() int {
	return len(n)
}

func (n NodeList) Capacity() int {
	return cap(n)
}

func (n *NodeList) Add(node INode) {
	(*n) = append((*n), node)
}

// Returns the element in the NodeSlice at the given index.
// The index loops around the bounds of the slice, such that NodeSlice.Get(-1)
// returns the last element in the slice.
func (n NodeList) Get(index int) INode {
	if len(n) == 0 {
		return nil
	}

	return n[index%len(n)]
}

func (n NodeList) First() INode {
	if len(n) == 0 {
		return nil
	}
	return n[0]
}

func (n NodeList) Last() INode {
	if len(n) == 0 {
		return nil
	}
	return n[len(n)-1]
}

func (n *NodeList) Clear() {
	// clear(*n)
	*n = (*n)[:0]
}

// Represents a set of options to control searching through a hierarchy of nodes recursively, evaluating filters to find
// nodes that fit criteria, and do so lazily (e.g. one at a time as necessary).
// The object can also process the results and output them to a NodeSlice (a slice of Nodes).
// Note that the filtering options do stack (e.g. if you specify `startingNode.Search().ByNames("Player").ByPropPair("hunger", 100)“,
// a node would have to be a descendant of `startingNode`, be named "Player", and have a property named "hunger" that has a value of 100
// to pass the filters and be considered "found").
type SearchOptions struct {
	Start          INode
	noStartingNode bool
	filterStack    func(node INode) bool
}

func (s SearchOptions) ByNames(names ...string) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		add := false
		name := node.Name()
		for _, n := range names {
			if name == n {
				add = true
				break
			}
		}
		if add {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByNamesPartial(names ...string) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		add := false
		name := node.Name()
		for _, n := range names {
			if strings.Contains(name, n) {
				add = true
				break
			}
		}
		if add {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByParentNames(names ...string) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		add := false
		if node.Parent() != nil {
			name := node.Parent().Name()
			for _, n := range names {
				if name == n {
					add = true
					break
				}
			}
			if add {
				if last != nil {
					return last(node)
				} else {
					return true
				}
			}

		}
		return false
	}
	return s
}

func (s SearchOptions) ByParentNamesPartial(names ...string) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		add := false
		if node.Parent() != nil {
			name := node.Parent().Name()
			for _, n := range names {
				if strings.Contains(name, n) {
					add = true
					break
				}
			}
			if add {
				if last != nil {
					return last(node)
				} else {
					return true
				}
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByPropNames(names ...string) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		add := false
		props := node.Properties()
		for _, n := range names {
			if props.Has(n) {
				add = true
				break
			}
		}
		if add {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByParentPropNames(names ...string) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		add := false
		if node.Parent() != nil {
			props := node.Parent().Properties()
			for _, n := range names {
				if props.Has(n) {
					add = true
					break
				}
			}
			if add {
				if last != nil {
					return last(node)
				} else {
					return true
				}
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByPropValues(values ...any) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		add := false
		props := node.Properties()
		for _, value := range values {
			for _, prop := range props.data {
				if prop.Value == value {
					add = true
					break
				}
			}
			if add {
				break
			}
		}
		if add {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByParentPropValues(values ...any) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		add := false

		if parent := node.Parent(); parent != nil {

			props := parent.Properties()
			for _, value := range values {
				for _, prop := range props.data {
					if prop.Value == value {
						add = true
						break
					}
				}
				if add {
					break
				}
			}

		}

		if add {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByPropPair(name string, value any) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		if prop := node.Properties().GetByName(name); prop != nil && prop.Value == value {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByParentPropPair(name string, value any) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		if node.Parent() != nil {
			if prop := node.Parent().Properties().GetByName(name); prop != nil && prop.Value == value {
				if last != nil {
					return last(node)
				} else {
					return true
				}
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByType(nt NodeType) SearchOptions {
	last := s.filterStack

	s.filterStack = func(node INode) bool {
		if node.Type().Is(nt) {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByParentType(nt NodeType) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		if node.Parent() != nil && node.Parent().Type().Is(nt) {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByPropHasBitfieldValue(bitValue Bitfield) SearchOptions {
	last := s.filterStack

	s.filterStack = func(node INode) bool {
		for _, prop := range node.Properties().data {

			if prop.IsBitfield() && prop.AsBitfield().Contains(bitValue) {
				if last != nil {
					return last(node)
				} else {
					return true
				}
			}

		}
		return false
	}
	return s
}

func (s SearchOptions) ByParentPropHasBitfieldValue(bitValue Bitfield) SearchOptions {
	last := s.filterStack

	s.filterStack = func(node INode) bool {
		if parent := node.Parent(); parent != nil {
			for _, prop := range parent.Properties().data {

				if prop.IsBitfield() && prop.AsBitfield().Contains(bitValue) {
					if last != nil {
						return last(node)
					} else {
						return true
					}
				}

			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByPropHasBitfieldValueByName(bitfieldValueName string) SearchOptions {
	last := s.filterStack

	bitfieldValue := Bitfield(0)

	s.filterStack = func(node INode) bool {

		if bitfieldValue == 0 {
			if scene := node.Scene(); scene != nil {
				bitfieldValue = scene.BitfieldValueByName(bitfieldValueName)
			}
		}

		for _, prop := range node.Properties().data {
			if prop.IsBitfield() {
				if scene := node.Scene(); scene != nil && prop.AsBitfield().Contains(bitfieldValue) {
					if last != nil {
						return last(node)
					} else {
						return true
					}
				}
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByParentPropHasBitfieldValueByName(bitfieldValueName string) SearchOptions {
	last := s.filterStack

	bitfieldValue := Bitfield(0)

	s.filterStack = func(node INode) bool {

		if bitfieldValue == 0 {
			if scene := node.Scene(); scene != nil {
				bitfieldValue = scene.BitfieldValueByName(bitfieldValueName)
			}
		}

		if parent := node.Parent(); parent != nil {
			for _, prop := range parent.Properties().data {
				if prop.IsBitfield() {
					if scene := node.Scene(); scene != nil && prop.AsBitfield().Contains(bitfieldValue) {
						if last != nil {
							return last(node)
						} else {
							return true
						}
					}
				}
			}
		}
		return false
	}
	return s
}

func (s SearchOptions) ByCustomFunc(customFunc func(INode) bool) SearchOptions {
	if customFunc == nil {
		return s
	}
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		if customFunc(node) {
			if last != nil {
				return last(node)
			} else {
				return true
			}
		}
		return false
	}
	return s
}

// Returns a copy of SearchOptions with the specified Nodes included as being precluded from testing and searching.
// Their descendants still are, though.
func (s SearchOptions) NotNode(node INode) SearchOptions {
	last := s.filterStack
	s.filterStack = func(n INode) bool {
		if node == n {
			return false
		}

		if last != nil {
			return last(node)
		}
		return true
	}
	return s
}

// Returns a copy of SearchOptions with the specified Nodes included as being precluded from testing and searching.
// Their descendants still are, though.
func (s SearchOptions) NotNodes(nodes NodeIterator) SearchOptions {
	last := s.filterStack
	s.filterStack = func(node INode) bool {
		found := false
		nodes.ForEach(func(n INode, index int) bool {
			if node == n {
				found = true
				return false
			}
			return true
		})

		if found {
			return false
		}

		if last != nil {
			return last(node)
		}
		return true
	}
	return s
}

// Returns a copy of SearchOptions with the option to preclude the starting node from being included in the iteration process.
// Its descendants still are, though.
func (s SearchOptions) WithStartingNode(withStartingNode bool) SearchOptions {
	s.noStartingNode = !withStartingNode
	return s
}

// Evaluates and returns all Nodes that pass the filter set in the form of a NodeList.
// If a NodeList `out` is passed in, it will be cleared and reused for this purpose to avoid memory allocation.
// Otherwise, a new one will be allocated for this purpose.
// If no nodes pass the filter set, the function returns a nil NodeList (slice).
func (s SearchOptions) GetNodeList() NodeList {
	var out = NodeList{}
	s.ForEach(func(node INode, index int) bool {
		out = append(out, node)
		return true
	})
	return out
}

// Evaluates and returns the first Node that passes the filter set.
// If no nodes pass the filter set, the function returns nil.
func (s SearchOptions) GetFirst() INode {
	var out INode
	s.ForEach(func(node INode, index int) bool {
		out = node
		return false
	})
	return out
}

// Evaluates and returns the first Node that passes the filter set.
// If no nodes pass the filter set, the function returns nil.
func (s SearchOptions) Get(index int) INode {
	var out INode
	i := 0
	s.ForEach(func(node INode, forEachIndex int) bool {
		if i == index {
			out = node
			return false
		}
		i++
		return true
	})
	return out
}

// Evaluates and returns the last Node that passes the filter set.
// If no nodes pass the filter set, the function returns nil.
func (s SearchOptions) GetLast() INode {
	var out INode
	s.ForEach(func(node INode, index int) bool {
		out = node
		return true
	})
	return out
}

// Evaluates and returns the number of Nodes that pass the filter set.
// If no nodes pass the filter set, the function returns 0.
func (s SearchOptions) GetCount() int {
	count := 0
	s.ForEach(func(node INode, index int) bool {
		count++
		return true
	})
	return count
}

// Loops through each node starting with the root node specified when creating the SearchOptions
// and runs the specified function on the hierarchy if the node has passed the filter options.
// If the function returns true, the next Node in the hierarchy is returned; if not, the loop is ended prematurely.
func (s SearchOptions) ForEach(forEach func(node INode, index int) bool) {

	if forEach == nil {
		return
	}

	index := 0

	if !s.noStartingNode {
		if s.filterStack == nil || s.filterStack(s.Start) {
			if !forEach(s.Start, index) {
				return
			}
			index++
		}
	}

	s.Start.ForEachChild(true, func(node INode, index int) bool {
		if s.filterStack != nil && !s.filterStack(node) {
			return true
		}
		if !forEach(node, index) {
			return false
		}
		index++
		return true
	})

}

// Creates a SearchOptions object starting with the given Node.
// You create a SearchOptions object with Node.Search(), specify filters like ByNames(), and then iterate over all
// passing Nodes with SearchOptions.ForEach, or get a slice with GetSlice(). You can also get just the first or last
// with GetFirst() and GetLast().
func (n *Node) Search() SearchOptions {
	return SearchOptions{
		Start: n,
		// filterStack: func(node INode) bool { return true},
	}
}
