package tetra3d

import (
	"regexp"
	"sort"
)

const (
	nfSortModeNone = iota
	nfSortModeAxisX
	nfSortModeAxisY
	nfSortModeAxisZ
	nfSortModeDistance
)

// NodeFilter represents a chain of node filters, executed in sequence to collect the desired nodes
// out of an entire hierarchy. The filters are executed lazily (so only one slice is allocated
// in the process, and possibly one more for the end result, if you get the result as not just a
// slice of INodes).
type NodeFilter struct {
	Filters        []func(INode) bool // The slice of filters that are currently active on the NodeFilter.
	Start          INode              // The start (root) of the filter.
	StopOnFiltered bool               // If the filter should continue through to a node's children if the node itself doesn't pass the filter
	MaxDepth       int                // How deep the node filter should search in the starting node's hierarchy; a value that is less than zero means the entire tree will be traversed.
	depth          int

	sortMode    int
	reverseSort bool
	sortTo      Vector
}

func newNodeFilter(startingNode INode) *NodeFilter {
	return &NodeFilter{
		Start:    startingNode,
		depth:    -1,
		MaxDepth: -1,
	}
}

func (nf *NodeFilter) execute(node INode) []INode {
	nf.depth++
	out := []INode{}
	added := true
	if node != nf.Start {
		add := true
		for _, filter := range nf.Filters {
			if !filter(node) {
				add = false
				added = false
			}
		}
		if add {
			out = append(out, node)
		}
	}
	if nf.MaxDepth < 0 || nf.depth <= nf.MaxDepth {

		if !nf.StopOnFiltered || added {

			for _, child := range node.Children() {
				out = append(out, nf.execute(child)...)
			}

		}

	}

	nf.depth--

	// If the depth is at -1, then this should be the end
	if nf.depth == -1 && nf.sortMode != nfSortModeNone {

		switch nf.sortMode {
		case nfSortModeAxisX:
			sort.SliceStable(out, func(i, j int) bool {
				if nf.reverseSort {
					return out[i].WorldPosition().X > out[j].WorldPosition().X
				}
				return out[i].WorldPosition().X < out[j].WorldPosition().X
			})
		case nfSortModeAxisY:
			sort.SliceStable(out, func(i, j int) bool {
				if nf.reverseSort {
					return out[i].WorldPosition().Y > out[j].WorldPosition().Y
				}
				return out[i].WorldPosition().Y < out[j].WorldPosition().Y
			})
		case nfSortModeAxisZ:
			sort.SliceStable(out, func(i, j int) bool {
				if nf.reverseSort {
					return out[i].WorldPosition().Z > out[j].WorldPosition().Z
				}
				return out[i].WorldPosition().Z < out[j].WorldPosition().Z
			})
		case nfSortModeDistance:
			sort.SliceStable(out, func(i, j int) bool {
				if nf.reverseSort {
					return out[i].WorldPosition().DistanceSquared(nf.sortTo) > out[j].WorldPosition().DistanceSquared(nf.sortTo)
				}
				return out[i].WorldPosition().DistanceSquared(nf.sortTo) < out[j].WorldPosition().DistanceSquared(nf.sortTo)
			})
		}

		nf.sortMode = nfSortModeNone
		nf.sortTo = Vector{}
		nf.reverseSort = false

	}

	return out
}

func (nf *NodeFilter) executeFilters(node INode, execute func(INode)) {
	nf.depth++

	successfulFilter := true
	if node != nf.Start {
		ok := true
		for _, filter := range nf.Filters {
			if !filter(node) {
				ok = false
				successfulFilter = false
			}
		}
		if ok {
			execute(node)
		}
	}
	if nf.MaxDepth < 0 || nf.depth <= nf.MaxDepth {

		if !nf.StopOnFiltered || successfulFilter {

			for _, child := range node.Children() {
				nf.executeFilters(child, execute)
			}

		}

	}
	nf.depth--
}

// First returns the first Node in the NodeFilter; if the NodeFilter is empty, this function returns nil.
func (nf NodeFilter) First() INode {
	out := nf.execute(nf.Start)
	if len(out) == 0 {
		return nil
	}
	return out[0]
}

// First returns the last Node in the NodeFilter; if the NodeFilter is empty, this function returns nil.
func (nf NodeFilter) Last() INode {
	out := nf.execute(nf.Start)
	if len(out) == 0 {
		return nil
	}
	return out[len(out)-1]
}

// Get returns the Node at the given index in the NodeFilter; if index is invalid (<0 or >= len(nodes)), this function returns nil.
func (nf NodeFilter) Get(index int) INode {
	out := nf.execute(nf.Start)
	if index < 0 || index >= len(out) {
		return nil
	}
	return out[index]
}

// ByFunc allows you to filter a given selection of nodes by the provided filter function (which takes a Node
// and returns a boolean, indicating whether or not to add that Node to the resulting NodeFilter).
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByFunc(filterFunc func(node INode) bool) NodeFilter {
	nf.Filters = append(nf.Filters, filterFunc)
	return nf
}

// ByProps allows you to filter a given selection of nodes by the provided set of property names.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByProps(propNames ...string) NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool {
		return node.Properties().Has(propNames...)
	})
	return nf
}

// ByParentProps allows you to filter a given selection of nodes if the node has a parent with the provided
// set of property names.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByParentProps(propNames ...string) NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool {
		return node.Parent() != nil && node.Parent().Properties().Has(propNames...)
	})
	return nf
}

