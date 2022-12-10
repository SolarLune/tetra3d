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
	TotalParts       int // Total number of draw calls
	BatchedParts     int // Total batched number of draw calls
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

	resultAccumulatedColorTexture *ebiten.Image // ResultAccumulatedColorTexture holds the previous frame's render result of rendering any models.
	accumulatedBackBuffer         *ebiten.Image
	AccumulateColorMode           int                      // The mode to use when rendering previous frames to the accumulation buffer. Defaults to AccumulateColorModeNone.
	AccumulateDrawOptions         *ebiten.DrawImageOptions // Draw image options to use when rendering frames to the accumulation buffer; use this to fade out or color previous frames.

	Near, Far   float64 // The near and far clipping plane. Near defaults to 0.1, Far to 100 (unless these settings are loaded from a camera in a GLTF file).
	Perspective bool    // If the Camera has a perspective projection. If not, it would be orthographic
	FieldOfView float64 // Vertical field of view in degrees for a perspective projection camera
	OrthoScale  float64 // Scale of the view for an orthographic projection camera in units horizontally

	DebugInfo DebugInfo

	depthShader              *ebiten.Shader
	clipAlphaCompositeShader *ebiten.Shader
	clipAlphaRenderShader    *ebiten.Shader
	colorShader              *ebiten.Shader
	sprite3DShader           *ebiten.Shader

	// Visibility check variables
	cameraForward          Vector
	cameraRight            Vector
	cameraUp               Vector
	sphereFactorY          float64
	sphereFactorX          float64
	sphereFactorTang       float64
	sphereFactorCalculated bool

	debugTextTexture *ebiten.Image
}

