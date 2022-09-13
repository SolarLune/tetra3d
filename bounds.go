package tetra3d

import (
	"math"
	"sort"

	"github.com/kvartborg/vector"
)

type Intersection struct {
	// The contact point between the two intersecting objects. Note that this may be the average
	// between the two overlapping shapes, rather than the point of contact specifically.
	StartingPoint vector.Vector // The starting point for the intersection; either the center of the object for sphere / aabb, the center of the closest point for capsules, or the triangle position for triangless.
	ContactPoint  vector.Vector // The contact point for the intersection.
	MTV           vector.Vector // MTV represents the minimum translation vector to remove the calling object from the intersecting object.
	Triangle      *Triangle     // Triangle represents the triangle that was intersected in intersection tests that involve triangle meshes; if no triangle mesh was tested against, then this will be nil.
	Normal        vector.Vector
}

// Slope returns the slope of the intersection's normal, in radians. This ranges from 0 (straight up) to pi (straight down).
func (intersection *Intersection) Slope() float64 {
	return vector.Y.Angle(intersection.Normal)
}

// SlideAgainstNormal takes an input vector and alters it to slide against the intersection's returned normal.
func (intersection *Intersection) SlideAgainstNormal(movementVec vector.Vector) vector.Vector {

	temp, _ := intersection.Normal.Cross(movementVec)

	if temp.Magnitude() == 0 {
		return movementVec.Invert()
	}

	out, _ := temp.Cross(intersection.Normal)

	return out

}

// Collision represents the result of a collision test. A Collision test may result in multiple intersections, and
// so an Collision holds each of these individual intersections in its Intersections slice.
// The intersections are sorted in order of distance from the starting point of the intersection (the center of the
// colliding sphere / aabb, the closest point in the capsule, the center of the closest triangle, etc) to the
// contact point.
type Collision struct {
	CollidedBoundingObject INode // The BoundingObject collided with
	// The root colliding object; this can be the same or different from the CollidedBoundingObject, depending
	// on which object was tested against (either an individual BoundingObject, or the parent / grandparent of a tree
	// that contains one or more BoundingObjects)
	CollidedRoot  INode
	Intersections []*Intersection // The slice of Intersections, one for each object or triangle intersected with, arranged in order of distance (far to close).
}

func newCollision(collidedObject INode) *Collision {
	return &Collision{
		CollidedBoundingObject: collidedObject,
		Intersections:          []*Intersection{},
	}
}

func (col *Collision) add(intersection *Intersection) *Collision {
	col.Intersections = append(col.Intersections, intersection)
	return col
}

// sort the intersections by distance from starting point (which should be the same for all collisions except for triangle-triangle) to contact point.
func (col *Collision) sortResults() {
	sort.Slice(col.Intersections, func(i, j int) bool {
		return fastVectorDistanceSquared(col.Intersections[i].StartingPoint, col.Intersections[i].ContactPoint) >
			fastVectorDistanceSquared(col.Intersections[j].StartingPoint, col.Intersections[j].ContactPoint)
	})
}

// AverageMTV returns the average MTV (minimum translation vector) from all Intersections contained within the Collision.
// To be specific, this isn't actually the pure average, but rather is the result of adding together all MTVs from Intersections
// in the Collision for the direction, and using the greatest MTV's magnitude for the distance of the returned vector. In other
// words, AverageMTV returns the MTV to move in that should resolve all intersections from the Collision.
func (col *Collision) AverageMTV() vector.Vector {
	greatestDist := 0.0
	mtv := vector.Vector{0, 0, 0}
	for _, inter := range col.Intersections {
		mag := inter.MTV.Magnitude()
		if mag > greatestDist {
			greatestDist = mag
		}
		vector.In(mtv).Add(inter.MTV)
	}
	vector.In(mtv).Unit().Scale(greatestDist)
	return mtv
}

