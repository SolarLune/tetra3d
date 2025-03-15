package tetra3d

import (
	"github.com/solarlune/tetra3d/math32"
)

// BoundingCapsule represents a 3D capsule, whose primary purpose is to perform intersection testing between itself and other Bounding Nodes.
type BoundingCapsule struct {
	*Node
	Height         float32
	Radius         float32
	internalSphere *BoundingSphere
}

// NewBoundingCapsule returns a new BoundingCapsule instance. Name is the name of the underlying Node for the Capsule, height is the total
// height of the Capsule, and radius is how big around the capsule is. Height has to be at least radius (otherwise, it would no longer be a capsule).
func NewBoundingCapsule(name string, height, radius float32) *BoundingCapsule {
	cap := &BoundingCapsule{
		Node:           NewNode(name),
		Height:         math32.Max(radius, height),
		Radius:         radius,
		internalSphere: NewBoundingSphere("internal capsule sphere", 0),
	}
	cap.owner = cap
	return cap
}

// Clone returns a new BoundingCapsule.
func (capsule *BoundingCapsule) Clone() INode {
	clone := NewBoundingCapsule(capsule.name, capsule.Height, capsule.Radius)
	clone.Node = capsule.Node.clone(clone).(*Node)
	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}
	return clone
}

// WorldRadius is the radius of the Capsule in world units, after taking into account its scale.
func (capsule *BoundingCapsule) WorldRadius() float32 {
	maxScale := float32(1.0)
	if capsule.Node != nil {
		scale := capsule.Node.LocalScale()
		maxScale = math32.Max(math32.Max(math32.Abs(scale.X), math32.Abs(scale.Y)), math32.Abs(scale.Z))
	}
	return capsule.Radius * maxScale
}

// Colliding returns true if the BoundingCapsule is intersecting the other BoundingObject.
func (capsule *BoundingCapsule) Colliding(other IBoundingObject) bool {
	return capsule.Collision(other) != nil
}

// Collision returns a Collision struct if the BoundingCapsule is intersecting another BoundingObject. If
// no intersection is reported, Collision returns nil.
func (capsule *BoundingCapsule) Collision(other IBoundingObject) *Collision {

	if other == capsule || other == nil {
		return nil
	}

	switch otherBounds := other.(type) {

	case *BoundingCapsule:
		return btCapsuleCapsule(capsule, otherBounds)

	case *BoundingSphere:
		intersection := btSphereCapsule(otherBounds, capsule)
		if intersection != nil {
			for _, inter := range intersection.Intersections {
				inter.MTV = inter.MTV.Invert()
				inter.Normal = inter.Normal.Invert()
			}
			intersection.Object = otherBounds
		}
		return intersection

	case *BoundingAABB:
		return btCapsuleAABB(capsule, otherBounds)

	case *BoundingTriangles:
		return btCapsuleTriangles(capsule, otherBounds)

	}

	panic("Unimplemented bounds type")

}

// CollisionTest performs a collision test using the provided collision test settings structure.
// Collisions reported will be sorted in distance from closest to furthest.
// The function will return the first collision found with the object; if no collision is found, then it returns nil.
func (capsule *BoundingCapsule) CollisionTest(settings CollisionTestSettings) *Collision {
	return commonCollisionTest(capsule, settings)
}

// PointInside returns true if the point provided is within the capsule.
func (capsule *BoundingCapsule) PointInside(point Vector3) bool {
	return capsule.ClosestPoint(point).Sub(point).Magnitude() < capsule.WorldRadius()
}

// ClosestPoint returns the closest point on the capsule's "central line" to the point provided. Essentially, ClosestPoint returns a point
// along the capsule's line in world coordinates, capped between its bottom and top.
func (capsule *BoundingCapsule) ClosestPoint(point Vector3) Vector3 {

	return ClosestPointOnLine(capsule.Bottom(), capsule.Top(), point)

	// up := capsule.Node.WorldRotation().Up()
	// pos := capsule.Node.WorldPosition()

	// start := pos
	// start.X += up.X * (-capsule.Height/2 + capsule.Radius)
	// start.Y += up.Y * (-capsule.Height/2 + capsule.Radius)
	// start.Z += up.Z * (-capsule.Height/2 + capsule.Radius)

	// end := pos
	// end.X += up.X * (capsule.Height/2 - capsule.Radius)
	// end.Y += up.Y * (capsule.Height/2 - capsule.Radius)
	// end.Z += up.Z * (capsule.Height/2 - capsule.Radius)

	// segment := end.Sub(start)

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

	// return start

}

