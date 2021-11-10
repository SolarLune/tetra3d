package jank3d

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/kvartborg/vector"
)

type DebugInfo struct {
	VertexTime   time.Duration
	RenderTime   time.Duration
	DrawnObjects int
	TotalObjects int
	DrawnTris    int
	TotalTris    int
}

type Camera struct {
	ColorTexture      *ebiten.Image
	DepthTexture      *ebiten.Image
	ColorIntermediate *ebiten.Image
	DepthIntermediate *ebiten.Image
	Texture           *ebiten.Image

	RenderDepth bool // If the Camera should attempt to render a depth texture; if this is true, then DepthTexture will hold the depth texture render results.

	DepthShader *ebiten.Shader
	ColorShader *ebiten.Shader
	Near, Far   float64
	Perspective bool
	fieldOfView float64

	Position vector.Vector
	Rotation Quaternion

	DebugInfo               DebugInfo
	DebugDrawWireframe      bool
	DebugDrawNormals        bool
	DebugDrawBoundingSphere bool
}

func NewCamera(w, h int) *Camera {

	cam := &Camera{
		ColorTexture:      ebiten.NewImage(w, h),
		DepthTexture:      ebiten.NewImage(w, h),
		ColorIntermediate: ebiten.NewImage(w, h),
		DepthIntermediate: ebiten.NewImage(w, h),
		RenderDepth:       true,
		Near:              0.1,
		Far:               100,

		Position: UnitVector(0),
		Rotation: NewQuaternion(vector.Vector{0, 1, 0}, 0),
	}

	depthShaderText := []byte(
		`package main

		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			depthValue := imageSrc0At(position.xy / imageSrcTextureSize())

			if depthValue.a == 0 || depthValue.r > color.r {
				return color
			}

			// return mix(
			// 	color, depthValue, step(depthValue.r, color.r) * depthValue.a,
			// )

		}

		`,
	)

	var err error

	cam.DepthShader, err = ebiten.NewShader(depthShaderText)

	if err != nil {
		panic(err)
	}

	colorShaderText := []byte(
		`package main

		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			// texSize := imageSrcTextureSize()

			intermediateValue := imageSrc1At(texCoord)
			
			if intermediateValue.a > 0 {
				return imageSrc0At(texCoord)
			}

		}

		`,
	)

	cam.ColorShader, err = ebiten.NewShader(colorShaderText)

	if err != nil {
		panic(err)
	}

	cam.SetPerspective(60)

	return cam
}

func (camera *Camera) ViewMatrix() Matrix4 {

	transform := Translate(-camera.Position[0], -camera.Position[1], -camera.Position[2])
	rotate := Rotate(camera.Rotation.Axis[0], camera.Rotation.Axis[1], camera.Rotation.Axis[2], camera.Rotation.Angle)
	// We invert the Z portion of the rotation because the Camera is looking down -Z
	rotate[2][0] *= -1
	rotate[2][1] *= -1
	rotate[2][2] *= -1
	transform = transform.Mult(rotate)

	return transform

}

func (camera *Camera) Projection() Matrix4 {
	return Perspective(camera.fieldOfView, camera.Near, camera.Far, float64(camera.ColorTexture.Bounds().Dx()), float64(camera.ColorTexture.Bounds().Dy()))
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

	v3 := vert[3]

	// If the trangle is beyond the screen, we'll just pretend it's not and limit it to the closest possible value > 0
	// TODO: Replace this with triangle clipping or fix whatever graphical glitch seems to arise periodically
	if v3 > 0 {
		v3 = -0.000001
	}

	// Again, this function should only be called with pre-transformed 4D vertex arguments.
	vx := vert[0] / v3 * -1
	vy := vert[1] / v3

	vect := vector.Vector{
		vx*width + (width / 2),
		vy*height + (height / 2),
	}

	return vect

}

// WorldToScreen transforms a 3D position in the world to screen coordinates.
func (camera *Camera) WorldToScreen(vert vector.Vector) vector.Vector {
	v := Translate(vert[0], vert[1], vert[2]).Mult(camera.ViewMatrix().Mult(camera.Projection()))
	return camera.viewPointToScreen(v.MultVecW(vector.Vector{0, 0, 0}))
}