// AverageNormal returns the average normal vector from all Intersections contained within the Collision.
func (col *Collision) AverageNormal() vector.Vector {
	normal := col.Intersections[0].Normal.Clone()
	for i := 1; i < len(col.Intersections); i++ {
		vector.In(normal).Add(col.Intersections[i].Normal)
	}
	vector.In(normal).Scale(1.0 / float64(len(col.Intersections)))
	return normal
}

// SlideAgainstAverageNormal takes an input movement vector and alters it to slide against the Collision's average normal.
func (col *Collision) SlideAgainstAverageNormal(movementVec vector.Vector) vector.Vector {

	averageNormal := col.AverageNormal()

	temp, _ := averageNormal.Cross(movementVec)

	if temp.Magnitude() == 0 {
		return movementVec.Invert()
	}

	out, _ := temp.Cross(averageNormal)

	return out
}

// AverageSlope returns the average slope of the Collision (ranging from 0, pointing straight up, to pi pointing straight down).
// This average is spread across all intersections contained within the Collision.
func (result *Collision) AverageSlope() float64 {
	slope := result.Intersections[0].Slope()
	slopeCount := 1.0
	for i := 1; i < len(result.Intersections); i++ {
		inter := result.Intersections[i]
		if inter.MTV.Magnitude() > 0 {
			slope += inter.Slope()
			slopeCount++
		}
	}
	return slope / slopeCount
}

// AverageContactPoint returns the average contact point out of the contact points of all Intersections
// contained within the Collision.
func (result *Collision) AverageContactPoint() vector.Vector {

	contactPoint := vector.Vector{0, 0, 0}

	for _, inter := range result.Intersections {
		vector.In(contactPoint).Add(inter.ContactPoint)
	}

	contactPoint[0] /= float64(len(result.Intersections))
	contactPoint[1] /= float64(len(result.Intersections))
	contactPoint[2] /= float64(len(result.Intersections))

	return contactPoint
}

// BoundingObject represents a Node type that can be tested for collision. The exposed functions are essentially just
// concerning whether an object that implements BoundingObject is colliding with another BoundingObject, and
// if so, by how much.
type BoundingObject interface {
	// Colliding returns true if the BoundingObject is intersecting the other BoundingObject.
	Colliding(other BoundingObject) bool
	// Collision returns a Collision if the BoundingObject is intersecting another BoundingObject. If
	// no intersection is reported, Collision returns nil.
	Collision(other BoundingObject) *Collision
	// CollisionTest performs an collision test if the bounding object were to move in the given direction in world space.
	// It returns all valid Collisions across all recursive children of the INodes slice passed in as others, testing against BoundingObjects in those trees.
	// To exemplify this, if you had a Model that had a BoundingObject child, and then tested the Model for collision,
	// the Model's children would be tested for collision (which means the BoundingObject), and the Model would be the
	// collided object. Of course, if you simply tested the BoundingObject directly, then it would return the BoundingObject as the collided
	// object.
	// Collisions will be sorted in order of distance. If no Collisions occurred, it will return an empty slice.
	CollisionTest(dx, dy, dz float64, others ...INode) []*Collision
	// CollisionTestVec performs an collision test if the bounding object were to move in the given direction in world space using a vector.
	// It returns all valid Collisions across all recursive children of the INodes slice passed in as others, testing against BoundingObjects in those trees.
	// To exemplify this, if you had a Model that had a BoundingObject child, and then tested the Model for collision,
	// the Model's children would be tested for collision (which means the BoundingObject), and the Model would be the
	// collided object. Of course, if you simply tested the BoundingObject directly, then it would return the BoundingObject as the collided
	// object.
	// Collisions will be sorted in order of distance. If no Collisions occurred, it will return an empty slice.
	CollisionTestVec(moveVec vector.Vector, others ...INode) []*Collision
}

// The below set of bt functions are used to test for intersection between BoundingObject pairs.
// I forget now, but I guess when I wrote this, bt* stood for Bounding Test, haha.

