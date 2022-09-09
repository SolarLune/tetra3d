package tetra3d

import (
	"strconv"
	"strings"

	"github.com/kvartborg/vector"
)

// NodeType represents a Node's type. Node types are categorized, and can be said to extend or "be of" more general types.
// For example, a BoundingSphere has a type of NodeTypeBoundingSphere. That type can also be said to be NodeTypeBoundingObject
// (because it is a bounding object). However, it is not of type NodeTypeBoundingTriangles, as that is a different category.
type NodeType string

const (
	NodeTypeNode   NodeType = "Node"       // NodeTypeNode represents any generic node
	NodeTypeModel  NodeType = "NodeModel"  // NodeTypeModel represents specifically a Model
	NodeTypeCamera NodeType = "NodeCamera" // NodeTypeCamera represents specifically a Camera
	NodeTypePath   NodeType = "NodePath"   // NodeTypePath represents specifically a Path
	NodeTypeGrid   NodeType = "NodeGrid"   // NodeTypeGrid represents specifically a Grid

	NodeTypeGridPoint NodeType = "Node_GridPoint" // NodeTypeGrid represents specifically a GridPoint (note the extra underscore to ensure !NodeTypeGridPoint.Is(NodeTypeGrid))

	NodeTypeBoundingObject    NodeType = "NodeBounding"          // NodeTypeBoundingObject represents any generic bounding object
	NodeTypeBoundingAABB      NodeType = "NodeBoundingAABB"      // NodeTypeBoundingAABB represents specifically a BoundingAABB
	NodeTypeBoundingCapsule   NodeType = "NodeBoundingCapsule"   // NodeTypeBoundingCapsule represents specifically a BoundingCapsule
	NodeTypeBoundingTriangles NodeType = "NodeBoundingTriangles" // NodeTypeBoundingTriangles represents specifically a BoundingTriangles object
	NodeTypeBoundingSphere    NodeType = "NodeBoundingSphere"    // NodeTypeBoundingSphere represents specifically a BoundingSphere BoundingObject

	NodeTypeLight            NodeType = "NodeLight"            // NodeTypeLight represents any generic light
	NodeTypeAmbientLight     NodeType = "NodeLightAmbient"     // NodeTypeAmbientLight represents specifically an ambient light
	NodeTypePointLight       NodeType = "NodeLightPoint"       // NodeTypePointLight represents specifically a point light
	NodeTypeDirectionalLight NodeType = "NodeLightDirectional" // NodeTypeDirectionalLight represents specifically a directional (sun) light
	NodeTypeCubeLight        NodeType = "NodeLightCube"        // NodeTypeCubeLight represents, specifically, a cube light
)

// Is returns true if a NodeType satisfies another NodeType category. A specific node type can be said to
// contain a more general one, but not vice-versa. For example, a Model (which has type NodeTypeModel) can be
// said to be a Node (NodeTypeNode), but the reverse is not true (a NodeTypeNode is not a NodeTypeModel).
func (nt NodeType) Is(other NodeType) bool {
	if nt == other {
		return true
	}
	return strings.Contains(string(nt), string(other))
}

