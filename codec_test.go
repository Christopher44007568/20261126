package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBlockModesRoundTrip(t *testing.T) {
	opaque := gradientBlock(255)
	mask := gradientBlock(255)
	for i := range mask {
		if i%5 == 0 {
			mask[i].A = 0
		}
	}
	alpha4 := gradientBlock(255)
	for i := range alpha4 {
		alpha4[i].A = uint8(20 + i*13)
	}
	var empty Block

	tests := []struct {
		name  string
		block Block
		mode  int
		size  int
	}{
		{"opaque", opaque, alphaOpaque, 17},
		{"mask", mask, alphaMask, 19},
		{"alpha4", alpha4, alphaNibble, 25},
		{"empty", empty, alphaEmpty, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enc := EncodeBlock(tc.block)
			if got := int(enc.Data[0] & 3); got != tc.mode {
				t.Fatalf("alpha mode=%d want %d", got, tc.mode)
			}
			if len(enc.Data) != tc.size {
				t.Fatalf("record size=%d want %d", len(enc.Data), tc.size)
			}
			if size, err := RecordSize(enc.Data[0]); err != nil || size != tc.size {
				t.Fatalf("RecordSize=%d,%v want %d", size, err, tc.size)
			}
			dec, err := DecodeBlock(enc.Data)
			if err != nil {
				t.Fatal(err)
			}
			checkBlockAlpha(t, tc.mode, tc.block, dec)
			if tc.mode == alphaEmpty {
				for _, p := range dec {
					if p.A != 0 {
						t.Fatal("empty block produced a visible texel")
					}
				}
			}
		})
	}
}

func TestBinaryAlphaMaskIsExact(t *testing.T) {
	var block Block
	for i := range block {
		block[i] = Pixel{R: uint8(i * 15), G: 200, B: 40, A: 255}
		if i == 0 || i == 3 || i == 6 || i == 10 || i == 15 {
			block[i].A = 0
		}
	}
	enc := EncodeBlock(block)
	dec, err := DecodeBlock(enc.Data)
	if err != nil {
		t.Fatal(err)
	}
	for i := range block {
		want := uint8(255)
		if block[i].A == 0 {
			want = 0
		}
		if dec[i].A != want {
			t.Fatalf("texel %d alpha=%d want %d", i, dec[i].A, want)
		}
	}
}

func TestAlpha4NeverCreatesFalseHoles(t *testing.T) {
	var block Block
	for i := range block {
		block[i] = Pixel{R: 90, G: 130, B: 170, A: uint8(1 + i*16)}
	}
	enc := EncodeBlock(block)
	dec, err := DecodeBlock(enc.Data)
	if err != nil {
		t.Fatal(err)
	}
	for i := range block {
		if block[i].A > 0 && dec[i].A == 0 {
			t.Fatalf("texel %d: nonzero alpha %d became transparent", i, block[i].A)
		}
		if block[i].A < 255 && dec[i].A == 255 {
			t.Fatalf("texel %d: partial alpha %d became opaque", i, block[i].A)
		}
	}
}

func TestScenariosRoundTripAndCompress(t *testing.T) {
	for _, scenario := range GenerateScenarios() {
		t.Run(scenario.ID, func(t *testing.T) {
			stream, _, err := CompressBitmap(scenario.Original)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecompressBitmap(stream)
			if err != nil {
				t.Fatal(err)
			}
			metrics, err := CompareMetrics(scenario.Original, decoded)
			if err != nil {
				t.Fatal(err)
			}
			SetCompressedSize(&metrics, len(stream))
			if metrics.VisibleRGBMAE > 1.5 {
				t.Fatalf("visible RGB MAE %.3f is too high", metrics.VisibleRGBMAE)
			}
			if metrics.CompressionRatio < 3.5 {
				t.Fatalf("compression ratio %.2f is too low", metrics.CompressionRatio)
			}
			if metrics.AlphaMAE > 0.01 {
				t.Fatalf("binary alpha changed, MAE %.3f", metrics.AlphaMAE)
			}
			if scenario.ID == "holes" {
				if metrics.AlphaClassErrors != 0 || metrics.HoleFillErrors != 0 || metrics.HoleErosionErrors != 0 {
					t.Fatalf("transparent hole broke: class=%d fill=%d erosion=%d",
						metrics.AlphaClassErrors, metrics.HoleFillErrors, metrics.HoleErosionErrors)
				}
				if metrics.BoundaryPixels == 0 {
					t.Fatal("hole boundary was not measured")
				}
			}
		})
	}
}

