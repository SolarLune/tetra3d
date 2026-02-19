package tetra3d

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/solarlune/tetra3d/math32"
)

// ILight represents an interface that is fulfilled by an object that emits light, returning the color a vertex should be given that Vertex and its model matrix.
type ILight interface {
	INode
	// beginRender is used to call any set-up code or to prepare math structures that are used when lighting the scene.
	// It gets called once when first rendering a set of Nodes.
	beginRender()

	// beginModel is, similarly to beginRender(), used to prepare or precompute any math necessary when lighting the scene.
	// It gets called once before lighting all visible triangles of a given Model.
	beginModel(model *Model)

	Light(meshPart *MeshPart, model *Model, targetColors VertexColorChannel, onlyVisible bool) // Light lights the triangles in the MeshPart, storing the result in the targetColors
	// color buffer. If onlyVisible is true, only the visible vertices will be lit; if it's false, they will all be lit.
	On() bool      // On is simply used to tell if a "generic" Light is on or not.
	SetOn(on bool) // SetOn sets whether the light is on or not

	Color() Color
	SetColor(c Color)
	Energy() float32
	SetEnergy(energy float32)
}

//---------------//

// AmbientLight represents an ambient light that colors the entire Scene.
type AmbientLight struct {
	*Node
	color Color // Color is the color of the PointLight.
	// energy is the overall energy of the Light. Internally, technically there's no difference between a brighter color and a
	// higher energy, but this is here for convenience / adherance to GLTF / 3D modelers.
	energy float32
	on     bool // If the light is on and contributing to the scene.

	result [3]float32
}

// NewAmbientLight returns a new AmbientLight.
func NewAmbientLight(name string, r, g, b, energy float32) *AmbientLight {
	amb := &AmbientLight{
		Node:   NewNode(name),
		color:  NewColor(r, g, b, 1),
		energy: energy,
		on:     true,
	}
	amb.owner = amb
	return amb
}

func (amb *AmbientLight) Clone() INode {

	clone := NewAmbientLight(amb.name, amb.color.R, amb.color.G, amb.color.B, amb.energy)
	clone.on = amb.on

	clone.Node = amb.Node.clone(clone).(*Node)

	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}

	return clone

}

func (amb *AmbientLight) beginRender() {
	amb.result = [3]float32{amb.color.R * amb.energy, amb.color.G * amb.energy, amb.color.B * amb.energy}
}

func (amb *AmbientLight) beginModel(model *Model) {}

// Light returns the light level for the ambient light. It doesn't use the provided Triangle; it takes it as an argument to simply adhere to the Light interface.
func (amb *AmbientLight) Light(meshPart *MeshPart, model *Model, targetColors VertexColorChannel, onlyVisible bool) {
	meshPart.ForEachVertexIndex(func(vertIndex int) {
		targetColors[vertIndex].R += amb.result[0]
		targetColors[vertIndex].G += amb.result[1]
		targetColors[vertIndex].B += amb.result[2]
	}, onlyVisible)
}

func (amb *AmbientLight) On() bool {
	return amb.on && amb.energy > 0
}

func (amb *AmbientLight) SetOn(on bool) {
	amb.on = on
}

func (amb *AmbientLight) Color() Color {
	return amb.color
}

func (amb *AmbientLight) SetColor(color Color) {
	amb.color = color
}

func (amb *AmbientLight) Energy() float32 {
	return amb.energy
}

func (amb *AmbientLight) SetEnergy(energy float32) {
	amb.energy = energy
}

// Type returns the NodeType for this object.
func (amb *AmbientLight) Type() NodeType {
	return NodeTypeAmbientLight
}

//---------------//

// PointLight represents a point light (naturally).
type PointLight struct {
	*Node
	// Range represents the distance after which the light fully attenuates. If this is 0 (the default),
	// it falls off using something akin to the inverse square law.
	Range float32
	// color is the color of the PointLight.
	color Color
	// energy is the overall energy of the Light, with 1.0 being full brightness. Internally, technically there's no
	// difference between a brighter color and a higher energy, but this is here for convenience / adherance to the
	// GLTF spec and 3D modelers.
	energy float32
	// If the light is on and contributing to the scene.
	on bool

	rangeSquared    float32
	workingPosition Vector3
}

// NewPointLight creates a new Point light.
func NewPointLight(name string, r, g, b, energy float32) *PointLight {
	point := &PointLight{
		Node:   NewNode(name),
		energy: energy,
		color:  NewColor(r, g, b, 1),
		on:     true,
	}
	point.owner = point
	return point
}

// Clone returns a new clone of the given point light.
func (p *PointLight) Clone() INode {

	clone := NewPointLight(p.name, p.color.R, p.color.G, p.color.B, p.energy)
	clone.on = p.on
	clone.Range = p.Range

	clone.Node = p.Node.clone(clone).(*Node)

	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}

	return clone

}

func (p *PointLight) beginRender() {
	p.rangeSquared = p.Range * p.Range
}

func (p *PointLight) beginModel(model *Model) {

	pos, sca, rot := model.Transform().Inverted().Decompose()

	// Rather than transforming all vertices of all triangles of a mesh, we can just transform the
	// point light's position by the inversion of the model's transform to get the same effect and save processing time.
	// The same technique is used for Sphere - Triangle collision in bounds.go.

	if model.skinned {
		p.workingPosition = p.WorldPosition()
	} else {
		p.workingPosition = rot.MultVec(p.WorldPosition()).Add(pos.Mult(Vector3{1 / sca.X, 1 / sca.Y, 1 / sca.Z}))
	}

}

