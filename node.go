package tetra3d

import (
	"strconv"
	"strings"
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
	// ID returns the object's unique ID.
	ID() uint64
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

	// ReindexChild moves the child in the calling Node's children slice to the specified newPosition.
	// The function returns the old index where the child Node was, or -1 if it wasn't a child of the calling Node.
	// The newPosition is clamped to the size of the node's children slice.
	ReindexChild(child INode, newPosition int) int

	// Index returns the index of the Node in its parent's children list.
	// If the node doesn't have a parent, its index will be -1.
	Index() int

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
	LocalPosition() Vector
	// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
	// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
	SetLocalPositionVec(position Vector)
	SetLocalPosition(x, y, z float64)
	// LocalScale returns the object's local scale (scale relative to its parent). If this object has no parent, the scale will be absolute.
	LocalScale() Vector
	// SetLocalScale sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
	// scale should be a 3D vector (i.e. X, Y, and Z components).
	SetLocalScaleVec(scale Vector)
	SetLocalScale(w, h, d float64)

	// WorldRotation returns an absolute rotation Matrix4 representing the object's rotation.
	WorldRotation() Matrix4
	// SetWorldRotation sets an object's global, world rotation to the provided rotation Matrix4.
	SetWorldRotation(rotation Matrix4)
	// WorldPosition returns the node's world position, taking into account its parenting hierarchy.
	WorldPosition() Vector
	// SetWorldPositionVec sets the world position of the given object using the provided position vector.
	// Note that this isn't as performant as setting the position locally.
	SetWorldPositionVec(position Vector)
	// SetWorldPosition sets the world position of the given object using the provided position arguments.
	// Note that this isn't as performant as setting the position locally.
	SetWorldPosition(x, y, z float64)
	// SetWorldX sets the x component of the Node's world position.
	SetWorldX(x float64)
	// SetWorldY sets the y component of the Node's world position.
	SetWorldY(x float64)
	// SetWorldZ sets the z component of the Node's world position.
	SetWorldZ(x float64)
	// WorldScale returns the object's absolute world scale as a 3D vector (i.e. X, Y, and Z components).
	WorldScale() Vector
	// SetWorldScaleVec sets the object's absolute world scale. scale should be a 3D vector (i.e. X, Y, and Z components).
	SetWorldScaleVec(scale Vector)

	// Move moves a Node in local space by the x, y, and z values provided.
	Move(x, y, z float64)
	// MoveVec moves a Node in local space using the vector provided.
	MoveVec(moveVec Vector)
	// Rotate rotates a Node on its local orientation on a vector composed of the given x, y, and z values, by the angle provided in radians.
	Rotate(x, y, z, angle float64)
	// RotateVec rotates a Node on its local orientation on the given vector, by the angle provided in radians.
	RotateVec(vec Vector, angle float64)
	// Grow scales the object additively (i.e. calling Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
	Grow(x, y, z float64)
	// GrowVec scales the object additively (i.e. calling Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
	GrowVec(vec Vector)

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
	// This is a useful function to debug the layout of a node tree, for example.
	HierarchyAsString() string

	// Path returns a string indicating the hierarchical path to get this Node from the root. The path returned will be absolute, such that
	// passing it to Get() called on the scene root node will return this node. The path returned will not contain the root node's name ("Root").
	Path() string

	// Properties returns this object's game Properties struct.
	Properties() *Properties

	// IsBone returns if the Node is a "bone" (a node that was a part of an armature and so can play animations back to influence a skinned mesh).
	IsBone() bool
	// IsRootBone() bool

	// AnimationPlayer returns the object's animation player - every object has an AnimationPlayer by default.
	AnimationPlayer() *AnimationPlayer

	setOriginalLocalPosition(Vector)
}

var nodeID uint64 = 0

// Node represents a minimal struct that fully implements the Node interface. Model and Camera embed Node
// into their structs to automatically easily implement Node.

type Node struct {
	id                    uint64 // Unique ID for this node
	name                  string
	position              Vector
	scale                 Vector
	rotation              Matrix4
	originalLocalPosition Vector
	visible               bool
	data                  interface{} // A place to store a pointer to something if you need it
	children              []INode
	parent                INode
	cachedTransform       Matrix4
	isTransformDirty      bool
	props                 *Properties // Tags is an unordered set of string tags, representing a means of identifying Nodes.
	animationPlayer       *AnimationPlayer
	inverseBindMatrix     Matrix4 // Specifically for bones in an armature used for animating skinned meshes
	isBone                bool
	collectionObjects     []INode // Returns if the node is a collection instance
	boneInfluence         Matrix4
	library               *Library // The Library this Node was instantiated from (nil if it wasn't instantiated with a library at all)
	scene                 *Scene
	onTransformUpdate     func()
}

