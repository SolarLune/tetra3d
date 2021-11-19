package tetra3d

import (
	"github.com/kvartborg/vector"
)

type Node struct {
	Name          string
	Parent        *Node
	LocalPosition vector.Vector
	LocalScale    vector.Vector
	LocalRotation Matrix4
	Children      []*Node
	Owner         interface{}
	Visible       bool
}

func NewNode(name string) *Node {
	return &Node{
		Name:          name,
		LocalPosition: vector.Vector{0, 0, 0},
		LocalScale:    vector.Vector{1, 1, 1},
		LocalRotation: NewMatrix4(),
		Children:      []*Node{},
		Visible:       true,
	}
}

func (node *Node) Clone() *Node {
	newNode := NewNode(node.Name)
	newNode.LocalPosition = node.LocalPosition.Clone()
	newNode.LocalScale = node.LocalScale.Clone()
	newNode.LocalRotation = node.LocalRotation.Clone()
	newNode.Parent = node.Parent
	for _, child := range node.Children {
		newNode.Children = append(newNode.Children, child.Clone())
	}
	newNode.Owner = node.Owner
	return newNode
}

// Transform returns a Matrix4 indicating the position, rotation, and scale of the Model.
func (node *Node) Transform() Matrix4 {

	// T * R * S * O

	transform := NewMatrix4Scale(node.LocalScale[0], node.LocalScale[1], node.LocalScale[2])
	transform = transform.Mult(node.LocalRotation)
	transform = transform.Mult(NewMatrix4Translate(node.LocalPosition[0], node.LocalPosition[1], node.LocalPosition[2]))

	if node.Parent != nil {
		transform = transform.Mult(node.Parent.Transform())
	}

	return transform

}

func (node *Node) updateLocalTransform(newParent *Node) {

	if node.Parent == newParent {
		return
	}

	if newParent != nil {

		parentTransform := newParent.Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		diff := node.LocalPosition.Sub(parentPos)
		diff[0] /= parentScale[0]
		diff[1] /= parentScale[1]
		diff[2] /= parentScale[2]
		node.LocalPosition = parentRot.Transposed().MultVec(diff)
		node.LocalRotation = node.LocalRotation.Mult(parentRot.Transposed())

		node.LocalScale[0] /= parentScale[0]
		node.LocalScale[1] /= parentScale[1]
		node.LocalScale[2] /= parentScale[2]

	} else {

		// Reverse

		parentTransform := node.Parent.Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		pr := parentRot.MultVec(node.LocalPosition)
		pr[0] *= parentScale[0]
		pr[1] *= parentScale[1]
		pr[2] *= parentScale[2]
		node.LocalPosition = parentPos.Add(pr)
		node.LocalRotation = node.LocalRotation.Mult(parentRot)

		node.LocalScale[0] *= parentScale[0]
		node.LocalScale[1] *= parentScale[1]
		node.LocalScale[2] *= parentScale[2]

	}

}

func (node *Node) WorldPosition() vector.Vector {
	position, _, _ := node.Transform().Decompose()
	return position
}

func (node *Node) SetWorldPosition(position vector.Vector) {

	if node.Parent != nil {

		parentTransform := node.Parent.Transform()
		parentPos, parentScale, parentRot := parentTransform.Decompose()

		pr := parentRot.Transposed().MultVec(position.Sub(parentPos))
		pr[0] /= parentScale[0]
		pr[1] /= parentScale[1]
		pr[2] /= parentScale[2]
		node.LocalPosition = pr

	} else {
		node.LocalPosition = position
	}

}

func (node *Node) WorldScale() vector.Vector {
	_, scale, _ := node.Transform().Decompose()
	return scale
}

func (node *Node) SetWorldScale(scale vector.Vector) {

	if node.Parent != nil {

		parentTransform := node.Parent.Transform()
		_, parentScale, _ := parentTransform.Decompose()

		node.LocalScale = vector.Vector{
			scale[0] / parentScale[0],
			scale[1] / parentScale[1],
			scale[2] / parentScale[2],
		}

	} else {
		node.LocalScale = scale
	}

}

func (node *Node) WorldRotation() Matrix4 {
	_, _, rotation := node.Transform().Decompose()
	return rotation
}

func (node *Node) SetWorldRotation(rotation Matrix4) {

	if node.Parent != nil {

		parentTransform := node.Parent.Transform()
		_, _, parentRot := parentTransform.Decompose()
		node.LocalRotation = parentRot.Transposed().Mult(rotation)

	} else {
		node.LocalRotation = rotation
	}

}

func (node *Node) AddChildren(children ...*Node) {
	for _, child := range children {
		if child.Parent != nil && child.Parent != node {
			child.Parent.RemoveChildren(child)
		}
		child.updateLocalTransform(node)
		child.Parent = node
		node.Children = append(node.Children, child)
	}
}

func (node *Node) RemoveChildren(children ...*Node) {

	for _, child := range children {
		for i, c := range node.Children {
			if c == child {
				child.updateLocalTransform(nil)
				child.Parent = nil
				node.Children[i] = nil
				node.Children = append(node.Children[:i], node.Children[i+1:]...)
			}
		}
	}

}

// ChildrenRecursive returns all related children Nodes underneath this one.
func (node *Node) ChildrenRecursive(onlyVisible bool) []*Node {
	out := []*Node{}
	for _, child := range node.Children {
		if !onlyVisible || child.Visible {
			out = append(out, child)
			out = append(out, child.ChildrenRecursive(onlyVisible)...)
		}
	}
	return out
}

func (node *Node) IsModel() bool {
	_, isModel := node.Owner.(*Model)
	return isModel
}

func (node *Node) AsModel() *Model {
	model, _ := node.Owner.(*Model)
	return model
}

func (node *Node) IsCamera() bool {
	_, isCamera := node.Owner.(*Camera)
	return isCamera
}

func (node *Node) AsCamera() *Camera {
	camera, _ := node.Owner.(*Camera)
	return camera
}

type Spatial interface {
	getNode() *Node
}
