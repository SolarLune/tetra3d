package tetra3d

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/solarlune/tetra3d/math32"
	"golang.org/x/image/font/basicfont"
)

// DebugInfo is a struct that holds debugging information for a Camera's render pass. These values are reset when Camera.Clear() is called.
type DebugInfo struct {
	FrameTime            time.Duration // Amount of CPU frame time spent transforming vertices and calling Image.DrawTriangles. Doesn't include time ebitengine spends flushing the command queue.
	AnimationTime        time.Duration // Amount of CPU frame time spent animating vertices.
	LightTime            time.Duration // Amount of CPU frame time spent lighting vertices.
	currentAnimationTime time.Duration
	currentLightTime     time.Duration
	currentFrameTime     time.Duration
	tickTime             time.Time
	DrawnParts           int // Number of draw calls, excluding those invisible or culled based on distance
	TotalParts           int // Total number of draw calls
	BatchedParts         int // Total batched number of draw calls
	DrawnTris            int // Number of drawn triangles, excluding those hidden from backface culling
	TotalTris            int // Total number of triangles
	LightCount           int // Total number of lights
	ActiveLightCount     int // Total active number of lights
}

type AccumulationColorMode int

const (
	AccumulationColorModeNone            AccumulationColorMode = iota // No accumulation buffer rendering
	AccumulationColorModeBelow                                        // Accumulation buffer is on and applies over time, renders ColorTexture after the accumulation result (which is, then, below)
	AccumulationColorModeAbove                                        // Accumulation buffer is on and applies over time, renders ColorTexture before the accumulation result (which is on top)
	AccumulationColorModeSingleLastFrame                              // Accumulation buffer is on and renders just the previous frame's ColorTexture result
)

// Camera represents a camera (where you look from) in Tetra3D.
type Camera struct {
	*Node

	RenderDepth                        bool // If the Camera should attempt to render a depth texture; if this is true, then DepthTexture() will hold the depth texture render results. Defaults to true.
	RenderNormals                      bool // If the Camera should attempt to render a normal texture; if this is true, then NormalTexture() will hold the normal texture render results. Defaults to false.
	SectorRendering                    bool // If the Camera should render using sectors or not; if no sectors are present, then it won't attempt to render with them. Defaults to false.
	SectorRenderDepth                  int  // How far out the Camera renders other sectors. Defaults to 1 (so the current sector and its immediate neighbors).
	PerspectiveCorrectedTextureMapping bool // If the Camera should render textures with perspective corrected texture mapping. Defaults to false.
	currentSector                      *Sector
	// How many lights (sorted by distance) should be used to render each object, maximum. If it's greater than 0,
	// then only that many lights will be considered. If less than or equal to 0 (the default), then all available lights will be used.
	MaxLightCount           int
	MaxLightCountFocusPoint INode // The focus point for determining light culling based on distance from the camera; when this is nil, the camera's world position is used

	resultColorTexture  *ebiten.Image // ColorTexture holds the color results of rendering any models.
	resultDepthTexture  *ebiten.Image // DepthTexture holds the depth results of rendering any models, if Camera.RenderDepth is on.
	resultNormalTexture *ebiten.Image // NormalTexture holds a texture indicating the normal render
	depthIntermediate   *ebiten.Image

	resultAccumulatedColorTexture *ebiten.Image // ResultAccumulatedColorTexture holds the previous frame's render result of rendering any models.
	accumulatedBackBuffer         *ebiten.Image
	// The mode to use when rendering previous frames to the accumulation buffer.
	// When set to anything but AccumulationColorModeNone, clearing the Camera will copy the previous frame's color texture render result to the accumulation buffer.
	// The AccumulationColorMode influences in what order the color texture and previous frame's acccumulation color mode result is drawn to the buffer.
	// Defaults to AccumulateColorModeNone.
	AccumulationColorMode AccumulationColorMode
	// Draw image options to use when rendering frames to the accumulation buffer; use this to fade out or color previous frames.
	// This should probably be set once, or once per frame before rendering; otherwise, the effects compound and it's impossible to see the result.
	AccumulationDrawOptions *ebiten.DrawImageOptions

	near, far   float32 // The near and far clipping plane. Near defaults to 0.1, Far to 100 (unless these settings are loaded from a camera in a GLTF file).
	perspective bool    // If the Camera has a perspective projection. If not, it would be orthographic
	fieldOfView float32 // Vertical field of view in degrees for a perspective projection camera
	orthoScale  float32 // Scale of the view for an orthographic projection camera in units horizontally

	// When set to a value > 0, it will snap all rendered models' vertices to a grid of the provided size (so VertexSnapping of 0.1 will snap all rendered positions to 0.1 intervals).
	// Note that snapping vertices is not free and does take some time to execute (so if you're up against a performance limit, turning off vertex snapping might make a bit of a difference).
	// Defaults to 0 (off).
	VertexSnapping float32

	DebugInfo        DebugInfo
	PrevFrametimes   []float32
	PrevDebugInfoCap int

	depthShader     *ebiten.Shader
	clipAlphaShader *ebiten.Shader
	colorShader     *ebiten.Shader
	sprite3DShader  *ebiten.Shader

	// Visibility check variables
	cameraForward          Vector3
	cameraRight            Vector3
	cameraUp               Vector3
	sphereFactorX          float32
	sphereFactorY          float32
	sphereFactorTang       float32
	sphereFactorCalculated bool
	updateProjectionMatrix bool
	cachedProjectionMatrix Matrix4

	debugTextTexture *ebiten.Image

	// DepthMargin is a margin in percentage on both the near and far plane to leave some
	// distance remaining in the depth buffer for triangle comparison.
	// This ensures that objects that are, for example, close to or far from the camera
	// don't begin Z-fighting unnecessarily. It defaults to 4% (on both ends).
	DepthMargin float32
}