// Light returns the R, G, and B values for the PointLight for all vertices of a given Triangle.
func (p *PointLight) Light(meshPart *MeshPart, model *Model, targetColors VertexColorChannel, onlyVisible bool) {

	// We calculate both the eye vector as well as the light vector so that if the camera passes behind the
	// lit face and backface culling is off, the triangle can still be lit or unlit from the other side. Otherwise,
	// if the triangle were lit by a light, it would appear lit regardless of the positioning of the camera.

	meshPart.ForEachVertexIndex(func(index int) {

		// TODO: Make lighting faster by returning early if the triangle is too far from the point light position

		var vertPos, vertNormal Vector3

		if model.skinned {
			vertPos = model.Mesh.vertexSkinnedPositions[index]
			vertNormal = model.Mesh.vertexSkinnedNormals[index]
		} else {
			vertPos = model.Mesh.VertexPositions[index]
			vertNormal = model.Mesh.VertexNormals[index]
		}

		distance := p.workingPosition.DistanceSquaredTo(vertPos)

		if p.Range > 0 {

			if distance > p.rangeSquared {
				return
			}

			// var triCenter Vector

			// if model.skinned {
			// 	v0 := model.Mesh.vertexSkinnedPositions[tri.VertexIndices[0]].Clone()
			// 	v1 := model.Mesh.vertexSkinnedPositions[tri.VertexIndices[1]]
			// 	v2 := model.Mesh.vertexSkinnedPositions[tri.VertexIndices[2]]
			// 	triCenter = Vector(vector.In(v0).Add(v1).Add(v2).Scale(1.0 / 3.0))
			// } else {
			// 	triCenter = tri.Center
			// }

			// dist := fastVectorDistanceSquared(point.workingPosition, triCenter)
			// if dist > (point.distanceSquared + (tri.MaxSpan * tri.MaxSpan)) {
			// 	return
			// }

		} else {
			if 1/distance*float32(p.energy) < 0.001 {
				return
			}
		}

		// If you're on the other side of the plane, just assume it's not visible.
		// if dot(model.Mesh.Triangles[triIndex].Normal, fastVectorSub(triCenter, point.cameraPosition).Unit()) < 0 {
		// 	return light
		// }

		lightVec := p.workingPosition.Sub(vertPos).Unit()
		if mat := meshPart.Material; mat != nil && mat.LightingMode == LightingModeFixedNormals {
			vertNormal = lightVec
		}

		diffuse := vertNormal.Dot(lightVec)

		if mat := meshPart.Material; mat != nil && mat.LightingMode == LightingModeDoubleSided {
			diffuse = math32.Abs(diffuse)
		}

		if diffuse > 0 {

			diffuseFactor := diffuse * (1.0 / (1.0 + (0.1 * distance))) * 2

			if p.Range > 0 {
				distClamp := math32.Clamp((p.rangeSquared-distance)/distance, 0, 1)
				diffuseFactor *= distClamp
			}

			targetColors[index].R += p.color.R * float32(diffuseFactor) * p.energy
			targetColors[index].G += p.color.G * float32(diffuseFactor) * p.energy
			targetColors[index].B += p.color.B * float32(diffuseFactor) * p.energy

		}

	}, onlyVisible)

}

func (p *PointLight) On() bool {
	return p.on && p.energy > 0
}

func (p *PointLight) SetOn(on bool) {
	p.on = on
}

func (p *PointLight) Color() Color {
	return p.color
}

func (p *PointLight) SetColor(color Color) {
	p.color = color
}

func (p *PointLight) Energy() float32 {
	return p.energy
}

func (p *PointLight) SetEnergy(energy float32) {
	p.energy = energy
}

// Type returns the NodeType for this object.
func (p *PointLight) Type() NodeType {
	return NodeTypePointLight
}

//---------------//

// DirectionalLight represents a directional light of infinite distance.
type DirectionalLight struct {
	*Node
	color Color // Color is the color of the light.
	// energy is the overall energy of the light. Internally, technically there's no difference between a brighter color and a
	// higher energy, but this is here for convenience / adherance to GLTF / 3D modelers.
	energy float32
	on     bool // If the light is on and contributing to the scene.

	workingForward       Vector3 // Internal forward vector so we don't have to calculate it for every triangle for every model using this light.
	workingModelRotation Matrix4 // Similarly, this is an internal rotational transform (without the transformation row) for the Model being lit.
}

// NewDirectionalLight creates a new Directional Light with the specified RGB color and energy (assuming 1.0 energy is standard / "100%" lighting).
func NewDirectionalLight(name string, r, g, b, energy float32) *DirectionalLight {
	sun := &DirectionalLight{
		Node:   NewNode(name),
		color:  NewColor(r, g, b, 1),
		energy: energy,
		on:     true,
	}
	sun.owner = sun
	return sun
}

// Clone returns a new DirectionalLight clone from the given DirectionalLight.
func (sun *DirectionalLight) Clone() INode {

	clone := NewDirectionalLight(sun.name, sun.color.R, sun.color.G, sun.color.B, sun.energy)

	clone.on = sun.on

	clone.Node = sun.Node.clone(clone).(*Node)
	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}

	return clone

}

func (sun *DirectionalLight) beginRender() {
	sun.workingForward = sun.WorldRotation().Forward() // Already reversed
}

func (sun *DirectionalLight) beginModel(model *Model) {
	if !model.skinned {
		sun.workingModelRotation = model.WorldRotation().Inverted().Transposed()
	}
}

