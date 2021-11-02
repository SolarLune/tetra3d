package ebiten3d

import (
	"sort"

	"github.com/kvartborg/vector"
)

// Model represents a singular visual instantiation of a Mesh. A Mesh contains the vertex information (what to draw); a Model references the Mesh to draw it with a specific
// Position, Rotation, and/or Scale (where and how to draw).
type Model struct {
	Mesh        *Mesh
	Position    vector.Vector
	Scale       vector.Vector
	Rotation    Quaternion
	closestTris []*Triangle
	FrustumCull bool
	Visible     bool
}

// NewModel creates a new Model (or instance) of the Mesh provided.
func NewModel(mesh *Mesh) *Model {

	return &Model{
		Mesh:        mesh,
		Position:    UnitVector(0),
		Rotation:    NewQuaternion(vector.Vector{0, 1, 0}, 0),
		Scale:       UnitVector(1),
		Visible:     true,
		FrustumCull: true,
	}

}

func (model *Model) Transform() Matrix4 {

	// T * R * S * O

	transform := NewMatrix4()
	transform = transform.Mult(Scale(model.Scale[0], model.Scale[1], model.Scale[2]))
	transform = transform.Mult(Rotate(model.Rotation.Axis, model.Rotation.Angle))
	transform = transform.Mult(Translate(-model.Position[0], model.Position[1], -model.Position[2]))
	return transform

}

// TransformedVertices returns the vertices of the Model, as transformed by the Camera's view matrix, sorted by distance to the Camera's position.
func (model *Model) TransformedVertices(viewMatrix Matrix4, cameraPosition vector.Vector) []vector.Vector {

	verts := []vector.Vector{}

	model.closestTris = append([]*Triangle{}, model.Mesh.Triangles...)

	mvp := model.Transform().Mult(viewMatrix)

	sort.SliceStable(model.closestTris, func(i, j int) bool {
		a := mvp.MultVecW(model.closestTris[i].Center())
		b := mvp.MultVecW(model.closestTris[j].Center())
		return cameraPosition.Sub(a).Magnitude() > cameraPosition.Sub(b).Magnitude()
	})

	for _, tri := range model.closestTris {
		for _, vert := range tri.Vertices {
			verts = append(verts, mvp.MultVecW(vert.Position))
		}
	}

	return verts

}
