package main

import (
	"encoding/binary"
	"fmt"
)

const streamMagic = "BD4X"

// CompressBitmap writes a self-contained stream: 8-byte image header followed
// by row-major 4x4 records.
func CompressBitmap(im *Bitmap) ([]byte, []EncodedBlock, error) {
	if im.W <= 0 || im.H <= 0 || im.W%BlockSide != 0 || im.H%BlockSide != 0 {
		return nil, nil, fmt.Errorf("bitmap %dx%d is not divisible by %d", im.W, im.H, BlockSide)
	}
	if im.W > 0xFFFF || im.H > 0xFFFF {
		return nil, nil, fmt.Errorf("bitmap %dx%d exceeds the 16-bit stream dimensions", im.W, im.H)
	}
	out := make([]byte, 8, 8+im.W*im.H)
	copy(out, streamMagic)
	binary.BigEndian.PutUint16(out[4:6], uint16(im.W))
	binary.BigEndian.PutUint16(out[6:8], uint16(im.H))

	records := make([]EncodedBlock, 0, (im.W/BlockSide)*(im.H/BlockSide))
	for y := 0; y < im.H; y += BlockSide {
		for x := 0; x < im.W; x += BlockSide {
			rec := EncodeBlock(im.Block(x, y))
			records = append(records, rec)
			out = append(out, rec.Data...)
		}
	}
	return out, records, nil
}

func DecompressBitmap(data []byte) (*Bitmap, error) {
	if len(data) < 8 || string(data[:4]) != streamMagic {
		return nil, fmt.Errorf("not a BD4X stream")
	}
	w := int(binary.BigEndian.Uint16(data[4:6]))
	h := int(binary.BigEndian.Uint16(data[6:8]))
	if w <= 0 || h <= 0 || w%BlockSide != 0 || h%BlockSide != 0 {
		return nil, fmt.Errorf("invalid dimensions %dx%d", w, h)
	}
	im := NewBitmap(w, h)
	offset := 8
	for by := 0; by < h; by += BlockSide {
		for bx := 0; bx < w; bx += BlockSide {
			if offset >= len(data) {
				return nil, fmt.Errorf("stream ends before block (%d,%d)", bx, by)
			}
			size, err := RecordSize(data[offset])
			if err != nil {
				return nil, fmt.Errorf("block (%d,%d): %w", bx, by, err)
			}
			if offset+size > len(data) {
				return nil, fmt.Errorf("block (%d,%d) is truncated", bx, by)
			}
			block, err := DecodeBlock(data[offset : offset+size])
			if err != nil {
				return nil, fmt.Errorf("block (%d,%d): %w", bx, by, err)
			}
			im.SetBlock(bx, by, block)
			offset += size
		}
	}
	if offset != len(data) {
		return nil, fmt.Errorf("%d trailing bytes in stream", len(data)-offset)
	}
	return im, nil
}
