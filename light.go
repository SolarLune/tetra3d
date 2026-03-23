package tetra3d

import (
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

	VertexLight(vertexIndex int, triangle *Triangle, model *Model, targetColors VertexColorChannel)
	Light(triangle *Triangle, model *Model, targetColors VertexColorChannel) // Light lights the triangles in the MeshPart, storing the result in the targetColors
	// color buffer. If onlyVisible is true, only the visible vertices will be lit; if it's false, they will all be lit.

	Color() Color4
	SetColor(c Color4)
	Energy() float32
	SetEnergy(energy float32)
}

type LightBase struct {
	*Node
	color Color4 // Color is the color of the PointLight.
	// energy is the overall energy of the Light. Internally, technically there's no difference between a brighter color and a
	// higher energy, but this is here for convenience / adherance to GLTF / 3D modelers.
	energy float32
	on     bool // If the light is on and contributing to the scene.
}

func newLightBase(name string, r, g, b, energy float32) *LightBase {
	lightBase := &LightBase{
		Node:   NewNode(name),
		color:  NewColor4(r, g, b, 1),
		energy: energy,
		on:     true,
	}
	return lightBase
}

func (l *LightBase) clone() *LightBase {
	newLB := newLightBase(l.name, l.color.R, l.color.G, l.color.B, l.energy)
	newLB.Node = l.Node.clone(l.owner).(*Node)
	return newLB
}

func (l *LightBase) Energy() float32 {
	return l.energy
}

func (l *LightBase) SetEnergy(energy float32) {
	if l.energy != energy {
		l.energy = energy
	}
}

func (l *LightBase) Color() Color4 {
	return l.color
}

func (l *LightBase) SetColor(color Color4) {
	if l.color != color {
		l.color = color
	}
}

//---------------//

// AmbientLight represents an ambient light that colors the entire Scene.
type AmbientLight struct {
	*LightBase

	result [3]float32
}

// NewAmbientLight returns a new AmbientLight.
func NewAmbientLight(name string, r, g, b, energy float32) *AmbientLight {
	amb := &AmbientLight{
		LightBase: newLightBase(name, r, g, b, energy),
	}
	amb.owner = amb
	return amb
}

func (amb *AmbientLight) Clone() INode {

	clone := NewAmbientLight(amb.name, amb.color.R, amb.color.G, amb.color.B, amb.energy)
	clone.on = amb.on

	clone.LightBase = amb.LightBase.clone()

	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}

	return clone

}

func (amb *AmbientLight) beginRender() {
	amb.result = [3]float32{amb.color.R * amb.energy, amb.color.G * amb.energy, amb.color.B * amb.energy}
}

func (amb *AmbientLight) beginModel(model *Model) {}

func (amb *AmbientLight) VertexLight(vertexIndex int, tri *Triangle, model *Model, targetColors VertexColorChannel) {
	targetColors[vertexIndex].R += amb.result[0]
	targetColors[vertexIndex].G += amb.result[1]
	targetColors[vertexIndex].B += amb.result[2]
}

// Light returns the light level for the ambient light. It doesn't use the provided Triangle; it takes it as an argument to simply adhere to the Light interface.
func (amb *AmbientLight) Light(tri *Triangle, model *Model, targetColors VertexColorChannel) {
	tri.ForEachVertexIndex(func(vertIndex int) {
		amb.VertexLight(vertIndex, tri, model, targetColors)
	})
}

// Type returns the NodeType for this object.
func (amb *AmbientLight) Type() NodeType {
	return NodeTypeAmbientLight
}

//---------------//

// PointLight represents a point light (naturally).
type PointLight struct {
	*LightBase
	// Range represents the distance after which the light fully attenuates. If this is 0 (the default),
	// it falls off using something akin to the inverse square law.
	lightRange        float32
	lightRangeSquared float32
	workingPosition   Vector3
}

// NewPointLight creates a new Point light.
func NewPointLight(name string, r, g, b, energy float32) *PointLight {
	point := &PointLight{
		LightBase: newLightBase(name, r, g, b, energy),
	}
	point.owner = point
	return point
}

// Clone returns a new clone of the given point light.
func (p *PointLight) Clone() INode {

	clone := NewPointLight(p.name, p.color.R, p.color.G, p.color.B, p.energy)
	clone.on = p.on
	clone.lightRange = p.lightRange
	clone.lightRangeSquared = p.lightRangeSquared

	clone.LightBase = p.LightBase.clone()

	if runCallbacks && clone.Callbacks().OnClone != nil {
		clone.Callbacks().OnClone(clone)
	}

	return clone

}

func (p *PointLight) beginRender() {}

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