// NewCamera creates a new Camera with the specified width and height.
func NewCamera(w, h int) *Camera {

	cam := &Camera{
		Node:        NewNode("Camera"),
		RenderDepth: true,
		Near:        0.1,
		Far:         100,

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

			discard()

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
			tex := imageSrc0UnsafeAt(texCoord)
			if (tex.a == 0) {
				discard()
			}
			return vec4(encodeDepth(color.r).rgb, tex.a)
			
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

			depthValue := imageSrc0UnsafeAt(texCoord)
			texture := imageSrc1UnsafeAt(texCoord)

			if depthValue.a == 0 || decodeDepth(depthValue) > texture.r {
				return texture
			}

			discard()

		}

		`,
	)

	cam.clipAlphaCompositeShader, err = ebiten.NewShader(clipCompositeShaderText)

	if err != nil {
		panic(err)
	}

	cam.colorShader, err = ebiten.NewShader([]byte(
		`package main

		var Fog vec4
		var FogRange [2]float
		var DitherSize float

		var BayerMatrix [16]float

		func decodeDepth(rgba vec4) float {
			return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
		}
		
		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			depth := imageSrc1UnsafeAt(texCoord)
			
			if depth.a > 0 {
				colorTex := imageSrc0UnsafeAt(texCoord)
				
				d := smoothstep(FogRange[0], FogRange[1], decodeDepth(depth))

				if Fog.a == 1 {
					colorTex.rgb += Fog.rgb * d * colorTex.a
				} else if Fog.a == 2 {
					colorTex.rgb -= Fog.rgb * d * colorTex.a
				} else if Fog.a == 3 {
					colorTex.rgb = mix(colorTex.rgb, Fog.rgb, d) * colorTex.a
				} else if Fog.a == 4 {
					colorTex.a *= abs(1-d)
					if DitherSize > 0 {
						yc := int(position.y / DitherSize)%4
						xc := int(position.x / DitherSize)%4
						colorTex.a *= step(0, abs(1-d) - BayerMatrix[(yc*4) + xc])
					}
					colorTex.rgb = mix(vec3(0, 0, 0), colorTex.rgb, colorTex.a)
				}

				return colorTex
			}

			// This should be discard as well, rather than alpha 0
			discard()

		}

		`,
	))

	if err != nil {
		panic(err)
	}

	sprite3DShaderText := []byte(
		`package main

		var SpriteDepth float

		func decodeDepth(rgba vec4) float {
			return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
		}

		func Fragment(position vec4, texCoord vec2, color vec4) vec4 {

			resultDepth := imageSrc1UnsafeAt(texCoord)

			if resultDepth.a == 0 || decodeDepth(resultDepth) > SpriteDepth {
				return imageSrc0UnsafeAt(texCoord)
			}

			discard()

		}

		`,
	)

	cam.sprite3DShader, err = ebiten.NewShader(sprite3DShaderText)

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
	clone.OrthoScale = camera.OrthoScale

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
	}

	camera.resultAccumulatedColorTexture = ebiten.NewImage(w, h)
	camera.accumulatedBackBuffer = ebiten.NewImage(w, h)
	camera.resultColorTexture = ebiten.NewImage(w, h)
	camera.resultDepthTexture = ebiten.NewImage(w, h)
	camera.colorIntermediate = ebiten.NewImage(w, h)
	camera.depthIntermediate = ebiten.NewImage(w, h)
	camera.clipAlphaIntermediate = ebiten.NewImage(w, h)
	camera.sphereFactorCalculated = false

}

// Size returns the width and height of the camera's backing color texture. All of the Camera's textures are the same size, so these
// same size values can also be used for the depth texture, the accumulation buffer, etc.
func (camera *Camera) Size() (w, h int) {
	return camera.resultColorTexture.Size()
}

// ViewMatrix returns the Camera's view matrix.
func (camera *Camera) ViewMatrix() Matrix4 {

	camPos := camera.WorldPosition().Invert()
	transform := NewMatrix4Translate(camPos.X, camPos.Y, camPos.Z)

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
// does this by taking outVec, a vertex (Vector) that it stores the values in and returns, which avoids reallocation.
func (camera *Camera) clipToScreen(vert Vector, vertID int, model *Model, width, height float64) Vector {

	v3 := vert.W

	if !camera.Perspective {
		v3 = 1.0
	}

	// If the trangle is beyond the screen, we'll just pretend it's not and limit it to the closest possible value > 0
	// TODO: Replace this with triangle clipping or fix whatever graphical glitch seems to arise periodically
	if v3 < 0 {
		v3 = 0.000001
	}

	// Again, this function should only be called with pre-transformed 4D vertex arguments.

	vert.X = (vert.X/v3)*width + (width / 2)
	vert.Y = (vert.Y/v3*-1)*height + (height / 2)
	vert.Z = vert.Z / v3
	vert.W = 1

	if model != nil && model.VertexClipFunction != nil {
		vert = model.VertexClipFunction(vert, vertID)
	}

	return vert

}

// ClipToScreen projects the pre-transformed vertex in View space and remaps it to screen coordinates.
func (camera *Camera) ClipToScreen(vert Vector) Vector {
	width, height := camera.resultColorTexture.Size()
	return camera.clipToScreen(vert, 0, nil, float64(width), float64(height))
}

// WorldToScreen transforms a 3D position in the world to screen coordinates.
func (camera *Camera) WorldToScreen(vert Vector) Vector {
	v := NewMatrix4Translate(vert.X, vert.Y, vert.Z).Mult(camera.ViewMatrix().Mult(camera.Projection()))
	return camera.ClipToScreen(v.MultVecW(NewVectorZero()))
}

// WorldToClip transforms a 3D position in the world to clip coordinates (before screen normalization).
func (camera *Camera) WorldToClip(vert Vector) Vector {
	v := NewMatrix4Translate(vert.X, vert.Y, vert.Z).Mult(camera.ViewMatrix().Mult(camera.Projection()))
	return v.MultVecW(NewVectorZero())
}

// PointInFrustum returns true if the point is visible through the camera frustum.
func (camera *Camera) PointInFrustum(point Vector) bool {

	diff := point.Sub(camera.WorldPosition())
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

	diff := sphere.WorldPosition().Sub(camera.WorldPosition())
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
	camera.DebugInfo.BatchedParts = 0
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
		if model, ok := node.(*Model); ok && model.DynamicBatchOwner == nil {
			meshes = append(meshes, model)
		}
	}

	camera.Render(scene, meshes...)

}

type renderPair struct {
	Model    *Model
	MeshPart *MeshPart
}

// Bayer Matrix for transparency dithering
var bayerMatrix = []float32{
	1.0 / 17.0, 9.0 / 17.0, 3.0 / 17.0, 11.0 / 17.0,
	13.0 / 17.0, 5.0 / 17.0, 15.0 / 17.0, 7.0 / 17.0,
	4.0 / 17.0, 12.0 / 17.0, 2.0 / 17.0, 10.0 / 17.0,
	16.0 / 17.0, 8.0 / 17.0, 14.0 / 17.0, 6.0 / 17.0,
}

// Render renders all of the models passed using the provided Scene's properties (fog, for example). Note that if Camera.RenderDepth
// is false, scenes rendered one after another in multiple Render() calls will be rendered on top of each other in the Camera's texture buffers.
// Note that for Models, each MeshPart of a Model has a maximum renderable triangle count of 21845.
func (camera *Camera) Render(scene *Scene, models ...*Model) {

	frametimeStart := time.Now()

	sceneLights := []ILight{}
	lights := sceneLights

	if scene.World == nil || scene.World.LightingOn {

		for _, l := range scene.Root.ChildrenRecursive() {
			if light, isLight := l.(ILight); isLight {
				camera.DebugInfo.LightCount++
				if light.IsOn() {
					sceneLights = append(sceneLights, light)
					light.beginRender()
					camera.DebugInfo.ActiveLightCount++
				}
			}
		}

		if scene.World != nil && scene.World.AmbientLight != nil && scene.World.AmbientLight.IsOn() {
			sceneLights = append(sceneLights, scene.World.AmbientLight)
			scene.World.AmbientLight.beginRender()
			camera.DebugInfo.LightCount++
			camera.DebugInfo.ActiveLightCount++
		}

	}

	// By multiplying the camera's position against the view matrix (which contains the negated camera position), we're left with just the rotation
	// matrix, which we feed into model.TransformedVertices() to draw vertices in order of distance.
	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	rectShaderOptions := &ebiten.DrawRectShaderOptions{}
	rectShaderOptions.Images[0] = camera.colorIntermediate
	rectShaderOptions.Images[1] = camera.depthIntermediate

	if scene != nil && scene.World != nil {

		rectShaderOptions.Uniforms = map[string]interface{}{
			"Fog":         scene.World.fogAsFloatSlice(),
			"FogRange":    scene.World.FogRange,
			"DitherSize":  scene.World.DitheredFogSize,
			"BayerMatrix": bayerMatrix,
		}

	} else {

		rectShaderOptions.Uniforms = map[string]interface{}{
			"Fog":      []float32{0, 0, 0, 0},
			"FogRange": []float32{0, 1},
		}

	}

	// Reusing vectors rather than reallocating for all triangles for all models
	clipped := Vector{0, 0, 0, 0}

	solids := []renderPair{}
	transparents := []renderPair{}

	depths := map[*Model]float64{}

	for _, model := range models {

		if !model.visible {
			continue
		}

		if len(model.DynamicBatchModels) > 0 {

			dynamicDepths := map[*Model]float64{}

			transparent := false

			for meshPart, modelSlice := range model.DynamicBatchModels {

				for _, child := range modelSlice {

					if !child.visible {
						continue
					}

					dynamicDepths[child] = camera.WorldToScreen(child.WorldPosition()).Z

					if !transparent {

						for _, mp := range child.Mesh.MeshParts {

							if child.isTransparent(mp) {
								transparent = true
								break
							}

						}

					}

				}

				sort.Slice(modelSlice, func(i, j int) bool {
					return dynamicDepths[modelSlice[i]] > dynamicDepths[modelSlice[j]]
				})

				if transparent {
					transparents = append(transparents, renderPair{model, meshPart})
					depths[model] = camera.WorldToScreen(model.WorldPosition()).Z
				} else {
					solids = append(solids, renderPair{model, meshPart})
					if !camera.RenderDepth {
						depths[model] = camera.WorldToScreen(model.WorldPosition()).Z
					}
				}

			}

		} else if model.Mesh != nil {

			modelIsTransparent := false

			for _, mp := range model.Mesh.MeshParts {
				if model.isTransparent(mp) {
					transparents = append(transparents, renderPair{model, mp})
					modelIsTransparent = true
				} else {
					solids = append(solids, renderPair{model, mp})
				}
			}

			if !camera.RenderDepth || modelIsTransparent {
				depths[model] = camera.WorldToScreen(model.WorldPosition()).Z
			}

		}

	}

	// If the camera isn't rendering depth, then we should sort models by distance to ensure things draw in something like the correct order
	if !camera.RenderDepth {

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

		// startingVertexListIndex := vertexListIndex

		model := rp.Model

		// Models without Meshes are essentially just "nodes" that just have a position. They aren't counted for rendering.
		if model.Mesh == nil {
			return
		}

		meshPart := rp.MeshPart
		mat := meshPart.Material

		lighting := false
		if scene.World != nil {
			if mat != nil {
				lighting = scene.World.LightingOn && !mat.Shadeless
			} else {
				lighting = scene.World.LightingOn
			}
		}

		camera.DebugInfo.TotalParts++
		camera.DebugInfo.TotalTris += meshPart.TriangleCount()

		model.Transform()

		if model.FrustumCulling {

			if !camera.SphereInFrustum(model.BoundingSphere) {
				return
			}

		}

		if model.DynamicBatchOwner != nil {
			camera.DebugInfo.BatchedParts++
		}

		sortingTris := model.ProcessVertices(vpMatrix, camera, meshPart, scene)

		srcW := 0.0
		srcH := 0.0

		if mat != nil && mat.Texture != nil {
			srcW = float64(mat.Texture.Bounds().Dx())
			srcH = float64(mat.Texture.Bounds().Dy())
		}

		if lighting {

			t := time.Now()

			lights = sceneLights

			if model.LightGroup != nil && model.LightGroup.Active {
				lights = model.LightGroup.Lights
				for _, l := range model.LightGroup.Lights {
					l.beginRender() // Call this because it's relatively cheap and necessary if a light doesn't exist in the Scene
				}
			}

			for _, light := range lights {
				light.beginModel(model)
			}

			camera.DebugInfo.lightTime += time.Since(t)

		}

		mesh := model.Mesh

		maxSpan := model.Mesh.Dimensions.MaxSpan()
		modelPos := model.WorldPosition()

		// Here we do all vertex transforms first because of data locality (it's faster to access all vertex transformations, then go back and do all UV values, etc)

		mpColor := model.Color.Clone()

		if meshPart.Material != nil {
			mpColor.MultiplyRGBA(meshPart.Material.Color.ToFloat32s())
		}

		if lighting && len(sortingTris) > 0 {

			t := time.Now()

			meshPart.ForEachVertexIndex(func(vertIndex int) {
				meshPart.Mesh.vertexLights[vertIndex].Set(0, 0, 0, 1)
			})

			for _, light := range lights {

				// Skip calculating lighting for objects that are too far away from light sources.
				if point, ok := light.(*PointLight); ok && point.Distance > 0 {
					dist := maxSpan + point.Distance
					if modelPos.DistanceSquared(point.WorldPosition()) > dist*dist {
						continue
					}
				} else if cube, ok := light.(*CubeLight); ok && cube.Distance > 0 {
					dist := maxSpan + cube.Distance
					if modelPos.DistanceSquared(cube.WorldPosition()) > dist*dist {
						continue
					}
				}

				light.Light(meshPart, model, mesh.vertexLights)

			}

			camera.DebugInfo.lightTime += time.Since(t)

		}

		meshPart.ForEachVertexIndex(

			func(vertIndex int) {

				// visible := true

				// if !meshPart.sortingTriangles[t].rendered {
				// 	visible = false
				// }

				// triIndex := meshPart.sortingTriangles[t].ID
				// tri := meshPart.Mesh.Triangles[triIndex]

				clipped = camera.clipToScreen(meshPart.Mesh.vertexTransforms[vertIndex], vertIndex, model, float64(camWidth), float64(camHeight))

				// if visible {
				// 	renderedAnything = trueForEachVertexIndex
				// }

				// meshPart.sortingTriangles[t].rendered = visible

				colorVertexList[vertexListIndex].DstX = float32(clipped.X)
				colorVertexList[vertexListIndex].DstY = float32(clipped.Y)

				depthVertexList[vertexListIndex].DstX = float32(clipped.X)
				depthVertexList[vertexListIndex].DstY = float32(clipped.Y)

				depthVertexList[vertexListIndex].ColorR = 1
				depthVertexList[vertexListIndex].ColorG = 1
				depthVertexList[vertexListIndex].ColorB = 1
				depthVertexList[vertexListIndex].ColorA = 1

				// We set the UVs back here because we might need to use them if the material has clip alpha enabled.
				uvU := float32(mesh.VertexUVs[vertIndex].X * srcW)
				// We do 1 - v here (aka Y in texture coordinates) because 1.0 is the top of the texture while 0 is the bottom in UV coordinates,
				// but when drawing textures 0 is the top, and the sourceHeight is the bottom.
				uvV := float32((1 - mesh.VertexUVs[vertIndex].Y) * srcH)

				colorVertexList[vertexListIndex].SrcX = uvU
				colorVertexList[vertexListIndex].SrcY = uvV

				// Vertex colors

				if activeChannel := mesh.VertexActiveColorChannel[vertIndex]; activeChannel >= 0 {
					colorVertexList[vertexListIndex].ColorR = mesh.VertexColors[vertIndex][activeChannel].R * mpColor.R
					colorVertexList[vertexListIndex].ColorG = mesh.VertexColors[vertIndex][activeChannel].G * mpColor.G
					colorVertexList[vertexListIndex].ColorB = mesh.VertexColors[vertIndex][activeChannel].B * mpColor.B
					colorVertexList[vertexListIndex].ColorA = mesh.VertexColors[vertIndex][activeChannel].A * mpColor.A
				} else {
					colorVertexList[vertexListIndex].ColorR = mpColor.R
					colorVertexList[vertexListIndex].ColorG = mpColor.G
					colorVertexList[vertexListIndex].ColorB = mpColor.B
					colorVertexList[vertexListIndex].ColorA = mpColor.A
				}

				if lighting {
					colorVertexList[vertexListIndex].ColorR *= mesh.vertexLights[vertIndex].R
					colorVertexList[vertexListIndex].ColorG *= mesh.vertexLights[vertIndex].G
					colorVertexList[vertexListIndex].ColorB *= mesh.vertexLights[vertIndex].B
				}

				if camera.RenderDepth {

					// We're adding 0.03 for a margin because for whatever reason, at close range / wide FOV,
					// depth can be negative but still be in front of the camera and not behind it.
					margin := 1.0
					depth := mesh.vertexTransforms[vertIndex].Z / (far + margin)
					if depth < 0 {
						depth = 0
					} else if depth > 1 {
						depth = 1
					}

					depthVertexList[vertexListIndex].ColorR = float32(depth)
					depthVertexList[vertexListIndex].ColorG = float32(depth)
					depthVertexList[vertexListIndex].ColorB = float32(depth)
					depthVertexList[vertexListIndex].ColorA = 1

					// We set the UVs back here because we might need to use them if the material has clip alpha enabled.
					depthVertexList[vertexListIndex].SrcX = uvU

					// We do 1 - v here (aka Y in texture coordinates) because 1.0 is the top of the texture while 0 is the bottom in UV coordinates,
					// but when drawing textures 0 is the top, and the sourceHeight is the bottom.
					depthVertexList[vertexListIndex].SrcY = uvV

				} else if scene.World != nil && scene.World.FogMode != FogOff {

					// We're adding 0.03 for a margin because for whatever reason, at close range / wide FOV,
					// depth can be negative but still be in front of the camera and not behind it.
					depth := float32((mesh.vertexTransforms[vertIndex].Z+near)/far + 0.03)
					if depth < 0 {
						depth = 0
					} else if depth > 1 {
						depth = 1
					}

					// depth = 1 - depth

					depth = scene.World.FogRange[0] + ((scene.World.FogRange[1]-scene.World.FogRange[0])*1 - depth)

					if scene.World.FogMode == FogAdd {
						colorVertexList[vertexListIndex].ColorR += scene.World.FogColor.R * depth
						colorVertexList[vertexListIndex].ColorG += scene.World.FogColor.G * depth
						colorVertexList[vertexListIndex].ColorB += scene.World.FogColor.B * depth
					} else if scene.World.FogMode == FogMultiply {
						colorVertexList[vertexListIndex].ColorR *= scene.World.FogColor.R * depth
						colorVertexList[vertexListIndex].ColorG *= scene.World.FogColor.G * depth
						colorVertexList[vertexListIndex].ColorB *= scene.World.FogColor.B * depth
					}

				}

				vertexListIndex++

			},
		)

		if vertexListIndex == 0 {
			return
		}

		for _, sortingTri := range sortingTris {

			for _, index := range sortingTri.Triangle.VertexIndices {
				indexList[indexListIndex] = uint16(index - sortingTri.Triangle.MeshPart.VertexIndexStart + indexListStart)
				indexListIndex++
			}

		}

		indexListStart = vertexListIndex

		// for i := 0; i < vertexListIndex; i++ {
		// 	indexList[i] = uint16(i)
		// }

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

				camera.clipAlphaIntermediate.DrawTrianglesShader(depthVertexList[:vertexListIndex], indexList[:indexListIndex], camera.clipAlphaRenderShader, &ebiten.DrawTrianglesShaderOptions{Images: [4]*ebiten.Image{img}})

				w, h := camera.depthIntermediate.Size()

				camera.depthIntermediate.DrawRectShader(w, h, camera.clipAlphaCompositeShader, &ebiten.DrawRectShaderOptions{Images: [4]*ebiten.Image{camera.resultDepthTexture, camera.clipAlphaIntermediate}})

			} else {
				shaderOpt := &ebiten.DrawTrianglesShaderOptions{
					Images: [4]*ebiten.Image{camera.resultDepthTexture},
				}

				camera.depthIntermediate.DrawTrianglesShader(depthVertexList[:vertexListIndex], indexList[:indexListIndex], camera.depthShader, shaderOpt)
			}

			if !model.isTransparent(meshPart) {
				camera.resultDepthTexture.DrawImage(camera.depthIntermediate, nil)
			}

		}

		t := &ebiten.DrawTrianglesOptions{}
		if model.ColorBlendingFunc != nil {
			t.ColorM = model.ColorBlendingFunc(model, meshPart) // Modify the model's appearance using its color blending function
		}
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
				camera.colorIntermediate.DrawTrianglesShader(colorVertexList[:vertexListIndex], indexList[:indexListIndex], mat.fragmentShader, mat.FragmentShaderOptions)
			} else {
				camera.colorIntermediate.DrawTriangles(colorVertexList[:vertexListIndex], indexList[:indexListIndex], img, t)
			}

			camera.resultColorTexture.DrawRectShader(w, h, camera.colorShader, rectShaderOptions)

		} else {

			if mat != nil {
				t.CompositeMode = mat.CompositeMode
			}

			if hasFragShader {
				camera.resultColorTexture.DrawTrianglesShader(colorVertexList[:vertexListIndex], indexList[:indexListIndex], mat.fragmentShader, mat.FragmentShaderOptions)
			} else {
				camera.resultColorTexture.DrawTriangles(colorVertexList[:vertexListIndex], indexList[:indexListIndex], img, t)
			}

		}

		camera.DebugInfo.DrawnTris += indexListIndex / 3
		camera.DebugInfo.DrawnParts++

		vertexListIndex = 0
		indexListIndex = 0
		indexListStart = 0

	}

	for _, pair := range solids {

		if !pair.Model.visible {
			continue
		}

		// Internally, the idea behind dynamic batching is that we simply hold off on flushing until the
		// end - this saves a lot of time if we're rendering singular low-poly objects, at the cost of each
		// object sharing the same material / object-level properties (color / material blending mode, for
		// example).
		if dyn := pair.Model.DynamicBatchModels; len(dyn) > 0 {

			modelSlice := dyn[pair.MeshPart]

			for _, merged := range modelSlice {

				if !merged.visible {
					continue
				}

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

			if !pair.Model.visible {
				continue
			}

			if dyn := pair.Model.DynamicBatchModels; len(dyn) > 0 {

				modelSlice := dyn[pair.MeshPart]

				for _, merged := range modelSlice {

					if !merged.visible {
						continue
					}

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

func encodeDepth(depth float64) *Color {

	r := math.Floor(depth*255) / 255
	_, f := math.Modf(depth * 255)
	g := math.Floor(f*255) / 255
	_, f = math.Modf(depth * 255 * 255)
	b := f

	if r < 0 {
		r = 0
	} else if r > 1 {
		r = 1
	}
	if g < 0 {
		g = 0
	} else if g > 1 {
		g = 1
	}
	if b < 0 {
		b = 0
	} else if b > 1 {
		b = 1
	}

	return NewColor(float32(r), float32(g), float32(b), 1)

}

type SpriteRender3d struct {
	Image         *ebiten.Image
	Options       *ebiten.DrawImageOptions
	WorldPosition Vector
}

// DrawImageIn3D draws an image on the screen in 2D, but at the screen position of the 3D world position provided
// and with depth intersection.
// This allows you to render 2D elements "at" a 3D position, and can be very useful in situations where you want
// a sprite to render at 100% size and no perspective or skewing, but still look like it's in the 3D space (like in
// a game with a fixed camera viewpoint).
func (camera *Camera) DrawImageIn3D(screen *ebiten.Image, renderSettings ...SpriteRender3d) {

	// TODO: Replace this with a more performant alternative, where we minimize shader / texture switches.

	for _, rs := range renderSettings {

		camera.colorIntermediate.Clear()

		px := camera.WorldToScreen(rs.WorldPosition)

		out := camera.ViewMatrix().Mult(camera.Projection()).MultVec(rs.WorldPosition)

		depth := float32(out.Z / camera.Far)

		if depth < 0 {
			depth = 0
		}
		if depth > 1 {
			depth = 1
		}

		opt := rs.Options

		if opt == nil {
			opt = &ebiten.DrawImageOptions{}
		}
		opt.GeoM.Translate(px.X, px.Y)

		camera.colorIntermediate.DrawImage(rs.Image, opt)

		colorTextureW, colorTextureH := camera.resultColorTexture.Size()
		rectShaderOptions := &ebiten.DrawRectShaderOptions{}
		rectShaderOptions.Images[0] = camera.colorIntermediate
		rectShaderOptions.Images[1] = camera.resultDepthTexture
		rectShaderOptions.Uniforms = map[string]interface{}{
			"SpriteDepth": depth,
		}
		screen.DrawRectShader(colorTextureW, colorTextureH, camera.sprite3DShader, rectShaderOptions)

	}

}

func (camera *Camera) drawCircle(screen *ebiten.Image, position Vector, radius float64, drawColor color.Color) {

	transformedCenter := camera.WorldToScreen(position)

	stepCount := float64(32)

	for i := 0; i < int(stepCount); i++ {

		x := (math.Sin(math.Pi*2*float64(i)/stepCount) * radius)
		y := (math.Cos(math.Pi*2*float64(i)/stepCount) * radius)

		x2 := (math.Sin(math.Pi*2*float64(i+1)/stepCount) * radius)
		y2 := (math.Cos(math.Pi*2*float64(i+1)/stepCount) * radius)

		ebitenutil.DrawLine(screen, transformedCenter.X+x, transformedCenter.Y+y, transformedCenter.X+x2, transformedCenter.Y+y2, drawColor)
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
		"TPS: %f\nFPS: %f\nTotal render frame-time: %s\nSkinned mesh animation time: %s\nLighting frame-time: %s\nDraw calls: %d/%d (%d batched)\nRendered triangles: %d/%d\nActive Lights: %d/%d",
		ebiten.CurrentTPS(),
		ebiten.CurrentFPS(),
		ft,
		at,
		lt,
		camera.DebugInfo.DrawnParts,
		camera.DebugInfo.TotalParts,
		camera.DebugInfo.BatchedParts,
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

				meshPart.ForEachTri(func(tri *Triangle) {

					indices := tri.VertexIndices
					v0 := camera.ClipToScreen(model.Mesh.vertexTransforms[indices[0]])
					v1 := camera.ClipToScreen(model.Mesh.vertexTransforms[indices[1]])
					v2 := camera.ClipToScreen(model.Mesh.vertexTransforms[indices[2]])

					if (v0.X < 0 && v1.X < 0 && v2.X < 0) ||
						(v0.Y < 0 && v1.Y < 0 && v2.Y < 0) ||
						(v0.X > float64(camWidth) && v1.X > float64(camWidth) && v2.X > float64(camWidth)) ||
						(v0.Y > float64(camHeight) && v1.Y > float64(camHeight) && v2.Y > float64(camHeight)) {
						return
					}

					c := color.ToRGBA64()
					ebitenutil.DrawLine(screen, float64(v0.X), float64(v0.Y), float64(v1.X), float64(v1.Y), c)
					ebitenutil.DrawLine(screen, float64(v1.X), float64(v1.Y), float64(v2.X), float64(v2.Y), c)
					ebitenutil.DrawLine(screen, float64(v2.X), float64(v2.Y), float64(v0.X), float64(v0.Y), c)

				})

			}

		} else if path, isPath := m.(*Path); isPath {

			points := path.Children()
			if path.Closed {
				points = append(points, points[0])
			}
			for i := 0; i < len(points)-1; i++ {
				p1 := camera.WorldToScreen(points[i].WorldPosition())
				p2 := camera.WorldToScreen(points[i+1].WorldPosition())
				ebitenutil.DrawLine(screen, p1.X, p1.Y, p2.X, p2.Y, color.ToRGBA64())
			}

		} else if grid, isGrid := m.(*Grid); isGrid {

			for _, point := range grid.Points() {
				p1 := camera.WorldToScreen(point.WorldPosition())
				ebitenutil.DrawCircle(screen, p1.X, p1.Y, 8, color.ToRGBA64())
				for _, connection := range point.Connections {
					p2 := camera.WorldToScreen(connection.WorldPosition())
					ebitenutil.DrawLine(screen, p1.X, p1.Y, p2.X, p2.Y, color.ToRGBA64())
				}
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

				for triIndex, sortingTri := range meshPart.sortingTriangles {

					screenPos := camera.WorldToScreen(model.Transform().MultVec(sortingTri.Triangle.Center))

					camera.DebugDrawText(screen, fmt.Sprintf("%d", triIndex), screenPos.X, screenPos.Y+(textScale*16), textScale, color)

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

			camera.DebugDrawText(screen, fmt.Sprintf("%d", len(model.Mesh.MeshParts)), screenPos.X, screenPos.Y+(textScale*16), textScale, color)

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
				ebitenutil.DrawLine(screen, center.X, center.Y, transformedNormal.X, transformedNormal.Y, color.ToRGBA64())

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
			ebitenutil.DrawLine(screen, pos.X, pos.Y, parentPos.X, parentPos.Y, c)
		}

	}

}

func (camera *Camera) DebugDrawText(screen *ebiten.Image, txtStr string, posX, posY, textScale float64, color *Color) {

	size := text.BoundString(basicfont.Face7x13, txtStr).Size()

	if camera.debugTextTexture == nil || size.X > camera.debugTextTexture.Bounds().Dx() || size.Y > camera.debugTextTexture.Bounds().Dy() {
		camera.debugTextTexture = ebiten.NewImage(size.X, size.Y+13)
	}

	camera.debugTextTexture.Clear()

	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(0, 13)
	text.DrawWithOptions(camera.debugTextTexture, txtStr, basicfont.Face7x13, opt)

	// TODO: This is slow, this could be way faster by drawing once to an image and then drawing that result

	dr := &ebiten.DrawImageOptions{}
	dr.ColorM.Scale(0, 0, 0, 1)

	periodOffset := 8.0

	for y := -1; y < 2; y++ {

		for x := -1; x < 2; x++ {

			dr.GeoM.Reset()
			dr.GeoM.Translate(posX+4+float64(x), posY+4+float64(y)+periodOffset)
			dr.GeoM.Scale(textScale, textScale)

			screen.DrawImage(camera.debugTextTexture, dr)
		}

	}

	dr.ColorM.Reset()
	dr.ColorM.Scale(color.ToFloat64s())

	dr.GeoM.Reset()
	dr.GeoM.Translate(posX+4, posY+4+periodOffset)
	dr.GeoM.Scale(textScale, textScale)

	screen.DrawImage(camera.debugTextTexture, dr)

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
// be drawn in the color provided for each kind of bounding object to the screen image provided. If the passed color is nil, that kind of shape
// won't be debug-rendered.
func (camera *Camera) DrawDebugBoundsColored(screen *ebiten.Image, rootNode INode, aabbColor, sphereColor, capsuleColor, trianglesColor, trianglesAABBColor, trianglesBroadphaseColor *Color) {

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	for _, n := range allModels {

		if b, isBounds := n.(IBoundingObject); isBounds {

			switch bounds := b.(type) {

			case *BoundingSphere:

				if sphereColor != nil {
					camera.drawSphere(screen, bounds, sphereColor)
				}

			case *BoundingCapsule:

				if capsuleColor != nil {

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

					lines := []Vector{
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
						ebitenutil.DrawLine(screen, start.X, start.Y, end.X, end.Y, capsuleColor.ToRGBA64())

					}

				}

			case *BoundingAABB:

				if aabbColor != nil {

					pos := bounds.WorldPosition()
					size := bounds.Dimensions.Size().Scale(0.5)

					ufr := camera.WorldToScreen(pos.Add(Vector{size.X, size.Y, size.Z, 0}))
					ufl := camera.WorldToScreen(pos.Add(Vector{-size.X, size.Y, size.Z, 0}))
					ubr := camera.WorldToScreen(pos.Add(Vector{size.X, size.Y, -size.Z, 0}))
					ubl := camera.WorldToScreen(pos.Add(Vector{-size.X, size.Y, -size.Z, 0}))

					dfr := camera.WorldToScreen(pos.Add(Vector{size.X, -size.Y, size.Z, 0}))
					dfl := camera.WorldToScreen(pos.Add(Vector{-size.X, -size.Y, size.Z, 0}))
					dbr := camera.WorldToScreen(pos.Add(Vector{size.X, -size.Y, -size.Z, 0}))
					dbl := camera.WorldToScreen(pos.Add(Vector{-size.X, -size.Y, -size.Z, 0}))

					lines := []Vector{
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
						ebitenutil.DrawLine(screen, start.X, start.Y, end.X, end.Y, aabbColor.ToRGBA64())

					}

				}

			case *BoundingTriangles:

				if trianglesBroadphaseColor != nil {

					for _, b := range bounds.Broadphase.allAABBPositions() {
						camera.DrawDebugBoundsColored(screen, b, trianglesBroadphaseColor, nil, nil, nil, nil, nil)
					}

				}

				if trianglesColor != nil {

					camWidth, camHeight := camera.resultColorTexture.Size()

					lines := []Vector{}

					mesh := bounds.Mesh

					for _, tri := range mesh.Triangles {

						mvpMatrix := bounds.Transform().Mult(camera.ViewMatrix().Mult(camera.Projection()))

						v0 := camera.ClipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.VertexIndices[0]]))
						v1 := camera.ClipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.VertexIndices[1]]))
						v2 := camera.ClipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.VertexIndices[2]]))

						if (v0.X < 0 && v1.X < 0 && v2.X < 0) ||
							(v0.Y < 0 && v1.Y < 0 && v2.Y < 0) ||
							(v0.X > float64(camWidth) && v1.X > float64(camWidth) && v2.X > float64(camWidth)) ||
							(v0.Y > float64(camHeight) && v1.Y > float64(camHeight) && v2.Y > float64(camHeight)) {
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
						ebitenutil.DrawLine(screen, start.X, start.Y, end.X, end.Y, triColor)

						start = lines[i+1]
						end = lines[i+2]
						ebitenutil.DrawLine(screen, start.X, start.Y, end.X, end.Y, triColor)

						start = lines[i+2]
						end = lines[i]
						ebitenutil.DrawLine(screen, start.X, start.Y, end.X, end.Y, triColor)

					}

				}

				if trianglesAABBColor != nil {
					camera.DrawDebugBoundsColored(screen, bounds.BoundingAABB, trianglesAABBColor, nil, nil, nil, nil, nil)
				}

			}

		}

	}

}

var defaultAABBColor = NewColor(0, 0.25, 1, 1)
var defaultSphereColor = NewColor(0.5, 0.25, 1.0, 1)
var defaultCapsuleColor = NewColor(0.25, 1, 0, 1)
var defaultTrianglesColor = NewColor(0, 0, 0, 0.5)
var defaultTrianglesBroadphaseColor = NewColor(1, 0, 0, 0.25)

// DrawDebugBounds will draw shapes approximating the shapes and positions of BoundingObjects underneath the rootNode. The shapes will
// be drawn using default colors to the screen image provided. renderBroadphaseCells indicates whether the broadphase cells for triangle
// objects should be rendered; similarly, renderTriangleAABB handles whether AABB bounding shapes for triangles should be rendered.
func (camera *Camera) DrawDebugBounds(screen *ebiten.Image, rootNode INode, renderBroadphaseCells, renderTriangleAABB bool) {
	var broadphaseColor *Color
	var trianglesAABBColor *Color

	if renderBroadphaseCells {
		broadphaseColor = defaultTrianglesBroadphaseColor
	}
	if renderTriangleAABB {
		trianglesAABBColor = defaultAABBColor
	}

	camera.DrawDebugBoundsColored(screen, rootNode, defaultAABBColor, defaultSphereColor, defaultCapsuleColor, defaultTrianglesColor, trianglesAABBColor, broadphaseColor)
}

// DrawDebugFrustums will draw shapes approximating the frustum spheres for objects underneath the rootNode.
// The shapes will be drawn in the color provided to the screen image provided.
func (camera *Camera) DrawDebugFrustums(screen *ebiten.Image, rootNode INode, color *Color) {

	allModels := append([]INode{rootNode}, rootNode.ChildrenRecursive()...)

	for _, n := range allModels {

		if model, ok := n.(*Model); ok {

			bounds := model.BoundingSphere
			camera.drawSphere(screen, bounds, color)

		}

	}

}

var debugIcosphereMesh = NewIcosphere(1)
var debugIcosphere = NewModel(debugIcosphereMesh, "debug icosphere")

func (camera *Camera) drawSphere(screen *ebiten.Image, sphere *BoundingSphere, color *Color) {

	debugIcosphere.SetLocalPositionVec(sphere.WorldPosition())
	s := sphere.WorldRadius()
	debugIcosphere.SetLocalScaleVec(Vector{s, s, s, 0})
	debugIcosphere.Transform()
	camera.DrawDebugWireframe(screen, debugIcosphere, color)

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
