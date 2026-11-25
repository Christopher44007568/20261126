// Package pfm reads and writes Portable Float Map files (the floating-point
// member of the Netpbm/PPM family, magic PF for color, Pf for grayscale).
// PFM stores raw little/big-endian IEEE-754 float32 scanlines with no gamma
// or chromaticity metadata, which is exactly why the companion PNG carries
// explicit color metadata.
package pfm

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"hdrexport/internal/hdr"
)

// Encode writes a color PFM (magic "PF") in little-endian byte order.
// The PFM scanline order is bottom-up, so rows are emitted reversed.
func Encode(w io.Writer, m *hdr.Image) error {
	bw := bufio.NewWriter(w)
	header := fmt.Sprintf("PF\n%d %d\n%g\n", m.W, m.H, -1.0)
	if _, err := bw.WriteString(header); err != nil {
		return err
	}
	row := make([]byte, m.W*3*4)
	for y := m.H - 1; y >= 0; y-- {
		off := 0
		for x := 0; x < m.W; x++ {
			i := (y*m.W + x) * 3
			for c := 0; c < 3; c++ {
				binary.LittleEndian.PutUint32(row[off:], math.Float32bits(m.Pix[i+c]))
				off += 4
			}
		}
		if _, err := bw.Write(row); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// Decode reads a PF or Pf file. Comments (# lines) and extra whitespace are
// tolerated. A negative scale means little-endian, positive big-endian; the
// scale magnitude itself is ignored (values are used as-is).
func Decode(r io.Reader) (*hdr.Image, error) {
	br := bufio.NewReader(r)
	magic, err := readToken(br)
	if err != nil {
		return nil, err
	}
	channels := 0
	switch string(magic) {
	case "PF":
		channels = 3
	case "Pf":
		channels = 1
	default:
		return nil, fmt.Errorf("pfm: unsupported magic %q", magic)
	}
	width, err := readIntToken(br)
	if err != nil {
		return nil, err
	}
	height, err := readIntToken(br)
	if err != nil {
		return nil, err
	}
	scale, err := readFloatToken(br)
	if err != nil {
		return nil, err
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("pfm: invalid dimensions %dx%d", width, height)
	}

	var order binary.ByteOrder = binary.LittleEndian
	if scale > 0 {
		order = binary.BigEndian
	}

	m := hdr.NewImage(width, height)
	rowBytes := width * channels * 4
	buf := make([]byte, rowBytes)
	// PFM stores the bottom row first; flip while reading.
	for y := height - 1; y >= 0; y-- {
		if _, err := io.ReadFull(br, buf); err != nil {
			return nil, fmt.Errorf("pfm: truncated pixel data: %w", err)
		}
		for x := 0; x < width; x++ {
			var rf, gf, bf float32
			rf = math.Float32frombits(order.Uint32(buf[x*channels*4:]))
			if channels == 3 {
				gf = math.Float32frombits(order.Uint32(buf[x*channels*4+4:]))
				bf = math.Float32frombits(order.Uint32(buf[x*channels*4+8:]))
			} else {
				gf, bf = rf, rf
			}
			m.Set(x, y, rf, gf, bf)
		}
	}
	return m, nil
}

func skipWhitespaceAndComments(br *bufio.Reader) error {
	for {
		b, err := br.ReadByte()
		if err != nil {
			return err
		}
		switch {
		case b == '#':
			if _, err := br.ReadBytes('\n'); err != nil {
				return err
			}
		case b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == 0x0b || b == 0x0c:
			continue
		default:
			return br.UnreadByte()
		}
	}
}

func readToken(br *bufio.Reader) ([]byte, error) {
	if err := skipWhitespaceAndComments(br); err != nil {
		return nil, err
	}
	var tok bytes.Buffer
	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			break
		}
		tok.WriteByte(b)
	}
	return tok.Bytes(), nil
}

func readIntToken(br *bufio.Reader) (int, error) {
	tok, err := readToken(br)
	if err != nil {
		return 0, err
	}
	var v int
	if _, err := fmt.Sscanf(string(tok), "%d", &v); err != nil {
		return 0, fmt.Errorf("pfm: bad integer %q", tok)
	}
	return v, nil
}

func readFloatToken(br *bufio.Reader) (float64, error) {
	tok, err := readToken(br)
	if err != nil {
		return 0, err
	}
	var v float64
	if _, err := fmt.Sscanf(string(tok), "%g", &v); err != nil {
		return 0, fmt.Errorf("pfm: bad scale %q", tok)
	}
	return v, nil
}
