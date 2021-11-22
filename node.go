package tetra3d

import (
	"github.com/kvartborg/vector"
)

// Node represents an object that exists in 3D space and can be positioned relative to an origin point.
// By default, this origin point is {0, 0, 0} (or world origin), but Nodes can be parented
// to other Nodes to change this origin (making their movements relative and their transforms
// successive). Models and Cameras are two examples of objects that fully implement the Node interface
// by means of NodeBase (and so can be considered Nodes).
type Node interface {
	Name() string
	SetName(name string)
	Clone() Node

	setParent(Node)
	Parent() Node
	Children() []Node
	ChildrenRecursive(onlyVisible bool) []Node
	AddChildren(...Node)
	RemoveChildren(...Node)
	updateLocalTransform(newParent Node)
	dirtyTransform()

	LocalRotation() Matrix4
	SetLocalRotation(rotation Matrix4)
	LocalPosition() vector.Vector
	SetLocalPosition(position vector.Vector)
	LocalScale() vector.Vector
	SetLocalScale(scale vector.Vector)

	WorldRotation() Matrix4
	SetWorldRotation(rotation Matrix4)
	WorldPosition() vector.Vector
	SetWorldPosition(position vector.Vector)
	WorldScale() vector.Vector
	SetWorldScale(scale vector.Vector)

	Transform() Matrix4

	Visible() bool
	SetVisible(visible bool)

	TreeToString() string

	Tags() *Tags
}

// Tags is an unordered set of string tags, representing a means of identifying Nodes.
type Tags struct {
	tags map[string]bool
}

// NewTags returns a new Tags object.
func NewTags() *Tags {
	return &Tags{map[string]bool{}}
}

// Clear clears the Tags object of all tags.
func (tags *Tags) Clear() {
	tags.tags = map[string]bool{}
}

// Add adds all of the tags specified to the Tags object.
func (tags *Tags) Add(nodeTags ...string) {
	for _, t := range nodeTags {
		tags.tags[t] = true
	}
}

// Remove removes all of the tags specified from the Tags object.
func (tags *Tags) Remove(nodeTags ...string) {
	for _, t := range nodeTags {
		delete(tags.tags, t)
	}
}

// Has returns true if the Tags object has all of the tags specified, and false otherwise.
func (tags *Tags) Has(nodeTags ...string) bool {
	for _, t := range nodeTags {
		if _, exists := tags.tags[t]; !exists {
			return false
		}
	}
	return true
}

// NodeBase represents a minimal struct that fully implements the Node interface. Model and Camera embed NodeBase
// into their structs to automatically easily implement Node.
type NodeBase struct {
	name             string
	parent           Node
	position         vector.Vector
	scale            vector.Vector
	rotation         Matrix4
	cachedTransform  Matrix4
	isTransformDirty bool
	children         []Node
	visible          bool
	tags             *Tags // Tags is an unordered set of string tags, representing a means of identifying Nodes.
}

// NewNodeBase returns a new NodeBase. You generally shouldn't need to call this.
func NewNodeBase(name string) *NodeBase {
	return &NodeBase{
		name:             name,
		position:         vector.Vector{0, 0, 0},
		scale:            vector.Vector{1, 1, 1},
		rotation:         NewMatrix4(),
		children:         []Node{},
		visible:          true,
		isTransformDirty: true,
		tags:             NewTags(),
	}
}

// Name returns the object's name.
func (nodeBase *NodeBase) Name() string {
	return nodeBase.name
}

// SetName sets the object's name.
func (nodeBase *NodeBase) SetName(name string) {
	nodeBase.name = name
}

// Clone returns a new Node.
func (nodeBase *NodeBase) Clone() Node {
	newNode := NewNodeBase(nodeBase.name)
	newNode.position = nodeBase.position.Clone()
	newNode.scale = nodeBase.scale.Clone()
	newNode.rotation = nodeBase.rotation.Clone()
	newNode.parent = nodeBase.parent
	for _, child := range nodeBase.children {
		newNode.children = append(newNode.children, child.Clone())
	}
	return newNode
}

// Transform returns a Matrix4 indicating the position, rotation, and scale of the object, transforming it by any parent's.
// If there's no change between the previous Transform() call and this one, Transform() will return a cached version of the
// transform for efficiency.
func (nodeBase *NodeBase) Transform() Matrix4 {

	// T * R * S * O

	if !nodeBase.isTransformDirty {
		return nodeBase.cachedTransform
	}

	transform := NewMatrix4Scale(nodeBase.scale[0], nodeBase.scale[1], nodeBase.scale[2])
	transform = transform.Mult(nodeBase.rotation)
	transform = transform.Mult(NewMatrix4Translate(nodeBase.position[0], nodeBase.position[1], nodeBase.position[2]))

	if nodeBase.parent != nil {
		transform = transform.Mult(nodeBase.parent.Transform())
	}

	nodeBase.cachedTransform = transform
	nodeBase.isTransformDirty = false

	return transform

}

