package tetra3d

import (
	"fmt"
	"strconv"
	"strings"
)

type SectorType int

const (
	SectorTypeObject SectorType = iota
	SectorTypeSector
	SectorTypeStandalone
)

// NodeType represents a Node's type. Node types are categorized, and can be said to extend or "be of" more general types.
// For example, a BoundingSphere has a type of NodeTypeBoundingSphere. That type can also be said to be NodeTypeBoundingObject
// (because it is a bounding object). However, it is not of type NodeTypeBoundingTriangles, as that is a different category.
type NodeType string

const (
	NodeTypeNode      NodeType = "NodeNode"       // NodeTypeNode represents specifically a node
	NodeTypeModel     NodeType = "NodeModel"      // NodeTypeModel represents specifically a Model
	NodeTypeCamera    NodeType = "NodeCamera"     // NodeTypeCamera represents specifically a Camera
	NodeTypePath      NodeType = "NodePath"       // NodeTypePath represents specifically a Path
	NodeTypeGrid      NodeType = "NodeGrid"       // NodeTypeGrid represents specifically a Grid
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
	SetData(data any)
	// Data returns a pointer to user-customizeable data that could be usefully stored on this node.
	Data() any
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
	getOwner() INode
	// IsDescendantOf returns if a Node is a descendant child of a parent Node.
	IsDescendantOf(parent INode) bool
	// Scene looks for the Node's parents recursively to return what scene it exists in.
	// If the node is not within a tree (i.e. unparented), this will return nil.
	Scene() *Scene
	// Root returns the root node in this tree by recursively traversing this node's hierarchy of
	// parents upwards.
	Root() *Node
	InSceneTree() bool // Returns if this node is in the scene tree.
	setCachedSceneRoot(root *Node)
	cachedSceneRoot() *Node

	// ReindexChild moves the child in the calling Node's children slice to the specified newPosition.
	// The function returns the old index where the child Node was, or -1 if it wasn't a child of the calling Node.
	// The newPosition is clamped to the size of the node's children slice.
	ReindexChild(child INode, newPosition int) int

	// Index returns the index of the Node in its parent's children list.
	// If the node doesn't have a parent, its index will be -1.
	Index() int

	// Children() returns the Node's children as an INode.
	Children() []INode

	// ForEachChild() runs a callback for each child in the Node's children set.
	// If the callback returns false, then the execution stops with the current child.
	ForEachChild(func(child INode, index, size int) bool)

	// SearchTree() returns a NodeFilter to search the given Node's hierarchical tree.
	SearchTree() *NodeFilter

	// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
	// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
	AddChildren(...INode)
	// RemoveChildren removes the provided children from this object.
	RemoveChildren(...INode)

	// updateLocalTransform(newParent INode)
	dirtyTransform()

	// ClearLocalTransform clears the local transform properties (position, scale, and rotation) for the Node, reverting it to essentially an
	// identity matrix (0, 0, 0 for position, 1, 1, 1 for scale, and an identity Matrix4 for rotation, indicating no rotation).
	// This can be useful because by default, when you parent one Node to another, the local transform properties (position,
	// scale, and rotation) are altered to keep the object in the same absolute location, even though the origin changes.
	ClearLocalTransform()

	// ResetWorldTransform resets the local transform properties (position, scale, and rotation) for the Node to the original transform when
	// the Node was first created / cloned / instantiated in the Scene.
	ResetWorldTransform()

	// ResetWorldPosition resets the Node's local position to the value the Node had when
	// it was first instantiated in the Scene or cloned.
	ResetWorldPosition()

	// ResetWorldScale resets the Node's local scale to the value the Node had when
	// it was first instantiated in the Scene or cloned.
	ResetWorldScale()

	// ResetWorldRotation resets the Node's local rotation to the value the Node had when
	// it was first instantiated in the Scene or cloned.
	ResetWorldRotation()

	setOriginalTransform()

	// SetWorldTransform sets the Node's global (world) transform to the full 4x4 transformation matrix provided.
	SetWorldTransform(transform Matrix4)

	// LocalRotation returns the object's local rotation Matrix4.
	LocalRotation() Matrix4
	// SetLocalRotation sets the object's local rotation Matrix4 (relative to any parent).
	SetLocalRotation(rotation Matrix4)

	// LocalPosition returns the object's local position as a Vector.
	LocalPosition() Vector3
	// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
	// relative to world origin (0, 0, 0).
	SetLocalPositionVec(position Vector3)
	// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
	// relative to world origin (0, 0, 0).
	SetLocalPosition(x, y, z float32)

	// LocalScale returns the object's local scale (scale relative to its parent). If this object has no parent, the scale will be absolute.
	LocalScale() Vector3
	// SetLocalScale sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
	// scale should be a 3D vector (i.e. X, Y, and Z components).
	SetLocalScaleVec(scale Vector3)
	SetLocalScale(w, h, d float32)

	// WorldRotation returns an absolute rotation Matrix4 representing the object's rotation.
	WorldRotation() Matrix4
	// SetWorldRotation sets an object's global, world rotation to the provided rotation Matrix4.
	SetWorldRotation(rotation Matrix4)
	// WorldPosition returns the node's world position, taking into account its parenting hierarchy.
	WorldPosition() Vector3
	// SetWorldPositionVec sets the world position of the given object using the provided position vector.
	// Note that this isn't as performant as setting the position locally.
	SetWorldPositionVec(position Vector3)
	// SetWorldPosition sets the world position of the given object using the provided position arguments.
	// Note that this isn't as performant as setting the position locally.
	SetWorldPosition(x, y, z float32)
	// SetWorldX sets the x component of the Node's world position.
	SetWorldX(x float32)
	// SetWorldY sets the y component of the Node's world position.
	SetWorldY(x float32)
	// SetWorldZ sets the z component of the Node's world position.
	SetWorldZ(x float32)
	// WorldScale returns the object's absolute world scale as a 3D vector (i.e. X, Y, and Z components).
	WorldScale() Vector3
	// SetWorldScaleVec sets the object's absolute world scale. scale should be a 3D vector (i.e. X, Y, and Z components).
	SetWorldScaleVec(scale Vector3)

	// Move moves a Node in local space by the x, y, and z values provided.
	Move(x, y, z float32)
	// MoveVec moves a Node in local space using the vector provided.
	MoveVec(moveVec Vector3)
	// Rotate rotates a Node on its local orientation on a vector composed of the given x, y, and z values, by the angle provided in radians.
	Rotate(x, y, z, angle float32)
	// RotateVec rotates a Node on its local orientation on the given vector, by the angle provided in radians.
	RotateVec(vec Vector3, angle float32)
	// Grow scales the object additively (i.e. calling Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
	Grow(x, y, z float32)
	// GrowVec scales the object additively (i.e. calling Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
	GrowVec(vec Vector3)

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

	// FindNode searches a node's hierarchy using a string to find the specified Node.
	FindNode(nodeName string) INode

	// HierarchyAsString returns a string displaying the hierarchy of this Node, and all recursive children.
	// This is a useful function to debug the layout of a node tree, for example.
	HierarchyAsString() string

	// Path returns a string indicating the hierarchical path to get this Node from the root. The path returned will be absolute, such that
	// passing it to Get() called on the scene root node will return this node. The path returned will not contain the root node's name ("Root").
	Path() string

	// Properties returns this object's game Properties struct.
	Properties() Properties

	// IsBone returns if the Node is a "bone" (a node that was a part of an armature and so can play animations back to influence a skinned mesh).
	IsBone() bool
	// IsRootBone() bool

	// AnimationPlayer returns the object's animation player - every object has an AnimationPlayer by default.
	AnimationPlayer() *AnimationPlayer

	// Sector returns the Sector this Node is in.
	Sector() *Sector
	sectorHierarchy() *Sector
	isInVisibleSector(sectorsModels []*Model) bool

	SetSectorType(sectorType SectorType)
	SectorType() SectorType

	// DistanceTo returns the distance between the given Nodes' centers.
	// Quick syntactic sugar for Node.WorldPosition().Distance(otherNode.WorldPosition()).
	DistanceTo(otherNode INode) float32

	// DistanceSquared returns the squared distance between the given Nodes' centers.
	// Quick syntactic sugar for Node.WorldPosition().DistanceSquared(otherNode.WorldPosition()).
	DistanceSquaredTo(otherNode INode) float32

	// VectorTo returns a vector from one Node to another.
	// Quick syntactic sugar for other.WorldPosition().Sub(node.WorldPosition()).
	VectorTo(otherNode INode) Vector3

	// Callbacks returns a Node's callbacks object. This object represents the callbacks that a Node has access to when events happen.
	Callbacks() *Callbacks

	getRunCallbacks() bool
	setRunCallbacks(bool)
}

