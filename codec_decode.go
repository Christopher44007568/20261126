package main

import (
	"encoding/binary"
	"fmt"
)

func DecodeBlock(data []byte) (Block, error) {
	var block Block
	if len(data) == 0 {
		return block, fmt.Errorf("empty block record")
	}
	mode := int(data[0] & 3)
	want, err := RecordSize(data[0])
	if err != nil {
		return block, err
	}
	if len(data) != want {
		return block, fmt.Errorf("block record has %d bytes, want %d", len(data), want)
	}
	if mode == alphaEmpty {
		return block, nil
	}

	base := binary.BigEndian.Uint16(data[1:3])
	baseR := int(expand5(uint8(base >> 11 & 0x1F)))
	baseG := int(expand6(uint8(base >> 5 & 0x3F)))
	baseB := int(expand5(uint8(base & 0x1F)))
	spans := newBitReader(data[3:5])
	qr, okR := spans.read(5)
	qg, okG := spans.read(5)
	qb, okB := spans.read(5)
	if !okR || !okG || !okB {
		return block, fmt.Errorf("truncated span fields")
	}
	spanR, spanG, spanB := spanFromQ(qr), spanFromQ(qg), spanFromQ(qb)

	codes := newBitReader(data[5:17])
	for i := range block {
		code, ok := codes.read(6)
		if !ok {
			return block, fmt.Errorf("truncated delta codes")
		}
		block[i].R = clamp8(baseR + increment(int(code&3), spanR))
		block[i].G = clamp8(baseG + increment(int(code>>2&3), spanG))
		block[i].B = clamp8(baseB + increment(int(code>>4&3), spanB))
		block[i].A = 255
	}

	switch mode {
	case alphaOpaque:
	case alphaMask:
		mask := binary.LittleEndian.Uint16(data[17:19])
		for i := range block {
			if mask&(1<<i) == 0 {
				block[i].A = 0
			}
		}
	case alphaNibble:
		for i := range block {
			q := data[17+i/2] >> (4 * (i & 1)) & 0x0F
			block[i].A = uint8((uint16(q)*255 + 7) / 15)
		}
	default:
		return block, fmt.Errorf("unsupported alpha mode %d", mode)
	}
	return block, nil
}
