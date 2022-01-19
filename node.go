package tetra3d

import (
	"strconv"
	"strings"

	"github.com/kvartborg/vector"
)

// INode represents an object that exists in 3D space and can be positioned relative to an origin point.
// By default, this origin point is {0, 0, 0} (or world origin), but Nodes can be parented
// to other Nodes to change this origin (making their movements relative and their transforms
// successive). Models and Cameras are two examples of objects that fully implement the INode interface
// by means of embedding Node.
type INode interface {
	Name() string
	SetName(name string)
	Clone() INode
	SetData(data interface{})
	Data() interface{}

	setParent(INode)
	Parent() INode
	Unparent()
	Root() INode
	Children() []INode
	ChildrenRecursive() []INode
	AddChildren(...INode)
	RemoveChildren(...INode)
	// updateLocalTransform(newParent INode)
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

	Move(x, y, z float64)
	MoveVec(moveVec vector.Vector)
	Rotate(x, y, z, angle float64)
	Grow(x, y, z float64)

	Transform() Matrix4

	Visible() bool
	SetVisible(visible, recursive bool)

	Get(path string) INode
	FindByName(name string) []INode
	FindByTags(tags ...string) []INode
	HierarchyAsString() string
	Path() string

	Tags() *Tags

	IsBone() bool
	// IsRootBone() bool
}

// Tags is an unordered set of string tags to values, representing a means of identifying Nodes or carrying data on Nodes.
type Tags struct {
	tags map[string]interface{}
}

// NewTags returns a new Tags object.
func NewTags() *Tags {
	return &Tags{map[string]interface{}{}}
}

func (tags *Tags) Clone() *Tags {
	newTags := NewTags()
	for k, v := range tags.tags {
		newTags.Set(k, v)
	}
	return newTags
}

// Clear clears the Tags object of all tags.
func (tags *Tags) Clear() {
	tags.tags = map[string]interface{}{}
}

// Set sets all the tag specified to the Tags object.
func (tags *Tags) Set(tagName string, value interface{}) {
	tags.tags[tagName] = value
}