var nodeID uint64 = 0

// Node represents a minimal struct that fully implements the Node interface. Model and Camera embed Node
// into their structs to automatically easily implement Node.

type Node struct {
	id                uint64 // Unique ID for this node
	owner             INode  // The owner; nil if this is a Node, set to the "owning" node type otherwise (e.g. node.owner = nil, model.owner (or model.Node.owner) = model)
	name              string
	position          Vector3
	scale             Vector3
	rotation          Matrix4
	originalTransform Matrix4
	visible           bool
	data              any // A place to store a pointer to something if you need it
	children          []INode
	parent            INode
	cachedTransform   Matrix4
	isTransformDirty  bool
	props             Properties // Properties is an unordered set of properties, representing a means of identifying and setting game properties on Nodes.
	animationPlayer   *AnimationPlayer
	inverseBindMatrix Matrix4 // Specifically for bones in an armature used for animating skinned meshes
	isBone            bool
	collectionObjects []INode // Returns if the node is a collection instance
	boneInfluence     Matrix4
	library           *Library // The Library this Node was instantiated from (nil if it wasn't instantiated with a library at all)
	scene             *Scene
	onTransformUpdate func()
	sectorType        SectorType

	cachedSceneRootNode *Node

	cachedSector *Sector

	runCallbacks bool
	callbacks    *Callbacks
}