// worldToView transforms a 3D position in the world to view coordinates (i.e. this type of coordinate needs to be mapped to the screen still).
func (camera *Camera) worldToView(vert vector.Vector) vector.Vector {
	v := Translate(vert[0], vert[1], vert[2]).Mult(camera.ViewMatrix().Mult(camera.Projection()))
	return v.MultVecW(vector.Vector{0, 0, 0})
}

// Clear is called at the beginning of a single rendered frame; it clears the backing textures before rendering.
func (camera *Camera) Clear() {
	camera.ColorTexture.Clear()
	camera.DepthTexture.Clear()
	camera.DebugInfo.VertexTime = 0
	camera.DebugInfo.DrawnObjects = 0
	camera.DebugInfo.TotalObjects = 0
	camera.DebugInfo.TotalTris = 0
	camera.DebugInfo.DrawnTris = 0
}

// Render renders the models passed - note that models rendered one after another in multiple Render() calls will be rendered on top of each other.
// Models need to be passed into the same Render() call to be sorted in depth.
func (camera *Camera) Render(models ...*Model) {

	if !camera.RenderDepth {

		sort.SliceStable(models, func(i, j int) bool {
			return models[i].Position.Sub(camera.Position).Magnitude() > models[j].Position.Sub(camera.Position).Magnitude()
		})

	}

	viewPos := camera.ViewMatrix().MultVec(camera.Position)
	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	frametimeStart := time.Now()

	camera.DebugInfo.TotalObjects += len(models)

	for _, model := range models {

		camera.DebugInfo.TotalTris += len(model.Mesh.Triangles)

		if !model.Visible {
			continue
		}

		if model.FrustumCulling {

			screenPos := camera.WorldToScreen(model.Position)
			capped := screenPos.Clone()
			r := camera.WorldToScreen(model.Position.Add(vector.Vector{model.BoundingSphereRadius() + model.Mesh.BoundingSphere.Position.Magnitude(), 0, 0}))
			radius := r.Sub(screenPos).Magnitude()

			w, h := camera.ColorTexture.Size()

			if capped[0] > float64(w) {
				capped[0] = float64(w)
			} else if capped[0] < 0 {
				capped[0] = 0
			}

			if capped[1] > float64(h) {
				capped[1] = float64(h)
			} else if capped[1] < 0 {
				capped[1] = 0
			}

			if capped.Sub(screenPos).Magnitude() > radius {
				continue
			}

		}

		model.TransformedVertices(vpMatrix, viewPos)

		vertexListIndex := 0

		// model.closestTris is set within TransformedVertices().
		for _, tri := range model.closestTris {

			v0 := tri.Vertices[0].transformed
			v1 := tri.Vertices[1].transformed
			v2 := tri.Vertices[2].transformed

			// Skip if all vertices are behind the camera; this fixes triangles drawing inverted when they move behind the camera, but it conversely makes it impossible
			// to draw triangles that are stretch on behind the camera (think the flooring and walls in a hallway that goes ahead of and behind the camera).
			// A workaround is to segment the hallway and not allow the camera to get overly close, but the only real solution is to TODO: get rid of this and
			// add triangle clipping so any triangles that are too long get clipped.
			if v0[3] > 0 && v1[3] > 0 && v2[3] > 0 {
				continue
			}

			// Backface Culling

			if model.BackfaceCulling {

				// SHOUTOUTS TO MOD DB FOR POINTING ME IN THE RIGHT DIRECTION FOR THIS BECAUSE GOOD LORDT:
				// https://moddb.fandom.com/wiki/Backface_culling#Polygons_in_object_space_are_transformed_into_world_space

				// We use Vertex.transformed[:3] here because the fourth W component messes up normal calculation otherwise
				normal := calculateNormal(tri.Vertices[0].transformed[:3], tri.Vertices[1].transformed[:3], tri.Vertices[2].transformed[:3])

				dot := normal.Dot(tri.Vertices[0].transformed[:3])

				if dot > 0 {
					continue
				}

			}

			p0 := camera.viewPointToScreen(v0)
			p1 := camera.viewPointToScreen(v1)
			p2 := camera.viewPointToScreen(v2)

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

		for i := 0; i < vertexListIndex; i++ {
			indexList[i] = uint16(i)
		}

		index := 0

		// Render the depth map here
		if camera.RenderDepth {

			for _, tri := range triList[:vertexListIndex/3] {

				for _, vert := range tri.Vertices {

					// Vertex colors

					depth := (math.Max(-vert.transformed[2]-camera.Near, 0) / camera.Far)
					vertexList[index].ColorR = float32(depth)
					vertexList[index].ColorG = float32(depth)
					vertexList[index].ColorB = float32(depth)
					vertexList[index].ColorA = 1
					index++

				}

			}

			shaderOpt := &ebiten.DrawTrianglesShaderOptions{
				Images: [4]*ebiten.Image{camera.DepthTexture},
			}
			camera.DepthIntermediate.Clear()
			camera.DepthIntermediate.DrawTrianglesShader(vertexList[:vertexListIndex], indexList[:vertexListIndex], camera.DepthShader, shaderOpt)
			camera.DepthTexture.DrawImage(camera.DepthIntermediate, nil)

		}

		srcW := 0.0
		srcH := 0.0

		if model.Mesh.Image != nil {
			srcW = float64(model.Mesh.Image.Bounds().Dx())
			srcH = float64(model.Mesh.Image.Bounds().Dy())
		}

		index = 0

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

				index++

			}

		}

		img := model.Mesh.Image
		if img == nil {
			img = defaultImg
		}

		t := &ebiten.DrawTrianglesOptions{}
		t.ColorM.Scale(model.Color.RGBA64())
		t.Filter = model.Mesh.FilterMode

		if camera.RenderDepth {

			camera.ColorIntermediate.Clear()

			camera.ColorIntermediate.DrawTriangles(vertexList[:vertexListIndex], indexList[:vertexListIndex], img, t)

			w, h := camera.ColorTexture.Size()
			opt := &ebiten.DrawRectShaderOptions{}
			opt.Images[0] = camera.ColorIntermediate
			opt.Images[1] = camera.DepthIntermediate

			camera.ColorTexture.DrawRectShader(w, h, camera.ColorShader, opt)

		} else {
			camera.ColorTexture.DrawTriangles(vertexList[:vertexListIndex], indexList[:vertexListIndex], img, t)
		}

		camera.DebugInfo.DrawnObjects++
		camera.DebugInfo.DrawnTris += vertexListIndex / 3

		if camera.DebugDrawWireframe {

			for i := 0; i < vertexListIndex; i += 3 {
				v1 := vertexList[i]
				v2 := vertexList[i+1]
				v3 := vertexList[i+2]
				ebitenutil.DrawLine(camera.ColorTexture, float64(v1.DstX), float64(v1.DstY), float64(v2.DstX), float64(v2.DstY), color.White)
				ebitenutil.DrawLine(camera.ColorTexture, float64(v2.DstX), float64(v2.DstY), float64(v3.DstX), float64(v3.DstY), color.White)
				ebitenutil.DrawLine(camera.ColorTexture, float64(v3.DstX), float64(v3.DstY), float64(v1.DstX), float64(v1.DstY), color.White)
			}

		}

		if camera.DebugDrawNormals {

			triIndex := 0
			for _, tri := range triList[:vertexListIndex/3] {
				center := camera.WorldToScreen(model.Transform().MultVecW(tri.Center))
				transformedNormal := camera.WorldToScreen(model.Transform().MultVecW(tri.Center.Add(tri.Normal.Scale(0.1))))
				ebitenutil.DrawLine(camera.ColorTexture, center[0], center[1], transformedNormal[0], transformedNormal[1], color.RGBA{60, 158, 255, 255})
				triIndex++
			}

		}

		if camera.DebugDrawBoundingSphere {

			sphere := model.Mesh.BoundingSphere

			transformedCenter := camera.WorldToScreen(model.Position.Add(sphere.Position))
			transformedRadius := camera.WorldToScreen(model.Position.Add(sphere.Position).Add(vector.Vector{model.BoundingSphereRadius(), 0, 0}))
			radius := transformedRadius.Sub(transformedCenter).Magnitude()

			stepCount := float64(32)

			for i := 0; i < int(stepCount); i++ {

				x := (math.Sin(math.Pi*2*float64(i)/stepCount) * radius)
				y := (math.Cos(math.Pi*2*float64(i)/stepCount) * radius)

				x2 := (math.Sin(math.Pi*2*float64(i+1)/stepCount) * radius)
				y2 := (math.Cos(math.Pi*2*float64(i+1)/stepCount) * radius)

				ebitenutil.DrawLine(camera.ColorTexture, transformedCenter[0]+x, transformedCenter[1]+y, transformedCenter[0]+x2, transformedCenter[1]+y2, color.White)

			}

		}

	}

	camera.DebugInfo.VertexTime += time.Since(frametimeStart)

}