func (p *PointLight) VertexLight(index int, tri *Triangle, model *Model, targetColors VertexColorChannel) {
	// TODO: Make lighting faster by returning early if the triangle is too far from the point light position

	var vertPos, vertNormal Vector3

	if model.skinned {
		vertPos = model.mesh.vertexSkinnedPositions[index]
		vertNormal = model.mesh.vertexSkinnedNormals[index]
	} else {
		vertPos = model.mesh.VertexPositions[index]
		vertNormal = model.mesh.VertexNormals[index]
	}

	distance := p.workingPosition.DistanceSquaredTo(vertPos)

	if p.lightRange > 0 {

		if distance > p.lightRangeSquared {
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
	if mat := tri.MeshPart.Material; mat != nil && mat.LightingMode == LightingModeFixedNormals {
		vertNormal = lightVec
	}

	diffuse := vertNormal.Dot(lightVec)

	if mat := tri.MeshPart.Material; mat != nil && mat.LightingMode == LightingModeDoubleSided {
		diffuse = math32.Abs(diffuse)
	}

	if diffuse > 0 {

		diffuseFactor := diffuse * (1.0 / (1.0 + (0.1 * distance))) * 2

		if p.lightRangeSquared > 0 {
			distClamp := math32.Clamp((p.lightRangeSquared-distance)/distance, 0, 1)
			diffuseFactor *= distClamp
		}

		targetColors[index].R += p.color.R * float32(diffuseFactor) * p.energy
		targetColors[index].G += p.color.G * float32(diffuseFactor) * p.energy
		targetColors[index].B += p.color.B * float32(diffuseFactor) * p.energy

	}

}

// Light returns the R, G, and B values for the PointLight for all vertices of a given Triangle.
func (p *PointLight) Light(tri *Triangle, model *Model, targetColors VertexColorChannel) {

	// We calculate both the eye vector as well as the light vector so that if the camera passes behind the
	// lit face and backface culling is off, the triangle can still be lit or unlit from the other side. Otherwise,
	// if the triangle were lit by a light, it would appear lit regardless of the positioning of the camera.

	tri.ForEachVertexIndex(func(index int) {

		p.VertexLight(index, tri, model, targetColors)

	})

}

func (p *PointLight) Range() float32 {
	return p.lightRange
}

func (p *PointLight) SetRange(lightRange float32) {
	p.lightRange = lightRange
	p.lightRangeSquared = lightRange * lightRange
}

// Type returns the NodeType for this object.
func (p *PointLight) Type() NodeType {
	return NodeTypePointLight
}

//---------------//

// DirectionalLight represents a directional light of infinite distance.
type DirectionalLight struct {
	*LightBase
	workingForward       Vector3 // Internal forward vector so we don't have to calculate it for every triangle for every model using this light.
	workingModelRotation Matrix4 // Similarly, this is an internal rotational transform (without the transformation row) for the Model being lit.
}

// NewDirectionalLight creates a new Directional Light with the specified RGB color and energy (assuming 1.0 energy is standard / "100%" lighting).
func NewDirectionalLight(name string, r, g, b, energy float32) *DirectionalLight {
	sun := &DirectionalLight{
		LightBase: newLightBase(name, r, g, b, energy),
	}
	sun.owner = sun
	return sun
}

// Clone returns a new DirectionalLight clone from the given DirectionalLight.
func (sun *DirectionalLight) Clone() INode {

	clone := NewDirectionalLight(sun.name, sun.color.R, sun.color.G, sun.color.B, sun.energy)

	clone.on = sun.on

	clone.LightBase = sun.LightBase.clone()

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

func (sun *DirectionalLight) VertexLight(index int, tri *Triangle, model *Model, targetColors VertexColorChannel) {

	var normal Vector3
	if model.skinned {
		// If it's skinned, we don't have to calculate the normal, as that's been pre-calc'd for us
		normal = model.mesh.vertexSkinnedNormals[index]
	} else {
		normal = sun.workingModelRotation.MultVec(model.mesh.VertexNormals[index])
	}

	if mat := tri.MeshPart.Material; mat != nil && mat.LightingMode == LightingModeFixedNormals {
		normal = sun.workingForward
	}

	diffuseFactor := normal.Dot(sun.workingForward)

	if mat := tri.MeshPart.Material; mat != nil && mat.LightingMode == LightingModeDoubleSided {
		diffuseFactor = math32.Abs(diffuseFactor)
	}

	if diffuseFactor <= 0 {
		return
	}

	targetColors[index].R += sun.color.R * float32(diffuseFactor) * sun.energy
	targetColors[index].G += sun.color.G * float32(diffuseFactor) * sun.energy
	targetColors[index].B += sun.color.B * float32(diffuseFactor) * sun.energy

}

// Light returns the R, G, and B values for the DirectionalLight for each vertex of the provided Triangle.
func (sun *DirectionalLight) Light(tri *Triangle, model *Model, targetColors VertexColorChannel) {

	tri.ForEachVertexIndex(func(index int) {
		sun.VertexLight(index, tri, model, targetColors)
	})

}

// Type returns the NodeType for this object.
func (sun *DirectionalLight) Type() NodeType {
	return NodeTypeDirectionalLight
}

// CubeLight represents an AABB volume that lights triangles.
type CubeLight struct {
	*LightBase
	Dimensions Dimensions // The overall dimensions of the CubeLight.
	// A value between 0 and 1 indicating how much opposite faces are still lit within the volume (i.e. at LightBleed = 0.0,
	// faces away from the light are dark; at 1.0, faces away from the light are fully illuminated)
	bleed             float32
	lightingAngle     Vector3 // The direction in which light is shining. Defaults to local Y down (0, -1, 0).
	workingDimensions Dimensions
	workingPosition   Vector3
	workingAngle      Vector3
}

// NewCubeLight creates a new CubeLight with the given dimensions.
func NewCubeLight(name string, dimensions Dimensions) *CubeLight {
	cube := &CubeLight{
		LightBase:     newLightBase(name, 1, 1, 1, 1),
		Dimensions:    dimensions,
		lightingAngle: Vector3{0, -1, 0},
	}
	cube.owner = cube
	return cube
}

// NewCubeLightFromModel creates a new CubeLight from the Model's dimensions and world transform.
// Note that this does not add it to the Model's hierarchy in any way - the newly created CubeLight
// is still its own Node.
func NewCubeLightFromModel(name string, model *Model) *CubeLight {
	cube := NewCubeLight(name, model.mesh.Dimensions)
	cube.SetWorldTransform(model.Transform())
	cube.lightingAngle = model.WorldRotation().Up().Invert()
	return cube
}

// Clone clones the CubeLight, returning a deep copy.
func (cube *CubeLight) Clone() INode {
	newCube := NewCubeLight(cube.name, cube.Dimensions)
	newCube.energy = cube.energy
	newCube.color = cube.color
	newCube.on = cube.on
	newCube.bleed = cube.bleed
	newCube.lightingAngle = cube.lightingAngle
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
	cube.workingAngle = cube.WorldRotation().MultVec(cube.lightingAngle.Invert())
}

func (cube *CubeLight) beginModel(model *Model) {

	p, s, r := model.Transform().Inverted().Decompose()
	_, _, _ = p, s, r

	// Rather than transforming all vertices of all triangles of a mesh, we can just transform the
	// point light's position by the inversion of the model's transform to get the same effect and save processing time.
	// The same technique is used for Sphere - Triangle collision in bounds.go.

	lightStartPos := cube.WorldPosition().Add(cube.lightingAngle.Invert().Scale(cube.Dimensions.Height() / 2))

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

		cube.workingDimensions = cube.workingDimensions.Canon()

	}

}

func (cube *CubeLight) VertexLight(index int, tri *Triangle, model *Model, targetColors VertexColorChannel) {

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
		vertPos = model.mesh.vertexSkinnedPositions[index]
		vertNormal = model.mesh.vertexSkinnedNormals[index]
	} else {
		vertPos = model.mesh.VertexPositions[index]
		vertNormal = model.mesh.VertexNormals[index]
	}

	var diffuse, diffuseFactor float32

	if mat := tri.MeshPart.Material; mat != nil && mat.LightingMode == LightingModeFixedNormals {
		vertNormal = cube.workingAngle
	}

	diffuse = vertNormal.Dot(cube.workingAngle)

	if mat := tri.MeshPart.Material; mat != nil && mat.LightingMode == LightingModeDoubleSided {
		diffuse = math32.Abs(diffuse)
	}

	if cube.bleed > 0 {

		if diffuse < 0 {
			diffuse = 0
		}

		// diffuse += cube.Bleed
		diffuse = cube.bleed + ((1 - cube.bleed) * diffuse)

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

}

// Light returns the R, G, and B values for the PointLight for all vertices of a given Triangle.
func (cube *CubeLight) Light(tri *Triangle, model *Model, targetColors VertexColorChannel) {

	tri.ForEachVertexIndex(func(index int) {
		cube.VertexLight(index, tri, model, targetColors)
	})

}

// Type returns the type of INode this is (NodeTypeCubeLight).
func (cube *CubeLight) Type() NodeType {
	return NodeTypeCubeLight
}

func (cube *CubeLight) LightingAngle() Vector3 {
	return cube.lightingAngle
}

func (cube *CubeLight) SetLightingAngle(angle Vector3) {
	cube.lightingAngle = angle
}

func (cube *CubeLight) Bleed() float32 {
	return cube.bleed
}

func (cube *CubeLight) SetBleed(bleed float32) {
	cube.bleed = bleed
}

// LightVolume represents a grid of colors representing light to color models that enter the space with.
type LightVolume struct {
	*LightBase

	useLightsRenderSet Set[ILight]

	cellSizeX float32
	cellSizeY float32
	cellSizeZ float32

	cellCountX int
	cellCountY int
	cellCountZ int

	dimensions Dimensions

	lightValues []*LightVolumeCell
}

// Creates a new light volume node of the given name, transformed dimensions, and light grid cellular size.
func NewLightVolume(name string, volumeDim Dimensions, cellSizeX, cellSizeY, cellSizeZ float32) *LightVolume {
	lv := &LightVolume{
		LightBase:          newLightBase(name, 1, 1, 1, 1),
		useLightsRenderSet: newSet[ILight](),
	}

	lv.owner = lv

	lv.LightVolumeResize(volumeDim, cellSizeX, cellSizeY, cellSizeZ)

	return lv
}

// Creates a new light volume node from the given Model's name and dimensions, with the given cellular size.
// Note that the model will still exist, so you'll need to dispose of it as you'd like.
func NewLightVolumeFromModel(model *Model, cellSizeX, cellSizeY, cellSizeZ float32) *LightVolume {
	vol := NewLightVolume(model.Name(), model.Dimensions(), cellSizeX, cellSizeY, cellSizeZ)
	vol.Node.SetWorldTransform(model.Transform())
	vol.scene = model.scene
	return vol
}

// Clones the LightVolume and returns another.
func (l *LightVolume) Clone() INode {
	newLightmap := NewLightVolume(l.Name(), l.dimensions, l.cellSizeX, l.cellSizeY, l.cellSizeZ)
	newLightmap.LightBase = l.LightBase.clone()
	newLightmap.dimensions = l.dimensions
	newLightmap.lightValues = append(newLightmap.lightValues, l.lightValues...)
	newLightmap.energy = l.energy
	newLightmap.color = l.color
	newLightmap.on = l.on
	return newLightmap
}

func (l *LightVolume) Dimensions() Dimensions {
	return l.dimensions
}

// Resizes the LightVolume to be of the given transformed dimensions and cellular size.
// Calling this after initial creation will move the Cell properties over to the newly resized LightVolume,
// meaning you can create a low resolution LightVolume, populate its cells, then resize it and blur or spread the light
// for a finer looking result with the original data.
// If the LightVolume already has the given dimensions and cellular size, then nothing is done.
func (l *LightVolume) LightVolumeResize(dimensions Dimensions, cellsizeX, cellsizeY, cellsizeZ float32) {

	// Don't resize if the sizes are 0 (light volume is being created)
	if cellsizeX == 0 || cellsizeY == 0 || cellsizeZ == 0 {
		return
	}

	oldLightValues := make([]*LightVolumeCell, 0, len(l.lightValues))

	for _, lv := range l.lightValues {
		oldLightValues = append(oldLightValues, lv.clone())
	}

	oldDimensions := l.dimensions
	oldCellSizeX := l.cellSizeX
	oldCellSizeY := l.cellSizeY
	oldCellSizeZ := l.cellSizeZ

	oldCellCountX := l.cellCountX
	oldCellCountY := l.cellCountY
	oldCellCountZ := l.cellCountZ

	l.lightValues = l.lightValues[:0]

	l.dimensions = dimensions
	l.cellSizeX = cellsizeX
	l.cellSizeY = cellsizeY
	l.cellSizeZ = cellsizeZ

	cellCountX := 0
	cellCountY := 0
	cellCountZ := 0

	for z := dimensions.Min.Z; z < dimensions.Max.Z; z += cellsizeZ {
		cellCountY = 0

		for y := dimensions.Min.Y; y < dimensions.Max.Y; y += cellsizeY {
			cellCountX = 0

			for x := dimensions.Min.X; x < dimensions.Max.X; x += cellsizeX {
				l.lightValues = append(l.lightValues, &LightVolumeCell{
					lightVolume: l,
					color:       NewColor4(0, 0, 0, 1),
					indexX:      cellCountX,
					indexY:      cellCountY,
					indexZ:      cellCountZ,
				})
				cellCountX++
			}

			cellCountY++

		}

		cellCountZ++

	}

	l.cellCountX = cellCountX
	l.cellCountY = cellCountY
	l.cellCountZ = cellCountZ

	if oldLightValues != nil {

		oldConvertToXYZIndices := func(wx, wy, wz float32) (x, y, z int) {

			x = int((wx - oldDimensions.Min.X) / oldCellSizeX)
			y = int((wy - oldDimensions.Min.Y) / oldCellSizeY)
			z = int((wz - oldDimensions.Min.Z) / oldCellSizeZ)

			return
		}

		getOgLightCell := func(indexX, indexY, indexZ int) *LightVolumeCell {
			if indexX < 0 || indexY < 0 || indexZ < 0 || indexX >= oldCellCountX || indexY >= oldCellCountY || indexZ >= oldCellCountZ {
				return nil
			}
			return oldLightValues[indexX+(indexY*oldCellCountX)+(indexZ*oldCellCountX*oldCellCountY)]
		}

		for z := dimensions.Min.Z; z < dimensions.Max.Z; z += cellsizeZ {

			for y := dimensions.Min.Y; y < dimensions.Max.Y; y += cellsizeY {

				for x := dimensions.Min.X; x < dimensions.Max.X; x += cellsizeX {

					if cell := l.getLightCell(l.convertToXYZIndices(x, y, z)); cell != nil {
						if oldCell := getOgLightCell(oldConvertToXYZIndices(x, y, z)); oldCell != nil {
							cell.color = oldCell.color
							cell.useLights = append(make([]ILight, 0, len(oldCell.useLights)), oldCell.useLights...)
							cell.blocked = oldCell.blocked
							cell.diffuseness = oldCell.diffuseness
							cell.lightDirection = oldCell.lightDirection
						}
					}

				}

			}

		}

	}

}

func (l *LightVolume) beginRender() {
	l.useLightsRenderSet.ForEach(func(light ILight) bool {
		light.beginRender()
		return true
	})
}

func (l *LightVolume) beginModel(model *Model) {

	l.useLightsRenderSet.ForEach(func(light ILight) bool {
		light.beginModel(model)
		return true
	})

}

func (l *LightVolume) VertexLight(vertIndex int, tri *Triangle, model *Model, targetColors VertexColorChannel) {

	objectShadingMode := 0

	cellColor := NewColor4(0, 0, 0, 1)

	transform := model.Transform()

	cellSize := NewVector3(l.cellSizeX, l.cellSizeY, l.cellSizeZ)

	if tri.MeshPart.Material != nil {
		objectShadingMode = tri.MeshPart.Material.LightVolumeShadingMode

		if tri.MeshPart.Material.LightVolumeShadingMode == LightVolumeShadingModePerObject {

			x, y, z := l.convertToXYZIndicesVec(model.WorldPosition())

			cell := l.getLightCell(x, y, z)
			if cell != nil {
				cellColor.R = cell.color.R
				cellColor.G = cell.color.G
				cellColor.B = cell.color.B
			}

		}

	}

	if objectShadingMode != LightVolumeShadingModePerObject {

		var vertPos Vector3
		var normal Vector3

		if model.skinned {
			vertPos = model.mesh.vertexSkinnedPositions[vertIndex]
			normal = model.mesh.vertexSkinnedNormals[vertIndex]

			if objectShadingMode == LightVolumeShadingModePerVertexWithNormal {
				vertPos = vertPos.Add(model.mesh.vertexSkinnedNormals[vertIndex])
			}

		} else {
			normal = model.mesh.VertexNormals[vertIndex]

			if objectShadingMode == LightVolumeShadingModePerVertexWithNormal {
				vertPos = transform.MultVec(model.mesh.VertexPositions[vertIndex].Add(normal.Mult(cellSize)))
			} else {
				vertPos = transform.MultVec(model.mesh.VertexPositions[vertIndex])
			}

		}

		if !l.dimensions.Contains(vertPos) {
			return
		}

		cellColor.R = 0
		cellColor.G = 0
		cellColor.B = 0

		x, y, z := l.convertToXYZIndicesVec(vertPos)

		cell := l.getLightCell(x, y, z)
		if cell != nil {

			if cellColor.Value() < cell.color.Value() {
				cellColor = cell.color
			}

			for _, light := range cell.useLights {
				light.VertexLight(vertIndex, tri, model, targetColors)
			}
		}

	}

	targetColors[vertIndex].R += l.color.R * cellColor.R * l.energy
	targetColors[vertIndex].G += l.color.G * cellColor.G * l.energy
	targetColors[vertIndex].B += l.color.B * cellColor.B * l.energy

}

// Lights the mesh part of the model being rendered, placing the results into the targeted VertexColorChannel, only doing so
// for the visible parts of the mesh if onlyVisible is set to true.
func (l *LightVolume) Light(tri *Triangle, model *Model, targetColors VertexColorChannel) {

	// TODO: If distance from the mesh mesh part is too big, then we can just forget about it?

	tri.ForEachVertexIndex(func(vertIndex int) {
		l.VertexLight(vertIndex, tri, model, targetColors)
	})

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

func (l *LightVolume) convertToWorldPosition(x, y, z int) Vector3 {

	return NewVector3(
		(float32(x)*l.cellSizeX)+l.dimensions.Min.X,
		(float32(y)*l.cellSizeY)+l.dimensions.Min.Y,
		(float32(z)*l.cellSizeZ)+l.dimensions.Min.Z,
	)
}

func (l *LightVolume) xyzIndicesInsideVolume(x, y, z int) bool {
	return x >= 0 && y >= 0 && z >= 0 && x < l.cellCountX && y < l.cellCountY && z < l.cellCountZ
}

func (l *LightVolume) LightVolumeCellAt(position Vector3) *LightVolumeCell {

	x, y, z := l.convertToXYZIndicesVec(position)
	return l.getLightCell(x, y, z)

}

func (l *LightVolume) LightVolumeForEachCell(forEach func(position Vector3, cell *LightVolumeCell)) {

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

				if cell := l.getLightCell(x, y, z); cell != nil {
					forEach(NewVector3(wx, wy, wz), cell)
				}

			}
		}
	}

}

