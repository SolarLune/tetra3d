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
	AvgLightTime     time.Duration // Amount of CPU frame time spent lighting vertices.
	animationTime    time.Duration
	lightTime        time.Duration
	frameTime        time.Duration
	frameCount       int
	tickTime         time.Time
	DrawnParts       int // Number of draw calls, excluding those invisible or culled based on distance
	TotalParts       int // Total number of objects
	DrawnTris        int // Number of drawn triangles, excluding those hidden from backface culling
	TotalTris        int // Total number of triangles
	LightCount       int // Total number of lights
	ActiveLightCount int // Total active number of lights
}

const (
	AccumlateColorModeNone            = iota // No accumulation buffer rendering
	AccumlateColorModeBelow                  // Accumulation buffer is on and applies over time, renders ColorTexture after the accumulation result (which is, then, below)
	AccumlateColorModeAbove                  // Accumulation buffer is on and applies over time, renders ColorTexture before the accumulation result (which is on top)
	AccumlateColorModeSingleLastFrame        // Accumulation buffer is on and renders just the previous frame's ColorTexture result
)

// Camera represents a camera (where you look from) in Tetra3D.
type Camera struct {
	*Node

	RenderDepth bool // If the Camera should attempt to render a depth texture; if this is true, then DepthTexture will hold the depth texture render results.

	resultColorTexture    *ebiten.Image // ColorTexture holds the color results of rendering any models.
	resultDepthTexture    *ebiten.Image // DepthTexture holds the depth results of rendering any models, if Camera.RenderDepth is on.
	colorIntermediate     *ebiten.Image
	depthIntermediate     *ebiten.Image
	clipAlphaIntermediate *ebiten.Image
	clipBehind            *ebiten.Image

	resultAccumulatedColorTexture *ebiten.Image // ResultAccumulatedColorTexture holds the previous frame's render result of rendering any models.
	accumulatedBackBuffer         *ebiten.Image
	AccumulateColorMode           int                      // The mode to use when rendering previous frames to the accumulation buffer. Defaults to AccumulateColorModeNone.
	AccumulateDrawOptions         *ebiten.DrawImageOptions // Draw image options to use when rendering frames to the accumulation buffer; use this to fade out or color previous frames.

	Near, Far   float64 // The near and far clipping plane.
	Perspective bool    // If the Camera has a perspective projection. If not, it would be orthographic
	FieldOfView float64 // Vertical field of view in degrees for a perspective projection camera
	OrthoScale  float64 // Scale of the view for an orthographic projection camera in units horizontally

	DebugInfo DebugInfo

	backfacePool             *VectorPool
	depthShader              *ebiten.Shader
	clipAlphaCompositeShader *ebiten.Shader
	clipAlphaRenderShader    *ebiten.Shader
	colorShader              *ebiten.Shader

	// Visibility check variables
	cameraForward          vector.Vector
	cameraRight            vector.Vector
	cameraUp               vector.Vector
	sphereFactorY          float64
	sphereFactorX          float64
	sphereFactorTang       float64
	sphereFactorCalculated bool
}

// NewCamera creates a new Camera with the specified width and height.
func NewCamera(w, h int) *Camera {

	cam := &Camera{
		Node:        NewNode("Camera"),
		RenderDepth: true,
		Near:        0.1,
		Far:         100,

		backfacePool:          NewVectorPool(3),
		AccumulateDrawOptions: &ebiten.DrawImageOptions{},
	}

	depthShaderText := []byte(
		`package main

		func encodeDepth(depth float) vec4 {
			r := floor(depth * 255) / 255
			g := floor(fract(depth * 255) * 255) / 255
			b := fract(depth * 255*255)
			return vec4(r, g, b, 1);
		}

		func decodeDepth(rgba vec4) float {
			return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
		}

		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			existingDepth := imageSrc0At(position.xy / imageSrcTextureSize())

			if existingDepth.a == 0 || decodeDepth(existingDepth) > color.r {
				return encodeDepth(color.r)
			}

			return vec4(0.0, 0.0, 0.0, 0.0)

		}

		`,
	)

	var err error

	cam.depthShader, err = ebiten.NewShader(depthShaderText)

	if err != nil {
		panic(err)
	}

	clipAlphaShaderText := []byte(
		`package main

		func encodeDepth(depth float) vec4 {
			r := floor(depth * 255) / 255
			g := floor(fract(depth * 255) * 255) / 255
			b := fract(depth * 255*255)
			return vec4(r, g, b, 1);
		}

		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
			tex := imageSrc0At(texCoord)
			if (tex.a == 0) {
				return vec4(0.0, 0.0, 0.0, 0.0)
			} else {
				return vec4(encodeDepth(color.r).rgb, tex.a)
			}
			// TODO: This shader needs to discard if tex.a is transparent. We can't sample the texture to return 
			// what's underneath here, so discard is basically necessary. We need to implement it once the dicard
			// keyword / function is implemented (if it ever is; hopefully it will be).
		}

		`,
	)

	cam.clipAlphaRenderShader, err = ebiten.NewShader(clipAlphaShaderText)

	if err != nil {
		panic(err)
	}

	clipCompositeShaderText := []byte(
		`package main

		func decodeDepth(rgba vec4) float {
			return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
		}

		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			depthValue := imageSrc0At(texCoord)
			texture := imageSrc1At(texCoord)

			if depthValue.a == 0 || decodeDepth(depthValue) > texture.r {
				return texture
			}

			return vec4(0.0, 0.0, 0.0, 0.0)

		}

		`,
	)

	cam.clipAlphaCompositeShader, err = ebiten.NewShader(clipCompositeShaderText)

	if err != nil {
		panic(err)
	}

	colorShaderText := []byte(
		`package main

		var Fog vec4
		var FogRange [2]float

		func decodeDepth(rgba vec4) float {
			return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
		}
		
		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			depth := imageSrc1At(texCoord)
			
			if depth.a > 0 {
				colorTex := imageSrc0At(texCoord)
				
				d := smoothstep(FogRange[0], FogRange[1], decodeDepth(depth))

				if Fog.a == 1 {
					colorTex.rgb += Fog.rgb * d * colorTex.a
				} else if Fog.a == 2 {
					colorTex.rgb -= Fog.rgb * d * colorTex.a
				} else if Fog.a == 3 {
					colorTex.rgb = mix(colorTex.rgb, Fog.rgb, d) * colorTex.a
				}

				return colorTex
			}

			// This should be discard as well, rather than alpha 0
			return vec4(0.0, 0.0, 0.0, 0.0)

		}

		`,
	)

	cam.colorShader, err = ebiten.NewShader(colorShaderText)

	if err != nil {
		panic(err)
	}

	if w != 0 && h != 0 {
		cam.Resize(w, h)
	}

	cam.SetPerspective(60)

	return cam
}

func (camera *Camera) Clone() INode {

	w, h := camera.resultColorTexture.Size()
	clone := NewCamera(w, h)

	clone.RenderDepth = camera.RenderDepth
	clone.Near = camera.Near
	clone.Far = camera.Far
	clone.Perspective = camera.Perspective
	clone.FieldOfView = camera.FieldOfView

	clone.AccumulateColorMode = camera.AccumulateColorMode
	clone.AccumulateDrawOptions = camera.AccumulateDrawOptions

	clone.Node = camera.Node.Clone().(*Node)
	for _, child := range camera.children {
		child.setParent(camera)
	}

	return clone

}