func btSphereSphere(sphereA, sphereB *BoundingSphere) *Collision {

	spherePos := sphereA.WorldPosition()
	bPos := sphereB.WorldPosition()

	sphereRadius := sphereA.WorldRadius()
	bRadius := sphereB.WorldRadius()

	delta := fastVectorSub(bPos, spherePos)
	dist := delta.Magnitude()
	delta = delta.Unit().Invert()

	s2 := sphereRadius + bRadius
	if dist > s2 {
		return nil
	}

	result := newCollision(sphereB)

	result.add(
		&Intersection{
			StartingPoint: spherePos,
			ContactPoint:  bPos.Add(delta.Scale(bRadius)),
			MTV:           delta.Scale(s2 - dist),
			Normal:        delta,
		},
	)

	return result

}

func btSphereAABB(sphere *BoundingSphere, aabb *BoundingAABB) *Collision {

	spherePos := sphere.WorldPosition()
	sphereRadius := sphere.WorldRadius()

	intersection := aabb.ClosestPoint(spherePos)

	distance := fastVectorSub(spherePos, intersection).Magnitude()

	if distance > sphereRadius {
		return nil
	}

	delta := fastVectorSub(spherePos, intersection).Unit().Scale(sphereRadius - distance)

	return newCollision(aabb).add(
		&Intersection{
			StartingPoint: spherePos,
			ContactPoint:  intersection,
			MTV:           delta,
			Normal:        aabbNormalGuess(delta),
		},
	)

}

func btSphereTriangles(sphere *BoundingSphere, triangles *BoundingTriangles) *Collision {

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !sphere.Colliding(triangles.BoundingAABB) {
		return nil
	}

	triTrans := triangles.Transform()
	invertedTransform := triTrans.Inverted()
	transformNoLoc := triTrans.SetRow(3, vector.Vector{0, 0, 0, 1})
	spherePos := invertedTransform.MultVec(sphere.WorldPosition())
	sphereRadius := sphere.WorldRadius() * math.Abs(math.Max(invertedTransform[0][0], math.Max(invertedTransform[1][1], invertedTransform[2][2])))

	result := newCollision(triangles)

	tris := triangles.Broadphase.GetTrianglesFromBounding(sphere)

	for triID := range tris {

		tri := triangles.Mesh.Triangles[triID]

		// MaxSpan / 0.66 because if you have a triangle where the two vertices are very close to each other, they'll pull the triangle center
		// towards them by twice as much as the third vertex (i.e. the center won't be in the center)
		if fastVectorSub(spherePos, tri.Center).Magnitude() > (tri.MaxSpan*0.66)+sphereRadius {
			continue
		}

		v0 := triangles.Mesh.VertexPositions[tri.ID*3]
		v1 := triangles.Mesh.VertexPositions[tri.ID*3+1]
		v2 := triangles.Mesh.VertexPositions[tri.ID*3+2]

		closest := closestPointOnTri(spherePos, v0, v1, v2)
		delta := fastVectorSub(spherePos, closest)

		if mag := delta.Magnitude(); mag <= sphereRadius {
			result.add(
				&Intersection{
					StartingPoint: spherePos,
					ContactPoint:  triangles.Transform().MultVec(closest),
					MTV:           transformNoLoc.MultVec(delta.Unit().Scale(sphereRadius - mag)),
					Triangle:      tri,
					Normal:        transformNoLoc.MultVec(tri.Normal).Unit(),
				},
			)
		}

	}

	if len(result.Intersections) == 0 {
		return nil
	}

	result.sortResults()

	return result

}

