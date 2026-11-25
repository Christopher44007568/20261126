package lut

import (
	"strings"
	"testing"
)

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

func TestIdentityRoundTrip(t *testing.T) {
	l := Identity(9)
	for _, c := range [][3]float64{{0, 0, 0}, {1, 1, 1}, {0.37, 0.5, 0.91}} {
		r, g, b := l.Apply(c[0], c[1], c[2])
		if !near(r, c[0]) || !near(g, c[1]) || !near(b, c[2]) {
			t.Fatalf("identity(%v) = %v,%v,%v", c, r, g, b)
		}
	}
}

func TestSwapRB(t *testing.T) {
	l := SwapRB(5)
	r, g, b := l.Apply(0.25, 0.5, 0.75)
	if !near(r, 0.75) || !near(g, 0.5) || !near(b, 0.25) {
		t.Fatalf("swap = %v,%v,%v", r, g, b)
	}
}

func TestNonUniformDomain(t *testing.T) {
	// Identity mapping sampled over a shifted, per-channel domain.
	min := [3]float64{0.1, 0.0, 0.2}
	max := [3]float64{0.9, 1.0, 0.6}
	l := Build(7, min, max, func(r, g, b float64) (float64, float64, float64) {
		return r, g, b
	})
	// Midpoint of the domain must map to itself.
	in := [3]float64{0.5, 0.5, 0.4}
	r, g, b := l.Apply(in[0], in[1], in[2])
	if !near(r, in[0]) || !near(g, in[1]) || !near(b, in[2]) {
		t.Fatalf("domain identity = %v,%v,%v", r, g, b)
	}
	// Ignoring the domain must produce a visible shift (use an
	// off-center input; at the domain midpoint both readings agree
	// for a linear ramp).
	r2, _, _ := l.ApplyNoDomain(0.3, 0.3, 0.3)
	if near(r2, 0.3) {
		t.Fatalf("ApplyNoDomain unexpectedly matched: %v", r2)
	}
}

func TestParseLineNumbers(t *testing.T) {
	cases := []struct {
		name string
		src  string
		line int
	}{
		{"bad size", "LUT_3D_SIZE two\n", 1},
		{"bad domain", "LUT_3D_SIZE 2\nDOMAIN_MIN 0 0 x\n", 2},
		{"short row", "LUT_3D_SIZE 2\n0 0\n", 2},
		{"bad number", "LUT_3D_SIZE 2\n0 0 0\n1 0 oops\n", 3},
		{"too few rows", "LUT_3D_SIZE 2\n0 0 0\n", 3},
	}
	for _, tc := range cases {
		_, err := Parse(strings.NewReader(tc.src))
		pe, ok := err.(*Error)
		if !ok {
			t.Fatalf("%s: error %v is not *Error", tc.name, err)
		}
		if pe.Line != tc.line {
			t.Fatalf("%s: got line %d, want %d (%v)", tc.name, pe.Line, tc.line, pe)
		}
	}
}

func TestParseTooManyRows(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("LUT_3D_SIZE 2\n")
	for i := 0; i < 9; i++ {
		sb.WriteString("0 0 0\n")
	}
	_, err := Parse(strings.NewReader(sb.String()))
	pe := err.(*Error)
	if pe.Line != 10 { // header line 1 + 9th row at line 10
		t.Fatalf("got line %d, want 10", pe.Line)
	}
}

func TestWriteParseRoundTrip(t *testing.T) {
	l := SwapRB(4)
	l.Title = "swap"
	var sb strings.Builder
	if err := l.Write(&sb); err != nil {
		t.Fatal(err)
	}
	got, err := Parse(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	if got.Size != 4 || got.Title != "swap" || len(got.Data) != len(l.Data) {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}