// INode represents an object that exists in 3D space and can be positioned relative to an origin point.
// By default, this origin point is {0, 0, 0} (or world origin), but Nodes can be parented
// to other Nodes to change this origin (making their movements relative and their transforms
// successive). Models and Cameras are two examples of objects that fully implement the INode interface
// by means of embedding Node.
type INode interface {
	// Name returns the object's name.
	Name() string
	// SetName sets the object's name.
	SetName(name string)
	// Clone returns a clone of the specified INode implementer.
	Clone() INode
	// SetData sets user-customizeable data that could be usefully stored on this node.
	SetData(data interface{})
	// Data returns a pointer to user-customizeable data that could be usefully stored on this node.
	Data() interface{}
	// Type returns the NodeType for this object.
	Type() NodeType
	setLibrary(lib *Library)
	// Library returns the source Library from which this Node was instantiated. If it was created through code, this will be nil.
	Library() *Library

	setParent(INode)

	// Parent returns the Node's parent. If the Node has no parent, this will return nil.
	Parent() INode
	// Unparent unparents the Node from its parent, removing it from the scenegraph.
	Unparent()
	// Scene looks for the Node's parents recursively to return what scene it exists in.
	// If the node is not within a tree (i.e. unparented), this will return nil.
	Scene() *Scene
	// Root returns the root node in this tree by recursively traversing this node's hierarchy of
	// parents upwards.
	Root() INode

	// Children() returns the Node's children as a NodeFilter.
	Children() NodeFilter
	// ChildrenRecursive() returns the Node's recursive children (i.e. children, grandchildren, etc)
	// as a NodeFilter.
	ChildrenRecursive() NodeFilter

	// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
	// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
	AddChildren(...INode)
	// RemoveChildren removes the provided children from this object.
	RemoveChildren(...INode)

	// updateLocalTransform(newParent INode)
	dirtyTransform()

	// ResetLocalTransform resets the local transform properties (position, scale, and rotation) for the Node. This can be useful because
	// by default, when you parent one Node to another, the local transform properties (position, scale, and rotation) are altered to keep the
	// object in the same absolute location, even though the origin changes.
	ResetLocalTransform()
	// SetWorldTransform sets the Node's global (world) transform to the full 4x4 transformation matrix provided.
	SetWorldTransform(transform Matrix4)

	// LocalRotation returns the object's local rotation Matrix4.
	LocalRotation() Matrix4
	// SetLocalRotation sets the object's local rotation Matrix4 (relative to any parent).
	SetLocalRotation(rotation Matrix4)
	LocalPosition() vector.Vector
	// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
	// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
	SetLocalPositionVec(position vector.Vector)
	SetLocalPosition(x, y, z float64)
	// LocalScale returns the object's local scale (scale relative to its parent). If this object has no parent, the scale will be absolute.
	LocalScale() vector.Vector
	// SetLocalScale sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
	// scale should be a 3D vector (i.e. X, Y, and Z components).
	SetLocalScaleVec(scale vector.Vector)
	SetLocalScale(w, h, d float64)

	// WorldRotation returns an absolute rotation Matrix4 representing the object's rotation.
	WorldRotation() Matrix4
	// SetWorldRotation sets an object's global, world rotation to the provided rotation Matrix4.
	SetWorldRotation(rotation Matrix4)
	WorldPosition() vector.Vector
	SetWorldPositionVec(position vector.Vector)
	SetWorldPosition(x, y, z float64)
	// WorldScale returns the object's absolute world scale as a 3D vector (i.e. X, Y, and Z components).
	WorldScale() vector.Vector
	// SetWorldScaleVec sets the object's absolute world scale. scale should be a 3D vector (i.e. X, Y, and Z components).
	SetWorldScaleVec(scale vector.Vector)

	// Move moves a Node in local space by the x, y, and z values provided.
	Move(x, y, z float64)
	// MoveVec moves a Node in local space using the vector provided.
	MoveVec(moveVec vector.Vector)
	// Rotate rotates a Node locally on the given vector with the given axis, by the angle provided in radians.
	Rotate(x, y, z, angle float64)
	// RotateVec rotates a Node locally on the given vector, by the angle provided in radians.
	RotateVec(vec vector.Vector, angle float64)
	// Grow scales the object additively (i.e. calling Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
	Grow(x, y, z float64)
	// GrowVec scales the object additively (i.e. calling Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
	GrowVec(vec vector.Vector)

	// Transform returns a Matrix4 indicating the global position, rotation, and scale of the object, transforming it by any parents'.
	// If there's no change between the previous Transform() call and this one, Transform() will return a cached version of the
	// transform for efficiency.
	Transform() Matrix4

	// Visible returns whether the Object is visible.
	Visible() bool
	// SetVisible sets the object's visibility. If recursive is true, all recursive children of this Node will have their visibility set the same way.
	SetVisible(visible, recursive bool)

	// Get searches a node's hierarchy using a string to find a specified node. The path is in the format of names of nodes, separated by forward
	// slashes ('/'), and is relative to the node you use to call Get. As an example of Get, if you had a cup parented to a desk, which was
	// parented to a room, that was finally parented to the root of the scene, it would be found at "Room/Desk/Cup". Note also that you can use "../" to
	// "go up one" in the hierarchy (so cup.Get("../") would return the Desk node).
	// Since Get uses forward slashes as path separation, it would be good to avoid using forward slashes in your Node names. Also note that Get()
	// trims the extra spaces from the beginning and end of Node Names, so avoid using spaces at the beginning or end of your Nodes' names.
	Get(path string) INode

	// HierarchyAsString returns a string displaying the hierarchy of this Node, and all recursive children.
	// Nodes will have a "+" next to their name, Models an "M", and Cameras a "C".
	// BoundingSpheres will have BS, BoundingAABB AABB, BoundingCapsule CAP, and BoundingTriangles TRI.
	// Lights will have an L next to their name.
	// This is a useful function to debug the layout of a node tree, for example.
	HierarchyAsString() string

	// Path returns a string indicating the hierarchical path to get this Node from the root. The path returned will be absolute, such that
	// passing it to Get() called on the scene root node will return this node. The path returned will not contain the root node's name ("Root").
	Path() string

	// Tags represents an unordered set of string tags that can be used to identify this object.
	Tags() *Tags

	// IsBone returns if the Node is a "bone" (a node that was a part of an armature and so can play animations back to influence a skinned mesh).
	IsBone() bool
	// IsRootBone() bool

	// AnimationPlayer returns the object's animation player - every object has an AnimationPlayer by default.
	AnimationPlayer() *AnimationPlayer

	setOriginalLocalPosition(vector.Vector)
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

// Get returns the value associated with the specified tag (key). If a tag with the
// passed name (tagName) doesn't exist, Get will return nil.
func (tags *Tags) Get(tagName string) interface{} {
	if tag, ok := tags.tags[tagName]; ok {
		return tag
	}
	return nil
}

// IsString returns true if the value associated with the specified tag is a string. If the
// tag doesn't exist or the tag's value isn't a string type, this returns false.
func (tags *Tags) IsString(tagName string) bool {
	if _, exists := tags.tags[tagName]; exists {
		if _, ok := tags.tags[tagName].(string); ok {
			return true
		}
	}
	return false
}

// GetAsString returns the value associated with the specified tag (key) as a string.
// If the tag doesn't exist, it will return an empty string.
// Note that this does not sanity check to ensure the tag is a string first.
func (tags *Tags) GetAsString(tagName string) string {

	if !tags.Has(tagName) {
		return ""
	}

	return tags.tags[tagName].(string)
}

// IsFloat returns true if the value associated with the specified tag is a float64. If the
// tag doesn't exist or the tag's value isn't a float64 type, this returns false.
func (tags *Tags) IsFloat(tagName string) bool {
	if _, exists := tags.tags[tagName]; exists {
		if _, ok := tags.tags[tagName].(float64); ok {
			return true
		}
	}
	return false
}

// GetAsFloat returns the value associated with the specified tag (key) as a float64.
// Note that this does not sanity check to ensure the tag exists first.
// If the tag doesn't exist, it will return 0.
// Note that this does not sanity check to ensure the tag is a float64 first.
func (tags *Tags) GetAsFloat(tagName string) float64 {
	if !tags.Has(tagName) {
		return 0
	}
	return tags.tags[tagName].(float64)
}

// IsInt returns true if the value associated with the specified tag is a int. If the
// tag doesn't exist or the tag's value isn't an int type, this returns false.
func (tags *Tags) IsInt(tagName string) bool {
	if _, exists := tags.tags[tagName]; exists {
		if _, ok := tags.tags[tagName].(int); ok {
			return true
		}
	}
	return false
}

// GetAsInt returns the value associated with the specified tag (key) as a float.
// If the tag doesn't exist, it will return 0.
// Note that this does not sanity check to ensure the tag is an int first.
func (tags *Tags) GetAsInt(tagName string) int {
	if !tags.Has(tagName) {
		return 0
	}
	return tags.tags[tagName].(int)
}

// IsColor returns true if the value associated with the specified tag is a color. If the
// tag doesn't exist or the tag's value isn't a *Color type, this returns false.
func (tags *Tags) IsColor(tagName string) bool {
	if _, exists := tags.tags[tagName]; exists {
		if _, ok := tags.tags[tagName].(*Color); ok {
			return true
		}
	}
	return false
}

// GetAsColor returns the value associated with the specified tag (key) as a *Color.
// If the tag doesn't exist, it will return a new color set to opaque white.
// Note that this does not sanity check to ensure the tag is a Color first.
func (tags *Tags) GetAsColor(tagName string) *Color {
	if !tags.Has(tagName) {
		return NewColor(1, 1, 1, 1)
	}
	return tags.tags[tagName].(*Color)
}

// IsVector returns true if the value associated with the specified tag is a vector. If the
// tag doesn't exist or the tag's value isn't a vector.Vector, this returns false.
func (tags *Tags) IsVector(tagName string) bool {
	if _, exists := tags.tags[tagName]; exists {
		if _, ok := tags.tags[tagName].(vector.Vector); ok {
			return true
		}
	}
	return false
}

// GetAsVector returns the value associated with the specified tag (key) as a *Color.
// Note that this does not sanity check to ensure the tag exists first.
// If the tag doesn't exist, it will return {0, 0, 0}.
// Note that this does not sanity check to ensure the tag is a vector first.
func (tags *Tags) GetAsVector(tagName string) vector.Vector {
	if !tags.Has(tagName) {
		return vector.Vector{0, 0, 0}
	}
	return tags.tags[tagName].(vector.Vector)
}

// Node represents a minimal struct that fully implements the Node interface. Model and Camera embed Node
// into their structs to automatically easily implement Node.
type Node struct {
	name                  string
	position              vector.Vector
	scale                 vector.Vector
	rotation              Matrix4
	originalLocalPosition vector.Vector
	visible               bool
	data                  interface{} // A place to store a pointer to something if you need it
	children              []INode
	parent                INode
	cachedTransform       Matrix4
	isTransformDirty      bool
	tags                  *Tags // Tags is an unordered set of string tags, representing a means of identifying Nodes.
	animationPlayer       *AnimationPlayer
	inverseBindMatrix     Matrix4 // Specifically for bones in an armature used for animating skinned meshes
	isBone                bool
	boneInfluence         Matrix4
	library               *Library // The Library this Node was instantiated from (nil if it wasn't instantiated with a library at all)
	scene                 *Scene
	onTransformUpdate     func()
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
		// We set this just in case we call a transform property getter before setting it and caching anything
		cachedTransform:       NewMatrix4(),
		originalLocalPosition: vector.Vector{0, 0, 0},
	}

	nb.animationPlayer = NewAnimationPlayer(nb)

	return nb
}