// Light returns the R, G, and B values for the DirectionalLight for each vertex of the provided Triangle.
func (sun *DirectionalLight) Light(meshPart *MeshPart, model *Model, targetColors VertexColorChannel, onlyVisible bool) {

	meshPart.ForEachVertexIndex(func(index int) {

		var normal Vector3
		if model.skinned {
			// If it's skinned, we don't have to calculate the normal, as that's been pre-calc'd for us
			normal = model.Mesh.vertexSkinnedNormals[index]
		} else {
			normal = sun.workingModelRotation.MultVec(model.Mesh.VertexNormals[index])
		}

		if mat := meshPart.Material; mat != nil && mat.LightingMode == LightingModeFixedNormals {
			normal = sun.workingForward
		}

		diffuseFactor := normal.Dot(sun.workingForward)

		if mat := meshPart.Material; mat != nil && mat.LightingMode == LightingModeDoubleSided {
			diffuseFactor = math32.Abs(diffuseFactor)
		}

		if diffuseFactor <= 0 {
			return
		}

		targetColors[index].R += sun.color.R * float32(diffuseFactor) * sun.energy
		targetColors[index].G += sun.color.G * float32(diffuseFactor) * sun.energy
		targetColors[index].B += sun.color.B * float32(diffuseFactor) * sun.energy

	}, onlyVisible)

}

func (sun *DirectionalLight) On() bool {
	return sun.on && sun.energy > 0
}

func (sun *DirectionalLight) SetOn(on bool) {
	sun.on = on
}

func (d *DirectionalLight) Color() Color {
	return d.color
}

func (d *DirectionalLight) SetColor(color Color) {
	d.color = color
}

func (d *DirectionalLight) Energy() float32 {
	return d.energy
}

func (d *DirectionalLight) SetEnergy(energy float32) {
	d.energy = energy
}

// Type returns the NodeType for this object.
func (sun *DirectionalLight) Type() NodeType {
	return NodeTypeDirectionalLight
}

// CubeLight represents an AABB volume that lights triangles.
type CubeLight struct {
	*Node
	Dimensions Dimensions // The overall dimensions of the CubeLight.
	energy     float32    // The overall energy of the CubeLight
	color      Color      // The color of the CubeLight
	on         bool       // If the CubeLight is on or not
	// A value between 0 and 1 indicating how much opposite faces are still lit within the volume (i.e. at LightBleed = 0.0,
	// faces away from the light are dark; at 1.0, faces away from the light are fully illuminated)
	Bleed             float32
	LightingAngle     Vector3 // The direction in which light is shining. Defaults to local Y down (0, -1, 0).
	workingDimensions Dimensions
	workingPosition   Vector3
	workingAngle      Vector3
}

// NewCubeLight creates a new CubeLight with the given dimensions.
func NewCubeLight(name string, dimensions Dimensions) *CubeLight {
	cube := &CubeLight{
		Node: NewNode(name),
		// Dimensions:    Dimensions{{-w / 2, -h / 2, -d / 2}, {w / 2, h / 2, d / 2}},
		Dimensions:    dimensions,
		energy:        1,
		color:         NewColor(1, 1, 1, 1),
		on:            true,
		LightingAngle: Vector3{0, -1, 0},
	}
	cube.owner = cube
	return cube
}

// NewCubeLightFromModel creates a new CubeLight from the Model's dimensions and world transform.
// Note that this does not add it to the Model's hierarchy in any way - the newly created CubeLight
// is still its own Node.
func NewCubeLightFromModel(name string, model *Model) *CubeLight {
	cube := NewCubeLight(name, model.Mesh.Dimensions)
	cube.SetWorldTransform(model.Transform())
	cube.LightingAngle = model.WorldRotation().Up().Invert()
	return cube
}

// Clone clones the CubeLight, returning a deep copy.
func (cube *CubeLight) Clone() INode {
	newCube := NewCubeLight(cube.name, cube.Dimensions)
	newCube.energy = cube.energy
	newCube.color = cube.color
	newCube.on = cube.on
	newCube.Bleed = cube.Bleed
	newCube.LightingAngle = cube.LightingAngle
	newCube.SetWorldTransform(cube.Transform())
	newCube.Node = cube.Node.clone(newCube).(*Node)
	if runCallbacks && newCube.Callbacks().OnClone != nil {
		newCube.Callbacks().OnClone(newCube)
	}
	return newCube
}

// TransformedDimensions returns the AABB volume dimensions of the CubeLight as they have been scaled, rotated, and positioned to follow the
// CubeLight's node.
func (cube *CubeLight) TransformedDimensions() Dimensions {

	p, s, r := cube.Transform().Decompose()

	corners := [][]float32{
		{1, 1, 1},
		{1, -1, 1},
		{-1, 1, 1},
		{-1, -1, 1},
		{1, 1, -1},
		{1, -1, -1},
		{-1, 1, -1},
		{-1, -1, -1},
	}

	dimensions := NewEmptyDimensions()

	for _, c := range corners {

		position := r.MultVec(Vector3{
			(cube.Dimensions.Width() * c[0] * s.X / 2),
			(cube.Dimensions.Height() * c[1] * s.Y / 2),
			(cube.Dimensions.Depth() * c[2] * s.Z / 2),
		})

		if dimensions.Min.X > position.X {
			dimensions.Min.X = position.X
		}

		if dimensions.Min.Y > position.Y {
			dimensions.Min.Y = position.Y
		}

		if dimensions.Min.Z > position.Z {
			dimensions.Min.Z = position.Z
		}

		if dimensions.Max.X < position.X {
			dimensions.Max.X = position.X
		}

		if dimensions.Max.Y < position.Y {
			dimensions.Max.Y = position.Y
		}

		if dimensions.Max.Z < position.Z {
			dimensions.Max.Z = position.Z
		}

	}

	dimensions.Min = dimensions.Min.Add(p)
	dimensions.Max = dimensions.Max.Add(p)

	return dimensions

}