func btAABBAABB(aabbA, aabbB *BoundingAABB) *Collision {

	aPos := aabbA.WorldPosition()
	bPos := aabbB.WorldPosition()
	aSize := aabbA.Dimensions.Size().Scale(0.5)
	bSize := aabbB.Dimensions.Size().Scale(0.5)

	dx := bPos[0] - aPos[0]
	px := (bSize[0] + aSize[0]) - math.Abs(dx)

	if px <= 0 {
		return nil
	}

	dy := bPos[1] - aPos[1]
	py := (bSize[1] + aSize[1]) - math.Abs(dy)

	if py <= 0 {
		return nil
	}

	dz := bPos[2] - aPos[2]
	pz := (bSize[2] + aSize[2]) - math.Abs(dz)

	if pz <= 0 {
		return nil
	}

	result := newCollision(aabbB)

	if px < py && px < pz {
		sx := -1.0
		if math.Signbit(dx) {
			sx = 1
		}

		result.add(&Intersection{
			StartingPoint: aPos,
			ContactPoint:  vector.Vector{aPos[0] + (aSize[0] * sx), bPos[1], bPos[2]},
			MTV:           vector.Vector{px * sx, 0, 0},
			Normal:        vector.Vector{sx, 0, 0},
		})

	} else if py < pz && py < px {
		sy := -1.0
		if math.Signbit(dy) {
			sy = 1
		}

		result.add(&Intersection{
			StartingPoint: aPos,
			ContactPoint:  vector.Vector{bPos[0], aPos[1] + (aSize[1] * sy), bPos[2]},
			MTV:           vector.Vector{0, py * sy, 0},
			Normal:        vector.Vector{0, sy, 0},
		})

	} else {

		sz := -1.0
		if math.Signbit(dz) {
			sz = 1
		}

		result.add(&Intersection{
			StartingPoint: aPos,
			ContactPoint:  vector.Vector{bPos[0], bPos[1], aPos[2] + (aSize[2] * sz)},
			MTV:           vector.Vector{0, 0, pz * sz},
			Normal:        vector.Vector{0, 0, sz},
		})

	}

	return result

}

func btAABBTriangles(box *BoundingAABB, triangles *BoundingTriangles) *Collision {
	// See https://gdbooks.gitbooks.io/3dcollisions/content/Chapter4/aabb-triangle.html

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !box.Colliding(triangles.BoundingAABB) {
		return nil
	}

	boxPos := box.WorldPosition()
	boxSize := box.Dimensions.Size().Scale(0.5)

	transform := triangles.Transform()
	transformNoLoc := transform.SetRow(3, vector.Vector{0, 0, 0, 1})

	result := newCollision(triangles)

	tris := triangles.Broadphase.GetTrianglesFromBounding(box)

	for triID := range tris {

		tri := triangles.Mesh.Triangles[triID]

		v0 := transform.MultVec(triangles.Mesh.VertexPositions[tri.ID*3]).Sub(boxPos)
		v1 := transform.MultVec(triangles.Mesh.VertexPositions[tri.ID*3+1]).Sub(boxPos)
		v2 := transform.MultVec(triangles.Mesh.VertexPositions[tri.ID*3+2]).Sub(boxPos)
		// tc := v0.Add(v1).Add(v2).Scale(1.0 / 3.0)

		ab := v1.Sub(v0).Unit()
		bc := v2.Sub(v1).Unit()
		ca := v0.Sub(v2).Unit()

		axes := []vector.Vector{

			vector.X,
			vector.Y,
			vector.Z,

			vectorCross(vector.X, ab, bc),
			vectorCross(vector.X, bc, ca),
			vectorCross(vector.X, ca, ab),

			vectorCross(vector.Y, ab, bc),
			vectorCross(vector.Y, bc, ca),
			vectorCross(vector.Y, ca, ab),

			vectorCross(vector.Z, ab, bc),
			vectorCross(vector.Z, bc, ca),
			vectorCross(vector.Z, ca, ab),

			transformNoLoc.MultVec(tri.Normal),
		}

		var overlapAxis vector.Vector
		smallestOverlap := math.MaxFloat64

		for _, axis := range axes {

			if axis == nil {
				return nil
			}

			axis = axis.Unit()

			p1 := project(axis, v0, v1, v2)

			r := boxSize[0]*math.Abs(dot(vector.X, axis)) +
				boxSize[1]*math.Abs(dot(vector.Y, axis)) +
				boxSize[2]*math.Abs(dot(vector.Z, axis))

			p2 := projection{
				Max: r,
				Min: -r,
			}

			overlap := p1.Overlap(p2)

			if !p1.IsOverlapping(p2) {
				overlapAxis = nil
				break
			}

			if overlap < smallestOverlap {
				smallestOverlap = overlap
				overlapAxis = axis
			}

		}

		if overlapAxis != nil {
			mtv := overlapAxis.Scale(smallestOverlap)

			result.add(&Intersection{
				StartingPoint: boxPos,
				ContactPoint:  closestPointOnTri(vector.Vector{0, 0, 0}, v0, v1, v2).Add(boxPos),
				MTV:           mtv,
				Triangle:      tri,
				Normal:        axes[12],
			})
		}

	}

	if len(result.Intersections) == 0 {
		return nil
	}

	result.sortResults()

	return result

}