// NewNode returns a new Node.
func NewNode(name string) *Node {

	nb := &Node{
		id:   nodeID,
		name: name,
		// position:         NewVectorZero(),
		scale:            Vector{1, 1, 1, 0},
		rotation:         NewMatrix4(),
		children:         []INode{},
		visible:          true,
		isTransformDirty: true,
		props:            NewProperties(),
		// We set this just in case we call a transform property getter before setting it and caching anything
		cachedTransform: NewMatrix4(),
		// originalLocalPosition: NewVectorZero(),
	}

	nodeID++

	nb.animationPlayer = NewAnimationPlayer(nb)

	return nb
}

// ID returns the object's unique ID.
func (node *Node) ID() uint64 {
	return node.id
}

func (node *Node) setOriginalLocalPosition(position Vector) {
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
	newNode.position = node.position
	newNode.scale = node.scale
	newNode.rotation = node.rotation.Clone()
	newNode.visible = node.visible
	newNode.data = node.data

	newNode.props = node.props.Clone()
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

	newNode.dirtyTransform()

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

	transform := NewMatrix4Scale(node.scale.X, node.scale.Y, node.scale.Z)
	transform = transform.Mult(node.rotation)
	transform = transform.Mult(NewMatrix4Translate(node.position.X, node.position.Y, node.position.Z))

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
func (node *Node) LocalPosition() Vector {
	return node.position
}

// ResetLocalTransform resets the local transform properties (position, scale, and rotation) for the Node. This can be useful because
// by default, when you parent one Node to another, the local transform properties (position, scale, and rotation) are altered to keep the
// object in the same absolute location, even though the origin changes.
func (node *Node) ResetLocalTransform() {
	node.position.X = 0
	node.position.Y = 0
	node.position.Z = 0
	node.scale.X = 1
	node.scale.Y = 1
	node.scale.Z = 1
	node.rotation = NewMatrix4()
	node.dirtyTransform()
}

// WorldPosition returns a 3D Vector consisting of the object's world position (position relative to the world origin point of {0, 0, 0}).
func (node *Node) WorldPosition() Vector {
	position := node.Transform().Row(3) // We don't want to have to decompose if we don't have to
	return position
}

// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalPosition(x, y, z float64) {
	node.position.X = x
	node.position.Y = y
	node.position.Z = z
	node.dirtyTransform()
}

// SetLocalPositionVec sets the object's local position (position relative to its parent). If this object has no parent, the position should be
// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalPositionVec(position Vector) {
	node.SetLocalPosition(position.X, position.Y, position.Z)
}

// SetWorldPositionVec sets the object's world position (position relative to the world origin point of {0, 0, 0}).
// position needs to be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetWorldPositionVec(position Vector) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		pr := parentRot.Transposed().MultVec(position.Sub(parentPos))
		pr.X /= parentScale.X
		pr.Y /= parentScale.Y
		pr.Z /= parentScale.Z

		node.position = pr

	} else {
		node.position.X = position.X
		node.position.Y = position.Y
		node.position.Z = position.Z
	}

	node.dirtyTransform()

}

// SetWorldPosition sets the object's world position (position relative to the world origin point of {0, 0, 0}).
func (node *Node) SetWorldPosition(x, y, z float64) {
	node.SetWorldPositionVec(Vector{x, y, z, 0})
}

// SetWorldX sets the X component of the object's world position.
func (node *Node) SetWorldX(x float64) {
	v := node.WorldPosition()
	v.X = x
	node.SetWorldPositionVec(v)
}

// SetWorldY sets the Y component of the object's world position.
func (node *Node) SetWorldY(y float64) {
	v := node.WorldPosition()
	v.Y = y
	node.SetWorldPositionVec(v)
}

// SetWorldZ sets the Z component of the object's world position.
func (node *Node) SetWorldZ(z float64) {
	v := node.WorldPosition()
	v.Z = z
	node.SetWorldPositionVec(v)
}

// LocalScale returns the object's local scale (scale relative to its parent). If this object has no parent, the scale will be absolute.
func (node *Node) LocalScale() Vector {
	return node.scale
}

// SetLocalScaleVec sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
// scale should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalScaleVec(scale Vector) {
	node.SetLocalScale(scale.X, scale.Y, scale.Z)
}

// SetLocalScale sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
func (node *Node) SetLocalScale(w, h, d float64) {
	node.scale.X = w
	node.scale.Y = h
	node.scale.Z = d
	node.dirtyTransform()
}

// WorldScale returns the object's absolute world scale as a 3D vector (i.e. X, Y, and Z components). Note that this is a bit slow as it
// requires decomposing the node's world transform, so you want to use node.LocalScale() if you can and performacne is a concern.
func (node *Node) WorldScale() Vector {
	_, scale, _ := node.Transform().Decompose()
	return scale
}