func (node *Node) setOriginalLocalPosition(position vector.Vector) {
	node.originalLocalPosition = position
}

// Name returns the object's name.
func (node *Node) Name() string {
	return node.name
}

// SetName sets the object's name.
func (node *Node) SetName(name string) {
	node.name = name
}

// Type returns the NodeType for this object.
func (node *Node) Type() NodeType {
	return NodeTypeNode
}

// Library returns the Library from which this Node was instantiated. If it was created through code, this will be nil.
func (node *Node) Library() *Library {
	return node.library
}

func (node *Node) setLibrary(library *Library) {
	node.library = library
}

// Clone returns a new Node.
func (node *Node) Clone() INode {
	newNode := NewNode(node.name)
	newNode.position = node.position.Clone()
	newNode.scale = node.scale.Clone()
	newNode.rotation = node.rotation.Clone()
	newNode.visible = node.visible
	newNode.data = node.data
	newNode.isTransformDirty = true
	newNode.tags = node.tags.Clone()
	newNode.animationPlayer = node.animationPlayer.Clone()
	newNode.library = node.library

	if node.animationPlayer.RootNode == node {
		newNode.animationPlayer.SetRoot(newNode)
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

	if node.isBone {
		node.boneInfluence = node.inverseBindMatrix.Mult(transform)
	}

	if node.onTransformUpdate != nil {
		node.onTransformUpdate()
	}

	// We want to call child.Transform() here to ensure the children also rebuild their transforms as necessary; otherwise,
	// children (i.e. BoundingAABBs) may not be rotating along with their owning Nodes (as they don't get rendered).
	for _, child := range node.children {
		child.Transform()
	}

	return transform

}

// SetWorldTransform sets the Node's global (world) transform to the full 4x4 transformation matrix provided.
func (node *Node) SetWorldTransform(transform Matrix4) {
	position, scale, rotationMatrix := transform.Decompose()
	node.SetWorldPositionVec(position)
	node.SetWorldScaleVec(scale)
	node.SetWorldRotation(rotationMatrix)
}

// dirtyTransform sets this Node and all recursive children's isTransformDirty flags to be true, indicating that they need to be
// rebuilt. This should be called when modifying the transformation properties (position, scale, rotation) of the Node.
func (node *Node) dirtyTransform() {

	for _, child := range node.ChildrenRecursive() {
		child.dirtyTransform()
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
	return node.position.Clone()
}

// ResetLocalTransform resets the local transform properties (position, scale, and rotation) for the Node. This can be useful because
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

// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalPosition(x, y, z float64) {
	node.position[0] = x
	node.position[1] = y
	node.position[2] = z
	node.dirtyTransform()
}

// SetLocalPositionVec sets the object's local position (position relative to its parent). If this object has no parent, the position should be
// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalPositionVec(position vector.Vector) {
	node.SetLocalPosition(position[0], position[1], position[2])
}