func btTrianglesTriangles(trianglesA, trianglesB *BoundingTriangles) *Collision {
	// See https://gdbooks.gitbooks.io/3dcollisions/content/Chapter4/aabb-triangle.html

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !trianglesA.BoundingAABB.Colliding(trianglesB.BoundingAABB) {
		return nil
	}

	transformA := trianglesA.Transform()
	transformB := trianglesB.Transform()

	transformedA := [][]vector.Vector{}
	transformedB := [][]vector.Vector{}

	result := newCollision(trianglesB)

	for _, meshPart := range trianglesA.Mesh.MeshParts {

		mesh := meshPart.Mesh

		for i := meshPart.TriangleStart; i < meshPart.TriangleEnd; i++ {

			tri := mesh.Triangles[i]

			v0 := transformA.MultVec(mesh.VertexPositions[tri.ID*3])
			v1 := transformA.MultVec(mesh.VertexPositions[tri.ID*3+1])
			v2 := transformA.MultVec(mesh.VertexPositions[tri.ID*3+2])

			transformedA = append(transformedA,
				[]vector.Vector{
					v0, v1, v2,
					v1.Sub(v0).Unit(),
					v2.Sub(v1).Unit(),
					v0.Sub(v2).Unit(),
					transformA.MultVec(tri.Normal),
				},
			)

		}

	}

	bTris := []*Triangle{}

	for _, meshPart := range trianglesB.Mesh.MeshParts {

		mesh := meshPart.Mesh

		for i := meshPart.TriangleStart; i < meshPart.TriangleEnd; i++ {

			tri := mesh.Triangles[i]

			v0 := transformB.MultVec(mesh.VertexPositions[tri.ID*3])
			v1 := transformB.MultVec(mesh.VertexPositions[tri.ID*3+1])
			v2 := transformB.MultVec(mesh.VertexPositions[tri.ID*3+2])

			bTris = append(bTris, tri)

			transformedB = append(transformedB,
				[]vector.Vector{
					v0, v1, v2,
					v1.Sub(v0).Unit(),
					v2.Sub(v1).Unit(),
					v0.Sub(v2).Unit(),
					transformB.MultVec(tri.Normal),
				},
			)

		}

	}

	for _, a := range transformedA {

		for bTriIndex, b := range transformedB {

			axes := []vector.Vector{

				vectorCross(a[3], b[3], b[4]),
				vectorCross(a[3], b[4], b[5]),
				vectorCross(a[3], b[5], b[6]),

				vectorCross(a[4], b[3], b[4]),
				vectorCross(a[4], b[4], b[5]),
				vectorCross(a[4], b[5], b[6]),

				vectorCross(a[5], b[3], b[4]),
				vectorCross(a[5], b[4], b[5]),
				vectorCross(a[5], b[5], b[6]),

				transformA.MultVec(a[6]),
				transformB.MultVec(b[6]),
			}

			var overlapAxis vector.Vector
			smallestOverlap := math.MaxFloat64

			for _, axis := range axes {

				if axis == nil {
					return nil
				}

				axis = axis.Unit()

				p1 := project(axis, a[0], a[1], a[2])
				p2 := project(axis, b[0], b[1], b[2])

				overlap := p1.Overlap(p2)

				if !p1.IsOverlapping(p2) {
					overlapAxis = nil
					break
				}

				if overlap < smallestOverlap {
					smallestOverlap = overlap
					overlapAxis = axis
				}

			}

			if overlapAxis != nil {
				mtv := overlapAxis.Scale(smallestOverlap)
				result.add(
					&Intersection{
						StartingPoint: transformA.MultVec(bTris[bTriIndex].Center),
						// ContactPoint: b[0].Add(b[1]).Add(b[2]).Scale(1.0 / 3.0),
						ContactPoint: trianglesB.WorldPosition().Add(mtv),
						MTV:          mtv,
						Triangle:     bTris[bTriIndex],
						Normal:       b[6],
					},
				)
			}

		}

	}

	if len(result.Intersections) == 0 {
		return nil
	}

	result.sortResults()

	return result

}

