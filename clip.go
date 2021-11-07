package jank3d

// type clipPlane struct {
// 	Depth, Normal vector.Vector
// }

// var clip = NewClipPlane(vector.Vector{0, 0, -1, 1}, vector.Vector{0, 0, 1, 1})

// func NewClipPlane(position, normal vector.Vector) clipPlane {
// 	return clipPlane{position, normal}
// }

// func (clip clipPlane) PointInFront(point vector.Vector) bool {
// 	return point.Sub(clip.Depth).Dot(clip.Normal) > -1
// }

// func (clip clipPlane) Intersection(start, end vector.Vector) vector.Vector {

// 	u := end.Sub(start)
// 	w := start.Sub(clip.Depth)
// 	d := clip.Normal.Dot(u)
// 	n := -clip.Normal.Dot(w)
// 	return start.Add(u.Scale(n / d))

// }