func (camera *Camera) Resize(w, h int) {

	if camera.resultColorTexture != nil {

		origW, origH := camera.resultColorTexture.Size()
		if w == origW && h == origH {
			return
		}

		camera.resultColorTexture.Dispose()
		camera.resultAccumulatedColorTexture.Dispose()
		camera.accumulatedBackBuffer.Dispose()
		camera.resultDepthTexture.Dispose()
		camera.colorIntermediate.Dispose()
		camera.depthIntermediate.Dispose()
		camera.clipAlphaIntermediate.Dispose()
		camera.clipBehind.Dispose()
	}

	camera.resultAccumulatedColorTexture = ebiten.NewImage(w, h)
	camera.accumulatedBackBuffer = ebiten.NewImage(w, h)
	camera.resultColorTexture = ebiten.NewImage(w, h)
	camera.resultDepthTexture = ebiten.NewImage(w, h)
	camera.colorIntermediate = ebiten.NewImage(w, h)
	camera.depthIntermediate = ebiten.NewImage(w, h)
	camera.clipAlphaIntermediate = ebiten.NewImage(w, h)
	camera.clipBehind = ebiten.NewImage(w, h)
	camera.sphereFactorCalculated = false

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

	if !camera.sphereFactorCalculated {
		angle := camera.FieldOfView * 3.1415 / 360
		camera.sphereFactorTang = math.Tan(angle)
		camera.sphereFactorY = 1.0 / math.Cos(angle)
		camera.sphereFactorX = 1.0 / math.Cos(math.Atan(camera.sphereFactorTang*camera.AspectRatio()))
		camera.sphereFactorCalculated = true
	}

	if camera.Perspective {
		return NewProjectionPerspective(camera.FieldOfView, camera.Near, camera.Far, float64(camera.resultColorTexture.Bounds().Dx()), float64(camera.resultColorTexture.Bounds().Dy()))
	}
	w, h := camera.resultColorTexture.Size()
	asr := float64(h) / float64(w)

	return NewProjectionOrthographic(camera.Near, camera.Far, 1*camera.OrthoScale, -1*camera.OrthoScale, asr*camera.OrthoScale, -asr*camera.OrthoScale)
	// return NewProjectionOrthographic(camera.Near, camera.Far, float64(camera.ColorTexture.Bounds().Dx())*camera.OrthoScale, float64(camera.ColorTexture.Bounds().Dy())*camera.OrthoScale)
}

// SetPerspective sets the Camera's projection to be a perspective projection. fovY indicates the vertical field of view (in degrees) for the camera's aperture.
func (camera *Camera) SetPerspective(fovY float64) {
	camera.FieldOfView = fovY
	camera.Perspective = true
	camera.sphereFactorCalculated = false
}

// SetOrthographic sets the Camera's projection to be an orthographic projection. orthoScale indicates the scale of the camera in units horizontally.
func (camera *Camera) SetOrthographic(orthoScale float64) {
	camera.Perspective = false
	camera.OrthoScale = orthoScale
	camera.sphereFactorCalculated = false
}

// We do this for each vertex for each triangle for each model, so we want to avoid allocating vectors if possible. clipToScreen
// does this by taking outVec, a vertex (vector.Vector) that it stores the values in and returns, which avoids reallocation.
func (camera *Camera) clipToScreen(vert, outVec vector.Vector, vertID int, mat *Material, width, height float64) vector.Vector {

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

	outVec[0] = (vert[0]/v3)*width + (width / 2)
	outVec[1] = (vert[1]/v3*-1)*height + (height / 2)
	outVec[2] = vert[2] / v3
	outVec[3] = 1

	if mat != nil && mat.VertexClipFunction != nil {
		outVec = mat.VertexClipFunction(outVec, vertID)
	}

	return outVec

}