func (capsule *BoundingCapsule) closestPointCapsules(other *BoundingCapsule) Vector3 {

	aNormal := capsule.Top().Sub(capsule.Bottom())
	aOffset := aNormal.Scale(capsule.Radius)
	aStart := capsule.Bottom().Add(aOffset)
	aEnd := capsule.Top().Add(aOffset)

	bNormal := other.Top().Sub(other.Bottom())
	bOffset := bNormal.Scale(other.Radius)
	bStart := other.Bottom().Add(bOffset)
	bEnd := other.Top().Add(bOffset)

	v0 := bStart.Sub(aStart)
	v1 := bEnd.Sub(aStart)
	v2 := bStart.Sub(aEnd)
	v3 := bEnd.Sub(aEnd)

	d0 := v0.Dot(v0)
	d1 := v1.Dot(v1)
	d2 := v2.Dot(v2)
	d3 := v3.Dot(v3)

	var bestA Vector3

	if d2 < d0 || d2 < d1 || d3 < d0 || d3 < d1 {
		bestA = aEnd
	} else {
		bestA = aStart
	}

	bestB := ClosestPointOnLine(bStart, bEnd, bestA)

	bestA = ClosestPointOnLine(aStart, aEnd, bestB)

	return bestA

	// capsule.

	// capsule A:
	// float3 a_Normal = normalize(a.tip – a.base);
	// float3 a_LineEndOffset = a_Normal * a.radius;
	// float3 a_A = a.base + a_LineEndOffset;
	// float3 a_B = a.tip - a_LineEndOffset;

	// // capsule B:
	// float3 b_Normal = normalize(b.tip – b.base);
	// float3 b_LineEndOffset = b_Normal * b.radius;
	// float3 b_A = b.base + b_LineEndOffset;
	// float3 b_B = b.tip - b_LineEndOffset;

	// // vectors between line endpoints:
	// float3 v0 = b_A – a_A;
	// float3 v1 = b_B – a_A;
	// float3 v2 = b_A – a_B;
	// float3 v3 = b_B – a_B;

	// // squared distances:
	// float d0 = dot(v0, v0);
	// float d1 = dot(v1, v1);
	// float d2 = dot(v2, v2);
	// float d3 = dot(v3, v3);

	// // select best potential endpoint on capsule A:
	// float3 bestA;
	// if (d2 < d0 || d2 < d1 || d3 < d0 || d3 < d1)
	// {
	// bestA = a_B;
	// }
	// else
	// {
	// bestA = a_A;
	// }

	// // select point on capsule B line segment nearest to best potential endpoint on A capsule:
	// float3 bestB = ClosestPointOnLineSegment(b_A, b_B, bestA);

	// // now do the same for capsule A segment:
	// bestA = ClosestPointOnLineSegment(a_A, a_B, bestB);

}

