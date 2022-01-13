package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

// IntersectionResult represents the result of an intersection test.
type IntersectionResult struct {
	// The contact point between the two intersecting objects. Note that this may be the average
	// between the two overlapping shapes, rather than the point of contact specifically.
	ContactPoint vector.Vector
	MTV          vector.Vector // MTV represents the minimum translation vector to remove the calling object from the intersecting object.
	// Normal       vector.Vector
}

func NewIntersectionResult(contactPoint, mtv vector.Vector) *IntersectionResult {
	return &IntersectionResult{
		ContactPoint: contactPoint,
		MTV:          mtv,
	}
}

// BoundingObject represents a Node type that can be tested for intersection.
type BoundingObject interface {
	Intersecting(other BoundingObject) bool
	Intersection(other BoundingObject) *IntersectionResult
}

func btSphereSphere(sphereA, sphereB *BoundingSphere) *IntersectionResult {

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

	return NewIntersectionResult(bPos.Add(delta.Scale(bRadius)), delta.Scale(s2-dist))

}

func btSphereAABB(sphere *BoundingSphere, aabb *BoundingAABB) *IntersectionResult {

	spherePos := sphere.WorldPosition()
	sphereRadius := sphere.WorldRadius()

	intersection := aabb.ClosestPoint(spherePos)

	distance := fastVectorSub(spherePos, intersection).Magnitude()

	if distance > sphereRadius {
		return nil
	}

	delta := fastVectorSub(spherePos, intersection).Unit().Scale(sphereRadius - distance)

	return NewIntersectionResult(intersection, delta)

}

func btSphereTriangles(sphere *BoundingSphere, triangles *BoundingTriangles) *IntersectionResult {

	triangles.BoundingAABB.SetLocalPosition(triangles.WorldPosition().Add(triangles.Mesh.Dimensions.Center()))
	triangles.BoundingAABB.SetLocalRotation(triangles.WorldRotation())
	triangles.BoundingAABB.SetLocalScale(triangles.WorldScale())

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !sphere.Intersecting(triangles.BoundingAABB) {
		return nil
	}

	invertedTransform := triangles.Node.Transform().Inverted()
	transformNoLoc := triangles.Node.Transform().SetRow(3, vector.Vector{0, 0, 0, 1})
	spherePos := invertedTransform.MultVec(sphere.WorldPosition())
	sphereRadius := sphere.WorldRadius() * math.Abs(math.Max(invertedTransform[0][0], math.Max(invertedTransform[1][1], invertedTransform[2][2])))

	var mtv vector.Vector
	var contact vector.Vector
	closestDist := math.MaxFloat64

	for _, tri := range triangles.Mesh.Triangles {

		v0 := tri.Vertices[0].Position
		v1 := tri.Vertices[1].Position
		v2 := tri.Vertices[2].Position

		closest := closestPointOnTri(spherePos, v0, v1, v2)
		delta := fastVectorSub(spherePos, closest)

		if delta.Magnitude() <= sphereRadius {

			d := fastVectorDistanceSquared(spherePos, closest)

			if contact == nil || d < closestDist {
				contact = closest
				closestDist = d
				mtv = transformNoLoc.MultVec(delta.Unit().Scale(sphereRadius - delta.Magnitude()))
			}

			// if contact == nil || d < closestDist {
			// 	contact = closest
			// 	closestDist = d
			// 	mtv = delta.Unit().Scale(sphereRadius - delta.Magnitude())
			// }

		}

	}

	if mtv == nil {
		return nil
	}

	return NewIntersectionResult(contact, mtv)

}