// ClipToScreen projects the pre-transformed vertex in View space and remaps it to screen coordinates.
func (camera *Camera) ClipToScreen(vert vector.Vector) vector.Vector {
	width, height := camera.resultColorTexture.Size()
	return camera.clipToScreen(vert, vector.Vector{0, 0, 0, 0}, -1, nil, float64(width), float64(height))
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

// PointInFrustum returns true if the point is visible through the camera frustum.
func (camera *Camera) PointInFrustum(point vector.Vector) bool {

	diff := fastVectorSub(point, camera.WorldPosition())
	pcZ := diff.Dot(camera.cameraForward)
	aspectRatio := camera.AspectRatio()

	if pcZ > camera.Far || pcZ < camera.Near {
		return false
	}

	if camera.Perspective {

		h := pcZ * math.Tan(camera.FieldOfView/2)

		pcY := diff.Dot(camera.cameraUp)

		if -h > pcY || pcY > h {
			return false
		}

		w := h * aspectRatio

		pcX := diff.Dot(camera.cameraRight)

		if -w > pcX || pcX > w {
			return false
		}

	} else {

		width := camera.OrthoScale / 2
		height := width / camera.AspectRatio()

		pcY := diff.Dot(camera.cameraUp)

		if -height > pcY || pcY > height {
			return false
		}

		pcX := diff.Dot(camera.cameraRight)

		if -width > pcX || pcX > width {
			return false
		}

	}

	return true

}

// SphereInFrustum returns true if the sphere would be visible through the camera frustum.
func (camera *Camera) SphereInFrustum(sphere *BoundingSphere) bool {

	radius := sphere.WorldRadius()

	diff := fastVectorSub(sphere.WorldPosition(), camera.WorldPosition())
	pcZ := diff.Dot(camera.cameraForward)

	if pcZ > camera.Far+radius || pcZ < camera.Near-radius {
		return false
	}

	if camera.Perspective {

		d := camera.sphereFactorY * radius
		pcZ *= camera.sphereFactorTang

		pcY := diff.Dot(camera.cameraUp)

		if pcY > pcZ+d || pcY < -pcZ-d {
			return false
		}

		pcZ *= camera.AspectRatio()
		d = camera.sphereFactorX * radius

		pcX := diff.Dot(camera.cameraRight)

		if pcX > pcZ+d || pcX < -pcZ-d {
			return false
		}

	} else {

		width := camera.OrthoScale
		height := width / camera.AspectRatio()

		pcY := diff.Dot(camera.cameraUp)

		if -height/2-radius > pcY || pcY > height/2+radius {
			return false
		}

		pcX := diff.Dot(camera.cameraRight)

		if -width/2-radius > pcX || pcX > width/2+radius {
			return false
		}

	}

	return true

}

// AspectRatio returns the camera's aspect ratio (width / height).
func (camera *Camera) AspectRatio() float64 {
	w, h := camera.resultColorTexture.Size()
	return float64(w) / float64(h)
}

// Clear should be called at the beginning of a single rendered frame and clears the Camera's backing textures before rendering.
// It also resets the debug values.
func (camera *Camera) Clear() {

	if camera.AccumulateColorMode != AccumlateColorModeNone {
		camera.accumulatedBackBuffer.Clear()
		camera.accumulatedBackBuffer.DrawImage(camera.resultAccumulatedColorTexture, nil)
		camera.resultAccumulatedColorTexture.Clear()
		switch camera.AccumulateColorMode {
		case AccumlateColorModeBelow:
			camera.resultAccumulatedColorTexture.DrawImage(camera.accumulatedBackBuffer, camera.AccumulateDrawOptions)
			camera.resultAccumulatedColorTexture.DrawImage(camera.resultColorTexture, nil)
		case AccumlateColorModeAbove:
			camera.resultAccumulatedColorTexture.DrawImage(camera.resultColorTexture, nil)
			camera.resultAccumulatedColorTexture.DrawImage(camera.accumulatedBackBuffer, camera.AccumulateDrawOptions)
		case AccumlateColorModeSingleLastFrame:
			camera.resultAccumulatedColorTexture.DrawImage(camera.resultColorTexture, nil)
		}
	}

	camera.resultColorTexture.Clear()

	if camera.RenderDepth {
		camera.resultDepthTexture.Clear()
	}

	if time.Since(camera.DebugInfo.tickTime).Milliseconds() >= 100 {

		if !camera.DebugInfo.tickTime.IsZero() {

			camera.DebugInfo.AvgFrameTime = camera.DebugInfo.frameTime / time.Duration(camera.DebugInfo.frameCount)

			camera.DebugInfo.AvgAnimationTime = camera.DebugInfo.animationTime / time.Duration(camera.DebugInfo.frameCount)

			camera.DebugInfo.AvgLightTime = camera.DebugInfo.lightTime / time.Duration(camera.DebugInfo.frameCount)

		}

		camera.DebugInfo.tickTime = time.Now()
		camera.DebugInfo.frameTime = 0
		camera.DebugInfo.animationTime = 0
		camera.DebugInfo.lightTime = 0
		camera.DebugInfo.frameCount = 0

	}
	camera.DebugInfo.DrawnParts = 0
	camera.DebugInfo.TotalParts = 0
	camera.DebugInfo.TotalTris = 0
	camera.DebugInfo.DrawnTris = 0
	camera.DebugInfo.LightCount = 0
	camera.DebugInfo.ActiveLightCount = 0

	cameraRot := camera.WorldRotation()
	camera.cameraForward = cameraRot.Forward().Invert()
	camera.cameraRight = cameraRot.Right()
	camera.cameraUp = cameraRot.Up()

}

// RenderNodes renders all nodes starting with the provided rootNode using the Scene's properties (fog, for example). Note that if Camera.RenderDepth
// is false, scenes rendered one after another in multiple RenderNodes() calls will be rendered on top of each other in the Camera's texture buffers.
// Note that for Models, each MeshPart of a Model has a maximum renderable triangle count of 21845.
func (camera *Camera) RenderNodes(scene *Scene, rootNode INode) {

	meshes := []*Model{}

	if model, isModel := rootNode.(*Model); isModel {
		meshes = append(meshes, model)
	}

	nodes := rootNode.ChildrenRecursive()

	for _, node := range nodes {
		if model, ok := node.(*Model); ok {
			meshes = append(meshes, model)
		}
	}

	camera.Render(scene, meshes...)

}

type renderPair struct {
	Model    *Model
	MeshPart *MeshPart
}

// Render renders all of the models passed using the provided Scene's properties (fog, for example). Note that if Camera.RenderDepth
// is false, scenes rendered one after another in multiple Render() calls will be rendered on top of each other in the Camera's texture buffers.
// Note that for Models, each MeshPart of a Model has a maximum renderable triangle count of 21845.
func (camera *Camera) Render(scene *Scene, models ...*Model) {

	frametimeStart := time.Now()

	lights := []Light{}

	if scene.LightingOn {

		for _, l := range scene.Root.ChildrenRecursive() {
			if light, isLight := l.(Light); isLight {
				camera.DebugInfo.LightCount++
				if light.isOn() {
					lights = append(lights, light)
					light.beginRender()
					camera.DebugInfo.ActiveLightCount++
				}
			}
		}

	}

	// By multiplying the camera's position against the view matrix (which contains the negated camera position), we're left with just the rotation
	// matrix, which we feed into model.TransformedVertices() to draw vertices in order of distance.
	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	rectShaderOptions := &ebiten.DrawRectShaderOptions{}
	rectShaderOptions.Images[0] = camera.colorIntermediate
	rectShaderOptions.Images[1] = camera.depthIntermediate

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

	// Reusing vectors rather than reallocating for all triangles for all models
	p0 := vector.Vector{0, 0, 0, 0}
	p1 := vector.Vector{0, 0, 0, 0}
	p2 := vector.Vector{0, 0, 0, 0}

	solids := []renderPair{}
	transparents := []renderPair{}

	depths := map[*Model]float64{}

	sorting := len(solids) > 0 && !camera.RenderDepth

	for _, model := range models {

		if len(model.DynamicBatchModels) > 0 {

			dynamicDepths := map[*Model]float64{}

			transparent := false

			for _, child := range model.DynamicBatchModels {

				dynamicDepths[child] = camera.WorldToScreen(child.WorldPosition())[2]

				for _, mp := range child.Mesh.MeshParts {
					if child.isTransparent(mp) {
						transparent = true
					}
				}
			}

			sort.SliceStable(model.DynamicBatchModels, func(i, j int) bool {
				return dynamicDepths[model.DynamicBatchModels[i]] > dynamicDepths[model.DynamicBatchModels[j]]
			})

			if transparent {
				transparents = append(transparents, renderPair{model, model.Mesh.MeshParts[0]})
			} else {
				solids = append(solids, renderPair{model, model.Mesh.MeshParts[0]})
			}

		} else if model.DynamicBatchOwner == nil && model.Mesh != nil {

			for _, mp := range model.Mesh.MeshParts {
				if model.isTransparent(mp) {
					transparents = append(transparents, renderPair{model, mp})
				} else {
					solids = append(solids, renderPair{model, mp})
				}
			}

			if sorting {
				depths[model] = camera.WorldToScreen(model.WorldPosition())[2]
			}

		}

	}

	// If the camera isn't rendering depth, then we should sort models by distance to ensure things draw in something like the correct order
	if sorting {

		sort.SliceStable(solids, func(i, j int) bool {
			return depths[solids[i].Model] > depths[solids[j].Model]
		})

	}

	camWidth, camHeight := camera.resultColorTexture.Size()

	far := camera.Far
	near := camera.Near
	if !camera.Perspective {
		far = 2.0
		near = 0
	}

	render := func(rp renderPair) {

		startingVertexListIndex := vertexListIndex

		model := rp.Model
		meshPart := rp.MeshPart
		mat := meshPart.Material

		lighting := scene.LightingOn
		if mat != nil {
			lighting = scene.LightingOn && !mat.Shadeless
		}

		// Models without Meshes are essentially just "nodes" that just have a position. They aren't counted for rendering.
		if model.Mesh == nil {
			return
		}

		if !model.visible {
			return
		}

		camera.DebugInfo.TotalParts++
		camera.DebugInfo.TotalTris += meshPart.TriangleCount()

		model.Transform()

		if model.FrustumCulling {

			if !camera.SphereInFrustum(model.BoundingSphere) {
				return
			}

		}

		camera.DebugInfo.DrawnParts++

		model.ProcessVertices(vpMatrix, camera, meshPart, scene)

		backfaceCulling := true
		if mat != nil {
			backfaceCulling = mat.BackfaceCulling
		}

		srcW := 0.0
		srcH := 0.0

		if mat != nil && mat.Texture != nil {
			srcW = float64(mat.Texture.Bounds().Dx())
			srcH = float64(mat.Texture.Bounds().Dy())
		}

		var img *ebiten.Image

		if mat != nil {
			img = mat.Texture
		}

		if img == nil {
			img = defaultImg
		}

		if lighting {

			t := time.Now()

			for _, light := range lights {
				light.beginModel(model, camera)
			}

			camera.DebugInfo.lightTime += time.Since(t)

		}

		mesh := model.Mesh

		// Here we do all vertex transforms first because of data locality (it's faster to access all vertex transformations, then go back and do all UV values, etc)

		for t := range meshPart.sortingTriangles {

			meshPart.sortingTriangles[t].rendered = false

			vertIndex := meshPart.sortingTriangles[t].ID * 3
			v0 := mesh.vertexTransforms[vertIndex]
			v1 := mesh.vertexTransforms[vertIndex+1]
			v2 := mesh.vertexTransforms[vertIndex+2]

			// Near-ish clipping (basically clip triangles that are wholly behind the camera)
			if v0[3] < 0 && v1[3] < 0 && v2[3] < 0 {
				continue
			}

			if v0[2] > far && v1[2] > far && v2[2] > far {
				continue
			}

			p0 = camera.clipToScreen(v0, p0, vertIndex, mat, float64(camWidth), float64(camHeight))
			p1 = camera.clipToScreen(v1, p1, vertIndex+1, mat, float64(camWidth), float64(camHeight))
			p2 = camera.clipToScreen(v2, p2, vertIndex+2, mat, float64(camWidth), float64(camHeight))

			// We can skip triangles that lie entirely outside of the view horizontally and vertically.
			if (p0[0] < 0 && p1[0] < 0 && p2[0] < 0) ||
				(p0[1] < 0 && p1[1] < 0 && p2[1] < 0) ||
				(p0[0] > float64(camWidth) && p1[0] > float64(camWidth) && p2[0] > float64(camWidth)) ||
				(p0[1] > float64(camHeight) && p1[1] > float64(camHeight) && p2[1] > float64(camHeight)) {
				continue
			}

			// This is a bit of a hacky way to do backface culling; it works, but it uses
			// the screen positions of the vertices to determine if the triangle should be culled.
			// In truth, it would be better to use the above approach, but that gives us visual
			// errors when faces are behind the camera unless we clip triangles. I don't really
			// feel like doing that right now, so here we are.

			if backfaceCulling {

				camera.backfacePool.Reset()
				n0 := camera.backfacePool.Sub(p0, p1)[:3]
				n1 := camera.backfacePool.Sub(p1, p2)[:3]
				nor := camera.backfacePool.Cross(n0, n1)

				if nor[2] > 0 {
					continue
				}

			}

			// Enforce maximum vertex count; note that this is lazy, which is NOT really a good way of doing this, as you can't really know ahead of time how many triangles may render.
			if vertexListIndex/3 >= ebiten.MaxIndicesNum/3 {
				if model.DynamicBatchOwner == nil {
					panic("error in rendering mesh [" + model.Mesh.Name + "] of model [" + model.name + "]. At " + fmt.Sprintf("%d", len(model.Mesh.Triangles)) + " triangles, it exceeds the maximum of 21845 rendered triangles total for one MeshPart; please break up the mesh into multiple MeshParts using materials, or split it up into models")
				} else {
					panic("error in rendering mesh [" + model.Mesh.Name + "] of model [" + model.name + "] underneath Dynamic merging owner " + model.DynamicBatchOwner.name + ". At " + fmt.Sprintf("%d", model.DynamicBatchOwner.DynamicBatchTriangleCount()) + " triangles, it exceeds the maximum of 21845 rendered triangles total for one MeshPart; please break up the mesh into multiple MeshParts using materials, or split it up into models")
				}
			}

			colorVertexList[vertexListIndex].DstX = float32(p0[0])
			colorVertexList[vertexListIndex].DstY = float32(p0[1])
			colorVertexList[vertexListIndex+1].DstX = float32(p1[0])
			colorVertexList[vertexListIndex+1].DstY = float32(p1[1])
			colorVertexList[vertexListIndex+2].DstX = float32(p2[0])
			colorVertexList[vertexListIndex+2].DstY = float32(p2[1])

			depthVertexList[vertexListIndex].DstX = float32(p0[0])
			depthVertexList[vertexListIndex].DstY = float32(p0[1])
			depthVertexList[vertexListIndex+1].DstX = float32(p1[0])
			depthVertexList[vertexListIndex+1].DstY = float32(p1[1])
			depthVertexList[vertexListIndex+2].DstX = float32(p2[0])
			depthVertexList[vertexListIndex+2].DstY = float32(p2[1])

			meshPart.sortingTriangles[t].rendered = true

			vertexListIndex += 3

		}

		if vertexListIndex == startingVertexListIndex {
			return
		}

		vertexListIndex = startingVertexListIndex

		for _, tri := range meshPart.sortingTriangles {

			if !tri.rendered {
				continue
			}

			for i := 0; i < 3; i++ {

				vertIndex := tri.ID*3 + i

				// We set the UVs back here because we might need to use them if the material has clip alpha enabled.
				u := float32(mesh.VertexUVs[vertIndex][0] * srcW)
				// We do 1 - v here (aka Y in texture coordinates) because 1.0 is the top of the texture while 0 is the bottom in UV coordinates,
				// but when drawing textures 0 is the top, and the sourceHeight is the bottom.
				v := float32((1 - mesh.VertexUVs[vertIndex][1]) * srcH)

				colorVertexList[vertexListIndex+i].SrcX = u
				colorVertexList[vertexListIndex+i].SrcY = v

				// Vertex colors

				if activeChannel := mesh.VertexActiveColorChannel[vertIndex]; activeChannel >= 0 {
					colorVertexList[vertexListIndex+i].ColorR = mesh.VertexColors[vertIndex][activeChannel].R
					colorVertexList[vertexListIndex+i].ColorG = mesh.VertexColors[vertIndex][activeChannel].G
					colorVertexList[vertexListIndex+i].ColorB = mesh.VertexColors[vertIndex][activeChannel].B
					colorVertexList[vertexListIndex+i].ColorA = mesh.VertexColors[vertIndex][activeChannel].A
				} else {
					colorVertexList[vertexListIndex+i].ColorR = 1
					colorVertexList[vertexListIndex+i].ColorG = 1
					colorVertexList[vertexListIndex+i].ColorB = 1
					colorVertexList[vertexListIndex+i].ColorA = 1
				}

				if camera.RenderDepth {

					// We're adding 0.03 for a margin because for whatever reason, at close range / wide FOV,
					// depth can be negative but still be in front of the camera and not behind it.
					depth := (mesh.vertexTransforms[vertIndex][2]+near)/far + 0.03
					if depth < 0 {
						depth = 0
					} else if depth > 1 {
						depth = 1
					}

					depthVertexList[vertexListIndex+i].ColorR = float32(depth)
					depthVertexList[vertexListIndex+i].ColorG = float32(depth)
					depthVertexList[vertexListIndex+i].ColorB = float32(depth)
					depthVertexList[vertexListIndex+i].ColorA = 1

					// We set the UVs back here because we might need to use them if the material has clip alpha enabled.
					depthVertexList[vertexListIndex+i].SrcX = u

					// We do 1 - v here (aka Y in texture coordinates) because 1.0 is the top of the texture while 0 is the bottom in UV coordinates,
					// but when drawing textures 0 is the top, and the sourceHeight is the bottom.
					depthVertexList[vertexListIndex+i].SrcY = v

				} else if scene.FogMode != FogOff {

					// We're adding 0.03 for a margin because for whatever reason, at close range / wide FOV,
					// depth can be negative but still be in front of the camera and not behind it.
					depth := float32((mesh.vertexTransforms[vertIndex][2]+near)/far + 0.03)
					if depth < 0 {
						depth = 0
					} else if depth > 1 {
						depth = 1
					}

					// depth = 1 - depth

					depth = scene.FogRange[0] + ((scene.FogRange[1]-scene.FogRange[0])*1 - depth)

					if scene.FogMode == FogAdd {
						colorVertexList[vertexListIndex+i].ColorR += scene.FogColor.R * depth
						colorVertexList[vertexListIndex+i].ColorG += scene.FogColor.G * depth
						colorVertexList[vertexListIndex+i].ColorB += scene.FogColor.B * depth
					} else if scene.FogMode == FogMultiply {
						colorVertexList[vertexListIndex+i].ColorR *= scene.FogColor.R * depth
						colorVertexList[vertexListIndex+i].ColorG *= scene.FogColor.G * depth
						colorVertexList[vertexListIndex+i].ColorB *= scene.FogColor.B * depth
					}

				}

			}

			if lighting {

				t := time.Now()

				addLightResults := [9]float32{}

				for _, light := range lights {
					lightResults := light.Light(tri.ID, model)
					for i := 0; i < 9; i++ {
						addLightResults[i] += lightResults[i]
					}
				}

				for i := 0; i < 3; i++ {
					colorVertexList[vertexListIndex+i].ColorR *= addLightResults[i*3]
					colorVertexList[vertexListIndex+i].ColorG *= addLightResults[i*3+1]
					colorVertexList[vertexListIndex+i].ColorB *= addLightResults[i*3+2]
				}

				camera.DebugInfo.lightTime += time.Since(t)

			}

			vertexListIndex += 3

		}

		for i := 0; i < vertexListIndex; i++ {
			indexList[i] = uint16(i)
		}

	}

	flush := func(rp renderPair) {

		if vertexListIndex == 0 {
			return
		}

		model := rp.Model
		meshPart := rp.MeshPart
		mat := meshPart.Material

		var img *ebiten.Image

		if mat != nil {
			img = mat.Texture
		}

		if img == nil {
			img = defaultImg
		}

		// Render the depth map here
		if camera.RenderDepth {

			// OK, so the general process for rendering to the depth texture is three-fold:
			// 1) For solid objects, we simply render all triangles using camera.DepthShader. This draws triangles using their vertices'
			// color channels to indicate depth. It reads camera.DepthTexture to discard fragments previously rendered with a darker color
			// (and so are closer, as the color ranges from 0 (black, close) to 1 (white, far)).

			// 2) For transparent objects, we do the above, but don't write the transparent object to the DepthTexture.
			// By not rendering the object to the DepthTexture but still rendering it to the DepthIntermediate texture, the object still renders to
			// camera.ColorTexture visibly, but does not obscure any objects behind them by writing to the depth texture, thereby
			// allowing them to be seen through the transparent materials.
			// Transparent objects are rendered in a second pass, in far-to-close ordering.

			// 3) For alpha clip objects, we first render the triangles to an intermediate texture (camera.AlphaClipIntermediate). We use the camera.ClipAlphaRenderShader to
			// render the triangles with their vertex colors, and use the texture's alpha channel for clipping. We then draw the intermediate render
			// to DepthIntermediate using the camera.ClipAlphaShaderComposite shader, which is used to both discard previous closer fragments on camera.DepthTexture,
			// as well as cut out the alpha of the actual texture according to the ClipIntermediate image. This ClipIntermediate rendering step would be unnecessary if we could render to DepthIntermediate using the material's
			// image while also reading the DepthTexture, but unfortunately, images can't currently be different sizes in Ebiten.
			// See: https://github.com/hajimehoshi/ebiten/issues/1870

			transparencyMode := TransparencyModeOpaque

			if mat != nil {
				transparencyMode = mat.TransparencyMode
			}

			camera.depthIntermediate.Clear()

			if transparencyMode == TransparencyModeAlphaClip {

				camera.clipAlphaIntermediate.Clear()

				camera.clipAlphaIntermediate.DrawTrianglesShader(depthVertexList[:vertexListIndex], indexList[:vertexListIndex], camera.clipAlphaRenderShader, &ebiten.DrawTrianglesShaderOptions{Images: [4]*ebiten.Image{img}})

				w, h := camera.depthIntermediate.Size()

				camera.depthIntermediate.DrawRectShader(w, h, camera.clipAlphaCompositeShader, &ebiten.DrawRectShaderOptions{Images: [4]*ebiten.Image{camera.resultDepthTexture, camera.clipAlphaIntermediate}})

			} else {
				shaderOpt := &ebiten.DrawTrianglesShaderOptions{
					Images: [4]*ebiten.Image{camera.resultDepthTexture},
				}

				camera.depthIntermediate.DrawTrianglesShader(depthVertexList[:vertexListIndex], indexList[:vertexListIndex], camera.depthShader, shaderOpt)
			}

			if !model.isTransparent(meshPart) {
				camera.resultDepthTexture.DrawImage(camera.depthIntermediate, nil)
			}

		}

		t := &ebiten.DrawTrianglesOptions{}
		t.ColorM = model.ColorBlendingFunc(model, meshPart) // Modify the model's appearance using its color blending function
		if mat != nil {
			t.Filter = mat.TextureFilterMode
			t.Address = mat.TextureWrapMode
		}

		hasFragShader := mat != nil && mat.fragmentShader != nil && mat.FragmentShaderOn
		w, h := camera.resultColorTexture.Size()

		// If rendering depth, and rendering through a custom fragment shader, we'll need to render the tris to the ColorIntermediate buffer using the custom shader.
		// If we're not rendering through a custom shader, we can render to ColorIntermediate and then composite that onto the finished ColorTexture.
		// If we're not rendering depth, but still rendering through the shader, we can render to the intermediate texture, and then from there composite.
		// Otherwise, we can just draw the triangles normally.

		t.CompositeMode = ebiten.CompositeModeSourceOver
		rectShaderOptions.CompositeMode = ebiten.CompositeModeSourceOver

		if camera.RenderDepth {

			camera.colorIntermediate.Clear()

			if mat != nil {
				rectShaderOptions.CompositeMode = mat.CompositeMode
			}

			if hasFragShader {
				camera.colorIntermediate.DrawTrianglesShader(colorVertexList[:vertexListIndex], indexList[:vertexListIndex], mat.fragmentShader, mat.FragmentShaderOptions)
			} else {
				camera.colorIntermediate.DrawTriangles(colorVertexList[:vertexListIndex], indexList[:vertexListIndex], img, t)
			}

			camera.resultColorTexture.DrawRectShader(w, h, camera.colorShader, rectShaderOptions)

		} else {

			if mat != nil {
				t.CompositeMode = mat.CompositeMode
			}

			if hasFragShader {
				camera.resultColorTexture.DrawTrianglesShader(colorVertexList[:vertexListIndex], indexList[:vertexListIndex], mat.fragmentShader, mat.FragmentShaderOptions)
			} else {
				camera.resultColorTexture.DrawTriangles(colorVertexList[:vertexListIndex], indexList[:vertexListIndex], img, t)
			}

		}

		camera.DebugInfo.DrawnTris += vertexListIndex / 3

		vertexListIndex = 0

	}

	for _, pair := range solids {

		// Internally, the idea behind dynamic batching is that we simply hold off on flushing until the
		// end - this saves a lot of time if we're rendering singular low-poly objects, at the cost of each
		// object sharing the same material / object-level properties (color / material blending mode, for
		// example).
		if dyn := pair.Model.DynamicBatchModels; len(dyn) > 0 {

			for _, merged := range dyn {
				for _, part := range merged.Mesh.MeshParts {
					render(renderPair{Model: merged, MeshPart: part})
				}
			}

			flush(pair)
		} else {
			render(pair)
			flush(pair)
		}

	}

	if len(transparents) > 0 {

		sort.SliceStable(transparents, func(i, j int) bool {
			return depths[transparents[i].Model] > depths[transparents[j].Model]
		})

		for _, pair := range transparents {

			if dyn := pair.Model.DynamicBatchModels; len(dyn) > 0 {

				for _, merged := range dyn {
					for _, part := range merged.Mesh.MeshParts {
						render(renderPair{Model: merged, MeshPart: part})
					}
				}

				flush(pair)
			} else {
				render(pair)
				flush(pair)
			}

		}

	}

	camera.DebugInfo.frameTime += time.Since(frametimeStart)

	camera.DebugInfo.frameCount++

}

func (camera *Camera) drawCircle(screen *ebiten.Image, position vector.Vector, radius float64, drawColor color.Color) {

	transformedCenter := camera.WorldToScreen(position)

	stepCount := float64(32)

	for i := 0; i < int(stepCount); i++ {

		x := (math.Sin(math.Pi*2*float64(i)/stepCount) * radius)
		y := (math.Cos(math.Pi*2*float64(i)/stepCount) * radius)

		x2 := (math.Sin(math.Pi*2*float64(i+1)/stepCount) * radius)
		y2 := (math.Cos(math.Pi*2*float64(i+1)/stepCount) * radius)

		ebitenutil.DrawLine(screen, transformedCenter[0]+x, transformedCenter[1]+y, transformedCenter[0]+x2, transformedCenter[1]+y2, drawColor)
	}

}

// DrawDebugRenderInfo draws render debug information (like number of drawn objects, number of drawn triangles, frame time, etc)
// at the top-left of the provided screen *ebiten.Image, using the textScale and color provided.
func (camera *Camera) DrawDebugRenderInfo(screen *ebiten.Image, textScale float64, color *Color) {

	m := camera.DebugInfo.AvgFrameTime.Round(time.Microsecond).Microseconds()
	ft := fmt.Sprintf("%.2fms", float32(m)/1000)

	m = camera.DebugInfo.AvgAnimationTime.Round(time.Microsecond).Microseconds()
	at := fmt.Sprintf("%.2fms", float32(m)/1000)

	m = camera.DebugInfo.AvgLightTime.Round(time.Microsecond).Microseconds()
	lt := fmt.Sprintf("%.2fms", float32(m)/1000)

	debugText := fmt.Sprintf(
		"TPS: %f\nFPS: %f\nTotal render frame-time: %s\nSkinned mesh animation time: %s\nLighting frame-time: %s\nDraw calls: %d/%d\nRendered triangles: %d/%d\nActive Lights: %d/%d",
		ebiten.CurrentTPS(),
		ebiten.CurrentFPS(),
		ft,
		at,
		lt,
		camera.DebugInfo.DrawnParts,
		camera.DebugInfo.TotalParts,
		camera.DebugInfo.DrawnTris,
		camera.DebugInfo.TotalTris,
		camera.DebugInfo.ActiveLightCount,
		camera.DebugInfo.LightCount)

	camera.DebugDrawText(screen, debugText, 0, 0, textScale, color)

}

// DrawDebugWireframe draws the wireframe triangles of all visible Models underneath the rootNode in the color provided to the screen
// image provided.
func (camera *Camera) DrawDebugWireframe(screen *ebiten.Image, rootNode INode, color *Color) {

	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	camWidth, camHeight := camera.resultColorTexture.Size()

	for _, m := range allModels {

		if model, isModel := m.(*Model); isModel {

			if model.FrustumCulling {

				model.Transform()
				if !camera.SphereInFrustum(model.BoundingSphere) {
					continue
				}

			}

			for _, meshPart := range model.Mesh.MeshParts {

				model.ProcessVertices(vpMatrix, camera, meshPart, nil)

				for i := 0; i < len(model.Mesh.vertexTransforms); i += 3 {

					v0 := camera.ClipToScreen(model.Mesh.vertexTransforms[i])
					v1 := camera.ClipToScreen(model.Mesh.vertexTransforms[i+1])
					v2 := camera.ClipToScreen(model.Mesh.vertexTransforms[i+2])

					if (v0[0] < 0 && v1[0] < 0 && v2[0] < 0) ||
						(v0[1] < 0 && v1[1] < 0 && v2[1] < 0) ||
						(v0[0] > float64(camWidth) && v1[0] > float64(camWidth) && v2[0] > float64(camWidth)) ||
						(v0[1] > float64(camHeight) && v1[1] > float64(camHeight) && v2[1] > float64(camHeight)) {
						continue
					}

					c := color.ToRGBA64()
					ebitenutil.DrawLine(screen, float64(v0[0]), float64(v0[1]), float64(v1[0]), float64(v1[1]), c)
					ebitenutil.DrawLine(screen, float64(v1[0]), float64(v1[1]), float64(v2[0]), float64(v2[1]), c)
					ebitenutil.DrawLine(screen, float64(v2[0]), float64(v2[1]), float64(v0[0]), float64(v0[1]), c)

				}

			}

		} else if curve, isCurve := m.(*Path); isCurve {

			points := curve.Children()
			if curve.Closed {
				points = append(points, points[0])
			}
			for i := 0; i < len(points)-1; i++ {
				px := camera.WorldToScreen(points[i].WorldPosition())
				py := camera.WorldToScreen(points[i+1].WorldPosition())
				ebitenutil.DrawLine(screen, px[0], px[1], py[0], py[1], color.ToRGBA64())
			}

		}

	}

}

// DrawDebugDrawOrder draws the drawing order of all triangles of all visible Models underneath the rootNode in the color provided to the screen
// image provided.
func (camera *Camera) DrawDebugDrawOrder(screen *ebiten.Image, rootNode INode, textScale float64, color *Color) {

	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	for _, m := range allModels {

		if model, isModel := m.(*Model); isModel {

			if model.FrustumCulling {

				model.Transform()
				if !camera.SphereInFrustum(model.BoundingSphere) {
					continue
				}

			}

			for _, meshPart := range model.Mesh.MeshParts {

				model.ProcessVertices(vpMatrix, camera, meshPart, nil)

				triangles := model.Mesh.Triangles

				for triIndex, sortingTri := range meshPart.sortingTriangles {

					screenPos := camera.WorldToScreen(model.Transform().MultVec(triangles[sortingTri.ID].Center))

					camera.DebugDrawText(screen, fmt.Sprintf("%d", triIndex), screenPos[0], screenPos[1]+(textScale*16), textScale, color)

				}

			}

		}

	}

}

// DrawDebugDrawCallCount draws the draw call count of all visible Models underneath the rootNode in the color provided to the screen
// image provided.
func (camera *Camera) DrawDebugDrawCallCount(screen *ebiten.Image, rootNode INode, textScale float64, color *Color) {

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	for _, m := range allModels {

		if model, isModel := m.(*Model); isModel {

			if model.FrustumCulling {

				model.Transform()
				if !camera.SphereInFrustum(model.BoundingSphere) {
					continue
				}

			}

			screenPos := camera.WorldToScreen(model.WorldPosition())

			camera.DebugDrawText(screen, fmt.Sprintf("%d", len(model.Mesh.MeshParts)), screenPos[0], screenPos[1]+(textScale*16), textScale, color)

		}

	}

}

// DrawDebugNormals draws the normals of visible models underneath the rootNode given to the screen. NormalLength is the length of the normal lines
// in units. Color is the color to draw the normals.
func (camera *Camera) DrawDebugNormals(screen *ebiten.Image, rootNode INode, normalLength float64, color *Color) {

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	for _, m := range allModels {

		if model, isModel := m.(*Model); isModel {

			if model.FrustumCulling {

				model.Transform()
				if !camera.SphereInFrustum(model.BoundingSphere) {
					continue
				}

			}

			for _, tri := range model.Mesh.Triangles {

				center := camera.WorldToScreen(model.Transform().MultVecW(tri.Center))
				transformedNormal := camera.WorldToScreen(model.Transform().MultVecW(tri.Center.Add(tri.Normal.Scale(normalLength))))
				ebitenutil.DrawLine(screen, center[0], center[1], transformedNormal[0], transformedNormal[1], color.ToRGBA64())

			}

		}

	}

}

// DrawDebugCenters draws the center positions of nodes under the rootNode using the color given to the screen image provided.
func (camera *Camera) DrawDebugCenters(screen *ebiten.Image, rootNode INode, color *Color) {

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	c := color.ToRGBA64()

	for _, node := range allModels {

		if node == camera {
			continue
		}

		camera.drawCircle(screen, node.WorldPosition(), 8, c)

		// If the node's parent is something, and its parent's parent is something (i.e. it's not the root)
		if node.Parent() != nil && node.Parent() != node.Root() {
			parentPos := camera.WorldToScreen(node.Parent().WorldPosition())
			pos := camera.WorldToScreen(node.WorldPosition())
			ebitenutil.DrawLine(screen, pos[0], pos[1], parentPos[0], parentPos[1], c)
		}

	}

}

func (camera *Camera) DebugDrawText(screen *ebiten.Image, txtStr string, posX, posY, textScale float64, color *Color) {

	dr := &ebiten.DrawImageOptions{}
	dr.ColorM.Scale(0, 0, 0, 1)

	periodOffset := 8.0

	for y := -1; y < 2; y++ {

		for x := -1; x < 2; x++ {

			dr.GeoM.Reset()
			dr.GeoM.Translate(posX+4+float64(x), posY+4+float64(y)+periodOffset)
			dr.GeoM.Scale(textScale, textScale)

			text.DrawWithOptions(screen, txtStr, basicfont.Face7x13, dr)
		}

	}

	dr.ColorM.Reset()
	dr.ColorM.Scale(color.ToFloat64s())

	dr.GeoM.Reset()
	dr.GeoM.Translate(posX+4, posY+4+periodOffset)
	dr.GeoM.Scale(textScale, textScale)

	text.DrawWithOptions(screen, txtStr, basicfont.Face7x13, dr)

}

// ColorTexture returns the camera's final result color texture from any previous Render() or RenderNodes() calls.
func (camera *Camera) ColorTexture() *ebiten.Image {
	return camera.resultColorTexture
}

// DepthTexture returns the camera's final result depth texture from any previous Render() or RenderNodes() calls. If Camera.RenderDepth is set to false,
// the function will return nil instead.
func (camera *Camera) DepthTexture() *ebiten.Image {
	if !camera.RenderDepth {
		return nil
	}
	return camera.resultDepthTexture
}

// AccumulationColorTexture returns the camera's final result accumulation color texture from previous renders. If the Camera's AccumulateColorMode
// property is set to AccumulateColorModeNone, the function will return nil instead.
func (camera *Camera) AccumulationColorTexture() *ebiten.Image {
	if camera.AccumulateColorMode == AccumlateColorModeNone {
		return nil
	}
	return camera.resultAccumulatedColorTexture
}

// DrawDebugBoundsColored will draw shapes approximating the shapes and positions of BoundingObjects underneath the rootNode. The shapes will
// be drawn in the color provided for each kind of bounding object to the screen image provided.
func (camera *Camera) DrawDebugBoundsColored(screen *ebiten.Image, rootNode INode, aabbColor, sphereColor, capsuleColor, trianglesColor *Color) {

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	camWidth, camHeight := camera.resultColorTexture.Size()

	for _, n := range allModels {

		if b, isBounds := n.(BoundingObject); isBounds {

			switch bounds := b.(type) {

			case *BoundingSphere:

				pos := bounds.WorldPosition()
				radius := bounds.WorldRadius()

				u := camera.WorldToScreen(pos.Add(vector.Y.Scale(radius)))
				d := camera.WorldToScreen(pos.Add(vector.Y.Scale(-radius)))
				r := camera.WorldToScreen(pos.Add(vector.X.Scale(radius)))
				l := camera.WorldToScreen(pos.Add(vector.X.Scale(-radius)))
				f := camera.WorldToScreen(pos.Add(vector.Z.Scale(radius)))
				b := camera.WorldToScreen(pos.Add(vector.Z.Scale(-radius)))

				lines := []vector.Vector{
					u, r, d, l,
					u, f, d, b, u,
					b, r, f, l, b,
				}

				for i := range lines {

					if i >= len(lines)-1 {
						break
					}

					start := lines[i]
					end := lines[i+1]
					ebitenutil.DrawLine(screen, start[0], start[1], end[0], end[1], sphereColor.ToRGBA64())

				}

			case *BoundingCapsule:

				pos := bounds.WorldPosition()
				radius := bounds.WorldRadius()
				height := bounds.Height / 2

				uv := bounds.WorldRotation().Up()
				rv := bounds.WorldRotation().Right()
				fv := bounds.WorldRotation().Forward()

				u := camera.WorldToScreen(pos.Add(uv.Scale(height)))

				ur := camera.WorldToScreen(pos.Add(uv.Scale(height - radius)).Add(rv.Scale(radius)))
				ul := camera.WorldToScreen(pos.Add(uv.Scale(height - radius)).Add(rv.Scale(-radius)))
				uf := camera.WorldToScreen(pos.Add(uv.Scale(height - radius)).Add(fv.Scale(radius)))
				ub := camera.WorldToScreen(pos.Add(uv.Scale(height - radius)).Add(fv.Scale(-radius)))

				d := camera.WorldToScreen(pos.Add(uv.Scale(-height)))

				dr := camera.WorldToScreen(pos.Add(uv.Scale(-(height - radius))).Add(rv.Scale(radius)))
				dl := camera.WorldToScreen(pos.Add(uv.Scale(-(height - radius))).Add(rv.Scale(-radius)))
				df := camera.WorldToScreen(pos.Add(uv.Scale(-(height - radius))).Add(fv.Scale(radius)))
				db := camera.WorldToScreen(pos.Add(uv.Scale(-(height - radius))).Add(fv.Scale(-radius)))

				lines := []vector.Vector{
					u, ur, dr, d, dl, ul,
					u, uf, df, d, db, ub, u,
					ul, uf, ur, ub, ul,
					dl, db, dr, df, dl,
				}

				for i := range lines {

					if i >= len(lines)-1 {
						break
					}

					start := lines[i]
					end := lines[i+1]
					ebitenutil.DrawLine(screen, start[0], start[1], end[0], end[1], capsuleColor.ToRGBA64())

				}

			case *BoundingAABB:

				pos := bounds.WorldPosition()
				size := bounds.Size.Scale(1.0 / 2.0)

				ufr := camera.WorldToScreen(pos.Add(vector.Vector{size[0], size[1], size[2]}))
				ufl := camera.WorldToScreen(pos.Add(vector.Vector{-size[0], size[1], size[2]}))
				ubr := camera.WorldToScreen(pos.Add(vector.Vector{size[0], size[1], -size[2]}))
				ubl := camera.WorldToScreen(pos.Add(vector.Vector{-size[0], size[1], -size[2]}))

				dfr := camera.WorldToScreen(pos.Add(vector.Vector{size[0], -size[1], size[2]}))
				dfl := camera.WorldToScreen(pos.Add(vector.Vector{-size[0], -size[1], size[2]}))
				dbr := camera.WorldToScreen(pos.Add(vector.Vector{size[0], -size[1], -size[2]}))
				dbl := camera.WorldToScreen(pos.Add(vector.Vector{-size[0], -size[1], -size[2]}))

				lines := []vector.Vector{
					ufr, ufl, ubl, ubr, ufr,
					dfr, dfl, dbl, dbr, dfr,
					ufr, ufl, dfl, dbl, ubl, ubr, dbr,
				}

				for i := range lines {

					if i >= len(lines)-1 {
						break
					}

					start := lines[i]
					end := lines[i+1]
					ebitenutil.DrawLine(screen, start[0], start[1], end[0], end[1], aabbColor.ToRGBA64())

				}

			case *BoundingTriangles:

				lines := []vector.Vector{}

				mesh := bounds.Mesh

				for _, tri := range mesh.Triangles {

					mvpMatrix := bounds.Transform().Mult(camera.ViewMatrix().Mult(camera.Projection()))

					v0 := camera.ClipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.ID*3]))
					v1 := camera.ClipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.ID*3+1]))
					v2 := camera.ClipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.ID*3+2]))

					if (v0[0] < 0 && v1[0] < 0 && v2[0] < 0) ||
						(v0[1] < 0 && v1[1] < 0 && v2[1] < 0) ||
						(v0[0] > float64(camWidth) && v1[0] > float64(camWidth) && v2[0] > float64(camWidth)) ||
						(v0[1] > float64(camHeight) && v1[1] > float64(camHeight) && v2[1] > float64(camHeight)) {
						continue
					}

					lines = append(lines, v0, v1, v2)

				}

				triColor := trianglesColor.ToRGBA64()

				for i := 0; i < len(lines); i += 3 {

					if i >= len(lines)-1 {
						break
					}

					start := lines[i]
					end := lines[i+1]
					ebitenutil.DrawLine(screen, start[0], start[1], end[0], end[1], triColor)

					start = lines[i+1]
					end = lines[i+2]
					ebitenutil.DrawLine(screen, start[0], start[1], end[0], end[1], triColor)

					start = lines[i+2]
					end = lines[i]
					ebitenutil.DrawLine(screen, start[0], start[1], end[0], end[1], triColor)

				}

				camera.DrawDebugBoundsColored(screen, bounds.BoundingAABB, aabbColor, sphereColor, capsuleColor, trianglesColor)

			}

		}

	}

}

