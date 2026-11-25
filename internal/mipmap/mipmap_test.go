package mipmap

import (
	"math"
	"testing"
)

func TestWrapChainConservesMeanTo1x1(t *testing.T) {
	base := patternedImage(16, 16)
	chain, reports := GenerateWrapMipChain(base)

	if got := chain[len(chain)-1]; got.Width != 1 || got.Height != 1 {
		t.Fatalf("final size = %dx%d, want 1x1", got.Width, got.Height)
	}
	if len(reports) != len(chain) {
		t.Fatalf("reports = %d, levels = %d", len(reports), len(chain))
	}

	target := base.Mean()
	for level, img := range chain {
		if err := AverageColorError(img, target); err > 1e-10 {
			t.Fatalf("level %d average color error = %.3e", level, err)
		}
	}
}

func TestOddDimensionsReach1x1WithFluxNormalization(t *testing.T) {
	base := patternedImage(37, 29)
	chain, _ := GenerateWrapMipChain(base)
	last := chain[len(chain)-1]
	if last.Width != 1 || last.Height != 1 {
		t.Fatalf("odd chain final size = %dx%d", last.Width, last.Height)
	}
	if err := MaxMeanColorDrift(chain, base.Mean()); err > 1e-10 {
		t.Fatalf("odd-size mean drift = %.3e", err)
	}
}

func TestConstantWrapChainHasNoEdgeDarkening(t *testing.T) {
	const gray = 0.37
	base := NewRGBImage(32, 32)
	base.FillGray(gray)
	chain, _ := GenerateWrapMipChain(base)

	for level, img := range chain {
		for i, value := range img.Pix {
			if math.Abs(value-gray) > 1e-10 {
				t.Fatalf("level %d sample %d = %.12f, want %.12f", level, i, value, gray)
			}
		}
		if deviation := math.Abs(BorderMeanLuminance(img, 1)/gray - 1); deviation > 1e-10 {
			t.Fatalf("level %d edge deviation = %.3e", level, deviation)
		}
	}
}

func TestNaiveZeroPaddingDarkensSeam(t *testing.T) {
	const gray = 0.5
	base := NewRGBImage(16, 16)
	base.FillGray(gray)
	chain := GenerateNaiveMipChain(base)

	firstBorder := BorderMeanLuminance(chain[1], 1) / gray
	if firstBorder >= 0.99 {
		t.Fatalf("naive first-level border ratio = %.4f, want visible darkening", firstBorder)
	}
	finalLuminance := chain[len(chain)-1].MeanLuminance() / gray
	if finalLuminance >= 0.25 {
		t.Fatalf("naive 1x1 brightness ratio = %.4f, want progressive black collapse", finalLuminance)
	}
}

func TestWrapDownsampleIsPeriodicAndShiftEquivariant(t *testing.T) {
	base := patternedImage(16, 16)
	shifted := NewRGBImage(base.Width, base.Height)
	for y := 0; y < base.Height; y++ {
		for x := 0; x < base.Width; x++ {
			r, g, b := base.WrapRGB(x+2, y+2)
			shifted.Set(x, y, r, g, b)
		}
	}

	expected, _ := DownsampleWrap(base, 1)
	actual, _ := DownsampleWrap(shifted, 1)
	for y := 0; y < actual.Height; y++ {
		for x := 0; x < actual.Width; x++ {
			er, eg, eb := expected.WrapRGB(x+1, y+1)
			ar, ag, ab := actual.RGB(x, y)
			if math.Abs(er-ar) > 1e-12 || math.Abs(eg-ag) > 1e-12 || math.Abs(eb-ab) > 1e-12 {
				t.Fatalf("shift mismatch at %d,%d: expected %.6f actual %.6f", x, y, er, ar)
			}
		}
	}
}

func TestFlatAndProceduralNormalsAreUnitLength(t *testing.T) {
	flat := NewHeightField(8, 8)
	for i := range flat.Pix {
		flat.Pix[i] = 0.42
	}
	normals := HeightToNormal(flat, 3)
	for i := 0; i < len(normals.Pix); i += 3 {
		if math.Abs(normals.Pix[i]) > 1e-12 || math.Abs(normals.Pix[i+1]) > 1e-12 || math.Abs(normals.Pix[i+2]-1) > 1e-12 {
			t.Fatalf("flat normal = %.3f %.3f %.3f", normals.Pix[i], normals.Pix[i+1], normals.Pix[i+2])
		}
	}

	_, height := NewDiagnosticTexture(32, true)
	for level, normals := range NormalMapMipChain(GenerateHeightMipChain(height), 2.0) {
		for i := 0; i < len(normals.Pix); i += 3 {
			nx, ny, nz := normals.Pix[i], normals.Pix[i+1], normals.Pix[i+2]
			length := math.Sqrt(nx*nx + ny*ny + nz*nz)
			if math.Abs(length-1) > 1e-12 {
				t.Fatalf("level %d normal length = %.12f", level, length)
			}
		}
	}
}

func patternedImage(width, height int) *RGBImage {
	img := NewRGBImage(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := 0.1 + 0.7*float64((x*7+y*3)%11)/10.0
			g := 0.2 + 0.6*float64((x*5+y*11)%13)/12.0
			b := 0.3 + 0.5*float64((x*13+y*2)%7)/6.0
			img.Set(x, y, r, g, b)
		}
	}
	return img
}