func btCapsuleCapsule(capsuleA, capsuleB *BoundingCapsule) *Collision {
	capsuleA.internalSphere.SetLocalScaleVec(capsuleA.LocalScale())

	// By getting the closest point to the world position (center), and then getting it again, we get closer to the
	// true closest point for both capsules, which is good enough for now lol
	caClosest := capsuleA.ClosestPoint(capsuleB.WorldPosition())
	cbClosest := capsuleB.ClosestPoint(capsuleA.WorldPosition())

	capsuleA.internalSphere.SetLocalPositionVec(capsuleA.ClosestPoint(cbClosest))
	capsuleA.internalSphere.Radius = capsuleA.Radius

	capsuleB.internalSphere.SetLocalScaleVec(capsuleB.LocalScale())
	capsuleB.internalSphere.SetLocalPositionVec(capsuleB.ClosestPoint(caClosest))
	capsuleB.internalSphere.Radius = capsuleB.Radius

	return btSphereSphere(capsuleA.internalSphere, capsuleB.internalSphere)
}

func btSphereCapsule(sphere *BoundingSphere, capsule *BoundingCapsule) *Collision {
	capsule.internalSphere.SetLocalScaleVec(capsule.LocalScale())
	capsule.internalSphere.SetLocalPositionVec(capsule.ClosestPoint(sphere.WorldPosition()))
	capsule.internalSphere.Radius = capsule.Radius
	return btSphereSphere(sphere, capsule.internalSphere)
}

func btCapsuleAABB(capsule *BoundingCapsule, aabb *BoundingAABB) *Collision {
	capsule.internalSphere.SetLocalScaleVec(capsule.LocalScale())
	capsule.internalSphere.SetLocalPositionVec(capsule.ClosestPoint(aabb.WorldPosition()))
	capsule.internalSphere.Radius = capsule.Radius
	return btSphereAABB(capsule.internalSphere, aabb)
}

