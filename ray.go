package tetra3d

import (
	"errors"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// RayHit represents the result of a raycast test.
type RayHit struct {
	Object   INode  // Object is a pointer to the BoundingObject that was struck by the raycast.
	Position Vector // Position is the world position that the object was struct.
	from     Vector // The starting position of the Ray
	Normal   Vector // Normal is the normal of the surface the ray struck.
	// What triangle the raycast hit - note that this is only set to a non-nil value for raycasts against BoundingTriangle objects
	Triangle              *Triangle
	untransformedPosition Vector // untransformed position of the ray test for BoundingTriangles tests
}

// Slope returns the slope of the RayHit's normal, in radians. This ranges from 0 (straight up) to pi (straight down).
func (r RayHit) Slope() float64 {
	return WorldUp.Angle(r.Normal)
}

// Distance returns the distance from the RayHit's originating ray source point to the struck position.
func (r RayHit) Distance() float64 {
	return r.from.Distance(r.Position)
}

const ErrorObjectHitNotBoundingTriangles = "error: object hit not a BoundingTriangles instance; no UV or vertex color data can be pulled from RayHit result"

// VertexColor returns the vertex color from the given channel in the position struck on the object struck,
// assuming it was a BoundingTriangles.
// The returned vertex color is linearly interpolated across the triangle just like it would be when a triangle is rendered.
// VertexColor will return a transparent color and an error if the BoundingObject hit was not a BoundingTriangles object, or if the channel index given
// is higher than the number of vertex color channels on the BoundingTriangles' mesh.
func (r RayHit) VertexColor(channelIndex int) (Color, error) {

	if r.Triangle == nil {
		return NewColor(0, 0, 0, 0), errors.New(ErrorObjectHitNotBoundingTriangles)
	}

	mesh := r.Object.(*BoundingTriangles).Mesh

	if len(mesh.VertexColors[0]) <= channelIndex {
		return NewColor(0, 0, 0, 0), errors.New(ErrorVertexChannelOutsideRange)
	}

	tri := r.Triangle
	u, v := pointInsideTriangle(r.untransformedPosition, mesh.VertexPositions[tri.VertexIndices[0]], mesh.VertexPositions[tri.VertexIndices[1]], mesh.VertexPositions[tri.VertexIndices[2]])

	vc1 := mesh.VertexColors[tri.VertexIndices[0]][channelIndex]
	vc2 := mesh.VertexColors[tri.VertexIndices[1]][channelIndex]
	vc3 := mesh.VertexColors[tri.VertexIndices[2]][channelIndex]

	output := vc1.Mix(vc2, float32(v)).Mix(vc3, float32(u))

	return output, nil

}

// UV returns the UV value from the position struck on the corresponding triangle for the BoundingObject struck,
// assuming the object struck was a BoundingTriangles.
// The returned UV value is linearly interpolated across the triangle just like it would be when a triangle is rendered.
// UV will return a zero Vector and an error if the BoundingObject hit was not a BoundingTriangles object.
func (r RayHit) UV() (Vector, error) {

	if r.Triangle == nil {
		return NewVector2d(0, 0), errors.New(ErrorObjectHitNotBoundingTriangles)
	}

	mesh := r.Object.(*BoundingTriangles).Mesh

	tri := r.Triangle
	u, v := pointInsideTriangle(r.untransformedPosition, mesh.VertexPositions[tri.VertexIndices[0]], mesh.VertexPositions[tri.VertexIndices[1]], mesh.VertexPositions[tri.VertexIndices[2]])

	uv1 := mesh.VertexUVs[tri.VertexIndices[0]]
	uv2 := mesh.VertexUVs[tri.VertexIndices[1]]
	uv3 := mesh.VertexUVs[tri.VertexIndices[2]]

	output := uv1.Lerp(uv2, v).Lerp(uv3, u)

	return output, nil

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
			from:     from,
			Normal:   strikePos.Sub(center).Unit(),
		}, true
	}

	return RayHit{}, false

}

