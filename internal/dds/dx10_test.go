package dds

import (
	"encoding/binary"
	"testing"
)

func buildDX10Fixture(t *testing.T, dxgi uint32, cube bool) []byte {
	t.Helper()
	header := make([]byte, 128+DX10HeaderSize)
	copy(header, Magic)
	put := func(off int, v uint32) { binary.LittleEndian.PutUint32(header[off:], v) }
	put(4, HeaderSize)
	put(8, flagCaps|flagHeight|flagWidth|flagPixelFormat|flagMipMapCount|flagPitch)
	put(12, 2)
	put(16, 2)
	put(20, 8)
	put(28, 2)
	put(76, 32)
	put(80, ddspfFourCC)
	copy(header[84:88], "DX10")
	put(112, 0)
	if cube {
		put(112, caps2Cubemap|cubeFaceAll)
	}
	put(128, dxgi)
	put(132, dimensionTexture2)
	if cube {
		put(136, dx10ResourceCube)
	}
	put(140, 1)

	faces := 1
	if cube {
		faces = 6
	}
	data := header
	for face := 0; face < faces; face++ {
		base := byte(face*30 + 10)
		data = append(data,
			base, 20, 30, 40,
			base+1, 22, 33, 44,
			base+2, 24, 36, 48,
			base+3, 26, 39, 52,
		)
		data = append(data, base+4, 40, 80, 255)
	}
	return data
}

func TestDX10FormatsAndMips(t *testing.T) {
	cases := []struct {
		dxgi uint32
		want [4]byte
	}{
		{28, [4]byte{10, 20, 30, 40}},
		{87, [4]byte{30, 20, 10, 40}},
		{88, [4]byte{30, 20, 10, 255}},
	}
	for _, tc := range cases {
		tex, err := Parse(buildDX10Fixture(t, tc.dxgi, false))
		if err != nil {
			t.Fatalf("DXGI %d: %v", tc.dxgi, err)
		}
		if tex.DataOffset != 148 || tex.FaceCount != 1 || tex.MipCount != 2 {
			t.Fatalf("DXGI %d summary: %+v", tc.dxgi, tex.Summary())
		}
		got := [4]byte{tex.Base(0).RGBA[0], tex.Base(0).RGBA[1], tex.Base(0).RGBA[2], tex.Base(0).RGBA[3]}
		if got != tc.want {
			t.Fatalf("DXGI %d pixel = %v, want %v", tc.dxgi, got, tc.want)
		}
	}
}

func TestDX10CubemapAndUnsupportedFormat(t *testing.T) {
	tex, err := Parse(buildDX10Fixture(t, 28, true))
	if err != nil {
		t.Fatalf("DX10 cubemap: %v", err)
	}
	if !tex.IsCubemap || tex.FaceCount != 6 || tex.MipCount != 2 {
		t.Fatalf("DX10 cubemap summary: %+v", tex.Summary())
	}

	data := buildDX10Fixture(t, 98, false) // BC7 is outside the subset.
	_, err = Parse(data)
	errs := FieldErrors(err)
	if len(errs) != 1 || errs[0].Field != "dx10.dxgiFormat" {
		t.Fatalf("unsupported DXGI diagnostics = %+v", errs)
	}
}

func TestLinearSizeDerivesRowPitch(t *testing.T) {
	f := fixture{
		width: 4, height: 4, bitCount: 32, pfFlags: ddspfRGB,
		rm: 0xFF, gm: 0xFF00, bm: 0xFF0000,
		sample: func(_, _, x, y int) uint32 { return uint32(y*4 + x) },
	}
	data := buildDDS(t, f)
	binary.LittleEndian.PutUint32(data[8:], flagCaps|flagHeight|flagWidth|flagPixelFormat|flagLinearSize)
	binary.LittleEndian.PutUint32(data[20:], 64)
	tex, err := Parse(data)
	if err != nil {
		t.Fatalf("linear-size fixture: %v", err)
	}
	if tex.DataSize != 64 || tex.Format != "RGBX8" {
		t.Fatalf("linear-size result: %+v", tex.Summary())
	}
}