// SetWorldScaleVec sets the object's absolute world scale. scale should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetWorldScaleVec(scale Vector) {
	node.SetWorldScale(scale.X, scale.Y, scale.Z)
}

// SetWorldScale sets the object's absolute world scale.
func (node *Node) SetWorldScale(w, h, d float64) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		_, parentScale, _ := parentTransform.Decompose()

		node.scale = Vector{
			w / parentScale.X,
			h / parentScale.Y,
			d / parentScale.Z,
			0,
		}

	} else {
		node.scale.X = w
		node.scale.Y = h
		node.scale.Z = d
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
	node.position.X += x
	node.position.Y += y
	node.position.Z += z
	node.dirtyTransform()
}

// MoveVec moves a Node in local space using the vector provided.
func (node *Node) MoveVec(vec Vector) {
	node.Move(vec.X, vec.Y, vec.Z)
}

// Rotate rotates a Node on its local orientation on a vector composed of the given x, y, and z values, by the angle provided in radians.
func (node *Node) Rotate(x, y, z, angle float64) {
	if x == 0 && y == 0 && z == 0 {
		return
	}
	localRot := node.LocalRotation()
	localRot = localRot.Rotated(x, y, z, angle)
	node.SetLocalRotation(localRot)
}

// RotateVec rotates a Node on its local orientation on the given vector, by the angle provided in radians.
func (node *Node) RotateVec(vec Vector, angle float64) {
	node.Rotate(vec.X, vec.Y, vec.Z, angle)
}

// Grow scales the object additively using the x, y, and z arguments provided (i.e. calling
// Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
func (node *Node) Grow(x, y, z float64) {
	if x == 0 && y == 0 && z == 0 {
		return
	}
	scale := node.LocalScale()
	scale.X += x
	scale.Y += y
	scale.Z += z
	node.SetLocalScale(scale.X, scale.Y, scale.Z)
}

// GrowVec scales the object additively using the scaling vector provided (i.e. calling
// Node.GrowVec(Vector{1, 0, 0}) will scale it +1 on the X-axis).
func (node *Node) GrowVec(vec Vector) {
	node.Grow(vec.X, vec.Y, vec.Z)
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
		child.dirtyTransform()
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
				child.dirtyTransform()
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

// ReindexChild moves the child in the calling Node's children slice to the specified newPosition.
// The function returns the old index where the child Node was, or -1 if it wasn't a child of the calling Node.
// The newPosition is clamped to the size of the node's children slice.
func (node *Node) ReindexChild(child INode, newPosition int) int {

	if oldIndex := child.Index(); oldIndex >= 0 {

		if newPosition < 0 {
			newPosition = 0
		} else if newPosition > len(node.children)-1 {
			newPosition = len(node.children) - 1
		}

		node.children = append(node.children[:oldIndex], node.children[oldIndex+1:]...)

		node.children = append(node.children, nil)
		copy(node.children[newPosition+1:], node.children[newPosition:])
		node.children[newPosition] = child

		return oldIndex
	}

	return -1

}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (node *Node) Index() int {
	if node.parent != nil {
		return node.parent.Children().Index(node)
	}
	return -1
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
func (node *Node) Properties() *Properties {
	return node.props
}

// HierarchyAsString returns a string displaying the hierarchy of this Node, and all recursive children.
// This is a useful function to debug the layout of a node tree, for example.
// All Nodes except for the top-level Node will show their type by means of a prefix ("MODEL" for Models, for example).
// All Nodes listed in the hierarchy will also show their world positions, truncated to the first 2 decimals.

func (node *Node) HierarchyAsString() string {

	var printNode func(node INode, level int) string

	printNode = func(node INode, level int) string {

		prefix := ""

		// We do this because calling Node.HierarchyAsString() will always return the base Node's
		// type, which is NodeTypeNode, unless we override this function, essentially, for each
		// "more specific type". To avoid doing this, I'm just going to have the first level node
		// look like : [-] .
		if level == 0 {
			prefix = "ROOT"
		} else {

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

		}

		str := ""

		if node.Parent() != nil {
			for i := 0; i < level; i++ {
				str += "    |"
			}
			str += "\n"
		}

		for i := 0; i < level; i++ {
			str += "    |"
		}

		wp := node.WorldPosition()
		floatTruncation := 2
		wpStr := "[" + strconv.FormatFloat(wp.X, 'f', floatTruncation, 64) + ", " + strconv.FormatFloat(wp.Y, 'f', floatTruncation, 64) + ", " + strconv.FormatFloat(wp.Z, 'f', floatTruncation, 64) + "]"

		if level > 0 {
			str += "-"
		}
		str += " [" + prefix + "] " + node.Name() + " : " + wpStr + "\n"

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