// Remove removes the tag specified from the Tags object.
func (tags *Tags) Remove(tag string) {
	delete(tags.tags, tag)
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

// Get returns the value associated with the specified tag (key).
func (tags *Tags) Get(tagName string) interface{} {
	return tags.tags[tagName]
}

// IsString returns true if the value associated with the specified tag is a string.
func (tags *Tags) IsString(tagName string) bool {
	if _, exists := tags.tags[tagName]; exists {
		if _, ok := tags.tags[tagName].(string); ok {
			return true
		}
	}
	return false
}

// GetAsString returns the value associated with the specified tag (key) as a string.
func (tags *Tags) GetAsString(tagName string) string {
	return tags.tags[tagName].(string)
}

// IsFloat returns true if the value associated with the specified tag is a float64.
func (tags *Tags) IsFloat(tagName string) bool {
	if _, exists := tags.tags[tagName]; exists {
		if _, ok := tags.tags[tagName].(float64); ok {
			return true
		}
	}
	return false
}

// GetAsFloat returns the value associated with the specified tag (key) as a float.
func (tags *Tags) GetAsFloat(tagName string) float64 {
	return tags.tags[tagName].(float64)
}

// Node represents a minimal struct that fully implements the Node interface. Model and Camera embed Node
// into their structs to automatically easily implement Node.
type Node struct {
	name              string
	parent            INode
	position          vector.Vector
	scale             vector.Vector
	rotation          Matrix4
	data              interface{} // A place to store a pointer to something if you need it
	cachedTransform   Matrix4
	isTransformDirty  bool
	onTransformUpdate func()
	children          []INode
	visible           bool
	tags              *Tags // Tags is an unordered set of string tags, representing a means of identifying Nodes.
	AnimationPlayer   *AnimationPlayer
	inverseBindMatrix Matrix4 // Specifically for bones in an armature used for animating skinned meshes
	isBone            bool
}

// NewNode returns a new Node.
func NewNode(name string) *Node {

	nb := &Node{
		name:             name,
		position:         vector.Vector{0, 0, 0},
		scale:            vector.Vector{1, 1, 1},
		rotation:         NewMatrix4(),
		children:         []INode{},
		visible:          true,
		isTransformDirty: true,
		tags:             NewTags(),
	}

	nb.AnimationPlayer = NewAnimationPlayer(nb)

	return nb
}

// Name returns the object's name.
func (node *Node) Name() string {
	return node.name
}

// SetName sets the object's name.
func (node *Node) SetName(name string) {
	node.name = name
}

// Clone returns a new Node.
func (node *Node) Clone() INode {
	newNode := NewNode(node.name)
	newNode.position = node.position.Clone()
	newNode.scale = node.scale.Clone()
	newNode.rotation = node.rotation.Clone()
	newNode.parent = node.parent
	newNode.AnimationPlayer = node.AnimationPlayer.Clone()
	newNode.tags = node.tags.Clone()

	if node.AnimationPlayer.RootNode == node {
		newNode.AnimationPlayer.SetRoot(newNode)
	}

	for _, child := range node.children {
		childClone := child.Clone()
		childClone.setParent(newNode)
		newNode.children = append(newNode.children, childClone)
	}

	for _, child := range newNode.children {
		if model, isModel := child.(*Model); isModel && model.SkinRoot == node {
			model.ReassignBones(newNode)
		}
	}

	newNode.isBone = node.isBone
	if newNode.isBone {
		newNode.inverseBindMatrix = node.inverseBindMatrix.Clone()
	}

	return newNode
}

// SetData sets user-customizeable data that could be usefully stored on this node.
func (node *Node) SetData(data interface{}) {
	node.data = data
}

// Data returns a pointer to user-customizeable data that could be usefully stored on this node.
func (node *Node) Data() interface{} {
	return node.data
}

// Transform returns a Matrix4 indicating the global position, rotation, and scale of the object, transforming it by any parents'.
// If there's no change between the previous Transform() call and this one, Transform() will return a cached version of the
// transform for efficiency.
func (node *Node) Transform() Matrix4 {

	// T * R * S * O

	if !node.isTransformDirty {
		return node.cachedTransform
	}

	transform := NewMatrix4Scale(node.scale[0], node.scale[1], node.scale[2])
	transform = transform.Mult(node.rotation)
	transform = transform.Mult(NewMatrix4Translate(node.position[0], node.position[1], node.position[2]))

	if node.parent != nil {
		transform = transform.Mult(node.parent.Transform())
	}

	node.cachedTransform = transform
	node.isTransformDirty = false
	if node.onTransformUpdate != nil {
		node.onTransformUpdate()
	}

	return transform

}

// dirtyTransform sets this Node and all recursive children's isTransformDirty flags to be true, indicating that they need to be
// rebuilt. This should be called when modifying the transformation properties (position, scale, rotation) of the Node.
func (node *Node) dirtyTransform() {

	if !node.isTransformDirty {

		for _, child := range node.ChildrenRecursive() {
			child.dirtyTransform()
		}

	}

	node.isTransformDirty = true

}

// updateLocalTransform updates the local transform properties for a Node given a change in parenting. This is done so that, for example,
// parenting an object with a given postiion, scale, and rotation keeps those visual properties when parenting (by updating them to take into
// account the parent's transforms as well).
// func (node *Node) updateLocalTransform(newParent INode) {

// 	if newParent != nil {

// 		parentTransform := newParent.Transform()
// 		parentPos, parentScale, parentRot := parentTransform.Decompose()

// 		diff := node.position.Sub(parentPos)
// 		diff[0] /= parentScale[0]
// 		diff[1] /= parentScale[1]
// 		diff[2] /= parentScale[2]
// 		node.position = parentRot.Transposed().MultVec(diff)
// 		node.rotation = node.rotation.Mult(parentRot.Transposed())

// 		node.scale[0] /= parentScale[0]
// 		node.scale[1] /= parentScale[1]
// 		node.scale[2] /= parentScale[2]

// 	} else {

// 		// Reverse

// 		parentTransform := node.Parent().Transform()
// 		parentPos, parentScale, parentRot := parentTransform.Decompose()

// 		pr := parentRot.MultVec(node.position)
// 		pr[0] *= parentScale[0]
// 		pr[1] *= parentScale[1]
// 		pr[2] *= parentScale[2]
// 		node.position = parentPos.Add(pr)
// 		node.rotation = node.rotation.Mult(parentRot)

// 		node.scale[0] *= parentScale[0]
// 		node.scale[1] *= parentScale[1]
// 		node.scale[2] *= parentScale[2]

// 	}

// 	node.dirtyTransform()

// }

// LocalPosition returns a 3D Vector consisting of the object's local position (position relative to its parent). If this object has no parent, the position will be
// relative to world origin (0, 0, 0).
func (node *Node) LocalPosition() vector.Vector {
	return node.position
}

// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalPosition(position vector.Vector) {
	node.position = position
	node.dirtyTransform()
}

