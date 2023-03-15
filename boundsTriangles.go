package tetra3d

import (
	"math"
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

// Colliding returns true if the BoundingTriangles object is intersecting the other specified BoundingObject.
func (bt *BoundingTriangles) Colliding(other IBoundingObject) bool {
	return bt.Collision(other) != nil
}

// Collision returns a Collision if the BoundingTriangles object is intersecting another BoundingObject. If
// no intersection is reported, Collision returns nil. (Note that BoundingTriangles > AABB collision is buggy at the moment.)
func (bt *BoundingTriangles) Collision(other IBoundingObject) *Collision {

	if other == bt || other == nil {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingAABB:
		intersection := otherBounds.Collision(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				inter.Normal = inter.Normal.Invert()
			}
			intersection.BoundingObject = otherBounds
		}
		return intersection

	case *BoundingSphere:
		intersection := otherBounds.Collision(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				inter.Normal = inter.Normal.Invert()
			}
			intersection.BoundingObject = otherBounds
		}
		return intersection

	case *BoundingTriangles:
		return btTrianglesTriangles(bt, otherBounds)

	case *BoundingCapsule:
		intersection := otherBounds.Collision(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				inter.Normal = inter.Normal.Invert()
			}
			intersection.BoundingObject = otherBounds
		}
		return intersection

	}

	panic("Unimplemented bounds type")

}

// CollisionTest performs a collision test using the provided collision test settings structure.
// As a nicety, CollisionTest also returns a distance-sorted slice of all of the Collisions (but you should rather
// handle collisions with intent using the OnCollision function of the CollisionTestSettings struct).
func (bt *BoundingTriangles) CollisionTest(settings CollisionTestSettings) []*Collision {
	return CommonCollisionTest(bt, settings)
}

type collisionPlane struct {
	Normal   Vector
	Distance float64
}

func newCollisionPlane() collisionPlane {
	return collisionPlane{}
}

func (plane *collisionPlane) Set(v0, v1, v2 Vector) {

	first := v1.Sub(v0)
	second := v2.Sub(v0)
	normal := first.Cross(second).Unit()
	distance := normal.Dot(v0)

	plane.Normal = normal
	plane.Distance = distance

}

func (plane *collisionPlane) ClosestPoint(point Vector) Vector {

	dist := plane.Normal.Dot(point) - plane.Distance
	return point.Sub(plane.Normal.Scale(dist))

}

func (plane *collisionPlane) RayAgainstPlane(from, to Vector) (Vector, bool) {

	dir := to.Sub(from).Unit()

	nd := dir.Dot(plane.Normal)
	pn := from.Dot(plane.Normal)

	if nd >= 0 {
		return Vector{}, false
	}

	t := (plane.Distance - pn) / nd

	if t >= 0 {
		return from.Add(dir.Scale(t)), true
	}

	return Vector{}, false

}

var colPlane = newCollisionPlane()

func closestPointOnTri(point, v0, v1, v2 Vector) Vector {

	colPlane.Set(v0, v1, v2)
	if planePoint := colPlane.ClosestPoint(point); colPlane.pointInsideTriangle(planePoint, v0, v1, v2) {
		return planePoint
	}

	ab := colPlane.closestPointOnLine(point, v0, v1)
	bc := colPlane.closestPointOnLine(point, v1, v2)
	ca := colPlane.closestPointOnLine(point, v2, v0)

	closest := ab
	closestDist := point.DistanceSquared(ab)

	bcDist := point.DistanceSquared(bc)
	caDist := point.DistanceSquared(ca)

	if bcDist < closestDist {
		closest = bc
		closestDist = bcDist
	}

	if caDist < closestDist {
		closest = ca
	}

	return closest

}

func (plane *collisionPlane) pointInsideTriangle(point, v0, v1, v2 Vector) bool {

	ca := v2.Sub(v0)
	ba := v1.Sub(v0)
	pa := point.Sub(v0)

	dot00 := ca.Dot(ca)
	dot01 := ca.Dot(ba)
	dot02 := ca.Dot(pa)

	dot11 := ba.Dot(ba)
	dot12 := ba.Dot(pa)

	invDenom := 1.0 / ((dot00 * dot11) - (dot01 * dot01))
	u := ((dot11 * dot02) - (dot01 * dot12)) * invDenom
	v := ((dot00 * dot12) - (dot01 * dot02)) * invDenom

	return (u >= 0) && (v >= 0) && (u+v < 1)

}

func (plane *collisionPlane) closestPointOnLine(point, start, end Vector) Vector {

	diff := end.Sub(start)
	dotA := point.Sub(start).Dot(diff)
	dotB := diff.Dot(diff)
	d := dotA / dotB
	if d > 1 {
		d = 1
	} else if d < 0 {
		d = 0
	}

	return start.Add(diff.Scale(d))

}

/////

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (bt *BoundingTriangles) AddChildren(children ...INode) {
	bt.addChildren(bt, children...)
}

// Unparent unparents the Camera from its parent, removing it from the scenegraph.
func (bt *BoundingTriangles) Unparent() {
	if bt.parent != nil {
		bt.parent.RemoveChildren(bt)
	}
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (bt *BoundingTriangles) Index() int {
	if bt.parent != nil {
		for i, c := range bt.parent.Children() {
			if c == bt {
				return i
			}
		}
	}
	return -1
}

// Type returns the NodeType for this object.
func (bt *BoundingTriangles) Type() NodeType {
	return NodeTypeBoundingTriangles
}
