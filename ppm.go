package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Gray8 is an 8-bit grayscale raster.
type Gray8 struct {
	W, H int
	Pix  []uint8
}

// RGB8 is an 8-bit per channel RGB raster.
type RGB8 struct {
	W, H int
	Pix  []uint8
}

func NewGray8(w, h int) *Gray8 {
	return &Gray8{W: w, H: h, Pix: make([]uint8, w*h)}
}

func NewRGB8(w, h int) *RGB8 {
	return &RGB8{W: w, H: h, Pix: make([]uint8, 3*w*h)}
}

func (g *Gray8) At(x, y int) uint8 { return g.Pix[y*g.W+x] }

func (r *RGB8) Set(x, y int, cr, cg, cb uint8) {
	i := 3 * (y*r.W + x)
	r.Pix[i], r.Pix[i+1], r.Pix[i+2] = cr, cg, cb
}

// WritePPM saves an RGB image as binary PPM (P6).
func (r *RGB8) WritePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	header := fmt.Sprintf("P6\n%d %d\n255\n", r.W, r.H)
	if _, err := bw.WriteString(header); err != nil {
		return err
	}
	if _, err := bw.Write(r.Pix); err != nil {
		return err
	}
	return bw.Flush()
}

// ReadPPM parses P6 (binary) and P3 (ASCII) PPM files.
func ReadPPM(path string) (*RGB8, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	br := bufio.NewReader(f)

	magic, err := br.ReadString('\n')
	if err != nil || (magic != "P6\n" && magic != "P3\n") {
		return nil, fmt.Errorf("unsupported PPM magic: %q", magic)
	}
	w, h, maxv, err := readPPMHeader(br)
	if err != nil {
		return nil, err
	}
	img := NewRGB8(w, h)
	if magic == "P6\n" {
		if maxv != 255 {
			return nil, fmt.Errorf("only maxval 255 supported, got %d", maxv)
		}
		if _, err := io.ReadFull(br, img.Pix); err != nil {
			return nil, err
		}
		return img, nil
	}
	for i := range img.Pix {
		var v int
		if _, err := fmt.Fscan(br, &v); err != nil {
			return nil, err
		}
		img.Pix[i] = uint8(v)
	}
	return img, nil
}

// readPPMHeader reads width, height and maxval while honoring '#' comments.
// It returns with the reader positioned exactly one byte past the single
// whitespace token following maxval, so P6 binary data stays byte-aligned.
func readPPMHeader(br *bufio.Reader) (int, int, int, error) {
	var fields [3]int
	n := 0
	for n < 3 {
		if err := skipWhitespaceAndComments(br); err != nil {
			return 0, 0, 0, err
		}
		token, err := readToken(br)
		if err != nil {
			return 0, 0, 0, err
		}
		v, err := strconv.Atoi(token)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid header token %q: %w", token, err)
		}
		fields[n] = v
		n++
	}
	return fields[0], fields[1], fields[2], nil
}

func skipWhitespaceAndComments(br *bufio.Reader) error {
	for {
		c, err := br.ReadByte()
		if err != nil {
			return err
		}
		if c == '#' {
			for c != '\n' {
				c, err = br.ReadByte()
				if err != nil {
					return err
				}
			}
			continue
		}
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return br.UnreadByte()
		}
	}
}

func readToken(br *bufio.Reader) (string, error) {
	var buf []byte
	for {
		c, err := br.ReadByte()
		if err != nil {
			if err == io.EOF && len(buf) > 0 {
				return string(buf), nil
			}
			return "", err
		}
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			break
		}
		buf = append(buf, c)
	}
	return string(buf), nil
}