// Blurs all of the cells in the LightVolume.
// radius is the number of cells around each cell to blur each cell's value.
// iterations is the number of times to run the blur.
// blurStrength is the strength of the blur, ranging from 0 to 1.
// Each application of the blur happens at the end of the iterations, not all at once.
func (l *LightVolume) LightVolumeBlur(cellRadius int, iterations int, blurStrength float32) {

	if iterations < 1 {
		iterations = 1
	}

	csx := l.cellSizeX / 2
	csy := l.cellSizeY / 2
	csz := l.cellSizeZ / 2

	ogLightValues := make([]*LightVolumeCell, 0, len(l.lightValues))

	for range iterations {

		ogLightValues = ogLightValues[:0]

		for _, l := range l.lightValues {
			newL := *l
			ogLightValues = append(ogLightValues, &newL)
		}

		getOgLightCell := func(indexX, indexY, indexZ int) *LightVolumeCell {
			if indexX < 0 || indexY < 0 || indexZ < 0 || indexX >= l.cellCountX || indexY >= l.cellCountY || indexZ >= l.cellCountZ {
				return nil
			}
			return ogLightValues[indexX+(indexY*l.cellCountX)+(indexZ*l.cellCountX*l.cellCountY)]
		}

		for wz := l.dimensions.Min.Z + csz; wz <= l.dimensions.Max.Z; wz += l.cellSizeZ {
			for wy := l.dimensions.Min.Y + csy; wy <= l.dimensions.Max.Y; wy += l.cellSizeY {
				for wx := l.dimensions.Min.X + csx; wx <= l.dimensions.Max.X; wx += l.cellSizeX {

					x, y, z := l.convertToXYZIndices(wx, wy, wz)

					count := 1
					current := l.getLightCell(x, y, z)
					baseColor := current.color

					for rz := -cellRadius; rz <= cellRadius; rz++ {
						for ry := -cellRadius; ry <= cellRadius; ry++ {
							for rx := -cellRadius; rx <= cellRadius; rx++ {
								if cell := getOgLightCell(x+rx, y+ry, z+rz); cell != nil && !cell.blocked {
									count++
									baseColor = baseColor.Add(cell.color)
								}
							}
						}
					}

					current.color = current.color.Lerp(baseColor.MultiplyScalarRGB(1.0/float32(count)), blurStrength)

				}
			}
		}
	}

}

