package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

// BoundingTriangles is a Node specifically for detecting a collision between any of the triangles from a mesh instance and another BoundingObject.
type BoundingTriangles struct {
	*Node
	BoundingAABB *BoundingAABB
	Broadphase   *Broadphase
	Mesh         *Mesh
}

// NewBoundingTriangles returns a new BoundingTriangles object. name is the name of the BoundingTriangles node, while mesh is a reference
// to the Mesh that the BoundingTriangles object should use for collision. broadphaseCellSize is how large the BoundingTriangle's broadphase
// collision cells should be - larger cells means more triangles are checked at a time when doing collision checks, while larger cells also means
// fewer broadphase cells need to be checked. Striking a balance is a good idea if you're setting this value by hand (by default, the grid size is
// the maximum dimension size / 20, rounded up (a grid of at least one cell every 20 Blender Units). A size of 0 disables the usage of the broadphase
// for collision checks.
func NewBoundingTriangles(name string, mesh *Mesh, broadphaseGridSize float64) *BoundingTriangles {
	margin := 0.25 // An additional margin to help ensure the broadphase is crossed before checking for collisions
	bt := &BoundingTriangles{
		Node:         NewNode(name),
		BoundingAABB: NewBoundingAABB("triangle broadphase aabb", mesh.Dimensions.Width()+margin, mesh.Dimensions.Height()+margin, mesh.Dimensions.Depth()+margin),
		Mesh:         mesh,
	}
	bt.Node.onTransformUpdate = bt.UpdateTransform

	// This initializes the broadphase using a default grid size.

	// If the object is too small (less than 5 units large), it may not be worth doing
	maxDim := bt.Mesh.Dimensions.MaxDimension()

	gridSize := 0

	if broadphaseGridSize > 0 {
		gridSize = int(math.Ceil(maxDim / broadphaseGridSize))
	}

	bt.Broadphase = NewBroadphase(gridSize, bt)

	return bt
}

// DisableBroadphase turns off the broadphase system for collision detection by settings its grid and cell size to 0.
// To turn broadphase collision back on, simply call Broadphase.Resize(gridSize) with a gridSize value above 0.
func (bt *BoundingTriangles) DisableBroadphase() {
	bt.Broadphase.Resize(0)
}

// Transform returns a Matrix4 indicating the global position, rotation, and scale of the object, transforming it by any parents'.
// If there's no change between the previous Transform() call and this one, Transform() will return a cached version of the
// transform for efficiency.
func (bt *BoundingTriangles) UpdateTransform() {

	transform := bt.Node.Transform()

	bt.BoundingAABB.SetWorldTransform(transform)
	rot := bt.WorldRotation().MultVec(bt.Mesh.Dimensions.Center())
	bt.BoundingAABB.MoveVec(rot)
	bt.BoundingAABB.Transform()

	if bt.Broadphase != nil {
		bt.Broadphase.Center.SetWorldTransform(transform)
		bt.Broadphase.Center.MoveVec(rot)
		bt.Broadphase.Center.Transform() // Update the transform
	}

}

// Clone returns a new BoundingTriangles Node with the same values set as the original.
func (bt *BoundingTriangles) Clone() INode {
	clone := NewBoundingTriangles(bt.name, bt.Mesh, 0) // Broadphase size is set to 0 so cloning doesn't create the broadphase triangle sets
	clone.Broadphase = bt.Broadphase.Clone()
	clone.Node = bt.Node.Clone().(*Node)
	clone.Node.onTransformUpdate = clone.UpdateTransform
	return clone
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (bt *BoundingTriangles) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	bt.addChildren(bt, children...)
}

// Colliding returns true if the BoundingTriangles object is intersecting the other specified BoundingObject.
func (bt *BoundingTriangles) Colliding(other BoundingObject) bool {
	return bt.Collision(other) != nil
}

// Collision returns a Collision if the BoundingTriangles object is intersecting another BoundingObject. If
// no intersection is reported, Collision returns nil. (Note that BoundingTriangles > AABB collision is buggy at the moment.)
func (bt *BoundingTriangles) Collision(other BoundingObject) *Collision {

	if other == bt {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingAABB:
		intersection := otherBounds.Collision(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				vector.In(inter.Normal).Invert()
			}
			intersection.CollidedObject = otherBounds
		}
		return intersection

	case *BoundingSphere:
		intersection := otherBounds.Collision(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				vector.In(inter.Normal).Invert()
			}
			intersection.CollidedObject = otherBounds
		}
		return intersection

	case *BoundingTriangles:
		return btTrianglesTriangles(bt, otherBounds)

	case *BoundingCapsule:
		intersection := otherBounds.Collision(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				vector.In(inter.Normal).Invert()
			}
			intersection.CollidedObject = otherBounds
		}
		return intersection

	}

	panic("Unimplemented bounds type")

}

