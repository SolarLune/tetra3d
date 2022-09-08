package tetra3d

type GridPoint struct {
	*Node
	Connections []*GridPoint
	prevLink    *GridPoint
}

func NewGridPoint(name string) *GridPoint {
	gridPoint := &GridPoint{
		Node:        NewNode(name),
		Connections: []*GridPoint{},
	}
	return gridPoint
}

func (point *GridPoint) Clone() INode {
	newPoint := NewGridPoint(point.name)
	newPoint.Connections = append(newPoint.Connections, point.Connections...)
	return newPoint
}

func (point *GridPoint) IsConnected(other *GridPoint) bool {

	for _, c := range point.Connections {
		if c == other {
			return true
		}
	}

	return false

}

func (point *GridPoint) IsOnSameGrid(other *GridPoint) bool {
	return point.parent == other.parent
}

func (point *GridPoint) Connect(other *GridPoint) {

	if !point.IsConnected(other) {
		point.Connections = append(point.Connections, other)
	}

	if !other.IsConnected(point) {
		other.Connections = append(other.Connections, point)
	}

}

func (point *GridPoint) PathTo(other *GridPoint) []*GridPoint {
	path := []*GridPoint{}
	if !point.IsOnSameGrid(other) {
		return nil
	}

	toCheck := []*GridPoint{point}
	checked := map[*GridPoint]bool{}

	point.prevLink = nil

	var next *GridPoint

	for true {

		next = toCheck[0]

		toCheck = toCheck[1:]
		checked[next] = true

		if next == other {
			break
		}

		for _, c := range next.Connections {
			if _, exists := checked[c]; !exists {
				c.prevLink = next
				toCheck = append(toCheck, c)
			}
		}

	}

	for next.prevLink != nil {
		path = append(path, next)
		next = next.prevLink
	}

	return path

}

////////////

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (point *GridPoint) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	point.addChildren(point, children...)
}

// Unparent unparents the Model from its parent, removing it from the scenegraph.
func (point *GridPoint) Unparent() {
	if point.parent != nil {
		point.parent.RemoveChildren(point)
	}
}

// Type returns the NodeType for this object.
func (point *GridPoint) Type() NodeType {
	return NodeTypeGridPoint
}

// /////

// type Grid struct {
// 	*Node
// }

// func NewGrid(name string) *Grid {
// 	grid := &Grid{
// 		Node: NewNode(name),
// 	}
// 	return grid
// }

// func (grid *Grid) Clone() INode {
// 	newGrid := NewGrid(grid.name)
// 	return newGrid
// }

// ////////////

// // AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// // hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
// func (grid *Grid) AddChildren(children ...INode) {
// 	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
// 	grid.addChildren(grid, children...)
// }

// // Unparent unparents the Model from its parent, removing it from the scenegraph.
// func (grid *Grid) Unparent() {
// 	if grid.parent != nil {
// 		grid.parent.RemoveChildren(grid)
// 	}
// }

// // Type returns the NodeType for this object.
// func (grid *Grid) Type() NodeType {
// 	return NodeTypeGrid
// }
