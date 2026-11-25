package main

import (
	"sort"
)

type Strategy string

const (
	StrategyFull   Strategy = "full"
	StrategyChunk  Strategy = "chunked"
	StrategyScreen Strategy = "screen-error"
)

type LoadStatus int

const (
	StatusRoot LoadStatus = iota
	StatusCoarse
	StatusLoading
	StatusExact
)

func (s LoadStatus) label() string {
	switch s {
	case StatusExact:
		return "EXACT"
	case StatusLoading:
		return "LOADING"
	case StatusCoarse:
		return "COARSE PARENT"
	default:
		return "ROOT PARENT"
	}
}

type SampleInfo struct {
	Tile          *Tile
	ResolvedLevel int
	IdealLevel    int
	Fallback      LoadStatus
	Loading       bool
	ErrorPixels   float64
}

func (i SampleInfo) Status() LoadStatus {
	if i.Loading {
		return StatusLoading
	}
	return i.Fallback
}

type Loader struct {
	Kind          Strategy
	Store         *TileStore
	Pending       map[NodeID]bool
	Priority      map[NodeID]float64
	FullRemaining int
	BytesLoaded   int
}

func NewLoader(kind Strategy, src *TextureSource) *Loader {
	l := &Loader{
		Kind:     kind,
		Store:    NewTileStore(src),
		Pending:  make(map[NodeID]bool),
		Priority: make(map[NodeID]float64),
	}
	if kind == StrategyFull {
		l.FullRemaining = fullBytes
	}
	return l
}

func (l *Loader) addPending(id NodeID, priority float64) {
	if !id.valid() || l.Store.Has(id) || l.Pending[id] {
		return
	}
	l.Pending[id] = true
	l.Priority[id] = priority
}

func (l *Loader) Plan(cam Camera, vp Viewport) {
	ideal := idealLOD(cam)
	switch l.Kind {
	case StrategyFull:
		return
	case StrategyChunk:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				id := NodeID{Level: maxLOD, X: x, Y: y}
				l.addPending(id, 0)
			}
		}
	case StrategyScreen:
		u0, v0, u1, v1 := visibleUVBounds(cam, vp)
		ids := nodesInBounds(ideal, u0, v0, u1, v1)
		desired := make(map[NodeID]bool, len(ids))
		for _, id := range ids {
			desired[id] = true
			center := nodeCenter(id)
			distance := (center.U-cam.Center.U)*(center.U-cam.Center.U) +
				(center.V-cam.Center.V)*(center.V-cam.Center.V)
			errorValue := nodeTexelPixels(id.Level, cam.Scale)
			l.addPending(id, errorValue*100-distance*10)
		}
		for id := range l.Pending {
			if id.Level == ideal && !desired[id] {
				delete(l.Pending, id)
				delete(l.Priority, id)
			}
		}
	}
}

func (l *Loader) Transfer(cam Camera) {
	if l.Kind == StrategyFull {
		if l.FullRemaining <= 0 {
			return
		}
		spent := minInt(frameBudget, l.FullRemaining)
		l.FullRemaining -= spent
		l.BytesLoaded += spent
		if l.FullRemaining == 0 {
			l.Store.AddFullTexture()
		}
		return
	}

	slots := frameBudget / tileBytes
	ids := make([]NodeID, 0, len(l.Pending))
	for id := range l.Pending {
		if !l.Store.Has(id) {
			ids = append(ids, id)
		}
	}
	sort.SliceStable(ids, func(i, j int) bool {
		a, b := ids[i], ids[j]
		if l.Kind == StrategyScreen {
			pa, pb := l.Priority[a], l.Priority[b]
			if pa != pb {
				return pa > pb
			}
		}
		if a.Level != b.Level {
			return a.Level < b.Level
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})

	for _, id := range ids {
		if slots <= 0 {
			break
		}
		delete(l.Pending, id)
		delete(l.Priority, id)
		l.Store.Add(id)
		l.BytesLoaded += tileBytes
		slots--
	}
}

func (l *Loader) Inspect(uv UV, cam Camera) SampleInfo {
	ideal := idealLOD(cam)
	tile, resolved := l.resolveBest(uv, ideal)
	info := SampleInfo{
		Tile:          tile,
		ResolvedLevel: resolved,
		IdealLevel:    ideal,
		ErrorPixels:   screenErrorPixels(resolved, cam),
	}

	if resolved >= ideal {
		info.Fallback = StatusExact
		return info
	}
	if resolved == 0 {
		info.Fallback = StatusRoot
	} else {
		info.Fallback = StatusCoarse
	}

	if l.Kind == StrategyFull && l.FullRemaining > 0 {
		info.Loading = true
		return info
	}
	requestLevel := ideal
	if l.Kind == StrategyChunk {
		requestLevel = maxLOD
	}
	if l.Pending[nodeAtUV(uv, requestLevel)] {
		info.Loading = true
	}
	return info
}

func (l *Loader) resolveBest(uv UV, ideal int) (*Tile, int) {
	for level := maxLOD; level >= ideal; level-- {
		id := nodeAtUV(uv, level)
		if l.Store.Has(id) {
			return l.Store.Source.Tiles[id], level
		}
	}
	return l.Store.Resolve(uv, ideal)
}

func visibleUVBounds(cam Camera, vp Viewport) (float64, float64, float64, float64) {
	a := cam.UVAt(vp.X, vp.Y, vp)
	b := cam.UVAt(vp.X+vp.Size-1, vp.Y+vp.Size-1, vp)
	u0 := clampFloat(minFloat(a.U, b.U), 0, 1)
	u1 := clampFloat(maxFloat(a.U, b.U), 0, 1)
	v0 := clampFloat(minFloat(a.V, b.V), 0, 1)
	v1 := clampFloat(maxFloat(a.V, b.V), 0, 1)
	return u0, v0, u1, v1
}

func nodesInBounds(level int, u0, v0, u1, v1 float64) []NodeID {
	count := 1 << level
	x0 := tileIndex(u0, count)
	x1 := tileIndex(mathNextBeforeOne(u1), count)
	y0 := tileIndex(v0, count)
	y1 := tileIndex(mathNextBeforeOne(v1), count)
	var ids []NodeID
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			ids = append(ids, NodeID{Level: level, X: x, Y: y})
		}
	}
	return ids
}

func mathNextBeforeOne(v float64) float64 {
	if v >= 1 {
		return 1 - 1e-9
	}
	return v
}

func nodeCenter(id NodeID) UV {
	n := float64(levelCount(id.Level))
	return UV{
		U: (float64(id.X) + 0.5) / n,
		V: (float64(id.Y) + 0.5) / n,
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
