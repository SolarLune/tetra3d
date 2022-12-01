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

// func BenchmarkMathArrayF64(b *testing.B) {

// 	b.StopTimer()

// 	maxSize := 1200

// 	vecs := make([]VectorFloat, 0, maxSize)

// 	for i := 0; i < maxSize; i++ {
// 		vecs = append(vecs, VectorFloat{rand.Float32(), rand.Float32(), rand.Float32()})
// 	}

// 	b.ReportAllocs()
// 	b.StartTimer()

// 	// Main point of benchmarking
// 	for z := 0; z < b.N; z++ {
// 		for i := 0; i < maxSize-1; i++ {
// 			vecs[i].Add(&vecs[i+1])
// 			vecs[i].Cross(&vecs[i+1])
// 		}
// 	}

// }
