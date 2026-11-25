package dds

import (
	"encoding/binary"
	"fmt"
	"testing"
)

type fixture struct {
	width, height int
	mips, faces   int
	pitch         int
	bitCount      int
	pfFlags       uint32
	rm, gm, bm, a uint32
	caps2         uint32
	sample        func(face, mip, x, y int) uint32
}

func buildDDS(t *testing.T, f fixture) []byte {
	t.Helper()
	fc := f.faces
	if fc == 0 {
		fc = 1
	}
	if f.mips == 0 {
		f.mips = 1
	}
	header := make([]byte, 128)
	copy(header, Magic)
	put := func(off int, v uint32) { binary.LittleEndian.PutUint32(header[off:], v) }
	put(4, HeaderSize)
	flags := uint32(flagCaps | flagHeight | flagWidth | flagPixelFormat | flagMipMapCount | flagPitch)
	if f.pitch == 0 {
		flags &^= flagPitch
	}
	put(8, flags)
	put(12, uint32(f.height))
	put(16, uint32(f.width))
	if f.pitch != 0 {
		put(20, uint32(f.pitch))
	}
	put(28, uint32(f.mips))
	put(76, 32)
	put(80, f.pfFlags)
	put(88, uint32(f.bitCount))
	put(92, f.rm)
	put(96, f.gm)
	put(100, f.bm)
	put(104, f.a)
	put(112, f.caps2)

	sampleBytes := (f.bitCount + 7) / 8
	data := header
	for face := 0; face < fc; face++ {
		for mip := 0; mip < f.mips; mip++ {
			w, h := mipSize(f.width, f.height, mip)
			pitch := w * sampleBytes
			if mip == 0 && f.pitch > 0 {
				pitch = f.pitch
			}
			for y := 0; y < h; y++ {
				rowBytes := pitch
				if rowBytes < w*sampleBytes {
					rowBytes = w * sampleBytes
				}
				row := make([]byte, rowBytes)
				for i := range row {
					row[i] = 0xEE
				}
				for x := 0; x < w; x++ {
					raw := f.sample(face, mip, x, y)
					at := x * sampleBytes
					switch sampleBytes {
					case 1:
						row[at] = byte(raw)
					case 2:
						binary.LittleEndian.PutUint16(row[at:], uint16(raw))
					case 4:
						binary.LittleEndian.PutUint32(row[at:], raw)
					default:
						t.Fatalf("unsupported fixture sample size %d", sampleBytes)
					}
				}
				data = append(data, row...)
			}
		}
	}
	return data
}

func TestParsePitchFaceMipAndBGRA(t *testing.T) {
	raw := fixture{
		width: 4, height: 4, mips: 3, faces: 6, pitch: 20,
		bitCount: 32, pfFlags: ddspfRGB | ddspfAlphaPixels,
		rm: 0x00FF0000, gm: 0x0000FF00, bm: 0x000000FF, a: 0xFF000000,
		caps2: caps2Cubemap | cubeFaceAll,
		sample: func(face, mip, x, y int) uint32 {
			return 0xFF000000 | uint32(face*40+mip*10+x+y)<<16 | uint32(face*20+mip*5)<<8 | uint32(x+y)
		},
	}
	data := buildDDS(t, raw)
	tex, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if tex.Width != 4 || tex.Height != 4 || tex.FaceCount != 6 || tex.MipCount != 3 {
		t.Fatalf("unexpected summary: %+v", tex.Summary())
	}
	if !tex.IsCubemap || tex.Format != "BGRA8" {
		t.Fatalf("cube/format not detected: cube=%v format=%q", tex.IsCubemap, tex.Format)
	}
	for face := 0; face < 6; face++ {
		wantSizes := [][2]int{{4, 4}, {2, 2}, {1, 1}}
		for mip, want := range wantSizes {
			img := tex.Faces[face].Mips[mip]
			if img.Width != want[0] || img.Height != want[1] {
				t.Fatalf("face %d mip %d is %dx%d, want %dx%d", face, mip, img.Width, img.Height, want[0], want[1])
			}
			at := ((0*4)*img.Width + 0) * 4
			wantG := byte(face*20 + mip*5)
			wantR := byte(face*40 + mip*10)
			if img.RGBA[at] != wantR || img.RGBA[at+1] != wantG || img.RGBA[at+2] != 0 || img.RGBA[at+3] != 255 {
				t.Fatalf("face %d mip %d pixel = %v, want [%d %d 0 255]",
					face, mip, img.RGBA[at:at+4], wantR, wantG)
			}
		}
	}
	if tex.Trailing != 0 || tex.DataSize != int64(len(data)-128) {
		t.Fatalf("data accounting: offset=%d size=%d trailing=%d", tex.DataOffset, tex.DataSize, tex.Trailing)
	}
}

