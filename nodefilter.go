package tetra3d

import (
	"log"
	"math/rand"
	"regexp"
	"sort"
	"sync"
	"time"
)

var localRandom = rand.New(rand.NewSource(time.Now().Unix()))

const (
	nfSortModeNone = iota
	nfSortModeAxisX
	nfSortModeAxisY
	nfSortModeAxisZ
	nfSortModeDistance
	nfSortModeRandom
)

// NodeFilter represents a chain of node filters, executed in sequence to collect the desired nodes
// out of an entire hierarchy. The filters are executed lazily (so only one slice is allocated
// in the process, and possibly one more for the end result, if you get the result as not just a
// slice of INodes).
type NodeFilter struct {
	Filters        []func(INode) bool // The slice of filters that are currently active on the NodeFilter.
	Start          INode              // The start (root) of the filter.
	stopOnFiltered bool               // If the filter should continue through to a node's children if the node itself doesn't pass the filter
	MaxDepth       int                // How deep the node filter should search in the starting node's hierarchy; a value that is less than zero means the entire tree will be traversed.
	depth          int
	sortMode       int
	reverseSort    bool
	sortTo         Vector

	multithreadingWG *sync.WaitGroup
}

func newNodeFilter(startingNode INode) *NodeFilter {
	return &NodeFilter{
		Start:            startingNode,
		depth:            -1,
		MaxDepth:         -1,
		multithreadingWG: &sync.WaitGroup{},
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

		if !nf.stopOnFiltered || added {

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
		case nfSortModeRandom:
			localRandom.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
		}

		nf.sortMode = nfSortModeNone
		nf.sortTo = Vector{}
		nf.reverseSort = false

	}

	return out
}

func (nf *NodeFilter) executeFilters(node INode, execute func(INode) bool, multithreading bool) bool {

	// TODO: Append filtered nodes to an internal shared slice so sorting could be available
	// while also not allocating memory constantly unnecessarily.

	if nf.depth < 0 && nf.sortMode != nfSortModeNone {
		log.Printf("Warning: NodeFilter executing on Node < %s > has sorting on it, but ForEach() cannot be used with sorting.\n", nf.Start.Path())
	}

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
			if multithreading {

				nf.multithreadingWG.Add(1)
				go func() {
					defer nf.multithreadingWG.Done()
					execute(node)
				}()

			} else {

				if !execute(node) {
					nf.depth--
					return false
				}

			}
		}
	}

	if nf.MaxDepth < 0 || nf.depth <= nf.MaxDepth {

		if !nf.stopOnFiltered || successfulFilter {

			includeChild := true
			node.ForEachChild(func(child INode) bool {
				if !nf.executeFilters(child, execute, multithreading) {
					includeChild = false
					nf.depth--
					return false
				}
				return true
			})

			if !includeChild {
				return false
			}

		}

	}
	nf.depth--

	if multithreading && node == nf.Start {
		nf.multithreadingWG.Wait()
	}

	return true
}

// First returns the first Node in the NodeFilter; if the NodeFilter is empty, this function returns nil.
func (nf NodeFilter) First() INode {
	var result INode
	nf.ForEach(func(node INode) bool { result = node; return false })
	return result
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

// ByPropNameRegex allows you to filter a given selection of nodes by the provided set of property names,
// evaluated as regex strings.
// If the Nodes in the filter have any property that satisfy one of the supplied regex string, they pass
// the filter and are included.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByPropNameRegex(propNameRegexStrings ...string) NodeFilter {

	nf.Filters = append(nf.Filters, func(node INode) bool {

		for _, propName := range propNameRegexStrings {

			for existingName := range node.Properties() {
				match, _ := regexp.MatchString(propName, existingName)
				if match {
					return true
				}
			}

		}
		return false

	})

	return nf

}

// ByProp allows you to filter a given selection of nodes by a property value check - if the nodes filtered
// have a property with the given value, they are included.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByProp(propName string, propValue any) NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool {
		value := node.Properties().Get(propName)
		return value != nil && value.Value == propValue
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

// ByRegex allows you to filter a given selection of nodes by their names using the given regex string.
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

// StopOnFiltered allows you to specify that if a node doesn't pass a filter, none of its children will pass as well.
func (nf NodeFilter) StopOnFiltered() NodeFilter {
	nf.stopOnFiltered = true
	return nf
}

func (nf NodeFilter) bySectors() NodeFilter {
	nf.Filters = append(nf.Filters, func(node INode) bool {
		return node.Type() == NodeTypeModel && node.(*Model).sector != nil
	})
	return nf
}

// ForEach executes the provided function on each filtered Node.
// This is done to avoid allocating a slice for the filtered Nodes.
// The function must return a boolean indicating whether to continue running on each node in the tree that fulfills the
// filter set (true) or not (false).
// ForEach does not work with any sorting, and will log a warning if you use sorting and ForEach on the same filter.
func (nf NodeFilter) ForEach(callback func(node INode) bool) {
	nf.executeFilters(nf.Start, callback, false)
}

// ForEachMultithreaded executes the provided function on each filtered Node.
// This is done to avoid allocating a slice for the filtered Nodes.
// ForEachMultithreaded will filter each node on the main thread, but will execute the callback function for each node
// on varying goroutines, syncing them all up once finished to resume execution on the main thread.
// You should be careful to avoid race conditions when using this function.
func (nf NodeFilter) ForEachMultithreaded(callback func(node INode)) {
	nf.executeFilters(nf.Start, func(i INode) bool { callback(i); return true }, true)
}

// Count returns the number of Nodes that fit the filter set.
func (nf NodeFilter) Count() int {
	count := 0
	nf.executeFilters(nf.Start, func(i INode) bool {
		count++
		return true
	}, false)
	return count
}

// Contains returns if the provided Node is contained in the NodeFilter.
func (nf NodeFilter) Contains(node INode) bool {
	return nf.Index(node) > 0
}

// Index returns the index of the given INode in the NodeFilter assuming the filter executes and results
// in an []INode. If it doesn't exist in the filter, then this function returns -1.
func (nf NodeFilter) Index(node INode) int {
	out := nf.execute(nf.Start)
	for index, child := range out {
		if child == node {
			return index
		}
	}
	return -1
}

// IsEmpty returns true if the NodeFilter contains no Nodes.
func (nf NodeFilter) IsEmpty() bool {
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
	nf.sortMode = nfSortModeAxisY
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

// SortReverse randomizes the elements returned by the NodeFilter finishing functions.
// Sorts do not combine.
func (nf NodeFilter) SortRandom() NodeFilter {
	nf.sortMode = nfSortModeRandom
	return nf
}

// SetDepth sets the maximum search depth of the NodeFilter to the value provided.
func (nf NodeFilter) SetMaxDepth(depth int) NodeFilter {
	nf.MaxDepth = depth
	return nf
}