// SetWorldPositionVec sets the object's world position (position relative to the world origin point of {0, 0, 0}).
// position needs to be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetWorldPositionVec(position vector.Vector) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		pr := parentRot.Transposed().MultVec(position.Sub(parentPos))
		pr[0] /= parentScale[0]
		pr[1] /= parentScale[1]
		pr[2] /= parentScale[2]

		node.position = pr

	} else {
		node.position[0] = position[0]
		node.position[1] = position[1]
		node.position[2] = position[2]
	}

	node.dirtyTransform()

}

// SetWorldPosition sets the object's world position (position relative to the world origin point of {0, 0, 0}).
func (node *Node) SetWorldPosition(x, y, z float64) {
	node.SetWorldPositionVec(vector.Vector{x, y, z})
}

// LocalScale returns the object's local scale (scale relative to its parent). If this object has no parent, the scale will be absolute.
func (node *Node) LocalScale() vector.Vector {
	return node.scale.Clone()
}

// SetLocalScaleVec sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
// scale should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalScaleVec(scale vector.Vector) {
	node.SetLocalScale(scale[0], scale[1], scale[2])
}

// SetLocalScale sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
func (node *Node) SetLocalScale(w, h, d float64) {
	node.scale[0] = w
	node.scale[1] = h
	node.scale[2] = d
	node.dirtyTransform()
}