// DrawDebugBounds will draw shapes approximating the shapes and positions of BoundingObjects underneath the rootNode. The shapes will
// be drawn in the color provided to the screen image provided.
func (camera *Camera) DrawDebugBounds(screen *ebiten.Image, rootNode INode, color *Color) {
	camera.DrawDebugBoundsColored(screen, rootNode, color, color, color, color)
}

// DrawDebugFrustums will draw shapes approximating the frustum spheres for objects underneath the rootNode.
// The shapes will be drawn in the color provided to the screen image provided.
func (camera *Camera) DrawDebugFrustums(screen *ebiten.Image, rootNode INode, color *Color) {

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	for _, n := range allModels {

		if model, ok := n.(*Model); ok {

			bounds := model.BoundingSphere

			pos := bounds.WorldPosition()
			radius := bounds.WorldRadius()

			u := camera.WorldToScreen(pos.Add(vector.Y.Scale(radius)))
			d := camera.WorldToScreen(pos.Add(vector.Y.Scale(-radius)))
			r := camera.WorldToScreen(pos.Add(vector.X.Scale(radius)))
			l := camera.WorldToScreen(pos.Add(vector.X.Scale(-radius)))
			f := camera.WorldToScreen(pos.Add(vector.Z.Scale(radius)))
			b := camera.WorldToScreen(pos.Add(vector.Z.Scale(-radius)))

			lines := []vector.Vector{
				u, r, d, l,
				u, f, d, b, u,
				b, r, f, l, b,
			}

			for i := range lines {

				if i >= len(lines)-1 {
					break
				}

				start := lines[i]
				end := lines[i+1]
				ebitenutil.DrawLine(screen, start[0], start[1], end[0], end[1], color.ToRGBA64())

			}

		}

	}

}

/////

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

// Type returns the NodeType for this object.
func (camera *Camera) Type() NodeType {
	return NodeTypeCamera
}