// NewNode returns a new Node.
func NewNode(name string) *Node {

	nb := &Node{
		id:   nodeID,
		name: name,
		// position:         NewVectorZero(),
		scale:            Vector3{1, 1, 1},
		rotation:         NewMatrix4(),
		children:         []INode{},
		visible:          true,
		isTransformDirty: true,
		props:            NewProperties(),
		// We set this just in case we call a transform property getter before setting it and caching anything
		cachedTransform: NewMatrix4(),
		runCallbacks:    true,
		callbacks:       &Callbacks{},
		// originalLocalPosition: NewVectorZero(),
		// sectorType: NewBitMask(0 + 1 + 2 + 3 + 4 + 5 + 6 + 7),
	}

	nodeID++

	nb.animationPlayer = NewAnimationPlayer(nb)

	return nb
}

// Callbacks returns a Node's callbacks object. This object represents the callbacks that a Node has access to when events happen.
func (node *Node) Callbacks() *Callbacks {
	return node.callbacks
}

func (node *Node) getRunCallbacks() bool {
	return node.runCallbacks
}

func (node *Node) setRunCallbacks(run bool) {
	node.runCallbacks = run
}

// ID returns the object's unique ID.
func (node *Node) ID() uint64 {
	return node.id
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
	return node.clone(nil)
}

func (node *Node) clone(newOwner INode) INode {
	newNode := NewNode(node.name)
	newNode.owner = newOwner
	newNode.scene = node.scene
	newNode.position = node.position
	newNode.scale = node.scale
	newNode.rotation = node.rotation.Clone()
	newNode.setOriginalTransform()
	newNode.visible = node.visible
	newNode.data = node.data
	newNode.sectorType = node.sectorType
	newNode.cachedSector = node.cachedSector
	newNode.library = node.library
	newNode.runCallbacks = node.runCallbacks

	// Clone the callbacks as well.
	newCallbacks := *node.callbacks
	newNode.callbacks = &newCallbacks

	newNode.props = node.props.Clone()
	newNode.animationPlayer = node.animationPlayer.Clone()

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

	if newOwner == nil && newNode.Callbacks() != nil && newNode.Callbacks().OnClone != nil {
		newNode.Callbacks().OnClone(newNode)
	}

	return newNode
}

// SetData sets user-customizeable data that could be usefully stored on this node.
func (node *Node) SetData(data any) {
	node.data = data
}

