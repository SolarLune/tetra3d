package tetra3d

import "github.com/solarlune/tetra3d/math32"

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
func NewBoundingTriangles(name string, mesh *Mesh, broadphaseGridSize float32) *BoundingTriangles {
	margin := float32(0.25) // An additional margin to help ensure the broadphase is crossed before checking for collisions
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
		gridSize = int(math32.Ceil(maxDim / broadphaseGridSize))
	}

	bt.Broadphase = NewBroadphase(gridSize, bt.WorldPosition(), mesh)

	bt.owner = bt

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
		bt.Broadphase.center.SetWorldTransform(transform)
		bt.Broadphase.center.MoveVec(rot)
		bt.Broadphase.center.Transform() // Update the transform
	}

}

// Clone returns a new BoundingTriangles Node with the same values set as the original.
func (bt *BoundingTriangles) Clone() INode {
	clone := NewBoundingTriangles(bt.name, bt.Mesh, 0) // Broadphase size is set to 0 so cloning doesn't create the broadphase triangle sets
	clone.Broadphase = bt.Broadphase.Clone()
	clone.Node = bt.Node.clone(clone).(*Node)
	clone.Node.onTransformUpdate = clone.UpdateTransform
	if clone.Callbacks() != nil && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}
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
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return if a collision was found with the sphere at the settings specified.
func (bt *BoundingTriangles) CollisionTest(settings CollisionTestSettings) bool {
	return commonCollisionTest(bt, settings)
}

type collisionPlane struct {
	Normal   Vector3
	Distance float32
}

func newCollisionPlane() collisionPlane {
	return collisionPlane{}
}

func (plane *collisionPlane) Set(v0, v1, v2 Vector3) {

	first := v1.Sub(v0)
	second := v2.Sub(v0)
	normal := first.Cross(second).Unit()
	distance := normal.Dot(v0)

	plane.Normal = normal
	plane.Distance = distance

}

func (plane *collisionPlane) ClosestPoint(point Vector3) Vector3 {

	dist := plane.Normal.Dot(point) - plane.Distance
	return point.Sub(plane.Normal.Scale(dist))

}

func (plane *collisionPlane) RayAgainstPlane(from, to Vector3, doublesided bool) (Vector3, bool) {

	dir := to.Sub(from).Unit()

	nd := dir.Dot(plane.Normal)
	pn := from.Dot(plane.Normal)

	if !doublesided && nd >= 0 {
		// if !doublesided && nd <= 0 {
		return Vector3{}, false
	}

	t := (plane.Distance - pn) / nd

	if t >= 0 {
		return from.Add(dir.Scale(t)), true
	}

	return Vector3{}, false

}

var colPlane = newCollisionPlane()

func closestPointOnTri(point, v0, v1, v2 Vector3) Vector3 {

	colPlane.Set(v0, v1, v2)
	if planePoint := colPlane.ClosestPoint(point); isPointInsideTriangle(planePoint, v0, v1, v2) {
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

func isPointInsideTriangle(point, v0, v1, v2 Vector3) bool {
	u, v := pointInsideTriangle(point, v0, v1, v2)
	return (u >= 0) && (v >= 0) && (u+v < 1)
}

func pointInsideTriangle(point, v0, v1, v2 Vector3) (u, v float32) {

	ca := v2.Sub(v0)
	ba := v1.Sub(v0)
	pa := point.Sub(v0)

	dot00 := ca.Dot(ca)
	dot01 := ca.Dot(ba)
	dot02 := ca.Dot(pa)

	dot11 := ba.Dot(ba)
	dot12 := ba.Dot(pa)

	invDenom := 1.0 / ((dot00 * dot11) - (dot01 * dot01))
	u = ((dot11 * dot02) - (dot01 * dot12)) * invDenom
	v = ((dot00 * dot12) - (dot01 * dot02)) * invDenom

	// return (u >= 0) && (v >= 0) && (u+v < 1)

	return
}

func (plane *collisionPlane) closestPointOnLine(point, start, end Vector3) Vector3 {

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

// Type returns the NodeType for this object.
func (bt *BoundingTriangles) Type() NodeType {
	return NodeTypeBoundingTriangles
}
