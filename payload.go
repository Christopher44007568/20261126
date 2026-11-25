package main

// lcg 是确定性线性同余随机源，使“远端”内容可复现且不依赖 math/rand。
type lcg struct {
	state uint32
}

func newLCG(seed uint32) *lcg {
	return &lcg{state: seed*2654435761 + 2463534242}
}

func (r *lcg) next() uint32 {
	r.state = r.state*1103515245 + 12345
	return r.state
}

func (r *lcg) n(n uint32) int { return int(r.next() % n) }

// generatePayload 模拟远端服务为子树生成高成本内容。
// Rows 用字符编码图案类型，渲染器据此绘制“完整”画面。
func generatePayload(id string, seed uint32) *Payload {
	rng := newLCG(seed)
	const rows = 12
	patterns := make([]string, rows)
	for i := range patterns {
		// 每行 16 个单元，字符决定该单元的图案/色调，全部确定性。
		buf := make([]byte, 16)
		for j := range buf {
			buf[j] = byte('A' + rng.n(6))
		}
		patterns[i] = string(buf)
	}
	return &Payload{SubtreeID: id, Seed: seed, Rows: patterns}
}

// payloadColor 将 payload 单元字符映射为确定性颜色。
func payloadColor(b byte) Color {
	switch b {
	case 'A':
		return RGB(0x3b, 0x82, 0xf6)
	case 'B':
		return RGB(0x34, 0xd3, 0x99)
	case 'C':
		return RGB(0xf5, 0x9e, 0x0b)
	case 'D':
		return RGB(0xef, 0x44, 0x44)
	case 'E':
		return RGB(0xa7, 0x8b, 0xfa)
	default:
		return RGB(0x22, 0xd3, 0xee)
	}
}