// Data returns a pointer to user-customizeable data that could be usefully stored on this node.
func (node *Node) Data() any {
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

	// TODO: I think I could speed up this area considerably.
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

	for _, child := range node.SearchTree().INodes() {
		child.dirtyTransform()
	}

	node.isTransformDirty = true
	node.cachedSector = nil

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
func (node *Node) LocalPosition() Vector3 {
	return node.position
}

// ClearLocalTransform clears the local transform properties (position, scale, and rotation) for the Node, reverting it to essentially an
// identity matrix (0, 0, 0 for position, 1, 1, 1 for scale, and an identity Matrix4 for rotation, indicating no rotation).
// This can be useful because by default, when you parent one Node to another, the local transform properties (position,
// scale, and rotation) are altered to keep the object in the same absolute location, even though the origin changes.
func (node *Node) ClearLocalTransform() {
	node.position.X = 0
	node.position.Y = 0
	node.position.Z = 0
	node.scale.X = 1
	node.scale.Y = 1
	node.scale.Z = 1
	node.rotation = NewMatrix4()
	node.dirtyTransform()
}

// ResetWorldTransform resets the Node's world transform properties (position, scale, and rotation) for the Node to the original
// values when the Node was first instantiated in the Scene or cloned.
func (node *Node) ResetWorldTransform() {
	node.SetWorldTransform(node.originalTransform)
	node.dirtyTransform()
}

// ResetWorldPosition resets the Node's local position to the value the Node had when
// it was first instantiated in the Scene or cloned.
func (node *Node) ResetWorldPosition() {
	p, _, _ := node.originalTransform.Decompose()
	node.SetWorldPositionVec(p)
	node.dirtyTransform()
}

// ResetWorldScale resets the Node's local scale to the value the Node had when
// it was first instantiated in the Scene or cloned.
func (node *Node) ResetWorldScale() {
	_, s, _ := node.originalTransform.Decompose()
	node.SetWorldScaleVec(s)
	node.dirtyTransform()
}

// ResetWorldRotation resets the Node's local rotation to the value the Node had when
// it was first instantiated in the Scene or cloned.
func (node *Node) ResetWorldRotation() {
	_, _, r := node.originalTransform.Decompose()
	node.SetWorldRotation(r)
	node.dirtyTransform()
}

func (node *Node) setOriginalTransform() {
	node.originalTransform = node.Transform()
}

// WorldPosition returns a 3D Vector consisting of the object's world position (position relative to the world origin point of {0, 0, 0}).
func (node *Node) WorldPosition() Vector3 {
	position := node.Transform().Row(3) // We don't want to have to decompose if we don't have to
	return position.To3D()
}

// SetLocalPosition sets the object's local position (position relative to its parent). If this object has no parent, the position should be
// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalPosition(x, y, z float32) {
	node.position.X = x
	node.position.Y = y
	node.position.Z = z
	node.dirtyTransform()
}

// SetLocalPositionVec sets the object's local position (position relative to its parent). If this object has no parent, the position should be
// relative to world origin (0, 0, 0). position should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalPositionVec(position Vector3) {
	node.SetLocalPosition(position.X, position.Y, position.Z)
}

// SetWorldPositionVec sets the object's world position (position relative to the world origin point of {0, 0, 0}).
// position needs to be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetWorldPositionVec(position Vector3) {

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
func (node *Node) SetWorldPosition(x, y, z float32) {
	node.SetWorldPositionVec(Vector3{x, y, z})
}

// SetWorldX sets the X component of the object's world position.
func (node *Node) SetWorldX(x float32) {
	v := node.WorldPosition()
	v.X = x
	node.SetWorldPositionVec(v)
}

// SetWorldY sets the Y component of the object's world position.
func (node *Node) SetWorldY(y float32) {
	v := node.WorldPosition()
	v.Y = y
	node.SetWorldPositionVec(v)
}

// SetWorldZ sets the Z component of the object's world position.
func (node *Node) SetWorldZ(z float32) {
	v := node.WorldPosition()
	v.Z = z
	node.SetWorldPositionVec(v)
}

// LocalScale returns the object's local scale (scale relative to its parent). If this object has no parent, the scale will be absolute.
func (node *Node) LocalScale() Vector3 {
	return node.scale
}

// SetLocalScaleVec sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
// scale should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetLocalScaleVec(scale Vector3) {
	node.SetLocalScale(scale.X, scale.Y, scale.Z)
}