type LightVolumePropagationDirection int

const (
	PropagationDirectionWorldDown LightVolumePropagationDirection = iota
	PropagationDirectionWorldUp
	PropagationDirectionWorldRight
	PropagationDirectionWorldLeft
	PropagationDirectionWorldForward
	PropagationDirectionWorldBack
)

// Propagates light through all of the cells in the LightVolume.
// iterations is the number of times to run the blur.
// propagationStrength is how strong the light propagates (so how the light darkens over
// each successive cell) and ranges from 0 to 1.
func (l *LightVolume) LightVolumePropagate(iterations int, propagationFalloff float32, propagationDirection LightVolumePropagationDirection) {

	if iterations < 1 {
		iterations = 1
	}

	csx := l.cellSizeX / 2
	csy := l.cellSizeY / 2
	csz := l.cellSizeZ / 2

	ogLightValues := make([]*LightVolumeCell, 0, len(l.lightValues))

	for range iterations {

		ogLightValues = ogLightValues[:0]

		for _, l := range l.lightValues {
			newL := *l
			ogLightValues = append(ogLightValues, &newL)
		}

		getOgLightCell := func(indexX, indexY, indexZ int) *LightVolumeCell {
			if indexX < 0 || indexY < 0 || indexZ < 0 || indexX >= l.cellCountX || indexY >= l.cellCountY || indexZ >= l.cellCountZ {
				return nil
			}
			return ogLightValues[indexX+(indexY*l.cellCountX)+(indexZ*l.cellCountX*l.cellCountY)]
		}

		for wz := l.dimensions.Min.Z + csz; wz <= l.dimensions.Max.Z; wz += l.cellSizeZ {
			for wy := l.dimensions.Min.Y + csy; wy <= l.dimensions.Max.Y; wy += l.cellSizeY {
				for wx := l.dimensions.Min.X + csx; wx <= l.dimensions.Max.X; wx += l.cellSizeX {

					x, y, z := l.convertToXYZIndices(wx, wy, wz)

					current := l.getLightCell(x, y, z)
					ogCurrent := getOgLightCell(x, y, z)

					switch propagationDirection {
					case PropagationDirectionWorldUp:
						y--
					case PropagationDirectionWorldDown:
						y++
					case PropagationDirectionWorldRight:
						x--
					case PropagationDirectionWorldLeft:
						x++
					case PropagationDirectionWorldBack:
						z++
					case PropagationDirectionWorldForward:
						z--
					}

					if cell := getOgLightCell(x, y, z); cell != nil && !cell.blocked {
						target := cell.color

						if target.Value() > ogCurrent.color.Value() {
							current.color = ogCurrent.color.Lerp(target, propagationFalloff)
						}
					}

				}
			}
		}
	}

}

