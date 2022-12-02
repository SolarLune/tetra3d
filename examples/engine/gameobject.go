package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
	"github.com/solarlune/tetra3d"
)

// The concept here is that one can tie Go structs to Tetra3D nodes by designing an interface that directly
// interacts with that node. In this example, we use the Node.SetData() function to store a Go object (in this case,
// a Player) in the Node. A GameObject is anything that fulfills the below interface, which means we can now
// loop through Nodes in our scene and call Update() on whatever's in their Data() space.

type GameObject interface {
	Node() tetra3d.INode
	Update()
}

type Player struct {
	node   tetra3d.INode
	Bounds *tetra3d.BoundingAABB
}

func NewPlayer(node tetra3d.INode) *Player {
	return &Player{
		node:   node,
		Bounds: node.ChildrenRecursive().ByType(tetra3d.NodeTypeBoundingAABB).First().(*tetra3d.BoundingAABB),
	}
}

func (player *Player) Update() {

	move := Vector{0, 0, 0}
	moveSpd := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		move[0] -= moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		move[0] += moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		move[2] -= moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		move[2] += moveSpd
	}

	collisions := player.Bounds.CollisionTestVec(
		move,
		player.node.Root().ChildrenRecursive().ByTags("solid")...,
	)

	player.node.MoveVec(move)

	for _, col := range collisions {

		if col.BoundingObject.Parent().Properties().Has("death") {
			player.node.Unparent() // Unparenting is the equivalent of destroying the node
		}

		player.node.MoveVec(col.AverageMTV())

	}

}

func (player *Player) Node() tetra3d.INode {
	return player.node
}