func btAABBAABB(aabbA, aabbB *BoundingAABB) *IntersectionResult {

	aPos := aabbA.WorldPosition()
	bPos := aabbB.WorldPosition()
	aSize := aabbA.Size.Scale(0.5)
	bSize := aabbB.Size.Scale(0.5)

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

	result := NewIntersectionResult(nil, nil)

	if px < py && px < pz {
		sx := -1.0
		if math.Signbit(dx) {
			sx = 1
		}

		result.MTV = vector.Vector{px * sx, 0, 0}
		// result.Normal = vector.Vector{sx, 0, 0}
		result.ContactPoint = vector.Vector{aPos[0] + (aSize[0] * sx), bPos[1], bPos[2]}

	} else if py < pz && py < px {
		sy := -1.0
		if math.Signbit(dy) {
			sy = 1
		}

		result.MTV = vector.Vector{0, py * sy, 0}
		// result.Normal = vector.Vector{0, sy, 0}
		result.ContactPoint = vector.Vector{bPos[0], aPos[1] + (aSize[1] * sy), bPos[2]}

	} else {

		sz := -1.0
		if math.Signbit(dz) {
			sz = 1
		}

		result.MTV = vector.Vector{0, 0, pz * sz}
		// result.Normal = vector.Vector{0, 0, sz}
		result.ContactPoint = vector.Vector{bPos[0], bPos[1], aPos[2] + (aSize[2] * sz)}

	}

	// xMin := math.Max(aPos[0]-aSize[0], bPos[0]-bSize[0])
	// xMax := math.Min(aPos[0]+aSize[0], bPos[0]+bSize[0])

	// if xMin > xMax || xMax < xMin {
	// 	return nil
	// }

	// yMin := math.Max(aPos[1]-aSize[1], bPos[1]-bSize[1])
	// yMax := math.Min(aPos[1]+aSize[1], bPos[1]+bSize[1])

	// if yMin > yMax || yMax < yMin {
	// 	return nil
	// }

	// zMin := math.Max(aPos[2]-aSize[2], bPos[2]-bSize[2])
	// zMax := math.Min(aPos[2]+aSize[2], bPos[2]+bSize[2])

	// if zMin > zMax || zMax < zMin {
	// 	return nil
	// }

	// delta := vector.Vector{0, 0, 0}

	// rectOverlap := vector.Vector{
	// 	xMax - xMin - 1,
	// 	yMax - yMin - 1,
	// 	zMax - zMin - 1,
	// }

	return result

}

func btAABBTriangles(box *BoundingAABB, triangles *BoundingTriangles) *IntersectionResult {
	// See https://gdbooks.gitbooks.io/3dcollisions/content/Chapter4/aabb-triangle.html

	triangles.BoundingAABB.SetLocalPosition(triangles.WorldPosition().Add(triangles.Mesh.Dimensions.Center()))
	triangles.BoundingAABB.SetLocalRotation(triangles.WorldRotation())
	triangles.BoundingAABB.SetLocalScale(triangles.WorldScale())

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !box.Intersecting(triangles.BoundingAABB) {
		return nil
	}

	boxPos := box.WorldPosition()
	boxSize := box.Size.Scale(0.5)

	transform := triangles.Transform()

	overallAxis := vector.Vector{0, 0, 0}
	overlapDistance := 0.0

	for _, tri := range triangles.Mesh.Triangles {

		v0 := transform.MultVec(tri.Vertices[0].Position).Sub(boxPos)
		v1 := transform.MultVec(tri.Vertices[1].Position).Sub(boxPos)
		v2 := transform.MultVec(tri.Vertices[2].Position).Sub(boxPos)
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

			transform.MultVec(tri.Normal),
		}

		var overlapAxis vector.Vector
		smallestOverlap := math.MaxFloat64

		for _, axis := range axes {

			if axis == nil {
				return nil
			}

			axis = axis.Unit()

			p1 := project(axis, v0, v1, v2)

			r := boxSize[0]*math.Abs(vector.X.Dot(axis)) +
				boxSize[1]*math.Abs(vector.Y.Dot(axis)) +
				boxSize[2]*math.Abs(vector.Z.Dot(axis))

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

		if overlapAxis != nil && smallestOverlap > overlapDistance {
			overallAxis[0] = overlapAxis[0]
			overallAxis[1] = overlapAxis[1]
			overallAxis[2] = overlapAxis[2]
			overlapDistance = smallestOverlap
		}

	}

	if overallAxis.Magnitude() == 0 {
		return nil
	}

	mtv := overallAxis.Scale(overlapDistance)
	return NewIntersectionResult(box.WorldPosition().Sub(mtv), mtv)

}

