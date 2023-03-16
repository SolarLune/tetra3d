package tetra3d

// ILight represents an interface that is fulfilled by an object that emits light, returning the color a vertex should be given that Vertex and its model matrix.
type ILight interface {
	// beginRender is used to call any set-up code or to prepare math structures that are used when lighting the scene.
	// It gets called once when first rendering a set of Nodes.
	beginRender()

	// beginModel is, similarly to beginRender(), used to prepare or precompute any math necessary when lighting the scene.
	// It gets called once before lighting all visible triangles of a given Model.
	beginModel(model *Model)

	Light(sortingTriangles *renderState, model *Model, targetColors []*Color) // Light lights the triangles
	// Light(triangles []sortingTriangle, model *Model, targetColors []*Color) // Light lights the triangles
	IsOn() bool    // isOn is simply used tfo tell if a "generic" Light is on or not.
	SetOn(on bool) // SetOn sets whether the light is on or not
}

//---------------//

// AmbientLight represents an ambient light that colors the entire Scene.
type AmbientLight struct {
	*Node
	Color *Color // Color is the color of the PointLight.
	// Energy is the overall energy of the Light. Internally, technically there's no difference between a brighter color and a
	// higher energy, but this is here for convenience / adherance to GLTF / 3D modelers.
	Energy float32
	On     bool // If the light is on and contributing to the scene.

	result [3]float32
}

// NewAmbientLight returns a new AmbientLight.
func NewAmbientLight(name string, r, g, b, energy float32) *AmbientLight {
	return &AmbientLight{
		Node:   NewNode(name),
		Color:  NewColor(r, g, b, 1),
		Energy: energy,
		On:     true,
	}
}

func (amb *AmbientLight) Clone() INode {

	clone := NewAmbientLight(amb.name, amb.Color.R, amb.Color.G, amb.Color.B, amb.Energy)
	clone.On = amb.On

	clone.Node = amb.Node.Clone().(*Node)
	for _, child := range amb.children {
		child.setParent(clone)
	}

	return clone

}

func (amb *AmbientLight) beginRender() {
	amb.result = [3]float32{amb.Color.R * amb.Energy, amb.Color.G * amb.Energy, amb.Color.B * amb.Energy}
}

func (amb *AmbientLight) beginModel(model *Model) {}

