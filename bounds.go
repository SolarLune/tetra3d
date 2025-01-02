package tetra3d

import (
	"math"
	"sort"

	"github.com/solarlune/tetra3d/math32"
)

type Intersection struct {
	StartingPoint Vector3   // The starting point for the initiating object in the intersection in world space; either the center of the object for sphere / aabb, the center of the closest point for capsules, or the triangle position for triangles.
	ContactPoint  Vector3   // The contact point for the intersection on the second, collided object in world space (i.e. the point of collision on the triangle in a Sphere>Triangles test).
	MTV           Vector3   // MTV represents the minimum translation vector to remove the calling object from the intersecting object.
	Triangle      *Triangle // Triangle represents the triangle that was intersected in intersection tests that involve triangle meshes; if no triangle mesh was tested against, then this will be nil.
	Normal        Vector3
}

// Slope returns the slope of the intersection's normal, in radians. This ranges from 0 (straight up) to pi (straight down).
func (intersection *Intersection) Slope() float32 {
	return WorldUp.Angle(intersection.Normal)
}

// SlideAgainstNormal takes an input vector and alters it to slide against the intersection's returned normal.
func (intersection *Intersection) SlideAgainstNormal(movementVec Vector3) Vector3 {

	temp := intersection.Normal.Cross(movementVec)

	if temp.Magnitude() == 0 {
		return Vector3{}
	}

	out := temp.Cross(intersection.Normal)

	return out

}

// Collision represents the result of a collision test. A Collision test may result in multiple intersections, and
// so an Collision holds each of these individual intersections in its Intersections slice.
// The intersections are sorted in order of distance from the starting point of the intersection (the center of the
// colliding sphere / aabb, the closest point in the capsule, the center of the closest triangle, etc) to the
// contact point.
type Collision struct {
	BoundingObject IBoundingObject // The BoundingObject collided with
	Intersections  []*Intersection // The slice of Intersections, one for each object or triangle intersected with, arranged in order of distance (far to close).
}

func newCollision(collidedObject IBoundingObject) *Collision {
	return &Collision{
		BoundingObject: collidedObject,
		Intersections:  []*Intersection{},
	}
}

func (col *Collision) add(intersection *Intersection) *Collision {
	col.Intersections = append(col.Intersections, intersection)
	return col
}

// sort the intersections by distance from starting point (which should be the same for all collisions except for triangle-triangle) to contact point.
func (col *Collision) sortResults() {
	sort.Slice(col.Intersections, func(i, j int) bool {
		return col.Intersections[i].StartingPoint.DistanceSquared(col.Intersections[i].ContactPoint) >
			col.Intersections[j].StartingPoint.DistanceSquared(col.Intersections[j].ContactPoint)
	})
}

// AverageMTV returns the average MTV (minimum translation vector) from all Intersections contained within the Collision.
// To be specific, this isn't actually the pure average, but rather is the result of adding together all MTVs from Intersections
// in the Collision for the direction, and using the greatest MTV's magnitude for the distance of the returned vector. In other
// words, AverageMTV returns the MTV to move in that should resolve all intersections from the Collision.
func (col *Collision) AverageMTV() Vector3 {
	greatestDist := float32(0.0)
	mtv := Vector3{}
	for _, inter := range col.Intersections {
		if inter.MTV.IsInf() || inter.MTV.IsNaN() {
			continue
		}
		mag := inter.MTV.Magnitude()
		if mag > greatestDist {
			greatestDist = mag
		}
		mtv = mtv.Add(inter.MTV)
	}
	mtv = mtv.Unit().Scale(greatestDist)
	return mtv
}

// AverageNormal returns the average normal vector from all Intersections contained within the Collision.
func (col *Collision) AverageNormal() Vector3 {
	normal := col.Intersections[0].Normal
	for i := 1; i < len(col.Intersections); i++ {
		normal = normal.Add(col.Intersections[i].Normal)
	}
	normal = normal.Scale(1.0 / float32(len(col.Intersections)))
	return normal
}