func (cube *CubeLight) beginRender() {
	cube.workingAngle = cube.WorldRotation().MultVec(cube.LightingAngle.Invert())
}

func (cube *CubeLight) beginModel(model *Model) {

	p, s, r := model.Transform().Inverted().Decompose()
	_, _, _ = p, s, r

	// Rather than transforming all vertices of all triangles of a mesh, we can just transform the
	// point light's position by the inversion of the model's transform to get the same effect and save processing time.
	// The same technique is used for Sphere - Triangle collision in bounds.go.

	lightStartPos := cube.WorldPosition().Add(cube.LightingAngle.Invert().Scale(cube.Dimensions.Height() / 2))

	cube.workingDimensions = cube.TransformedDimensions()

	if model.skinned {
		cube.workingPosition = lightStartPos
	} else {

		sDen := Vector3{1 / s.X, 1 / s.Y, 1 / s.Z}

		cube.workingPosition = r.MultVec(lightStartPos).Add(p.Mult(sDen))

		// point.workingPosition = r.MultVec(point.WorldPosition()).Add(p.Mult(Vector{1 / s.X, 1 / s.Y, 1 / s.Z, s.W}))

		cube.workingDimensions.Min = r.MultVec(cube.workingDimensions.Min.Mult(s))
		cube.workingDimensions.Max = r.MultVec(cube.workingDimensions.Max.Mult(s))

		cube.workingDimensions.Min = cube.workingDimensions.Min.Add(p)
		cube.workingDimensions.Max = cube.workingDimensions.Max.Add(p)

	}

}

// Light returns the R, G, and B values for the PointLight for all vertices of a given Triangle.
func (cube *CubeLight) Light(meshPart *MeshPart, model *Model, targetColors VertexColorChannel, onlyVisible bool) {

	meshPart.ForEachVertexIndex(func(index int) {

		// TODO: Make lighting faster by returning early if the triangle is too far from the point light position

		// We calculate both the eye vector as well as the light vector so that if the camera passes behind the
		// lit face and backface culling is off, the triangle can still be lit or unlit from the other side. Otherwise,
		// if the triangle were lit by a light, it would appear lit regardless of the positioning of the camera.

		// var triCenter Vector

		// if model.skinned {
		// 	v0 := model.Mesh.vertexSkinnedPositions[triIndex*3].Clone()
		// 	v1 := model.Mesh.vertexSkinnedPositions[triIndex*3+1]
		// 	v2 := model.Mesh.vertexSkinnedPositions[triIndex*3+2]
		// 	vector.In(v0).Add(v1).Add(v2).Scale(1.0 / 3.0)
		// 	triCenter = v0
		// } else {
		// 	triCenter = model.Mesh.Triangles[triIndex].Center
		// }

		// dist := fastVectorDistanceSquared(cube.workingPosition, triCenter)

		// if cube.Distance > 0 && dist > distanceSquared {
		// 	return light
		// }

		// If you're on the other side of the plane, just assume it's not visible.
		// if dot(model.Mesh.Triangles[triIndex].Normal, fastVectorSub(triCenter, point.cameraPosition).Unit()) < 0 {
		// 	return light
		// }

		var vertPos, vertNormal Vector3

		if model.skinned {
			vertPos = model.Mesh.vertexSkinnedPositions[index]
			vertNormal = model.Mesh.vertexSkinnedNormals[index]
		} else {
			vertPos = model.Mesh.VertexPositions[index]
			vertNormal = model.Mesh.VertexNormals[index]
		}

		var diffuse, diffuseFactor float32

		if mat := meshPart.Material; mat != nil && mat.LightingMode == LightingModeFixedNormals {
			vertNormal = cube.workingAngle
		}

		diffuse = vertNormal.Dot(cube.workingAngle)

		if mat := meshPart.Material; mat != nil && mat.LightingMode == LightingModeDoubleSided {
			diffuse = math32.Abs(diffuse)
		}

		if cube.Bleed > 0 {

			if diffuse < 0 {
				diffuse = 0
			}

			// diffuse += cube.Bleed
			diffuse = cube.Bleed + ((1 - cube.Bleed) * diffuse)

			if diffuse > 1 {
				diffuse = 1
			}

		}

		if diffuse < 0 {
			return
		} else {

			if !vertPos.IsInsideDimensions(cube.workingDimensions) {
				diffuseFactor = 0
			} else {
				diffuseFactor = diffuse
			}

		}

		if diffuseFactor <= 0 {
			return
		}

		targetColors[index].R += cube.color.R * float32(diffuseFactor) * cube.energy
		targetColors[index].G += cube.color.G * float32(diffuseFactor) * cube.energy
		targetColors[index].B += cube.color.B * float32(diffuseFactor) * cube.energy

	}, onlyVisible)

}

// On returns if the CubeLight is on or not.
func (cube *CubeLight) On() bool {
	return cube.on
}

// SetOn sets the CubeLight to be on or off.
func (cube *CubeLight) SetOn(on bool) {
	cube.on = on
}

