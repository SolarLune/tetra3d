package tetra3d

import (
	"math"
	"testing"
)

// The depth shaders of NewCamera pack a depth from 0 to 1 into the red, green,
// and blue bytes of the depth texture. These functions repeat their encodeDepth
// and decodeDepth in float32, with an explicit conversion after each operation,
// as a GPU without fused operations computes them. The GPU stores each channel
// as round(x * 255).

// depthUnits is the number of steps of the three-byte encoding, 255 cubed.
const depthUnits = 255 * 65025

func fract32(x float32) float32 { return x - float32(math.Floor(float64(x))) }
func floor32(x float32) float32 { return float32(math.Floor(float64(x))) }

// store returns the byte that the GPU stores for a channel value.
func store(x float32) float32 { return float32(math.Round(float64(x) * 255)) }

// oldEncodeDepth is encodeDepth before the fix: blue comes from a second
// product, (depth*255)*255, which rounds apart from the product of green.
func oldEncodeDepth(d float32) (r, g, b float32) {
	x := float32(d * 255)
	r = float32(floor32(x) / 255)
	g = float32(floor32(float32(fract32(x)*255)) / 255)
	b = fract32(float32(x * 255))
	return store(r), store(g), store(b)
}

// newEncodeDepth is encodeDepth: green and blue come from one product.
func newEncodeDepth(d float32) (r, g, b float32) {
	x := float32(d * 255)
	r = float32(floor32(x) / 255)
	y := float32(fract32(x) * 255)
	return store(r), store(float32(floor32(y) / 255)), store(fract32(y))
}

// oldDecodeDepth is decodeDepth before the fix. The channels are the floats
// that the texture unit gives for the bytes.
func oldDecodeDepth(r, g, b float32) float32 {
	return float32(float32(r+float32(g/255)) + float32(b/65025))
}

// newDecodeDepth is decodeDepth: it rounds each channel to its byte first.
func newDecodeDepth(r, g, b float32) float32 {
	cr := floor32(float32(r*255) + 0.5)
	cg := floor32(float32(g*255) + 0.5)
	cb := floor32(float32(b*255) + 0.5)
	return float32(float32(float32(cr*65025)+float32(cg*255))+cb) / depthUnits
}

// exact is the depth that three stored bytes hold.
func exact(r, g, b float32) float64 {
	return (float64(r)*65025 + float64(g)*255 + float64(b)) / depthUnits
}

// TestDepthStep checks the depth step of the encoding for a camera from 0.3 m
// to 1,000 m with the default DepthMargin of 0.04: 16.6 mm for two bytes, and
// 0.065 mm for three.
func TestDepthStep(t *testing.T) {
	near, far, margin := 0.3, 1000.0, 0.04
	spread := (far - near) * (1 + 2*margin)
	if two := spread / 65025 * 1000; math.Abs(two-16.6) > 0.05 {
		t.Errorf("two-byte step is %.2f mm, want 16.6 mm", two)
	}
	if three := spread / depthUnits * 1000; math.Abs(three-0.065) > 0.001 {
		t.Errorf("three-byte step is %.4f mm, want 0.065 mm", three)
	}
}

// TestEncodeDepthCarry checks that the stored bytes hold the depth to one
// step. The old encoder took blue from a second product, so where green
// carries, blue could wrap without green, and the stored depth moved by one
// green step: 255 steps, 16.6 mm for a camera from 0.3 m to 1,000 m.
func TestEncodeDepthCarry(t *testing.T) {
	const n = 2_000_000
	// sweep returns the depths off by more than limit steps, and the worst.
	sweep := func(enc func(float32) (float32, float32, float32), limit float64) (bad int, worst float64) {
		for i := range n {
			d := float32(float64(i) / n)
			e := math.Abs(exact(enc(d))-float64(d)) * depthUnits
			worst = max(worst, e)
			if e > limit {
				bad++
			}
		}
		return
	}
	bad, worst := sweep(oldEncodeDepth, 128)
	if bad == 0 || worst < 254 {
		t.Errorf("old encoder: %d of %d depths off by more than 128 steps, worst %.1f steps; want the carry error of about 255 steps", bad, n, worst)
	}
	t.Logf("old encoder: %d of %d depths (%.2f%%) off by more than 128 steps, worst %.1f steps", bad, n, 100*float64(bad)/n, worst)
	if bad, worst := sweep(newEncodeDepth, 1); bad > 0 {
		t.Errorf("new encoder: %d of %d depths off by more than one step, worst %.1f steps", bad, n, worst)
	}
}

// TestDecodeDepthInexactChannels checks that the decode holds when the texture
// unit converts a byte to a float with a small error. Mesa's software
// renderer, llvmpipe, reads byte c as about c/255 * (1 - 1.7e-4), which moved
// the old decode by up to about 11 steps for each unit of red, and the
// depth test between draw calls failed by up to 20 mm.
func TestDecodeDepthInexactChannels(t *testing.T) {
	const relErr = 1.7e-4
	read := func(c float32) float32 { return float32(float64(c) / 255 * (1 - relErr)) }
	oldWorst, newWorst := 0.0, 0.0
	for _, c := range [][3]float32{{0, 0, 0}, {3, 250, 255}, {13, 100, 200}, {100, 100, 1}, {200, 7, 128}, {255, 0, 0}} {
		want := exact(c[0], c[1], c[2])
		r, g, b := read(c[0]), read(c[1]), read(c[2])
		oldWorst = max(oldWorst, math.Abs(float64(oldDecodeDepth(r, g, b))-want)*depthUnits)
		newWorst = max(newWorst, math.Abs(float64(newDecodeDepth(r, g, b))-want)*depthUnits)
	}
	if oldWorst < 255 {
		t.Errorf("old decode: worst error %.1f steps; want more than one green step, 255 steps", oldWorst)
	}
	if newWorst > 1 {
		t.Errorf("new decode: worst error %.1f steps, want at most one", newWorst)
	}
}

// TestDepthRoundTrip checks that the new decode reads what the new encoder
// stores, and orders two depths one step apart, over the whole range.
func TestDepthRoundTrip(t *testing.T) {
	for i := 0; i <= depthUnits; i += 997 {
		d := float32(float64(i) / depthUnits)
		r, g, b := newEncodeDepth(d)
		got := newDecodeDepth(r/255, g/255, b/255)
		if e := math.Abs(float64(got)-float64(d)) * depthUnits; e > 1 {
			t.Fatalf("depth %v: decoded %v, %.1f steps off", d, got, e)
		}
		if nearer := float32(float64(i-2) / depthUnits); i >= 2 && !(got > nearer) {
			t.Fatalf("depth %v: decoded %v is not behind %v, two steps nearer", d, got, nearer)
		}
	}
}
