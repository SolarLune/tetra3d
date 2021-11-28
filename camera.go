package tetra3d

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
	"golang.org/x/image/font/basicfont"
)

// DebugInfo is a struct that holds debugging information for a Camera's render pass. These values are reset when Camera.Clear() is called.
type DebugInfo struct {
	AvgFrameTime     time.Duration // Amount of CPU frame time spent transforming vertices. Doesn't necessarily include CPU time spent sending data to the GPU.
	AvgAnimationTime time.Duration // Amount of CPU frame time spent animating vertices.
	animationTime    time.Duration
	frameTime        time.Duration
	tickTime         time.Time
	DrawnObjects     int // Number of rendered objects, excluding those invisible or culled based on distance
	TotalObjects     int // Total number of objects
	DrawnTris        int // Number of drawn triangles, excluding those hidden from backface culling
	TotalTris        int // Total number of triangles
}

// Camera represents a camera (where you look from) in Tetra3D.
type Camera struct {
	*Node
	ColorTexture      *ebiten.Image
	DepthTexture      *ebiten.Image
	ColorIntermediate *ebiten.Image
	DepthIntermediate *ebiten.Image

	RenderDepth bool // If the Camera should attempt to render a depth texture; if this is true, then DepthTexture will hold the depth texture render results.

	DepthShader *ebiten.Shader
	ColorShader *ebiten.Shader
	Near, Far   float64
	Perspective bool
	FieldOfView float64 // Vertical field of view in degrees for a perspective projection camera
	OrthoScale  float64 // Scale of the view for an orthographic projection camera in units horizontally

	DebugInfo          DebugInfo
	DebugDrawWireframe bool
	DebugDrawNormals   bool
	DebugDrawNodes     bool
	// DebugDrawBoundingSphere bool

	FrustumSphere *BoundingSphere
}

// NewCamera creates a new Camera with the specified width and height.
func NewCamera(w, h int) *Camera {

	cam := &Camera{
		Node:        NewNode("Camera"),
		RenderDepth: true,
		Near:        0.1,
		Far:         100,

		FrustumSphere: NewBoundingSphere(nil, vector.Vector{0, 0, 0}, 0),
	}

	depthShaderText := []byte(
		`package main

		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			depthValue := imageSrc0At(position.xy / imageSrcTextureSize())

			if depthValue.a == 0 || depthValue.r >= color.r {
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

		var Fog vec4
		var FogRange [2]float

		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			// texSize := imageSrcTextureSize()

			depth := imageSrc1At(texCoord)
			
			if depth.a > 0 {
				color := imageSrc0At(texCoord)
				
				d := smoothstep(FogRange[0], FogRange[1], depth.r)

				if Fog.a == 1 {
					color.rgb += Fog.rgb * d
				} else if Fog.a == 2 {
					color.rgb -= Fog.rgb * d
				} else if Fog.a == 3 {
					color.rgb = mix(color.rgb, Fog.rgb, d)
				}

				return color
			}

		}

		`,
	)

	cam.ColorShader, err = ebiten.NewShader(colorShaderText)

	if err != nil {
		panic(err)
	}

	if w != 0 && h != 0 {
		cam.Resize(w, h)
	}

	cam.SetPerspective(60)

	return cam
}

func (camera *Camera) Resize(w, h int) {

	if camera.ColorTexture != nil {
		camera.ColorTexture.Dispose()
		camera.DepthTexture.Dispose()
		camera.ColorIntermediate.Dispose()
		camera.DepthIntermediate.Dispose()
	}

	camera.ColorTexture = ebiten.NewImage(w, h)
	camera.DepthTexture = ebiten.NewImage(w, h)
	camera.ColorIntermediate = ebiten.NewImage(w, h)
	camera.DepthIntermediate = ebiten.NewImage(w, h)

}

// ViewMatrix returns the Camera's view matrix.
func (camera *Camera) ViewMatrix() Matrix4 {

	transform := NewMatrix4Translate(-camera.WorldPosition()[0], -camera.WorldPosition()[1], -camera.WorldPosition()[2])

	// We invert the rotation because the Camera is looking down -Z
	transform = transform.Mult(camera.WorldRotation().Transposed())

	return transform

}

