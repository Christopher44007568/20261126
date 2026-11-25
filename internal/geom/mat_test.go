package geom

import (
	"math"
	"testing"
)

func TestInvertSimple(t *testing.T) {
	a := Mat3{M: [9]float64{
		1, 2, 0,
		0, 1, 0,
		0, 0, 1,
	}}
	inv := a.Invert()
	want := Mat3{M: [9]float64{
		1, -2, 0,
		0, 1, 0,
		0, 0, 1,
	}}
	for i := range inv.M {
		if math.Abs(inv.M[i]-want.M[i]) > 1e-9 {
			t.Fatalf("inv[%d]=%v want %v\nfull=%v", i, inv.M[i], want.M[i], inv.M)
		}
	}
}

func TestInvertRoundTrip(t *testing.T) {
	a := Mat3{M: [9]float64{
		430, 0, 168000,
		0, 430, 126000,
		0, 0, 700,
	}}
	p := a.Mul(a.Invert())
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			want := 0.0
			if r == c {
				want = 1
			}
			if math.Abs(p.M[r*3+c]-want) > 1e-6 {
				t.Fatalf("round trip not identity: %v", p.M)
			}
		}
	}
}