// dirtyTransform sets this NodeBase and all recursive children's isTransformDirty flags to be true, indicating that they need to be
// rebuilt. This should be called when modifying the transformation properties (position, scale, rotation) of the NodeBase.
func (nodeBase *NodeBase) dirtyTransform() {

	if nodeBase.isTransformDirty {

		for _, child := range nodeBase.ChildrenRecursive(false) {
			child.dirtyTransform()
		}

	}

	nodeBase.isTransformDirty = true

}

// updateLocalTransform updates the local transform properties for a NodeBase given a change in parenting. This is done so that, for example,
// parenting an object with a given postiion, scale, and rotation keeps those visual properties when parenting (by updating them to take into
// account the parent's transforms as well).
func (nodeBase *NodeBase) updateLocalTransform(newParent Node) {

	if newParent != nil {

		parentTransform := newParent.Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		diff := nodeBase.position.Sub(parentPos)
		diff[0] /= parentScale[0]
		diff[1] /= parentScale[1]
		diff[2] /= parentScale[2]
		nodeBase.position = parentRot.Transposed().MultVec(diff)
		nodeBase.rotation = nodeBase.rotation.Mult(parentRot.Transposed())

		nodeBase.scale[0] /= parentScale[0]
		nodeBase.scale[1] /= parentScale[1]
		nodeBase.scale[2] /= parentScale[2]

	} else {

		// Reverse

		parentTransform := nodeBase.Parent().Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		pr := parentRot.MultVec(nodeBase.position)
		pr[0] *= parentScale[0]
		pr[1] *= parentScale[1]
		pr[2] *= parentScale[2]
		nodeBase.position = parentPos.Add(pr)
		nodeBase.rotation = nodeBase.rotation.Mult(parentRot)

		nodeBase.scale[0] *= parentScale[0]
		nodeBase.scale[1] *= parentScale[1]
		nodeBase.scale[2] *= parentScale[2]

	}

	nodeBase.dirtyTransform()

}

// LocalPosition returns a 3D Vector consisting of the object's local position (position relative to its parent). If this object has no parent, the position will be
// relative to world origin (0, 0, 0).
func (nodeBase *NodeBase) LocalPosition() vector.Vector {
	return nodeBase.position
}

// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
func (nodeBase *NodeBase) SetLocalPosition(position vector.Vector) {
	nodeBase.position = position
	nodeBase.dirtyTransform()
}

// WorldPosition returns a 3D Vector consisting of the object's world position (position relative to the world origin point of {0, 0, 0}).
func (nodeBase *NodeBase) WorldPosition() vector.Vector {
	position, _, _ := nodeBase.Transform().Decompose()
	return position
}

// SetWorldPosition sets the object's world position (position relative to the world origin point of {0, 0, 0}).
// position needs to be a 3D vector (i.e. X, Y, and Z components).
func (nodeBase *NodeBase) SetWorldPosition(position vector.Vector) {

	if nodeBase.parent != nil {

		parentTransform := nodeBase.parent.Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		pr := parentRot.Transposed().MultVec(position.Sub(parentPos))
		pr[0] /= parentScale[0]
		pr[1] /= parentScale[1]
		pr[2] /= parentScale[2]

		nodeBase.position = pr

	} else {
		nodeBase.position = position
	}

	nodeBase.dirtyTransform()

}

// LocalScale returns the object's local scale (scale relative to its parent). If this object has no parent, the scale will be absolute.
func (nodeBase *NodeBase) LocalScale() vector.Vector {
	return nodeBase.scale
}

// SetLocalScale sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
// scale should be a 3D vector (i.e. X, Y, and Z components).
func (nodeBase *NodeBase) SetLocalScale(scale vector.Vector) {
	nodeBase.scale = scale
	nodeBase.dirtyTransform()
}

// WorldScale returns the object's absolute world scale as a 3D vector (i.e. X, Y, and Z components).
func (nodeBase *NodeBase) WorldScale() vector.Vector {
	_, scale, _ := nodeBase.Transform().Decompose()
	return scale
}