func boundingAABBRayTest(from, to Vector, test *BoundingAABB) (RayHit, bool) {

	rayLine := to.Sub(from)
	rayLineUnit := rayLine.Unit()

	pos := test.WorldPosition()

	t1 := (test.Dimensions.Min.X + pos.X - from.X) / rayLineUnit.X
	t2 := (test.Dimensions.Max.X + pos.X - from.X) / rayLineUnit.X
	t3 := (test.Dimensions.Min.Y + pos.Y - from.Y) / rayLineUnit.Y
	t4 := (test.Dimensions.Max.Y + pos.Y - from.Y) / rayLineUnit.Y
	t5 := (test.Dimensions.Min.Z + pos.Z - from.Z) / rayLineUnit.Z
	t6 := (test.Dimensions.Max.Z + pos.Z - from.Z) / rayLineUnit.Z

	tmin := max(max(min(t1, t2), min(t3, t4)), min(t5, t6))
	tmax := min(min(max(t1, t2), max(t3, t4)), max(t5, t6))

	if math.IsNaN(tmin) || math.IsNaN(tmax) {
		return RayHit{}, false
	}

	if tmin < 0 {
		return RayHit{}, false
	}

	if tmin > tmax {
		return RayHit{}, false
	}

	vecLength := tmin

	if tmin < 0 {
		vecLength = tmax
	}

	if vecLength*vecLength > rayLine.MagnitudeSquared() {
		return RayHit{}, false
	}

	contact := from.Add(rayLineUnit.Scale(vecLength))

	return RayHit{
		Object:   test,
		Position: contact,
		Normal:   test.normalFromContactPoint(contact),
		from:     from,
	}, true

}