// // Propagates light through all of the cells in the LightVolume.
// // iterations is the number of times to run the blur.
// // propagationStrength is how strong the light propagates (so how the light darkens over
// // each successive cell) and ranges from 0 to 1.
// func (l *LightVolume) LightVolumePropagate(iterations int, propagationFalloff float32, propagationDirection Vector3, propagationBleed float32) {

// 	propagationDirection = propagationDirection.Invert()

// 	if iterations < 1 {
// 		iterations = 1
// 	}

// 	csx := l.cellSizeX / 2
// 	csy := l.cellSizeY / 2
// 	csz := l.cellSizeZ / 2

// 	ogLightValues := make([]*LightVolumeCell, 0, len(l.lightValues))

// 	for range iterations {

// 		ogLightValues = ogLightValues[:0]

// 		for _, l := range l.lightValues {
// 			newL := *l
// 			ogLightValues = append(ogLightValues, &newL)
// 		}

// 		getOgLightCell := func(indexX, indexY, indexZ int) *LightVolumeCell {
// 			if indexX < 0 || indexY < 0 || indexZ < 0 || indexX >= l.cellCountX || indexY >= l.cellCountY || indexZ >= l.cellCountZ {
// 				return nil
// 			}
// 			return ogLightValues[indexX+(indexY*l.cellCountX)+(indexZ*l.cellCountX*l.cellCountY)]
// 		}

