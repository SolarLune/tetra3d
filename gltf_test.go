package tetra3d

import (
	"bytes"
	"os"
	"testing"
)

func BenchmarkLoadGLTFData(b *testing.B) {
	b.StopTimer()
	data, err := os.ReadFile("./examples/logo/tetra3d.glb")
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_, err = LoadGLTFData(bytes.NewBuffer(data), nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
