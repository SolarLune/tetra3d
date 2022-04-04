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

func (bt *BoundingTriangles) Transform() Matrix4 {

	transformDirty := bt.Node.isTransformDirty

	transform := bt.Node.Transform()

	if transformDirty {
		bt.BoundingAABB.SetWorldTransform(transform)
		rot := bt.WorldRotation().MultVec(bt.Mesh.Dimensions.Center())
		bt.BoundingAABB.MoveVec(rot)
		bt.BoundingAABB.Transform()
	}

	return transform

}

func (bt *BoundingTriangles) Clone() INode {
	clone := NewBoundingTriangles(bt.name, bt.Mesh)
	clone.Node = bt.Node.Clone().(*Node)
	return clone
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (bt *BoundingTriangles) AddChildren(children ...INode) {
	// We do this manually so that addChildren() parents the children to the Model, rather than to the Model.NodeBase.
	bt.addChildren(bt, children...)
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
		intersection := otherBounds.Intersection(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
			}
		}
		return intersection

	case *BoundingSphere:
		intersection := otherBounds.Intersection(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
			}
		}
		return intersection

	case *BoundingTriangles:
		return btTrianglesTriangles(bt, otherBounds)

	case *BoundingCapsule:
		intersection := otherBounds.Intersection(bt)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
			}
		}
		return intersection

	}

	return nil
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
		VectorPool: NewVectorPool(16),
	}
}

func (plane *collisionPlane) Set(v0, v1, v2 vector.Vector) {

	first := plane.VectorPool.Sub(v1, v0)
	second := plane.VectorPool.Sub(v2, v0)
	normal := plane.VectorPool.Cross(first, second).Unit()
	distance := dot(normal, v0)

	plane.Normal = normal
	plane.Distance = distance

}

func (plane *collisionPlane) ClosestPoint(point vector.Vector) vector.Vector {

	dist := dot(plane.Normal, point) - plane.Distance
	return plane.VectorPool.Sub(point, plane.Normal.Scale(dist))[:3]

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

	ca := plane.VectorPool.Sub(v2, v0)[:3]
	ba := plane.VectorPool.Sub(v1, v0)[:3]
	pa := plane.VectorPool.Sub(point, v0)[:3]

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
	d := math.Min(math.Max(dotA/dotB, 0), 1)
	return plane.VectorPool.Add(start, diff.Scale(d))

}