// ResetLocalTransformProperties resets the local transform properties (position, scale, and rotation) for the Node. This can be useful because
// by default, when you parent one Node to another, the local transform properties (position, scale, and rotation) are altered to keep the
// object in the same absolute location, even though the origin changes.
func (node *Node) ResetLocalTransform() {
	node.position[0] = 0
	node.position[1] = 0
	node.position[2] = 0
	node.scale[0] = 1
	node.scale[1] = 1
	node.scale[2] = 1
	node.rotation = NewMatrix4()
	node.dirtyTransform()
}

// WorldPosition returns a 3D Vector consisting of the object's world position (position relative to the world origin point of {0, 0, 0}).
func (node *Node) WorldPosition() vector.Vector {
	position := node.Transform().Row(3)[:3] // We don't want to have to decompose if we don't have to
	return position
}

// SetWorldPosition sets the object's world position (position relative to the world origin point of {0, 0, 0}).
// position needs to be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetWorldPosition(position vector.Vector) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		pr := parentRot.Transposed().MultVec(position.Sub(parentPos))
		pr[0] /= parentScale[0]
		pr[1] /= parentScale[1]
		pr[2] /= parentScale[2]

		node.position = pr

	} else {
		node.position = position
	}

	node.dirtyTransform()

}

// LocalScale returns the object's local scale (scale relative to its parent). If this object has no parent, the scale will be absolute.
func (node *Node) LocalScale() vector.Vector {
	return node.scale
}

// SetLocalScale sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
// scale should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalScale(scale vector.Vector) {
	node.scale = scale
	node.dirtyTransform()
}

// WorldScale returns the object's absolute world scale as a 3D vector (i.e. X, Y, and Z components).
func (node *Node) WorldScale() vector.Vector {
	_, scale, _ := node.Transform().Decompose()
	return scale
}

// SetWorldScale sets the object's absolute world scale. scale should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetWorldScale(scale vector.Vector) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		_, parentScale, _ := parentTransform.Decompose()

		node.scale = vector.Vector{
			scale[0] / parentScale[0],
			scale[1] / parentScale[1],
			scale[2] / parentScale[2],
		}

	} else {
		node.scale = scale
	}

	node.dirtyTransform()
}

// LocalRotation returns the object's local rotation Matrix4.
func (node *Node) LocalRotation() Matrix4 {
	return node.rotation
}

// SetLocalRotation sets the object's local rotation Matrix4 (relative to any parent).
func (node *Node) SetLocalRotation(rotation Matrix4) {
	node.rotation = rotation
	node.dirtyTransform()
}

// WorldRotation returns an absolute rotation Matrix4 representing the object's rotation.
func (node *Node) WorldRotation() Matrix4 {
	_, _, rotation := node.Transform().Decompose()
	return rotation
}

// SetWorldRotation sets an object's rotation to the provided rotation Matrix4.
func (node *Node) SetWorldRotation(rotation Matrix4) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		_, _, parentRot := parentTransform.Decompose()
		node.rotation = parentRot.Transposed().Mult(rotation)

	} else {
		node.rotation = rotation
	}

	node.dirtyTransform()
}

func (node *Node) Move(x, y, z float64) {
	localPos := node.LocalPosition()
	localPos[0] += x
	localPos[1] += y
	localPos[2] += z
	node.SetLocalPosition(localPos)
}