// WorldScale returns the object's absolute world scale as a 3D vector (i.e. X, Y, and Z components). Note that this is a bit slow as it
// requires decomposing the node's world transform, so you want to use node.LocalScale() if you can and performacne is a concern.
func (node *Node) WorldScale() vector.Vector {
	_, scale, _ := node.Transform().Decompose()
	return scale
}

// SetWorldScaleVec sets the object's absolute world scale. scale should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetWorldScaleVec(scale vector.Vector) {
	node.SetWorldScale(scale[0], scale[1], scale[2])
}

// SetWorldScale sets the object's absolute world scale.
func (node *Node) SetWorldScale(w, h, d float64) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		_, parentScale, _ := parentTransform.Decompose()

		node.scale = vector.Vector{
			w / parentScale[0],
			h / parentScale[1],
			d / parentScale[2],
		}

	} else {
		node.scale[0] = w
		node.scale[1] = h
		node.scale[2] = d
	}

	node.dirtyTransform()

}

// LocalRotation returns the object's local rotation Matrix4.
func (node *Node) LocalRotation() Matrix4 {
	return node.rotation.Clone()
}

// SetLocalRotation sets the object's local rotation Matrix4 (relative to any parent).
func (node *Node) SetLocalRotation(rotation Matrix4) {
	node.rotation.Set(rotation)
	node.dirtyTransform()
}