// 		for wz := l.dimensions.Min.Z + csz; wz <= l.dimensions.Max.Z; wz += l.cellSizeZ {
// 			for wy := l.dimensions.Min.Y + csy; wy <= l.dimensions.Max.Y; wy += l.cellSizeY {
// 				for wx := l.dimensions.Min.X + csx; wx <= l.dimensions.Max.X; wx += l.cellSizeX {

// 					x, y, z := l.convertToXYZIndices(wx, wy, wz)

// 					current := l.getLightCell(x, y, z)
// 					ogCurrent := getOgLightCell(x, y, z)

// 					for rz := -1; rz <= 1; rz++ {
// 						for ry := -1; ry <= 1; ry++ {
// 							for rx := -1; rx <= 1; rx++ {

// 								if rx == 0 && ry == 0 && rz == 0 {
// 									continue
// 								}

// 								if cell := getOgLightCell(x+rx, y+ry, z+rz); cell != nil && !cell.blocked {
// 									target := cell.color

// 									if !propagationDirection.IsZero() {
// 										vec := NewVector3(float32(rx), float32(ry), float32(rz)).Unit()
// 										dot := vec.Dot(propagationDirection)
// 										if dot < 0 {
// 											dot = 0
// 										}
// 										if dot > 1 {
// 											dot = 1
// 										}