func (camera *Camera) DrawDebugText(screen *ebiten.Image, textScale float64) {
	dr := &ebiten.DrawImageOptions{}
	dr.GeoM.Translate(0, textScale*16)
	dr.GeoM.Scale(textScale, textScale)
	text.DrawWithOptions(screen, fmt.Sprintf(
		"FPS: %f\nFrame time: %fms\nRendered Objects:%d/%d\nRendered Triangles: %d/%d",
		ebiten.CurrentFPS(),
		camera.DebugInfo.VertexTime.Seconds()*1000,
		camera.DebugInfo.DrawnObjects,
		camera.DebugInfo.TotalObjects,
		camera.DebugInfo.DrawnTris,
		camera.DebugInfo.TotalTris),
		debugFontFace, dr)

}

// func (camera *Camera) clipEdge(start, end vector.Vector) []vector.Vector {

// 	startIn := start[3] < 0
// 	endIn := end[3] < 0

// 	newStart := start
// 	newEnd := end

// 	if (startIn && !endIn) || (endIn && !startIn) {

// 		d0 := start[3]
// 		d1 := end[3]
// 		factor := 1.0 / (d0 - d1)

// 		// factor*(d1 * v0.p - d0 * v1.p)
// 		v := start.Sub(end).Scale(factor).Scale(factor)
// 		fmt.Println(v)