// CollisionTest performs an collision test if the bounding object were to move in the given direction in world space.
// It returns all valid Collisions across all BoundingObjects passed in as others. Collisions will be sorted in order of
// distance. If no Collisions occurred, it will return an empty slice.
func (bt *BoundingTriangles) CollisionTest(dx, dy, dz float64, others ...BoundingObject) []*Collision {
	return commonCollisionTest(bt, dx, dy, dz, others...)
}

// CollisionTestVec performs an collision test if the bounding object were to move in the given direction in world space
// using a vector. If no vector is supplied, it's equivalent to {0, 0, 0}. It returns all valid Collisions across all
// BoundingObjects passed in as others. Collisions will be sorted in order of distance. If no Collisions occurred, it will return an empty slice.
func (bt *BoundingTriangles) CollisionTestVec(moveVec vector.Vector, others ...BoundingObject) []*Collision {
	if moveVec == nil {
		return commonCollisionTest(bt, 0, 0, 0, others...)
	}
	return commonCollisionTest(bt, moveVec[0], moveVec[1], moveVec[2], others...)
}

// Type returns the NodeType for this object.
func (bt *BoundingTriangles) Type() NodeType {
	return NodeTypeBoundingTriangles
}

type collisionPlane struct {
	Normal     vector.Vector
	Distance   float64
	VectorPool *VectorPool
}

func newCollisionPlane() *collisionPlane {
	return &collisionPlane{
		VectorPool: NewVectorPool(16, false),
	}
}

func (plane *collisionPlane) Set(v0, v1, v2 vector.Vector) {

	first := plane.VectorPool.Sub(v1, v0)
	second := plane.VectorPool.Sub(v2, v0)
	normal := plane.VectorPool.Cross(first, second)
	vector.In(normal).Unit()
	distance := dot(normal, v0)

	plane.Normal = normal
	plane.Distance = distance

}

func (plane *collisionPlane) ClosestPoint(point vector.Vector) vector.Vector {

	dist := dot(plane.Normal, point) - plane.Distance
	return plane.VectorPool.Sub(point, plane.Normal.Scale(dist))

}

var colPlane = newCollisionPlane()

func closestPointOnTri(point, v0, v1, v2 vector.Vector) vector.Vector {

	colPlane.VectorPool.Reset()

	colPlane.Set(v0, v1, v2)
	if planePoint := colPlane.ClosestPoint(point); colPlane.pointInsideTriangle(planePoint, v0, v1, v2) {
		return planePoint
	}

	ab := colPlane.closestPointOnLine(point, v0, v1)
	bc := colPlane.closestPointOnLine(point, v1, v2)
	ca := colPlane.closestPointOnLine(point, v2, v0)

	closest := ab
	closestDist := fastVectorDistanceSquared(point, ab)

	bcDist := fastVectorDistanceSquared(point, bc)
	caDist := fastVectorDistanceSquared(point, ca)

	if bcDist < closestDist {
		closest = bc
		closestDist = bcDist
	}

	if caDist < closestDist {
		closest = ca
	}

	return closest

}

func (plane *collisionPlane) pointInsideTriangle(point, v0, v1, v2 vector.Vector) bool {

	ca := plane.VectorPool.Sub(v2, v0)
	ba := plane.VectorPool.Sub(v1, v0)
	pa := plane.VectorPool.Sub(point, v0)

	dot00 := dot(ca, ca)
	dot01 := dot(ca, ba)
	dot02 := dot(ca, pa)

	dot11 := dot(ba, ba)
	dot12 := dot(ba, pa)

	invDenom := 1.0 / ((dot00 * dot11) - (dot01 * dot01))
	u := ((dot11 * dot02) - (dot01 * dot12)) * invDenom
	v := ((dot00 * dot12) - (dot01 * dot02)) * invDenom

	return (u >= 0) && (v >= 0) && (u+v < 1)

}

func (plane *collisionPlane) closestPointOnLine(point, start, end vector.Vector) vector.Vector {

	diff := plane.VectorPool.Sub(end, start)
	dotA := dot(plane.VectorPool.Sub(point, start), diff)
	dotB := dot(diff, diff)
	d := dotA / dotB
	if d > 1 {
		d = 1
	} else if d < 0 {
		d = 0
	}
	diff[0] *= d
	diff[1] *= d
	diff[2] *= d

	return plane.VectorPool.Add(start, diff)

}