// Returns the closest point on the capsule and the closest point on the given line.
func (capsule *BoundingCapsule) nearestPointsToLine(lineStart, lineEnd Vector3) (Vector3, Vector3) {

	eps := float32(1e-6)

	up := capsule.Node.WorldRotation().Up()
	heightScale := (capsule.Height / 2) - capsule.WorldRadius()
	if heightScale < 0 {
		heightScale = 0
	}

	capStart := capsule.Node.WorldPosition().Add(up.Scale(-heightScale))
	capEnd := capsule.Node.WorldPosition().Add(up.Scale(heightScale))

	r := lineStart.Sub(capStart)
	u := capEnd.Sub(capStart)
	v := lineEnd.Sub(lineStart)

	ru := r.Dot(u)
	rv := r.Dot(v)
	uu := u.Dot(u)
	uv := u.Dot(v)
	vv := v.Dot(v)

	det := uu*vv - uv*uv
	var t float32
	var s float32

	// If determinant is 0, then go with the first calculation because number / 0 makes s and t NaNs.
	if det < eps*uu*vv || det == 0 {
		s = math32.Clamp(ru/uu, 0, 1)
		t = 0
	} else {
		s = math32.Clamp((ru*vv-rv*uv)/det, 0, 1)
		t = math32.Clamp((ru*uv-rv*uu)/det, 0, 1)
	}

	S := math32.Clamp((t*uv+ru)/uu, 0, 1)
	T := math32.Clamp((s*uv-rv)/vv, 0, 1)

	closestCapsulePoint := capStart.Add(u.Scale(S))
	closestRayPoint := lineStart.Add(v.Scale(T))

	return closestCapsulePoint, closestRayPoint
	// B := b0 + T*v

	// return A, B, (B - A):Length()

	//////

	// pa := capsule.Bottom()
	// pb := capsule.Top()

	// ba := pb.Sub(pa)
	// oa := start.Sub(pa)

	// ra := capsule.WorldRadius()
	// ro := start
	// rd := end

	// baba := ba.Dot(ba)
	// bard := ba.Dot(rd)
	// baoa := ba.Dot(oa)
	// rdoa := rd.Dot(oa)
	// oaoa := oa.Dot(oa)

	// a := baba - bard*bard
	// b := baba*rdoa - baoa*bard
	// c := baba*oaoa - baoa*baoa - ra*ra*baba
	// h := b*b - a*c

	// if h >= 0 {
	// 	t := (-b - Sqrt(h)) / a
	// 	y := baoa + t*bard

	// 	if y > 0 && y < baba {
	// 		fmt.Println("hit", t, pa.Add(ba.Scale(t)))
	// 		return pa.Add(ba.Scale(t)), true
	// 	}

	// 	var oc Vector

	// 	if y <= 0 {
	// 		oc = oa
	// 	} else {
	// 		oc = ro.Sub(pb)
	// 	}

	// 	b = rd.Dot(oc)
	// 	c = oc.Dot(oc) - ra*ra
	// 	h = b*b - c

	// 	if h > 0 {
	// 		return pa.Add(ba.Scale(-b - Sqrt(h))), true
	// 	}

	// }

	// return Vector{}, false

	// v3f ba = v3f_sub(pb, pa);
	// v3f oa = v3f_sub(ro, pa);

	// float baba = v3f_dot(ba, ba);
	// float bard = v3f_dot(ba, rd);
	// float baoa = v3f_dot(ba, oa);
	// float rdoa = v3f_dot(rd, oa);
	// float oaoa = v3f_dot(oa, oa);

	// float a = baba      - bard*bard;
	// float b = baba*rdoa - baoa*bard;
	// float c = baba*oaoa - baoa*baoa - ra*ra*baba;
	// float h = b*b - a*c;

	// if (h >= 0.0f) {
	// 		float t = (-b-sqrt(h))/a;
	// 		float y = baoa + t*bard;

	// 		// body
	// 		if (y > 0.0f && y < baba) {
	// 				*distance = t;
	// 				return true;
	// 		}

	// 		// caps
	// 		v3f oc = (y <= 0.0) ? oa : v3f_sub(ro, pb);

	// 		b = v3f_dot(rd, oc);
	// 		c = v3f_dot(oc, oc) - ra*ra;
	// 		h = b*b - c;

	// 		if (h > 0.0f) {
	// 				*distance = -b - sqrtf(h);
	// 				return true;
	// 		}
	// }
	// return false;

	//////

	// eps := 0.00001

	// up := capsule.Node.WorldRotation().Up().Unit()
	// // radius := capsule.WorldRadius()
	// s := (capsule.Height / 2)
	// top := capsule.Node.WorldPosition().Add(up.Scale(s))
	// bottom := capsule.Node.WorldPosition().Add(up.Scale(-s))

	// p1 := bottom
	// p2 := top

	// p3 := start
	// p4 := end

	// p13 := p1.Sub(p3)
	// p43 := p4.Sub(p3)

	// if p43.IsZero() {
	// 	return Vector{}, false
	// }

	// p21 := p2.Sub(p1)

	// if p21.IsZero() {
	// 	return Vector{}, false
	// }

	// d1343 := p13.Dot(p43)
	// d4321 := p43.Dot(p21)
	// d1321 := p13.Dot(p21)
	// d4343 := p43.Dot(p43)
	// d2121 := p21.Dot(p21)

	// denom := d2121*d4343 - d4321*d4321

	// if Abs(denom) < eps {
	// 	return Vector{}, false
	// }

	// numer := d1343*d4321 - d1321*d4343
	// mua := numer / denom
	// mub := (d1343 + d4321*mua) / d4343

	// pa := p1.Add(p21.Scale(mua))
	// pb := p3.Add(p43.Scale(mub))

	// fmt.Println(pa.Sub(pb).Magnitude(), mua, mub)

	// if (mua >= 0 && mua <= 1) && (mub >= 0 && mub <= 1) {
	// 	return pa, true
	// }

	// return Vector{}, false

	////

	// aNormal := capsule.Node.WorldRotation().Up().Unit()
	// aOffset := aNormal.Scale(capsule.Radius)
	// aStart := capsule.Bottom().Add(aOffset)
	// aEnd := capsule.Top().Add(aOffset)

	// v0 := start.Sub(aStart)
	// v1 := end.Sub(aStart)
	// v2 := start.Sub(aEnd)
	// v3 := end.Sub(aEnd)

	// d0 := v0.Dot(v0)
	// d1 := v1.Dot(v1)
	// d2 := v2.Dot(v2)
	// d3 := v3.Dot(v3)

	// var bestA Vector

	// if d2 < d0 || d2 < d1 || d3 < d0 || d3 < d1 {
	// 	bestA = aEnd
	// } else {
	// 	bestA = aStart
	// }

	// bestB := ClosestPointOnLine(start, end, bestA)

	// bestA = ClosestPointOnLine(aStart, aEnd, bestB)

	// return bestA

	/////

	// capsule.

	// capsule A:
	// float3 a_Normal = normalize(a.tip – a.base);
	// float3 a_LineEndOffset = a_Normal * a.radius;
	// float3 a_A = a.base + a_LineEndOffset;
	// float3 a_B = a.tip - a_LineEndOffset;

	// // capsule B:
	// float3 b_Normal = normalize(b.tip – b.base);
	// float3 b_LineEndOffset = b_Normal * b.radius;
	// float3 b_A = b.base + b_LineEndOffset;
	// float3 b_B = b.tip - b_LineEndOffset;

	// // vectors between line endpoints:
	// float3 v0 = b_A – a_A;
	// float3 v1 = b_B – a_A;
	// float3 v2 = b_A – a_B;
	// float3 v3 = b_B – a_B;

	// // squared distances:
	// float d0 = dot(v0, v0);
	// float d1 = dot(v1, v1);
	// float d2 = dot(v2, v2);
	// float d3 = dot(v3, v3);

	// // select best potential endpoint on capsule A:
	// float3 bestA;
	// if (d2 < d0 || d2 < d1 || d3 < d0 || d3 < d1)
	// {
	// bestA = a_B;
	// }
	// else
	// {
	// bestA = a_A;
	// }

	// // select point on capsule B line segment nearest to best potential endpoint on A capsule:
	// float3 bestB = ClosestPointOnLineSegment(b_A, b_B, bestA);

	// // now do the same for capsule A segment:
	// bestA = ClosestPointOnLineSegment(a_A, a_B, bestB);

}

// lineTop returns the world position of the internal top end of the BoundingCapsule's line (i.e. this subtracts the
// capsule's radius).
func (capsule *BoundingCapsule) lineTop() Vector3 {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(capsule.Height/2 - capsule.Radius))
}

// Top returns the world position of the top of the BoundingCapsule.
func (capsule *BoundingCapsule) Top() Vector3 {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(capsule.Height / 2))
}

// lineBottom returns the world position of the internal bottom end of the BoundingCapsule's line (i.e. this subtracts the
// capsule's radius).
func (capsule *BoundingCapsule) lineBottom() Vector3 {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(-capsule.Height/2 + capsule.Radius))
}

// Bottom returns the world position of the bottom of the BoundingCapsule.
func (capsule *BoundingCapsule) Bottom() Vector3 {
	up := capsule.Node.WorldRotation().Up()
	return capsule.Node.WorldPosition().Add(up.Scale(-capsule.Height / 2))
}

/////

// Type returns the NodeType for this object.
func (capsule *BoundingCapsule) Type() NodeType {
	return NodeTypeBoundingCapsule
}