// WorldRotation returns an absolute rotation Matrix4 representing the object's rotation. Note that this is a bit slow as it
// requires decomposing the node's world transform, so you want to use node.LocalRotation() if you can and performacne is a concern.
func (node *Node) WorldRotation() Matrix4 {
	_, _, rotation := node.Transform().Decompose()
	return rotation
}

// SetWorldRotation sets an object's rotation to the provided rotation Matrix4.
func (node *Node) SetWorldRotation(rotation Matrix4) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		_, _, parentRot := parentTransform.Decompose()
		node.rotation.Set(parentRot.Transposed().Mult(rotation))

	} else {
		node.rotation.Set(rotation)
	}

	node.dirtyTransform()
}

// Move moves a Node in local space by the x, y, and z values provided.
func (node *Node) Move(x, y, z float64) {
	if x == 0 && y == 0 && z == 0 {
		return
	}
	node.position[0] += x
	node.position[1] += y
	node.position[2] += z
	node.dirtyTransform()
}

// MoveVec moves a Node in local space using the vector provided.
func (node *Node) MoveVec(vec vector.Vector) {
	node.Move(vec[0], vec[1], vec[2])
}

// Rotate rotates a Node locally on a vector with the given axis, by the angle provided in radians.
func (node *Node) Rotate(x, y, z, angle float64) {
	if x == 0 && y == 0 && z == 0 {
		return
	}
	localRot := node.LocalRotation()
	localRot = localRot.Rotated(x, y, z, angle)
	node.SetLocalRotation(localRot)
}

// RotateVec rotates a Node locally on the given vector, by the angle provided in radians.
func (node *Node) RotateVec(vec vector.Vector, angle float64) {
	node.Rotate(vec[0], vec[1], vec[2], angle)
}

// Grow scales the object additively using the x, y, and z arguments provided (i.e. calling
// Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
func (node *Node) Grow(x, y, z float64) {
	if x == 0 && y == 0 && z == 0 {
		return
	}
	scale := node.LocalScale()
	scale[0] += x
	scale[1] += y
	scale[2] += z
	node.SetLocalScale(scale[0], scale[1], scale[2])
}

// GrowVec scales the object additively using the scaling vector provided (i.e. calling
// Node.GrowVec(vector.Vector{1, 0, 0}) will scale it +1 on the X-axis).
func (node *Node) GrowVec(vec vector.Vector) {
	node.Grow(vec[0], vec[1], vec[2])
}

// Parent returns the Node's parent. If the Node has no parent, this will return nil.
func (node *Node) Parent() INode {
	return node.parent
}

// setParent sets the Node's parent.
func (node *Node) setParent(parent INode) {
	node.parent = parent
}

