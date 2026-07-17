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
	node     tetra3d.INode
	Collider *tetra3d.ColliderAABB
}

func NewPlayer(node tetra3d.INode) *Player {
	return &Player{
		node:     node,
		Collider: node.Get("ColliderAABB").(*tetra3d.ColliderAABB),
	}
}

func (player *Player) Update() {

	move := tetra3d.Vector3{}
	moveSpd := float32(0.1)

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

	player.Collider.CollisionTest(

		tetra3d.CollisionTestSettings{

			TestAgainst: player.node.Root().Search().ByParentPropHasBitfieldValueByName("Solid"),

			OnCollision: func(col *tetra3d.Collision, index, count int) bool {

				if col.Object.Parent().PropertiesContainsBitByName("Death") {
					player.node.Unparent() // Unparenting is the equivalent of destroying the node
				}

				player.node.MoveVec(col.MaxMTV())

				return true
			},
		},
	)

}

func (player *Player) Node() tetra3d.INode {
	return player.node
}
