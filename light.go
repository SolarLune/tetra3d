package tetra3d

import (
	"math"

	"github.com/kvartborg/vector"
)

// Light represents an interface that is fulfilled by an object that emits light, returning the color a vertex should be given that Vertex and its model matrix.
type Light interface {
	begin()
	Light(vertex *Vertex, model Matrix4) (float32, float32, float32)
	isOn() bool
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

func (amb *AmbientLight) begin() {}

func (amb *AmbientLight) Clone() INode {

	clone := NewAmbientLight(amb.name, amb.Color.R, amb.Color.G, amb.Color.B, amb.Energy)
	clone.On = amb.On

	clone.Node = amb.Node.Clone().(*Node)
	for _, child := range amb.children {
		child.setParent(amb)
	}

	return clone

}

// Light returns the global light level for the ambient light. It doesn't use the vertex or model arguments; this is just to make it adhere to the Light interface.
func (amb *AmbientLight) Light(vertex *Vertex, model Matrix4) (float32, float32, float32) {
	return amb.Color.R * amb.Energy, amb.Color.G * amb.Energy, amb.Color.B * amb.Energy
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

//---------------//

// PointLight represents a point light of infinite point-ness.
type PointLight struct {
	*Node
	Distance float64 // Distance represents the distance after which the light fully attenuates. If this is 0 (the default), it falls off using something akin to the inverse square law.
	Color    *Color  // Color is the color of the PointLight.
	// Energy is the overall energy of the Light. Internally, technically there's no difference between a brighter color and a
	// higher energy, but this is here for convenience / adherance to GLTF / 3D modelers.
	Energy     float32
	On         bool // If the light is on and contributing to the scene.
	vectorPool *VectorPool
}

// NewPointLight creates a new Point light.
func NewPointLight(name string, r, g, b, energy float32) *PointLight {
	return &PointLight{
		Node:       NewNode(name),
		Distance:   0,
		Energy:     energy,
		Color:      NewColor(r, g, b, 1),
		vectorPool: NewVectorPool(2),
		On:         true,
	}
}

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

func (point *PointLight) begin() {

}

// Light returns the R, G, and B values for the point light given the vertex and model matrix provided.
func (point *PointLight) Light(vertex *Vertex, model Matrix4) (float32, float32, float32) {

	point.vectorPool.Reset()

	lightPos := point.WorldPosition()
	vertPos := point.vectorPool.MultVec(model, vertex.Position)

	// Model movement shouldn't be taken into account for normal adjustment
	model = model.SetRow(3, vector.Vector{0, 0, 0, 1})

	normal := point.vectorPool.MultVec(model, vertex.triangle.Normal)

	lightVec := fastVectorSub(lightPos, vertPos).Unit()

	diffuse := normal.Dot(lightVec)
	if diffuse < 0 {
		diffuse = 0
	}

	var diffuseFactor float64
	distance := fastVectorDistanceSquared(lightPos, vertPos)

	if point.Distance == 0 {
		diffuseFactor = diffuse * (1.0 / (1.0 + (0.1 * distance))) * 2
	} else {
		pd := math.Pow(point.Distance, 2)
		diffuseFactor = diffuse * math.Max(math.Min(1.0-(math.Pow((distance/pd), 4)), 1), 0)
	}

	// diffuseFactor := diffuse * (1.0 / (1.0 + (0.25 * distance * distance / (point.Distance * point.Distance))))

	return point.Color.R * float32(diffuseFactor) * point.Energy, point.Color.G * float32(diffuseFactor) * point.Energy, point.Color.B * float32(diffuseFactor) * point.Energy

}

// func (point *PointLight) Light(vertex *Vertex, model Matrix4) (float32, float32, float32) {

// 	var diffuseFactor float64

// 	return point.Color.R * float32(diffuseFactor) * point.Energy, point.Color.G * float32(diffuseFactor) * point.Energy, point.Color.B * float32(diffuseFactor) * point.Energy

// }

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

//---------------//

// DirectionalLight represents a directional light of infinite distance.
type DirectionalLight struct {
	*Node
	Color *Color // Color is the color of the PointLight.
	// Energy is the overall energy of the Light. Internally, technically there's no difference between a brighter color and a
	// higher energy, but this is here for convenience / adherance to GLTF / 3D modelers.
	Energy float32
	On     bool // If the light is on and contributing to the scene.

	forward vector.Vector // internal forward vector so we don't have to calculate it for every vertex for every model using this light
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

func (sun *DirectionalLight) Clone() INode {

	clone := NewDirectionalLight(sun.name, sun.Color.R, sun.Color.G, sun.Color.B, sun.Energy)

	clone.On = sun.On

	clone.Node = sun.Node.Clone().(*Node)
	for _, child := range sun.children {
		child.setParent(sun)
	}

	return clone

}

func (sun *DirectionalLight) begin() {
	sun.forward = sun.WorldRotation().Forward()
}

// Light returns the R, G, and B values for the point light given the vertex and model matrix provided.
func (sun *DirectionalLight) Light(vertex *Vertex, model Matrix4) (float32, float32, float32) {

	// Model movement shouldn't be taken into account for normal adjustment

	normal := model.SetRow(3, vector.Vector{0, 0, 0, 1}).MultVec(vertex.triangle.Normal)

	diffuseFactor := math.Max(normal.Dot(sun.forward), 0.0)

	return sun.Color.R * float32(diffuseFactor) * sun.Energy, sun.Color.G * float32(diffuseFactor) * sun.Energy, sun.Color.B * float32(diffuseFactor) * sun.Energy

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
