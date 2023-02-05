package tetra3d

import (
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// RayHit represents the result of a raycast test.
type RayHit struct {
	Object   INode  // Object is a pointer to the object that was struck by the raycast.
	Position Vector // Position is the world position that the object was struct.
	Normal   Vector // Normal is the normal of the surface the ray struck.
	// What triangle the raycast hit - note that this is only set to a non-nil value for raycasts against BoundingTriangle objects
	Triangle *Triangle
}

// Slope returns the slope of the RayHit's normal, in radians. This ranges from 0 (straight up) to pi (straight down).
func (r RayHit) Slope() float64 {
	return WorldUp.Angle(r.Normal)
}

func sphereRayTest(center Vector, radius float64, from, to Vector) (RayHit, bool) {

	line := to.Sub(from)
	dir := line.Unit()

	e := center.Sub(from)

	esq := e.MagnitudeSquared()
	a := e.Dot(dir)
	b := math.Sqrt(esq - (a * a))
	f := math.Sqrt((radius * radius) - (b * b))

	vecLength := 0.0

	if radius*radius-esq+a*a < 0 {
		vecLength = -1
	} else if esq < radius*radius {
		vecLength = a + f
	} else {
		vecLength = a - f
	}

	if vecLength*vecLength > line.MagnitudeSquared() {
		return RayHit{}, false
	}

	if vecLength >= 0 {
		strikePos := from.Add(dir.Scale(vecLength))
		return RayHit{
			Position: strikePos,
			Normal:   strikePos.Sub(center).Unit(),
		}, true
	}

	return RayHit{}, false

}

var rayCylinder = NewBoundingTriangles("ray cylinder test", NewCylinderMesh(32, 1, false), 0)

// TODO: Add SphereCast?

// RayTest casts a ray from the "from" world position to the "to" world position, testing against the provided
// IBoundingObjects.
// RayTest returns a slice of RayHit objects sorted from closest to furthest. Note that
// each object can only be struck once by the raycast, with the exception of BoundingTriangles objects (since a
// single ray may strike multiple triangles).
func RayTest(from, to Vector, testAgainst ...IBoundingObject) []RayHit {

	rays := []RayHit{}

	for _, i := range testAgainst {

		i.Transform() // Make sure the transform is updated before the test

		switch test := i.(type) {

		case *BoundingSphere:

			if result, ok := sphereRayTest(test.WorldPosition(), test.WorldRadius(), from, to); ok {
				result.Object = test
				rays = append(rays, result)
			}

		case *BoundingCapsule:

			radius := test.WorldRadius()

			first := test.lineTop()
			second := test.lineBottom()

			// We want to check the pole that's closest to the casting from point first, but we have to check both, technically,
			// since we don't know which pole the ray may pierce.
			if from.Distance(first) > from.Distance(second) {
				tmp := first
				first = second
				second = tmp
			}

			if result, ok := sphereRayTest(first, radius, from, to); ok {
				result.Object = test
				rays = append(rays, result)
			} else {

				// test the cylinder in-between. This is done with a plain cylinder mesh because I'm too stupid to
				// figure out how to do it otherwise lmbo

				rayCylinder.SetWorldTransform(test.Transform())
				rayCylinder.SetLocalScale(radius, test.Height/2, radius)

				if results := RayTest(from, to, rayCylinder); len(results) > 0 {
					for _, r := range results {
						r.Object = test
					}
					rays = append(rays, results...)
				} else if result, ok := sphereRayTest(second, radius, from, to); ok {
					result.Object = test
					rays = append(rays, result)
				}

			}

			// segment := to.Sub(from)

			// pos := test.lineTop()
			// bottom := test.lineBottom()

			// if from.Distance(pos) > from.Distance(bottom) {
			// 	pos = bottom
			// }

			// if from.Distance(pos) > from.Distance(test.WorldPosition()) {
			// 	pos = test.WorldPosition()
			// }

			// t := pos.Sub(from).Dot(segment) / segment.Dot(segment)
			// e := from.Add(segment.Scale(t))

			// fmt.Println(pos, bottom, t)

			// if t < 0 {
			// 	continue
			// }

			// if t > 1 {
			// 	continue
			// }

			// segment := to.Sub(from)

			// t := pos.Dot(segment) / segment.Dot(segment)

			// if t > 1 {
			// 	t = 1
			// } else if t < 0 {
			// 	t = 0
			// }

			// fmt.Println(t)

			// point := from
			// point.X += segment.X * t
			// point.Y += segment.Y * t
			// point.Z += segment.Z * t

			// point = point.Sub(start)
			// t := point.Dot(segment) / segment.Dot(segment)
			// if t > 1 {
			// 	t = 1
			// } else if t < 0 {
			// 	t = 0
			// }

			// start.X += segment.X * t
			// start.Y += segment.Y * t
			// start.Z += segment.Z * t

			// if result := sphereRayTest(test.ClosestPoint(e), test.WorldRadius(), from, to); result != nil {
			// 	result.Object = test
			// 	rays = append(rays, result)
			// }

		case *BoundingAABB:

			line := to.Sub(from)
			dir := line.Unit()

			pos := test.WorldPosition()

			t1 := (test.Dimensions.Min.X + pos.X - from.X) / dir.X
			t2 := (test.Dimensions.Max.X + pos.X - from.X) / dir.X
			t3 := (test.Dimensions.Min.Y + pos.Y - from.Y) / dir.Y
			t4 := (test.Dimensions.Max.Y + pos.Y - from.Y) / dir.Y
			t5 := (test.Dimensions.Min.Z + pos.Z - from.Z) / dir.Z
			t6 := (test.Dimensions.Max.Z + pos.Z - from.Z) / dir.Z

			tmin := max(max(min(t1, t2), min(t3, t4)), min(t5, t6))
			tmax := min(min(max(t1, t2), max(t3, t4)), max(t5, t6))

			if tmin < 0 {
				continue
			}

			if tmin > tmax {
				continue
			}

			vecLength := tmin

			if tmin < 0 {
				vecLength = tmax
			}

			if vecLength*vecLength > line.MagnitudeSquared() {
				continue
			}

			contact := from.Add(dir.Scale(vecLength))

			rays = append(rays, RayHit{
				Object:   test,
				Position: contact,
				Normal:   aabbNormalGuess(contact.Sub(pos)),
			})

		case *BoundingTriangles:

			if test.BoundingAABB.PointInside(from) || test.BoundingAABB.PointInside(to) || len(RayTest(from, to, test.BoundingAABB)) > 0 {

				_, _, r := test.Transform().Decompose()

				invertedTransform := test.Transform().Inverted()
				invFrom := invertedTransform.MultVec(from)
				invTo := invertedTransform.MultVec(to)
				plane := newCollisionPlane()
				maxRayDist := to.DistanceSquared(from)

				for _, tri := range test.Mesh.Triangles {

					// If the distance from the start point to the triangle is longer than the ray,
					// then we know it can't be struck and we can bail early
					if invFrom.DistanceSquared(tri.Center) > maxRayDist+(tri.MaxSpan*tri.MaxSpan) {
						continue
					}

					fs := tri.Normal.Dot(invFrom.Sub(tri.Center))
					ts := tri.Normal.Dot(invTo.Sub(tri.Center))

					// If the start and end points of the ray lie on the same side of the triangle,
					// then we know the triangle can't be struck and we can bail early
					if (fs > 0 && ts > 0) || (fs < 0 && ts < 0) {
						continue
					}

					v0 := test.Mesh.VertexPositions[tri.VertexIndices[0]]
					v1 := test.Mesh.VertexPositions[tri.VertexIndices[1]]
					v2 := test.Mesh.VertexPositions[tri.VertexIndices[2]]

					plane.Set(v0, v1, v2)

					if vec, ok := plane.RayAgainstPlane(invFrom, invTo); ok {

						if plane.pointInsideTriangle(vec, v0, v1, v2) {

							rays = append(rays, RayHit{
								Object:   test,
								Position: test.Transform().MultVec(vec),
								Triangle: tri,
								Normal:   r.MultVec(tri.Normal),
							})

							// break

						}

					}
				}

			}

		}

	}

	sort.Slice(rays, func(i, j int) bool {
		return rays[i].Position.Distance(from) < rays[j].Position.Distance(from)
	})

	return rays

}

// MouseRayTest casts a ray forward from the mouse's position onscreen, testing against the provided
// IBoundingObjects. depth indicates how far the cast ray should extend forwards in world units.
// The function returns a slice of RayHit objects indicating objects struck by the ray, sorted from closest
// to furthest.
// Note that each object can only be struck once by the raycast, with the exception of BoundingTriangles
// objects (since a single ray may strike multiple triangles).
func (camera *Camera) MouseRayTest(depth float64, testAgainst ...IBoundingObject) []RayHit {

	from := camera.WorldPosition()

	mx, my := ebiten.CursorPosition()

	if ebiten.CursorMode() == ebiten.CursorModeCaptured {
		mx = camera.ColorTexture().Bounds().Dx() / 2
		my = camera.ColorTexture().Bounds().Dy() / 2
	}

	to := camera.ScreenToWorld(mx, my, depth)

	return RayTest(from, to, testAgainst...)

}
