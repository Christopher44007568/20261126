package preview

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"dds-preview/internal/dds"
)

func buildPreviewDDS() []byte {
	return buildPreviewDDSFaces(1)
}

func buildPreviewDDSFaces(faces int) []byte {
	header := make([]byte, 128)
	copy(header, "DDS ")
	put := func(off int, v uint32) { binary.LittleEndian.PutUint32(header[off:], v) }
	put(4, 124)
	put(8, 0x1|0x2|0x4|0x8|0x1000|0x20000)
	put(12, 4)
	put(16, 4)
	put(20, 20)
	put(28, 2)
	put(76, 32)
	put(80, 1|0x40)
	put(88, 32)
	put(92, 0xFF)
	put(96, 0xFF00)
	put(100, 0xFF0000)
	put(104, 0xFF000000)
	put(112, 0)
	if faces > 1 {
		put(112, 0x200|0x400|0x800|0x1000|0x2000|0x4000|0x8000)
	}

	data := header
	for face := 0; face < faces; face++ {
		for mip, dim := range []int{4, 2} {
			rowBytes := dim * 4
			for y := 0; y < dim; y++ {
				row := make([]byte, rowBytes)
				for x := 0; x < dim; x++ {
					at := x * 4
					row[at] = byte(20*mip + x*40 + face)
					row[at+1] = byte(30 + y*50)
					row[at+2] = byte(200 - mip*80)
					row[at+3] = 255
				}
				if mip == 0 {
					row = append(row, 0xAA, 0xBB, 0xCC, 0xDD)
				}
				data = append(data, row...)
			}
		}
	}
	return data
}

func TestPreviewKeepsLastValidTexture(t *testing.T) {
	valid := buildPreviewDDS()
	p := New()
	res := p.Load(valid, "first.dds")
	if !res.OK || p.current == nil {
		t.Fatalf("first load: ok=%v current=%v errors=%+v", res.OK, p.current, res.Errors)
	}
	out := t.TempDir()
	if err := WritePreview(p.current, res, out); err != nil {
		t.Fatalf("WritePreview: %v", err)
	}
	if err := WriteReport(res, out); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}
	spherePath := filepath.Join(out, "sphere.ppm")
	before, err := os.ReadFile(spherePath)
	if err != nil {
		t.Fatal(err)
	}
	mipBefore, err := os.ReadFile(filepath.Join(out, "mip_1.ppm"))
	if err != nil {
		t.Fatal(err)
	}

	short := valid[:len(valid)-1]
	bad := p.Load(short, "truncated.dds")
	if bad.OK || !bad.KeptPrevious || bad.Current == nil {
		t.Fatalf("truncated load: %+v", bad)
	}
	if bad.Current.Width != 4 || bad.Current.MipCount != 2 || bad.PreviousSource != "first.dds" {
		t.Fatalf("previous texture was not retained: %+v", bad.Current)
	}
	if len(bad.Errors) != 1 || bad.Errors[0].Field != "data.face[0].mip[1]" {
		t.Fatalf("field diagnostics: %+v", bad.Errors)
	}
	if err := WriteReport(bad, out); err != nil {
		t.Fatalf("WriteReport invalid: %v", err)
	}
	after, err := os.ReadFile(spherePath)
	if err != nil {
		t.Fatal(err)
	}
	mipAfter, err := os.ReadFile(filepath.Join(out, "mip_1.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || !bytes.Equal(mipBefore, mipAfter) {
		t.Fatal("invalid load changed previously rendered images")
	}

	reportBytes, err := os.ReadFile(filepath.Join(out, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report Result
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatal(err)
	}
	if report.OK || !report.KeptPrevious || report.Current == nil || report.Errors[0].Field == "" {
		t.Fatalf("report = %+v", report)
	}
}

func TestPPMAndSphereDimensions(t *testing.T) {
	tex, err := dds.Parse(buildPreviewDDS())
	if err != nil {
		t.Fatal(err)
	}
	ppm, err := PPM(tex.Base(0))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(ppm, []byte("P6\n4 4\n255\n")) {
		t.Fatalf("PPM header = %q", ppm[:16])
	}
	sphere := RenderSphere(tex)
	if sphere.Width != tex.Width || sphere.Height != tex.Height {
		t.Fatalf("sphere is %dx%d, want original %dx%d", sphere.Width, sphere.Height, tex.Width, tex.Height)
	}
	if _, err := PPM(sphere); err != nil {
		t.Fatalf("sphere PPM: %v", err)
	}
}

func TestPPMChannelOrder(t *testing.T) {
	img := dds.Image{Width: 2, Height: 1, RGBA: []byte{
		10, 20, 30, 255,
		40, 50, 60, 255,
	}}
	ppm, err := PPM(img)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("P6\n2 1\n255\n\x0a\x14\x1e\x28\x32\x3c")
	if !bytes.Equal(ppm, want) {
		t.Fatalf("PPM = %q, want %q", ppm, want)
	}
}

func TestInvalidDimensionsKeepPreviousTexture(t *testing.T) {
	valid := buildPreviewDDS()
	p := New()
	if res := p.Load(valid, "good.dds"); !res.OK {
		t.Fatal("valid fixture rejected")
	}
	out := t.TempDir()
	res := p.Load(valid, "good.dds")
	if err := WritePreview(p.current, res, out); err != nil {
		t.Fatal(err)
	}
	sphereBefore, err := os.ReadFile(filepath.Join(out, "sphere.ppm"))
	if err != nil {
		t.Fatal(err)
	}

	invalid := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(invalid[16:], 0)
	bad := p.Load(invalid, "bad-dimensions.dds")
	if bad.OK || !bad.KeptPrevious || bad.Current == nil {
		t.Fatalf("invalid dimensions: %+v", bad)
	}
	if len(bad.Errors) == 0 || bad.Errors[0].Field != "header.dwWidth/dwHeight" {
		t.Fatalf("field diagnostics: %+v", bad.Errors)
	}
	sphereAfter, err := os.ReadFile(filepath.Join(out, "sphere.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sphereBefore, sphereAfter) {
		t.Fatal("invalid dimensions changed the rendered sphere")
	}
}

func TestCubeArtifactsCoverEveryFaceAndMip(t *testing.T) {
	data := buildPreviewDDSFaces(6)
	p := New()
	res := p.Load(data, "cube.dds")
	if !res.OK || res.Current == nil || res.Current.FaceCount != 6 {
		t.Fatalf("cube load: %+v", res)
	}
	out := t.TempDir()
	if err := WritePreview(p.current, res, out); err != nil {
		t.Fatalf("WritePreview cube: %v", err)
	}
	if len(res.Artifacts) != 13 {
		t.Fatalf("artifacts = %d, want 12 mips + sphere", len(res.Artifacts))
	}
	for face := 0; face < 6; face++ {
		for mip := 0; mip < 2; mip++ {
			path := filepath.Join(out, "face_"+strconv.Itoa(face)+"_mip_"+strconv.Itoa(mip)+".ppm")
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("missing %s: %v", path, err)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(out, "sphere.ppm")); err != nil {
		t.Fatalf("missing cube sphere: %v", err)
	}
}
