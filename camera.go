package jank

import (
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/kvartborg/vector"
)

type Camera struct {
	ColorTexture *ebiten.Image
	Near, Far    float64
	Perspective  bool
	fieldOfView  float64

	Position vector.Vector
	Rotation Quaternion // Quaternion, essentially

	DebugDrawCount int
	Wireframe      bool
}

func NewCamera(w, h int) *Camera {

	cam := &Camera{
		ColorTexture: ebiten.NewImage(w, h),
		Near:         0.1,
		Far:          100,

		Position: UnitVector(0),
		Rotation: NewQuaternion(vector.Vector{0, 1, 0}, 0),
		// Draw:    &Drawcalls{},
	}

	cam.SetPerspective(45)

	// cam.LookAt(cam.Position.Add(vector.Vector{0, 0, -1}), vector.Y)

	// cam.DepthTexture = ebiten.NewImage(buffer.Size())

	return cam
}

func (camera *Camera) ViewMatrix() Matrix4 {

	transform := Translate(camera.Position[0], -camera.Position[1], camera.Position[2])
	transform = transform.Mult(Rotate(camera.Rotation.Axis, camera.Rotation.Angle))
	transform = transform.Mult(Perspective(camera.fieldOfView, camera.Near, camera.Far, float64(camera.ColorTexture.Bounds().Dx()), float64(camera.ColorTexture.Bounds().Dy())))

	return transform

}

// TODO: Implement some kind of LookAt functionality.
// func (camera *Camera) LookAt(target, up vector.Vector) { }

// TODO: Implement orthographic perspective.
// func (camera *Camera) SetOrthographic() { }

func (camera *Camera) SetPerspective(fovY float64) {
	camera.fieldOfView = fovY
	camera.Perspective = true
}

// viewPointToScreen projects the pre-transformed vertex in View space and remaps it to screen coordinates.
func (camera *Camera) viewPointToScreen(vert vector.Vector) vector.Vector {

	w, h := camera.ColorTexture.Size()
	width, height := float64(w), float64(h)

	// Again, this function should only be called with pre-transformed 4D vertex arguments.
	vx := vert[0] / vert[3]
	vy := vert[1] / vert[3]

	vect := vector.Vector{
		vx*width + (width / 2),
		vy*height + (height / 2),
	}

	return vect

}

// WorldToScreen transforms a 3D position in the world to screen coordinates.
func (camera *Camera) WorldToScreen(vert vector.Vector) vector.Vector {
	// The negating of these values are the same as the negating in Model.Transform() and the opposite of Camera.ViewMatrix().
	v := Translate(-vert[0], vert[1], -vert[2]).Mult(camera.ViewMatrix())
	return camera.viewPointToScreen(v.MultVecW(vector.Vector{0, 0, 0}))
}

// worldToView transforms a 3D position in the world to view coordinates (i.e. this type of coordinate needs to be mapped to the screen still).
func (camera *Camera) worldToView(vert vector.Vector) vector.Vector {
	// The negating of these values are the same as the negating in Model.Transform() and the opposite of Camera.ViewMatrix().
	v := Translate(-vert[0], vert[1], -vert[2]).Mult(camera.ViewMatrix())
	return v.MultVecW(vector.Vector{0, 0, 0})
}

// Clear is called at the beginning of a single rendered frame; it clears the backing textures before rendering.
func (camera *Camera) Clear() {
	camera.ColorTexture.Clear()
	camera.DebugDrawCount = 0
}

func (camera *Camera) pointInsideScreen(vert vector.Vector) bool {

	w, h := camera.ColorTexture.Size()
	width := float64(w)
	height := float64(h)
	return vert[0] >= 0 && vert[0] <= width && vert[1] >= 0 && vert[1] <= height

}

