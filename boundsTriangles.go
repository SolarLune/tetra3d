package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

// BoundingTriangles is a Node specifically for detecting a collision between any of the triangles from a mesh instance and another BoundingObject.
type BoundingTriangles struct {
	*Node
	BoundingAABB *BoundingAABB
	Mesh         *Mesh
}

// NewBoundingTriangles returns a new BoundingTriangles object.
func NewBoundingTriangles(name string, mesh *Mesh) *BoundingTriangles {
	return &BoundingTriangles{
		Node:         NewNode(name),
		BoundingAABB: NewBoundingAABB("triangle broadphase aabb", mesh.Dimensions.Width(), mesh.Dimensions.Height(), mesh.Dimensions.Depth()),
		Mesh:         mesh,
	}
}

// Intersecting returns true if the BoundingTriangles object is intersecting the other specified BoundingObject.
func (bt *BoundingTriangles) Intersecting(other BoundingObject) bool {
	return bt.Intersection(other) != nil
}

// Intersection returns an IntersectionResult if the BoundingTriangles object is intersecting another BoundingObject. If
// no intersection is reported, Intersection returns nil. (Note that BoundingTriangles > AABB collision is buggy at the moment.)
func (bt *BoundingTriangles) Intersection(other BoundingObject) *IntersectionResult {

	if other == bt {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingAABB:
		oi := otherBounds.Intersection(bt)
		if oi != nil {
			oi.MTV = oi.MTV.Invert()
		}
		return oi

	case *BoundingSphere:
		oi := otherBounds.Intersection(bt)
		if oi != nil {
			oi.MTV = oi.MTV.Invert()
		}
		return oi

	case *BoundingTriangles:
		return btTrianglesTriangles(bt, otherBounds)

	case *BoundingCapsule:
		oi := otherBounds.Intersection(bt)
		if oi != nil {
			oi.MTV = oi.MTV.Invert()
		}
		return oi

	}

	return nil
}

type collisionPlane struct {
	Normal   vector.Vector
	Distance float64
}

func newCollisionPlane() *collisionPlane {
	return &collisionPlane{}
}

func (plane *collisionPlane) Set(v0, v1, v2 vector.Vector) {

	first := v1.Sub(v0)
	second := v2.Sub(v0)
	normal, _ := first.Cross(second)
	normal = normal.Unit()
	distance := normal.Dot(v0)

	plane.Normal = normal
	plane.Distance = distance

}

func (plane *collisionPlane) ClosestPoint(point vector.Vector) vector.Vector {

	dist := plane.Normal.Dot(point) - plane.Distance
	return point.Sub(plane.Normal.Scale(dist))

}

var colPlane = newCollisionPlane()

func closestPointOnTri(point, v0, v1, v2 vector.Vector) vector.Vector {

	colPlane.Set(v0, v1, v2)
	if planePoint := colPlane.ClosestPoint(point); pointInsideTriangle(planePoint, v0, v1, v2) {
		return planePoint
	}

	ab := closestPointOnLine(point, v0, v1)
	bc := closestPointOnLine(point, v1, v2)
	ca := closestPointOnLine(point, v2, v0)

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

func pointInsideTriangle(point, v0, v1, v2 vector.Vector) bool {

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

	// v0 = v0.Sub(point)
	// v1 = v1.Sub(point)
	// v2 = v2.Sub(point)

	// u, _ := v1.Cross(v2)
	// v, _ := v2.Cross(v0)
	// w, _ := v0.Cross(v1)

	// if u.Dot(v) < 0 {
	// 	return false
	// }

	// if u.Dot(w) < 0 {
	// 	return false
	// }

	// return true

}

func closestPointOnLine(point, start, end vector.Vector) vector.Vector {

	diff := end.Sub(start)
	dotA := fastVectorSub(point, start).Dot(diff)
	dotB := diff.Dot(diff)
	d := math.Min(math.Max(dotA/dotB, 0), 1)
	return start.Add(diff.Scale(d))

}