// 										target = Color4{0, 0, 0, 1}.Lerp(target, dot+propagationBleed)
// 									}

// 									if target.Value() > ogCurrent.color.Value() {
// 										current.color = ogCurrent.color.Lerp(target, propagationFalloff)
// 									}
// 								}
// 							}
// 						}
// 					}

// 				}
// 			}
// 		}
// 	}

// }

// func (l *LightVolume) LightVolumeProjectTextureOntoCells(cells []*LightVolumeCell, projectionDirection LightVolumePropagationDirection, lightEnergy float32, textureWidth, textureHeight int, texturePixels []byte) {

// 	startX := math.MaxInt
// 	startY := math.MaxInt
// 	startZ := math.MaxInt

// 	endX := math.MinInt
// 	endY := math.MinInt
// 	endZ := math.MinInt

// 	if cells == nil {
// 		cells = l.lightValues
// 	}

// 	for _, cell := range cells {
// 		if cell.indexX < startX {
// 			startX = cell.indexX
// 		}

// 		if cell.indexY < startY {
// 			startY = cell.indexY
// 		}

// 		if cell.indexZ < startZ {
// 			startZ = cell.indexZ
// 		}

// 		if cell.indexX > endX {
// 			endX = cell.indexX
// 		}

// 		if cell.indexY > endY {
// 			endY = cell.indexY
// 		}

// 		if cell.indexZ > endZ {
// 			endZ = cell.indexZ
// 		}
// 	}

// 	for _, cell := range cells {

// 		switch projectionDirection {
// 		case PropagationDirectionWorldRight:
// 			tx := float32(cell.indexZ-startZ) / float32(endZ-startZ)
// 			ty := float32(cell.indexY-startY) / float32(endY-startY)

// 			ty = 1 - ty

// 			ttx := int(tx * float32(textureWidth))
// 			tty := int(ty * float32(textureHeight))

// 			if ttx < 0 {
// 				ttx = 0
// 			}

// 			if tty < 0 {
// 				tty = 0
// 			}

// 			if ttx >= textureWidth {
// 				ttx = textureWidth - 1
// 			}

// 			if tty >= textureHeight {
// 				tty = textureHeight - 1
// 			}

// 			// fmt.Println(cell.indexZ, cell.indexY, ttx, tty)
// 			// fmt.Println((ttx + tty*textureWidth) * 4)

// 			cR := texturePixels[((tty*textureWidth)+ttx)*4]
// 			cG := texturePixels[((tty*textureWidth)+ttx)*4+1]
// 			cB := texturePixels[((tty*textureWidth)+ttx)*4+2]
// 			cA := texturePixels[((tty*textureWidth)+ttx)*4+3]

// 			colorA := float32(cA) / 255
// 			colorR := float32(cR) / 255 * colorA
// 			colorG := float32(cG) / 255 * colorA
// 			colorB := float32(cB) / 255 * colorA

// 			cell.SetColorRGB(colorR*lightEnergy, colorG*lightEnergy, colorB*lightEnergy)

// 		}

// 	}

// }

