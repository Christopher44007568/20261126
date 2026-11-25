package dds

import (
	"encoding/binary"
	"fmt"
)

// Parse decodes a complete DDS subset file. It validates every face, mip and
// row range before decoding, so a truncated file never yields a Texture.
func Parse(data []byte) (*Texture, error) {
	h, derr := parseHeader(data)
	if derr != nil {
		return nil, derr
	}

	derr = &DecodeError{}
	totalData := int64(0)
	for fi := 0; fi < h.faceCount; fi++ {
		for mi := 0; mi < h.mipCount; mi++ {
			w, ht, pitch, bytes, ferr := h.levelLayout(mi)
			if ferr != nil {
				derr.Errors = append(derr.Errors, *ferr)
				return nil, derr
			}
			start := h.dataOffset + totalData
			end := start + int64(bytes)
			if end > int64(len(data)) {
				derr.addAt(fmt.Sprintf("data.face[%d].mip[%d]", fi, mi), start,
					"texture data is truncated; the last valid texture must be kept",
					fmt.Sprintf("complete %dx%d mip with pitch %d, ending at byte %d", w, ht, pitch, end),
					fmt.Sprintf("file ends at byte %d", len(data)))
				return nil, derr
			}
			totalData += int64(bytes)
		}
	}

	t := &Texture{
		Width: h.width, Height: h.height, MipCount: h.mipCount,
		FaceCount: h.faceCount, IsCubemap: h.isCubemap,
		Format: describeMasks(h.format), PixelFormat: h.format,
		DataOffset: h.dataOffset, DataSize: totalData,
	}
	t.Trailing = int64(len(data)) - h.dataOffset - totalData
	if h.format.Name == "" {
		h.format.Name = t.Format
		t.PixelFormat.Name = t.Format
	}
	if h.format.Name != "" {
		t.Format = h.format.Name
	}

	var offset int64
	for fi := 0; fi < h.faceCount; fi++ {
		face := Face{}
		for mi := 0; mi < h.mipCount; mi++ {
			w, ht, pitch, bytes, _ := h.levelLayout(mi)
			img, err := decodeMip(data[h.dataOffset+offset:h.dataOffset+offset+int64(bytes)], w, ht, pitch, h)
			if err != nil {
				return nil, err
			}
			face.Mips = append(face.Mips, img)
			offset += int64(bytes)
		}
		t.Faces = append(t.Faces, face)
	}
	return t, nil
}

func decodeMip(rowData []byte, width, height, pitch int, h *header) (Image, error) {
	img := Image{Width: width, Height: height, RGBA: make([]byte, width*height*4)}
	sourceBytes := sampleBytes(h.bitCount)
	for y := 0; y < height; y++ {
		row := rowData[y*pitch : y*pitch+width*sourceBytes]
		for x := 0; x < width; x++ {
			var sample uint32
			at := x * sourceBytes
			switch sourceBytes {
			case 1:
				sample = uint32(row[at])
			case 2:
				sample = uint32(binary.LittleEndian.Uint16(row[at : at+2]))
			case 3:
				sample = uint32(row[at]) | uint32(row[at+1])<<8 | uint32(row[at+2])<<16
			case 4:
				sample = binary.LittleEndian.Uint32(row[at : at+4])
			default:
				return Image{}, &DecodeError{Errors: []FieldError{{
					Field: "header.ddspf.dwRGBBitCount", Offset: 88,
					Message: fmt.Sprintf("cannot decode %d-bit samples", h.bitCount),
				}}}
			}
			r, g, b, a := expandSample(sample, h)
			at = (y*width + x) * 4
			img.RGBA[at], img.RGBA[at+1], img.RGBA[at+2], img.RGBA[at+3] = r, g, b, a
		}
	}
	return img, nil
}

func expandSample(sample uint32, h *header) (byte, byte, byte, byte) {
	r := bitsToByte(sample, h.rMask)
	g := bitsToByte(sample, h.gMask)
	b := bitsToByte(sample, h.bMask)
	a := byte(255)
	if h.aMask != 0 {
		a = bitsToByte(sample, h.aMask)
	}

	hasColor := h.rMask != 0 || h.gMask != 0 || h.bMask != 0
	switch {
	case !hasColor:
		r, g, b = 255, 255, 255
	case h.rMask != 0 && h.gMask == 0 && h.bMask == 0:
		g, b = r, r // luminance/R-only texture
	case h.rMask == 0 && h.gMask != 0 && h.bMask == 0:
		r, b = g, g
	case h.rMask == 0 && h.gMask == 0 && h.bMask != 0:
		r, g = b, b
	}
	return r, g, b, a
}

// FieldErrors returns structured diagnostics from err when err came from
// Parse, or nil for unrelated I/O and programming errors.
func FieldErrors(err error) []FieldError {
	if de, ok := err.(*DecodeError); ok {
		return de.Errors
	}
	return nil
}