func TestDecodeChannelMasks(t *testing.T) {
	cases := []struct {
		name   string
		rm, gm uint32
		bm, am uint32
		raw    uint32
		want   [4]byte
	}{
		{"RGBA", 0xFF, 0xFF00, 0xFF0000, 0xFF000000, 0x281E140A, [4]byte{10, 20, 30, 40}},
		{"BGRA", 0xFF0000, 0xFF00, 0xFF, 0xFF000000, 0x281E140A, [4]byte{30, 20, 10, 40}},
		{"BGRX", 0xFF0000, 0xFF00, 0xFF, 0, 0x281E140A, [4]byte{30, 20, 10, 255}},
		{"L8", 0xFF, 0, 0, 0, 77, [4]byte{77, 77, 77, 255}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := fixture{
				width: 1, height: 1, bitCount: 32,
				pfFlags: ddspfRGB, rm: tc.rm, gm: tc.gm, bm: tc.bm, a: tc.am,
				sample: func(_, _, _, _ int) uint32 { return tc.raw },
			}
			if tc.name == "L8" {
				f.bitCount = 8
			}
			tex, err := Parse(buildDDS(t, f))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got := [4]byte{tex.Base(0).RGBA[0], tex.Base(0).RGBA[1], tex.Base(0).RGBA[2], tex.Base(0).RGBA[3]}
			if got != tc.want {
				t.Fatalf("RGBA = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTruncationReturnsNoTexture(t *testing.T) {
	f := fixture{
		width: 4, height: 4, mips: 3, faces: 1, pitch: 24,
		bitCount: 32, pfFlags: ddspfRGB | ddspfAlphaPixels,
		rm: 0xFF, gm: 0xFF00, bm: 0xFF0000, a: 0xFF000000,
		sample: func(face, mip, x, y int) uint32 { return 0xFFFFFFFF },
	}
	valid := buildDDS(t, f)
	tex, err := Parse(valid)
	if err != nil {
		t.Fatalf("valid fixture: %v", err)
	}
	if tex == nil {
		t.Fatal("valid parse returned nil texture")
	}
	short := valid[:len(valid)-3]
	tex, err = Parse(short)
	if err == nil || tex != nil {
		t.Fatalf("truncated parse returned texture=%v err=%v", tex, err)
	}
	errs := FieldErrors(err)
	if len(errs) != 1 || errs[0].Field != "data.face[0].mip[2]" {
		t.Fatalf("diagnostics = %+v", errs)
	}
}

func TestInvalidPitchAndDimensions(t *testing.T) {
	f := fixture{
		width: 4, height: 2, bitCount: 32, pfFlags: ddspfRGB,
		rm: 0xFF, gm: 0xFF00, bm: 0xFF0000, pitch: 12,
		sample: func(_, _, _, _ int) uint32 { return 0 },
	}
	_, err := Parse(buildDDS(t, f))
	errs := FieldErrors(err)
	if len(errs) == 0 || errs[0].Field != "header.dwPitchOrLinearSize" {
		t.Fatalf("small pitch diagnostics = %+v", errs)
	}

	f.pitch = 16
	data := buildDDS(t, f)
	binary.LittleEndian.PutUint32(data[16:], uint32(0xFFFFFFFF))
	if _, err := Parse(data); err == nil {
		t.Fatal("negative width accepted")
	}
}

func TestCubemapRequiresAllFaces(t *testing.T) {
	f := fixture{
		width: 1, height: 1, bitCount: 32, pfFlags: ddspfRGB,
		rm: 0xFF, gm: 0xFF00, bm: 0xFF0000,
		caps2:  caps2Cubemap | 0x400,
		sample: func(_, _, _, _ int) uint32 { return 0 },
	}
	_, err := Parse(buildDDS(t, f))
	errs := FieldErrors(err)
	if len(errs) == 0 || errs[0].Field != "header.dwCaps2" {
		t.Fatalf("cubemap diagnostics = %+v", errs)
	}
}

func TestFullMipLimit(t *testing.T) {
	f := fixture{
		width: 2, height: 2, mips: 3, bitCount: 32, pfFlags: ddspfRGB,
		rm: 0xFF, gm: 0xFF00, bm: 0xFF0000,
		sample: func(_, _, _, _ int) uint32 { return 0 },
	}
	_, err := Parse(buildDDS(t, f))
	errs := FieldErrors(err)
	if len(errs) == 0 || errs[0].Field != "header.dwMipMapCount" {
		t.Fatalf("mip diagnostics = %+v, want %s", errs, fmt.Sprint("header.dwMipMapCount"))
	}
}