// SetLocalScale sets the object's local scale (scale relative to its parent). If this object has no parent, the scale would be absolute.
func (node *Node) SetLocalScale(w, h, d float32) {
	node.scale.X = w
	node.scale.Y = h
	node.scale.Z = d
	node.dirtyTransform()
}

// WorldScale returns the object's absolute world scale as a 3D vector (i.e. X, Y, and Z components). Note that this is a bit slow as it
// requires decomposing the node's world transform, so you want to use node.LocalScale() if you can and performacne is a concern.
func (node *Node) WorldScale() Vector3 {
	_, scale, _ := node.Transform().Decompose()
	return scale
}

// SetWorldScaleVec sets the object's absolute world scale. scale should be a 3D vector (i.e. X, Y, and Z components).
func (node *Node) SetWorldScaleVec(scale Vector3) {
	node.SetWorldScale(scale.X, scale.Y, scale.Z)
}

// SetWorldScale sets the object's absolute world scale.
func (node *Node) SetWorldScale(w, h, d float32) {

	if node.parent != nil {

		parentTransform := node.parent.Transform()
		_, parentScale, _ := parentTransform.Decompose()

		node.scale = Vector3{
			w / parentScale.X,
			h / parentScale.Y,
			d / parentScale.Z,
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
	if rotation.IsZero() {
		return
	}
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
func (node *Node) Move(x, y, z float32) {
	if x == 0 && y == 0 && z == 0 {
		return
	}
	node.position.X += x
	node.position.Y += y
	node.position.Z += z
	node.dirtyTransform()
}

// MoveVec moves a Node in local space using the vector provided.
func (node *Node) MoveVec(vec Vector3) {
	node.Move(vec.X, vec.Y, vec.Z)
}

// Rotate rotates a Node on its local orientation on a vector composed of the given x, y, and z values, by the angle provided in radians.
func (node *Node) Rotate(x, y, z, angle float32) {
	if x == 0 && y == 0 && z == 0 {
		return
	}
	localRot := node.LocalRotation()
	localRot = localRot.Rotated(x, y, z, angle)
	node.SetLocalRotation(localRot)
}

// RotateVec rotates a Node on its local orientation on the given vector, by the angle provided in radians.
func (node *Node) RotateVec(vec Vector3, angle float32) {
	node.Rotate(vec.X, vec.Y, vec.Z, angle)
}

// Grow scales the object additively using the x, y, and z arguments provided (i.e. calling
// Node.Grow(1, 0, 0) will scale it +1 on the X-axis).
func (node *Node) Grow(x, y, z float32) {
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
func (node *Node) GrowVec(vec Vector3) {
	node.Grow(vec.X, vec.Y, vec.Z)
}

// Parent returns the Node's parent. If the Node has no parent, this will return nil.
func (node *Node) Parent() INode {
	if node.parent != nil {
		return node.parent.getOwner()
	}
	return nil
}

// setParent sets the Node's parent.
func (node *Node) setParent(parent INode) {
	// fmt.Println(node, "parent:", parent, parent != nil)
	if parent != nil {
		node.parent = parent.getOwner()
		node.cachedSceneRootNode = parent.Root()
	} else {
		node.parent = nil
		node.cachedSceneRootNode = nil
	}
	node.SearchTree().ForEach(func(child INode) bool { child.setCachedSceneRoot(node.cachedSceneRootNode); return true })
}

func (node *Node) setCachedSceneRoot(root *Node) {
	node.cachedSceneRootNode = root
}

func (node *Node) cachedSceneRoot() *Node {
	return node.cachedSceneRootNode
}

// Scene looks for the Node's parents recursively to return what scene it exists in.
// If the node is not within a tree (i.e. unparented), this will return nil.
func (node *Node) Scene() *Scene {
	root := node.Root()
	if root != nil {
		return root.scene
	}
	return nil
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (node *Node) AddChildren(children ...INode) {

	me := node.getOwner()

	for _, child := range children {

		prevParent := child.Parent()
		if prevParent != nil {

			if prevParent == me {
				continue
			}

			runCallbacks := child.getRunCallbacks()

			child.setRunCallbacks(false) // Override the on tree change rq
			prevParent.RemoveChildren(child.getOwner())
			child.setRunCallbacks(runCallbacks) // Override the on tree change rq
		}

		child.setParent(me)
		child.dirtyTransform()
		node.children = append(node.children, child.getOwner())

		if child.Callbacks() != nil && child.Callbacks().OnReparent != nil {
			child.Callbacks().OnReparent(child, prevParent, me)
		}

	}

}

// getOwner returns either this Node or the Node that owns the Node, in cases where it is embedded (e.g. Model, Camera, etc).
// We do this so that parenting works as expected (parenting to and from the owning Node (the Model, camera, etc)
// rather than the Node that it embeds (e.g. Model.Node, Camera.Node, etc)).
func (node *Node) getOwner() INode {
	if node.owner != nil {
		return node.owner
	}
	return node
}

// RemoveChildren removes the provided children from this object.
func (node *Node) RemoveChildren(children ...INode) {

	for _, c1 := range children {

		child1 := c1.getOwner()

		for i, c2 := range node.children {

			child2 := c2.getOwner()

			if child2 == child1 {
				// child.updateLocalTransform(nil)
				prevParent := child1.Parent()
				child1.setParent(nil)
				child1.dirtyTransform()

				node.children[i] = nil
				node.children = append(node.children[:i], node.children[i+1:]...)

				if child1.Callbacks() != nil && child1.Callbacks().OnReparent != nil {
					child1.Callbacks().OnReparent(child1, prevParent, nil)
				}
				break
			}
		}

	}

}

// Unparent unparents the Node from its parent, removing it from the scenegraph.
func (node *Node) Unparent() {
	if node.parent != nil {
		node.parent.RemoveChildren(node)
	}
}

// ReindexChild moves the child in the calling Node's children slice to the specified newPosition.
// The function returns the old index where the child Node was, or -1 if it wasn't a child of the calling Node.
// The newPosition is clamped to the size of the node's children slice.
func (node *Node) ReindexChild(child INode, newIndex int) int {

	oldIndex := child.Index()

	if oldIndex >= 0 {

		if newIndex < 0 {
			newIndex = 0
		} else if newIndex > len(node.children)-1 {
			newIndex = len(node.children) - 1
		}

		node.children = append(node.children[:oldIndex], node.children[oldIndex+1:]...)

		node.children = append(node.children, nil)
		copy(node.children[newIndex+1:], node.children[newIndex:])
		node.children[newIndex] = child

	}

	return oldIndex

}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (node *Node) Index() int {
	var index = -1
	if node.parent != nil {
		node.parent.ForEachChild(func(child INode, i, size int) bool {
			if child == node {
				index = i
				return false
			}
			return true
		})
	}
	return index
}

// Children returns the Node's children as a slice of INodes.
func (node *Node) Children() []INode {
	return append(make([]INode, 0, len(node.children)), node.children...)
}

// ForEachChild runs the provided callback function for each child in the node's children slice.
// If the callback returns false, then it stops execution with the current child.
// The function loops through the children list in reverse so removing children is non-problematic.
func (node *Node) ForEachChild(forEach func(child INode, index, size int) bool) {
	size := len(node.children)
	for i := len(node.children) - 1; i >= 0; i-- {
		if !forEach(node.children[i], i, size) {
			return
		}
	}
}

// SearchTree returns a NodeFilter to search through and filter a Node's hierarchy.
func (node *Node) SearchTree() *NodeFilter {
	return newNodeFilter(node)
}

// FindNode searches through a Node's tree for the node by name. This is mostly syntactic sugar for
// Node.SearchTree().ByName(nodeName).First().
func (node *Node) FindNode(nodeName string) INode {
	return node.SearchTree().ByName(nodeName).First()
}

// Visible returns whether the Object is visible.
func (node *Node) Visible() bool {
	return node.visible
}

// SetVisible sets the object's visibility. If recursive is true, all recursive children of this Node will have their visibility set the same way.
func (node *Node) SetVisible(visible bool, recursive bool) {
	if recursive {
		for _, child := range node.SearchTree().INodes() {
			child.SetVisible(visible, true)
		}
	}
	node.visible = visible
}

// Properties represents an unordered set of game properties that can be used to identify this object.
func (node *Node) Properties() Properties {
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

			if nodeType.Is(NodeTypeCamera) {
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
			} else if nodeType.Is(NodeTypeModel) {
				prefix = "MODEL"
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
		wpStr := "[" + strconv.FormatFloat(float64(wp.X), 'f', floatTruncation, 64) + ", " + strconv.FormatFloat(float64(wp.Y), 'f', floatTruncation, 64) + ", " + strconv.FormatFloat(float64(wp.Z), 'f', floatTruncation, 64) + "]"

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

// If enabled, Nodes, Materials, and other object types will be represented by their names when
// printed directly. Otherwise, they will be represented by their pointer locations, like default.
var ReadableReferences = true

func (node *Node) String() string {
	if ReadableReferences {
		path := node.Path()
		if path == "" {
			path = "{ no path }"
		}
		return "< " + node.Name() + " : " + path + " : " + fmt.Sprintf("%d", node.id) + " >"
	} else {
		return fmt.Sprintf("%p", node)
	}
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
// If the Node is not in a scene tree (i.e. does not have a path to a root node), Path() will return a blank string.
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
// returns the node itself. If the node has no path to the scene root, this function returns nil.
func (node *Node) Root() *Node {

	if node.cachedSceneRootNode != nil {
		return node.cachedSceneRootNode
	}

	if node.parent != nil {
		parent := node.parent
		for parent != nil {
			if parent == parent.cachedSceneRoot() {
				node.cachedSceneRootNode = parent.(*Node)
				return node.cachedSceneRootNode
			}
			parent = parent.Parent()
		}
	}

	return nil

}

// InSceneTree returns if a Node is connected to the scene's root node through hierarchy.
func (node *Node) InSceneTree() bool {
	return node.Root() != nil
}

// IsDescendantOf returns if a Node is a descendant of a parent Node.
func (node *Node) IsDescendantOf(parent INode) bool {
	p := node.Parent()
	for p != nil {
		if p == parent {
			return true
		}
		p = p.Parent()
	}
	return false
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

func (node *Node) sectorHierarchy() *Sector {

	if node.parent != nil {
		model, ok := node.parent.(*Model)
		if ok && model.sector != nil {
			return model.sector
		} else {
			parentSector := node.parent.sectorHierarchy()
			if parentSector != nil {
				return parentSector
			}
		}
	}

	return nil

}

// Sector returns the Sector this node is in hierarchically. If that fails, then
// Sector() will search the scene tree spatially to see which of the sectors the
// calling Node lies in.
func (node *Node) Sector() *Sector {

	if node.cachedSector != nil {
		return node.cachedSector
	}

	if sectorHierarchy := node.sectorHierarchy(); sectorHierarchy != nil {
		node.cachedSector = sectorHierarchy
	} else {

		root := node.Root()

		if root == nil {
			node.cachedSector = nil
		} else {

			pos := node.WorldPosition()

			root.SearchTree().bySectors().ForEach(func(child INode) bool {
				sectorModel := child.(*Model)
				if sectorModel.sector.AABB.PointInside(pos) {
					node.cachedSector = sectorModel.sector
					return false
				}
				return true
			})

		}

	}

	return node.cachedSector

}

func (node *Node) isInVisibleSector(sectorModels []*Model) bool {
	for _, model := range sectorModels {
		if model.sector.rendering && model.sector.AABB.PointInside(node.WorldPosition()) {
			return true
		}
	}
	return false
}

// SectorType returns the current SectorType of the Node.
func (node *Node) SectorType() SectorType {
	return node.sectorType
}

// SetSectorType sets the current SectorType of the Node. Note that setting this to Sector won't work.
func (node *Node) SetSectorType(sectorType SectorType) {
	node.sectorType = sectorType
}

// DistanceTo returns the distance between the given Nodes' centers.
// Quick syntactic sugar for Node.WorldPosition().Distance(otherNode.WorldPosition()).
func (node *Node) DistanceTo(other INode) float32 {
	return node.WorldPosition().Distance(other.WorldPosition())
}

// DistanceSquaredTo returns the squared distance between the given Nodes' centers.
// Quick syntactic sugar for Node.WorldPosition().DistanceSquared(otherNode.WorldPosition()).
func (node *Node) DistanceSquaredTo(other INode) float32 {
	return node.WorldPosition().DistanceSquared(other.WorldPosition())
}

// VectorTo returns a vector from one Node to another.
// Quick syntactic sugar for other.WorldPosition().Sub(node.WorldPosition()).
func (node *Node) VectorTo(other INode) Vector3 {
	return other.WorldPosition().Sub(node.WorldPosition())
}
