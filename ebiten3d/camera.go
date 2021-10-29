package ebiten3d

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

type Camera struct {
	ColorTexture *ebiten.Image
	Near, Far    float64

	Position     vector.Vector
	LookAtTarget vector.Vector

	Transform  Matrix4
	Projection Matrix4
	// Drawcalls   []Drawcall
}

func NewCamera(w, h int) *Camera {

	cam := &Camera{
		ColorTexture: ebiten.NewImage(w, h),
		Near:         0.1,
		Far:          100,

		Position:     UnitVector(0),
		LookAtTarget: UnitVector(0),
		// Draw:    &Drawcalls{},
	}

	cam.SetPerspectiveView(75)

	// cam.DepthTexture = ebiten.NewImage(buffer.Size())

	return cam
}

func (camera *Camera) UpdateTransform() {

	// T * R * S * O

	camera.Transform = LookAt(camera.LookAtTarget, camera.Position, vector.Y)

	// camera.Transform = NewMatrix4()
	// camera.Transform = camera.Transform.Dot(Scale(camera.Scale[0], camera.Scale[1], camera.Scale[2]))
	// camera.Transform = camera.Transform.Dot(Translate(camera.Position[0], camera.Position[1], camera.Position[2]))

}

func (camera *Camera) SetPerspectiveView(fovy float64) {
	ymax := camera.Near * math.Tan((fovy*math.Pi)/360)
	width, height := camera.ColorTexture.Size()
	aspect := float64(width) / float64(height)
	xmax := ymax * aspect
	camera.Projection = frustum(-xmax, xmax, -ymax, ymax, camera.Near, camera.Far)
}

func (camera *Camera) ProjectToScreen(vert vector.Vector) vector.Vector {

	w, h := camera.ColorTexture.Size()
	width, height := float64(w), float64(h)

	vd := (-vert[3]) / camera.Far
	vx := vert[0] * (1 - vd)
	vy := vert[1] * (1 - vd)

	fmt.Println(vert)

	vect := vector.Vector{
		vx*width + (width / 2),
		vy*height + (height / 2),
	}

	return vect

}

// func (camera *Camera) ProjectToScreen(vert vector.Vector) vector.Vector {

// 	w, h := camera.ColorTexture.Size()
// 	width, height := float64(w), float64(h)

// 	// vd := (-vert[2] * 100) / camera.Far
// 	vd := (-vert[2] * 100) / camera.Far
// 	vx := vert[0] * (1 - vd)
// 	vy := vert[1] * (1 - vd)

// 	vect := vector.Vector{
// 		vx*width + (width / 2),
// 		vy*height + (height / 2),
// 	}

// 	return vect

// }

func (camera *Camera) Begin() {
	camera.UpdateTransform()
	camera.ColorTexture.Clear()
}

func (camera *Camera) Render(model *Model) {

	drawOptions := &ebiten.DrawTrianglesOptions{}

	// projection * view * model

	model.UpdateTransform()

	mvpMatrix := camera.Projection.Mult(camera.Transform).Mult(model.Transform)

	verts := model.TransformedVertices(mvpMatrix)

	for index, vert := range verts {
		projected := camera.ProjectToScreen(vert)
		vertexList[index].DstX = float32(projected[0])
		vertexList[index].DstY = float32(projected[1])
	}

	i := 0

	lookDir := camera.LookVector()
	drawnTriangleCount := 0

	for _, tri := range model.Mesh.Triangles {

		// Backface Culling
		if model.Mesh.BackfaceCulling {

			normal := calculateNormal(
				verts[tri.Indices[0]][:3],
				verts[tri.Indices[1]][:3],
				verts[tri.Indices[2]][:3],
			)

			result := lookDir.Dot(normal)

			if result > 0 {
				continue
			}

		}

		for _, index := range tri.Indices {
			// vertexList[index].ColorR = float32(tri.Normal.X()) * 10
			// vertexList[index].ColorG = float32(tri.Normal.Y()) * 10
			// vertexList[index].ColorB = float32(tri.Normal.Z()) * 10
			indexList[i] = index
			i++
		}

		drawnTriangleCount++

	}

	indices := indexList[:drawnTriangleCount*3]
	// sort.Slice(indices, func(i, j int) bool {
	// 	return indices[i]
	// })

	camera.ColorTexture.DrawTriangles(vertexList[:len(model.Mesh.Vertices)], indices, defaultImg, drawOptions)

}

func (camera *Camera) LookVector() vector.Vector {
	return camera.Position.Sub(camera.LookAtTarget).Unit()
}

// Cribbed from https://github.com/fogleman/fauxgl vvvvvv

func frustum(l, r, b, t, n, f float64) Matrix4 {
	t1 := 2 * n
	t2 := r - l
	t3 := t - b
	t4 := f - n
	return Matrix4{
		{t1 / t2, 0, (r + l) / t2, 0},
		{0, t1 / t3, (t + b) / t3, 0},
		{0, 0, (-(f + n)) / t4, (-t1 * f) / t4},
		{0, 0, -1, 0},
	}
}

// func (camera *Camera) SetOrthographicView() {

// 	w, h := camera.ColorTexture.Size()

// 	width := float64(w)
// 	height := float64(h)

// 	l, t := -width/2, -height/2
// 	r, b := width/2, height/2
// 	n, f := camera.Near, camera.Far

// 	camera.Projection = Matrix4{
// 		{2 / (r - l), 0, 0, -(r + l) / (r - l)},
// 		{0, 2 / (t - b), 0, -(t + b) / (t - b)},
// 		{0, 0, -2 / (f - n), -(f + n) / (f - n)},
// 		{0, 0, 0, 1}}
// }

//Cribbed from https://github.com/fogleman/fauxgl ^^^^^^^
