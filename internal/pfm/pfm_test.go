package pfm

import (
	"bytes"
	"math"
	"testing"

	"hdrexport/internal/hdr"
)

func TestRoundTrip(t *testing.T) {
	src := hdr.NewImage(7, 4)
	values := []float32{0, 0.25, 1, 2.5, 10, -0.5, 1e-3}
	for y := 0; y < 4; y++ {
		for x := 0; x < 7; x++ {
			v := values[x] + float32(y)*0.125
			src.Set(x, y, v, v*0.9, v*0.7)
		}
	}
	var buf bytes.Buffer
	if err := Encode(&buf, src); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if got := string(buf.Bytes()[:3]); got != "PF\n" {
		t.Fatalf("magic = %q, want PF\\n", got)
	}
	got, err := Decode(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.W != src.W || got.H != src.H {
		t.Fatalf("dims = %dx%d, want %dx%d", got.W, got.H, src.W, src.H)
	}
	var maxErr float64
	for i := range src.Pix {
		d := math.Abs(float64(got.Pix[i] - src.Pix[i]))
		if d > maxErr {
			maxErr = d
		}
	}
	if maxErr > 0 {
		t.Fatalf("float32 round trip must be exact, max err %g", maxErr)
	}
}

func TestDecodeBigEndianAndComments(t *testing.T) {
	// Hand-built 2x1 big-endian grayscale PFM with a comment line.
	pix := make([]byte, 8)
	bits := math.Float32bits(3.5)
	for i := 0; i < 4; i++ {
		pix[i] = byte(bits >> (24 - 8*i))
	}
	bits = math.Float32bits(1.25)
	for i := 0; i < 4; i++ {
		pix[4+i] = byte(bits >> (24 - 8*i))
	}
	data := append([]byte("Pf\n# hello\n2 1\n+1.0\n"), pix...)
	m, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if r, _, _ := m.At(0, 0); math.Abs(float64(r)-3.5) > 1e-6 {
		t.Fatalf("pixel 0 = %v, want 3.5", r)
	}
	if r, _, _ := m.At(1, 0); math.Abs(float64(r)-1.25) > 1e-6 {
		t.Fatalf("pixel 1 = %v, want 1.25", r)
	}
}