func (cube *CubeLight) Color() Color {
	return cube.color
}

func (cube *CubeLight) SetColor(color Color) {
	cube.color = color
}

func (cube *CubeLight) Energy() float32 {
	return cube.energy
}

func (cube *CubeLight) SetEnergy(energy float32) {
	cube.energy = energy
}

// Type returns the type of INode this is (NodeTypeCubeLight).
func (cube *CubeLight) Type() NodeType {
	return NodeTypeCubeLight
}

/////

// func LightVolumeSpecialCopy() Color {
// 	return NewColor(lightVolumeSpecialCopy, 0, 0, 0)
// }

// const lightVolumeSpecialCopy = -99899

// var LightVolumeSpecialCopy = NewColor(-9999, -8888, -7777, 0)

// LightVolume represents a grid of colors representing light to color models that enter the space with.
type LightVolume struct {
	*Node

	cellSizeX float32
	cellSizeY float32
	cellSizeZ float32

	cellCountX int
	cellCountY int
	cellCountZ int

	dimensions Dimensions

	lightValues []Color

	energy float32
	color  Color
	on     bool
}

// Creates a new light volume node of the given name, transformed dimensions, and light grid cellular size.
func NewLightVolume(name string, volumeDim Dimensions, cellSizeX, cellSizeY, cellSizeZ float32) *LightVolume {
	lv := &LightVolume{
		Node:   NewNode(name),
		energy: 1,
		on:     true,
		color:  NewColor(1, 1, 1, 1),
	}

	lv.owner = lv

	lv.Resize(volumeDim, cellSizeX, cellSizeY, cellSizeZ)

	return lv
}

// Creates a new light volume node from the given Model's name and dimensions, with the given cellular size.
// Note that the model will still exist, so you'll need to dispose of it as you'd like.
func NewLightVolumeFromModel(model *Model, cellSizeX, cellSizeY, cellSizeZ float32) *LightVolume {
	vol := NewLightVolume(model.Name(), model.Dimensions(), cellSizeX, cellSizeY, cellSizeZ)
	vol.Node.SetWorldTransform(model.Transform())
	return vol
}

// Clones the LightVolume and returns another.
func (l *LightVolume) Clone() INode {
	newLightmap := NewLightVolume(l.Name(), l.dimensions, l.cellSizeX, l.cellSizeY, l.cellSizeZ)
	newLightmap.lightValues = append(newLightmap.lightValues, l.lightValues...)
	newLightmap.energy = l.energy
	newLightmap.color = l.color
	newLightmap.on = l.on
	return newLightmap
}

// Resizes the LightVolume to be of the given transformed dimensions and cellular size.
func (l *LightVolume) Resize(dimensions Dimensions, cellsizeX, cellsizeY, cellsizeZ float32) {

	l.dimensions = dimensions
	l.cellSizeX = cellsizeX
	l.cellSizeY = cellsizeY
	l.cellSizeZ = cellsizeZ

	cellCountX := 0
	cellCountY := 0
	cellCountZ := 0

	for z := dimensions.Min.Z; z < dimensions.Max.Z; z += cellsizeZ {
		cellCountZ++
		cellCountY = 0

		for y := dimensions.Min.Y; y < dimensions.Max.Y; y += cellsizeY {
			cellCountY++
			cellCountX = 0

			for x := dimensions.Min.X; x < dimensions.Max.X; x += cellsizeX {
				cellCountX++
				l.lightValues = append(l.lightValues, NewColor(0, 0, 0, 1))
			}

		}

	}

	l.cellCountX = cellCountX
	l.cellCountY = cellCountY
	l.cellCountZ = cellCountZ

}

func (l *LightVolume) beginRender() {}

func (l *LightVolume) beginModel(model *Model) {}

// Lights the mesh part of the model being rendered, placing the results into the targeted VertexColorChannel, only doing so
// for the visible parts of the mesh if onlyVisible is set to true.
func (l *LightVolume) Light(meshPart *MeshPart, model *Model, targetColors VertexColorChannel, onlyVisible bool) {

	cellColor := NewColor(0, 0, 0, 1)

	// TODO: If distance from the mesh mesh part is too big, then we can just forget about it?

	transform := model.Transform()

	center := l.dimensions.Center()

	// Half the max span so that each vertex is, at maximum, as far from the center
	// as the distance from min corner to max corner / 2
	d := l.dimensions.MaxSpan() / 2
	maxDist := d * d

	meshPart.ForEachVertexIndex(func(vertIndex int) {

		var vertPos Vector3

		if model.skinned {
			vertPos = model.Mesh.vertexSkinnedPositions[vertIndex]
		} else {
			vertPos = transform.MultVec(model.Mesh.VertexPositions[vertIndex])
		}

		if vertPos.DistanceSquaredTo(center) > maxDist {
			return
		}

		cellColor.R = 0
		cellColor.G = 0
		cellColor.B = 0

		x, y, z := l.convertToXYZIndicesVec(vertPos)

		if l.xyzIndicesInsideVolume(x, y, z) {

			lvColor := l.getLightValue(x, y, z)

			cellColor.R = lvColor.R
			cellColor.G = lvColor.G
			cellColor.B = lvColor.B

		}

		targetColors[vertIndex].R += l.color.R * cellColor.R * l.energy
		targetColors[vertIndex].G += l.color.G * cellColor.G * l.energy
		targetColors[vertIndex].B += l.color.B * cellColor.B * l.energy

	}, onlyVisible)

}