// SlideAgainstAverageNormal takes an input movement vector and alters it to slide against the Collision's average normal.
func (col *Collision) SlideAgainstAverageNormal(movementVec Vector3) Vector3 {

	averageNormal := col.AverageNormal()

	temp := averageNormal.Cross(movementVec)

	if temp.Magnitude() == 0 {
		return Vector3{}
	}

	out := temp.Cross(averageNormal)

	return out
}

// AverageSlope returns the average slope of the Collision (ranging from 0, pointing straight up, to pi pointing straight down).
// This average is spread across all intersections contained within the Collision.
func (result *Collision) AverageSlope() float32 {
	slope := result.Intersections[0].Slope()
	slopeCount := float32(1.0)
	for i := 1; i < len(result.Intersections); i++ {
		inter := result.Intersections[i]
		if inter.MTV.Magnitude() > 0 {
			slope += inter.Slope()
			slopeCount++
		}
	}
	return slope / slopeCount
}

// AverageContactPoint returns the average world contact point out of the contact points of all Intersections
// contained within the Collision.
func (result *Collision) AverageContactPoint() Vector3 {

	contactPoint := Vector3{}

	for _, inter := range result.Intersections {
		contactPoint = contactPoint.Add(inter.ContactPoint)
	}

	contactPoint = contactPoint.Divide(float32(len(result.Intersections)))

	return contactPoint
}

// CollisionTestSettings controls how a CollisionTest() call evaluates.
type CollisionTestSettings struct {

	// TestAgainst controls what objects to test against.
	TestAgainst NodeIterator

	// OnCollision is a callback function that is called when a valid collision test happens between the
	// calling object and one of the valid INodes contained within the Others slice. The callback should return a boolean
	// indicating if the test should continue after evaluating this collision (true) or not (false).
	//
	// Because this function is called whenever a Collision is found, anything done in this function will influence
	// following possible Collisions. To illustrate this, let's say that you had object A that is colliding with objects B and C.
	// If the collision with B is detected first, and you move A away so that it is no longer colliding with B or C, then the collision
	// with C would not be detected (and OnCollision would not be called in this case).
	//
	// OnCollision is called in order of distance to intersection points (so two objects might have the same distance to their respective intersection points of a larger object).
	// This being the case, you may need to store and re-sort the collisions to fit your needs.
	// index is the index of the collision out of the total number of collisions, which is the count.
	OnCollision func(col *Collision, index, count int) bool
}

// IBoundingObject represents a Node type that can be tested for collision. The exposed functions are essentially just
// concerning whether an object that implements IBoundingObject is colliding with another IBoundingObject, and
// if so, by how much.
type IBoundingObject interface {
	INode
	// Colliding returns true if the BoundingObject is intersecting the other BoundingObject.
	Colliding(other IBoundingObject) bool
	// Collision returns a Collision if the BoundingObject is intersecting another BoundingObject. If
	// no intersection is reported, Collision returns nil.
	Collision(other IBoundingObject) *Collision

	// CollisionTest performs a distance-ordered collision test using the provided collision test settings structure.
	CollisionTest(settings CollisionTestSettings) bool
}

