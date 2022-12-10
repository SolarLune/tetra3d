package main

import (
	"github.com/hajimehoshi/ebiten/v2"
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

	move := tetra3d.NewVectorZero()
	moveSpd := 0.1

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		move.X -= moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		move.X += moveSpd
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		move.Z -= moveSpd
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		move.Z += moveSpd
	}

	player.node.MoveVec(move)

	player.Bounds.CollisionTest(

		tetra3d.CollisionTestSettings{

			Others: player.node.Root().ChildrenRecursive().ByTags("solid"),

			HandleCollision: func(col *tetra3d.Collision) bool {

				if col.BoundingObject.Parent().Properties().Has("death") {
					player.node.Unparent() // Unparenting is the equivalent of destroying the node
				}

				player.node.MoveVec(col.AverageMTV())

				return true
			},
		},
	)

}

func (player *Player) Node() tetra3d.INode {
	return player.node
}