func (l *LightVolume) convertToXYZIndices(wx, wy, wz float32) (x, y, z int) {

	x = int((wx - l.dimensions.Min.X) / l.cellSizeX)
	y = int((wy - l.dimensions.Min.Y) / l.cellSizeY)
	z = int((wz - l.dimensions.Min.Z) / l.cellSizeZ)

	return
}

func (l *LightVolume) convertToXYZIndicesVec(position Vector3) (x, y, z int) {
	return l.convertToXYZIndices(position.X, position.Y, position.Z)
}

func (l *LightVolume) xyzIndicesInsideVolume(x, y, z int) bool {
	return x >= 0 && y >= 0 && z >= 0 && x < l.cellCountX && y < l.cellCountY && z < l.cellCountZ
}

func (l *LightVolume) LightCellWriteToPosition(position Vector3, color Color) {

	x, y, z := l.convertToXYZIndicesVec(position)

	if l.xyzIndicesInsideVolume(x, y, z) {
		l.setLightValue(x, y, z, color)
	}

}

func (l *LightVolume) LightCellReadFromPosition(position Vector3, color Color) Color {

	x, y, z := l.convertToXYZIndicesVec(position)

	if l.xyzIndicesInsideVolume(x, y, z) {
		return l.getLightValue(x, y, z)
	}

	return NewColor(0, 0, 0, 0)

}

func (l *LightVolume) LightCellReadWriteAll(forEach func(position Vector3, in Color) Color) {

	if forEach == nil {
		return
	}

	csx := l.cellSizeX / 2
	csy := l.cellSizeY / 2
	csz := l.cellSizeZ / 2

	// Alreay transformed
	for wz := l.dimensions.Min.Z + csz; wz <= l.dimensions.Max.Z; wz += l.cellSizeZ {
		for wy := l.dimensions.Min.Y + csy; wy <= l.dimensions.Max.Y; wy += l.cellSizeY {
			for wx := l.dimensions.Min.X + csx; wx <= l.dimensions.Max.X; wx += l.cellSizeX {

				x, y, z := l.convertToXYZIndices(wx, wy, wz)

				if l.xyzIndicesInsideVolume(x, y, z) {
					out := forEach(NewVector3(wx, wy, wz), l.getLightValue(x, y, z))
					l.setLightValue(x, y, z, out)
				}

			}
		}
	}

}

// func (l *LightVolume) handleSpecialColors() {

// 	csx := l.cellSizeX / 2
// 	csy := l.cellSizeY / 2
// 	csz := l.cellSizeZ / 2

// 	// Alreay transformed
// 	for wz := l.dimensions.Min.Z + csz; wz <= l.dimensions.Max.Z; wz += l.cellSizeZ {
// 		for wy := l.dimensions.Min.Y + csy; wy <= l.dimensions.Max.Y; wy += l.cellSizeY {
// 			for wx := l.dimensions.Min.X + csx; wx <= l.dimensions.Max.X; wx += l.cellSizeX {

// 				x, y, z := l.convertToXYZIndices(wx, wy, wz)

// 				if l.xyzIndicesInsideVolume(x, y, z) {
// 					color := l.getLightValue(x, y, z)
// 					if color.R == lightVolumeSpecialCopy {
// 						set := false
// 						for nz := z - 1; nz <= z+1; nz++ {
// 							for ny := y - 1; ny <= y+1; ny++ {
// 								for nx := x - 1; nx <= x+1; nx++ {
// 									if nx == 0 && ny == 0 && nz == 0 {
// 										continue
// 									}
// 									if l.xyzIndicesInsideVolume(nx, ny, nz) {
// 										newColor := l.getLightValue(nx, ny, nz)
// 										if newColor.R != lightVolumeSpecialCopy {
// 											l.setLightValue(x, y, z, newColor)
// 											set = true
// 										}
// 									}
// 									if set {
// 										break
// 									}
// 								}
// 								if set {
// 									break
// 								}
// 							}
// 							if set {
// 								break
// 							}
// 						}
// 					}
// 				}

// 			}
// 		}
// 	}
// }

// Blurs the cells in the LightVolume.
// radius is the number of cells around each cell to blur each cell's value.
// iterations is the number of times to run the blur.
func (l *LightVolume) LightCellBlur(cellRadius int, iterations int) {

	if cellRadius < 1 {
		cellRadius = 1
	}

	if iterations < 1 {
		iterations = 1
	}

	csx := l.cellSizeX / 2
	csy := l.cellSizeY / 2
	csz := l.cellSizeZ / 2

	for range iterations {

		ogLightValues := append(make([]Color, 0, len(l.lightValues)), l.lightValues...)

		getLightValueSpread := func(indexX, indexY, indexZ int) Color {
			if indexX < 0 || indexY < 0 || indexZ < 0 || indexX >= l.cellCountX || indexY >= l.cellCountY || indexZ >= l.cellCountZ {
				return NewColor(0, 0, 0, 0)
			}
			return ogLightValues[indexX+(indexY*l.cellCountX)+(indexZ*l.cellCountX*l.cellCountY)]
		}

		for wz := l.dimensions.Min.Z + csz; wz <= l.dimensions.Max.Z; wz += l.cellSizeZ {
			for wy := l.dimensions.Min.Y + csy; wy <= l.dimensions.Max.Y; wy += l.cellSizeY {
				for wx := l.dimensions.Min.X + csx; wx <= l.dimensions.Max.X; wx += l.cellSizeX {

					x, y, z := l.convertToXYZIndices(wx, wy, wz)

					if l.xyzIndicesInsideVolume(x, y, z) {

						count := 1
						current := getLightValueSpread(x, y, z)

						for i := 1; i <= cellRadius; i++ {

							for rz := -cellRadius; rz <= cellRadius; rz++ {
								for ry := -cellRadius; ry <= cellRadius; ry++ {
									for rx := -cellRadius; rx <= cellRadius; rx++ {

										if rx == 0 && ry == 0 && rz == 0 {
											continue
										}
										if l.xyzIndicesInsideVolume(x+rx, y+ry, z+rz) {
											c := getLightValueSpread(x+rx, y+ry, z+rz)
											if c.A > 0 {
												count++
												current = current.Add(c)
											}
										}
									}
								}
							}

						}

						l.setLightValue(x, y, z, current.MultiplyScalarRGB(1/float32(count)))

					}

				}
			}
		}

	}
}

