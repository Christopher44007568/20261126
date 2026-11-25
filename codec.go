package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	BlockSide     = 4
	BlockPixels   = BlockSide * BlockSide
	codecVersion  = 0xD0
	codecVersionM = 0xFC
)

const (
	alphaOpaque = iota
	alphaMask
	alphaNibble
	alphaEmpty
)

type Block [BlockPixels]Pixel

// EncodedBlock is one self-describing 4x4 record. The first byte contains the
// codec version and alpha mode, so records in a stream need no side table.
type EncodedBlock struct {
	Data []byte
}

// BlockInfo exposes the fields needed by the textual report without making
// the report depend on bit-level implementation details.
type BlockInfo struct {
	AlphaMode int
	Base      Pixel
	SpanR     int
	SpanG     int
	SpanB     int
	AlphaMask uint16
}

// RecordSize returns the number of bytes occupied by a record with header h.
func RecordSize(h byte) (int, error) {
	if h&codecVersionM != codecVersion {
		return 0, fmt.Errorf("bad block header %#02x", h)
	}
	switch int(h & 3) {
	case alphaOpaque:
		return 17, nil
	case alphaMask:
		return 19, nil
	case alphaNibble:
		return 25, nil
	case alphaEmpty:
		return 1, nil
	default:
		return 0, errors.New("unreachable alpha mode")
	}
}

func (e EncodedBlock) Size() int { return len(e.Data) }

func (e EncodedBlock) Info() (BlockInfo, error) {
	var info BlockInfo
	if len(e.Data) == 0 {
		return info, errors.New("empty block record")
	}
	info.AlphaMode = int(e.Data[0] & 3)
	if info.AlphaMode == alphaEmpty {
		if len(e.Data) != 1 {
			return info, errors.New("trailing bytes after empty block")
		}
		return info, nil
	}
	want, err := RecordSize(e.Data[0])
	if err != nil {
		return info, err
	}
	if len(e.Data) != want {
		return info, fmt.Errorf("block record has %d bytes, want %d", len(e.Data), want)
	}

	base := binary.BigEndian.Uint16(e.Data[1:3])
	info.Base = Pixel{
		R: expand5(uint8(base >> 11 & 0x1F)),
		G: expand6(uint8(base >> 5 & 0x3F)),
		B: expand5(uint8(base & 0x1F)),
		A: 255,
	}
	br := newBitReader(e.Data[3:5])
	qr, _ := br.read(5)
	qg, _ := br.read(5)
	qb, _ := br.read(5)
	info.SpanR = spanFromQ(qr)
	info.SpanG = spanFromQ(qg)
	info.SpanB = spanFromQ(qb)
	if info.AlphaMode == alphaMask {
		info.AlphaMask = binary.LittleEndian.Uint16(e.Data[17:19])
	}
	return info, nil
}

type bitWriter struct {
	buf []byte
	pos uint
}

func (w *bitWriter) write(v, n uint) {
	for i := uint(0); i < n; i++ {
		bit := (v >> i) & 1
		if w.pos%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		w.buf[len(w.buf)-1] |= byte(bit << (w.pos % 8))
		w.pos++
	}
}

type bitReader struct {
	buf []byte
	pos uint
}

func newBitReader(buf []byte) *bitReader { return &bitReader{buf: buf} }

func (r *bitReader) read(n uint) (uint, bool) {
	if r.pos+n > uint(len(r.buf)*8) {
		return 0, false
	}
	var v uint
	for i := uint(0); i < n; i++ {
		v |= uint(r.buf[r.pos/8]>>(r.pos%8)&1) << i
		r.pos++
	}
	return v, true
}

func expand5(v uint8) uint8 { return uint8((uint16(v)*255 + 15) / 31) }
func expand6(v uint8) uint8 { return uint8((uint16(v)*255 + 31) / 63) }

func spanFromQ(q uint) int { return int((q*255 + 15) / 31) }

func increment(code, span int) int {
	if code >= 3 {
		return span
	}
	return (code*span + 1) / 3
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