// Render renders the models passed - note that models rendered one after another in multiple Render() calls will be rendered on top of each other.
// Models need to be passed into the same Render() call to be sorted in depth.
func (camera *Camera) Render(models ...*Model) {

	sort.SliceStable(models, func(i, j int) bool {
		return models[i].Position.Sub(camera.Position).Magnitude() > models[j].Position.Sub(camera.Position).Magnitude()
	})

	viewMatrix := camera.ViewMatrix()

	for _, model := range models {

		if !model.Visible {
			continue
		}

		// TODO: Enhance this with a comparison, not of the (usually central) position directly, but of the extant of the bounding box / sphere of the model.
		if model.FrustumCull {

			viewPos := camera.worldToView(model.Position)
			screenPos := camera.WorldToScreen(model.Position)

			// Either the object lies outside of the screen (screenPos[0] < 0, as an example) or behind it (viewPos[3] > 0)
			if !camera.pointInsideScreen(screenPos) || viewPos[3] > 0 {
				continue
			}

		}

		verts := model.TransformedVertices(viewMatrix, camera.Position)

		vertexListIndex := 0

		modelTransform := model.Transform()

		for index := 0; index < len(verts); index += 3 {

			v0 := verts[index]
			v1 := verts[index+1]
			v2 := verts[index+2]

			// // Skip if all vertices are behind the camera; this works well, but it's faster to avoid rendering objects if the center is behind the camera, as above
			// if v0[3] > 0 && v1[3] > 0 && v2[3] > 0 {
			// 	continue
			// }

			// Backface Culling

			tri := model.closestTris[index/3]

			// TODO: Mis-behaves when viewing objects from behind at certain distances
			if model.Mesh.BackfaceCulling {

				normal := modelTransform.MultVec(tri.Normal)

				// dot := modelTransform.MultVec(camera.Position).Sub(modelTransform.MultVec(tri.Vertices[0].Position)).Dot(normal)

				eye := tri.Vertices[0].Position.Sub(camera.Position).Unit()

				dot := eye.Dot(normal)

				if dot < 0 {
					continue
				}

			}

			p0 := camera.viewPointToScreen(v0)
			p1 := camera.viewPointToScreen(v1)
			p2 := camera.viewPointToScreen(v2)

			// Skip if the vertex is wholly outside of the screen - This is one approach to do this that is relatively safe and simple, but it's more performant to do this as early as
			// possible (i.e. before transforming vertex transformation), rather than later.
			// if !camera.pointInsideScreen(p0) && !camera.pointInsideScreen(p1) && !camera.pointInsideScreen(p2) {
			// 	continue
			// }

			triList[vertexListIndex/3] = tri

			vertexList[vertexListIndex].DstX = float32(int(p0[0]))
			vertexList[vertexListIndex].DstY = float32(int(p0[1]))

			vertexList[vertexListIndex+1].DstX = float32(int(p1[0]))
			vertexList[vertexListIndex+1].DstY = float32(int(p1[1]))

			vertexList[vertexListIndex+2].DstX = float32(int(p2[0]))
			vertexList[vertexListIndex+2].DstY = float32(int(p2[1]))

			vertexListIndex += 3

		}

		if vertexListIndex == 0 {
			continue
		}

		srcW := 0.0
		srcH := 0.0

		if model.Mesh.Image != nil {
			srcW = float64(model.Mesh.Image.Bounds().Dx())
			srcH = float64(model.Mesh.Image.Bounds().Dy())
		}

		index := 0

		for _, tri := range triList[:vertexListIndex/3] {

			for _, vert := range tri.Vertices {

				// Vertex colors

				vertexList[index].ColorR = vert.Color.R
				vertexList[index].ColorG = vert.Color.G
				vertexList[index].ColorB = vert.Color.B
				vertexList[index].ColorA = vert.Color.A

				vertexList[index].SrcX = float32(vert.UV[0] * srcW)

				// We do 1 - v here (aka Y in texture coordinates) because 1.0 is the top of the texture while 0 is the bottom in UV coordinates,
				// but when drawing textures 0 is the top, and the sourceHeight is the bottom.
				vertexList[index].SrcY = float32((1 - vert.UV[1]) * srcH)

				indexList[index] = uint16(index)

				index++

			}

		}

		img := model.Mesh.Image
		if img == nil {
			img = defaultImg
		}

		if camera.Wireframe {

			for i := 0; i < vertexListIndex-2; i++ {
				v1 := vertexList[i]
				v2 := vertexList[i+1]
				ebitenutil.DrawLine(camera.ColorTexture, float64(v1.DstX), float64(v1.DstY), float64(v2.DstX), float64(v2.DstY), color.RGBA{uint8(v1.ColorR * 255), uint8(v1.ColorG * 255), uint8(v1.ColorB * 255), uint8(v1.ColorA * 255)})
			}

		} else {

			t := &ebiten.DrawTrianglesOptions{}
			t.ColorM.Scale(model.Color.RGBA64())
			t.Filter = model.Mesh.FilterMode
			camera.ColorTexture.DrawTriangles(vertexList[:vertexListIndex], indexList[:vertexListIndex], img, t)

		}

		camera.DebugDrawCount++

	}

}