func (l *LightVolume) LightValuesSave() []Color {
	return append(make([]Color, 0, len(l.lightValues)), l.lightValues...)
}

func (l *LightVolume) LightValuesLoad(colors []Color, forEach func(current, loading Color) Color) {
	for i := 0; i < len(colors); i++ {
		l.lightValues[i] = forEach(l.lightValues[i], colors[i])
	}
}

func (l *LightVolume) getLightValue(indexX, indexY, indexZ int) Color {
	if indexX < 0 || indexY < 0 || indexZ < 0 || indexX >= l.cellCountX || indexY >= l.cellCountY || indexZ >= l.cellCountZ {
		return NewColor(0, 0, 0, 0)
	}
	return l.lightValues[indexX+(indexY*l.cellCountX)+(indexZ*l.cellCountX*l.cellCountY)]
}

func (l *LightVolume) setLightValue(indexX, indexY, indexZ int, color Color) {
	if indexX < 0 || indexY < 0 || indexZ < 0 || indexX >= l.cellCountX || indexY >= l.cellCountY || indexZ >= l.cellCountZ {
		return
	}
	l.lightValues[indexX+(indexY*l.cellCountX)+(indexZ*l.cellCountX*l.cellCountY)] = color
}

func (l *LightVolume) DebugDraw(screen *ebiten.Image, camera *Camera) {

	l.LightCellReadWriteAll(func(position Vector3, in Color) Color {

		// pos := camera.ClipToScreen(position)
		pos := camera.WorldToScreenPixels(position)
		if pos.Z < 0 {
			return in
		}
		dist := 1 - (camera.WorldPosition().DistanceTo(position) / camera.far)
		if dist < 0 {
			return in
		}

		v := uint16(65535 * dist)
		color := color.NRGBA64{0, v, 0, v}

		// color = in.ToNRGBA64()

		vector.StrokeLine(screen, pos.X-8, pos.Y, pos.X+8, pos.Y, 2, color, false)
		vector.StrokeLine(screen, pos.X, pos.Y-8, pos.X, pos.Y+8, 2, color, false)
		return in

	})

}

// Returns the size of light volume cells on the X axis.
func (l *LightVolume) CellSizeX() float32 {
	return l.cellSizeX
}

// Returns the size of light volume cells on the Y axis.
func (l *LightVolume) CellSizeY() float32 {
	return l.cellSizeY
}

// Returns the size of light volume cells on the Z axis.
func (l *LightVolume) CellSizeZ() float32 {
	return l.cellSizeZ
}

// Returns the size of light volume cells on the axis determined by the dimension argument (0 = X, 1 = Y, otherwise = Z).
func (l *LightVolume) CellSize(dimension int) float32 {
	if dimension == 0 {
		return l.cellSizeX
	} else if dimension == 1 {
		return l.cellSizeY
	}
	return l.cellSizeZ
}

// Returns the number of light volume cells on the X axis.
func (l *LightVolume) CellCountX() int {
	return l.cellCountX
}

// Returns the number of light volume cells on the Y axis.
func (l *LightVolume) CellCountY() int {
	return l.cellCountY
}

// Returns the number of light volume cells on the Z axis.
func (l *LightVolume) CellCountZ() int {
	return l.cellCountZ
}

// Returns the number of volume cells on the axis determined by the dimension argument (0 = X, 1 = Y, otherwise = Z).
func (l *LightVolume) CellCount(dimension int) int {
	if dimension == 0 {
		return l.cellCountX
	} else if dimension == 1 {
		return l.cellCountY
	}
	return l.cellCountZ
}

// On returns if the lightmap is on or not.
func (l *LightVolume) On() bool {
	return l.on
}

// SetOn sets the lightmap to be on or off.
func (l *LightVolume) SetOn(on bool) {
	l.on = on
}

// Returns the global color associated with the LightVolume (which multiplies to color the cells of the LightVolume).
func (l *LightVolume) Color() Color {
	return l.color
}

// Sets the global color associated with the LightVolume (which multiplies to color the cells of the LightVolume).
func (l *LightVolume) SetColor(color Color) {
	l.color = color
}

// Returns the energy associated with the LightVolume (which multiplies to color the cells of the LightVolume).
func (l *LightVolume) Energy() float32 {
	return l.energy
}

// Sets the energy associated with the LightVolume (which multiplies to color the cells of the LightVolume).
func (l *LightVolume) SetEnergy(energy float32) {
	l.energy = energy
}

// Type returns the type of INode this is (NodeTypeLightVolume).
func (l *LightVolume) Type() NodeType {
	return NodeTypeLightVolume
}

/////

// type polygonLightCell struct {
// 	Color    *Color
// 	Distance float32
// }