// Projection returns the Camera's projection matrix.
func (camera *Camera) Projection() Matrix4 {
	if camera.Perspective {
		return NewProjectionPerspective(camera.FieldOfView, camera.Near, camera.Far, float64(camera.ColorTexture.Bounds().Dx()), float64(camera.ColorTexture.Bounds().Dy()))
	}
	w, h := camera.ColorTexture.Size()
	asr := float64(h) / float64(w)

	return NewProjectionOrthographic(camera.Near, camera.Far, 1*camera.OrthoScale, -1*camera.OrthoScale, asr*camera.OrthoScale, -asr*camera.OrthoScale)
	// return NewProjectionOrthographic(camera.Near, camera.Far, float64(camera.ColorTexture.Bounds().Dx())*camera.OrthoScale, float64(camera.ColorTexture.Bounds().Dy())*camera.OrthoScale)
}

// SetPerspective sets the Camera's projection to be a perspective projection. fovY indicates the vertical field of view (in degrees) for the camera's aperture.
func (camera *Camera) SetPerspective(fovY float64) {
	camera.FieldOfView = fovY
	camera.Perspective = true
}

// SetOrthographic sets the Camera's projection to be an orthographic projection. orthoScale indicates the scale of the camera in units horizontally.
func (camera *Camera) SetOrthographic(orthoScale float64) {
	camera.Perspective = false
	camera.OrthoScale = orthoScale
}

// ClipToScreen projects the pre-transformed vertex in View space and remaps it to screen coordinates.
func (camera *Camera) ClipToScreen(vert vector.Vector) vector.Vector {

	w, h := camera.ColorTexture.Size()
	width, height := float64(w), float64(h)

	v3 := vert[3]

	if !camera.Perspective {
		v3 = 1.0
	}

	// If the trangle is beyond the screen, we'll just pretend it's not and limit it to the closest possible value > 0
	// TODO: Replace this with triangle clipping or fix whatever graphical glitch seems to arise periodically
	if v3 < 0 {
		v3 = 0.000001
	}

	// Again, this function should only be called with pre-transformed 4D vertex arguments.
	vx := vert[0] / v3
	vy := vert[1] / v3 * -1

	vect := vector.Vector{
		vx*width + (width / 2),
		vy*height + (height / 2),
		vert[2],
		vert[3],
	}

	return vect

}

// WorldToScreen transforms a 3D position in the world to screen coordinates.
func (camera *Camera) WorldToScreen(vert vector.Vector) vector.Vector {
	v := NewMatrix4Translate(vert[0], vert[1], vert[2]).Mult(camera.ViewMatrix().Mult(camera.Projection()))
	return camera.ClipToScreen(v.MultVecW(vector.Vector{0, 0, 0}))
}

// WorldToClip transforms a 3D position in the world to clip coordinates (before screen normalization).
func (camera *Camera) WorldToClip(vert vector.Vector) vector.Vector {
	v := NewMatrix4Translate(vert[0], vert[1], vert[2]).Mult(camera.ViewMatrix().Mult(camera.Projection()))
	return v.MultVecW(vector.Vector{0, 0, 0})
}

// Clear should be called at the beginning of a single rendered frame. It clears the backing textures before rendering.
func (camera *Camera) Clear() {

	camera.ColorTexture.Clear()
	camera.DepthTexture.Clear()

	if camera.DebugInfo.tickTime.IsZero() || time.Since(camera.DebugInfo.tickTime).Milliseconds() >= 100 {
		camera.DebugInfo.tickTime = time.Now()
		camera.DebugInfo.AvgFrameTime = camera.DebugInfo.frameTime
		camera.DebugInfo.AvgAnimationTime = camera.DebugInfo.animationTime
	}
	camera.DebugInfo.frameTime = 0
	camera.DebugInfo.animationTime = 0
	camera.DebugInfo.DrawnObjects = 0
	camera.DebugInfo.TotalObjects = 0
	camera.DebugInfo.TotalTris = 0
	camera.DebugInfo.DrawnTris = 0

}

