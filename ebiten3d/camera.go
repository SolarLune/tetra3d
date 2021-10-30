package ebiten3d

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kvartborg/vector"
)

type Camera struct {
	ColorTexture *ebiten.Image
	Near, Far    float64

	Position vector.Vector
	lookDir  vector.Vector

	EyeMatrix  Matrix4
	Transform  Matrix4
	Projection Matrix4
	// Drawcalls   []Drawcall
}

func NewCamera(w, h int) *Camera {

	cam := &Camera{
		ColorTexture: ebiten.NewImage(w, h),
		Near:         0.1,
		Far:          100,

		Position:  UnitVector(0),
		EyeMatrix: NewMatrix4(),
		Transform: NewMatrix4(),
		// Draw:    &Drawcalls{},
	}

	cam.LookAt(cam.Position.Add(vector.Vector{0, 0, 1}), vector.Y)

	cam.SetPerspectiveView(75)

	// cam.DepthTexture = ebiten.NewImage(buffer.Size())

	return cam
}

func (camera *Camera) UpdateTransform() {

	camera.Transform = NewMatrix4()
	camera.Transform = camera.Transform.Mult(Translate(-camera.Position[0], -camera.Position[1], camera.Position[2]))
	// camera.Transform = camera.Transform.Mult(camera.EyeMatrix)

}

// LookAt sets the Camera's rotation such that it is looking at the target 3D Vector in world space.
func (camera *Camera) LookAt(target, up vector.Vector) {
	camera.lookDir = target.Sub(camera.Position).Unit()
	camera.EyeMatrix = LookAt(target, camera.Position, up)
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

	vx := (vert[0] / vert[3])
	vy := ((1 - vert[1]) / vert[3])

	vect := vector.Vector{
		vx*width + (width / 2),
		vy*height + (height / 2),
	}

	return vect

}

func (camera *Camera) Begin() {
	camera.UpdateTransform()
	camera.ColorTexture.Clear()
}

func (camera *Camera) Render(model *Model) {

	// projection * view * model

	model.UpdateTransform()

	mvpMatrix := camera.Projection.Mult(camera.EyeMatrix).Mult(camera.Transform).Mult(model.Transform)

	verts := model.TransformedVertices(mvpMatrix, camera)

	vertexListIndex := 0

	tris := []*Triangle{}

	for index := 0; index < len(verts); index += 3 {

		// If any vertex is behind the camera, then skip it
		for i := index; i < index+3; i++ {

			if verts[i][0] < 0 || verts[i][1] < 0 || verts[i][2] < 0 {
				continue
			}

		}

		v0 := verts[index]
		v1 := verts[index+1]
		v2 := verts[index+2]

		// Backface Culling

		if model.Mesh.BackfaceCulling {

			normal := calculateNormal(
				v0[:3],
				v1[:3],
				v2[:3],
			)

			// result := camera.Transform.Forward().Dot(normal)

			center := v0.Add(v1).Add(v2).Scale(1.0 / 3.0)
			result := camera.Position.Sub(center).Unit().Dot(normal)

			if result > 0 {
				continue
			}

		}

		tris = append(tris, model.closestTris[index/3])

		projected := camera.ProjectToScreen(v0)
		vertexList[vertexListIndex].DstX = float32(int(projected[0]))
		vertexList[vertexListIndex].DstY = float32(int(projected[1]))

		projected = camera.ProjectToScreen(v1)
		vertexList[vertexListIndex+1].DstX = float32(int(projected[0]))
		vertexList[vertexListIndex+1].DstY = float32(int(projected[1]))

		projected = camera.ProjectToScreen(v2)
		vertexList[vertexListIndex+2].DstX = float32(int(projected[0]))
		vertexList[vertexListIndex+2].DstY = float32(int(projected[1]))

		vertexListIndex += 3
	}

	srcW := 0.0
	srcH := 0.0

	if model.Mesh.Image != nil {
		srcW = float64(model.Mesh.Image.Bounds().Dx())
		srcH = float64(model.Mesh.Image.Bounds().Dy())
	}

	index := 0

	for _, tri := range tris {

		for _, vert := range tri.Vertices {

			// Vertex colors

			vertexList[index].ColorR = float32(vert.Color[0])
			vertexList[index].ColorG = float32(vert.Color[1])
			vertexList[index].ColorB = float32(vert.Color[2])
			vertexList[index].ColorA = float32(vert.Color[3])

			vertexList[index].SrcX = float32(vert.UV[0] * srcW)
			vertexList[index].SrcY = float32(vert.UV[1] * srcH)

			indexList[index] = uint16(index)

			index++

		}

	}

	// }

	img := model.Mesh.Image
	if img == nil {
		img = defaultImg
	}

	camera.ColorTexture.DrawTriangles(vertexList[:vertexListIndex], indexList[:vertexListIndex], img, nil)

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
