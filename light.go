package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

// Light represents an interface that is fulfilled by an object that emits light, returning the color a vertex should be given that Vertex and its model matrix.
type Light interface {
	// beginRender() is used to call any set-up code or to prepare math structures that are used when lighting the scene.
	// It gets called once when first rendering a set of Nodes.
	beginRender()

	// beginModel() is, similarly to beginRender(), used to prepare or precompute any math necessary when lighting the scene.
	// It gets called once before lighting all visible triangles of a given Model.
	beginModel(model *Model, camera *Camera)

	Light(triIndex int, model *Model) [9]float32 // Light() returns the R, G, and B colors used to light the given triangle
	isOn() bool                                  // isOn() is simply used to tell if a "generic" Light is on or not.
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
		child.setParent(amb)
	}

	return clone

}

func (amb *AmbientLight) beginRender() {}

func (amb *AmbientLight) beginModel(model *Model, camera *Camera) {}

// Light returns the light level for the ambient light. It doesn't use the provided Triangle; it takes it as an argument to simply adhere to the Light interface.
func (amb *AmbientLight) Light(triIndex int, model *Model) [9]float32 {
	r := amb.Color.R * amb.Energy
	g := amb.Color.G * amb.Energy
	b := amb.Color.B * amb.Energy
	return [9]float32{r, g, b, r, g, b, r, g, b}
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

func (amb *AmbientLight) isOn() bool {
	return amb.On
}

// Type returns the NodeType for this object.
func (amb *AmbientLight) Type() NodeType {
	return NodeTypeAmbientLight
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

	workingPosition vector.Vector
	cameraPosition  vector.Vector
}

// NewPointLight creates a new Point light.
func NewPointLight(name string, r, g, b, energy float32) *PointLight {
	return &PointLight{
		Node:     NewNode(name),
		Distance: 0,
		Energy:   energy,
		Color:    NewColor(r, g, b, 1),
		On:       true,
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
}

func (point *PointLight) beginModel(model *Model, camera *Camera) {

	p, _, r := model.Transform().Inverted().Decompose()

	// Rather than transforming all vertices of all triangles of a mesh, we can just transform the
	// point light's position by the inversion of the model's transform to get the same effect and save processing time.
	// The same technique is used for Sphere - Triangle collision in bounds.go.

	if model.Skinned {
		point.cameraPosition = camera.WorldPosition()
		point.workingPosition = point.WorldPosition()
	} else {
		point.cameraPosition = r.MultVec(camera.WorldPosition()).Add(p)
		point.workingPosition = r.MultVec(point.WorldPosition()).Add(p)
	}

}

// Light returns the R, G, and B values for the PointLight for all vertices of a given Triangle.
func (point *PointLight) Light(triIndex int, model *Model) [9]float32 {

	light := [9]float32{}

	// TODO: Make lighting faster by returning early if the triangle is too far from the point light position

	// We calculate both the eye vector as well as the light vector so that if the camera passes behind the
	// lit face and backface culling is off, the triangle can still be lit or unlit from the other side. Otherwise,
	// if the triangle were lit by a light, it would appear lit regardless of the positioning of the camera.

	var triCenter vector.Vector

	if model.Skinned {
		v0 := model.Mesh.vertexSkinnedPositions[triIndex*3].Clone()
		v1 := model.Mesh.vertexSkinnedPositions[triIndex*3+1]
		v2 := model.Mesh.vertexSkinnedPositions[triIndex*3+2]
		triCenter = vector.Vector(vector.In(v0).Add(v1).Add(v2).Scale(1.0 / 3.0))
	} else {
		triCenter = model.Mesh.Triangles[triIndex].Center
	}

	dist := fastVectorSub(point.workingPosition, triCenter)

	distanceSquared := math.Pow(point.Distance, 2)

	if point.Distance > 0 && dist.Magnitude() > distanceSquared {
		return light
	}

	// If you're on the other side of the plane, just assume it's not visible.
	// if dot(model.Mesh.Triangles[triIndex].Normal, fastVectorSub(triCenter, point.cameraPosition).Unit()) < 0 {
	// 	return light
	// }

	var vertPos, vertNormal vector.Vector

	for i := 0; i < 3; i++ {

		if model.Skinned {
			vertPos = model.Mesh.vertexSkinnedPositions[triIndex*3+i]
			vertNormal = model.Mesh.vertexSkinnedNormals[triIndex*3+i]
		} else {
			vertPos = model.Mesh.VertexPositions[triIndex*3+i]
			vertNormal = model.Mesh.VertexNormals[triIndex*3+i]
		}

		lightVec := vector.In(fastVectorSub(point.workingPosition, vertPos)).Unit()
		diffuse := dot(vertNormal, vector.Vector(lightVec))

		if diffuse < 0 {
			diffuse = 0
		}

		var diffuseFactor float64
		distance := fastVectorDistanceSquared(point.workingPosition, vertPos)

		if point.Distance == 0 {
			diffuseFactor = diffuse * (1.0 / (1.0 + (0.1 * distance))) * 2
		} else {
			diffuseFactor = diffuse * math.Max(math.Min(1.0-(math.Pow((distance/distanceSquared), 4)), 1), 0)
		}

		light[(i * 3)] = point.Color.R * float32(diffuseFactor) * point.Energy
		light[(i*3)+1] = point.Color.G * float32(diffuseFactor) * point.Energy
		light[(i*3)+2] = point.Color.B * float32(diffuseFactor) * point.Energy

	}

	return light

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

func (point *PointLight) isOn() bool {
	return point.On
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

	workingForward       vector.Vector // Internal forward vector so we don't have to calculate it for every triangle for every model using this light.
	workingModelRotation Matrix4       // Similarly, this is an internal rotational transform (without the transformation row) for the Model being lit.
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
		child.setParent(sun)
	}

	return clone

}

func (sun *DirectionalLight) beginRender() {
	sun.workingForward = sun.WorldRotation().Forward() // Already reversed
}

func (sun *DirectionalLight) beginModel(model *Model, camera *Camera) {
	if !model.Skinned {
		sun.workingModelRotation = model.WorldRotation().Inverted().Transposed()
	}
}

// Light returns the R, G, and B values for the DirectionalLight for each vertex of the provided Triangle.
func (sun *DirectionalLight) Light(triIndex int, model *Model) [9]float32 {

	light := [9]float32{}

	for i := 0; i < 3; i++ {

		var normal vector.Vector
		if model.Skinned {
			// If it's skinned, we don't have to calculate the normal, as that's been pre-calc'd for us
			normal = model.Mesh.vertexSkinnedNormals[triIndex*3+i]
		} else {
			normal = sun.workingModelRotation.MultVec(model.Mesh.VertexNormals[triIndex*3+i])
		}

		diffuseFactor := dot(normal, sun.workingForward)
		if diffuseFactor < 0 {
			diffuseFactor = 0
		}

		light[i*3] = sun.Color.R * float32(diffuseFactor) * sun.Energy
		light[i*3+1] = sun.Color.G * float32(diffuseFactor) * sun.Energy
		light[i*3+2] = sun.Color.B * float32(diffuseFactor) * sun.Energy

	}

	return light

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

func (sun *DirectionalLight) isOn() bool {
	return sun.On
}

// Type returns the NodeType for this object.
func (sun *DirectionalLight) Type() NodeType {
	return NodeTypeDirectionalLight
}
