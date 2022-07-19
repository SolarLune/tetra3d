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
	node tetra3d.INode
}

func NewPlayer(node tetra3d.INode) *Player {
	return &Player{
		node: node,
	}
}

func (player *Player) Update() {

	move := vector.Vector{0, 0, 0}
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

	player.node.MoveVec(move)

	playerBounds := player.node.ChildrenRecursive().ByType(tetra3d.NodeTypeBoundingAABB)[0].(*tetra3d.BoundingAABB)

	movementResolution := vector.Vector{0, 0, 0}

	for _, b := range player.node.Root().ChildrenRecursive().ByType(tetra3d.NodeTypeBounding) {
		bounds := b.(tetra3d.BoundingObject)

		if result := playerBounds.Collision(bounds); result != nil {

			if b.Parent().Tags().Has("death") {
				// Die
				player.node.Unparent() // Unparenting is the equivalent of destroying the node
			}

			// An object can intersect with multiple other BoundingObjects at the same time, so we
			// combine them all together here. This isn't necessarily the only way to resolve collisions,
			// but it is one way.
			for _, intersection := range result.Intersections {
				movementResolution = movementResolution.Add(intersection.MTV)
			}

		}
	}
	player.node.MoveVec(movementResolution)

}

func (player *Player) Node() tetra3d.INode {
	return player.node
}