// NewCamera creates a new Camera with the specified width and height.
func NewCamera(w, h int) *Camera {

	cam := &Camera{
		Node:        NewNode("Camera"),
		RenderDepth: true,
		near:        0.1,
		far:         100,

		DepthMargin: 0.04,

		orthoScale: 20,

		SectorRendering:   false,
		SectorRenderDepth: 1,

		PrevDebugInfoCap: 20,
	}

	cam.owner = cam

	depthShaderText := []byte(
		`package main

		//kage:unit pixels

		func encodeDepth(depth float) vec4 {
			r := floor(depth * 255) / 255
			g := floor(fract(depth * 255) * 255) / 255
			b := fract(depth * 255*255)
			return vec4(r, g, b, 1);
		}

		func decodeDepth(rgba vec4) float {
			return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
		}

		func dstPosToSrcPos(dstPos vec2) vec2 {
			return dstPos.xy - imageDstOrigin() + imageSrc0Origin()
		}

		func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {

			existingDepth := imageSrc0UnsafeAt(dstPosToSrcPos(dstPos.xy))

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
		`//kage:unit pixels
		package main

		var PerspectiveCorrection int
		var TextureMapMode int
		var TextureMapScreenSize float

		func encodeDepth(depth float) vec4 {
			r := floor(depth * 255) / 255
			g := floor(fract(depth * 255) * 255) / 255
			b := fract(depth * 255*255)
			return vec4(r, g, b, 1);
		}

		func decodeDepth(rgba vec4) float {
			return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
		}

		func dstPosToSrcPos(dstPos vec2) vec2 {
			return dstPos.xy - imageDstOrigin() + imageSrc0Origin()
		}

		func Fragment(dstPos vec4, srcPos vec2, vc, custom vec4) vec4 {

			color := vc
			srcSize := imageSrc1Size()

			// There's atlassing going on behind the scenes here, so:
			// Subtract the source position by the src texture's origin on the atlas.
			// This gives us the actual pixel coordinates.
			tx := srcPos

			if TextureMapMode == 1 {
				tx = (dstPos.xy * TextureMapScreenSize) + (srcPos - imageSrc0Origin())
			} else if PerspectiveCorrection > 0 {
				tx *= 1.0 / custom.x
			}

			// Divide by the source image size to get the UV coordinates.
			tx /= srcSize

			// Apply fract() to loop the UV coords around [0-1].
			tx = fract(tx)

			// Multiply by the size to get the pixel coordinates again.
			tx *= srcSize

			tex := imageSrc1UnsafeAt(tx)

			if (tex.a == 0) {
				discard()
			}

			depthValue := imageSrc0UnsafeAt(dstPosToSrcPos(dstPos.xy))

			if depthValue.a == 0 || decodeDepth(depthValue) > color.r {
				return vec4(encodeDepth(color.r).rgb, tex.a)
			}

			discard()

		}

		`,
	)

	cam.clipAlphaShader, err = ebiten.NewShader(clipAlphaShaderText)

	if err != nil {
		panic(err)
	}

	cam.colorShader, err = ExtendBase3DShader("")

	if err != nil {
		panic(err)
	}

	sprite3DShaderText := []byte(
		`package main
		//kage:unit pixels

		var SpriteDepth float

		func decodeDepth(rgba vec4) float {
			return rgba.r + (rgba.g / 255) + (rgba.b / 65025)
		}

		func dstPosToSrcPos(dstPos vec2) vec2 {
			return dstPos.xy - imageDstOrigin() + imageSrc0Origin()
		}

		func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {

			resultDepth := imageSrc1UnsafeAt(dstPosToSrcPos(dstPos.xy))

			if resultDepth.a == 0 || decodeDepth(resultDepth) > SpriteDepth {
				return imageSrc0UnsafeAt(srcPos)
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

	cam.SetPerspective(true)
	cam.SetFieldOfView(60)

	return cam
}

// Clone clones the Camera and returns it.
func (camera *Camera) Clone() INode {

	clone := NewCamera(camera.Size())

	clone.RenderDepth = camera.RenderDepth
	clone.near = camera.near
	clone.far = camera.far
	clone.perspective = camera.perspective
	clone.fieldOfView = camera.fieldOfView
	clone.orthoScale = camera.orthoScale
	clone.SectorRendering = camera.SectorRendering
	clone.SectorRenderDepth = camera.SectorRenderDepth
	clone.PerspectiveCorrectedTextureMapping = camera.PerspectiveCorrectedTextureMapping

	clone.AccumulationColorMode = camera.AccumulationColorMode
	if camera.AccumulationDrawOptions != nil {
		newOptions := *camera.AccumulationDrawOptions
		clone.AccumulationDrawOptions = &newOptions
	}

	clone.Node = camera.Node.clone(clone).(*Node)
	clone.MaxLightCount = camera.MaxLightCount
	if camera.MaxLightCountFocusPoint != nil {
		clone.MaxLightCountFocusPoint = camera.MaxLightCountFocusPoint
	}

	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}

	return clone

}

// Resize resizes the backing texture for the Camera to the specified width and height. If the camera already has a backing texture and the
// width and height are already set to the specified arguments, then the function does nothing.
func (camera *Camera) Resize(w, h int) {

	if camera.resultColorTexture != nil {

		origW, origH := camera.Size()
		if w == origW && h == origH {
			return
		}

		camera.resultColorTexture.Dispose()
		camera.resultAccumulatedColorTexture.Dispose()
		camera.resultNormalTexture.Dispose()
		camera.accumulatedBackBuffer.Dispose()
		camera.resultDepthTexture.Dispose()
		camera.depthIntermediate.Dispose()
	}

	bounds := image.Rect(0, 0, w, h)
	opt := &ebiten.NewImageOptions{
		Unmanaged: true,
	}
	camera.resultAccumulatedColorTexture = ebiten.NewImageWithOptions(bounds, opt)
	camera.accumulatedBackBuffer = ebiten.NewImageWithOptions(bounds, opt)
	camera.resultColorTexture = ebiten.NewImageWithOptions(bounds, opt)
	camera.resultDepthTexture = ebiten.NewImageWithOptions(bounds, opt)
	camera.resultNormalTexture = ebiten.NewImageWithOptions(bounds, opt)
	camera.depthIntermediate = ebiten.NewImageWithOptions(bounds, opt)
	camera.sphereFactorCalculated = false
	camera.updateProjectionMatrix = true

}

// Size returns the width and height of the camera's backing color texture. All of the Camera's textures are the same size, so these
// same size values can also be used for the depth texture, the accumulation buffer, etc.
func (camera *Camera) Size() (w, h int) {
	size := camera.resultColorTexture.Bounds().Size()
	return size.X, size.Y
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

	if !camera.updateProjectionMatrix {
		return camera.cachedProjectionMatrix
	}

	camera.updateProjectionMatrix = false

	if !camera.sphereFactorCalculated {
		angle := camera.fieldOfView * 3.1415 / 360
		camera.sphereFactorTang = math32.Tan(angle)
		camera.sphereFactorY = 1.0 / math32.Cos(angle)
		camera.sphereFactorX = 1.0 / math32.Cos(math32.Atan(camera.sphereFactorTang*camera.AspectRatio()))
		camera.sphereFactorCalculated = true
	}

	if camera.perspective {
		camera.cachedProjectionMatrix = NewProjectionPerspective(camera.fieldOfView, camera.near, camera.far, float32(camera.resultColorTexture.Bounds().Dx()), float32(camera.resultColorTexture.Bounds().Dy()))
	} else {

		w, h := camera.resultColorTexture.Size()
		asr := float32(h) / float32(w)

		camera.cachedProjectionMatrix = NewProjectionOrthographic(camera.near, camera.far, 1*camera.orthoScale, -1*camera.orthoScale, asr*camera.orthoScale, -asr*camera.orthoScale)
	}

	return camera.cachedProjectionMatrix
	// return NewProjectionOrthographic(camera.Near, camera.far, float32(camera.ColorTexture.Bounds().Dx())*camera.orthoScale, float32(camera.ColorTexture.Bounds().Dy())*camera.orthoScale)
}

// SetPerspective sets the Camera's projection to be a perspective (true) or orthographic (false) projection.
func (camera *Camera) SetPerspective(perspective bool) {
	if camera.perspective == perspective {
		return
	}
	camera.perspective = perspective
	camera.sphereFactorCalculated = false
	camera.updateProjectionMatrix = true
}

// Perspective returns whether the Camera is perspective or not (orthographic).
func (camera *Camera) Perspective() bool {
	return camera.perspective
}

// SetFieldOfView sets the vertical field of the view of the camera in degrees.
func (camera *Camera) SetFieldOfView(fovY float32) {
	if camera.fieldOfView == fovY {
		return
	}
	camera.fieldOfView = fovY
	camera.sphereFactorCalculated = false
	camera.updateProjectionMatrix = true
}

// FieldOfView returns the vertical field of view in degrees.
func (camera *Camera) FieldOfView() float32 {
	return camera.fieldOfView
}

// SetOrthoScale sets the scale of an orthographic camera in world units across (horizontally).
func (camera *Camera) SetOrthoScale(scale float32) {
	if camera.orthoScale == scale {
		return
	}
	camera.orthoScale = scale
	camera.sphereFactorCalculated = false
	camera.updateProjectionMatrix = true
}

// OrthoScale returns the scale of an orthographic camera in world units across (horizontally).
func (camera *Camera) OrthoScale() float32 {
	return camera.orthoScale
}

// Near returns the near plane of a camera.
func (camera *Camera) Near() float32 {
	return camera.near
}

// SetNear sets the near plane of a camera.
func (camera *Camera) SetNear(near float32) {
	if camera.near == near {
		return
	}
	camera.near = near
	camera.sphereFactorCalculated = false
	camera.updateProjectionMatrix = true
}

// Far returns the far plane of a camera.
func (camera *Camera) Far() float32 {
	return camera.far
}

// SetFar sets the far plane of the camera.
func (camera *Camera) SetFar(far float32) {
	if camera.far == far {
		return
	}
	camera.far = far
	camera.sphereFactorCalculated = false
	camera.updateProjectionMatrix = true
}

// We do this for each vertex for each triangle for each model, so we want to avoid allocating vectors if possible. clipToScreen
// does this by taking outVec, a vertex (Vector) that it stores the values in and returns, which avoids reallocation.
func (camera *Camera) clipToScreen(vert Vector4, vertID int, model *Model, width, height, halfWidth, halfHeight float32, limitW bool) Vector4 {

	v3 := vert.W

	if limitW {

		if !camera.perspective {
			v3 = 1.0
		}

		// If the trangle is beyond the screen, we'll just pretend it's not and limit it to the closest possible value > 0
		// If it's too small, there will be visual artifacts when the camera is right up against surfaces
		// If it's too large, then textures and vertices will appear to warp and bend "around" the screen, towards the "back" of the camera
		if v3 < 0 {
			v3 = 0.00005
		}

	} else {
		if v3 < 0 {
			v3 *= -1
		}
	}

	// Again, this function should only be called with pre-transformed 4D vertex arguments.

	// It's 1 frame faster on the stress test not to have to calculate the half screen width and height here.
	vert.X = (vert.X/v3)*width + halfWidth
	vert.Y = (vert.Y/-v3)*height + halfHeight
	vert.Z = vert.Z / v3
	vert.W = 1

	return vert

}

// ClipToScreen projects the pre-transformed vertex in View space and remaps it to screen coordinates.
func (camera *Camera) ClipToScreen(vert Vector3) Vector3 {
	width, height := camera.Size()
	return camera.clipToScreen(vert.To4D(), 0, nil, float32(width), float32(height), float32(width)/2, float32(height)/2, false).To3D()
}

// WorldToScreenPixels transforms a 3D position in the world to a position onscreen, with X and Y representing the pixels.
// The Z coordinate indicates depth away from the camera in 3D world units.
func (camera *Camera) WorldToScreenPixels(vert Vector3) Vector3 {
	v := NewMatrix4Translate(vert.X, vert.Y, vert.Z).Mult(camera.ViewMatrix().Mult(camera.Projection()))
	width, height := camera.Size()
	return camera.clipToScreen(v.MultVecW(Vector3{}), 0, nil, float32(width), float32(height), float32(width)/2, float32(height)/2, false).To3D()
}

// WorldToScreen transforms a 3D position in the world to a 2D vector, with X and Y ranging from -1 to 1.
// The Z coordinate indicates depth away from the camera in 3D world units.
func (camera *Camera) WorldToScreen(vert Vector3) Vector3 {
	v := camera.WorldToScreenPixels(vert)
	w, h := camera.Size()
	v.X /= float32(w) / 2
	v.X -= 1
	v.Y /= float32(h) / 2
	v.Y -= 1
	return v
}

// ScreenToWorldPixels converts an x and y pixel position on screen to a 3D point in front of the camera.
// The depth argument changes how deep the returned Vector is in 3D world units.
func (camera *Camera) ScreenToWorldPixels(x, y int, depth float32) Vector3 {

	w := camera.ColorTexture().Bounds().Dx()
	h := camera.ColorTexture().Bounds().Dy()

	x = math32.Clamp(x, 0, w)
	y = math32.Clamp(y, 0, h)

	// For whatever reason, the depth isn't being properly transformed, so I do it manually sorta below
	vec := Vector3{float32(x)/float32(w) - 0.5, -(float32(y)/float32(h) - 0.5), -1}

	return camera.screenToWorldTransform(vec, depth)

}

// ScreenToWorld converts an x and y position on screen to a 3D point in front of the camera.
// x and y are values ranging between -0.5 and 0.5 for both the horizontal and vertical axes.
// The depth argument changes how deep the returned Vector is in 3D world units.
func (camera *Camera) ScreenToWorld(x, y, depth float32) Vector3 {
	// x = math32.Clamp(x, -0.5, 0.5)
	// y = math32.Clamp(y, -0.5, 0.5)
	vec := Vector3{x, y, -1}
	return camera.screenToWorldTransform(vec, depth)
}

func (camera *Camera) screenToWorldTransform(vec Vector3, depth float32) Vector3 {

	projection := camera.ViewMatrix().Mult(camera.Projection()).Inverted()

	vecOut := projection.MultVecW(vec)

	vecOut.X /= vecOut.W
	vecOut.Y /= vecOut.W
	vecOut.Z /= vecOut.W

	// TODO: This part shouldn't be necessary if the inverted projection matrix properly transformed the depth part
	// back into the Vector, but for whatever reason, it's not working, so here I basically hack in a manual solution
	diff := vecOut.To3D().Sub(camera.WorldPosition()).Unit()

	return camera.WorldPosition().Add(diff.Scale(depth))

}

// WorldUnitToViewRangePercentage converts a unit of world space into a percentage of the view range.
// Basically, let's say a camera's far range is 100 and its near range is 0.
// If you called camera.WorldUnitToViewRangePercentage(50), it would return 0.5.
// This function is primarily useful for custom depth functions operating on Materials.
func (camera *Camera) WorldUnitToViewRangePercentage(unit float32) float32 {
	return unit / (camera.far - camera.near)
}

// WorldToClip transforms a 3D position in the world to clip coordinates (before screen normalization).
func (camera *Camera) WorldToClip(vert Vector3) Vector3 {
	v := NewMatrix4Translate(vert.X, vert.Y, vert.Z).Mult(camera.ViewMatrix().Mult(camera.Projection()))
	return v.MultVecW(Vector3{}).To3D()
}

// PointInFrustum returns true if the point is visible through the camera frustum.
func (camera *Camera) PointInFrustum(point Vector3) bool {

	diff := point.Sub(camera.WorldPosition())
	pcZ := diff.Dot(camera.cameraForward)
	aspectRatio := camera.AspectRatio()

	if pcZ > camera.far || pcZ < camera.near {
		return false
	}

	if camera.perspective {

		h := pcZ * camera.sphereFactorTang

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

		width := camera.orthoScale / 2
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

	if pcZ > camera.far+radius || pcZ < camera.near-radius {
		return false
	}

	if camera.perspective {

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

		width := camera.orthoScale
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

// ModelInFrustum returns if a model is onscreen when viewed through a Camera.
func (camera *Camera) ModelInFrustum(model *Model) bool {
	model.Transform() // Make sure to update the transform of the Model as necessary.
	return camera.SphereInFrustum(model.frustumCullingSphere)
}

// AspectRatio returns the camera's aspect ratio (width / height).
func (camera *Camera) AspectRatio() float32 {
	w, h := camera.Size()
	return float32(w) / float32(h)
}

// Clear should be called at the beginning of a single rendered frame and clears the Camera's backing textures
// before rendering.
// It also resets the debug values.
// This function will use the world's clear color, assuming the Camera is in a Scene that has a World.
// If the Scene has no World, or the Camera is not in the Scene, then it will clear using transparent black.
func (camera *Camera) Clear() {

	if scene := camera.Scene(); scene != nil {
		if world := scene.World; world != nil {
			camera.ClearWithColor(world.ClearColor)
			return
		}
	}

	camera.ClearWithColor(NewColor(0, 0, 0, 0))

}

// ClearWithColor should be called at the beginning of a single rendered frame and clears the Camera's backing textures
// to the desired color before rendering.
// It also resets the debug values.
func (camera *Camera) ClearWithColor(clear Color) {

	rgba := clear.ToNRGBA64()

	if camera.AccumulationColorMode != AccumulationColorModeNone {
		camera.accumulatedBackBuffer.Clear()
		camera.accumulatedBackBuffer.DrawImage(camera.resultAccumulatedColorTexture, nil)
		camera.resultAccumulatedColorTexture.Clear()
		switch camera.AccumulationColorMode {
		case AccumulationColorModeBelow:
			camera.resultAccumulatedColorTexture.DrawImage(camera.accumulatedBackBuffer, camera.AccumulationDrawOptions)
			camera.resultAccumulatedColorTexture.DrawImage(camera.resultColorTexture, nil)
		case AccumulationColorModeAbove:
			camera.resultAccumulatedColorTexture.DrawImage(camera.resultColorTexture, nil)
			camera.resultAccumulatedColorTexture.DrawImage(camera.accumulatedBackBuffer, camera.AccumulationDrawOptions)
		case AccumulationColorModeSingleLastFrame:
			camera.resultAccumulatedColorTexture.DrawImage(camera.resultColorTexture, camera.AccumulationDrawOptions)
		}
	}

	camera.resultColorTexture.Fill(rgba)

	if camera.RenderDepth {
		camera.resultDepthTexture.Clear()
	}

	if camera.RenderNormals {
		camera.resultNormalTexture.Clear()
	}

	if time.Since(camera.DebugInfo.tickTime).Milliseconds() >= 100 {

		camera.PrevFrametimes = append(camera.PrevFrametimes, float32(camera.DebugInfo.FrameTime.Seconds()))

		for len(camera.PrevFrametimes) > camera.PrevDebugInfoCap {
			camera.PrevFrametimes = camera.PrevFrametimes[1:]
		}

		camera.DebugInfo.FrameTime = camera.DebugInfo.currentFrameTime
		camera.DebugInfo.AnimationTime = camera.DebugInfo.currentAnimationTime
		camera.DebugInfo.LightTime = camera.DebugInfo.currentLightTime
		camera.DebugInfo.tickTime = time.Now()
	}

	camera.DebugInfo.currentFrameTime = 0
	camera.DebugInfo.currentAnimationTime = 0
	camera.DebugInfo.currentLightTime = 0
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

// RenderScene renders the provided Scene.
// Note that if Camera.RenderDepth is false, scenes rendered one after another in multiple RenderScene() calls will be rendered on top of
// each other in the Camera's texture buffers. Note that each MeshPart of a Model has a maximum renderable triangle count of 21845.
func (camera *Camera) RenderScene(scene *Scene) {
	camera.RenderNodes(scene, scene.Root)
}

var meshes []*Model
var lights []ILight

// RenderNodes renders all nodes starting with the provided rootNode using the Scene's properties (fog, for example). Note that if Camera.RenderDepth
// is false, scenes rendered one after another in multiple RenderScene() calls will be rendered on top of each other in the Camera's texture buffers.
// Note that each MeshPart of a Model has a maximum renderable triangle count of 21845.
func (camera *Camera) RenderNodes(scene *Scene, rootNode INode) {

	meshes = meshes[:0]
	lights = lights[:0]

	if model, isModel := rootNode.(*Model); isModel {
		meshes = append(meshes, model)
	}

	if camera.SectorRendering {

		// Gather sectors
		sectors := rootNode.SearchTree().bySectors()
		sectorModels := sectors.Models()

		var insideSector *Sector

		sectors.ForEach(func(node INode) bool {

			sectorModel := node.(*Model)
			sector := node.(*Model).sector
			sector.rendering = false

			// Set them all to be invisible by default
			if sectorModel.DynamicBatchOwner == nil {
				sector.rendering = false
			} else {
				// Making a sector dynamically batched is just way too much to deal with, I'm sorry
				panic("Can't make a sector " + sectorModel.Path() + " dynamically batched as well")
			}

			if sector.AABB.PointInside(camera.WorldPosition()) {
				if insideSector == nil || sector.AABB.Dimensions.MaxSpan() < insideSector.AABB.Dimensions.MaxSpan() {
					insideSector = sector
				}
			}

			return true
		})

		camera.currentSector = insideSector
		if insideSector != nil {
			insideSector.rendering = true

			// Make neighbors visible
			for r := range insideSector.NeighborsWithinRange(camera.SectorRenderDepth) {
				r.rendering = true
			}

			rootNode.SearchTree().ByType(NodeTypeModel).ForEach(func(node INode) bool {
				model := node.(*Model)

				if model.sector != nil && model.sector.rendering {
					if model.sector.rendering {
						meshes = append(meshes, model)
					}
				} else if model.DynamicBatchOwner == nil {

					// If something is dynamically batching, then we don't want to deal with sectors, because the batched objects belong to sectors.
					if model.DynamicBatcher() {
						meshes = append(meshes, model)
					} else if model.SectorType() == SectorTypeStandalone || (model.SectorType() == SectorTypeObject && model.isInVisibleSector(sectorModels)) {
						meshes = append(meshes, model)
					} else if s := model.sectorHierarchy(); s != nil && s.rendering {
						meshes = append(meshes, model)
					}

				}

				return true
			})

			rootNode.SearchTree().ByType(NodeTypeLight).ForEach(func(node INode) bool {
				light := node.(ILight)
				if light.SectorType() == SectorTypeStandalone || (light.SectorType() == SectorTypeObject && light.isInVisibleSector(sectorModels)) {
					lights = append(lights, light)
				} else if s := light.sectorHierarchy(); s != nil && s.rendering {
					lights = append(lights, light)
				}
				return true
			})

		}

	} else {

		rootNode.SearchTree().ForEach(func(node INode) bool {

			// Avoid allocating new model / lights slices
			if m, ok := node.(*Model); ok && m.DynamicBatchOwner == nil {
				meshes = append(meshes, m)
			} else if l, ok := node.(ILight); ok {
				lights = append(lights, l)
			}

			return true
		})

	}

	camera.Render(scene, lights, meshes...)

}

// RenderImageSequence runs a render function for each frame in an image sequence.
// frameCount is the number of frames to render, and renderFunc is a callback to be called
// for each frame; it should perform any rendering functions that would write to the returned image sequence textures each frame.
// The size of each image in the returned sequence would be the size of the camera.
//
// Say you had a camera pointed at a model and wanted to make a 60 frame image sequence of it spinning.
// An example would be something like:
//
//	camera.RenderImageSequence(60, func(frameIndex int){
//		camera.Clear() 	// Clear the camera
//		camera.RenderScene(scene) // Renders to the camera's color texture; gets copied to an image in the sequence
//		model.Rotate(0, 1, 0, Pi * 2 / 60) // Rotates the model 2pi over 60 frames
//	})
func (camera *Camera) RenderImageSequence(frameCount int, renderFunc func(frameIndex int)) []*ebiten.Image {

	images := []*ebiten.Image{}

	for i := 0; i < frameCount; i++ {
		img := ebiten.NewImage(camera.Size())
		renderFunc(i)
		img.DrawImage(camera.ColorTexture(), nil)
		images = append(images, img)
	}

	return images

}

// CurrentSector returns the current sector the Camera is in, if sector-based rendering is enabled.
func (camera *Camera) CurrentSector() *Sector { return camera.currentSector }

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

var sceneLights []ILight

// Render renders all of the models passed using the provided Scene's properties (fog, for example) and lights provided. Note that if Camera.RenderDepth
// is false, scenes rendered one after another in multiple Render() calls will be rendered on top of each other in the Camera's texture buffers.
// Also, the function will automatically include the Scene's world ambient light, if there is a world.
func (camera *Camera) Render(scene *Scene, lights []ILight, models ...*Model) {

	scene.HandleAutobatch()

	frametimeStart := time.Now()

	sceneLights = sceneLights[:0]

	if scene.World != nil {

		camera.DebugInfo.LightCount++
		if scene.World.LightingOn && scene.World.AmbientLight.IsOn() {
			scene.World.AmbientLight.beginRender()
			sceneLights = append(sceneLights, scene.World.AmbientLight)
		}

	}

	for _, light := range lights {
		camera.DebugInfo.LightCount++
		if (scene.World == nil || scene.World.LightingOn) && light.IsOn() {
			light.beginRender()
			sceneLights = append(sceneLights, light)
		}
	}

	// if scene.World == nil || scene.World.LightingOn {

	// 	for _, l := range scene.Root.SearchTree().INodes() {
	// 		if light, isLight := l.(ILight); isLight {
	// 			camera.DebugInfo.LightCount++
	// 			if light.IsOn() {
	// 				sceneLights = append(sceneLights, light)
	// 				light.beginRender()
	// 				camera.DebugInfo.ActiveLightCount++
	// 			}
	// 		}
	// 	}

	// 	if scene.World != nil && scene.World.AmbientLight != nil && scene.World.AmbientLight.IsOn() {
	// 		sceneLights = append(sceneLights, scene.World.AmbientLight)
	// 		scene.World.AmbientLight.beginRender()
	// 		camera.DebugInfo.LightCount++
	// 		camera.DebugInfo.ActiveLightCount++
	// 	}

	// }

	// By multiplying the camera's position against the view matrix (which contains the negated camera position), we're left with just the rotation
	// matrix, which we feed into model.TransformedVertices() to draw vertices in order of distance.
	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	colorPassShaderOptions := &ebiten.DrawTrianglesShaderOptions{}

	// Reusing vectors rather than reallocating for all triangles for all models
	solids := []renderPair{}
	transparents := []renderPair{}

	depths := map[*Model]float32{}

	cameraPos := camera.WorldPosition()

	depthMarginPercentage := (camera.far - camera.near) * camera.DepthMargin

	camSpread := camera.far - camera.near + (depthMarginPercentage * 2)

	for _, model := range models {

		if !model.DynamicBatcher() {

			if !model.autoBatched || model.AutoBatchMode != AutoBatchStatic {
				for range model.Mesh.MeshParts {
					camera.DebugInfo.TotalParts++
				}
			}

		}

		if !model.visible || model.DynamicBatchOwner != nil {
			continue
		}

		if !model.DynamicBatcher() {

			if model.FrustumCulling {

				if !camera.ModelInFrustum(model) {
					continue
				}
				model.refreshVertexVisibility()

			}

			if model.Mesh != nil {

				modelIsTransparent := false

				for _, mp := range model.Mesh.MeshParts {

					if !mp.isVisible() {
						continue
					}

					if model.isTransparent(mp) {
						transparents = append(transparents, renderPair{model, mp})
						modelIsTransparent = true
					} else {
						solids = append(solids, renderPair{model, mp})
					}
				}

				if !camera.RenderDepth || modelIsTransparent {
					depths[model] = cameraPos.DistanceSquaredTo(model.WorldPosition())
					// depths[model] = camera.WorldToScreen(model.WorldPosition()).Z
				}

			}

		} else {

			transparent := false

			// TODO: Review depth sorting for dynamic batchers

			for meshPart, modelSlice := range model.DynamicBatchModels {

				if !meshPart.isVisible() {
					continue
				}

				for _, child := range modelSlice {

					if !child.visible {
						continue
					}

					if !transparent {

						for _, mp := range child.Mesh.MeshParts {

							if mp.isVisible() && child.isTransparent(mp) {
								transparent = true
								break
							}

						}

					}

				}

				if transparent {
					transparents = append(transparents, renderPair{model, meshPart})
					depths[model] = cameraPos.DistanceSquaredTo(model.WorldPosition())
				} else {
					solids = append(solids, renderPair{model, meshPart})
					if !camera.RenderDepth {
						depths[model] = cameraPos.DistanceSquaredTo(model.WorldPosition())
					}
				}

				camera.DebugInfo.TotalParts += len(modelSlice)

			}

		}

	}

	// If the camera isn't rendering depth, then we should sort models by distance to ensure things draw in something like the correct order
	if !camera.RenderDepth {

		sort.SliceStable(solids, func(i, j int) bool {
			return depths[solids[i].Model] > depths[solids[j].Model]
		})

	}

	camWidth := camera.resultColorTexture.Bounds().Dx()
	camHeight := camera.resultColorTexture.Bounds().Dy()

	if camera.MaxLightCount > 0 {

		var pos Vector3
		if camera.MaxLightCountFocusPoint != nil {
			pos = camera.MaxLightCountFocusPoint.WorldPosition()
		} else {
			pos = camera.WorldPosition()
		}

		sort.SliceStable(sceneLights, func(i, j int) bool {
			// We sort ambient lights and directional lights as being closest to the camera, naturally
			if _, ok := sceneLights[i].(*AmbientLight); ok {
				return true
			} else if _, ok := sceneLights[j].(*AmbientLight); ok {
				return false
			} else if _, ok := sceneLights[i].(*DirectionalLight); ok {
				return true
			} else if _, ok := sceneLights[j].(*DirectionalLight); ok {
				return false
			}
			return pos.DistanceSquaredTo(sceneLights[i].WorldPosition()) < pos.DistanceSquaredTo(sceneLights[j].WorldPosition())
		})
		sceneLights = sceneLights[:math32.Min(camera.MaxLightCount, len(sceneLights))]
	}

	for _, light := range sceneLights {
		if scene.World != nil && scene.World.LightingOn && light.IsOn() {
			camera.DebugInfo.ActiveLightCount++
		}
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
				lighting = scene.World.LightingOn && !mat.Shadeless && !model.Shadeless
			} else {
				lighting = scene.World.LightingOn && !model.Shadeless
			}
		}

		camera.DebugInfo.TotalTris += meshPart.TriangleCount()

		if model.DynamicBatchOwner != nil {
			camera.DebugInfo.BatchedParts++
		}

		globalSortingTriangleBucket.sortMode = TriangleSortModeBackToFront
		if meshPart.Material != nil {
			globalSortingTriangleBucket.sortMode = meshPart.Material.TriangleSortMode
		}

		model.ProcessVertices(vpMatrix, camera, meshPart, true)

		srcW := float32(0.0)
		srcH := float32(0.0)

		if mat != nil && mat.Texture != nil && mat.UseTexture {
			srcW = float32(mat.Texture.Bounds().Dx())
			srcH = float32(mat.Texture.Bounds().Dy())
		}

		if lighting {

			t := time.Now()

			if model.LightGroup != nil && model.LightGroup.Active {
				sceneLights = model.LightGroup.Lights
				for _, l := range sceneLights {
					l.beginRender() // Call this because it's relatively cheap and necessary if a light doesn't exist in the Scene
				}
			}

			for _, light := range sceneLights {
				light.beginModel(model)
			}

			camera.DebugInfo.currentLightTime += time.Since(t)

		}

		mesh := model.Mesh

		maxSpan := model.Mesh.Dimensions.MaxSpan()
		modelPos := model.WorldPosition()

		// Here we do all vertex transforms first because of data locality (it's faster to access all vertex transformations, then go back and do all UV values, etc)

		mpColor := model.Color

		if meshPart.Material != nil {
			mpColor = mpColor.MultiplyRGBA(meshPart.Material.Color.ToFloat32s())
		}

		if lighting && !globalSortingTriangleBucket.IsEmpty() {

			t := time.Now()

			meshPart.ForEachVertexIndex(func(vertIndex int) {
				mesh.vertexLights[vertIndex] = Color{0, 0, 0, 1}
			}, true)

			for _, light := range sceneLights {

				// Skip calculating lighting for objects that are too far away from light sources.
				if point, ok := light.(*PointLight); ok && point.Range > 0 {
					dist := maxSpan + point.Range
					if modelPos.DistanceSquaredTo(point.WorldPosition()) > dist*dist {
						continue
					}
					// } else if cube, ok := light.(*CubeLight); ok && cube.Range > 0 {
					// 	dist := maxSpan + cube.Range
					// 	if modelPos.DistanceSquared(cube.WorldPosition()) > dist*dist {
					// 		continue
					// 	}
				}

				light.Light(meshPart, model, mesh.vertexLights, true)

			}

			camera.DebugInfo.currentLightTime += time.Since(t)

		}

		halfCamWidth, halfCamHeight := float32(camWidth)/2, float32(camHeight)/2

		// TODO: Implement PS1-style automatic tesselation

		meshPartVertexIndexStart := meshPart.VertexIndexStart

		customDepthFunctionSet := mat != nil && mat.CustomDepthFunction != nil
		vertexClipFunctionOn := model != nil && model.VertexClipFunction != nil

		for vertIndex := meshPart.VertexIndexStart; vertIndex < meshPart.VertexIndexEnd; vertIndex++ {

			// We clip the vertices to the screen here manually because it wasn't being inlined previously.

			// CLIP SCREEN START

			w := mesh.vertexTransforms[vertIndex].W

			if !camera.perspective {
				w = 1.0
			}

			// If the trangle is beyond the screen, we'll just pretend it's not and limit it to the closest possible value > 0
			// If it's too small, there will be visual artifacts when the camera is right up against surfaces
			// If it's too large, then textures and vertices will appear to warp and bend "around" the screen, towards the "back" of the camera
			if w < 0 {
				w = 0.00005
			}

			// It's 1 frame faster on the stress test not to have to calculate the half screen width and height here.
			mesh.vertexTransforms[vertIndex].X = (mesh.vertexTransforms[vertIndex].X/w)*float32(camWidth) + halfCamWidth
			mesh.vertexTransforms[vertIndex].Y = (mesh.vertexTransforms[vertIndex].Y/-w)*float32(camHeight) + halfCamHeight
			// mesh.vertexTransforms[vertIndex].Z /= w

			if vertexClipFunctionOn {
				model.VertexClipFunction(&mesh.vertexTransforms[vertIndex], vertIndex)
			}

			// CLIP SCREEN END

			colorVertexList[vertexListIndex].DstX = float32(mesh.vertexTransforms[vertIndex].X)
			colorVertexList[vertexListIndex].DstY = float32(mesh.vertexTransforms[vertIndex].Y)
			depthVertexList[vertexListIndex].DstX = float32(mesh.vertexTransforms[vertIndex].X)
			depthVertexList[vertexListIndex].DstY = float32(mesh.vertexTransforms[vertIndex].Y)

			var uvU, uvV float32

			// We set the UVs back here because we might need to use them if the material has clip alpha enabled.
			// We do 1 - v here (aka Y in texture coordinates) because 1.0 is the top of the texture while 0 is the bottom in UV coordinates,
			// but when drawing textures 0 is the top, and the sourceHeight is the bottom.
			if camera.PerspectiveCorrectedTextureMapping {
				uvU = float32((mesh.VertexUVs[vertIndex].X / w) * srcW)
				uvV = float32(((1 - mesh.VertexUVs[vertIndex].Y) / w) * srcH)
			} else {
				uvU = float32(mesh.VertexUVs[vertIndex].X * srcW)
				uvV = float32((1 - mesh.VertexUVs[vertIndex].Y) * srcH)
			}

			if mat != nil && mat.TextureMapMode == TextureMapModeScreen {
				uvU = mat.TextureMapScreenOffset.X
				uvV = mat.TextureMapScreenOffset.Y
			}

			colorVertexList[vertexListIndex].SrcX = uvU
			colorVertexList[vertexListIndex].SrcY = uvV

			if camera.PerspectiveCorrectedTextureMapping {
				d := 1.0 / float32(w)
				colorVertexList[vertexListIndex].Custom0 = d // Set the perspective divide here
				depthVertexList[vertexListIndex].Custom0 = d
				// normalVertexList[vertexListIndex].Custom0 = d
			}

			depthVertexList[vertexListIndex].SrcX = uvU
			depthVertexList[vertexListIndex].SrcY = uvV

			if camera.RenderNormals {

				normalVertexList[vertexListIndex].DstX = float32(mesh.vertexTransforms[vertIndex].X)
				normalVertexList[vertexListIndex].DstY = float32(mesh.vertexTransforms[vertIndex].Y)

				normalVertexList[vertexListIndex].SrcX = uvU
				normalVertexList[vertexListIndex].SrcY = uvV

				normalVertexList[vertexListIndex].ColorR = float32(mesh.vertexTransformedNormals[vertIndex].X*0.5 + 0.5)
				normalVertexList[vertexListIndex].ColorG = float32(mesh.vertexTransformedNormals[vertIndex].Y*0.5 + 0.5)
				normalVertexList[vertexListIndex].ColorB = float32(mesh.vertexTransformedNormals[vertIndex].Z*0.5 + 0.5)

			}

			// Vertex colors

			if activeChannel := mesh.VertexActiveColorChannel; activeChannel >= 0 {
				colorVertexList[vertexListIndex].ColorR = mesh.VertexColors[activeChannel][vertIndex].R * mpColor.R
				colorVertexList[vertexListIndex].ColorG = mesh.VertexColors[activeChannel][vertIndex].G * mpColor.G
				colorVertexList[vertexListIndex].ColorB = mesh.VertexColors[activeChannel][vertIndex].B * mpColor.B
				colorVertexList[vertexListIndex].ColorA = mesh.VertexColors[activeChannel][vertIndex].A * mpColor.A
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

				// 3/28/23, TODO: We used to use the transformed vertex positions for rendering the depth texture. Note
				// that this makes fog shift aggressively as you turn the camera, more noticeably in first-person games
				// (as points closer to the corners of the screen are mathematically closer to the camera because of projection).
				// In an attempt to fix this, I used the below, now commented-out depth function. This attempt did fix the fog,
				// but it also made depth sorting buggier, which is unacceptable. For now, I've reverted this change to have
				// better depth sorting. This is currently fine, though the fog issue should be resolved at some point in the future.

				// See this Discord conversation for the visualization of the issue:
				// https://discord.com/channels/842049801528016967/844522898126536725/1090223569247674488

				// p, s, r := model.Transform().Inverted().Decompose()
				// invertedCameraPos := r.MultVec(camera.WorldPosition()).Add(p.Mult(Vector{1 / s.X, 1 / s.Y, 1 / s.Z, s.W}))
				// depth := (invertedCameraPos.Distance(mesh.VertexPositions[vertIndex]) - camera.near) / (camera.far - camera.near)

				vertexDepth := mesh.vertexTransforms[vertIndex].Z

				if customDepthFunctionSet {
					vertexDepth = mat.CustomDepthFunction(model, camera, meshPart, vertIndex, vertexDepth)
				}

				depth := (vertexDepth + depthMarginPercentage) / camSpread

				if depth < 0 {
					depth = 0
				} else if depth > 1 {
					depth = 1
				}

				depthVertexList[vertexListIndex].ColorR = float32(depth)
				depthVertexList[vertexListIndex].ColorG = float32(depth)
				depthVertexList[vertexListIndex].ColorB = float32(depth)
				depthVertexList[vertexListIndex].ColorA = 1

			} else if scene.World != nil && scene.World.FogOn {

				vertexDepth := mesh.vertexTransforms[vertIndex].Z

				if customDepthFunctionSet {
					vertexDepth = mat.CustomDepthFunction(model, camera, meshPart, vertIndex, vertexDepth)
				}

				depth := (vertexDepth + depthMarginPercentage) / camSpread

				if depth < 0 {
					depth = 0
				} else if depth > 1 {
					depth = 1
				}

				// depth = 1 - depth

				depth = float32(scene.World.FogRange[0] + ((scene.World.FogRange[1]-scene.World.FogRange[0])*1 - float32(depth)))

				if scene.World.FogMode == FogAdd {
					colorVertexList[vertexListIndex].ColorR += scene.World.FogColor.R * float32(depth)
					colorVertexList[vertexListIndex].ColorG += scene.World.FogColor.G * float32(depth)
					colorVertexList[vertexListIndex].ColorB += scene.World.FogColor.B * float32(depth)
				} else if scene.World.FogMode == FogSub {
					colorVertexList[vertexListIndex].ColorR *= scene.World.FogColor.R * float32(depth)
					colorVertexList[vertexListIndex].ColorG *= scene.World.FogColor.G * float32(depth)
					colorVertexList[vertexListIndex].ColorB *= scene.World.FogColor.B * float32(depth)
				}

			}

			vertexListIndex++

		}

		if vertexListIndex == 0 {
			return
		}

		globalSortingTriangleBucket.ForEach(func(triIndex, triID int, vertexIndices []int) {

			for _, index := range vertexIndices {
				indexList[indexListIndex] = uint16(index - meshPartVertexIndexStart + indexListStart)
				indexListIndex++
			}

		})

		indexListStart = vertexListIndex

		// for i := 0; i < vertexListIndex; i++ {
		// 	indexList[i] = uint16(i)
		// }

	}

	flush := func(rp renderPair) {

		if vertexListIndex == 0 || indexListIndex == 0 {
			vertexListIndex = 0
			indexListIndex = 0
			indexListStart = 0
			return
		}

		model := rp.Model
		meshPart := rp.MeshPart
		mat := meshPart.Material

		var img *ebiten.Image

		if mat != nil && mat.UseTexture {
			img = mat.Texture
		}

		if img == nil {
			img = defaultImg
		}

		perspectiveCorrection := 0
		if camera.PerspectiveCorrectedTextureMapping {
			perspectiveCorrection = 1
		}

		colorPassOptions := &ebiten.DrawTrianglesOptions{}

		textureFilterMode := 0
		textureMapMode := 0
		textureMapScreenSize := float32(1.0)

		if mat != nil {
			colorPassOptions.Filter = mat.TextureFilterMode
			colorPassOptions.Address = mat.textureWrapMode

			textureMapMode = mat.TextureMapMode
			textureMapScreenSize = mat.TextureMapScreenSize

			switch mat.TextureFilterMode {
			case ebiten.FilterNearest:
				textureFilterMode = 0
			case ebiten.FilterLinear:
				textureFilterMode = 1
			}

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

				shaderOpt := &ebiten.DrawTrianglesShaderOptions{
					Images: [4]*ebiten.Image{camera.resultDepthTexture, img},
					Uniforms: map[string]any{
						"PerspectiveCorrection": perspectiveCorrection,
						"TextureMapMode":        textureMapMode,
						"TextureMapScreenSize":  textureMapScreenSize,
					},
				}
				camera.depthIntermediate.DrawTrianglesShader(depthVertexList[:vertexListIndex], indexList[:indexListIndex], camera.clipAlphaShader, shaderOpt)

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

		hasFragShader := mat != nil && mat.fragmentShader != nil && mat.FragmentShaderOn

		// If rendering depth, and rendering through a custom fragment shader, we'll need to render the tris to the ColorIntermediate buffer using the custom shader.
		// If we're not rendering through a custom shader, we can render to ColorIntermediate and then composite that onto the finished ColorTexture.
		// If we're not rendering depth, but still rendering through the shader, we can render to the intermediate texture, and then from there composite.
		// Otherwise, we can just draw the triangles normally.

		// Removing this because using ColorM has been deprecated by the colorm package and has no analog for rendering using shaders, which is what we now always do
		// if model.ColorBlendingFunc != nil {
		// 	colorPassOptions.ColorM = model.ColorBlendingFunc(model, meshPart) // Modify the model's appearance using its color blending function
		// }

		colorPassOptions.Blend = ebiten.BlendSourceOver
		colorPassShaderOptions.Blend = ebiten.BlendSourceOver

		if mat != nil {
			colorPassOptions.Blend = mat.Blend
			colorPassShaderOptions.Blend = mat.Blend
		}

		if scene != nil && scene.World != nil {

			colorPassShaderOptions.Uniforms = map[string]any{
				"Fog":                   scene.World.fogAsFloatSlice(),
				"FogRange":              scene.World.FogRange,
				"DitherSize":            scene.World.DitheredFogSize,
				"FogCurve":              float32(scene.World.FogCurve),
				"BayerMatrix":           bayerMatrix,
				"PerspectiveCorrection": perspectiveCorrection,
				"TextureFilterMode":     textureFilterMode,
				"TextureMapMode":        textureMapMode,
				"TextureMapScreenSize":  textureMapScreenSize,
			}

		} else {

			colorPassShaderOptions.Uniforms = map[string]any{
				"Fog":                   []float32{0, 0, 0, 0},
				"FogRange":              []float32{0, 1},
				"PerspectiveCorrection": perspectiveCorrection,
			}

		}

		colorPassShaderOptions.Images[0] = img
		colorPassShaderOptions.Images[1] = camera.depthIntermediate

		fogless := float32(0)
		if mat != nil && mat.Fogless {
			fogless = 1
		}

		if camera.RenderNormals {
			colorPassShaderOptions.Images[0] = defaultImg
			colorPassShaderOptions.Uniforms["Fogless"] = 1 // No fog in a normal render
			camera.resultNormalTexture.DrawTrianglesShader(normalVertexList[:vertexListIndex], indexList[:indexListIndex], camera.colorShader, colorPassShaderOptions)
			// camera.resultNormalTexture.DrawTrianglesShader(colorVertexList[:vertexListIndex], indexList[:indexListIndex], camera.colorShader, colorPassShaderOptions)
		} else {
			colorPassShaderOptions.Uniforms["Fogless"] = fogless
		}

		if camera.RenderDepth {

			if hasFragShader {

				if mat.FragmentShaderOptions != nil {
					colorPassShaderOptions.Blend = mat.FragmentShaderOptions.Blend
					colorPassShaderOptions.AntiAlias = mat.FragmentShaderOptions.AntiAlias
					colorPassShaderOptions.FillRule = mat.FragmentShaderOptions.FillRule
					colorPassShaderOptions.Blend = mat.FragmentShaderOptions.Blend
					for k, v := range mat.FragmentShaderOptions.Uniforms {
						colorPassShaderOptions.Uniforms[k] = v
					}
					if len(mat.FragmentShaderOptions.Images) > 0 && mat.FragmentShaderOptions.Images[0] != nil {
						colorPassShaderOptions.Images[0] = mat.FragmentShaderOptions.Images[0]
					}
					if len(mat.FragmentShaderOptions.Images) > 1 && mat.FragmentShaderOptions.Images[1] != nil {
						colorPassShaderOptions.Images[1] = mat.FragmentShaderOptions.Images[1]
					}
					if len(mat.FragmentShaderOptions.Images) > 2 && mat.FragmentShaderOptions.Images[2] != nil {
						colorPassShaderOptions.Images[2] = mat.FragmentShaderOptions.Images[2]
					}
					if len(mat.FragmentShaderOptions.Images) > 3 && mat.FragmentShaderOptions.Images[3] != nil {
						colorPassShaderOptions.Images[3] = mat.FragmentShaderOptions.Images[3]
					}
				}
				camera.resultColorTexture.DrawTrianglesShader(colorVertexList[:vertexListIndex], indexList[:indexListIndex], mat.fragmentShader, colorPassShaderOptions)
			} else {
				camera.resultColorTexture.DrawTrianglesShader(colorVertexList[:vertexListIndex], indexList[:indexListIndex], camera.colorShader, colorPassShaderOptions)
			}

			// camera.resultColorTexture.DrawRectShader(w, h, camera.colorShader, rectShaderOptions)

		} else {

			if hasFragShader {
				// TODO: Review usage of FragmentShaderOptions here.
				camera.resultColorTexture.DrawTrianglesShader(colorVertexList[:vertexListIndex], indexList[:indexListIndex], mat.fragmentShader, mat.FragmentShaderOptions)
			} else {
				camera.resultColorTexture.DrawTriangles(colorVertexList[:vertexListIndex], indexList[:indexListIndex], img, colorPassOptions)
			}

		}

		camera.DebugInfo.DrawnTris += indexListIndex / 3
		camera.DebugInfo.DrawnParts++

		vertexListIndex = 0
		indexListIndex = 0
		indexListStart = 0

	}

	sort.SliceStable(transparents, func(i, j int) bool {
		return depths[transparents[i].Model] > depths[transparents[j].Model]
	})

	renderPasses := [][]renderPair{
		solids, transparents,
	}

	for _, pass := range renderPasses {

		for _, pair := range pass {

			// Automatically statically batched models can't render
			if !pair.Model.visible || pair.Model.AutoBatchMode == AutoBatchStatic {
				continue
			}

			// Internally, the idea behind dynamic batching is that we simply hold off on flushing until the
			// end - this saves a lot of time if we're rendering singular low-poly objects, at the cost of each
			// object sharing the same material / object-level properties (color / material blending mode, for
			// example).

			if pair.Model.DynamicBatcher() {

				modelSlice := pair.Model.DynamicBatchModels[pair.MeshPart]

				sort.Slice(modelSlice, func(i, j int) bool {
					return camera.DistanceSquaredTo(modelSlice[i]) > camera.DistanceSquaredTo(modelSlice[j])
				})

				for _, merged := range modelSlice {

					if !merged.visible {
						continue
					}

					if merged.FrustumCulling {
						merged.Transform()
						if !camera.SphereInFrustum(merged.frustumCullingSphere) {
							continue
						}
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

	camera.DebugInfo.currentFrameTime += time.Since(frametimeStart)

}

// packFloat packs two numbers into a single float32 with a given precision. 128 is a good number.
func packFloat(input1, input2, precision float32) float32 {
	a := float32(int(input1 * precision))
	b := input2 * 0.05
	return a + b
}

type DrawSprite3dSettings struct {
	// The image to render
	Image *ebiten.Image
	// How much to offset the depth - useful if you want the object to appear at a position,
	// but in front of or behind other objects. Negative is towards the camera, positive is away.
	// The offset ranges from 0 to 1.
	DepthOffset   float32
	WorldPosition Vector3 // The position of the sprite in 3D space
}

var spriteRender3DVerts = []ebiten.Vertex{
	{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
}

var spriteRender3DIndices = []uint16{
	0, 1, 2,
	2, 3, 0,
}

// RenderSprite3D draws an image on the screen in 2D, but at the screen position of the 3D world position provided,
// and with depth intersection.
// This allows you to render 2D elements "at" a 3D position, and can be very useful in situations where you want
// a sprite to render at 100% size and no perspective or skewing, but still look like it's in the 3D space (like in
// a game with a fixed camera viewpoint).
func (camera *Camera) RenderSprite3D(screen *ebiten.Image, renderSettings ...DrawSprite3dSettings) {

	// TODO: Replace this with a more performant alternative, where we minimize shader / texture switches.
	// TODO 2: Add the ability to resize sprites rendered through this function.

	depthMarginPercentage := (camera.far - camera.near) * camera.DepthMargin

	camViewProj := camera.ViewMatrix().Mult(camera.Projection())

	for _, rs := range renderSettings {

		px := camera.WorldToScreenPixels(rs.WorldPosition)

		out := camViewProj.MultVec(rs.WorldPosition)

		depth := float32((out.Z + depthMarginPercentage) / (camera.far - camera.near + (depthMarginPercentage * 2)))

		if depth < 0 {
			depth = 0
		}
		if depth > 1 {
			depth = 1
		}

		depth += float32(rs.DepthOffset)

		imageW := float32(rs.Image.Bounds().Dx())
		imageH := float32(rs.Image.Bounds().Dy())
		halfImageW := imageW / 2
		halfImageH := imageH / 2

		spriteRender3DVerts[0].DstX = float32(px.X) - halfImageW
		spriteRender3DVerts[0].DstY = float32(px.Y) - halfImageH

		spriteRender3DVerts[1].DstX = float32(px.X) + halfImageW
		spriteRender3DVerts[1].DstY = float32(px.Y) - halfImageH
		spriteRender3DVerts[1].SrcX = imageW

		spriteRender3DVerts[2].DstX = float32(px.X) + halfImageW
		spriteRender3DVerts[2].DstY = float32(px.Y) + halfImageH
		spriteRender3DVerts[2].SrcX = imageW
		spriteRender3DVerts[2].SrcY = imageH

		spriteRender3DVerts[3].DstX = float32(px.X) - halfImageW
		spriteRender3DVerts[3].DstY = float32(px.Y) + halfImageH
		spriteRender3DVerts[3].SrcY = imageH

		shaderOptions := &ebiten.DrawTrianglesShaderOptions{}
		shaderOptions.Images[0] = rs.Image
		shaderOptions.Images[1] = camera.resultDepthTexture
		shaderOptions.Uniforms = map[string]any{
			"SpriteDepth": depth,
		}
		screen.DrawTrianglesShader(spriteRender3DVerts, spriteRender3DIndices, camera.sprite3DShader, shaderOptions)

	}

}

type DynamicRenderSettings struct {
	Model       *Model // The model to be cloned for usage
	clonedModel *Model

	Position Vector3 // The position to render the Model at. Ignored if Transform is set.
	Scale    Vector3 // The scale to render the Model at. Ignored if Transform is set.
	Rotation Matrix4 // The rotation to render the Model with. Ignored if Transform is set.

	Transform Matrix4 // A transform to use for rendering; this is used if it is non-zero.

	Color Color // The color to render the Model.
}

var dynamicRenderMesh = NewCubeMesh()
var dynamicRenderOwner = NewModel("dynamic batch render cubes", dynamicRenderMesh)

// DynamicRender quickly and easily renders an object with the desired setup.
// The Camera must be present in a Scene to perform this function.
func (camera *Camera) DynamicRender(settings ...DynamicRenderSettings) error {

	// TODO: Optimize this with perhaps background cubes that are dynamically batched so they could be rendered in one
	// Render call.

	if scene := camera.Scene(); scene != nil {

		if len(settings) == 0 {
			return errors.New("no render settings to render")
		}

		dynamicRenderOwner.Mesh.MeshParts[0].Material = settings[0].Model.Mesh.MeshParts[0].Material

		dynamicRenderOwner.FrustumCulling = false

		dynamicRenderOwner.DynamicBatchClear()

		for _, setting := range settings {

			if setting.clonedModel == nil {
				if setting.Model == nil {
					return errors.New("no model specified for usage with Camera.DynamicRender(); this function requires a model to clone")
				}
				setting.clonedModel = setting.Model.Clone().(*Model)
			}

			model := setting.clonedModel
			if !setting.Transform.IsZero() {
				model.SetWorldTransform(setting.Transform)
			} else {
				model.SetLocalPositionVec(setting.Position)
				model.SetLocalScaleVec(setting.Scale)
				if !setting.Rotation.IsZero() {
					model.SetLocalRotation(setting.Rotation)
				}
			}
			model.Color = setting.Color
			dynamicRenderOwner.DynamicBatchAdd(dynamicRenderOwner.Mesh.MeshParts[0], model)

		}

		lights := scene.Root.SearchTree().ILights()
		camera.Render(scene, lights, dynamicRenderOwner)

	} else {
		return errors.New("camera is not in a scene; cannot render elements")
	}

	return nil

}

// DrawDebugRenderInfo draws render debug information (like number of drawn objects, number of drawn triangles, frame time, etc)
// at the top-left of the provided screen *ebiten.Image, using the textScale and color provided.
// Note that the frame-time mentioned here is purely the time that Tetra3D spends sending render commands to the command queue.
// Any additional time that Ebitengine takes to flush that queue is not included in this average frame-time value, and is not
// visible outside of debugging and profiling, like with pprof.
func (camera *Camera) DrawDebugRenderInfo(screen *ebiten.Image, textScale float32, col Color) {

	m := camera.DebugInfo.FrameTime.Round(time.Microsecond).Microseconds()
	ft := fmt.Sprintf("%.2fms", float32(m)/1000)

	m = camera.DebugInfo.AnimationTime.Round(time.Microsecond).Microseconds()
	at := fmt.Sprintf("%.2fms", float32(m)/1000)

	m = camera.DebugInfo.LightTime.Round(time.Microsecond).Microseconds()
	lt := fmt.Sprintf("%.2fms", float32(m)/1000)

	sectorName := "<Sector Rendering Off>"
	if camera.SectorRendering {
		if camera.currentSector == nil {
			sectorName = "None"
		} else {
			sectorName = camera.currentSector.Model.name
		}
	}

	debugText := fmt.Sprintf(
		"TPS: %f\nFPS: %f\nTotal render frame-time: %s\nSkinned mesh animation time: %s\nLighting frame-time: %s\nDraw calls: %d/%d (%d dynamically batched)\nRendered triangles: %d/%d\nActive Lights: %d/%d"+
			"\nCamera World Position: %s\nCurrent Sector: %s",
		ebiten.ActualTPS(),
		ebiten.ActualFPS(),
		ft,
		at,
		lt,
		camera.DebugInfo.DrawnParts,
		camera.DebugInfo.TotalParts,
		camera.DebugInfo.BatchedParts,
		camera.DebugInfo.DrawnTris,
		camera.DebugInfo.TotalTris,
		camera.DebugInfo.ActiveLightCount,
		camera.DebugInfo.LightCount,
		camera.WorldPosition(),
		sectorName,
	)

	camera.DrawDebugText(screen, debugText, 0, 0, textScale, col)

	barWidth := float32(64)
	barHeight := float32(24)

	vector.StrokeRect(screen, 128, 16, barWidth, barHeight, 1, color.White, false)

	for i := 0; i < len(camera.PrevFrametimes)-1; i++ {

		ip := 1.0 / float32(camera.PrevDebugInfoCap)

		db := camera.PrevFrametimes[i]
		nextdb := camera.PrevFrametimes[i+1]

		maxFt := 1.0 / float32(ebiten.TPS())

		drawColor := color.RGBA{128, 255, 0, 255}

		if db/maxFt > 1 {
			drawColor = color.RGBA{255, 0, 0, 255}
		}

		cappedFrametimePerc := math32.Clamp(db/maxFt, 0, 1) * (barHeight - 3)
		cappedNextFrametimePerc := math32.Clamp((nextdb)/maxFt, 0, 1) * (barHeight - 3)

		vector.StrokeLine(screen, 128+1+(float32(i)*ip*barWidth), 38-cappedFrametimePerc, 128+1+(float32(i+1)*ip*barWidth), 38-cappedNextFrametimePerc, 2, drawColor, false)
	}

}

// DrawDebugWireframe draws the wireframe triangles of all visible Models underneath the rootNode in the color provided to the screen
// image provided.
func (camera *Camera) DrawDebugWireframe(screen *ebiten.Image, rootNode INode, color Color) {

	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	allModels := append([]INode{rootNode}, rootNode.SearchTree().INodes()...)

	camWidth, camHeight := camera.Size()

	halfCamWidth, halfCamHeight := float32(camWidth)/2, float32(camHeight)/2

	for _, m := range allModels {

		if model, isModel := m.(*Model); isModel {

			if model.FrustumCulling {
				model.Transform()
				if !camera.SphereInFrustum(model.frustumCullingSphere) {
					continue
				}

			}

			// if model.dynamicBatcher {
			// 	for _, m := range model.DynamicBatchModels {
			// 		for _, model := range m {
			// 			if model.Root() == nil {
			// 				camera.DrawDebugWireframe(screen, model, color)
			// 			}
			// 		}
			// 	}
			// }

			for _, meshPart := range model.Mesh.MeshParts {

				globalSortingTriangleBucket.sortMode = TriangleSortModeBackToFront
				if meshPart.Material != nil {
					globalSortingTriangleBucket.sortMode = meshPart.Material.TriangleSortMode
				}

				model.ProcessVertices(vpMatrix, camera, meshPart, false)

				globalSortingTriangleBucket.ForEach(func(triIndex, triID int, vertexIndices []int) {

					v0 := camera.clipToScreen(model.Mesh.vertexTransforms[vertexIndices[0]], vertexIndices[0], model, float32(camWidth), float32(camHeight), halfCamWidth, halfCamHeight, true)
					v1 := camera.clipToScreen(model.Mesh.vertexTransforms[vertexIndices[1]], vertexIndices[1], model, float32(camWidth), float32(camHeight), halfCamWidth, halfCamHeight, true)
					v2 := camera.clipToScreen(model.Mesh.vertexTransforms[vertexIndices[2]], vertexIndices[2], model, float32(camWidth), float32(camHeight), halfCamWidth, halfCamHeight, true)

					if (v0.X < 0 && v1.X < 0 && v2.X < 0) ||
						(v0.Y < 0 && v1.Y < 0 && v2.Y < 0) ||
						(v0.X > float32(camWidth) && v1.X > float32(camWidth) && v2.X > float32(camWidth)) ||
						(v0.Y > float32(camHeight) && v1.Y > float32(camHeight) && v2.Y > float32(camHeight)) {
						return
					}

					c := color.ToNRGBA64()
					vector.StrokeLine(screen, float32(v0.X), float32(v0.Y), float32(v1.X), float32(v1.Y), 1, c, false)
					vector.StrokeLine(screen, float32(v1.X), float32(v1.Y), float32(v2.X), float32(v2.Y), 1, c, false)
					vector.StrokeLine(screen, float32(v2.X), float32(v2.Y), float32(v0.X), float32(v0.Y), 1, c, false)

				})

			}

		} else if path, isPath := m.(*Path); isPath {

			points := path.Children()
			if path.Closed {
				points = append(points, points[0])
			}
			for i := 0; i < len(points)-1; i++ {
				p1 := camera.WorldToScreenPixels(points[i].WorldPosition())
				p2 := camera.WorldToScreenPixels(points[i+1].WorldPosition())
				vector.StrokeLine(screen, p1.X, p1.Y, p2.X, p2.Y, 1, color.ToNRGBA64(), false)
			}

		} else if grid, isGrid := m.(*Grid); isGrid {

			for _, point := range grid.Points() {

				p1 := camera.WorldToScreenPixels(point.WorldPosition())
				vector.StrokeCircle(screen, p1.X, p1.Y, 8, 1, image.White, false)
				for _, connection := range point.Connections {
					p2 := camera.WorldToScreenPixels(connection.To.WorldPosition())
					vector.StrokeLine(screen, p1.X, p1.Y, p2.X, p2.Y, 1, color.ToNRGBA64(), false)
				}

			}

		}

	}

}

// DrawDebugDrawOrder draws the drawing order of all triangles of all visible Models underneath the rootNode in the color provided to the screen
// image provided.
func (camera *Camera) DrawDebugDrawOrder(screen *ebiten.Image, rootNode INode, textScale float32, color Color) {

	vpMatrix := camera.ViewMatrix().Mult(camera.Projection())

	allModels := append([]INode{rootNode}, rootNode.SearchTree().INodes()...)

	for _, m := range allModels {

		if model, isModel := m.(*Model); isModel {

			if model.FrustumCulling {

				model.Transform()
				if !camera.SphereInFrustum(model.frustumCullingSphere) {
					continue
				}

			}

			for _, meshPart := range model.Mesh.MeshParts {

				globalSortingTriangleBucket.sortMode = TriangleSortModeBackToFront
				if meshPart.Material != nil {
					globalSortingTriangleBucket.sortMode = meshPart.Material.TriangleSortMode
				}

				model.ProcessVertices(vpMatrix, camera, meshPart, false)

				globalSortingTriangleBucket.ForEach(func(triIndex, triID int, vertexIndices []int) {

					screenPos := camera.WorldToScreenPixels(model.Transform().MultVec(model.Mesh.Triangles[triID].Center))

					camera.DrawDebugText(screen, fmt.Sprintf("%d", triIndex), screenPos.X, screenPos.Y+(textScale*16), textScale, color)

				})

			}

		}

	}

}

// DrawDebugDrawCallCount draws the draw call count of all visible Models underneath the rootNode in the color provided to the screen
// image provided.
func (camera *Camera) DrawDebugDrawCallCount(screen *ebiten.Image, rootNode INode, textScale float32, color Color) {

	allModels := append([]INode{rootNode}, rootNode.SearchTree().INodes()...)

	for _, m := range allModels {

		if model, isModel := m.(*Model); isModel {

			if model.FrustumCulling {

				model.Transform()
				if !camera.SphereInFrustum(model.frustumCullingSphere) {
					continue
				}

			}

			screenPos := camera.WorldToScreenPixels(model.WorldPosition())

			camera.DrawDebugText(screen, fmt.Sprintf("%d", len(model.Mesh.MeshParts)), screenPos.X, screenPos.Y+(textScale*16), textScale, color)

		}

	}

}

// DrawDebugNormals draws the normals of visible models underneath the rootNode given to the screen. NormalLength is the length of the normal lines
// in units. Color is the color to draw the normals.
func (camera *Camera) DrawDebugNormals(screen *ebiten.Image, rootNode INode, normalLength float32, color Color) {

	allModels := append([]INode{rootNode}, rootNode.SearchTree().INodes()...)

	for _, m := range allModels {

		if model, isModel := m.(*Model); isModel {

			if model.FrustumCulling {

				model.Transform()
				if !camera.SphereInFrustum(model.frustumCullingSphere) {
					continue
				}

			}

			for _, tri := range model.Mesh.Triangles {

				center := camera.WorldToScreenPixels(model.Transform().MultVecW(tri.Center).To3D())
				transformedNormal := camera.WorldToScreenPixels(model.Transform().MultVecW(tri.Center.Add(tri.Normal.Scale(normalLength))).To3D())
				vector.StrokeLine(screen, center.X, center.Y, transformedNormal.X, transformedNormal.Y, 1, color.ToNRGBA64(), false)

			}

		}

	}

}

// DrawDebugCenters draws the center positions of nodes under the rootNode using the color given to the screen image provided.
func (camera *Camera) DrawDebugCenters(screen *ebiten.Image, rootNode INode, color Color) {

	allModels := append([]INode{rootNode}, rootNode.SearchTree().INodes()...)

	c := color.ToNRGBA64()

	for _, node := range allModels {

		if node == camera {
			continue
		}

		px := camera.WorldToScreenPixels(node.WorldPosition())
		vector.StrokeCircle(screen, px.X, px.Y, 6, 1, c, false)

		// If the node's parent is something, and its parent's parent is something (i.e. it's not the root)
		if node.Parent() != nil && node.Parent() != node.Root() {
			parentPos := camera.WorldToScreenPixels(node.Parent().WorldPosition())
			pos := camera.WorldToScreenPixels(node.WorldPosition())
			vector.StrokeLine(screen, pos.X, pos.Y, parentPos.X, parentPos.Y, 1, c, false)
		}

	}

}

func (camera *Camera) DrawDebugText(screen *ebiten.Image, txtStr string, posX, posY, textScale float32, color Color) {

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
	dr.ColorScale.Scale(0, 0, 0, 1)

	periodOffset := float32(8.0)

	for y := -1; y < 2; y++ {

		for x := -1; x < 2; x++ {

			dr.GeoM.Reset()
			dr.GeoM.Translate(float64(posX+4+float32(x)), float64(posY+4+float32(y)+periodOffset))
			dr.GeoM.Scale(float64(textScale), float64(textScale))

			screen.DrawImage(camera.debugTextTexture, dr)
		}

	}

	dr.ColorScale.Reset()
	dr.ColorScale.ScaleWithColor(color.ToNRGBA64())

	dr.GeoM.Reset()
	dr.GeoM.Translate(float64(posX+4), float64(posY+4+periodOffset))
	dr.GeoM.Scale(float64(textScale), float64(textScale))

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

// NormalTexture returns the camera's final result normal texture from any previous Render() or RenderNodes() calls. If Camera.RenderNormals is set to false,
// the function will return nil instead.
func (camera *Camera) NormalTexture() *ebiten.Image {
	if !camera.RenderNormals {
		return nil
	}
	return camera.resultNormalTexture
}

// AccumulationColorTexture returns the camera's final result accumulation color texture from previous renders. If the Camera's AccumulateColorMode
// property is set to AccumulateColorModeNone, the function will return nil instead.
func (camera *Camera) AccumulationColorTexture() *ebiten.Image {
	if camera.AccumulationColorMode == AccumulationColorModeNone {
		return nil
	}
	return camera.resultAccumulatedColorTexture
}

// DrawDebugBoundsColoredSettings is a struct used to define how to draw debug information showing the positions of IBoundingObjects
// in a node tree.
type DrawDebugBoundsColoredSettings struct {
	RenderAABBs bool  // Whether BoundingAABBs should be rendered or not
	AABBColor   Color // The color used to render BoundingAABBs

	RenderSpheres bool  // Whether BoundingSpheres should be rendered or not
	SphereColor   Color // The color used to render BoundingSpheres

	RenderCapsules bool  // Whether BoundingCapsules should be rendered or not
	CapsuleColor   Color // The color used to render BoundingSpheres

	RenderTriangles               bool  // Whether BoundingTriangles should be rendered or not
	TrianglesColor                Color // The color used to render BoundingTriangles
	RenderTrianglesBackfaceCulled bool

	RenderTrianglesAABB bool  // Whether the AABB surrounding BoundingTriangles should be rendered not
	TrianglesAABBColor  Color // The color used to render the AABB surrounding BoundingTriangles

	RenderBroadphases bool  // Whether the broadphase cells surrounding BoundingTriangles should be rendered or not
	BroadphaseColor   Color // The color used to render broadphase cells
}

// DrawDebugBoundsColored will draw shapes approximating the shapes and positions of BoundingObjects underneath the rootNode. The shapes will
// be drawn in the color provided for each kind of bounding object to the screen image provided.
func (camera *Camera) DrawDebugBoundsColored(screen *ebiten.Image, rootNode INode, options DrawDebugBoundsColoredSettings) {

	allModels := append([]INode{rootNode}, rootNode.SearchTree().INodes()...)

	for _, n := range allModels {

		p, s, r := n.Transform().Inverted().Decompose()
		invertedCamPos := r.MultVec(camera.WorldPosition()).Add(p.Mult(Vector3{1 / s.X, 1 / s.Y, 1 / s.Z}))
		invertedCamForward := camera.WorldRotation().Forward().Invert()

		if b, isBounds := n.(IBoundingObject); isBounds {

			switch bounds := b.(type) {

			case *BoundingSphere:

				if options.RenderSpheres {
					camera.drawSphere(screen, bounds, options.SphereColor)
				}

			case *BoundingCapsule:

				if options.RenderCapsules {

					pos := bounds.WorldPosition()
					radius := bounds.WorldRadius()
					height := bounds.Height / 2

					uv := bounds.WorldRotation().Up()
					rv := bounds.WorldRotation().Right()
					fv := bounds.WorldRotation().Forward()

					u := camera.WorldToScreenPixels(pos.Add(uv.Scale(height)))

					ur := camera.WorldToScreenPixels(pos.Add(uv.Scale(height - radius)).Add(rv.Scale(radius)))
					ul := camera.WorldToScreenPixels(pos.Add(uv.Scale(height - radius)).Add(rv.Scale(-radius)))
					uf := camera.WorldToScreenPixels(pos.Add(uv.Scale(height - radius)).Add(fv.Scale(radius)))
					ub := camera.WorldToScreenPixels(pos.Add(uv.Scale(height - radius)).Add(fv.Scale(-radius)))

					d := camera.WorldToScreenPixels(pos.Add(uv.Scale(-height)))

					dr := camera.WorldToScreenPixels(pos.Add(uv.Scale(-(height - radius))).Add(rv.Scale(radius)))
					dl := camera.WorldToScreenPixels(pos.Add(uv.Scale(-(height - radius))).Add(rv.Scale(-radius)))
					df := camera.WorldToScreenPixels(pos.Add(uv.Scale(-(height - radius))).Add(fv.Scale(radius)))
					db := camera.WorldToScreenPixels(pos.Add(uv.Scale(-(height - radius))).Add(fv.Scale(-radius)))

					lines := []Vector3{
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
						vector.StrokeLine(screen, start.X, start.Y, end.X, end.Y, 1, options.CapsuleColor.ToNRGBA64(), false)

					}

				}

			case *BoundingAABB:

				if options.RenderAABBs {

					pos := bounds.WorldPosition()
					size := bounds.Dimensions.Size().Scale(0.5)

					ufr := camera.WorldToScreenPixels(pos.Add(Vector3{size.X, size.Y, size.Z}))
					ufl := camera.WorldToScreenPixels(pos.Add(Vector3{-size.X, size.Y, size.Z}))
					ubr := camera.WorldToScreenPixels(pos.Add(Vector3{size.X, size.Y, -size.Z}))
					ubl := camera.WorldToScreenPixels(pos.Add(Vector3{-size.X, size.Y, -size.Z}))

					dfr := camera.WorldToScreenPixels(pos.Add(Vector3{size.X, -size.Y, size.Z}))
					dfl := camera.WorldToScreenPixels(pos.Add(Vector3{-size.X, -size.Y, size.Z}))
					dbr := camera.WorldToScreenPixels(pos.Add(Vector3{size.X, -size.Y, -size.Z}))
					dbl := camera.WorldToScreenPixels(pos.Add(Vector3{-size.X, -size.Y, -size.Z}))

					lines := []Vector3{
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
						vector.StrokeLine(screen, start.X, start.Y, end.X, end.Y, 1, options.AABBColor.ToNRGBA64(), false)

					}

				}

			case *BoundingTriangles:

				if options.RenderBroadphases {

					for _, b := range bounds.Broadphase.allAABBPositions() {
						camera.DrawDebugBoundsColored(screen, b, DrawDebugBoundsColoredSettings{
							RenderAABBs: true,
							AABBColor:   options.BroadphaseColor,
						})
					}

				}

				if options.RenderTriangles {

					camWidth, camHeight := camera.Size()

					lines := []Vector3{}

					mesh := bounds.Mesh

					halfCamWidth, halfCamHeight := float32(camWidth)/2, float32(camHeight)/2

					transformNoLoc := bounds.Transform()
					transformNoLoc.SetRow(3, Vector4{0, 0, 0, 1})

					for _, tri := range mesh.Triangles {

						mvpMatrix := bounds.Transform().Mult(camera.ViewMatrix().Mult(camera.Projection()))

						if options.RenderTrianglesBackfaceCulled && ((camera.perspective && tri.Normal.Dot(invertedCamPos.Sub(tri.Center)) < 0) || (!camera.perspective && tri.Normal.Dot(invertedCamForward) > 0)) {
							continue
						}

						v0 := camera.clipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.VertexIndices[0]]), 0, nil, float32(camWidth), float32(camHeight), halfCamWidth, halfCamHeight, false).To3D()
						v1 := camera.clipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.VertexIndices[1]]), 0, nil, float32(camWidth), float32(camHeight), halfCamWidth, halfCamHeight, false).To3D()
						v2 := camera.clipToScreen(mvpMatrix.MultVecW(mesh.VertexPositions[tri.VertexIndices[2]]), 0, nil, float32(camWidth), float32(camHeight), halfCamWidth, halfCamHeight, false).To3D()

						if (v0.X < 0 && v1.X < 0 && v2.X < 0) ||
							(v0.Y < 0 && v1.Y < 0 && v2.Y < 0) ||
							(v0.X > float32(camWidth) && v1.X > float32(camWidth) && v2.X > float32(camWidth)) ||
							(v0.Y > float32(camHeight) && v1.Y > float32(camHeight) && v2.Y > float32(camHeight)) {
							continue
						}

						lines = append(lines, v0, v1, v2)

					}

					triColor := options.TrianglesColor.ToNRGBA64()

					for i := 0; i < len(lines); i += 3 {

						if i >= len(lines)-1 {
							break
						}

						start := lines[i]
						end := lines[i+1]
						vector.StrokeLine(screen, start.X, start.Y, end.X, end.Y, 1, triColor, false)

						start = lines[i+1]
						end = lines[i+2]
						vector.StrokeLine(screen, start.X, start.Y, end.X, end.Y, 1, triColor, false)

						start = lines[i+2]
						end = lines[i]
						vector.StrokeLine(screen, start.X, start.Y, end.X, end.Y, 1, triColor, false)

					}

				}

				if options.RenderTrianglesAABB {
					camera.DrawDebugBoundsColored(screen, bounds.BoundingAABB, DrawDebugBoundsColoredSettings{
						RenderTrianglesAABB: true,
						TrianglesAABBColor:  options.TrianglesAABBColor,
					})
				}

			}

		}

	}

}

// DefaultDrawDebugBoundsSettings returns the default settings for drawing the debug bounds for a nodegraph.
func DefaultDrawDebugBoundsSettings() DrawDebugBoundsColoredSettings {
	return DrawDebugBoundsColoredSettings{
		RenderAABBs:         true,
		RenderSpheres:       true,
		RenderCapsules:      true,
		RenderTriangles:     true,
		RenderTrianglesAABB: true,
		RenderBroadphases:   false,

		AABBColor:          NewColor(0, 0.25, 1, 0.5),
		SphereColor:        NewColor(0.5, 0.25, 1.0, 0.5),
		CapsuleColor:       NewColor(0.25, 1, 0, 0.5),
		TrianglesColor:     NewColor(0, 0, 0, 0.25),
		TrianglesAABBColor: NewColor(1, 0.5, 0, 0.25),
		BroadphaseColor:    NewColor(1, 0, 0, 0.1),
	}
}

// DrawDebugFrustums will draw shapes approximating the frustum spheres for objects underneath the rootNode.
// The shapes will be drawn in the color provided to the screen image provided.
func (camera *Camera) DrawDebugFrustums(screen *ebiten.Image, rootNode INode, color Color) {

	allModels := append([]INode{rootNode}, rootNode.SearchTree().INodes()...)

	for _, n := range allModels {

		if model, ok := n.(*Model); ok {

			bounds := model.frustumCullingSphere
			camera.drawSphere(screen, bounds, color)

		}

	}

}

var debugIcosphereMesh = NewIcosphereMesh(1)
var debugIcosphere = NewModel("debug icosphere", debugIcosphereMesh)

func (camera *Camera) drawSphere(screen *ebiten.Image, sphere *BoundingSphere, color Color) {

	debugIcosphere.SetLocalPositionVec(sphere.WorldPosition())
	s := sphere.WorldRadius()
	debugIcosphere.SetLocalScaleVec(Vector3{s, s, s})
	debugIcosphere.Transform()
	camera.DrawDebugWireframe(screen, debugIcosphere, color)

}

/////

// Type returns the NodeType for this object.
func (camera *Camera) Type() NodeType {
	return NodeTypeCamera
}