func btCapsuleTriangles(capsule *BoundingCapsule, triangles *BoundingTriangles) *Collision {

	capsule.internalSphere.SetLocalScaleVec(capsule.LocalScale())
	capsule.internalSphere.SetLocalPositionVec(capsule.ClosestPoint(triangles.BoundingAABB.WorldPosition()))
	capsule.internalSphere.Radius = capsule.Radius

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !capsule.internalSphere.Colliding(triangles.BoundingAABB) {
		return nil
	}

	triTrans := triangles.Transform()
	invertedTransform := triTrans.Inverted()
	transformNoLoc := triTrans.SetRow(3, vector.Vector{0, 0, 0, 1})

	capsuleRadius := capsule.WorldRadius() * math.Abs(math.Max(invertedTransform[0][0], math.Max(invertedTransform[1][1], invertedTransform[2][2])))
	capsuleRadiusSquared := math.Pow(capsuleRadius, 2)

	capsuleTop := invertedTransform.MultVec(capsule.lineTop())
	capsuleBottom := invertedTransform.MultVec(capsule.lineBottom())
	capsulePosition := invertedTransform.MultVec(capsule.WorldPosition())
	capsuleLine := capsuleTop.Sub(capsuleBottom)
	capSpread := capsuleLine.Magnitude() + capsuleRadius
	capDot := dot(capsuleLine, capsuleLine)

	var closestCapsulePoint vector.Vector

	result := newCollision(triangles)

	tris := triangles.Broadphase.GetTrianglesFromBounding(capsule)

	spherePos := vector.Vector{0, 0, 0}

	closestSub := vector.Vector{0, 0, 0}

	for triID := range tris {

		tri := triangles.Mesh.Triangles[triID]

		if fastVectorDistanceSquared(capsulePosition, tri.Center) > math.Pow((tri.MaxSpan*0.66)+capSpread, 2) {
			continue
		}

		if fastVectorDistanceSquared(tri.Center, capsuleTop) < fastVectorDistanceSquared(tri.Center, capsuleBottom) {
			closestCapsulePoint = capsuleTop
		} else {
			closestCapsulePoint = capsuleBottom
		}

		v0 := triangles.Mesh.VertexPositions[tri.ID*3]
		v1 := triangles.Mesh.VertexPositions[tri.ID*3+1]
		v2 := triangles.Mesh.VertexPositions[tri.ID*3+2]

		closest := closestPointOnTri(closestCapsulePoint, v0, v1, v2)
		closestSub[0] = closest[0] - capsuleBottom[0]
		closestSub[1] = closest[1] - capsuleBottom[1]
		closestSub[2] = closest[2] - capsuleBottom[2]

		// Doing this manually to avoid doing as much as possible~

		t := dot(closestSub, capsuleLine) / capDot

		if t > 1 {
			t = 1
		}
		if t < 0 {
			t = 0
		}

		spherePos[0] = capsuleBottom[0] + (capsuleLine[0] * t)
		spherePos[1] = capsuleBottom[1] + (capsuleLine[1] * t)
		spherePos[2] = capsuleBottom[2] + (capsuleLine[2] * t)

		delta := fastVectorSub(spherePos, closest)

		if mag := fastVectorMagnitudeSquared(delta); mag <= capsuleRadiusSquared {

			result.add(
				&Intersection{
					StartingPoint: closest,
					ContactPoint:  triTrans.MultVec(closest),
					MTV:           transformNoLoc.MultVec(delta.Unit().Scale(capsuleRadiusSquared - mag)),
					Triangle:      tri,
					Normal:        transformNoLoc.MultVec(tri.Normal).Unit(),
				},
			)

		}

		// if fastVectorSub(capsulePosition, tri.Center).Magnitude() > (tri.MaxSpan*0.66)+capSpread {
		// 	continue
		// }

		// if fastVectorDistanceSquared(tri.Center, capsuleTop) < fastVectorDistanceSquared(tri.Center, capsuleBottom) {
		// 	closestCapsulePoint = capsuleTop
		// } else {
		// 	closestCapsulePoint = capsuleBottom
		// }

		// v0 := triangles.Mesh.VertexPositions[tri.ID*3]
		// v1 := triangles.Mesh.VertexPositions[tri.ID*3+1]
		// v2 := triangles.Mesh.VertexPositions[tri.ID*3+2]

		// closest := closestPointOnTri(closestCapsulePoint, v0, v1, v2)

		// // Doing this manually to avoid doing as much as possible~

		// t := dot(closest.Sub(capsuleBottom), capsuleLine) / capDot
		// t = math.Max(math.Min(t, 1), 0)
		// spherePos := capsuleBottom.Add(capsuleLine.Scale(t))

		// delta := fastVectorSub(spherePos, closest)

		// if mag := delta.Magnitude(); mag <= capsuleRadius {

		// 	result.add(
		// 		&Intersection{
		// 			StartingPoint: closest,
		// 			ContactPoint:  triangles.Transform().MultVec(closest),
		// 			MTV:           transformNoLoc.MultVec(delta.Unit().Scale(capsuleRadius - mag)),
		// 			Triangle:      tri,
		// 			Normal:        transformNoLoc.MultVec(tri.Normal).Unit(),
		// 		},
		// 	)

		// }

	}

	if len(result.Intersections) == 0 {
		return nil
	}

	result.sortResults()

	return result

}

