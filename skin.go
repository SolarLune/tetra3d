package tetra3d

import (
	"github.com/kvartborg/vector"
)

type Skin struct {
	Enabled             bool
	skinMatrix          Matrix4
	invertedModelMatrix Matrix4
}

func NewSkin() *Skin {
	skin := &Skin{
		Enabled:             true,
		skinMatrix:          NewMatrix4(),
		invertedModelMatrix: NewMatrix4(), // InvertedModelMatrix is calculated in the Camera.Render() function.
	}
	return skin
}

// Transform transforms the input vertex using the weights set up to bend the vertex according to nodes playing back a movement by means of an animation.
func (skin *Skin) Transform(vertex *Vertex) vector.Vector {

	// Avoid reallocating a new matrix for every vertex; that's wasteful
	skin.skinMatrix.Clear()

	for boneIndex, bone := range vertex.Bones {

		weightPerc := float64(vertex.Weights[boneIndex])

		influence := fastMatrixMult(fastMatrixMult(bone.inverseBindMatrix, bone.Transform()), skin.invertedModelMatrix)

		if weightPerc == 1 {
			skin.skinMatrix = influence
		} else {
			skin.skinMatrix = skin.skinMatrix.Add(influence.ScaleByScalar(weightPerc))
		}

	}

	vertOut := skin.skinMatrix.MultVecW(vertex.Position)

	return vertOut

}