// type PolygonLight struct {
// 	*Node
// 	Distance float32
// 	Energy   float32
// 	Model    *Model
// 	On       bool
// 	gridSize float32

// 	lightCells [][][]*polygonLightCell
// }

// func NewPolygonLight(model *Model, energy float32, gridSize float32) *PolygonLight {

// 	poly := &PolygonLight{
// 		Model:      model,
// 		Distance:   10,
// 		Energy:     energy,
// 		On:         true,
// 		lightCells: [][][]*polygonLightCell{},
// 	}

// 	poly.Node = model.Node

// 	poly.ResizeGrid(gridSize)

// 	return poly
// }

// func (poly *PolygonLight) ResizeGrid(gridSize float32) {

// 	poly.lightCells = [][][]*polygonLightCell{}

// 	dim := poly.Model.Mesh.Dimensions

// 	transform := poly.Model.Transform()

// 	zd := int(Ceil(dim.Depth() / gridSize))
// 	yd := int(Ceil(dim.Height() / gridSize))
// 	xd := int(Ceil(dim.Width() / gridSize))

// 	poly.lightCells = make([][][]*polygonLightCell, zd)

// 	for z := 0; z < zd; z++ {

// 		poly.lightCells[z] = make([][]*polygonLightCell, yd)

// 		for y := 0; y < yd; y++ {

// 			poly.lightCells[z][y] = make([]*polygonLightCell, xd)

// 			for x := 0; x < xd; x++ {

// 				gridPos := Vector{
// 					(float32(x) - float32(xd/2)) * gridSize,
// 					(float32(y) - float32(yd/2)) * gridSize,
// 					(float32(z) - float32(zd/2)) * gridSize,
// 				}
// 				gridPos = transform.MultVec(gridPos)

// 				// closest := poly.Model.Mesh.Triangles[0]
// 				closestDistance := Maxfloat32

// 				for _, tri := range poly.Model.Mesh.Triangles {

// 					tc := transform.MultVec(tri.Center)

// 					dist := fastVectorDistanceSquared(tc, gridPos)
// 					if dist < closestDistance {
// 						// closest = tri
// 						closestDistance = dist
// 					}

// 				}

// 				poly.lightCells[z][y][x] = &polygonLightCell{
// 					Color:    NewColor(1, 1, 1, 1),
// 					Distance: closestDistance,
// 				}

// 				// if closestDistance <= gridSize*gridSize {
// 				// 	poly.lightPoints[z][y][x] = NewColor(1, 1, 1, 1)
// 				// } else {
// 				// 	poly.lightPoints[z][y][x] = nil
// 				// }

// 			}

// 		}

// 	}

// 	poly.gridSize = gridSize

// }

// func (poly *PolygonLight) Clone() INode {
// 	clone := NewPolygonLight(poly.Model.Clone().(*Model),
// 		poly.Energy,
// 		poly.gridSize,
// 	)
// 	clone.Node = clone.Model.Node
// 	clone.Distance = poly.Distance
// 	return clone
// }

// func (poly *PolygonLight) beginRender() {}

// func (poly *PolygonLight) beginModel(model *Model) {}

// func (poly *PolygonLight) Light(triIndex uint16, model *Model) [9]float32 {

// 	light := [9]float32{}

// 	var vertPos, vertNormal Vector

// 	color := poly.Model.Color

// 	// dist := tc.Sub(poly.centerPoints[0]).Magnitude()

// 	distanceSquared := poly.Distance * poly.Distance

// 	lightPos := Vector{0, 0, 0}

// 	for i := 0; i < 3; i++ {

// 		if model.skinned {
// 			vertPos = model.Mesh.vertexSkinnedPositions[triIndex*3+i]
// 			vertNormal = model.Mesh.vertexSkinnedNormals[triIndex*3+i]
// 		} else {
// 			vertPos = model.Mesh.VertexPositions[triIndex*3+i]
// 			vertNormal = model.Mesh.VertexNormals[triIndex*3+i]
// 		}

// 		lightVec := vector.In(fastVectorSub(lightPos, vertPos)).Unit()
// 		diffuse := dot(vertNormal, Vector(lightVec))

// 		if diffuse < 0 {
// 			diffuse = 0
// 		}

// 		var diffuseFactor float32
// 		distance := fastVectorDistanceSquared(lightPos, vertPos)

// 		diffuseFactor = diffuse * Max(Min(1.0-(Pow((distance/distanceSquared), 4)), 1), 0)

// 		light[i] = color.R * float32(diffuseFactor) * poly.Energy
// 		light[i+1] = color.G * float32(diffuseFactor) * poly.Energy
// 		light[i+2] = color.B * float32(diffuseFactor) * poly.Energy

// 	}

// 	return light
// }

// func (poly *PolygonLight) IsOn() bool {
// 	return poly.On && poly.Energy > 0
// }

// func (poly *PolygonLight) SetOn(on bool) {
// 	poly.On = on
// }

// // Type returns the NodeType for this object.
// func (poly *PolygonLight) Type() NodeType {
// 	return NodeTypePolygonLight
// }

// func (poly *PolygonLight) ToString() string {

// 	str := ""

// 	for z := range poly.lightCells {
// 		for y := range poly.lightCells[z] {
// 			for _, cell := range poly.lightCells[z][y] {
// 				if cell.Distance < 2 {
// 					str += "x"
// 				} else {
// 					str += " "
// 				}
// 			}
// 		}
// 		str += "\n"
// 	}

// 	return str

// }