// ByName allows you to filter a given selection of nodes if their names are wholly equal
// to the provided name string.
// If a Node's name doesn't match, it isn't added to the filter results.
func (nf NodeFilter) ByName(name string) NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool { return node.Name() == name })
	return nf
}

// ByRegex allows you to filter a given selection of nodes by their namers using the given regex string.
// If you want to filter a selection of nodes in such a way that only nodes that have names that
// >contain< the given text (i.e. strings.Contains()) are selected for filtering, you can just pass
// that string into ByRegex directly.
// If the regexp string is invalid or no matching Nodes are found, the node isn't
// added to the filter results. See https://regexr.com/ for regex help / reference.
func (nf NodeFilter) ByRegex(regexString string) NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool {
		match, _ := regexp.MatchString(regexString, node.Name())
		return match
	})
	return nf
}

// ByType allows you to filter a given selection of nodes by the provided NodeType.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByType(nodeType NodeType) NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool {
		return node.Type().Is(nodeType)
	})
	return nf
}

// Not allows you to filter OUT a given NodeFilter of nodes.
// If a node is in the provided slice of INodes, then it will not be added to the
// final NodeFilter.
func (nf NodeFilter) Not(others ...INode) NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool {
		for _, other := range others {
			if node == other {
				return false
			}
		}
		return true
	})
	return nf
}

func (nf NodeFilter) bySectors() NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool {
		return node.Type() == NodeTypeModel && node.(*Model).sector != nil
	})
	return nf
}

// ForEach executes the provided function on each filtered Node, without allocating memory for a slice of the nodes.
func (nf NodeFilter) ForEach(f func(node INode)) {
	nf.executeFilters(nf.Start, f)
}

// Contains returns if the provided Node is contained in the NodeFilter.
func (nf NodeFilter) Contains(node INode) bool {
	return nf.Index(node) > 0
}

// Index returns the index of the given INode in the NodeFilter; if it doesn't exist in the filter,
// then this function returns -1.
func (nf NodeFilter) Index(node INode) int {
	out := nf.execute(nf.Start)
	for index, child := range out {
		if child == node {
			return index
		}
	}
	return -1
}

// Empty returns true if the NodeFilter contains no Nodes.
func (nf NodeFilter) Empty() bool {
	return len(nf.execute(nf.Start)) == 0
}

// INodes returns the NodeFilter's results as a slice of INodes.
func (nf NodeFilter) INodes() []INode {
	return nf.execute(nf.Start)
}

// IBoundingObjects returns a slice of the IBoundingObjects contained within the NodeFilter.
func (nf NodeFilter) IBoundingObjects() []IBoundingObject {
	out := nf.execute(nf.Start)
	boundings := make([]IBoundingObject, 0, len(out))
	for _, n := range out {
		if b, ok := n.(IBoundingObject); ok {
			boundings = append(boundings, b)
		}
	}
	return boundings
}

// IBoundingObjectsWithProps is a helper function that returns a slice of IBoundingObjects who have parents
// with the specified properties.
func (nf NodeFilter) IBoundingObjectsWithProps(props ...string) []IBoundingObject {
	return nf.ByParentProps(props...).IBoundingObjects()
}

// Models returns a slice of the Models contained within the NodeFilter.
func (nf NodeFilter) Models() []*Model {
	out := nf.execute(nf.Start)
	models := make([]*Model, 0, len(out))
	for _, n := range out {
		if m, ok := n.(*Model); ok {
			models = append(models, m)
		}
	}
	return models
}

// Lights returns a slice of the ILights contained within the NodeFilter.
func (nf NodeFilter) Lights() []ILight {
	out := nf.execute(nf.Start)
	lights := make([]ILight, 0, len(out))
	for _, n := range out {
		if m, ok := n.(ILight); ok {
			lights = append(lights, m)
		}
	}
	return lights
}

// Grids returns a slice of the Grid nodes contained within the NodeFilter.
func (nf NodeFilter) Grids() []*Grid {
	out := nf.execute(nf.Start)
	grids := make([]*Grid, 0, len(out))
	for _, n := range out {
		if m, ok := n.(*Grid); ok {
			grids = append(grids, m)
		}
	}
	return grids
}

// SortByX applies an X-axis sort on the results of the NodeFilter.
// Sorts do not combine.
func (nf NodeFilter) SortByX() NodeFilter {
	nf.sortMode = nfSortModeAxisX
	return nf
}

// SortByY applies an Y-axis sort on the results of the NodeFilter.
// Sorts do not combine.
func (nf NodeFilter) SortByY() NodeFilter {
	nf.sortMode = nfSortModeAxisX
	return nf
}

// SortByZ applies an Z-axis sort on the results of the NodeFilter.
// Sorts do not combine.
func (nf NodeFilter) SortByZ() NodeFilter {
	nf.sortMode = nfSortModeAxisZ
	return nf
}

// SortByDistance sorts the results of the NodeFilter by their distances to the specified INode.
// Sorts do not combine.
func (nf NodeFilter) SortByDistance(to Vector) NodeFilter {
	nf.sortMode = nfSortModeDistance
	nf.sortTo = to
	return nf
}

// SortReverse reverses any sorting performed on the NodeFilter.
func (nf NodeFilter) SortReverse() NodeFilter {
	nf.reverseSort = true
	return nf
}