// RenderNodes renders all nodes starting with the provided rootNode using the Scene's properties (fog, for example).
func (camera *Camera) RenderNodes(scene *Scene, rootNode INode) {

	meshes := []*Model{}

	if model, isModel := rootNode.(*Model); isModel {
		meshes = append(meshes, model)
	}

	nodes := rootNode.ChildrenRecursive(true)

	for _, node := range nodes {
		if model, ok := node.(*Model); ok {
			meshes = append(meshes, model)
		}
	}

	camera.Render(scene, meshes...)

	if camera.DebugDrawNodes {

		for _, node := range rootNode.ChildrenRecursive(false) {

			if node == camera {
				continue
			}

			camera.drawCircle(node.WorldPosition(), 8, color.RGBA{0, 127, 255, 255})

			if node.Parent() != nil && node.Parent() != scene.Root {
				parentPos := camera.WorldToScreen(node.Parent().WorldPosition())
				pos := camera.WorldToScreen(node.WorldPosition())
				ebitenutil.DrawLine(camera.ColorTexture, pos[0], pos[1], parentPos[0], parentPos[1], color.RGBA{0, 192, 255, 255})
			}

		}

	}

}

// func (camera *Camera) RenderNodes(scene *Scene, nodes ...Node) {

// 	meshes := []*Model{}

// 	for _, node := range nodes {

// 		if model, isModel := node.(*Model); isModel {
// 			meshes = append(meshes, model)
// 		}

// 	}

// 	camera.Render(scene, meshes...)

// }