// 		// v[3] *= -1

// 		if startIn {
// 			newEnd = v
// 		} else {
// 			newStart = v
// 		}

// 		fmt.Println("old start, end", start, end)
// 		if startIn {
// 			fmt.Println("new end", newEnd)
// 		} else {
// 			fmt.Println("new start", newStart)
// 		}

// 	}

// 	return []vector.Vector{newStart, newEnd}

// }

// func (camera *Camera) clipTriangle(verts ...vector.Vector) []vector.Vector {

// 	out := []vector.Vector{}

// 	if verts[0][3] > 0 && verts[1][3] > 0 && verts[2][3] > 0 {
// 		return out
// 	} else if verts[0][3] < 0 && verts[1][3] < 0 && verts[2][3] < 0 {
// 		return verts
// 	}

// 	newVerts := append([]vector.Vector{}, camera.clipEdge(verts[0], verts[1])...)
// 	newVerts = append(newVerts, camera.clipEdge(verts[1], verts[2])...)
// 	newVerts = append(newVerts, camera.clipEdge(verts[2], verts[0])...)

// 	added := func(vert vector.Vector) bool {
// 		for _, v := range out {
// 			if vert[0] == v[0] && vert[1] == v[1] && vert[2] == v[2] && vert[3] == v[3] {
// 				return true
// 			}
// 		}
// 		return false
// 	}

// 	for _, v := range newVerts {
// 		if !added(v) {
// 			out = append(out, v)
// 		}
// 	}

// 	fmt.Println("clipped: ", out)

// 	// for i := 0; i < len(verts); i++ {

// 	// 	vert := verts[i]
// 	// 	var prevVertex vector.Vector
// 	// 	if i == 0 {
// 	// 		prevVertex = verts[len(verts)-1]
// 	// 	} else {
// 	// 		prevVertex = verts[i-1]
// 	// 	}

// 	// 	if clip.PointInFront(vert) {
// 	// 		fmt.Println("intersection")
// 	// 		out = append(out, clip.Intersection(prevVertex, vert))
// 	// 	} else {
// 	// 		out = append(out, verts[i])
// 	// 	}

// 	// }

// 	fmt.Println(out)

// 	return out

// }
