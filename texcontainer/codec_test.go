package texcontainer

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"texturelab/colorspace"
)

func TestEncodeDecodeRoundTripPreservesMarkerAndMipPixels(t *testing.T) {
	texture := &Texture{
		Format: FormatRGBA8,
		Space:  colorspace.SRGB,
		Metadata: map[string]string{
			"asset": "test-albedo",
			"z-key": "last",
			"a-key": "first",
		},
		Levels: []Level{
			{Width: 4, Height: 4, Pixels: patternedPixels(4, 4)},
			{Width: 2, Height: 2, Pixels: patternedPixels(2, 2)},
		},
	}
	if err := texture.ValidateMipChain(); err != nil {
		t.Fatal(err)
	}

	data, err := Encode(texture)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, Magic[:]) {
		t.Fatal("container does not start with TXCON magic")
	}

	decoded, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, texture) {
		t.Fatalf("round trip mismatch\ngot:  %#v\nwant: %#v", decoded, texture)
	}

	info, err := Inspect(data)
	if err != nil {
		t.Fatal(err)
	}
	if info.ColorSpace != "srgb" || info.LevelCount != 2 {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.Levels[0].AbsoluteOffset != info.PixelSectionOffset {
		t.Fatalf("level zero offset %d != pixel section %d",
			info.Levels[0].AbsoluteOffset, info.PixelSectionOffset)
	}
	if info.Levels[0].CRC32 == "" || info.Levels[0].CRC32 == "00000000" {
		t.Fatalf("missing real CRC: %+v", info.Levels[0])
	}
}

func TestDecodeDetectsPixelCorruption(t *testing.T) {
	texture := &Texture{
		Format: FormatRGBA8,
		Space:  colorspace.Linear,
		Levels: []Level{{Width: 2, Height: 2, Pixels: patternedPixels(2, 2)}},
	}
	data, err := Encode(texture)
	if err != nil {
		t.Fatal(err)
	}
	info, err := Inspect(data)
	if err != nil {
		t.Fatal(err)
	}
	offset := info.Levels[0].AbsoluteOffset
	data[offset] ^= 0xff

	_, err = Decode(data)
	if err == nil || !strings.Contains(err.Error(), "CRC mismatch") {
		t.Fatalf("expected CRC mismatch, got %v", err)
	}
}

func TestValidateMipChainRejectsSkippedHalfSize(t *testing.T) {
	texture := &Texture{
		Format: FormatRGBA8,
		Space:  colorspace.Linear,
		Levels: []Level{
			{Width: 8, Height: 8, Pixels: patternedPixels(8, 8)},
			{Width: 2, Height: 2, Pixels: patternedPixels(2, 2)},
		},
	}
	err := texture.ValidateMipChain()
	if err == nil || !strings.Contains(err.Error(), "want 4x4") {
		t.Fatalf("expected half-size validation error, got %v", err)
	}
}

func patternedPixels(width, height int) []byte {
	pixels := make([]byte, width*height*BytesPerPixel)
	for i := 0; i < width*height; i++ {
		pixels[i*4+0] = byte((i*17 + 3) % 256)
		pixels[i*4+1] = byte((i*31 + 7) % 256)
		pixels[i*4+2] = byte((i*47 + 11) % 256)
		pixels[i*4+3] = 255
	}
	return pixels
}
