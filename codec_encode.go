package main

import "encoding/binary"

type channelFit struct {
	baseQ uint8
	spanQ uint8
	codes [BlockPixels]uint8
}

func EncodeBlock(block Block) EncodedBlock {
	mode := blockAlphaMode(block)
	out := make([]byte, 0, 25)
	out = append(out, byte(codecVersion)|byte(mode))
	if mode == alphaEmpty {
		return EncodedBlock{Data: out}
	}

	var values, weights [3][BlockPixels]uint8
	for i, p := range block {
		values[0][i], values[1][i], values[2][i] = p.R, p.G, p.B
		if p.A > 0 {
			weights[0][i], weights[1][i], weights[2][i] = 1, 1, 1
		}
	}
	rf := fitChannel(values[0], weights[0], 31)
	gf := fitChannel(values[1], weights[1], 63)
	bf := fitChannel(values[2], weights[2], 31)

	base := uint16(rf.baseQ)<<11 | uint16(gf.baseQ)<<5 | uint16(bf.baseQ)
	out = append(out, byte(base>>8), byte(base))
	var spans bitWriter
	spans.write(uint(rf.spanQ), 5)
	spans.write(uint(gf.spanQ), 5)
	spans.write(uint(bf.spanQ), 5)
	out = append(out, spans.buf...)
	var codes bitWriter
	for i := 0; i < BlockPixels; i++ {
		code := uint(rf.codes[i]) | uint(gf.codes[i])<<2 | uint(bf.codes[i])<<4
		codes.write(code, 6)
	}
	out = append(out, codes.buf...)

	switch mode {
	case alphaMask:
		var mask uint16
		for i, p := range block {
			if p.A > 0 {
				mask |= 1 << i
			}
		}
		var raw [2]byte
		binary.LittleEndian.PutUint16(raw[:], mask)
		out = append(out, raw[:]...)
	case alphaNibble:
		for i := 0; i < BlockPixels; i += 2 {
			a0 := alphaQuant(block[i].A)
			a1 := alphaQuant(block[i+1].A)
			out = append(out, a0|a1<<4)
		}
	}
	return EncodedBlock{Data: out}
}

func blockAlphaMode(block Block) int {
	allEmpty, allOpaque, binary := true, true, true
	for _, p := range block {
		if p.A != 0 {
			allEmpty = false
		}
		if p.A != 255 {
			allOpaque = false
		}
		if p.A != 0 && p.A != 255 {
			binary = false
		}
	}
	switch {
	case allEmpty:
		return alphaEmpty
	case allOpaque:
		return alphaOpaque
	case binary:
		return alphaMask
	default:
		return alphaNibble
	}
}

func fitChannel(values, weights [BlockPixels]uint8, maxLevel int) channelFit {
	var best channelFit
	bestCost := ^uint64(0)
	for baseQ := 0; baseQ <= maxLevel; baseQ++ {
		var base int
		if maxLevel == 31 {
			base = int(expand5(uint8(baseQ)))
		} else {
			base = int(expand6(uint8(baseQ)))
		}
		for spanQ := 0; spanQ <= 31; spanQ++ {
			span := spanFromQ(uint(spanQ))
			var levels [4]int
			for code := range levels {
				levels[code] = base + increment(code, span)
			}
			var cost uint64
			var codes [BlockPixels]uint8
			for i, value := range values {
				if weights[i] == 0 {
					continue
				}
				diff := func(a, b int) int {
					if a > b {
						return a - b
					}
					return b - a
				}
				chosen, least := 0, 1<<30
				for code, level := range levels {
					d := diff(level, int(value))
					if d < least {
						chosen, least = code, d
					}
				}
				codes[i] = uint8(chosen)
				cost += uint64(least * least)
			}
			if cost < bestCost {
				bestCost = cost
				best = channelFit{baseQ: uint8(baseQ), spanQ: uint8(spanQ), codes: codes}
			}
		}
	}
	return best
}

func alphaQuant(a uint8) uint8 {
	q := uint8((uint16(a)*15 + 127) / 255)
	if a > 0 && q == 0 {
		q = 1
	}
	if a < 255 && q == 15 {
		q = 14
	}
	return q
}
