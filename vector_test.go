package tetra3d

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/solarlune/tetra3d/math32"
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
		vecs := make([][3]float32, 0, 100)
		vecs = append(vecs, [3]float32{0, 0, 0})
	}

}

func BenchmarkAllocateVectorStructs(b *testing.B) {

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		vecs := make([]Vector3, 0, 100)
		vecs = append(vecs, Vector3{0, 0, 0})
	}

}

// func BenchmarkMathExternalVector(b *testing.B) {

// 	b.StopTimer()

// 	maxSize := 1200

// 	vecs := make([]Vector, 0, maxSize)

// 	for i := 0; i < maxSize; i++ {
// 		vecs = append(vecs, vector.Vector{rand.float32(), rand.float32(), rand.float32()})
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

	vecs := make([]Vector3, 0, maxSize)

	for i := 0; i < maxSize; i++ {
		vecs = append(vecs, Vector3{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32()})
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

type testVectorFloat [4]float32

func BenchmarkMathArrayF64(b *testing.B) {

	b.StopTimer()

	maxSize := 1200

	vecs := make([]testVectorFloat, 0, maxSize)

	for i := 0; i < maxSize; i++ {
		vecs = append(vecs, testVectorFloat{rand.Float32(), rand.Float32(), rand.Float32()})
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

// Standard testing doesn't seem to pick up on the time spent navigating to the next piece of data (Vector) in the slice.
// Because of this, the tests below simply time the main portion of the test.
func BenchmarkMVPVector(b *testing.B) {

	maxSize := 600_000

	type Vector64 struct {
		X, Y, Z float32
	}
	vecs := make([]Vector64, 0, maxSize)
	for i := 0; i < maxSize; i++ {
		vecs = append(vecs, Vector64{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32()})
	}

	type Matrix64 [4][4]float32

	mvp := Matrix64{
		{0.1, 0.32, 1.23, 0},
		{0.121, 0.93121, 2.79, 0},
		{0.94, .79, 8.13, 0},
		{0, 0, 0, 1},
	}

	t := time.Now()

	for vecIndex := 0; vecIndex < maxSize; vecIndex++ {
		NX := mvp[0][0]*vecs[vecIndex].X + mvp[1][0]*vecs[vecIndex].Y + mvp[2][0]*vecs[vecIndex].Z
		NY := mvp[0][1]*vecs[vecIndex].X + mvp[1][1]*vecs[vecIndex].Y + mvp[2][1]*vecs[vecIndex].Z
		NZ := mvp[0][2]*vecs[vecIndex].X + mvp[1][2]*vecs[vecIndex].Y + mvp[2][2]*vecs[vecIndex].Z
		vecs[vecIndex].X = NX
		vecs[vecIndex].Y = NY
		vecs[vecIndex].Z = NZ
	}

	fmt.Println(time.Since(t).Microseconds())

}

func BenchmarkMVPVector32(b *testing.B) {

	maxSize := 600_000

	type Vector32 struct {
		X, Y, Z float32
	}
	vecs := make([]Vector32, 0, maxSize)
	for i := 0; i < maxSize; i++ {
		vecs = append(vecs, Vector32{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32()})
	}

	type Matrix32 [4][4]float32

	mvp := Matrix32{
		{0.1, 0.32, 1.23, 0},
		{0.121, 0.93121, 2.79, 0},
		{0.94, .79, 8.13, 0},
		{0, 0, 0, 1},
	}

	t := time.Now()

	for vecIndex := 0; vecIndex < maxSize; vecIndex++ {
		NX := mvp[0][0]*vecs[vecIndex].X + mvp[1][0]*vecs[vecIndex].Y + mvp[2][0]*vecs[vecIndex].Z
		NY := mvp[0][1]*vecs[vecIndex].X + mvp[1][1]*vecs[vecIndex].Y + mvp[2][1]*vecs[vecIndex].Z
		NZ := mvp[0][2]*vecs[vecIndex].X + mvp[1][2]*vecs[vecIndex].Y + mvp[2][2]*vecs[vecIndex].Z
		vecs[vecIndex].X = NX
		vecs[vecIndex].Y = NY
		vecs[vecIndex].Z = NZ
	}

	fmt.Println(time.Since(t).Microseconds())

}

func BenchmarkSqrt(b *testing.B) {

	var result float32

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		result = float32(math.Sqrt(float64(rand.Float32()) * 1000))
	}

	b.StopTimer()

	_ = result

	b.ReportAllocs()

}

func BenchmarkSqrt32(b *testing.B) {

	var result float32

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		result = math32.Sqrt(rand.Float32() * 1000)
	}

	b.StopTimer()

	_ = result

	b.ReportAllocs()

}

func BenchmarkSqrtGeneric(b *testing.B) {

	var result float32

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		result = math32.Sqrt(rand.Float32() * 1000)
	}

	b.StopTimer()

	_ = result

	b.ReportAllocs()

}

func BenchmarkMathTest(b *testing.B) {

	var result float64

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		result = math.Max(math.Max(rand.Float64(), rand.Float64()), rand.Float64())
	}

	b.StopTimer()

	_ = result

	b.ReportAllocs()

}

func helper(numbers ...float64) float64 {
	max := numbers[0]
	for i := 1; i < len(numbers)-1; i++ {
		if max > numbers[i] {
			max = numbers[i]
		}
	}
	return max
}

func BenchmarkMathTest2(b *testing.B) {

	var result float64

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		result = helper(rand.Float64(), rand.Float64(), rand.Float64())
	}

	b.StopTimer()

	_ = result

	b.ReportAllocs()

}