func commonCollisionTest(node INode, dx, dy, dz float64, others ...INode) []*Collision {

	var ogPos vector.Vector

	// If dx, dy, and dz are 0, we don't need to reposition the node for the collision test.
	if dx != 0 || dy != 0 || dz != 0 {
		ogPos = node.LocalPosition()
		node.Move(dx, dy, dz)
	}

	collisions := []*Collision{}

	var test func(checking, parent INode)

	test = func(checking, parent INode) {

		if c, ok := checking.(BoundingObject); ok {

			if collision := node.(BoundingObject).Collision(c); collision != nil {
				collision.CollidedRoot = parent
				collisions = append(collisions, collision)
			}

		}

		for _, child := range checking.Children() {
			test(child, parent)
		}

	}

	for _, o := range others {
		test(o, o)
	}

	// Sort the IntersectionResults by distance (closer intersections come up "sooner").
	sort.Slice(collisions, func(i, j int) bool {
		return fastVectorDistanceSquared(collisions[i].AverageContactPoint(), collisions[i].Intersections[0].StartingPoint) >
			fastVectorDistanceSquared(collisions[j].AverageContactPoint(), collisions[j].Intersections[0].StartingPoint)
	})

	if dx != 0 || dy != 0 || dz != 0 {
		node.SetLocalPositionVec(ogPos)
	}

	return collisions

}

type projection struct {
	Min, Max float64
}

func project(axis vector.Vector, points ...vector.Vector) projection {

	projection := projection{}
	projection.Min = dot(axis, points[0])
	projection.Max = projection.Min

	for _, point := range points[1:] {
		p := dot(axis, point)
		if p < projection.Min {
			projection.Min = p
		} else if p > projection.Max {
			projection.Max = p
		}

	}

	// margin := 0.01
	// projection.Min -= margin
	// projection.Max += margin

	return projection

}

func (projection projection) Overlap(other projection) float64 {
	if !projection.IsOverlapping(other) {
		return 0
	}

	if projection.Max > other.Min {
		return projection.Max - other.Min
	}
	return projection.Min - other.Max

}

func (projection projection) IsOverlapping(other projection) bool {
	return !(projection.Min > other.Max || other.Min > projection.Max)
}

var sphereCheck = NewBoundingSphere("sphere check", 1)

// SphereCheck performs a quick bounding sphere check at the specified X, Y, and Z position with the radius given,
// against the bounding objects provided in "others".
func SphereCheck(x, y, z, radius float64, others ...INode) []*Collision {
	sphereCheck.SetLocalPosition(x, y, z)
	sphereCheck.Radius = radius
	return commonCollisionTest(sphereCheck, 0, 0, 0, others...)
}

// SphereCheck performs a quick bounding sphere check at the specified position with the radius given, against the
// bounding objects provided in "others".
func SphereCheckVec(position vector.Vector, radius float64, others ...INode) []*Collision {
	sphereCheck.SetLocalPositionVec(position)
	sphereCheck.Radius = radius
	return commonCollisionTest(sphereCheck, 0, 0, 0, others...)
}