func (l *LightVolume) getLightCell(indexX, indexY, indexZ int) *LightVolumeCell {
	if indexX < 0 || indexY < 0 || indexZ < 0 || indexX >= l.cellCountX || indexY >= l.cellCountY || indexZ >= l.cellCountZ {
		return nil
	}
	return l.lightValues[indexX+(indexY*l.cellCountX)+(indexZ*l.cellCountX*l.cellCountY)]
}

func (l *LightVolume) LightVolumeDebugDraw(screen *ebiten.Image, camera *Camera) {

	l.LightVolumeForEachCell(func(position Vector3, cell *LightVolumeCell) {

		if cell.blocked {
			return
		}

		// pos := camera.ClipToScreen(position)
		pos := camera.WorldToScreenPixels(position)
		if pos.Z < 0 {
			return
		}
		dist := 1 - (camera.WorldPosition().DistanceTo(position) / camera.far)
		if dist < 0 {
			return
		}

		strokeColor := cell.color.MultiplyScalarRGB(0.5).AddRGBA(0.5, 0.5, 0.5, 0)

		// v := uint16(65535 * dist)
		// color := color.NRGBA64{0, v, 0, v}

		// color = in.ToNRGBA64()

		vector.StrokeLine(screen, pos.X-8, pos.Y, pos.X+8, pos.Y, 2, strokeColor.ToNRGBA64(), false)
		vector.StrokeLine(screen, pos.X, pos.Y-8, pos.X, pos.Y+8, 2, strokeColor.ToNRGBA64(), false)
		return

	})

}

func (l *LightVolume) distance(rx, ry, rz float32) float32 {
	return math32.Sqrt(rx*rx + ry*ry + rz*rz)
}

// Returns the size of light volume cells on the X axis.
func (l *LightVolume) LightVolumeCellSizeX() float32 {
	return l.cellSizeX
}

// Returns the size of light volume cells on the Y axis.
func (l *LightVolume) LightVolumeCellSizeY() float32 {
	return l.cellSizeY
}

// Returns the size of light volume cells on the Z axis.
func (l *LightVolume) LightVolumeCellSizeZ() float32 {
	return l.cellSizeZ
}

// Returns the size of light volume cells on the axis determined by the dimension argument (0 = X, 1 = Y, otherwise = Z).
func (l *LightVolume) LightVolumeCellSize(dimension int) float32 {
	if dimension == 0 {
		return l.cellSizeX
	} else if dimension == 1 {
		return l.cellSizeY
	}
	return l.cellSizeZ
}

// Returns the number of light volume cells on the X axis.
func (l *LightVolume) LightVolumeCellCountX() int {
	return l.cellCountX
}

// Returns the number of light volume cells on the Y axis.
func (l *LightVolume) LightVolumeCellCountY() int {
	return l.cellCountY
}

// Returns the number of light volume cells on the Z axis.
func (l *LightVolume) LightVolumeCellCountZ() int {
	return l.cellCountZ
}

// Returns the number of volume cells on the axis determined by the dimension argument (0 = X, 1 = Y, otherwise = Z).
func (l *LightVolume) LightVolumeCellCount(dimension int) int {
	if dimension == 0 {
		return l.cellCountX
	} else if dimension == 1 {
		return l.cellCountY
	}
	return l.cellCountZ
}

// Type returns the type of INode this is (NodeTypeLightVolume).
func (l *LightVolume) Type() NodeType {
	return NodeTypeLightVolume
}

// LightVolumeCell represents a cell in the LightVolume.
type LightVolumeCell struct {
	lightVolume    *LightVolume
	lightDirection Vector3
	diffuseness    float32
	color          Color4
	blocked        bool
	indexX         int
	indexY         int
	indexZ         int
	data           any
	useLights      []ILight
}

func (c *LightVolumeCell) clone() *LightVolumeCell {
	newCell := *c
	return &newCell
}

func (c LightVolumeCell) IndexX() int {
	return c.indexX
}

func (c LightVolumeCell) IndexY() int {
	return c.indexY
}

func (c LightVolumeCell) IndexZ() int {
	return c.indexZ
}

func (c LightVolumeCell) Color() Color4 {
	return c.color
}

func (c *LightVolumeCell) SetColor(color Color4) {
	c.color = color
}

func (c *LightVolumeCell) SetColorRGB(r, g, b float32) {
	c.color.R = r
	c.color.G = g
	c.color.B = b
}

func (c *LightVolumeCell) SetColorMonochrome(value float32) {
	c.color.R = value
	c.color.G = value
	c.color.B = value
}

func (c LightVolumeCell) Blocked() bool {
	return c.blocked
}

func (c *LightVolumeCell) SetBlocked(blocked bool) {
	c.blocked = blocked
}

func (c *LightVolumeCell) UseLights(lights ...ILight) {
	for _, l := range lights {
		c.lightVolume.useLightsRenderSet.Add(l)
	}
	c.useLights = append(c.useLights, lights...)
}

func (c *LightVolumeCell) LightsInUse() []ILight {
	return append(make([]ILight, 0, len(c.useLights)), c.useLights...)
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
