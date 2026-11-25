package main

import (
	"math"
)

const (
	sourceSize  = 512
	tilePixels  = 64
	maxLOD      = 3
	tileBytes   = 4096
	fullBytes   = 32 * tileBytes
	frameBudget = 4 * tileBytes
)

type UV struct {
	U, V float64
}

type NodeID struct {
	Level int
	X     int
	Y     int
}

func (id NodeID) valid() bool {
	if id.Level < 0 || id.Level > maxLOD {
		return false
	}
	n := 1 << id.Level
	return id.X >= 0 && id.Y >= 0 && id.X < n && id.Y < n
}

type Tile struct {
	ID   NodeID
	Size int
	Pix  []Color
}

func (t *Tile) sample(u, v float64) Color {
	x := clampFloat(u*float64(t.Size)-0.5, 0, float64(t.Size-1))
	y := clampFloat(v*float64(t.Size)-0.5, 0, float64(t.Size-1))
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := minInt(x0+1, t.Size-1)
	y1 := minInt(y0+1, t.Size-1)
	fx := x - float64(x0)
	fy := y - float64(y0)

	c00 := t.Pix[y0*t.Size+x0]
	c10 := t.Pix[y0*t.Size+x1]
	c01 := t.Pix[y1*t.Size+x0]
	c11 := t.Pix[y1*t.Size+x1]

	r := bilinear(c00.R, c10.R, c01.R, c11.R, fx, fy)
	g := bilinear(c00.G, c10.G, c01.G, c11.G, fx, fy)
	b := bilinear(c00.B, c10.B, c01.B, c11.B, fx, fy)
	return rgb(int(r), int(g), int(b))
}

func bilinear(a, b, c, d uint8, fx, fy float64) float64 {
	top := float64(a)*(1-fx) + float64(b)*fx
	bottom := float64(c)*(1-fx) + float64(d)*fx
	return top*(1-fy) + bottom*fy
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

type TextureSource struct {
	Tiles map[NodeID]*Tile
}

func NewTextureSource() *TextureSource {
	high := generateSourceTexture(sourceSize)
	mips := make([][]Color, maxLOD+1)
	mips[0] = high
	for level := 1; level <= maxLOD; level++ {
		prevSize := sourceSize >> (level - 1)
		nextSize := sourceSize >> level
		mips[level] = downsampleBox(mips[level-1], prevSize, prevSize, nextSize, nextSize)
	}

	src := &TextureSource{Tiles: make(map[NodeID]*Tile)}
	for lod := 0; lod <= maxLOD; lod++ {
		mipIndex := maxLOD - lod
		mip := mips[mipIndex]
		mipSize := sourceSize >> mipIndex
		count := 1 << lod
		for ty := 0; ty < count; ty++ {
			for tx := 0; tx < count; tx++ {
				tile := &Tile{ID: NodeID{Level: lod, X: tx, Y: ty}, Size: tilePixels}
				tile.Pix = make([]Color, tilePixels*tilePixels)
				offX := tx * tilePixels
				offY := ty * tilePixels
				for y := 0; y < tilePixels; y++ {
					for x := 0; x < tilePixels; x++ {
						tile.Pix[y*tilePixels+x] = mip[(offY+y)*mipSize+offX+x]
					}
				}
				src.Tiles[tile.ID] = tile
			}
		}
	}
	return src
}

func generateSourceTexture(size int) []Color {
	pix := make([]Color, size*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			u := (float64(x) + 0.5) / float64(size)
			v := (float64(y) + 0.5) / float64(size)

			wave := 0.5 + 0.5*math.Sin(18*u+6*math.Sin(9*v))
			radial := 0.5 + 0.5*math.Sin(42*math.Hypot(u-0.68, v-0.56))
			band := 0.5 + 0.5*math.Sin((u*1.7-v)*24)

			r := 34 + 72*u + 34*wave + 18*band
			g := 74 + 82*v + 28*radial + 20*wave
			b := 104 + 70*(1-u) + 36*band + 18*radial

			if contour := math.Abs(math.Sin(34 * math.Hypot(u-0.28, v-0.34))); contour < 0.055 {
				r -= 34
				g -= 24
				b -= 10
			}
			if x%32 == 0 || y%32 == 0 {
				r -= 18
				g -= 14
				b -= 8
			}
			if x%64 == 0 || y%64 == 0 {
				r -= 38
				g -= 30
				b -= 18
			}
			if x%16 == 0 || y%16 == 0 {
				r += 10
				g += 12
				b += 12
			}

			pix[y*size+x] = rgb(int(clampFloat(r, 22, 232)), int(clampFloat(g, 30, 236)), int(clampFloat(b, 48, 242)))
		}
	}
	return pix
}

func downsampleBox(src []Color, sw, sh, dw, dh int) []Color {
	dst := make([]Color, dw*dh)
	xStep := sw / dw
	yStep := sh / dh
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var sr, sg, sb, count int
			for yy := y * yStep; yy < (y+1)*yStep; yy++ {
				for xx := x * xStep; xx < (x+1)*xStep; xx++ {
					c := src[yy*sw+xx]
					sr += int(c.R)
					sg += int(c.G)
					sb += int(c.B)
					count++
				}
			}
			dst[y*dw+x] = rgb(sr/count, sg/count, sb/count)
		}
	}
	return dst
}

type TileStore struct {
	Source   *TextureSource
	resident map[NodeID]bool
	Full     bool
}

func NewTileStore(src *TextureSource) *TileStore {
	store := &TileStore{Source: src, resident: make(map[NodeID]bool)}
	store.resident[NodeID{Level: 0, X: 0, Y: 0}] = true
	return store
}

func (s *TileStore) Has(id NodeID) bool {
	return s.resident[id]
}

func (s *TileStore) Add(id NodeID) {
	if id.valid() {
		s.resident[id] = true
	}
}

func (s *TileStore) AddFullTexture() {
	s.Full = true
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			s.Add(NodeID{Level: maxLOD, X: x, Y: y})
		}
	}
}

func (s *TileStore) Resolve(uv UV, idealLevel int) (*Tile, int) {
	level := minInt(idealLevel, maxLOD)
	for level >= 0 {
		count := 1 << level
		id := NodeID{
			Level: level,
			X:     tileIndex(uv.U, count),
			Y:     tileIndex(uv.V, count),
		}
		if s.resident[id] {
			return s.Source.Tiles[id], level
		}
		level--
	}
	root := NodeID{}
	return s.Source.Tiles[root], 0
}

func tileIndex(v float64, count int) int {
	idx := int(math.Floor(v * float64(count)))
	if idx < 0 {
		return 0
	}
	if idx >= count {
		return count - 1
	}
	return idx
}

func nodeAtUV(uv UV, level int) NodeID {
	count := 1 << level
	return NodeID{Level: level, X: tileIndex(uv.U, count), Y: tileIndex(uv.V, count)}
}