func btTrianglesTriangles(trianglesA, trianglesB *BoundingTriangles) *IntersectionResult {
	// See https://gdbooks.gitbooks.io/3dcollisions/content/Chapter4/aabb-triangle.html

	trianglesA.BoundingAABB.SetLocalPosition(trianglesA.WorldPosition().Add(trianglesA.Mesh.Dimensions.Center()))
	trianglesA.BoundingAABB.SetLocalRotation(trianglesA.WorldRotation())
	trianglesA.BoundingAABB.SetLocalScale(trianglesA.WorldScale())

	trianglesB.BoundingAABB.SetLocalPosition(trianglesB.WorldPosition().Add(trianglesB.Mesh.Dimensions.Center()))
	trianglesB.BoundingAABB.SetLocalRotation(trianglesB.WorldRotation())
	trianglesB.BoundingAABB.SetLocalScale(trianglesB.WorldScale())

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !trianglesA.BoundingAABB.Intersecting(trianglesB.BoundingAABB) {
		return nil
	}

	transformA := trianglesA.Transform()
	transformB := trianglesB.Transform()

	overallOverlap := vector.Vector{0, 0, 0}
	overlapCount := 0.0

	transformedA := [][]vector.Vector{}
	transformedB := [][]vector.Vector{}

	for _, tri := range trianglesA.Mesh.Triangles {

		v0 := transformA.MultVec(tri.Vertices[0].Position)
		v1 := transformA.MultVec(tri.Vertices[1].Position)
		v2 := transformA.MultVec(tri.Vertices[2].Position)

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

	for _, tri := range trianglesB.Mesh.Triangles {

		v0 := transformB.MultVec(tri.Vertices[0].Position)
		v1 := transformB.MultVec(tri.Vertices[1].Position)
		v2 := transformB.MultVec(tri.Vertices[2].Position)

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

	for _, a := range transformedA {

		for _, b := range transformedB {

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
				overallOverlap[0] += overlapAxis[0] * smallestOverlap
				overallOverlap[1] += overlapAxis[1] * smallestOverlap
				overallOverlap[2] += overlapAxis[2] * smallestOverlap
				overlapCount++
			}

		}

	}

	if overlapCount == 0 {
		return nil
	}

	overallOverlap[0] /= overlapCount
	overallOverlap[1] /= overlapCount
	overallOverlap[2] /= overlapCount

	return NewIntersectionResult(trianglesA.WorldPosition().Sub(overallOverlap), overallOverlap)

}

func btCapsuleCapsule(capsuleA, capsuleB *BoundingCapsule) *IntersectionResult {
	capsuleA.internalSphere.SetLocalScale(capsuleA.LocalScale())

	// By getting the closest point to the world position (center), and then getting it again, we get closer to the
	// true closest point for both capsules, which is good enough for now lol
	caClosest := capsuleA.ClosestPoint(capsuleB.WorldPosition())
	cbClosest := capsuleB.ClosestPoint(capsuleA.WorldPosition())

	capsuleA.internalSphere.SetLocalPosition(capsuleA.ClosestPoint(cbClosest))
	capsuleA.internalSphere.Radius = capsuleA.Radius

	capsuleB.internalSphere.SetLocalScale(capsuleB.LocalScale())
	capsuleB.internalSphere.SetLocalPosition(capsuleB.ClosestPoint(caClosest))
	capsuleB.internalSphere.Radius = capsuleB.Radius

	return btSphereSphere(capsuleA.internalSphere, capsuleB.internalSphere)
}

func btSphereCapsule(sphere *BoundingSphere, capsule *BoundingCapsule) *IntersectionResult {
	capsule.internalSphere.SetLocalScale(capsule.LocalScale())
	capsule.internalSphere.SetLocalPosition(capsule.ClosestPoint(sphere.WorldPosition()))
	capsule.internalSphere.Radius = capsule.Radius
	return btSphereSphere(sphere, capsule.internalSphere)
}

func btCapsuleAABB(capsule *BoundingCapsule, aabb *BoundingAABB) *IntersectionResult {
	capsule.internalSphere.SetLocalScale(capsule.LocalScale())
	capsule.internalSphere.SetLocalPosition(capsule.ClosestPoint(aabb.WorldPosition()))
	capsule.internalSphere.Radius = capsule.Radius
	return btSphereAABB(capsule.internalSphere, aabb)
}

func btCapsuleTriangles(capsule *BoundingCapsule, triangles *BoundingTriangles) *IntersectionResult {

	capsule.internalSphere.SetLocalScale(capsule.LocalScale())
	capsule.internalSphere.SetLocalPosition(capsule.ClosestPoint(triangles.WorldPosition()))
	capsule.internalSphere.Radius = capsule.Radius

	triangles.BoundingAABB.SetLocalPosition(triangles.WorldPosition().Add(triangles.Mesh.Dimensions.Center()))
	triangles.BoundingAABB.SetLocalRotation(triangles.WorldRotation())
	triangles.BoundingAABB.SetLocalScale(triangles.WorldScale())

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !capsule.internalSphere.Intersecting(triangles.BoundingAABB) {
		return nil
	}

	invertedTransform := triangles.Node.Transform().Inverted()
	transformNoLoc := triangles.Node.Transform().SetRow(3, vector.Vector{0, 0, 0, 1})

	capsuleRadius := capsule.WorldRadius() * math.Abs(math.Max(invertedTransform[0][0], math.Max(invertedTransform[1][1], invertedTransform[2][2])))

	var mtv vector.Vector
	var contact vector.Vector
	closestDist := math.MaxFloat64

	capsuleTop := invertedTransform.MultVec(capsule.Top())
	capsuleBottom := invertedTransform.MultVec(capsule.Bottom())
	capsuleLine := capsuleTop.Sub(capsuleBottom)

	var closestCapsulePoint vector.Vector

	for _, tri := range triangles.Mesh.Triangles {

		v0 := tri.Vertices[0].Position
		v1 := tri.Vertices[1].Position
		v2 := tri.Vertices[2].Position

		if fastVectorDistanceSquared(tri.Center, capsuleTop) < fastVectorDistanceSquared(tri.Center, capsuleBottom) {
			closestCapsulePoint = capsuleTop
		} else {
			closestCapsulePoint = capsuleBottom
		}

		closest := closestPointOnTri(closestCapsulePoint, v0, v1, v2)

		// Doing this manually to avoid doing as much as possible~

		t := closest.Sub(capsuleBottom).Dot(capsuleLine) / capsuleLine.Dot(capsuleLine)
		t = math.Max(math.Min(t, 1), 0)
		spherePos := capsuleBottom.Add(capsuleLine.Scale(t))

		// spherePos := invertedTransform.MultVec(capsule.ClosestPoint(closest))

		// spherePos := capsule.ClosestPoint(closest)

		delta := fastVectorSub(spherePos, closest)

		if delta.Magnitude() <= capsuleRadius {

			d := fastVectorDistanceSquared(spherePos, closest)
			if contact == nil || d < closestDist {
				contact = closest
				closestDist = d
				mtv = transformNoLoc.MultVec(delta.Unit().Scale(capsuleRadius - delta.Magnitude()))
			}

		}

	}

	if mtv == nil {
		return nil
	}

	return NewIntersectionResult(contact, mtv)

}

type projection struct {
	Min, Max float64
}

func project(axis vector.Vector, points ...vector.Vector) projection {

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
