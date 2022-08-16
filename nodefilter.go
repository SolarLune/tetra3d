package tetra3d

import (
	"strings"
)

// NodeFilter represents a filterable selection of INodes. For example, `
// filter := scene.Root.ChildrenRecursive()` returns a NodeFilter composed of all nodes
// underneath the root (excluding the root itself). From there, you can use additional
// functions on the NodeFilter to filter it down further:
// `filter = filter.ByName("player lamp", true).ByType(tetra3d.NodeTypeLight)`.
type NodeFilter []INode

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
	out := make([]INode, 0, len(nf))
	i := 0
	for _, node := range nf {
		if filterFunc(node) {
			out[i] = node
			i++
		}
	}
	return out
}

// ByName allows you to filter a given selection of nodes by the given name. If wildcard is true,
// the nodes' names can contain the name provided; otherwise, they have to match exactly.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByName(name string, exactMatch bool) NodeFilter {
	out := make([]INode, 0, len(nf))
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
	return out
}

// ByType allows you to filter a given selection of nodes by the provided NodeType.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByType(nodeType NodeType) NodeFilter {

	out := make([]INode, 0, len(nf))

	for _, node := range nf {

		if node.Type().Is(nodeType) {
			out = append(out, node)
		}

	}

	return out

}

// ByTags allows you to filter a given selection of nodes by the provided set of tag names.
// If no matching Nodes are found, an empty NodeFilter is returned.
func (nf NodeFilter) ByTags(tagNames ...string) NodeFilter {
	out := make([]INode, 0, len(nf))
	for _, node := range nf {
		if node.Tags().Has(tagNames...) {
			out = append(out, node)
		}
	}
	return out
}

// Children filters out a selection of Nodes, returning a NodeFilter composed strictly of that selection's children.
func (nf NodeFilter) Children() NodeFilter {
	out := make([]INode, 0, len(nf))
	for _, node := range nf {
		out = append(out, node.Children()...)
	}
	return out
}

// ChildrenRecursive filters out a selection of Nodes, returning a NodeFilter composed strictly of that selection's
// recursive children.
func (nf NodeFilter) ChildrenRecursive() NodeFilter {
	out := make([]INode, 0, len(nf))
	for _, node := range nf {
		out = append(out, node.ChildrenRecursive()...)
	}
	return out
}

// Empty returns true if the NodeFilter contains no Nodes.
func (nf NodeFilter) Empty() bool {
	return len(nf) == 0
}

// BoundingObjects returns a slice of the BoundingObjects contained within the NodeFilter. This is particularly useful when you're using
// NodeFilters to filter down a selection of Nodes that you then need to pass into BoundingObject.CollisionTest().
func (nc NodeFilter) BoundingObjects() []BoundingObject {
	boundings := make([]BoundingObject, 0, len(nc))
	for _, n := range nc {
		if b, ok := n.(BoundingObject); ok {
			boundings = append(boundings, b)
		}
	}
	return boundings
}

// Models returns a slice of the *Models contained within the NodeFilter.
func (nc NodeFilter) Models() []*Model {
	models := make([]*Model, 0, len(nc))
	for _, n := range nc {
		if m, ok := n.(*Model); ok {
			models = append(models, m)
		}
	}
	return models
}

// Lights returns a slice of the ILights contained within the NodeFilter.
func (nc NodeFilter) Lights() []ILight {
	lights := make([]ILight, 0, len(nc))
	for _, n := range nc {
		if m, ok := n.(ILight); ok {
			lights = append(lights, m)
		}
	}
	return lights
}
