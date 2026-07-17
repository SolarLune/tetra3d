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
	Normal        Vector3   // The colliding normal of the intersection
}

func (intersection *Intersection) Distance() float32 {
	return intersection.StartingPoint.DistanceTo(intersection.ContactPoint)
}

func (intersection *Intersection) DistanceSquared() float32 {
	return intersection.StartingPoint.DistanceSquaredTo(intersection.ContactPoint)
}

// Collision represents the result of a collision test. A Collision test may result in multiple intersections, and
// so an Collision holds each of these individual intersections in its Intersections slice.
// The intersections are sorted in order of distance from the starting point of the intersection (the center of the
// colliding sphere / aabb, the closest point in the capsule, the center of the closest triangle, etc) to the
// contact point.
type Collision struct {
	Object Collider // The Collider collided with
	// The slice of Intersections, one for each object or triangle intersected with.
	// The slice is sorted in order by overall magnitude of minimum translation vector (e.g. how far
	// into the intersection the object is), and if those values are similar, then by perpendicularity to
	// the colliding object
	Intersections []*Intersection
}

func newCollision(collidedObject Collider) *Collision {
	return &Collision{
		Object:        collidedObject,
		Intersections: []*Intersection{},
	}
}

func (col *Collision) add(intersection *Intersection) *Collision {
	col.Intersections = append(col.Intersections, intersection)
	return col
}

