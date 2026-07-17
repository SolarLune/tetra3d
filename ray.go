package tetra3d

import (
	"errors"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/solarlune/tetra3d/math32"
)

// RayHit represents the result of a raycast test.
type RayHit struct {
	Object   INode   // Object is a pointer to the Collider that was struck by the raycast.
	Position Vector3 // Position is the world position that the object was struct.
	from     Vector3 // The starting position of the Ray
	Normal   Vector3 // Normal is the normal of the surface the ray struck.

	Triangle              *Triangle // What triangle the raycast hit for rays that strike ColliderTriangles objects
	untransformedPosition Vector3   // untransformed position of the ray test for ColliderTriangles tests
}

// Slope returns the slope of the RayHit's normal, in radians. This ranges from 0 (straight up) to pi (straight down).
func (r RayHit) Slope() float32 {
	return WorldUp.Angle(r.Normal)
}

// Distance returns the distance from the RayHit's originating ray source point to the struck position.
func (r RayHit) Distance() float32 {
	return r.from.DistanceTo(r.Position)
}

// Returns if the Ray hit a material by a given name or not.
func (r RayHit) HitMaterialByName(name string) bool {
	return r.Triangle != nil && r.Triangle.MeshPart.Material != nil && r.Triangle.MeshPart.Material.name == name
}

// Returns if the Ray hit a material with a property of a given name or not.
func (r RayHit) HitMaterialByPropName(propName string) bool {
	return r.Triangle != nil && r.Triangle.MeshPart.Material != nil && r.Triangle.MeshPart.Material.properties.Has(propName)
}

const ErrorObjectHitNotColliderTriangles = "error: object hit not a ColliderTriangles instance; no UV or vertex color data can be pulled from RayHit result"
const ErrorObjectHitNotChildOfModel = "error: object hit not a child of a Model"
const ErrorVertexChannelOutsideRange = "error: vertex color channel not found by given name"

// Returns the vertex color from the given channel in the position struck on the object struck.
// Only works when testing against ColliderTriangles objects.
// The returned vertex color is linearly interpolated across the triangle just like it would be when a triangle is rendered.
// The function will return transparent black and an error if the Collider hit was not a ColliderTriangles object, or if the channel index given
// is higher than the number of vertex color channels on the ColliderTriangles' mesh.
func (r RayHit) VertexColor(channelIndex int) (Color4, error) {

	if r.Triangle == nil {
		return NewColor4(0, 0, 0, 0), errors.New(ErrorObjectHitNotColliderTriangles)
	}

	mesh := r.Object.(*ColliderTriangles).Mesh

	if len(mesh.VertexColors[0].colors) <= channelIndex {
		return NewColor4(0, 0, 0, 0), errors.New(ErrorVertexChannelOutsideRange)
	}

	tri := r.Triangle
	u, v := pointInsideTriangle(r.untransformedPosition, mesh.VertexPositions[tri.VertexIndexA], mesh.VertexPositions[tri.VertexIndexB], mesh.VertexPositions[tri.VertexIndexC])

	vc1 := mesh.VertexColors[channelIndex].colors[tri.VertexIndexA]
	vc2 := mesh.VertexColors[channelIndex].colors[tri.VertexIndexB]
	vc3 := mesh.VertexColors[channelIndex].colors[tri.VertexIndexC]

	output := vc1.Mix(vc2, float32(v)).Mix(vc3, float32(u))

	return output, nil

}

// UV returns the UV value from the position struck on the corresponding triangle for the Collider struck,
// assuming the object struck was a ColliderTriangles.
// The returned UV value is linearly interpolated across the triangle just like it would be when a triangle is rendered.
// UV will return a zero Vector and an error if the Collider hit was not a ColliderTriangles object.
func (r RayHit) UV() (Vector2, error) {

	if r.Triangle == nil {
		return Vector2{}, errors.New(ErrorObjectHitNotColliderTriangles)
	}

	mesh := r.Object.(*ColliderTriangles).Mesh

	tri := r.Triangle
	u, v := pointInsideTriangle(r.untransformedPosition, mesh.VertexPositions[tri.VertexIndexA], mesh.VertexPositions[tri.VertexIndexB], mesh.VertexPositions[tri.VertexIndexC])

	uv1 := mesh.VertexUVs[tri.VertexIndexA]
	uv2 := mesh.VertexUVs[tri.VertexIndexB]
	uv3 := mesh.VertexUVs[tri.VertexIndexC]

	output := uv1.Lerp(uv2, v).Lerp(uv3, u)

	return output, nil

}