// SetWorldScale sets the object's absolute world scale. scale should be a 3D vector (i.e. X, Y, and Z components).
func (nodeBase *NodeBase) SetWorldScale(scale vector.Vector) {

	if nodeBase.parent != nil {

		parentTransform := nodeBase.parent.Transform()
		_, parentScale, _ := parentTransform.Decompose()

		nodeBase.scale = vector.Vector{
			scale[0] / parentScale[0],
			scale[1] / parentScale[1],
			scale[2] / parentScale[2],
		}

	} else {
		nodeBase.scale = scale
	}

	nodeBase.dirtyTransform()
}

// LocalRotation returns the object's local rotation Matrix4.
func (nodeBase *NodeBase) LocalRotation() Matrix4 {
	return nodeBase.rotation
}

// SetLocalRotation sets the object's local rotation Matrix4 (relative to any parent).
func (nodeBase *NodeBase) SetLocalRotation(rotation Matrix4) {
	nodeBase.rotation = rotation
	nodeBase.dirtyTransform()
}

// WorldRotation returns an absolute rotation Matrix4 representing the object's rotation.
func (nodeBase *NodeBase) WorldRotation() Matrix4 {
	_, _, rotation := nodeBase.Transform().Decompose()
	return rotation
}

// SetWorldRotation sets an object's rotation to the provided rotation Matrix4.
func (nodeBase *NodeBase) SetWorldRotation(rotation Matrix4) {

	if nodeBase.parent != nil {

		parentTransform := nodeBase.parent.Transform()
		_, _, parentRot := parentTransform.Decompose()
		nodeBase.rotation = parentRot.Transposed().Mult(rotation)

	} else {
		nodeBase.rotation = rotation
	}

	nodeBase.dirtyTransform()
}

// Parent returns the object's parent. If the object has no parent, this will return nil.
func (nodeBase *NodeBase) Parent() Node {
	return nodeBase.parent
}

func (nodeBase *NodeBase) setParent(node Node) {
	nodeBase.parent = node
}

// addChildren adds the children to the parent node, but sets their parent to be the parent node passed. This is done so children have the
// correct, specific Node as parent (because I can't really think of a better way to do this rn). Basically, without this approach,
// after parent.AddChildren(child), child.Parent() wouldn't be parent, but rather parent.NodeBase, which is no good.
func (nodeBase *NodeBase) addChildren(parent Node, children ...Node) {
	for _, child := range children {
		child.updateLocalTransform(parent)
		if child.Parent() != nil {
			child.Parent().RemoveChildren(child)
		}
		child.setParent(parent)
		nodeBase.children = append(nodeBase.children, child)
	}
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (nodeBase *NodeBase) AddChildren(children ...Node) {
	nodeBase.addChildren(nodeBase, children...)
}

// RemoveChildren removes the provided children from this object.
func (nodeBase *NodeBase) RemoveChildren(children ...Node) {

	for _, child := range children {
		for i, c := range nodeBase.children {
			if c == child {
				child.updateLocalTransform(nil)
				child.setParent(nil)
				nodeBase.children[i] = nil
				nodeBase.children = append(nodeBase.children[:i], nodeBase.children[i+1:]...)
			}
		}
	}

}

// Children() returns the Node's children.
func (nodeBase *NodeBase) Children() []Node {
	return append([]Node{}, nodeBase.children...)
}

// ChildrenRecursive returns all related children Nodes underneath this one. If onlyVisible is true, ChildrenRecursive will only return
// Nodes that are visible.
func (nodeBase *NodeBase) ChildrenRecursive(onlyVisible bool) []Node {
	out := []Node{}
	for _, child := range nodeBase.children {
		if !onlyVisible || child.Visible() {
			out = append(out, child)
			out = append(out, child.ChildrenRecursive(onlyVisible)...)
		}
	}
	return out
}

// Visible returns whether the Object is visible.
func (nodeBase *NodeBase) Visible() bool {
	return nodeBase.visible
}

// SetVisible sets the object's visibility.
func (nodeBase *NodeBase) SetVisible(visible bool) {
	nodeBase.visible = visible
}

// Tags represents an unordered set of string tags that can be used to identify this object.
func (nodeBase *NodeBase) Tags() *Tags {
	return nodeBase.tags
}

// TreeToString returns a string displaying the hierarchy of this Node, and all recursive children. This is a useful
// function to debug the layout of a node tree, for example.
func (nodeBase *NodeBase) TreeToString() string {

	var printNode func(node Node, level int) string

	printNode = func(node Node, level int) string {

		str := ""
		if level == 0 {
			str = "+: " + node.Name() + "\n"
		} else {

			for i := 0; i < level; i++ {
				str += "    "
			}

			str += "\\-: " + node.Name() + "\n"
		}

		for _, child := range node.Children() {
			str += printNode(child, level+1)
		}

		return str
	}

	return printNode(nodeBase, 0)
}