// Light returns the light level for the ambient light. It doesn't use the provided Triangle; it takes it as an argument to simply adhere to the Light interface.
func (amb *AmbientLight) Light(sortingTriangles *renderState, model *Model, targetColors []*Color) {
	sortingTriangles.ForEachIndex(func(vertIndex int) {
		targetColors[vertIndex].AddRGBA(amb.result[0], amb.result[1], amb.result[2], 0)
	})
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (amb *AmbientLight) AddChildren(children ...INode) {
	amb.addChildren(amb, children...)
}

// Unparent unparents the AmbientLight from its parent, removing it from the scenegraph.
func (amb *AmbientLight) Unparent() {
	if amb.parent != nil {
		amb.parent.RemoveChildren(amb)
	}
}

func (amb *AmbientLight) IsOn() bool {
	return amb.On && amb.Energy > 0
}

func (amb *AmbientLight) SetOn(on bool) {
	amb.On = on
}

// Type returns the NodeType for this object.
func (amb *AmbientLight) Type() NodeType {
	return NodeTypeAmbientLight
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (amb *AmbientLight) Index() int {
	if amb.parent != nil {
		for i, c := range amb.parent.Children() {
			if c == amb {
				return i
			}
		}
	}
	return -1
}

//---------------//

// PointLight represents a point light (naturally).
type PointLight struct {
	*Node
	// Distance represents the distance after which the light fully attenuates. If this is 0 (the default),
	// it falls off using something akin to the inverse square law.
	Distance float64
	// Color is the color of the PointLight.
	Color *Color
	// Energy is the overall energy of the Light, with 1.0 being full brightness. Internally, technically there's no
	// difference between a brighter color and a higher energy, but this is here for convenience / adherance to the
	// GLTF spec and 3D modelers.
	Energy float32
	// If the light is on and contributing to the scene.
	On bool

	distanceSquared float64
	workingPosition Vector
}

// NewPointLight creates a new Point light.
func NewPointLight(name string, r, g, b, energy float32) *PointLight {
	return &PointLight{
		Node:   NewNode(name),
		Energy: energy,
		Color:  NewColor(r, g, b, 1),
		On:     true,
	}
}

// Clone returns a new clone of the given point light.
func (point *PointLight) Clone() INode {

	clone := NewPointLight(point.name, point.Color.R, point.Color.G, point.Color.B, point.Energy)
	clone.On = point.On
	clone.Distance = point.Distance

	clone.Node = point.Node.Clone().(*Node)
	for _, child := range point.children {
		child.setParent(point)
	}

	return clone

}

func (point *PointLight) beginRender() {
	point.distanceSquared = point.Distance * point.Distance
}

func (point *PointLight) beginModel(model *Model) {

	p, s, r := model.Transform().Inverted().Decompose()

	// Rather than transforming all vertices of all triangles of a mesh, we can just transform the
	// point light's position by the inversion of the model's transform to get the same effect and save processing time.
	// The same technique is used for Sphere - Triangle collision in bounds.go.

	if model.skinned {
		point.workingPosition = point.WorldPosition()
	} else {
		point.workingPosition = r.MultVec(point.WorldPosition()).Add(p.Mult(Vector{1 / s.X, 1 / s.Y, 1 / s.Z, s.W}))
	}

}

// Light returns the R, G, and B values for the PointLight for all vertices of a given Triangle.
func (point *PointLight) Light(sortingTriangles *renderState, model *Model, targetColors []*Color) {

	// We calculate both the eye vector as well as the light vector so that if the camera passes behind the
	// lit face and backface culling is off, the triangle can still be lit or unlit from the other side. Otherwise,
	// if the triangle were lit by a light, it would appear lit regardless of the positioning of the camera.

	sortingTriangles.ForEachIndex(func(index int) {

		// TODO: Make lighting faster by returning early if the triangle is too far from the point light position

		distance := 0.0

		var vertPos, vertNormal Vector

		if model.skinned {
			vertPos = model.Mesh.vertexSkinnedPositions[index]
			vertNormal = model.Mesh.vertexSkinnedNormals[index]
		} else {
			vertPos = model.Mesh.VertexPositions[index]
			vertNormal = model.Mesh.VertexNormals[index]
		}

		distance = point.workingPosition.DistanceSquared(vertPos)

		if point.Distance > 0 {

			if distance > point.distanceSquared {
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
			if 1/distance*float64(point.Energy) < 0.001 {
				return
			}
		}

		// If you're on the other side of the plane, just assume it's not visible.
		// if dot(model.Mesh.Triangles[triIndex].Normal, fastVectorSub(triCenter, point.cameraPosition).Unit()) < 0 {
		// 	return light
		// }

		lightVec := point.workingPosition.Sub(vertPos).Unit()
		diffuse := vertNormal.Dot(lightVec)

		if diffuse > 0 {

			diffuseFactor := 0.0

			if point.Distance == 0 {
				diffuseFactor = diffuse * (1.0 / (1.0 + (0.1 * distance))) * 2
			} else {
				diffuseFactor = diffuse * clamp(1.0-(pow((distance/point.distanceSquared), 4)), 0, 1)
			}

			targetColors[index].AddRGBA(
				point.Color.R*float32(diffuseFactor)*point.Energy,
				point.Color.G*float32(diffuseFactor)*point.Energy,
				point.Color.B*float32(diffuseFactor)*point.Energy,
				0,
			)

		}

	})

}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (point *PointLight) AddChildren(children ...INode) {
	point.addChildren(point, children...)
}

// Unparent unparents the PointLight from its parent, removing it from the scenegraph.
func (point *PointLight) Unparent() {
	if point.parent != nil {
		point.parent.RemoveChildren(point)
	}
}

func (point *PointLight) IsOn() bool {
	return point.On && point.Energy > 0
}

func (point *PointLight) SetOn(on bool) {
	point.On = on
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (point *PointLight) Index() int {
	if point.parent != nil {
		for i, c := range point.parent.Children() {
			if c == point {
				return i
			}
		}
	}
	return -1
}

// Type returns the NodeType for this object.
func (point *PointLight) Type() NodeType {
	return NodeTypePointLight
}

//---------------//

// DirectionalLight represents a directional light of infinite distance.
type DirectionalLight struct {
	*Node
	Color *Color // Color is the color of the PointLight.
	// Energy is the overall energy of the Light. Internally, technically there's no difference between a brighter color and a
	// higher energy, but this is here for convenience / adherance to GLTF / 3D modelers.
	Energy float32
	On     bool // If the light is on and contributing to the scene.

	workingForward       Vector  // Internal forward vector so we don't have to calculate it for every triangle for every model using this light.
	workingModelRotation Matrix4 // Similarly, this is an internal rotational transform (without the transformation row) for the Model being lit.
}

// NewDirectionalLight creates a new Directional Light with the specified RGB color and energy (assuming 1.0 energy is standard / "100%" lighting).
func NewDirectionalLight(name string, r, g, b, energy float32) *DirectionalLight {
	return &DirectionalLight{
		Node:   NewNode(name),
		Color:  NewColor(r, g, b, 1),
		Energy: energy,
		On:     true,
	}
}

// Clone returns a new DirectionalLight clone from the given DirectionalLight.
func (sun *DirectionalLight) Clone() INode {

	clone := NewDirectionalLight(sun.name, sun.Color.R, sun.Color.G, sun.Color.B, sun.Energy)

	clone.On = sun.On

	clone.Node = sun.Node.Clone().(*Node)
	for _, child := range sun.children {
		child.setParent(clone)
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
func (sun *DirectionalLight) Light(sortingTriangles *renderState, model *Model, targetColors []*Color) {

	sortingTriangles.ForEachIndex(func(index int) {

		var normal Vector
		if model.skinned {
			// If it's skinned, we don't have to calculate the normal, as that's been pre-calc'd for us
			normal = model.Mesh.vertexSkinnedNormals[index]
		} else {
			normal = sun.workingModelRotation.MultVec(model.Mesh.VertexNormals[index])
		}

		diffuseFactor := normal.Dot(sun.workingForward)

		if diffuseFactor <= 0 {
			return
		}

		targetColors[index].AddRGBA(
			sun.Color.R*float32(diffuseFactor)*sun.Energy,
			sun.Color.G*float32(diffuseFactor)*sun.Energy,
			sun.Color.B*float32(diffuseFactor)*sun.Energy,
			0,
		)

	})

}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (sun *DirectionalLight) AddChildren(children ...INode) {
	sun.addChildren(sun, children...)
}

// Unparent unparents the DirectionalLight from its parent, removing it from the scenegraph.
func (sun *DirectionalLight) Unparent() {
	if sun.parent != nil {
		sun.parent.RemoveChildren(sun)
	}
}

func (sun *DirectionalLight) IsOn() bool {
	return sun.On && sun.Energy > 0
}

func (sun *DirectionalLight) SetOn(on bool) {
	sun.On = on
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (sun *DirectionalLight) Index() int {
	if sun.parent != nil {
		for i, c := range sun.parent.Children() {
			if c == sun {
				return i
			}
		}
	}
	return -1
}

// Type returns the NodeType for this object.
func (sun *DirectionalLight) Type() NodeType {
	return NodeTypeDirectionalLight
}

// CubeLight represents an AABB volume that lights triangles.
type CubeLight struct {
	*Node
	Dimensions Dimensions // The overall dimensions of the CubeLight.
	Distance   float64    // The distance of the CubeLight - at 0, all triangles within the CubeLight AABB volume are lit evenly. At any other value, the CubeLight lights from the top down to the distance value specified.
	Energy     float32    // The overall energy of the CubeLight
	Color      *Color     // The color of the CubeLight
	On         bool       // If the CubeLight is on or not
	// A value between 0 and 1 indicating how much opposite faces are still lit within the volume (i.e. at LightBleed = 0.0,
	// faces away from the light are dark; at 1.0, faces away from the light are fully illuminated)
	Bleed                  float64
	LightingAngle          Vector // The direction in which light is shining. Defaults to local Y down (0, -1, 0).
	workingDimensions      Dimensions
	workingPosition        Vector
	workingAngle           Vector
	workingDistanceSquared float64
}

// NewCubeLight creates a new CubeLight with the given dimensions.
func NewCubeLight(name string, dimensions Dimensions) *CubeLight {
	cube := &CubeLight{
		Node: NewNode(name),
		// Dimensions:    Dimensions{{-w / 2, -h / 2, -d / 2}, {w / 2, h / 2, d / 2}},
		Dimensions:    dimensions,
		Distance:      0,
		Energy:        1,
		Color:         NewColor(1, 1, 1, 1),
		On:            true,
		LightingAngle: Vector{0, -1, 0, 0},
	}
	return cube
}

// NewCubeLightFromModel creates a new CubeLight from the Model's dimensions and world transform.
// Note that this does not add it to the Model's hierarchy in any way - the newly created CubeLight
// is still its own Node.
func NewCubeLightFromModel(name string, model *Model) *CubeLight {
	cube := NewCubeLight(name, model.Mesh.Dimensions)
	cube.SetWorldTransform(model.Transform())
	return cube
}

// Clone clones the CubeLight, returning a deep copy.
func (cube *CubeLight) Clone() INode {
	newCube := NewCubeLight(cube.name, cube.Dimensions)
	newCube.Distance = cube.Distance
	newCube.Energy = cube.Energy
	newCube.Color = cube.Color.Clone()
	newCube.On = cube.On
	newCube.Bleed = cube.Bleed
	newCube.LightingAngle = cube.LightingAngle
	newCube.SetWorldTransform(cube.Transform())
	newCube.Node = cube.Node.Clone().(*Node)
	for _, child := range newCube.children {
		child.setParent(newCube)
	}
	return newCube
}

// TransformedDimensions returns the AABB volume dimensions of the CubeLight as they have been scaled, rotated, and positioned to follow the
// CubeLight's node.
func (cube *CubeLight) TransformedDimensions() Dimensions {

	p, s, r := cube.Transform().Decompose()

	corners := [][]float64{
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

		position := r.MultVec(Vector{
			(cube.Dimensions.Width() * c[0] * s.X / 2),
			(cube.Dimensions.Height() * c[1] * s.Y / 2),
			(cube.Dimensions.Depth() * c[2] * s.Z / 2),
			0,
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
	cube.workingDistanceSquared = cube.Distance * cube.Distance
}

func (cube *CubeLight) beginModel(model *Model) {

	p, s, r := model.Transform().Inverted().Decompose()

	// Rather than transforming all vertices of all triangles of a mesh, we can just transform the
	// point light's position by the inversion of the model's transform to get the same effect and save processing time.
	// The same technique is used for Sphere - Triangle collision in bounds.go.

	lightStartPos := cube.WorldPosition().Add(cube.LightingAngle.Invert().Scale(cube.Dimensions.Height() / 2).Sub(cube.Dimensions.Center()))

	cube.workingDimensions = cube.TransformedDimensions()

	if model.skinned {
		cube.workingPosition = lightStartPos
	} else {
		// point.cameraPosition = r.MultVec(camera.WorldPosition()).Add(p)
		cube.workingPosition = r.MultVec(lightStartPos).Add(p)

		cube.workingDimensions.Min = r.MultVec(cube.workingDimensions.Min)
		cube.workingDimensions.Max = r.MultVec(cube.workingDimensions.Max)

		cube.workingDimensions.Min = cube.workingDimensions.Min.Add(p.Mult(Vector{1 / s.X, 1 / s.Y, 1 / s.Z, 0}))
		cube.workingDimensions.Max = cube.workingDimensions.Max.Add(p.Mult(Vector{1 / s.X, 1 / s.Y, 1 / s.Z, 0}))

	}

}

// Light returns the R, G, and B values for the PointLight for all vertices of a given Triangle.
func (cube *CubeLight) Light(sortingTriangles *renderState, model *Model, targetColors []*Color) {

	sortingTriangles.ForEachIndex(func(index int) {

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

		var vertPos, vertNormal Vector

		if model.skinned {
			vertPos = model.Mesh.vertexSkinnedPositions[index]
			vertNormal = model.Mesh.vertexSkinnedNormals[index]
		} else {
			vertPos = model.Mesh.VertexPositions[index]
			vertNormal = model.Mesh.VertexNormals[index]
		}

		var diffuse, diffuseFactor float64

		diffuse = vertNormal.Dot(cube.workingAngle)

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

			if cube.Distance == 0 {
				diffuseFactor = diffuse
			} else {
				diffuseFactor = diffuse * ((cube.workingDistanceSquared - vertPos.DistanceSquared(cube.workingPosition)) / cube.workingDistanceSquared)
			}

			if !cube.workingDimensions.Inside(vertPos) {
				diffuseFactor = 0
			}

		}

		if diffuseFactor <= 0 {
			return
		}

		targetColors[index].AddRGBA(
			cube.Color.R*float32(diffuseFactor)*cube.Energy,
			cube.Color.G*float32(diffuseFactor)*cube.Energy,
			cube.Color.B*float32(diffuseFactor)*cube.Energy,
			0,
		)

	})

}

// IsOn returns if the CubeLight is on or not.
func (cube *CubeLight) IsOn() bool {
	return cube.On
}

// SetOn sets the CubeLight to be on or off.
func (cube *CubeLight) SetOn(on bool) {
	cube.On = on
}

// AddChildren parents the provided children Nodes to the passed parent Node, inheriting its transformations and being under it in the scenegraph
// hierarchy. If the children are already parented to other Nodes, they are unparented before doing so.
func (cube *CubeLight) AddChildren(children ...INode) {
	cube.addChildren(cube, children...)
}

// Unparent unparents the CubeLight from its parent, removing it from the scenegraph.
func (cube *CubeLight) Unparent() {
	if cube.parent != nil {
		cube.parent.RemoveChildren(cube)
	}
}

// Index returns the index of the Node in its parent's children list.
// If the node doesn't have a parent, its index will be -1.
func (cube *CubeLight) Index() int {
	if cube.parent != nil {
		for i, c := range cube.parent.Children() {
			if c == cube {
				return i
			}
		}
	}
	return -1
}

// Type returns the type of INode this is (NodeTypeCubeLight).
func (cube *CubeLight) Type() NodeType {
	return NodeTypeCubeLight
}

// type polygonLightCell struct {
// 	Color    *Color
// 	Distance float64
// }

// type PolygonLight struct {
// 	*Node
// 	Distance float64
// 	Energy   float32
// 	Model    *Model
// 	On       bool
// 	gridSize float64

// 	lightCells [][][]*polygonLightCell
// }

// func NewPolygonLight(model *Model, energy float32, gridSize float64) *PolygonLight {

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

// func (poly *PolygonLight) ResizeGrid(gridSize float64) {

// 	poly.lightCells = [][][]*polygonLightCell{}

// 	dim := poly.Model.Mesh.Dimensions

// 	transform := poly.Model.Transform()

// 	zd := int(math.Ceil(dim.Depth() / gridSize))
// 	yd := int(math.Ceil(dim.Height() / gridSize))
// 	xd := int(math.Ceil(dim.Width() / gridSize))

// 	poly.lightCells = make([][][]*polygonLightCell, zd)

// 	for z := 0; z < zd; z++ {

// 		poly.lightCells[z] = make([][]*polygonLightCell, yd)

// 		for y := 0; y < yd; y++ {

// 			poly.lightCells[z][y] = make([]*polygonLightCell, xd)

// 			for x := 0; x < xd; x++ {

// 				gridPos := Vector{
// 					(float64(x) - float64(xd/2)) * gridSize,
// 					(float64(y) - float64(yd/2)) * gridSize,
// 					(float64(z) - float64(zd/2)) * gridSize,
// 				}
// 				gridPos = transform.MultVec(gridPos)

// 				// closest := poly.Model.Mesh.Triangles[0]
// 				closestDistance := math.MaxFloat64

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

// 		var diffuseFactor float64
// 		distance := fastVectorDistanceSquared(lightPos, vertPos)

// 		diffuseFactor = diffuse * math.Max(math.Min(1.0-(math.Pow((distance/distanceSquared), 4)), 1), 0)

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
