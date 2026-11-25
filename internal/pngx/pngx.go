// Package pngx is a small dependency-free PNG encoder for 8-bit RGB/RGBA
// images. Unlike image/png in the standard library it emits explicit color
// metadata chunks (sRGB, gAMA, cHRM) plus tEXt conversion notes, so viewers
// know the pixels are already sRGB-encoded and must not re-apply gamma.
package pngx

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io"
)

const pngSignature = "\x89PNG\r\n\x1a\n"

// TextEntry is a tEXt chunk: keyword plus Latin-1/ASCII text.
type TextEntry struct {
	Keyword string
	Text    string
}

// Meta carries the color metadata written alongside the pixels.
type Meta struct {
	// Gamma is the display transfer exponent expressed as image gamma
	// (1/encoding exponent); 1/2.2 -> 0.45455 for sRGB.
	Gamma uint32
	// Chromaticities in PNG fixed-point (CIE xy * 100000).
	WhiteX, WhiteY, RedX, RedY, GreenX, GreenY, BlueX, BlueY uint32
	// Rendering intent of the sRGB chunk (0 = perceptual).
	SRGBIntent byte
	Text       []TextEntry
}

// SRGBMeta returns the standard sRGB metadata block plus conversion notes.
func SRGBMeta(notes []TextEntry) Meta {
	return Meta{
		Gamma:  45455,
		WhiteX: 31270, WhiteY: 32900,
		RedX: 64000, RedY: 33000,
		GreenX: 30000, GreenY: 60000,
		BlueX: 15000, BlueY: 6000,
		SRGBIntent: 0,
		Text:       notes,
	}
}

type chunkWriter struct {
	w io.Writer
}

func (c *chunkWriter) write(typ string, data []byte) error {
	if _, err := c.w.Write(uint32Bytes(uint32(len(data)))); err != nil {
		return err
	}
	if _, err := c.w.Write([]byte(typ)); err != nil {
		return err
	}
	if _, err := c.w.Write(data); err != nil {
		return err
	}
	h := crc32.NewIEEE()
	h.Write([]byte(typ))
	h.Write(data)
	_, err := c.w.Write(uint32Bytes(h.Sum32()))
	return err
}

func uint32Bytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// Encode writes an 8-bit PNG. rgb must have w*h*channels bytes with channels
// 3 (RGB) or 4 (RGBA), rows top to bottom.
func Encode(w io.Writer, rgb []byte, width, height, channels int, meta Meta) error {
	if channels != 3 && channels != 4 {
		panic("pngx: only 8-bit RGB and RGBA are supported")
	}
	cw := &chunkWriter{w: w}
	if _, err := io.WriteString(w, pngSignature); err != nil {
		return err
	}

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(height))
	ihdr[8] = 8
	if channels == 4 {
		ihdr[9] = 6
	} else {
		ihdr[9] = 2
	}
	if err := cw.write("IHDR", ihdr); err != nil {
		return err
	}

	srgb := []byte{meta.SRGBIntent}
	if err := cw.write("sRGB", srgb); err != nil {
		return err
	}
	if err := cw.write("gAMA", uint32Bytes(meta.Gamma)); err != nil {
		return err
	}
	chrm := make([]byte, 32)
	vals := []uint32{meta.WhiteX, meta.WhiteY, meta.RedX, meta.RedY,
		meta.GreenX, meta.GreenY, meta.BlueX, meta.BlueY}
	for i, v := range vals {
		binary.BigEndian.PutUint32(chrm[i*4:], v)
	}
	if err := cw.write("cHRM", chrm); err != nil {
		return err
	}
	for _, t := range meta.Text {
		var buf bytes.Buffer
		buf.WriteString(t.Keyword)
		buf.WriteByte(0)
		buf.WriteString(t.Text)
		if err := cw.write("tEXt", buf.Bytes()); err != nil {
			return err
		}
	}

	filtered := filterScanlines(rgb, width, height, channels)
	var zbuf bytes.Buffer
	zw := zlib.NewWriter(&zbuf)
	if _, err := zw.Write(filtered); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := cw.write("IDAT", zbuf.Bytes()); err != nil {
		return err
	}
	return cw.write("IEND", nil)
}

// filterScanlines applies per-row adaptive filtering (none/sub/up/avg/paeth),
// picking the filter with the smallest sum of absolute byte differences.
func filterScanlines(rgb []byte, w, h, bpp int) []byte {
	rowLen := w * bpp
	out := make([]byte, 0, h*(rowLen+1))
	zero := make([]byte, rowLen)
	prev := zero
	for y := 0; y < h; y++ {
		cur := rgb[y*rowLen : (y+1)*rowLen]
		candidates := make([][]byte, 5)
		for f := 0; f < 5; f++ {
			candidates[f] = applyFilter(byte(f), cur, prev, bpp)
		}
		best := 0
		bestScore := score(candidates[0])
		for f := 1; f < 5; f++ {
			if s := score(candidates[f]); s < bestScore {
				best, bestScore = f, s
			}
		}
		out = append(out, byte(best))
		out = append(out, candidates[best]...)
		prev = cur
	}
	return out
}

func score(b []byte) int {
	s := 0
	for _, v := range b {
		// signed magnitude so filters centred near zero win
		if iv := int(v); iv > 127 {
			s += 256 - iv
		} else {
			s += int(v)
		}
	}
	return s
}

func applyFilter(f byte, cur, prev []byte, bpp int) []byte {
	n := len(cur)
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		x := cur[i]
		var a, b, c byte
		if i >= bpp {
			a = cur[i-bpp]
		}
		if prev != nil {
			b = prev[i]
			if i >= bpp {
				c = prev[i-bpp]
			}
		}
		switch f {
		case 0:
			out[i] = x
		case 1:
			out[i] = x - a
		case 2:
			out[i] = x - b
		case 3:
			out[i] = x - byte((int(a)+int(b))/2)
		case 4:
			out[i] = x - paeth(a, b, c)
		}
	}
	return out
}

func paeth(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa, pb, pc := abs(p-int(a)), abs(p-int(b)), abs(p-int(c))
	switch {
	case pa <= pb && pa <= pc:
		return a
	case pb <= pc:
		return b
	default:
		return c
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