// The below set of bt functions are used to test for intersection between BoundingObject pairs.
// I forget now, but I guess when I wrote this, bt* stood for Bounding Test, haha.
func btSphereSphere(sphereA, sphereB *BoundingSphere) *Collision {

	spherePos := sphereA.WorldPosition()
	bPos := sphereB.WorldPosition()

	sphereRadius := sphereA.WorldRadius()
	bRadius := sphereB.WorldRadius()

	delta := bPos.Sub(spherePos)
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

	distance := spherePos.Distance(intersection)

	if distance > sphereRadius {
		return nil
	}

	delta := spherePos.Sub(intersection).Unit().Scale(sphereRadius - distance)

	return newCollision(aabb).add(
		&Intersection{
			StartingPoint: spherePos,
			ContactPoint:  intersection,
			MTV:           delta,
			Normal:        aabb.normalFromContactPoint(intersection),
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
	transformNoLoc := triTrans.Clone()
	transformNoLoc.SetRow(3, Vector4{0, 0, 0, 1})
	sphereWorldPosition := sphere.WorldPosition()
	spherePos := invertedTransform.MultVec(sphereWorldPosition)
	sphereRadius := sphere.WorldRadius() * math32.Abs(math32.Max(invertedTransform[0][0], math32.Max(invertedTransform[1][1], invertedTransform[2][2])))

	result := newCollision(triangles)

	tris := triangles.Broadphase.TrianglesFromBoundingObject(sphere)

	for triID := range tris {

		tri := triangles.Mesh.Triangles[triID]

		// MaxSpan / 0.66 because if you have a triangle where the two vertices are very close to each other, they'll pull the triangle center
		// towards them by twice as much as the third vertex (i.e. the center won't be in the center)
		if spherePos.Distance(tri.Center) > (tri.MaxSpan*0.66)+sphereRadius {
			continue
		}

		v0 := triangles.Mesh.VertexPositions[tri.VertexIndices[0]]
		v1 := triangles.Mesh.VertexPositions[tri.VertexIndices[1]]
		v2 := triangles.Mesh.VertexPositions[tri.VertexIndices[2]]

		closest := closestPointOnTri(spherePos, v0, v1, v2)
		delta := spherePos.Sub(closest)

		if mag := delta.Magnitude(); mag <= sphereRadius {
			result.add(
				&Intersection{
					StartingPoint: sphereWorldPosition,
					ContactPoint:  triTrans.MultVec(closest),
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

	dx := bPos.X - aPos.X
	px := (bSize.X + aSize.X) - math32.Abs(dx)

	if px <= 0 {
		return nil
	}

	dy := bPos.Y - aPos.Y
	py := (bSize.Y + aSize.Y) - math32.Abs(dy)

	if py <= 0 {
		return nil
	}

	dz := bPos.Z - aPos.Z
	pz := (bSize.Z + aSize.Z) - math32.Abs(dz)

	if pz <= 0 {
		return nil
	}

	result := newCollision(aabbB)

	if px < py && px < pz {
		sx := float32(-1.0)
		if math32.Signbit(dx) {
			sx = 1
		}

		result.add(&Intersection{
			StartingPoint: aPos,
			ContactPoint:  Vector3{aPos.X + (aSize.X * sx), bPos.Y, bPos.Z},
			MTV:           Vector3{px * sx, 0, 0},
			Normal:        Vector3{sx, 0, 0},
		})

	} else if py < pz && py < px {
		sy := float32(-1.0)
		if math32.Signbit(dy) {
			sy = 1
		}

		result.add(&Intersection{
			StartingPoint: aPos,
			ContactPoint:  Vector3{bPos.X, aPos.Y + (aSize.Y * sy), bPos.Z},
			MTV:           Vector3{0, py * sy, 0},
			Normal:        Vector3{0, sy, 0},
		})

	} else {

		sz := float32(-1.0)
		if math32.Signbit(dz) {
			sz = 1
		}

		result.add(&Intersection{
			StartingPoint: aPos,
			ContactPoint:  Vector3{bPos.X, bPos.Y, aPos.Z + (aSize.Z * sz)},
			MTV:           Vector3{0, 0, pz * sz},
			Normal:        Vector3{0, 0, sz},
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
	transformNoLoc := transform.Clone()
	transformNoLoc.SetRow(3, Vector4{0, 0, 0, 1})

	result := newCollision(triangles)

	tris := triangles.Broadphase.TrianglesFromBoundingObject(box)

	for triID := range tris {

		tri := triangles.Mesh.Triangles[triID]

		v0 := transform.MultVec(triangles.Mesh.VertexPositions[tri.VertexIndices[0]]).Sub(boxPos)
		v1 := transform.MultVec(triangles.Mesh.VertexPositions[tri.VertexIndices[1]]).Sub(boxPos)
		v2 := transform.MultVec(triangles.Mesh.VertexPositions[tri.VertexIndices[2]]).Sub(boxPos)
		// tc := v0.Add(v1).Add(v2).Scale(1.0 / 3.0)

		ab := v1.Sub(v0).Unit()
		bc := v2.Sub(v1).Unit()
		ca := v0.Sub(v2).Unit()

		axes := []Vector3{

			WorldRight,
			WorldUp,
			WorldBackward,

			WorldRight.Cross(ab),
			WorldRight.Cross(bc),
			WorldRight.Cross(ca),

			WorldUp.Cross(ab),
			WorldUp.Cross(bc),
			WorldUp.Cross(ca),

			WorldBackward.Cross(ab),
			WorldBackward.Cross(bc),
			WorldBackward.Cross(ca),

			transformNoLoc.MultVec(tri.Normal),
		}

		var overlapAxis Vector3
		smallestOverlap := float32(math.MaxFloat32)

		for _, axis := range axes {

			if axis.IsZero() {
				return nil
			}

			axis = axis.Unit()

			p1 := project(axis, v0, v1, v2)

			r := boxSize.X*math32.Abs(WorldRight.Dot(axis)) +
				boxSize.Y*math32.Abs(WorldUp.Dot(axis)) +
				boxSize.Z*math32.Abs(WorldBackward.Dot(axis))

			p2 := projection{
				Max: r,
				Min: -r,
			}

			overlap := p1.Overlap(p2)

			if !p1.IsOverlapping(p2) {
				overlapAxis = Vector3{}
				break
			}

			if overlap < smallestOverlap {
				smallestOverlap = overlap
				overlapAxis = axis
			}

		}

		if !overlapAxis.IsZero() {
			mtv := overlapAxis.Scale(smallestOverlap)

			result.add(&Intersection{
				StartingPoint: boxPos,
				ContactPoint:  closestPointOnTri(Vector3{0, 0, 0}, v0, v1, v2).Add(boxPos),
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

	transformedA := [][]Vector3{}
	transformedB := [][]Vector3{}

	result := newCollision(trianglesB)

	for _, meshPart := range trianglesA.Mesh.MeshParts {

		mesh := meshPart.Mesh

		meshPart.ForEachTri(func(tri *Triangle) {

			v0 := transformA.MultVec(mesh.VertexPositions[tri.VertexIndices[0]])
			v1 := transformA.MultVec(mesh.VertexPositions[tri.VertexIndices[1]])
			v2 := transformA.MultVec(mesh.VertexPositions[tri.VertexIndices[2]])

			transformedA = append(transformedA,
				[]Vector3{
					v0, v1, v2,
					v1.Sub(v0).Unit(),
					v2.Sub(v1).Unit(),
					v0.Sub(v2).Unit(),
					transformA.MultVec(tri.Normal),
				},
			)

		})

	}

	bTris := []*Triangle{}

	for _, meshPart := range trianglesB.Mesh.MeshParts {

		mesh := meshPart.Mesh

		meshPart.ForEachTri(func(tri *Triangle) {

			v0 := transformB.MultVec(mesh.VertexPositions[tri.VertexIndices[0]])
			v1 := transformB.MultVec(mesh.VertexPositions[tri.VertexIndices[1]])
			v2 := transformB.MultVec(mesh.VertexPositions[tri.VertexIndices[2]])

			bTris = append(bTris, tri)

			transformedB = append(transformedB,
				[]Vector3{
					v0, v1, v2,
					v1.Sub(v0).Unit(),
					v2.Sub(v1).Unit(),
					v0.Sub(v2).Unit(),
					transformB.MultVec(tri.Normal),
				},
			)

		})

	}

	for _, a := range transformedA {

		for bTriIndex, b := range transformedB {

			axes := []Vector3{

				a[3].Cross(b[3]),
				a[3].Cross(b[4]),
				a[3].Cross(b[5]),

				a[4].Cross(b[3]),
				a[4].Cross(b[4]),
				a[4].Cross(b[5]),

				a[5].Cross(b[3]),
				a[5].Cross(b[4]),
				a[5].Cross(b[5]),

				transformA.MultVec(a[6]),
				transformB.MultVec(b[6]),
			}

			var overlapAxis Vector3
			smallestOverlap := float32(math.MaxFloat32)

			for _, axis := range axes {

				if axis.IsZero() {
					return nil
				}

				axis = axis.Unit()

				p1 := project(axis, a[0], a[1], a[2])
				p2 := project(axis, b[0], b[1], b[2])

				overlap := p1.Overlap(p2)

				if !p1.IsOverlapping(p2) {
					overlapAxis = Vector3{}
					break
				}

				if overlap < smallestOverlap {
					smallestOverlap = overlap
					overlapAxis = axis
				}

			}

			if !overlapAxis.IsZero() {
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

	col := btSphereSphere(capsuleA.internalSphere, capsuleB.internalSphere)
	if col != nil {
		col.BoundingObject = capsuleB
	}
	return col
}

func btSphereCapsule(sphere *BoundingSphere, capsule *BoundingCapsule) *Collision {
	capsule.internalSphere.SetLocalScaleVec(capsule.LocalScale())
	capsule.internalSphere.SetLocalPositionVec(capsule.ClosestPoint(sphere.WorldPosition()))
	capsule.internalSphere.Radius = capsule.Radius
	col := btSphereSphere(sphere, capsule.internalSphere)
	if col != nil {
		col.BoundingObject = capsule
	}
	return col
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
	transformNoLoc := triTrans.Clone()
	transformNoLoc.SetRow(3, Vector4{0, 0, 0, 1})

	capsuleRadius := capsule.WorldRadius() * math32.Abs(math32.Max(invertedTransform[0][0], math32.Max(invertedTransform[1][1], invertedTransform[2][2])))

	capsuleTop := invertedTransform.MultVec(capsule.lineTop())
	capsuleBottom := invertedTransform.MultVec(capsule.lineBottom())
	capsulePosition := invertedTransform.MultVec(capsule.WorldPosition())
	capsuleLine := capsuleTop.Sub(capsuleBottom)
	capSpread := capsuleLine.Magnitude() + capsuleRadius
	capDot := capsuleLine.Dot(capsuleLine)

	var closestCapsulePoint Vector3

	result := newCollision(triangles)

	tris := triangles.Broadphase.TrianglesFromBoundingObject(capsule)

	spherePos := Vector3{}

	closestSub := Vector3{}

	for triID := range tris {

		tri := triangles.Mesh.Triangles[triID]

		if capsulePosition.DistanceSquared(tri.Center) > math32.Pow((tri.MaxSpan*0.66)+capSpread, 2) {
			continue
		}

		if tri.Center.DistanceSquared(capsuleTop) < tri.Center.DistanceSquared(capsuleBottom) {
			closestCapsulePoint = capsuleTop
		} else {
			closestCapsulePoint = capsuleBottom
		}

		v0 := triangles.Mesh.VertexPositions[tri.VertexIndices[0]]
		v1 := triangles.Mesh.VertexPositions[tri.VertexIndices[1]]
		v2 := triangles.Mesh.VertexPositions[tri.VertexIndices[2]]

		closest := closestPointOnTri(closestCapsulePoint, v0, v1, v2)
		closestSub.X = closest.X - capsuleBottom.X
		closestSub.Y = closest.Y - capsuleBottom.Y
		closestSub.Z = closest.Z - capsuleBottom.Z

		// Doing this manually to avoid doing as much as possible~

		t := closestSub.Dot(capsuleLine) / capDot

		if t > 1 {
			t = 1
		}
		if t < 0 {
			t = 0
		}

		spherePos.X = capsuleBottom.X + (capsuleLine.X * t)
		spherePos.Y = capsuleBottom.Y + (capsuleLine.Y * t)
		spherePos.Z = capsuleBottom.Z + (capsuleLine.Z * t)

		delta := spherePos.Sub(closest)

		if mag := delta.Magnitude(); mag <= capsuleRadius {

			result.add(
				&Intersection{
					StartingPoint: closestCapsulePoint,
					ContactPoint:  triTrans.MultVec(closest),
					MTV:           transformNoLoc.MultVec(delta.Unit().Scale(capsuleRadius - mag)),
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
		// t = Max(Min(t, 1), 0)
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

var internalCollisionList = []*Collision{}

func commonCollisionTest(node INode, settings CollisionTestSettings) bool {

	internalCollisionList = internalCollisionList[:0]

	settings.TestAgainst.ForEach(func(checking INode) bool {

		bounds, ok := checking.(IBoundingObject)

		if !ok || node == checking || (settings.OnCollision == nil && len(internalCollisionList) > 0) {
			return true
		}

		if collision := node.(IBoundingObject).Collision(bounds); collision != nil {
			internalCollisionList = append(internalCollisionList, collision)
		}

		return true

	})

	if settings.OnCollision != nil {

		// Sort the IntersectionResults by distance (closer intersections come up "sooner").
		sort.Slice(internalCollisionList, func(i, j int) bool {
			return internalCollisionList[i].AverageContactPoint().DistanceSquared(internalCollisionList[i].Intersections[0].StartingPoint) >
				internalCollisionList[j].AverageContactPoint().DistanceSquared(internalCollisionList[j].Intersections[0].StartingPoint)
		})

		for i, c := range internalCollisionList {
			if !settings.OnCollision(c, i, len(internalCollisionList)) {
				break
			}
		}

	}

	return len(internalCollisionList) > 0

}

type projection struct {
	Min, Max float32
}

func project(axis Vector3, points ...Vector3) projection {

	projection := projection{}
	projection.Min = axis.Dot(points[0])
	projection.Max = projection.Min

	for _, point := range points[1:] {
		p := axis.Dot(point)
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

func (projection projection) Overlap(other projection) float32 {
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

var sphereTestObject = NewBoundingSphere("sphere check", 1)

// CollisionTestSphere performs a quick bounding sphere check at the specified X, Y, and Z position with the radius given,
// against the bounding objects provided in "others".
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return if a collision was found with the sphere at the settings specified.
func CollisionTestSphere(x, y, z, radius float32, settings CollisionTestSettings) bool {
	sphereTestObject.SetLocalPosition(x, y, z)
	sphereTestObject.Radius = radius
	return commonCollisionTest(sphereTestObject, settings)
}

// CollisionTestSphereVec performs a quick bounding sphere check at the specified position with the radius given, against the
// bounding objects provided in "others".
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return if a collision was found with the sphere at the settings specified.
func CollisionTestSphereVec(position Vector3, radius float32, settings CollisionTestSettings) bool {
	return CollisionTestSphere(position.X, position.Y, position.Z, radius, settings)
}

var aabbTestObject = NewBoundingAABB("aabb check", 1, 1, 1)

// CollisionTestAABB performs a quick bounding AABB check at the specified x, y, and z position using the collision settings
// provided. The bounding AABB will have the provided width, height, and depth.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return if a collision was found with the sphere at the settings specified.
// Note that AABB tests with BoundingTriangles are currently buggy.
func CollisionTestAABB(x, y, z, width, height, depth float32, settings CollisionTestSettings) bool {
	aabbTestObject.SetLocalPosition(x, y, z)
	aabbTestObject.SetDimensions(width, height, depth)
	return commonCollisionTest(aabbTestObject, settings)
}

// CollisionTestAABBVec places a bounding AABB at the position given with the specified size to perform a collision test.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return if a collision was found with the sphere at the settings specified.
// Note that AABB tests with BoundingTriangles are currently buggy.
func CollisionTestAABBVec(position, size Vector3, settings CollisionTestSettings) bool {
	return CollisionTestAABB(position.X, position.Y, position.Z, size.X, size.Y, size.Z, settings)
}

var capsuleTestObject = NewBoundingCapsule("capsule check", 2, 1)

// CollisionTestCapsule performs a quick bounding capsule check at the specified position
// and size using the collision settings provided.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return if a collision was found with the sphere at the settings specified.
func CollisionTestCapsule(x, y, z, radius, height float32, settings CollisionTestSettings) bool {
	capsuleTestObject.SetLocalPosition(x, y, z)
	capsuleTestObject.Radius = radius
	capsuleTestObject.Height = height
	return commonCollisionTest(capsuleTestObject, settings)
}

// CollisionTestCapsuleVec places a bounding capsule at the position given with the specified
// radius and height to perform a collision test.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return if a collision was found with the sphere at the settings specified.
func CollisionTestCapsuleVec(position Vector3, radius, height float32, settings CollisionTestSettings) bool {
	return CollisionTestCapsule(position.X, position.Y, position.Z, radius, height, settings)
}
