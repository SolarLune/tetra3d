package tetra3d

import (
	"math/rand"
	"testing"
)

// func BenchmarkAllocateVectors(b *testing.B) {

// 	b.ReportAllocs()

// 	for i := 0; i < b.N; i++ {
// 		vecs := make([]Vector, 0, 100)
// 		vecs = append(vecs, vector.Vector{0, 0, 0})
// 	}

// }

func BenchmarkAllocateArraysF64(b *testing.B) {

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vecs := make([][3]float64, 0, 100)
		vecs = append(vecs, [3]float64{0, 0, 0})
	}

}

func BenchmarkAllocateVectorStructs(b *testing.B) {

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vecs := make([]Vector, 0, 100)
		vecs = append(vecs, Vector{0, 0, 0, 0})
	}

}

// func BenchmarkMathExternalVector(b *testing.B) {

// 	b.StopTimer()

// 	maxSize := 1200

// 	vecs := make([]Vector, 0, maxSize)

// 	for i := 0; i < maxSize; i++ {
// 		vecs = append(vecs, vector.Vector{rand.Float64(), rand.Float64(), rand.Float64()})
// 	}

// 	b.ReportAllocs()
// 	b.StartTimer()

// 	// Main point of benchmarking
// 	for z := 0; z < b.N; z++ {
// 		for i := 0; i < maxSize-1; i++ {
// 			vector.In(vecs[i]).Add(vecs[i+1])
// 			vectorCrossUnsafe(vecs[i], vecs[i+1], vecs[i])
// 		}
// 	}

// }

func BenchmarkMathInternalVector(b *testing.B) {

	b.StopTimer()

	maxSize := 1200

	vecs := make([]Vector, 0, maxSize)

	for i := 0; i < maxSize; i++ {
		vecs = append(vecs, Vector{X: rand.Float64(), Y: rand.Float64(), Z: rand.Float64()})
	}

	b.ReportAllocs()
	b.StartTimer()

	// Main point of benchmarking
	for z := 0; z < b.N; z++ {
		for i := 0; i < maxSize-1; i++ {
			vecs[i] = vecs[i].Add(vecs[i+1]).Cross(vecs[i+1])
		}
	}

}

// Benchmark function for previous iteration of vectors, which were type definitions that just pointed to [4]float32.

type testVectorFloat [4]float64

func BenchmarkMathArrayF64(b *testing.B) {

	b.StopTimer()

	maxSize := 1200

	vecs := make([]testVectorFloat, 0, maxSize)

	for i := 0; i < maxSize; i++ {
		vecs = append(vecs, testVectorFloat{rand.Float64(), rand.Float64(), rand.Float64()})
	}

	b.ReportAllocs()
	b.StartTimer()

	// Main point of benchmarking
	for z := 0; z < b.N; z++ {
		for i := 0; i < maxSize-1; i++ {
			// Add
			vecs[i][0] += vecs[i+1][0]
			vecs[i][1] += vecs[i+1][1]
			vecs[i][2] += vecs[i+1][2]

			// Cross
			ogVecY := vecs[i][1]
			ogVecZ := vecs[i][2]

			vecs[i][2] = vecs[i][0]*vecs[i+1][1] - vecs[i+1][0]*vecs[i][1]
			vecs[i][1] = ogVecZ*vecs[i+1][1] - vecs[i+1][2]*vecs[i][0]
			vecs[i][0] = ogVecY*vecs[i+1][2] - vecs[i+1][1]*ogVecZ
		}
	}

}

// type testPaddedVector struct {
// 	X, Y, Z, W float32
// 	_          uint8
// }

// func BenchmarkPaddedVector(b *testing.B) {

// 	maxSize := 1200

// 	vecs := make([]testPaddedVector, 0, maxSize)

// 	for i := 0; i < maxSize; i++ {
// 		vecs = append(vecs, testPaddedVector{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32()})
// 	}

// 	b.StartTimer()

// 	// Main point of benchmarking
// 	for z := 0; z < b.N; z++ {
// 		for i := 0; i < maxSize-1; i++ {
// 			n := rand.Intn(maxSize - 1)
// 			// Add
// 			vecs[i].X += vecs[n].X
// 			vecs[i].Y += vecs[n].Y
// 			vecs[i].Z += vecs[n].Z

// 			// Cross
// 			ogVecY := vecs[n].Y
// 			ogVecZ := vecs[n].Z

// 			vecs[i].Z = vecs[i].X*vecs[n].Y - vecs[n].X*vecs[i].Y
// 			vecs[i].Y = ogVecZ*vecs[n].Y - vecs[n].Z*vecs[i].X
// 			vecs[i].X = ogVecY*vecs[n].Z - vecs[n].Y*ogVecZ
// 		}
// 	}

// 	b.StopTimer()

// 	b.ReportAllocs()

// }
