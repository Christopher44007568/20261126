package pngx

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestEncodeDecodesWithStdlib(t *testing.T) {
	w, h := 8, 5
	rgb := make([]byte, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 3
			rgb[i] = uint8(x * 30)
			rgb[i+1] = uint8(y * 50)
			rgb[i+2] = 128
		}
	}
	meta := SRGBMeta([]TextEntry{{Keyword: "Source", Text: "test"}})
	var buf bytes.Buffer
	if err := Encode(&buf, rgb, w, h, 3, meta); err != nil {
		t.Fatalf("encode: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("stdlib png.Decode rejected output: %v", err)
	}
	if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
		t.Fatalf("decoded bounds = %v", img.Bounds())
	}
}

func TestMetadataChunksPresent(t *testing.T) {
	meta := SRGBMeta(nil)
	var buf bytes.Buffer
	if err := Encode(&buf, make([]byte, 4*1*3), 4, 1, 3, meta); err != nil {
		t.Fatalf("encode: %v", err)
	}
	raw := buf.Bytes()
	want := map[string]bool{"IHDR": false, "sRGB": false, "gAMA": false,
		"cHRM": false, "IDAT": false, "IEND": false}
	pos := 8
	for pos+12 <= len(raw) {
		n := int(binary.BigEndian.Uint32(raw[pos:]))
		typ := string(raw[pos+4 : pos+8])
		if _, ok := want[typ]; ok {
			want[typ] = true
		}
		pos += 12 + n
	}
	for typ, found := range want {
		if !found {
			t.Fatalf("chunk %s missing", typ)
		}
	}
}
