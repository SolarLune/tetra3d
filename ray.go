package tetra3d

import (
	"errors"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/solarlune/tetra3d/math32"
)

// RayHit represents the result of a raycast test.
type RayHit struct {
	Object   INode   // Object is a pointer to the BoundingObject that was struck by the raycast.
	Position Vector3 // Position is the world position that the object was struct.
	from     Vector3 // The starting position of the Ray
	Normal   Vector3 // Normal is the normal of the surface the ray struck.

	// What triangle the raycast hit - note that this is only set to a non-nil value for raycasts against BoundingTriangle objects
	Triangle              *Triangle
	untransformedPosition Vector3 // untransformed position of the ray test for BoundingTriangles tests
}

// Slope returns the slope of the RayHit's normal, in radians. This ranges from 0 (straight up) to pi (straight down).
func (r RayHit) Slope() float32 {
	return WorldUp.Angle(r.Normal)
}

// Distance returns the distance from the RayHit's originating ray source point to the struck position.
func (r RayHit) Distance() float32 {
	return r.from.DistanceTo(r.Position)
}

const ErrorObjectHitNotBoundingTriangles = "error: object hit not a BoundingTriangles instance; no UV or vertex color data can be pulled from RayHit result"

// Returns the vertex color from the given channel in the position struck on the object struck.
// Only works when testing against BoundingTriangles objects.
// The returned vertex color is linearly interpolated across the triangle just like it would be when a triangle is rendered.
// The function will return transparent black and an error if the BoundingObject hit was not a BoundingTriangles object, or if the channel index given
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

	vc1 := mesh.VertexColors[channelIndex][tri.VertexIndices[0]]
	vc2 := mesh.VertexColors[channelIndex][tri.VertexIndices[1]]
	vc3 := mesh.VertexColors[channelIndex][tri.VertexIndices[2]]

	output := vc1.Mix(vc2, float32(v)).Mix(vc3, float32(u))

	return output, nil

}

// Returns the lighting color for the given position struck on the object struck.
// Only works when testing against BoundingTriangles objects, and those for visible objects.
// Note that this assumes the object rendered is rendered once; rendering multiple times will reset the lighting values between renders.
// The function will return transparent black and an error if the BoundingObject hit was not a BoundingTriangles object.
func (r RayHit) LightColor() (Color, error) {

	if r.Triangle == nil {
		return NewColor(0, 0, 0, 0), errors.New(ErrorObjectHitNotBoundingTriangles)
	}

	tri := r.Triangle

	// Shadeless materials return full bright white
	if tri.MeshPart.Material != nil && tri.MeshPart.Material.Shadeless {
		return NewColor(1, 1, 1, 1), nil
	}

	mesh := r.Object.(*BoundingTriangles).Mesh

	u, v := pointInsideTriangle(r.untransformedPosition, mesh.VertexPositions[tri.VertexIndices[0]], mesh.VertexPositions[tri.VertexIndices[1]], mesh.VertexPositions[tri.VertexIndices[2]])

	vc1 := mesh.vertexLights[tri.VertexIndices[0]]
	vc2 := mesh.vertexLights[tri.VertexIndices[1]]
	vc3 := mesh.vertexLights[tri.VertexIndices[2]]

	output := vc1.Mix(vc2, float32(v)).Mix(vc3, float32(u))

	return output, nil

}

