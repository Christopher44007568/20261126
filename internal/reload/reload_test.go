package reload

import (
	"testing"

	"hdrexport/internal/hdr"
)

func TestCompareIdentical(t *testing.T) {
	a := hdr.NewImage(4, 3)
	for i := range a.Pix {
		a.Pix[i] = float32(i%7) * 0.37
	}
	b := a.Clone()
	m, before, after := Compare(a, b, 1)
	if m.MaxAbsErr != 0 || m.ByteDiffs != 0 || m.MaxByteD != 0 {
		t.Fatalf("identical images must compare equal, got %+v", m)
	}
	if len(before) != len(after) {
		t.Fatal("rendered buffers length mismatch")
	}
	sheet := BuildSheet(before, after, 4, 3, m)
	if sheet.W <= 4 || sheet.H <= 3 {
		t.Fatalf("sheet not built: %dx%d", sheet.W, sheet.H)
	}
}