// sort the intersections by distance from starting point (which should be the same for all collisions except for triangle-triangle) to contact point.
func (col *Collision) sortResults() {
	sort.Slice(col.Intersections, func(i, j int) bool {

		// Sort by MTV values if there's a significant difference
		mi := col.Intersections[i].MTV.MagnitudeSquared()
		mj := col.Intersections[j].MTV.MagnitudeSquared()

		if math32.Abs(mi-mj) < 0.000001 {
			// Otherwise sort by perpendicularity (e.g. which intersection is more directly "facing" the colliding object)
			mvi := col.Intersections[i].ContactPoint.Sub(col.Intersections[i].StartingPoint).Unit()
			mvj := col.Intersections[j].ContactPoint.Sub(col.Intersections[j].StartingPoint).Unit()
			return col.Intersections[i].Normal.Dot(mvi) < col.Intersections[j].Normal.Dot(mvj)
		}

		return mi > mj

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

// Returns the maximum MTV to be moved on all axes to escape intersection with all intersection points.
func (col *Collision) MaxMTV() Vector3 {
	mtv := Vector3{}
	for _, inter := range col.Intersections {
		if inter.MTV.IsInf() || inter.MTV.IsNaN() {
			continue
		}
		if math32.Abs(inter.MTV.X) > math32.Abs(mtv.X) {
			mtv.X = inter.MTV.X
		}
		if math32.Abs(inter.MTV.Y) > math32.Abs(mtv.Y) {
			mtv.Y = inter.MTV.Y
		}
		if math32.Abs(inter.MTV.Z) > math32.Abs(mtv.Z) {
			mtv.Z = inter.MTV.Z
		}
	}
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

// AverageSlope returns the average slope of the Collision (ranging from -1 to 1, with 0 pointing straight up).
// This average is spread across all intersections contained within the Collision.
func (result *Collision) AverageSlope() float32 {
	slope := result.Intersections[0].Normal.Slope()
	slopeCount := float32(1.0)
	for i := 1; i < len(result.Intersections); i++ {
		inter := result.Intersections[i]
		if inter.MTV.Magnitude() > 0 {
			slope += inter.Normal.Slope()
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

var collidedMaterialSet = newOrderedSet[*Material]()

func (result *Collision) ForEachCollidedMaterial(forEach func(mat *Material) bool) {
	collidedMaterialSet.Clear()
	for _, inter := range result.Intersections {
		if inter.Triangle != nil && inter.Triangle.MeshPart.Material != nil {
			collidedMaterialSet.Add(inter.Triangle.MeshPart.Material)
		}
	}
	collidedMaterialSet.ForEach(func(element *Material) bool {
		return forEach(element)
	})
}

// Returns if the object in the collision has a parent with a material with the given name.
func (result *Collision) CollidedWithMaterialByName(matName string) bool {
	for _, inter := range result.Intersections {
		if inter.Triangle != nil && inter.Triangle.MeshPart.Material != nil && inter.Triangle.MeshPart.Material.name == matName {
			return true
		}
	}
	if parent := result.Object.Parent(); parent != nil {
		if model, ok := parent.Parent().(*Model); ok {
			for _, meshPart := range model.Mesh().MeshParts {
				if meshPart.Material != nil && meshPart.Material.name == matName {
					return true
				}
			}
		}
	}
	return false
}

// Returns if the object in the collision has a parent with a material with a property with the given name.
func (result *Collision) CollidedWithMaterialPropName(matHasPropNamed string) bool {
	for _, inter := range result.Intersections {
		if inter.Triangle != nil && inter.Triangle.MeshPart.Material != nil {
			if inter.Triangle.MeshPart.Material.properties.Has(matHasPropNamed) {
				return true
			}
		}
	}
	if parent := result.Object.Parent(); parent != nil {
		if model, ok := parent.Parent().(*Model); ok {
			for _, meshPart := range model.Mesh().MeshParts {
				if meshPart.Material != nil {
					if meshPart.Material.properties.Has(matHasPropNamed) {
						return true
					}
				}
			}
		}
	}
	return false
}

// Returns if the object in the collision has a parent with a property of the given name.
func (result *Collision) CollidedWithParentPropName(parentHasPropNamed string) bool {
	if result.Object.Parent() != nil {
		return result.Object.Parent().Properties().Has(parentHasPropNamed)
	}
	return false
}

// Returns if the object in the collision has a parent with a property of the given name with the given value.
func (result *Collision) CollidedWithParentPropPair(parentHasPropNamed string, value any) bool {
	if result.Object.Parent() != nil {
		prop := result.Object.Parent().Properties().GetByName(parentHasPropNamed)
		if prop != nil {
			return prop.Value == value
		}
		return false
	}
	return false
}

// Returns if the object in the collision has a parent with a bitfield value property with a bit of the given name flipped.
func (result *Collision) CollidedWithParentBitfieldValueName(bitValueName string) bool {
	if result.Object.Parent() != nil {
		return result.Object.Parent().PropertiesContainsBitByName(bitValueName)
	}
	return false
}

// CollisionTestSettings controls how a CollisionTest() call evaluates.
type CollisionTestSettings struct {

	// TODO: Add backface culling for collision tests

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

// Collider represents a Node type that can be tested for collision. The exposed functions are essentially just
// concerning whether an object that implements Collider is colliding with another Collider, and
// if so, by how much.
type Collider interface {
	INode
	// Colliding returns true if the Collider is intersecting the other Collider.
	Colliding(other Collider) bool
	// Collision returns a Collision if the Collider is intersecting another Collider. If
	// no intersection is reported, Collision returns nil.
	Collision(other Collider) *Collision

	// CollisionTest performs a distance-ordered collision test using the provided collision test settings structure
	// and returns the first collision found (if none are found, then it returns nil). To perform a function against
	// multiple objects, use the OnCollision function in the CollisionTestSettings object.
	CollisionTest(settings CollisionTestSettings) *Collision

	// Returns the transformed dimensions of the bounding object.
	Dimensions() Dimensions
}

// The below set of bt functions are used to test for intersection between Collider pairs.
// I forget now, but I guess when I wrote this, bt* stood for Bounding Test, haha.
func btSphereSphere(sphereA, sphereB *ColliderSphere) *Collision {

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

func btSphereAABB(sphere *ColliderSphere, aabb *ColliderAABB) *Collision {

	spherePos := sphere.WorldPosition()
	sphereRadius := sphere.WorldRadius()

	intersection := aabb.ClosestPoint(spherePos)

	distance := spherePos.DistanceTo(intersection)

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

func btSphereTriangles(sphere *ColliderSphere, triangles *ColliderTriangles) *Collision {

	aabbDim := triangles.colliderAABB.Dimensions()

	sphereWorldPosition := sphere.WorldPosition()
	sphereWorldRadius := sphere.WorldRadius()

	// If we're not intersecting the triangle's AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if sphereWorldPosition.X-sphereWorldRadius > aabbDim.Max.X ||
		sphereWorldPosition.X+sphereWorldRadius < aabbDim.Min.X ||
		sphereWorldPosition.Y-sphereWorldRadius > aabbDim.Max.Y ||
		sphereWorldPosition.Y+sphereWorldRadius < aabbDim.Min.Y ||
		sphereWorldPosition.Z-sphereWorldRadius > aabbDim.Max.Z ||
		sphereWorldPosition.Z+sphereWorldRadius < aabbDim.Min.Z {
		return nil
	}

	triTrans := triangles.Transform()
	invertedTransform := triTrans.Inverted()
	transformNoLoc := triTrans.Clone()
	transformNoLoc.SetRow(3, Vector4{0, 0, 0, 1})
	spherePos := invertedTransform.MultVec(sphereWorldPosition)
	sphereRadius := sphereWorldRadius * math32.Abs(math32.Max(invertedTransform[0][0], math32.Max(invertedTransform[1][1], invertedTransform[2][2])))
	sphereRadiusSquared := sphereRadius * sphereRadius

	result := newCollision(triangles)

	for _, tri := range triangles.Mesh.Triangles {
		// triangles.broadphase.ForEachTriangleFromCollider(sphere, func(tri *Triangle) bool {

		if tri.MeshPart.Material != nil && !tri.MeshPart.Material.ReportCollisions {
			continue
		}

		// We no longer need to avoid checking triangles that are too far away because our Broadphase does that for us

		v0 := triangles.Mesh.VertexPositions[tri.VertexIndexA]
		v1 := triangles.Mesh.VertexPositions[tri.VertexIndexB]
		v2 := triangles.Mesh.VertexPositions[tri.VertexIndexC]

		if (spherePos.X-sphereRadius > v0.X && spherePos.X-sphereRadius > v1.X && spherePos.X-sphereRadius > v2.X) ||
			(spherePos.X+sphereRadius < v0.X && spherePos.X+sphereRadius < v1.X && spherePos.X+sphereRadius < v2.X) ||
			(spherePos.Y-sphereRadius > v0.Y && spherePos.Y-sphereRadius > v1.Y && spherePos.Y-sphereRadius > v2.Y) ||
			(spherePos.Y+sphereRadius < v0.Y && spherePos.Y+sphereRadius < v1.Y && spherePos.Y+sphereRadius < v2.Y) ||
			(spherePos.Z-sphereRadius > v0.Z && spherePos.Z-sphereRadius > v1.Z && spherePos.Z-sphereRadius > v2.Z) ||
			(spherePos.Z+sphereRadius < v0.Z && spherePos.Z+sphereRadius < v1.Z && spherePos.Z+sphereRadius < v2.Z) {
			continue
		}

		closest := closestPointOnTri(spherePos, v0, v1, v2, tri.Normal)
		delta := spherePos.Sub(closest)

		if mag := delta.MagnitudeSquared(); mag <= sphereRadiusSquared {
			result.add(
				&Intersection{
					StartingPoint: sphereWorldPosition,
					ContactPoint:  triTrans.MultVec(closest),
					MTV:           transformNoLoc.MultVec(delta.Unit().Scale(sphereRadius - delta.Magnitude())),
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

func btAABBAABB(aabbA, aabbB *ColliderAABB) *Collision {

	aPos := aabbA.WorldPosition()
	bPos := aabbB.WorldPosition()
	aSize := aabbA.dimensions.Size().Scale(0.5)
	bSize := aabbB.dimensions.Size().Scale(0.5)

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

func btAABBTriangles(box *ColliderAABB, triangles *ColliderTriangles) *Collision {
	// See https://gdbooks.gitbooks.io/3dcollisions/content/Chapter4/aabb-triangle.html

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !box.Colliding(triangles.colliderAABB) {
		return nil
	}

	boxPos := box.WorldPosition()
	boxSize := box.dimensions.Size().Scale(0.5)

	transform := triangles.Transform()
	transformNoLoc := transform.Clone()
	transformNoLoc.SetRow(3, Vector4{0, 0, 0, 1})

	result := newCollision(triangles)

	// aabbWorldRight := project(WorldRight, NewVector3(boxSize.X, 0, 0))
	// aabbWorldUp := project(WorldUp, NewVector3(0, boxSize.Y, 0))
	// aabbWorldForward := project(WorldDown, NewVector3(0, 0, boxSize.Z))

	// Pretty sure I can avoid transforming the vertices (and normal) by transforming the AABB axes by an inverse

	for _, tri := range triangles.Mesh.Triangles {
		// triangles.broadphase.ForEachTriangleFromCollider(box, func(tri *Triangle) bool {

		if tri.MeshPart.Material != nil && !tri.MeshPart.Material.ReportCollisions {
			continue
		}

		v0 := transform.MultVec(triangles.Mesh.VertexPositions[tri.VertexIndexA])
		v1 := transform.MultVec(triangles.Mesh.VertexPositions[tri.VertexIndexB])
		v2 := transform.MultVec(triangles.Mesh.VertexPositions[tri.VertexIndexC])

		axes := []Vector3{

			WorldRight,
			WorldLeft,
			WorldUp,
			WorldDown,
			WorldForward,
			WorldBackward,

			transformNoLoc.MultVec(tri.Normal),
		}

		var overlapAxis Vector3
		smallestOverlap := math32.MaxFloat32

		for _, axis := range axes {

			projectA := NewProjection(axis,
				// boxPos,
				boxPos.Add(WorldRight.Scale(boxSize.X)).Add(WorldUp.Scale(boxSize.Y)).Add(WorldForward.Scale(boxSize.Z)),
				boxPos.Add(WorldRight.Scale(-boxSize.X)).Add(WorldUp.Scale(boxSize.Y)).Add(WorldForward.Scale(boxSize.Z)),
				boxPos.Add(WorldRight.Scale(boxSize.X)).Add(WorldUp.Scale(boxSize.Y)).Add(WorldForward.Scale(-boxSize.Z)),
				boxPos.Add(WorldRight.Scale(-boxSize.X)).Add(WorldUp.Scale(boxSize.Y)).Add(WorldForward.Scale(-boxSize.Z)),

				boxPos.Add(WorldRight.Scale(boxSize.X)).Add(WorldUp.Scale(-boxSize.Y)).Add(WorldForward.Scale(boxSize.Z)),
				boxPos.Add(WorldRight.Scale(-boxSize.X)).Add(WorldUp.Scale(-boxSize.Y)).Add(WorldForward.Scale(boxSize.Z)),
				boxPos.Add(WorldRight.Scale(boxSize.X)).Add(WorldUp.Scale(-boxSize.Y)).Add(WorldForward.Scale(-boxSize.Z)),
				boxPos.Add(WorldRight.Scale(-boxSize.X)).Add(WorldUp.Scale(-boxSize.Y)).Add(WorldForward.Scale(-boxSize.Z)),
			)
			projectB := NewProjection(axis, v0, v1, v2)

			overlap := projectA.Overlap(projectB)

			if projectA.IsOverlapping(projectB) {

				if math32.Abs(overlap) < smallestOverlap {
					overlapAxis = axis.Invert()
					smallestOverlap = overlap
				}

			} else {
				continue // Move on to next tri
			}

		}

		// for _, axis := range axes {

		// 	if axis.IsZero() {
		// 		return false
		// 	}

		// 	axis = axis.Unit()

		// 	p1 := project(axis, v0, v1, v2)

		// 	r := boxSize.X*math32.Abs(WorldRight.Dot(axis)) +
		// 		boxSize.Y*math32.Abs(WorldUp.Dot(axis)) +
		// 		boxSize.Z*math32.Abs(WorldBackward.Dot(axis))

		// 	p2 := projection{
		// 		Max: r,
		// 		Min: -r,
		// 	}

		// 	overlap := p1.Overlap(p2)

		// 	if !p1.IsOverlapping(p2) {
		// 		overlapAxis = Vector3{}
		// 		break
		// 	}

		// 	if overlap < smallestOverlap {
		// 		smallestOverlap = overlap
		// 		overlapAxis = axis
		// 	}

		// }

		if !overlapAxis.IsZero() {

			// smallestOverlap += math32.Sign(smallestOverlap) * 0.1

			mtv := overlapAxis.Scale(smallestOverlap)

			triCenter := v0.Add(v1).Add(v2).Scale(1.0 / 3.0)
			boxPoint := box.ClosestPoint(triCenter)

			result.add(&Intersection{
				StartingPoint: boxPos,
				ContactPoint:  closestPointOnTri(boxPoint, v0, v1, v2, axes[6]),
				MTV:           mtv,
				Triangle:      tri,
				Normal:        axes[6],
			})

		}

	}

	if len(result.Intersections) == 0 {
		return nil
	}

	result.sortResults()

	return result

}

func btTrianglesTriangles(trianglesA, trianglesB *ColliderTriangles) *Collision {
	// See https://gdbooks.gitbooks.io/3dcollisions/content/Chapter4/aabb-triangle.html

	// If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	if !trianglesA.colliderAABB.Colliding(trianglesB.colliderAABB) {
		return nil
	}

	transformA := trianglesA.Transform()
	transformB := trianglesB.Transform()

	transformedA := [][]Vector3{}
	transformedB := [][]Vector3{}

	result := newCollision(trianglesB)

	for _, meshPart := range trianglesA.Mesh.MeshParts {

		if meshPart.Material != nil && !meshPart.Material.ReportCollisions {
			continue
		}

		mesh := meshPart.Mesh

		meshPart.ForEachTri(func(tri *Triangle) {

			v0 := transformA.MultVec(mesh.VertexPositions[tri.VertexIndexA])
			v1 := transformA.MultVec(mesh.VertexPositions[tri.VertexIndexB])
			v2 := transformA.MultVec(mesh.VertexPositions[tri.VertexIndexC])

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

		if meshPart.Material != nil && !meshPart.Material.ReportCollisions {
			continue
		}

		mesh := meshPart.Mesh

		meshPart.ForEachTri(func(tri *Triangle) {

			v0 := transformB.MultVec(mesh.VertexPositions[tri.VertexIndexA])
			v1 := transformB.MultVec(mesh.VertexPositions[tri.VertexIndexB])
			v2 := transformB.MultVec(mesh.VertexPositions[tri.VertexIndexC])

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

				p1 := NewProjection(axis, a[0], a[1], a[2])
				p2 := NewProjection(axis, b[0], b[1], b[2])

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

func btCapsuleCapsule(capsuleA, capsuleB *ColliderCapsule) *Collision {
	capsuleA.internalSphere.SetLocalScaleVec(capsuleA.LocalScale())

	// By getting the closest point to the world position (center), and then getting it again, we get closer to the
	// true closest point for both capsules, which is good enough for now lol
	caClosest := capsuleA.NearestPointOnLine(capsuleB.WorldPosition())
	cbClosest := capsuleB.NearestPointOnLine(capsuleA.WorldPosition())

	capsuleA.internalSphere.SetLocalPositionVec(capsuleA.NearestPointOnLine(cbClosest))
	capsuleA.internalSphere.radius = capsuleA.radius

	capsuleB.internalSphere.SetLocalScaleVec(capsuleB.LocalScale())
	capsuleB.internalSphere.SetLocalPositionVec(capsuleB.NearestPointOnLine(caClosest))
	capsuleB.internalSphere.radius = capsuleB.radius

	col := btSphereSphere(capsuleA.internalSphere, capsuleB.internalSphere)
	if col != nil {
		col.Object = capsuleB
	}
	return col
}

func btSphereCapsule(sphere *ColliderSphere, capsule *ColliderCapsule) *Collision {
	capsule.internalSphere.SetLocalScaleVec(capsule.LocalScale())
	capsule.internalSphere.SetLocalPositionVec(capsule.NearestPointOnLine(sphere.WorldPosition()))
	capsule.internalSphere.radius = capsule.radius
	col := btSphereSphere(sphere, capsule.internalSphere)
	if col != nil {
		col.Object = capsule
	}
	return col
}

func btCapsuleAABB(capsule *ColliderCapsule, aabb *ColliderAABB) *Collision {
	capsule.internalSphere.SetLocalScaleVec(capsule.LocalScale())
	capsule.internalSphere.SetLocalPositionVec(capsule.NearestPointOnLine(aabb.WorldPosition()))
	capsule.internalSphere.radius = capsule.radius
	return btSphereAABB(capsule.internalSphere, aabb)
}

func btCapsuleTriangles(capsule *ColliderCapsule, triangles *ColliderTriangles) *Collision {

	capTop := capsule.Top()
	capBottom := capsule.Bottom()
	rad := capsule.WorldRadius()
	aabbDim := triangles.colliderAABB.Dimensions()

	if (capTop.X-rad > aabbDim.Max.X && capBottom.X-rad > aabbDim.Max.X) ||
		(capTop.X+rad < aabbDim.Min.X && capBottom.X+rad < aabbDim.Min.X) ||
		(capTop.Y-rad > aabbDim.Max.Y && capBottom.Y-rad > aabbDim.Max.Y) ||
		(capTop.Y+rad < aabbDim.Min.Y && capBottom.Y+rad < aabbDim.Min.Y) ||
		(capTop.Z-rad > aabbDim.Max.Z && capBottom.Z-rad > aabbDim.Max.Z) ||
		(capTop.Z+rad < aabbDim.Min.Z && capBottom.Z+rad < aabbDim.Min.Z) {
		return nil
	}

	capsule.internalSphere.SetLocalScaleVec(capsule.LocalScale())
	capsule.internalSphere.SetLocalPositionVec(capsule.NearestPointOnLine(triangles.colliderAABB.WorldPosition()))
	capsule.internalSphere.radius = capsule.radius

	// // If we're not intersecting the triangle's bounding AABB, we couldn't possibly be colliding with any of the triangles, so we're good
	// if !capsule.internalSphere.Colliding(triangles.colliderAABB) {
	// 	return nil
	// }

	triTrans := triangles.Transform()
	invertedTransform := triTrans.Inverted()
	transformNoLoc := triTrans.Clone()
	transformNoLoc.SetRow(3, Vector4{0, 0, 0, 1})

	capsuleRadius := capsule.WorldRadius() * math32.Abs(math32.Max(invertedTransform[0][0], math32.Max(invertedTransform[1][1], invertedTransform[2][2])))
	capsuleRadiusSquared := capsuleRadius * capsuleRadius

	capsuleTop := invertedTransform.MultVec(capsule.lineTop())
	capsuleBottom := invertedTransform.MultVec(capsule.lineBottom())
	capsuleLine := capsuleTop.Sub(capsuleBottom)
	capDot := capsuleLine.Dot(capsuleLine)

	var closestCapsulePoint Vector3

	result := newCollision(triangles)

	spherePos := Vector3{}

	closestSub := Vector3{}

	// TODO: Review Broadphase to see why it's so slow / if it's still necessary
	// triangles.broadphase.ForEachTriangleFromCollider(capsule, func(tri *Triangle) bool {

	for _, tri := range triangles.Mesh.Triangles {

		if tri.MeshPart.Material != nil && !tri.MeshPart.Material.ReportCollisions {
			continue
		}

		// if capsulePosition.DistanceSquaredTo(tri.Center) > math32.Pow((tri.MaxSpan*0.66)+capSpread, 2) {
		// 	return true
		// }

		if tri.Center.DistanceSquaredTo(capsuleTop) < tri.Center.DistanceSquaredTo(capsuleBottom) {
			closestCapsulePoint = capsuleTop
		} else {
			closestCapsulePoint = capsuleBottom
		}

		v0 := triangles.Mesh.VertexPositions[tri.VertexIndexA]
		v1 := triangles.Mesh.VertexPositions[tri.VertexIndexB]
		v2 := triangles.Mesh.VertexPositions[tri.VertexIndexC]

		if (capsuleTop.X-capsuleRadius > v0.X && capsuleTop.X-capsuleRadius > v1.X && capsuleTop.X-capsuleRadius > v2.X &&
			capsuleBottom.X-capsuleRadius > v0.X && capsuleBottom.X-capsuleRadius > v1.X && capsuleBottom.X-capsuleRadius > v2.X) ||
			(capsuleTop.X+capsuleRadius < v0.X && capsuleTop.X+capsuleRadius < v1.X && capsuleTop.X+capsuleRadius < v2.X &&
				capsuleBottom.X+capsuleRadius < v0.X && capsuleBottom.X+capsuleRadius < v1.X && capsuleBottom.X+capsuleRadius < v2.X) ||
			(capsuleTop.Y-capsuleRadius > v0.Y && capsuleTop.Y-capsuleRadius > v1.Y && capsuleTop.Y-capsuleRadius > v2.Y &&
				capsuleBottom.Y-capsuleRadius > v0.Y && capsuleBottom.Y-capsuleRadius > v1.Y && capsuleBottom.Y-capsuleRadius > v2.Y) ||
			(capsuleTop.Y+capsuleRadius < v0.Y && capsuleTop.Y+capsuleRadius < v1.Y && capsuleTop.Y+capsuleRadius < v2.Y &&
				capsuleBottom.Y+capsuleRadius < v0.Y && capsuleBottom.Y+capsuleRadius < v1.Y && capsuleBottom.Y+capsuleRadius < v2.Y) ||
			(capsuleTop.Z-capsuleRadius > v0.Z && capsuleTop.Z-capsuleRadius > v1.Z && capsuleTop.Z-capsuleRadius > v2.Z &&
				capsuleBottom.Z+capsuleRadius > v0.Z && capsuleBottom.Z+capsuleRadius > v1.Z && capsuleBottom.Z+capsuleRadius > v2.Z) ||
			(capsuleTop.Z-capsuleRadius < v0.Z && capsuleTop.Z-capsuleRadius < v1.Z && capsuleTop.Z-capsuleRadius < v2.Z &&
				capsuleBottom.Z+capsuleRadius < v0.Z && capsuleBottom.Z+capsuleRadius < v1.Z && capsuleBottom.Z+capsuleRadius < v2.Z) {
			continue
		}

		closest := closestPointOnTri(closestCapsulePoint, v0, v1, v2, tri.Normal)
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

		if mag := delta.MagnitudeSquared(); mag <= capsuleRadiusSquared {

			result.add(
				&Intersection{
					StartingPoint: closestCapsulePoint,
					ContactPoint:  triTrans.MultVec(closest),
					MTV:           transformNoLoc.MultVec(delta.Unit().Scale(capsuleRadius - delta.Magnitude())),
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

var internalCollisionList = []*Collision{}

func commonCollisionTest(node INode, settings CollisionTestSettings) *Collision {

	if settings.TestAgainst == nil {
		return nil
	}

	internalCollisionList = internalCollisionList[:0]

	settings.TestAgainst.ForEach(func(n INode, index int) bool {

		collider, _ := n.(Collider)

		// Not a collider, colliding against itself
		if collider == nil || node == collider {
			return true
		}

		if collision := node.(Collider).Collision(collider); collision != nil {
			internalCollisionList = append(internalCollisionList, collision)
		}

		return true
	})

	var result *Collision

	// Sort the collisions by distance to the intersection start point (so closer collisions come up sooner).
	if len(internalCollisionList) > 0 {

		sort.Slice(internalCollisionList, func(i, j int) bool {

			mi := internalCollisionList[i].Intersections[0].MTV.MagnitudeSquared()
			mj := internalCollisionList[j].Intersections[0].MTV.MagnitudeSquared()

			if math32.Abs(mi-mj) < 0.000001 {

				return internalCollisionList[i].Intersections[0].ContactPoint.DistanceSquaredTo(internalCollisionList[i].Intersections[0].StartingPoint) >
					internalCollisionList[j].Intersections[0].ContactPoint.DistanceSquaredTo(internalCollisionList[j].Intersections[0].StartingPoint)

			}

			return mi > mj

		})

		// sort.Slice(internalCollisionList, func(i, j int) bool {
		// 	return internalCollisionList[i].Intersections[0].ContactPoint.DistanceSquaredTo(internalCollisionList[i].Intersections[0].StartingPoint) >
		// 		internalCollisionList[j].Intersections[0].ContactPoint.DistanceSquaredTo(internalCollisionList[j].Intersections[0].StartingPoint)
		// })
		result = internalCollisionList[0]
	}

	if settings.OnCollision != nil {

		for i, c := range internalCollisionList {
			if !settings.OnCollision(c, i, len(internalCollisionList)) {
				break
			}
		}

	}

	return result

}

// Represents a distance between two points in space along a single axis.
type Projection struct {
	Min, Max float32
}

// Projects points on an axis to create a Projection, a distance between those points.
// Useful for checking if zones overlap across on a single axis in 3D space.
// For example, you can use this to get the distance from two points on a single axis thusly:
//
// `NewProjection(tetra3d.WorldRight, object1.WorldPosition(), object2.WorldPosition()).Distance()`
//
// Or create two projections on the same axis and use Projection.IsOverlapping(other) to see if they overlap.
func NewProjection(axis Vector3, points ...Vector3) Projection {

	projection := Projection{}
	projection.Min = axis.Dot(points[0])
	projection.Max = projection.Min

	for i := 1; i < len(points); i++ {
		p := axis.Dot(points[i])
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

func (projection Projection) Overlap(other Projection) float32 {

	if projection.Min > other.Max {
		return 0
	} else if projection.Max < other.Min {
		return 0
	}

	return math32.Max(other.Max, projection.Max) - math32.Max(other.Min, projection.Min)

	// if projection.Max > other.Min {
	// 	return projection.Max - other.Min
	// }
	// return projection.Min - other.Max
}

func (projection Projection) IsOverlapping(other Projection) bool {
	return !(projection.Min > other.Max || other.Min > projection.Max)
}

func (p Projection) Distance() float32 {
	return p.Max - p.Min
}

var sphereTestObject = NewColliderSphere("sphere check", 1)

// CollisionTestSphere performs a quick collider sphere check at the specified X, Y, and Z position with the radius given,
// against the collider objects provided in "others".
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the sphere at the settings specified.
func CollisionTestSphere(x, y, z, radius float32, settings CollisionTestSettings) *Collision {
	sphereTestObject.SetLocalPosition(x, y, z)
	sphereTestObject.radius = radius
	return commonCollisionTest(sphereTestObject, settings)
}

// CollisionTestSphereVec performs a quick collider sphere check at the specified position with the radius given, against the
// collider objects provided in "others".
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the sphere at the settings specified.
func CollisionTestSphereVec(position Vector3, radius float32, settings CollisionTestSettings) *Collision {
	return CollisionTestSphere(position.X, position.Y, position.Z, radius, settings)
}

var aabbTestObject = NewColliderAABB("aabb check", 1, 1, 1)

// CollisionTestAABB performs a quick collider AABB check at the specified x, y, and z position using the collision settings
// provided. The collider AABB will have the provided width, height, and depth.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the AABB at the settings specified.
// Note that AABB tests with ColliderTriangles are currently buggy.
func CollisionTestAABB(x, y, z, width, height, depth float32, settings CollisionTestSettings) *Collision {
	aabbTestObject.SetLocalPosition(x, y, z)
	aabbTestObject.SetDimensions(width, height, depth)
	return commonCollisionTest(aabbTestObject, settings)
}

// CollisionTestAABBVec places a collider AABB at the position given with the specified size to perform a collision test.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the AABB at the settings specified.
// Note that AABB tests with ColliderTriangles are currently buggy.
func CollisionTestAABBVec(position, size Vector3, settings CollisionTestSettings) *Collision {
	return CollisionTestAABB(position.X, position.Y, position.Z, size.X, size.Y, size.Z, settings)
}

var capsuleTestObject = NewColliderCapsule("capsule check", 1, 2, 1)

// CollisionTestCapsule performs a quick collider capsule check at the specified position
// and size using the collision settings provided.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the capsule at the settings specified.
func CollisionTestCapsule(x, y, z, radius, height float32, up ColliderCapsuleUpAxis, settings CollisionTestSettings) *Collision {
	capsuleTestObject.SetLocalPosition(x, y, z)
	capsuleTestObject.radius = radius
	capsuleTestObject.height = height
	capsuleTestObject.up = up
	return commonCollisionTest(capsuleTestObject, settings)
}

// CollisionTestCapsuleVec places a collider capsule at the position given with the specified
// radius and height to perform a collision test.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the capsule at the settings specified.
func CollisionTestCapsuleVec(position Vector3, radius, height float32, up ColliderCapsuleUpAxis, settings CollisionTestSettings) *Collision {
	return CollisionTestCapsule(position.X, position.Y, position.Z, radius, height, up, settings)
}