func boundingTrianglesRayTest(from, to Vector, test *BoundingTriangles, doublesided bool) (RayHit, bool) {

	rayDistSquared := to.DistanceSquared(from)

	check := false

	if test.BoundingAABB.PointInside(from) || test.BoundingAABB.PointInside(to) {
		check = true
	} else {
		if _, ok := boundingAABBRayTest(from, to, test.BoundingAABB); ok {
			check = true
		}
	}

	if check {

		_, _, r := test.Transform().Decompose()

		invertedTransform := test.Transform().Inverted()
		invFrom := invertedTransform.MultVec(from)
		invTo := invertedTransform.MultVec(to)
		plane := newCollisionPlane()

		for _, tri := range test.Mesh.Triangles {

			// If the distance from the start point to the triangle is longer than the ray,
			// then we know it can't be struck and we can bail early
			if invFrom.DistanceSquared(tri.Center) > rayDistSquared+(tri.MaxSpan*tri.MaxSpan) {
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

			if vec, ok := plane.RayAgainstPlane(invFrom, invTo, doublesided); ok {

				if isPointInsideTriangle(vec, v0, v1, v2) {

					return RayHit{
						Object:                test,
						Position:              test.Transform().MultVec(vec),
						untransformedPosition: vec,
						from:                  from,
						Triangle:              tri,
						Normal:                r.MultVec(tri.Normal),
					}, true

					// break

				}

			}
		}

	}

	return RayHit{}, false

}

var rayCylinder = NewBoundingTriangles("ray cylinder test", NewCylinderMesh(32, 1, false), 0)

var internalRayTest = []RayHit{}

// TODO: Add SphereCast?

// RayTestOptions is a struct designed to control what options to use when performing a ray test.
type RayTestOptions struct {
	From, To Vector // From and To are the starting and ending points of the ray test.

	// If cast rays can strike both sides of BoundingTriangles triangles or not.
	// TODO: Implement this for all collision types, not just triangles.
	Doublesided bool

	// TestAgainst is a slice of IBoundingObject nodes to check for collision against.
	TestAgainst []IBoundingObject

	// OnHit is a callback called for each hit a cast Ray returns, sorted by distance from the starting point.
	// OnHit is called for each object in order of distance to the starting point.
	// OnHit is only called once for each object, apart from BoundingTriangles, as a single ray can hit multiple triangles of a BoundingTriangles mesh.
	// hitIndex is the index of the hit out of the maximum number of hits found by the function (hitCount).
	// The returned boolean indicates whether to keep iterating through all found rayhits, or to stop after the current one.
	OnHit func(hit RayHit, hitIndex, hitCount int) bool
}

// RayTest casts a ray from the "from" world position to the "to" world position, testing against the provided
// IBoundingObjects.
// RayTest returns a slice of RayHit objects sorted from closest to furthest. Note that
// each object can only be struck once by the raycast, with the exception of BoundingTriangles objects (since a
// single ray may strike multiple triangles).
func RayTest(options RayTestOptions) bool {

	// We re-use the internal raytest function to avoid reallocating a slice.
	internalRayTest = internalRayTest[:0]

	for _, node := range options.TestAgainst {

		node.Transform() // Make sure the transform is updated before the test

		switch test := node.(type) {

		case *BoundingSphere:

			if result, ok := sphereRayTest(test.WorldPosition(), test.WorldRadius(), options.From, options.To); ok {
				result.Object = test
				internalRayTest = append(internalRayTest, result)
			}

		case *BoundingCapsule:

			radius := test.WorldRadius()

			first := test.lineTop()
			second := test.lineBottom()

			// We want to check the pole that's closest to the casting from point first, but we have to check both, technically,
			// since we don't know which pole the ray may pierce.
			if options.From.Distance(first) > options.From.Distance(second) {
				tmp := first
				first = second
				second = tmp
			}

			// test the cylinder in-between. This is done with a plain cylinder mesh because I'm too stupid to
			// figure out how to do it otherwise lmbo
			rayCylinder.SetWorldTransform(test.Transform())
			rayCylinder.SetLocalScale(radius, (test.Height/2)-radius, radius)

			if result, ok := boundingTrianglesRayTest(options.From, options.To, rayCylinder, options.Doublesided); ok {
				result.Object = test
				internalRayTest = append(internalRayTest, result)
			} else if result, ok := sphereRayTest(first, radius, options.From, options.To); ok {
				result.Object = test
				internalRayTest = append(internalRayTest, result)
			} else if result, ok := sphereRayTest(second, radius, options.From, options.To); ok {
				result.Object = test
				internalRayTest = append(internalRayTest, result)
			}

		case *BoundingAABB:

			if result, ok := boundingAABBRayTest(options.From, options.To, test); ok {
				internalRayTest = append(internalRayTest, result)
			}

		case *BoundingTriangles:

			if result, ok := boundingTrianglesRayTest(options.From, options.To, test, options.Doublesided); ok {
				internalRayTest = append(internalRayTest, result)
			}

		}

		// If we're not paying attention to the ray test results specifically, then we can bail after any valid
		// ray test result.
		if options.OnHit == nil && len(internalRayTest) > 0 {
			return true
		}

	}

	if options.OnHit != nil {

		sort.Slice(internalRayTest, func(i, j int) bool {
			return internalRayTest[i].Position.DistanceSquared(options.From) < internalRayTest[j].Position.DistanceSquared(options.From)
		})

		for i, r := range internalRayTest {
			if !options.OnHit(r, i, len(internalRayTest)) {
				break
			}
		}

	}

	return len(internalRayTest) > 0

}

// MouseRayTestOptions is a struct designed to control what options to use when
// performing a ray test from the Camera towards the Mouse.
type MouseRayTestOptions struct {
	// Depth is the distance to extend the ray in world units; defaults to the Camera's far plane.
	Depth float64
	// If cast rays can strike both sides of BoundingTriangles triangles or not.
	Doublesided bool
	// TestAgainstNodes is used to specify a slice of IBoundingObjects to test against.
	TestAgainst []IBoundingObject
	// OnHit is a callback called for each hit a cast Ray returns, sorted by distance from the starting point (the camera's position).
	// OnHit is called for each object in order of distance to the starting point.
	// OnHit is only called once for each object, apart from BoundingTriangles, as a single ray can hit multiple triangles of a BoundingTriangles mesh.
	// hitIndex is the index of the hit out of the maximum number of hits found by the function (hitCount).
	// The returned boolean indicates whether to keep iterating through all found rayhits, or to stop after the current one.
	OnHit func(hit RayHit, hitIndex int, hitCount int) bool
}

// MouseRayTest casts a ray forward from the mouse's position onscreen, testing against the provided
// IBoundingObjects found in the MouseRayTestOptions struct.
// The function calls the callback found in the MouseRayTestOptions struct for each object struck by the ray.
// The function returns a boolean indicating if any objects were struck at all.
// Note that each object can only be struck once by the raycast, with the exception of BoundingTriangles
// objects (since a single ray may strike multiple triangles).
func (camera *Camera) MouseRayTest(options MouseRayTestOptions) bool {

	if options.Depth <= 0 {
		options.Depth = camera.far
	}

	from := camera.WorldPosition()

	mx, my := ebiten.CursorPosition()

	if ebiten.CursorMode() == ebiten.CursorModeCaptured {
		mx = camera.ColorTexture().Bounds().Dx() / 2
		my = camera.ColorTexture().Bounds().Dy() / 2
	}

	to := camera.ScreenToWorldPixels(mx, my, options.Depth)

	return RayTest(RayTestOptions{
		From:        from,
		To:          to,
		OnHit:       options.OnHit,
		TestAgainst: options.TestAgainst,
		Doublesided: options.Doublesided,
	})

}