// Render renders all of the models passed using the provided Scene's properties (fog, for example). Note that if Camera.RenderDepth
// is false, scenes rendered one after another in multiple Render() calls will be rendered on top of each other in the Camera's texture buffers.
func (camera *Camera) Render(scene *Scene, models ...*Model) {

	// If the camera isn't rendering depth, then we should sort models by distance to ensure things draw in something of the correct order
	if !camera.RenderDepth {

		sort.SliceStable(models, func(i, j int) bool {
			return models[i].WorldPosition().Sub(camera.WorldPosition()).Magnitude() > models[j].LocalPosition().Sub(camera.WorldPosition()).Magnitude()
		})

	}

	// By multiplying the camera's position against the view matrix (which contains the negated camera position), we're left with just the rotation
	// matrix, which we feed into model.TransformedVertices() to draw vertices in order of distance.
	pureViewRot := camera.ViewMatrix().MultVec(camera.WorldPosition())
	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	// Update the camera's frustum sphere
	dist := camera.Far - camera.Near
	forward := camera.WorldRotation().Forward().Invert()
	camera.FrustumSphere.LocalPosition = camera.WorldPosition().Add(forward.Scale(dist / 2))
	camera.FrustumSphere.LocalRadius = dist / 2

	frametimeStart := time.Now()

	rectShaderOptions := &ebiten.DrawRectShaderOptions{}
	rectShaderOptions.Images[0] = camera.ColorIntermediate
	rectShaderOptions.Images[1] = camera.DepthIntermediate

	if scene != nil {

		rectShaderOptions.Uniforms = map[string]interface{}{
			"Fog":      scene.fogAsFloatSlice(),
			"FogRange": scene.FogRange,
		}

	} else {

		rectShaderOptions.Uniforms = map[string]interface{}{
			"Fog":      []float32{0, 0, 0, 0},
			"FogRange": []float32{0, 1},
		}

	}

	for _, model := range models {

		// Models without Meshes are essentially just "nodes" that just have a position. They aren't counted for rendering.
		if model.Mesh == nil {
			continue
		}

		camera.DebugInfo.TotalObjects++

		camera.DebugInfo.TotalTris += len(model.Mesh.Triangles)

		if !model.visible {
			continue
		}

		if model.FrustumCulling {

			// This is the simplest method of simple frustum culling where we simply test the distances between objects.

			modelPos, _, _ := model.Transform().Decompose()

			if modelPos.Sub(camera.FrustumSphere.Position()).Magnitude() > model.BoundingSphere.Radius()+camera.FrustumSphere.Radius() {
				continue
			}

		}

		tris := model.TransformedVertices(vpMatrix, pureViewRot, camera)

		vertexListIndex := 0

		for _, tri := range tris {

			v0 := tri.Vertices[0].transformed
			v1 := tri.Vertices[1].transformed
			v2 := tri.Vertices[2].transformed

			// Near-ish clipping (basically clip triangles that are wholly behind the camera)
			if v0[3] < 0 && v1[3] < 0 && v2[3] < 0 {
				continue
			}

			if v0[2] > camera.Far && v1[2] > camera.Far && v2[2] > camera.Far {
				continue
			}

			// Backface Culling

			// if model.BackfaceCulling {

			// 	// SHOUTOUTS TO MOD DB FOR POINTING ME IN THE RIGHT DIRECTION FOR THIS BECAUSE GOOD LORDT:
			// 	// https://moddb.fandom.com/wiki/Backface_culling#Polygons_in_object_space_are_transformed_into_world_space

			// 	// We use Vertex.transformed[:3] here because the fourth W component messes up normal calculation otherwise
			// 	normal := calculateNormal(tri.Vertices[0].transformed[:3], tri.Vertices[1].transformed[:3], tri.Vertices[2].transformed[:3])

			// 	dot := normal.Dot(tri.Vertices[0].transformed[:3])

			// 	// A little extra to make sure we draw walls if you're peeking around them with a higher FOV
			// 	if dot < -0.1 {
			// 		continue
			// 	}

			// }

			p0 := camera.ClipToScreen(v0)
			p1 := camera.ClipToScreen(v1)
			p2 := camera.ClipToScreen(v2)

			// This is a bit of a hacky way to do backface culling; it works, but it uses
			// the screen positions of the vertices to determine if the triangle should be culled.
			// In truth, it would be better to use the above approach, but that gives us visual
			// errors when faces are behind the camera unless we clip triangles. I don't really
			// feel like doing that right now, so here we are.

			if model.BackfaceCulling {

				n0 := p0[:3].Sub(p1[:3]).Unit()
				n1 := p1[:3].Sub(p2[:3]).Unit()
				nor, _ := n0.Cross(n1)

				if nor[2] > 0 {
					continue
				}

			}

			triList[vertexListIndex/3] = tri

			vertexList[vertexListIndex].DstX = float32(math.Round(p0[0]))
			vertexList[vertexListIndex].DstY = float32(math.Round(p0[1]))

			vertexList[vertexListIndex+1].DstX = float32(math.Round(p1[0]))
			vertexList[vertexListIndex+1].DstY = float32(math.Round(p1[1]))

			vertexList[vertexListIndex+2].DstX = float32(math.Round(p2[0]))
			vertexList[vertexListIndex+2].DstY = float32(math.Round(p2[1]))

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

			far := camera.Far
			if !camera.Perspective {
				far = 2.0
			}

			for _, tri := range triList[:vertexListIndex/3] {

				for _, vert := range tri.Vertices {

					depth := (math.Max(vert.transformed[2], 0) / far)

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

		if model.Mesh.Material != nil && model.Mesh.Material.Image != nil {
			srcW = float64(model.Mesh.Material.Image.Bounds().Dx())
			srcH = float64(model.Mesh.Material.Image.Bounds().Dy())
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

		var img *ebiten.Image

		if model.Mesh.Material != nil {
			img = model.Mesh.Material.Image
		}

		if img == nil {
			img = defaultImg
		}

		t := &ebiten.DrawTrianglesOptions{}
		t.ColorM.Scale(model.Color.RGBA64()) // Mix in the Model's color
		t.Filter = model.Mesh.FilterMode

		if camera.RenderDepth {

			camera.ColorIntermediate.Clear()

			camera.ColorIntermediate.DrawTriangles(vertexList[:vertexListIndex], indexList[:vertexListIndex], img, t)

			w, h := camera.ColorTexture.Size()

			camera.ColorTexture.DrawRectShader(w, h, camera.ColorShader, rectShaderOptions)

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

		// if camera.DebugDrawBoundingSphere {

		// 	sphere := model.Mesh.BoundingSphere

		// 	transformedCenter := camera.WorldToScreen(model.Position.Add(sphere.Position))
		// 	transformedRadius := camera.WorldToScreen(model.Position.Add(sphere.Position).Add(vector.Vector{model.BoundingSphereRadius(), 0, 0}))
		// 	radius := transformedRadius.Sub(transformedCenter).Magnitude()

		// 	stepCount := float64(32)

		// 	for i := 0; i < int(stepCount); i++ {

		// 		x := (math.Sin(math.Pi*2*float64(i)/stepCount) * radius)
		// 		y := (math.Cos(math.Pi*2*float64(i)/stepCount) * radius)

		// 		x2 := (math.Sin(math.Pi*2*float64(i+1)/stepCount) * radius)
		// 		y2 := (math.Cos(math.Pi*2*float64(i+1)/stepCount) * radius)

		// 		ebitenutil.DrawLine(camera.ColorTexture, transformedCenter[0]+x, transformedCenter[1]+y, transformedCenter[0]+x2, transformedCenter[1]+y2, color.White)

		// 	}

		// }

	}

	camera.DebugInfo.frameTime += time.Since(frametimeStart)

}

func (camera *Camera) drawCircle(position vector.Vector, radius float64, drawColor color.RGBA) {

	transformedCenter := camera.WorldToScreen(position)

	stepCount := float64(32)

	for i := 0; i < int(stepCount); i++ {

		x := (math.Sin(math.Pi*2*float64(i)/stepCount) * radius)
		y := (math.Cos(math.Pi*2*float64(i)/stepCount) * radius)

		x2 := (math.Sin(math.Pi*2*float64(i+1)/stepCount) * radius)
		y2 := (math.Cos(math.Pi*2*float64(i+1)/stepCount) * radius)

		ebitenutil.DrawLine(camera.ColorTexture, transformedCenter[0]+x, transformedCenter[1]+y, transformedCenter[0]+x2, transformedCenter[1]+y2, drawColor)
	}

}

// DrawDebugText draws render debug information (like number of drawn objects, number of drawn triangles, frame time, etc)
// at the top-left of the provided screen *ebiten.Image.
func (camera *Camera) DrawDebugText(screen *ebiten.Image, textScale float64) {
	dr := &ebiten.DrawImageOptions{}
	dr.GeoM.Translate(0, textScale*16)
	dr.GeoM.Scale(textScale, textScale)

	m := camera.DebugInfo.AvgFrameTime.Round(time.Microsecond).Microseconds()
	ft := fmt.Sprintf("%.2fms", float32(m)/1000)

	m = camera.DebugInfo.AvgAnimationTime.Round(time.Microsecond).Microseconds()
	at := fmt.Sprintf("%.2fms", float32(m)/1000)

	text.DrawWithOptions(screen, fmt.Sprintf(
		"FPS: %f\nFrame time: %s\nSkinned mesh animation time: %s\nRendered objects:%d/%d\nRendered triangles: %d/%d",
		ebiten.CurrentFPS(),
		ft,
		at,
		camera.DebugInfo.DrawnObjects,
		camera.DebugInfo.TotalObjects,
		camera.DebugInfo.DrawnTris,
		camera.DebugInfo.TotalTris),
		basicfont.Face7x13, dr)

}

func (camera *Camera) Clone() INode {

	w, h := camera.ColorTexture.Size()
	clone := NewCamera(w, h)

	clone.RenderDepth = camera.RenderDepth
	clone.Near = camera.Near
	clone.Far = camera.Far
	clone.Perspective = camera.Perspective
	clone.FieldOfView = camera.FieldOfView

	clone.Node = camera.Node.Clone().(*Node)
	for _, child := range camera.children {
		child.setParent(camera)
	}

	return clone

}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (camera *Camera) AddChildren(children ...INode) {
	camera.addChildren(camera, children...)
}

// Unparent unparents the Camera from its parent, removing it from the scenegraph.
func (camera *Camera) Unparent() {
	if camera.parent != nil {
		camera.parent.RemoveChildren(camera)
	}
}
