package tetra3d

import (
	"testing"
)

func BenchmarkMatrixInversion(b *testing.B) {

	b.ReportAllocs()

	mat := NewMatrix4Rotate(0, 1, 0.2, 0.24).Mult(NewMatrix4Translate(1, 4, -12))

	for i := 0; i < b.N; i++ {
		mat.Inverted()
	}

}

func TestMatrixInversion(t *testing.T) {

	matrices := []Matrix4{
		NewMatrix4Rotate(0, 1, 0, 0.1),
		NewMatrix4Translate(-10, 0.1, 3232.1976),
		NewMatrix4Scale(10, 0.1, -0.45),
		NewMatrix4Translate(-1, -1, -1).Mult(NewMatrix4Rotate(1, 0, 0.1, 0.334)).Mult(NewMatrix4Scale(10, 1, 0)),
	}

	for i, mat := range matrices {

		// As far as I understand it, the Inverted() function only works if multiplying a matrix by the inversion
		// gives you the identity matrix (i.e. it's the inverse, or "negative" Matrix to reverse changes)
		if !mat.Mult(mat.Inverted()).IsIdentity() {
			t.Fatal("failed on matrix #", i, ": matrix * matrix.Inverted() is not identity")
		}

	}

}

// func BenchmarkMatrixInversionOld(b *testing.B) {

// 	b.ReportAllocs()

// 	mat := NewMatrix4Rotate(0, 1, 0.2, 0.24).Mult(NewMatrix4Translate(1, 4, -12))

// 	for i := 0; i < b.N; i++ {
// 		mat.oldInverted()
// 	}

// }