func (node *Node) MoveVec(vec vector.Vector) {
	localPos := node.LocalPosition()
	localPos[0] += vec[0]
	localPos[1] += vec[1]
	localPos[2] += vec[2]
	node.SetLocalPosition(localPos)
}

func (node *Node) Rotate(x, y, z, angle float64) {
	localRot := node.LocalRotation()
	localRot = localRot.Rotated(x, y, z, angle)
	node.SetLocalRotation(localRot)
}

// Grow scales the object additively (i.e. calling Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
func (node *Node) Grow(x, y, z float64) {
	scale := node.LocalScale()
	scale[0] += x
	scale[1] += y
	scale[2] += z
	node.SetLocalScale(scale)
}

// Parent returns the Node's parent. If the Node has no parent, this will return nil.
func (node *Node) Parent() INode {
	return node.parent
}

// setParent sets the Node's parent.
func (node *Node) setParent(parent INode) {
	node.parent = parent
}

// addChildren adds the children to the parent node, but sets their parent to be the parent node passed. This is done so children have the
// correct, specific Node as parent (because I can't really think of a better way to do this rn). Basically, without this approach,
// after parent.AddChildren(child), child.Parent() wouldn't be parent, but rather parent.Node, which is no good.
func (node *Node) addChildren(parent INode, children ...INode) {
	for _, child := range children {
		// child.updateLocalTransform(parent)
		if child.Parent() != nil {
			child.Parent().RemoveChildren(child)
		}
		child.setParent(parent)
		node.children = append(node.children, child)
	}
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (node *Node) AddChildren(children ...INode) {
	node.addChildren(node, children...)
}

// RemoveChildren removes the provided children from this object.
func (node *Node) RemoveChildren(children ...INode) {

	for _, child := range children {
		for i, c := range node.children {
			if c == child {
				// child.updateLocalTransform(nil)
				child.setParent(nil)
				node.children[i] = nil
				node.children = append(node.children[:i], node.children[i+1:]...)
				break
			}
		}
	}

}

// Unparent unparents the Node from its parent, removing it from the scenegraph. Note that this needs to be overridden for objects that embed Node.
func (node *Node) Unparent() {
	if node.parent != nil {
		node.parent.RemoveChildren(node)
	}
}

// Children() returns the Node's children.
func (node *Node) Children() []INode {
	return append([]INode{}, node.children...)
}

// ChildrenRecursive returns all related children Nodes underneath this one.
func (node *Node) ChildrenRecursive() []INode {
	out := []INode{}
	for _, child := range node.children {
		out = append(out, child)
		out = append(out, child.ChildrenRecursive()...)
	}
	return out
}

// Visible returns whether the Object is visible.
func (node *Node) Visible() bool {
	return node.visible
}

// SetVisible sets the object's visibility. If recursive is true, all recursive children of this Node will have their visibility set the same way.
func (node *Node) SetVisible(visible bool, recursive bool) {
	node.visible = visible
	if recursive {
		for _, child := range node.ChildrenRecursive() {
			child.SetVisible(visible, true)
		}
	}
}

// Tags represents an unordered set of string tags that can be used to identify this object.
func (node *Node) Tags() *Tags {
	return node.tags
}

// HierarchyAsString returns a string displaying the hierarchy of this Node, and all recursive children.
// Nodes will have a "+" next to their name, Models an "M", and Cameras a "C".
// BoundingSpheres will have BS, BoundingAABB AABB, BoundingCapsule CAP, and BoundingTriangles TRI.
// Lights will have an L next to their name.
// This is a useful function to debug the layout of a node tree, for example.
func (node *Node) HierarchyAsString() string {

	var printNode func(node INode, level int) string

	printNode = func(node INode, level int) string {

		nodeType := "+"

		if _, ok := node.(*Model); ok {
			nodeType = "M"
		} else if _, ok := node.(Light); ok {
			nodeType = "L"
		} else if _, ok := node.(*Camera); ok {
			nodeType = "C"
		} else if _, ok := node.(*BoundingSphere); ok {
			nodeType = "BS"
		} else if _, ok := node.(*BoundingAABB); ok {
			nodeType = "AABB"
		} else if _, ok := node.(*BoundingCapsule); ok {
			nodeType = "CAP"
		} else if _, ok := node.(*BoundingTriangles); ok {
			nodeType = "TRI"
		}

		str := ""
		if level == 0 {
			str = "-: [+] " + node.Name() + "\n"
		} else {

			for i := 0; i < level; i++ {
				str += "    "
			}

			wp := node.LocalPosition()
			wpStr := "[" + strconv.FormatFloat(wp[0], 'f', -1, 64) + ", " + strconv.FormatFloat(wp[1], 'f', -1, 64) + ", " + strconv.FormatFloat(wp[2], 'f', -1, 64) + "]"

			str += "\\-: [" + nodeType + "] " + node.Name() + " : " + wpStr + "\n"
		}

		for _, child := range node.Children() {
			str += printNode(child, level+1)
		}

		return str
	}

	return printNode(node, 0)
}

// Get searches a node's hierarchy using a string to find a specified node. The path is in the format of names of nodes, separated by forward
// slashes ('/'), and is relative to the node you use to call Get. As an example of Get, if you had a cup parented to a desk, which was
// parented to a room, that was finally parented to the root of the scene, it would be found at "Room/Desk/Cup". Note also that you can use "../" to
// "go up one" in the hierarchy (so cup.Get("../") would return the Desk node).
// Since Get uses forward slashes as path separation, it would be good to avoid using forward slashes in your object names.
func (node *Node) Get(path string) INode {

	var search func(node INode) INode

	split := strings.Split(path, `/`)

	search = func(node INode) INode {

		if node == nil {
			return nil
		} else if len(split) == 0 {
			return node
		}

		if split[0] == ".." {
			split = split[1:]
			return search(node.Parent())
		}

		for _, child := range node.Children() {

			if child.Name() == split[0] {

				if len(split) <= 1 {
					return child
				} else {
					split = split[1:]
					return search(child)
				}

			}

		}

		return nil

	}

	return search(node)

}

// Path returns a string indicating the hierarchical path to get this Node from the root. The path returned will be absolute, such that
// passing it to Get() called on the scene root node will return this node. The path returned will not contain the root node's name ("Root").
func (node *Node) Path() string {

	root := node.Root()

	if root == nil {
		return ""
	}

	parent := node.Parent()
	path := node.Name()

	for parent != nil && parent != root {
		path = parent.Name() + "/" + path
		parent = parent.Parent()
	}

	return path

}

// FindByName allows you to search the node's recursive children, returning a slice
// of Nodes with the name given. If none are found, an empty slice is returned.
// After finding a Node, you can convert it to a more specific type as necessary via type assertion.
func (node *Node) FindByName(name string) []INode {
	out := []INode{}
	for _, node := range node.ChildrenRecursive() {
		if node.Name() == name {
			out = append(out, node)
		}
	}
	return out
}

// FindBytags allows you to search the node's recursive children, returning a slice
// of Nodes with the tags given. If none are found, an empty slice is returned.
// After finding a Node, you can convert it to a more specific type as necessary via type assertion.
func (node *Node) FindByTags(tagNames ...string) []INode {
	out := []INode{}
	for _, node := range node.ChildrenRecursive() {
		if node.Tags().Has(tagNames...) {
			out = append(out, node)
		}
	}
	return out
}

// Root returns the root node in this tree by recursively traversing this node's hierarchy of
// parents upwards.
func (node *Node) Root() INode {

	if node.parent == nil {
		return node
	}

	parent := node.Parent()

	for parent != nil {
		next := parent.Parent()
		if next == nil {
			break
		}
		parent = next
	}

	return parent

}

// IsBone returns if the Node is a "bone" (a node that was a part of an armature and so can play animations back to influence a skinned mesh).
func (node *Node) IsBone() bool {
	return node.isBone
}

// // IsRootBone returns if the Node SHOULD be the root of an Armature (a Node that was the base of an armature).
// func (node *Node) IsRootBone() bool {
// 	return node.IsBone() && (node.parent == nil || !node.parent.IsBone())
// }