// UV returns the UV value from the position struck on the corresponding triangle for the BoundingObject struck,
// assuming the object struck was a BoundingTriangles.
// The returned UV value is linearly interpolated across the triangle just like it would be when a triangle is rendered.
// UV will return a zero Vector and an error if the BoundingObject hit was not a BoundingTriangles object.
func (r RayHit) UV() (Vector2, error) {

	if r.Triangle == nil {
		return Vector2{}, errors.New(ErrorObjectHitNotBoundingTriangles)
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

func boundingSphereRayTest(center Vector3, radius float32, from, to Vector3) (RayHit, bool) {

	// normal := to.Sub(from)
	// normalUnit := normal.Unit()

	// e := center.Sub(from)

	// esq := e.MagnitudeSquared()
	// a := e.Dot(normalUnit)
	// b := Sqrt(esq - (a * a))
	// // if IsNaN(b) {
	// // 	return RayHit{}, false
	// // }
	// f := Sqrt((radius * radius) - (b * b))

	// // fmt.Println("a:", esq-(a*a), esq, a, b, radius*radius-esq+a*a)
	// t := 0.0

	// if radius*radius-esq+a*a < 0 {
	// 	return RayHit{}, false
	// } else if esq < radius*radius {
	// 	t = a + f
	// } else {
	// 	t = a - f
	// }

	// if t < 0 {
	// 	return RayHit{}, false
	// }

	// if t*t > normal.MagnitudeSquared() {
	// 	return RayHit{}, false
	// }

	// hitPos := from.Add(normalUnit.Scale(t))

	// return RayHit{
	// 	from:     from,
	// 	Position: hitPos,
	// 	Normal:   hitPos.Sub(center).Unit(),
	// }, true

	///////

	m := from.Sub(center)
	vec := to.Sub(from)
	normal := vec.Unit()
	b := m.Dot(normal)
	c := m.Dot(m) - radius*radius

	if c > 0 && b > 0 {
		return RayHit{}, false
	}

	discr := b*b - c

	if discr < 0 {
		return RayHit{}, false
	}

	t := -b - math32.Sqrt(discr)

	if t < 0 {
		t = 0
	}

	if t*t > vec.MagnitudeSquared() {
		return RayHit{}, false
	}

	strikePos := from.Add(normal.Scale(t))
	return RayHit{
		Position: strikePos,
		from:     from,
		Normal:   strikePos.Sub(center).Unit(),
	}, true

	//////////

	// line := to.Sub(from)
	// dir := line.Unit()

	// e := center.Sub(from)

	// esq := e.MagnitudeSquared()
	// a := e.Dot(dir)
	// b := Sqrt(esq - (a * a))
	// f := Sqrt((radius * radius) - (b * b))

	// vecLength := 0.0

	// if radius*radius-esq+a*a < 0 {
	// 	vecLength = -1
	// } else if esq < radius*radius {
	// 	vecLength = a + f
	// } else {
	// 	vecLength = a - f
	// }

	// if vecLength*vecLength > line.MagnitudeSquared() {
	// 	return RayHit{}, false
	// }

	// if vecLength >= 0 {
	// 	strikePos := from.Add(dir.Scale(vecLength))
	// 	return RayHit{
	// 		Position: strikePos,
	// 		from:     from,
	// 		Normal:   strikePos.Sub(center).Unit(),
	// 	}, true
	// }

	// return RayHit{}, false

}

func boundingAABBRayTest(from, to Vector3, test *BoundingAABB) (RayHit, bool) {

	rayLine := to.Sub(from)
	rayLineUnit := rayLine.Unit()

	pos := test.WorldPosition()

	t1 := (test.Dimensions.Min.X + pos.X - from.X) / rayLineUnit.X
	t2 := (test.Dimensions.Max.X + pos.X - from.X) / rayLineUnit.X
	t3 := (test.Dimensions.Min.Y + pos.Y - from.Y) / rayLineUnit.Y
	t4 := (test.Dimensions.Max.Y + pos.Y - from.Y) / rayLineUnit.Y
	t5 := (test.Dimensions.Min.Z + pos.Z - from.Z) / rayLineUnit.Z
	t6 := (test.Dimensions.Max.Z + pos.Z - from.Z) / rayLineUnit.Z

	tmin := math32.Max(math32.Max(math32.Min(t1, t2), math32.Min(t3, t4)), math32.Min(t5, t6))
	tmax := math32.Min(math32.Min(math32.Max(t1, t2), math32.Max(t3, t4)), math32.Max(t5, t6))

	if math32.IsNaN(tmin) || math32.IsNaN(tmax) {
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

func boundingTrianglesRayTest(from, to Vector3, test *BoundingTriangles, doublesided bool) []RayHit {

	check := false

	if test.BoundingAABB.PointInside(from) || test.BoundingAABB.PointInside(to) {
		check = true
	} else if _, ok := boundingAABBRayTest(from, to, test.BoundingAABB); ok {
		check = true
	}

	results := []RayHit{}

	if check {

		center := from.Lerp(to, 0.5)
		radius := from.DistanceTo(to)

		_, _, r := test.Transform().Decompose()

		invertedTransform := test.Transform().Inverted()
		invFrom := invertedTransform.MultVec(from)
		invTo := invertedTransform.MultVec(to)
		plane := newCollisionPlane()

		test.Broadphase.ForEachTriangleInRange(center, radius, func(triID int) bool {

			tri := test.Mesh.Triangles[triID]

			fs := tri.Normal.Dot(invFrom.Sub(tri.Center))
			ts := tri.Normal.Dot(invTo.Sub(tri.Center))

			// If the start and end points of the ray lie on the same side of the triangle,
			// then we know the triangle can't be struck and we can bail early
			if (fs > 0 && ts > 0) || (fs < 0 && ts < 0) {
				return true
			}

			v0 := test.Mesh.VertexPositions[tri.VertexIndices[0]]
			v1 := test.Mesh.VertexPositions[tri.VertexIndices[1]]
			v2 := test.Mesh.VertexPositions[tri.VertexIndices[2]]

			// Skip because the triangle is degenerate and a collision plane cannot be set from it
			// (i.e. for whatever reason, at least two vertices share location, so it's no longer a pure triangle)
			if v0.Equals(v1) || v1.Equals(v2) || v2.Equals(v0) {
				return true
			}

			plane.Set(v0, v1, v2)

			if vec, ok := plane.RayAgainstPlane(invFrom, invTo, doublesided); ok {

				if isPointInsideTriangle(vec, v0, v1, v2) {

					results = append(results, RayHit{
						Object:                test,
						Position:              test.Transform().MultVec(vec),
						untransformedPosition: vec,
						from:                  from,
						Triangle:              tri,
						Normal:                r.MultVec(tri.Normal),
					})

				}

			}

			return true

		})

	}

	return results

}

var internalRayTest = []RayHit{}

// TODO: Add SphereCast?

// RayTestOptions is a struct designed to control what options to use when performing a ray test.
type RayTestOptions struct {
	From Vector3 // The position to cast rays from.
	To   Vector3 // The position to cast rays to.

	// The positions to cast rays from and to in pairs.
	// By specifying multiple locations in this location pair set, you can cast multiple rays at the same time.
	// If the number of positions provided is odd (e.g. only a From and no matching To), that ray is not cast.
	// Overrides the options object's From and To properties.
	Positions []Vector3

	// If cast rays can strike both sides of BoundingTriangles triangles or not.
	// TODO: Implement this for all collision types, not just triangles.
	Doublesided bool

	// TestAgainst is used to specify a selection of BoundingObjects to test against - this can be either a NodeFilter or a NodeCollection (a slice of Nodes).
	TestAgainst NodeIterator

	// OnHit is a callback called for each hit a cast Ray returns, sorted by distance from the starting point.
	// OnHit is called for each object in order of distance to the starting point.
	// OnHit is only called once for each object, apart from BoundingTriangles, as a single ray can hit multiple triangles of a BoundingTriangles mesh.
	// index is the index of the hit out of the maximum number of hits found by the function (count).
	// The returned boolean indicates whether to keep iterating through all found rayhits, or to stop after the current one.
	OnHit func(hit RayHit, index, count int) bool
}

// AddPosition adds a position pair to cast rays from and to.
func (r RayTestOptions) AddPosition(from, to Vector3) RayTestOptions {
	r.Positions = append(r.Positions, from, to)
	return r
}

// WithOnHit sets the callback to be called for each hit a cast Ray returns, sorted by distance from the starting point.
func (r RayTestOptions) WithOnHit(onHit func(hit RayHit, index, count int) bool) RayTestOptions {
	r.OnHit = onHit
	return r
}

func (r RayTestOptions) WithDoublesided(doubleSided bool) RayTestOptions {
	r.Doublesided = doubleSided
	return r
}

func (r RayTestOptions) WithTestAgainst(iterator NodeIterator) RayTestOptions {
	r.TestAgainst = iterator
	return r
}

// RayTest casts a ray from the "from" world position to the "to" world position for each position pair
// while testing against the provided IBoundingObjects.
// The function returns the first struck object; if none were struck, it returns nil.
func RayTest(options RayTestOptions) *RayHit {

	// We re-use the internal raytest function to avoid reallocating a slice.
	internalRayTest = internalRayTest[:0]

	options.TestAgainst.ForEach(func(node INode) bool {

		node.Transform() // Make sure the transform is updated before the test

		testFunc := func(from, to Vector3) {

			switch test := node.(type) {

			case *BoundingSphere:

				if result, ok := boundingSphereRayTest(test.WorldPosition(), test.WorldRadius(), from, to); ok {
					result.Object = test
					internalRayTest = append(internalRayTest, result)
				}

			case *BoundingCapsule:

				closestPoint, _ := test.nearestPointsToLine(from, to)

				if result, ok := boundingSphereRayTest(closestPoint, test.WorldRadius(), from, to); ok {
					result.Object = test
					internalRayTest = append(internalRayTest, result)
				}

			case *BoundingAABB:

				if result, ok := boundingAABBRayTest(from, to, test); ok {
					internalRayTest = append(internalRayTest, result)
				}

			case *BoundingTriangles:

				// Raycasting against triangles can hit multiple triangles, so we can't bail early and have to return all potential hits
				internalRayTest = append(internalRayTest, boundingTrianglesRayTest(from, to, test, options.Doublesided)...)

			}

		}

		if len(options.Positions) > 0 {
			for i := 0; i < len(options.Positions); i += 2 {
				if len(options.Positions) < i+1 { // Break out if it's not a pair
					break
				}
				testFunc(options.Positions[i], options.Positions[i+1])
			}
		} else {
			testFunc(options.From, options.To)
		}

		return true

	})

	sort.Slice(internalRayTest, func(i, j int) bool {
		return internalRayTest[i].Position.DistanceSquaredTo(internalRayTest[i].from) < internalRayTest[j].Position.DistanceSquaredTo(internalRayTest[j].from)
	})

	if options.OnHit != nil {

		for i, r := range internalRayTest {
			if !options.OnHit(r, i, len(internalRayTest)) {
				break
			}
		}

	}

	if len(internalRayTest) > 0 {
		return &internalRayTest[0]
	}
	return nil

}

// MouseRayTestOptions is a struct designed to control what options to use when
// performing a ray test from the Camera towards the Mouse.
type MouseRayTestOptions struct {
	// Depth is the distance to extend the ray in world units; defaults to the Camera's far plane.
	Depth float32
	// If cast rays can strike both sides of BoundingTriangles triangles or not.
	Doublesided bool
	// TestAgainst is used to specify a selection of BoundingObjects to test against - this can be either a NodeFilter or a NodeCollection (a slice of Nodes).
	TestAgainst NodeIterator
	// OnHit is a callback called for each hit a cast Ray returns, sorted by distance from the starting point (the camera's position).
	// OnHit is called for each object in order of distance to the starting point.
	// OnHit is only called once for each object, apart from BoundingTriangles, as a single ray can hit multiple triangles of a BoundingTriangles mesh.
	// hitIndex is the index of the hit out of the maximum number of hits found by the function (hitCount).
	// The returned boolean indicates whether to keep iterating through all found rayhits, or to stop after the current one.
	OnHit func(hit RayHit, hitIndex int, hitCount int) bool
}

// WithDepth controls the maximum distance out from the camera the ray is cast.
func (r MouseRayTestOptions) WithDepth(depth float32) MouseRayTestOptions {
	r.Depth = depth
	return r
}

// WithOnHit sets the callback to be called for each hit a cast Ray returns, sorted by distance from the starting point.
func (r MouseRayTestOptions) WithOnHit(onHit func(hit RayHit, index, count int) bool) MouseRayTestOptions {
	r.OnHit = onHit
	return r
}

// WithDoublesided indicates if the ray test is double-sided or single-sided.
func (r MouseRayTestOptions) WithDoublesided(doubleSided bool) MouseRayTestOptions {
	r.Doublesided = doubleSided
	return r
}

// WithTestAgainst sets the test against iterator for the mouse raytest option set.
func (r MouseRayTestOptions) WithTestAgainst(iterator NodeIterator) MouseRayTestOptions {
	r.TestAgainst = iterator
	return r
}

// MouseRayTest casts a ray forward from the mouse's position onscreen, testing against the provided
// IBoundingObjects found in the MouseRayTestOptions struct.
// The function calls the callback found in the MouseRayTestOptions struct for each object struck by the ray.
// The function returns the first struck object; if none were struck, it returns nil.
// Note that each object can only be struck once by the raycast, with the exception of BoundingTriangles
// objects (since a single ray may strike multiple triangles).
func (camera *Camera) MouseRayTest(options MouseRayTestOptions) *RayHit {

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
