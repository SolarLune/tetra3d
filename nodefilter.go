package tetra3d

import "strings"

type NodeFilter []INode

func newNodeFilter(nodes ...INode) NodeFilter {
	return NodeFilter(nodes)
}

// First returns the first Node in the NodeFilter; if the NodeFilter is empty, this function returns nil.
func (nf NodeFilter) First() INode {
	if len(nf) > 0 {
		return nf[0]
	}
	return nil
}

// First returns the last Node in the NodeFilter; if the NodeFilter is empty, this function returns nil.
func (nf NodeFilter) Last() INode {
	if len(nf) > 0 {
		return nf[len(nf)-1]
	}
	return nil
}

// Get returns the Node at the given index in the NodeFilter; if index is invalid (<0 or >= len(nodes)), this function returns nil.
func (nf NodeFilter) Get(index int) INode {
	if index < 0 || index >= len(nf) {
		return nil
	}
	return nf[index]
}

// ByFunc allows you to filter a given selection of nodes by the provided filter function (which takes a Node
// and returns a boolean, indicating whether or not to add that Node to the resulting NodeFilter).
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByFunc(filterFunc func(node INode) bool) NodeFilter {
	out := make([]INode, len(nf))
	i := 0
	for _, node := range nf {
		if filterFunc(node) {
			out[i] = node
			i++
		}
	}
	return newNodeFilter(out...)
}

// ByName allows you to filter a given selection of nodes by the given name. If wildcard is true,
// the nodes' names can contain the name provided; otherwise, they have to match exactly.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByName(name string, exactMatch bool) NodeFilter {
	out := []INode{}
	for _, node := range nf {

		var nameExists bool

		if exactMatch {
			nameExists = node.Name() == name
		} else {
			nameExists = strings.Contains(node.Name(), name)
		}

		if nameExists {
			out = append(out, node)
		}
	}
	return newNodeFilter(out...)
}

// ByType allows you to filter a given selection of nodes by the provided NodeType.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByType(nodeType NodeType) NodeFilter {

	out := []INode{}

	for _, node := range nf {

		if node.Type().Is(nodeType) {
			out = append(out, node)
		}

	}

	return newNodeFilter(out...)

}

// ByTags allows you to filter a given selection of nodes by the provided set of tag names.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByTags(tagNames ...string) NodeFilter {
	out := []INode{}
	for _, node := range nf {
		if node.Tags().Has(tagNames...) {
			out = append(out, node)
		}
	}
	return newNodeFilter(out...)
}

// Empty returns true if the NodeFilter contains no Nodes.
func (nf NodeFilter) Empty() bool {
	return len(nf) == 0
}