// Scene looks for the Node's parents recursively to return what scene it exists in.
// If the node is not within a tree (i.e. unparented), this will return nil.
func (node *Node) Scene() *Scene {
	root := node.Root()
	if root != nil {
		return root.(*Node).scene
	}
	return nil
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
func (node *Node) Children() NodeFilter {
	return append(make(NodeFilter, 0, len(node.children)), node.children...)
}

// ChildrenRecursive() returns the Node's recursive children (i.e. children, grandchildren, etc)
// as a NodeFilter.
func (node *Node) ChildrenRecursive() NodeFilter {
	out := node.Children()

	for _, child := range node.children {
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
	if recursive && node.visible != visible {
		for _, child := range node.ChildrenRecursive() {
			child.SetVisible(visible, true)
		}
	}
	node.visible = visible
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

		prefix := ""

		nodeType := node.Type()

		if nodeType.Is(NodeTypeModel) {
			prefix = "MODEL"
		} else if nodeType.Is(NodeTypeCamera) {
			prefix = "CAM"
		} else if nodeType.Is(NodeTypePath) {
			prefix = "PATH"
		} else if nodeType.Is(NodeTypeGrid) {
			prefix = "GRID"
		} else if nodeType.Is(NodeTypeGridPoint) {
			prefix = "GPOINT"
		} else if nodeType.Is(NodeTypeAmbientLight) {
			prefix = "AMB"
		} else if nodeType.Is(NodeTypeDirectionalLight) {
			prefix = "DIR"
		} else if nodeType.Is(NodeTypePointLight) {
			prefix = "POINT"
		} else if nodeType.Is(NodeTypeCubeLight) {
			prefix = "CUBE"
		} else if nodeType.Is(NodeTypeBoundingSphere) {
			prefix = "BS"
		} else if nodeType.Is(NodeTypeBoundingAABB) {
			prefix = "AABB"
		} else if nodeType.Is(NodeTypeBoundingCapsule) {
			prefix = "CAP"
		} else if nodeType.Is(NodeTypeBoundingTriangles) {
			prefix = "TRI"
		} else {
			prefix = "NODE"
		}

		str := ""

		for i := 0; i < level; i++ {
			str += "    "
		}

		wp := node.LocalPosition()
		wpStr := "[" + strconv.FormatFloat(wp[0], 'f', -1, 64) + ", " + strconv.FormatFloat(wp[1], 'f', -1, 64) + ", " + strconv.FormatFloat(wp[2], 'f', -1, 64) + "]"

		str += "\\-: [" + prefix + "] " + node.Name() + " : " + wpStr + "\n"

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
// Since Get uses forward slashes as path separation, it would be good to avoid using forward slashes in your Node names. Also note that Get()
// trims the extra spaces from the beginning and end of Node Names, so avoid using spaces at the beginning or end of your Nodes' names.
func (node *Node) Get(path string) INode {

	var search func(node INode) INode

	split := []string{}

	for _, s := range strings.Split(path, `/`) {
		if len(strings.TrimSpace(s)) > 0 {
			split = append(split, s)
		}
	}

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

// Root returns the root node in the scene by recursively traversing this node's hierarchy of
// parents upwards. If, instead, the node this is called on is the root (and so has no parents), this function
// returns the node itself. If the node is not the root and it has no path to the scene root, this
// function returns nil.
func (node *Node) Root() INode {

	if node.parent == nil {
		if node.scene != nil {
			return node
		}
		return nil
	}

	return node.parent.Root()

}

// IsBone returns if the Node is a "bone" (a node that was a part of an armature and so can play animations back to influence a skinned mesh).
func (node *Node) IsBone() bool {
	return node.isBone
}

// // IsRootBone returns if the Node SHOULD be the root of an Armature (a Node that was the base of an armature).
// func (node *Node) IsRootBone() bool {
// 	return node.IsBone() && (node.parent == nil || !node.parent.IsBone())
// }

// AnimationPlayer returns the object's animation player - every object has an AnimationPlayer by default.
func (node *Node) AnimationPlayer() *AnimationPlayer {
	return node.animationPlayer
}