func capsuleSphereRayTest(center Vector3, radius float32, from, to Vector3) (RayHit, bool) {

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

func capsuleAABBRayTest(from, to Vector3, test *ColliderAABB) (RayHit, bool) {

	rayLine := to.Sub(from)
	rayLineUnit := rayLine.Unit()

	pos := test.WorldPosition()

	t1 := (test.dimensions.Min.X + pos.X - from.X) / rayLineUnit.X
	t2 := (test.dimensions.Max.X + pos.X - from.X) / rayLineUnit.X
	t3 := (test.dimensions.Min.Y + pos.Y - from.Y) / rayLineUnit.Y
	t4 := (test.dimensions.Max.Y + pos.Y - from.Y) / rayLineUnit.Y
	t5 := (test.dimensions.Min.Z + pos.Z - from.Z) / rayLineUnit.Z
	t6 := (test.dimensions.Max.Z + pos.Z - from.Z) / rayLineUnit.Z

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

var ColliderTrianglesRayTestResults = []RayHit{}

func ColliderTrianglesRayTest(from, to Vector3, test *ColliderTriangles, doublesided bool) []RayHit {

	check := false

	if test.colliderAABB.PointInside(from) || test.colliderAABB.PointInside(to) {
		check = true
	} else if _, ok := capsuleAABBRayTest(from, to, test.colliderAABB); ok {
		check = true
	}

	ColliderTrianglesRayTestResults = ColliderTrianglesRayTestResults[:0]

	if check {

		r := test.Transform().DecomposeRotation()

		invertedTransform := test.Transform().Inverted()
		invFrom := invertedTransform.MultVec(from)
		invTo := invertedTransform.MultVec(to)
		dir := invTo.Sub(invFrom).Unit()
		plane := newCollisionPlane()

		// TODO: Review this - it seems like it's drastically faster (~4x) to not use the broadphase object for raytests, even for meshes with relatively high triangle counts.
		// I guess the broadphase checking code is just way too slow? So we're just not going to use it for raycasting, I guess.

		// test.broadphase.ForEachTriangleFromCollider(workingAABB, func(tri *Triangle) bool {

		for _, tri := range test.Mesh.Triangles {

			v0 := test.Mesh.VertexPositions[tri.VertexIndexA]
			v1 := test.Mesh.VertexPositions[tri.VertexIndexB]
			v2 := test.Mesh.VertexPositions[tri.VertexIndexC]

			// Very simple culling - if the ray's start and end are wholly too far to the
			// right, left, above, below, ahead, or behind all vertices, then it can't possibly strike the triangle.
			// An additional ~30% speedup.
			if (invFrom.X > v0.X && invFrom.X > v1.X && invFrom.X > v2.X &&
				invTo.X > v0.X && invTo.X > v1.X && invTo.X > v2.X) ||
				(invFrom.X < v0.X && invFrom.X < v1.X && invFrom.X < v2.X &&
					invTo.X < v0.X && invTo.X < v1.X && invTo.X < v2.X) ||
				(invFrom.Y > v0.Y && invFrom.Y > v1.Y && invFrom.Y > v2.Y &&
					invTo.Y > v0.Y && invTo.Y > v1.Y && invTo.Y > v2.Y) ||
				(invFrom.Y < v0.Y && invFrom.Y < v1.Y && invFrom.Y < v2.Y &&
					invTo.Y < v0.Y && invTo.Y < v1.Y && invTo.Y < v2.Y) ||
				(invFrom.Z > v0.Z && invFrom.Z > v1.Z && invFrom.Z > v2.Z &&
					invTo.Z > v0.Z && invTo.Z > v1.Z && invTo.Z > v2.Z) ||
				(invFrom.Z < v0.Z && invFrom.Z < v1.Z && invFrom.Z < v2.Z &&
					invTo.Z < v0.Z && invTo.Z < v1.Z && invTo.Z < v2.Z) {
				continue
			}

			// Skip if it's not collideable
			if tri.MeshPart.Material != nil && !tri.MeshPart.Material.ReportRays {
				continue
			}

			fs := tri.Normal.Dot(invFrom.Sub(tri.Center))
			ts := tri.Normal.Dot(invTo.Sub(tri.Center))

			// If the start and end points of the ray lie on the same side of the triangle,
			// then we know the triangle can't be struck and we can bail early
			if (fs > 0 && ts > 0) || (fs < 0 && ts < 0) {
				continue
			}

			// Skip because the triangle is degenerate and a collision plane cannot be set from it
			// (i.e. for whatever reason, at least two vertices share location, so it's no longer a pure triangle)
			if v0.Equals(v1) || v1.Equals(v2) || v2.Equals(v0) {
				continue
			}

			// We can manually set the plane rather than using Plane.Set() as we have the triangle normal already.
			plane.Normal = tri.Normal
			plane.Distance = tri.Normal.Dot(v0)

			if vec, ok := plane.RayAgainstPlane(invFrom, dir, doublesided); ok {

				if isPointInsideTriangle(vec, v0, v1, v2) {

					ColliderTrianglesRayTestResults = append(ColliderTrianglesRayTestResults, RayHit{
						Object:                test,
						Position:              test.Transform().MultVec(vec),
						untransformedPosition: vec,
						from:                  from,
						Triangle:              tri,
						Normal:                r.MultVec(tri.Normal),
					})

				}

			}

		}

	}

	return ColliderTrianglesRayTestResults

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

	// If cast rays can strike both sides of ColliderTriangles triangles or not.
	// TODO: Implement this for all collision types, not just triangles.
	Doublesided bool

	// TestAgainst is used to specify a selection of Colliders to test against - this can be either a NodeFilter or a NodeCollection (a slice of Nodes).
	TestAgainst NodeIterator

	// OnHit is a callback called for each hit a cast Ray returns, sorted by distance from the starting point.
	// OnHit is called for each object in order of distance to the starting point.
	// OnHit is only called once for each object, apart from ColliderTriangles, as a single ray can hit multiple triangles of a ColliderTriangles mesh.
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
// while testing against the provided IColliders.
// The function returns the first struck object; if none were struck, it returns nil.
func RayTest(options RayTestOptions) *RayHit {

	// We re-use the internal raytest function to avoid reallocating a slice.
	internalRayTest = internalRayTest[:0]

	options.TestAgainst.ForEach(func(node INode, index int) bool {

		node.Transform() // Make sure the transform is updated before the test

		testFunc := func(from, to Vector3) {

			switch test := node.(type) {

			case *ColliderSphere:

				if result, ok := capsuleSphereRayTest(test.WorldPosition(), test.WorldRadius(), from, to); ok {
					result.Object = test
					internalRayTest = append(internalRayTest, result)
				}

			case *ColliderCapsule:

				closestPoint, _ := test.nearestPointsToLine(from, to)

				if result, ok := capsuleSphereRayTest(closestPoint, test.WorldRadius(), from, to); ok {
					result.Object = test
					internalRayTest = append(internalRayTest, result)
				}

			case *ColliderAABB:

				if result, ok := capsuleAABBRayTest(from, to, test); ok {
					internalRayTest = append(internalRayTest, result)
				}

			case *ColliderTriangles:

				// Raycasting against triangles can hit multiple triangles, so we can't bail early and have to return all potential hits
				internalRayTest = append(internalRayTest, ColliderTrianglesRayTest(from, to, test, options.Doublesided)...)

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

	if len(internalRayTest) > 1 {
		sort.Slice(internalRayTest, func(i, j int) bool {
			return internalRayTest[i].Position.DistanceSquaredTo(internalRayTest[i].from) < internalRayTest[j].Position.DistanceSquaredTo(internalRayTest[j].from)
		})
	}

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
	// If cast rays can strike both sides of ColliderTriangles triangles or not.
	Doublesided bool
	// TestAgainst is used to specify a selection of Colliders to test against - this can be either a NodeFilter or a NodeCollection (a slice of Nodes).
	TestAgainst NodeIterator
	// OnHit is a callback called for each hit a cast Ray returns, sorted by distance from the starting point (the camera's position).
	// OnHit is called for each object in order of distance to the starting point.
	// OnHit is only called once for each object, apart from ColliderTriangles, as a single ray can hit multiple triangles of a ColliderTriangles mesh.
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
// IColliders found in the MouseRayTestOptions struct.
// The function calls the callback found in the MouseRayTestOptions struct for each object struck by the ray.
// The function returns the first struck object; if none were struck, it returns nil.
// Note that each object can only be struck once by the raycast, with the exception of ColliderTriangles
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