func TestAlphaAuditCatchesBreakageHiddenByRGBMAE(t *testing.T) {
	holes := transparentHolesTexture()
	filled := cloneBitmap(holes)
	eroded := cloneBitmap(holes)

	fillX, fillY := holeEdge(holes, true)
	if fillX < 0 {
		t.Fatal("no transparent texel found")
	}
	filled.Pix[fillY*filled.W+fillX].A = 255
	erodeX, erodeY := holeEdge(holes, false)
	if erodeX < 0 {
		t.Fatal("no visible texel found")
	}
	eroded.Pix[erodeY*eroded.W+erodeX].A = 0

	for _, tc := range []struct {
		name string
		im   *Bitmap
	}{
		{"fill", filled},
		{"erode", eroded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := CompareMetrics(holes, tc.im)
			if err != nil {
				t.Fatal(err)
			}
			if m.VisibleRGBMAE != 0 {
				t.Fatalf("test setup should leave visible RGB MAE at zero, got %.6f", m.VisibleRGBMAE)
			}
			if m.AlphaClassErrors != 1 {
				t.Fatalf("alpha audit missed breakage: class errors=%d", m.AlphaClassErrors)
			}
			if tc.name == "fill" && m.HoleFillErrors != 1 {
				t.Fatalf("hole fill errors=%d want 1", m.HoleFillErrors)
			}
			if tc.name == "erode" && m.HoleErosionErrors != 1 {
				t.Fatalf("hole erosion errors=%d want 1", m.HoleErosionErrors)
			}
		})
	}
}

func TestErrorRenderMarksAlphaBreakInMagenta(t *testing.T) {
	orig := NewBitmap(4, 4)
	dec := NewBitmap(4, 4)
	for i := range orig.Pix {
		orig.Pix[i] = Pixel{A: 255}
		dec.Pix[i] = Pixel{A: 255}
	}
	dec.Set(1, 1, Pixel{R: 30, G: 40, B: 50, A: 0})
	panel := renderErrorPanel(orig, dec, 4)
	p := panel.At(1*4+2, 1*4+2)
	if p.R != 255 || p.G != 0 || p.B != 198 {
		t.Fatalf("alpha break pixel=%+v, want magenta", p)
	}
}

func TestMalformedStreamsAreRejected(t *testing.T) {
	stream, _, err := CompressBitmap(GenerateScenarios()[0].Original)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecompressBitmap(stream[:len(stream)-1]); err == nil {
		t.Fatal("truncated stream accepted")
	}
	if _, err := DecompressBitmap(append(stream, 0)); err == nil {
		t.Fatal("stream with trailing bytes accepted")
	}
}

func TestRunCreatesArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := run(dir, 1); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"overview.png", "original.png", "decoded.png", "error_x8.png",
		"report.html", "report.txt", "metrics.json", "compressed_hex.txt",
	} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "streams", "holes.bd4")); err != nil {
		t.Fatal(err)
	}
}

func gradientBlock(alpha uint8) Block {
	var b Block
	for i := range b {
		b[i] = Pixel{
			R: uint8(16 + i*13),
			G: uint8(200 - i*9),
			B: uint8(40 + (i%4)*17),
			A: alpha,
		}
	}
	return b
}

func checkBlockAlpha(t *testing.T, mode int, src, dec Block) {
	t.Helper()
	switch mode {
	case alphaEmpty:
	case alphaOpaque:
		for i := range src {
			if dec[i].A != 255 {
				t.Fatalf("opaque texel %d alpha=%d", i, dec[i].A)
			}
		}
	case alphaMask:
		for i := range src {
			want := uint8(255)
			if src[i].A == 0 {
				want = 0
			}
			if dec[i].A != want {
				t.Fatalf("mask texel %d alpha=%d want %d", i, dec[i].A, want)
			}
		}
	case alphaNibble:
		for i := range src {
			if src[i].A > 0 && dec[i].A == 0 {
				t.Fatalf("alpha4 texel %d became transparent", i)
			}
			if src[i].A < 255 && dec[i].A == 255 {
				t.Fatalf("alpha4 texel %d became opaque", i)
			}
		}
	}
}

func cloneBitmap(im *Bitmap) *Bitmap {
	out := NewBitmap(im.W, im.H)
	copy(out.Pix, im.Pix)
	return out
}

func holeEdge(im *Bitmap, transparent bool) (int, int) {
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			p := im.At(x, y)
			if transparent && p.A != 0 {
				continue
			}
			if !transparent && p.A == 0 {
				continue
			}
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= im.W || ny >= im.H {
					continue
				}
				if (im.At(nx, ny).A == 0) != (p.A == 0) {
					return x, y
				}
			}
		}
	}
	return -1, -1
}

func TestStreamMagicIsStable(t *testing.T) {
	stream, _, err := CompressBitmap(transparentHolesTexture())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stream[:4], []byte("BD4X")) {
		t.Fatalf("magic=%q", stream[:4])
	}
}
