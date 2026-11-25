package atlas

import (
	"errors"
	"fmt"
	"sort"
)

var (
	ErrNoSprites     = errors.New("atlas: no sprites to pack")
	ErrDoesNotFit    = errors.New("atlas: sprite does not fit in atlas")
	ErrPackingFailed = errors.New("atlas: rectangle packing failed")
)

// Sprite is the minimum geometric description of an input sub-texture.
type Sprite struct {
	ID string `json:"id"`
	W  int    `json:"width"`
	H  int    `json:"height"`
}

// Placement describes where one source sprite lives in the level-zero atlas.
type Placement struct {
	ID      string `json:"id"`
	Rotated bool   `json:"rotated"`

	// SourceW and SourceH preserve the original, unrotated dimensions.
	SourceW int `json:"source_width"`
	SourceH int `json:"source_height"`

	// Frame is the actual texel rectangle in the packed atlas. When Rotated
	// is true, Frame.W == SourceH and Frame.H == SourceW.
	Frame Rect `json:"frame"`

	// Cell is the non-overlapping packed rectangle including all padding.
	Cell Rect `json:"cell"`
}

// UVBounds returns normalized OpenGL-style bottom-left? Here v=0 is top row in
// image coordinates. The sampler consistently treats V as top-to-bottom.
func (p Placement) UVBounds(atlasW, atlasH int) (u0, v0, u1, v1 float64) {
	f := p.Frame
	return float64(f.X) / float64(atlasW),
		float64(f.Y) / float64(atlasH),
		float64(f.MaxX()) / float64(atlasW),
		float64(f.MaxY()) / float64(atlasH)
}

type PackOptions struct {
	// Width and Height optionally request a fixed atlas size. Zero enables
	// deterministic power-of-two square search.
	Width  int `json:"width"`
	Height int `json:"height"`

	MinSize int `json:"min_size"`
	MaxSize int `json:"max_size"`

	Policy PaddingPolicy `json:"policy"`

	AllowRotation bool `json:"allow_rotation"`

	// Alignment aligns packed cell origins. Set it to the deepest mip scale
	// (1 << (MipLevels-1)) to make mip boundaries phase-aligned.
	Alignment int `json:"alignment"`
}

type Layout struct {
	Width       int           `json:"width"`
	Height      int           `json:"height"`
	PackPadding int           `json:"pack_padding"`
	Alignment   int           `json:"alignment"`
	Policy      PaddingPolicy `json:"policy"`
	Placements  []Placement   `json:"placements"`
}

func (l *Layout) PlacementByID(id string) (Placement, bool) {
	for _, p := range l.Placements {
		if p.ID == id {
			return p, true
		}
	}
	return Placement{}, false
}

type packItem struct {
	sprite Sprite
	index  int
}

type candidate struct {
	freeIndex int
	x, y      int
	w, h      int
	rotated   bool
	score1    int
	score2    int
}

// Pack performs deterministic MaxRects/best-short-side packing with optional
// 90-degree rotation and mip-safe padding reservations.
func Pack(sprites []Sprite, opts PackOptions) (*Layout, error) {
	if len(sprites) == 0 {
		return nil, ErrNoSprites
	}

	opts.Policy = opts.Policy.Normalized()
	if opts.Alignment <= 0 {
		opts.Alignment = 1
	}
	pad := opts.Policy.PackPadding()

	items := make([]packItem, len(sprites))
	for i, s := range sprites {
		if s.ID == "" {
			s.ID = fmt.Sprintf("sprite-%02d", i)
		}
		if s.W <= 0 || s.H <= 0 {
			return nil, fmt.Errorf("%w: %s has invalid size %dx%d", ErrDoesNotFit, s.ID, s.W, s.H)
		}
		items[i] = packItem{sprite: s, index: i}
	}

	sort.SliceStable(items, func(i, j int) bool {
		ai := maxInt(items[i].sprite.W, items[i].sprite.H)
		aj := maxInt(items[j].sprite.W, items[j].sprite.H)
		if ai != aj {
			return ai > aj
		}
		areaI := items[i].sprite.W * items[i].sprite.H
		areaJ := items[j].sprite.W * items[j].sprite.H
		if areaI != areaJ {
			return areaI > areaJ
		}
		return items[i].sprite.ID < items[j].sprite.ID
	})

	if opts.Width > 0 && opts.Height > 0 {
		return packFixed(items, opts, opts.Width, opts.Height, pad)
	}

	minSize := opts.MinSize
	if minSize <= 0 {
		minSize = 64
	}
	maxSize := opts.MaxSize
	if maxSize <= 0 {
		maxSize = 4096
	}

	size := nextPow2(minSize)
	var lastErr error
	for size <= maxSize {
		layout, err := packFixed(items, opts, size, size, pad)
		if err == nil {
			return layout, nil
		}
		if !errors.Is(err, ErrPackingFailed) && !errors.Is(err, ErrDoesNotFit) {
			return nil, err
		}
		lastErr = err
		size *= 2
	}
	if lastErr == nil {
		lastErr = ErrPackingFailed
	}
	return nil, fmt.Errorf("%w: no square atlas up to %d", lastErr, maxSize)
}

func packFixed(items []packItem, opts PackOptions, width, height, pad int) (*Layout, error) {
	align := opts.Alignment
	freeRects := []Rect{R(0, 0, width, height)}
	placements := make([]Placement, len(items))

	for _, item := range items {
		best, err := bestCandidate(freeRects, item.sprite, pad, align, opts.AllowRotation)
		if err != nil {
			return nil, fmt.Errorf("%w: %s (%dx%d) in %dx%d: %v",
				ErrPackingFailed, item.sprite.ID, item.sprite.W, item.sprite.H, width, height, err)
		}

		cell := R(best.x, best.y, best.w, best.h)
		frame := R(cell.X+pad, cell.Y+pad, item.sprite.W, item.sprite.H)
		sourceW, sourceH := item.sprite.W, item.sprite.H
		if best.rotated {
			frame.W, frame.H = item.sprite.H, item.sprite.W
		}
		if !R(0, 0, width, height).Contains(cell) || !cell.Contains(frame) {
			return nil, fmt.Errorf("%w: invalid placement for %s", ErrPackingFailed, item.sprite.ID)
		}

		placements[item.index] = Placement{
			ID: item.sprite.ID, Rotated: best.rotated,
			SourceW: sourceW, SourceH: sourceH,
			Frame: frame, Cell: cell,
		}
		freeRects = placeMaxRects(freeRects, cell)
	}

	return &Layout{
		Width: width, Height: height, PackPadding: pad,
		Alignment: align, Policy: opts.Policy.Normalized(),
		Placements: placements,
	}, nil
}

func bestCandidate(freeRects []Rect, s Sprite, pad, align int, allowRotation bool) (candidate, error) {
	var best candidate
	found := false

	orientations := []struct {
		w, h    int
		rotated bool
	}{
		{s.W + 2*pad, s.H + 2*pad, false},
	}
	if allowRotation {
		orientations = append(orientations, struct {
			w, h    int
			rotated bool
		}{s.H + 2*pad, s.W + 2*pad, true})
	}

	for fi, free := range freeRects {
		for _, o := range orientations {
			x := alignUp(free.X, align)
			y := alignUp(free.Y, align)
			if x+o.w > free.MaxX() || y+o.h > free.MaxY() {
				continue
			}
			leftoverX := free.W - o.w
			leftoverY := free.H - o.h
			score1 := minInt(leftoverX, leftoverY)
			score2 := maxInt(leftoverX, leftoverY)
			c := candidate{
				freeIndex: fi, x: x, y: y, w: o.w, h: o.h,
				rotated: o.rotated, score1: score1, score2: score2,
			}
			if !found || c.score1 < best.score1 ||
				(c.score1 == best.score1 && c.score2 < best.score2) ||
				(c.score1 == best.score1 && c.score2 == best.score2 && c.x < best.x) ||
				(c.score1 == best.score1 && c.score2 == best.score2 && c.x == best.x && c.y < best.y) {
				best = c
				found = true
			}
		}
	}
	if !found {
		return candidate{}, ErrDoesNotFit
	}
	return best, nil
}

func placeMaxRects(freeRects []Rect, used Rect) []Rect {
	next := make([]Rect, 0, len(freeRects)*2)
	for _, free := range freeRects {
		if !free.Intersects(used) {
			next = append(next, free)
			continue
		}

		if used.X > free.X && used.X < free.MaxX() {
			next = append(next, R(free.X, free.Y, used.X-free.X, free.H))
		}
		if used.MaxX() < free.MaxX() {
			next = append(next, R(used.MaxX(), free.Y, free.MaxX()-used.MaxX(), free.H))
		}
		if used.Y > free.Y && used.Y < free.MaxY() {
			next = append(next, R(free.X, free.Y, free.W, used.Y-free.Y))
		}
		if used.MaxY() < free.MaxY() {
			next = append(next, R(free.X, used.MaxY(), free.W, free.MaxY()-used.MaxY()))
		}
	}

	pruned := make([]Rect, 0, len(next))
	for i, r := range next {
		contained := false
		for j, q := range next {
			if i == j || r.Empty() || !q.Contains(r) {
				continue
			}
			if !r.Equal(q) || j < i {
				contained = true
				break
			}
		}
		if !contained && !r.Empty() {
			pruned = append(pruned, r)
		}
	}
	return pruned
}

func alignUp(v, align int) int {
	if align <= 1 {
		return v
	}
	r := v % align
	if r == 0 {
		return v
	}
	return v + align - r
}

func nextPow2(v int) int {
	if v <= 1 {
		return 1
	}
	p := 1
	for p < v {
		p <<= 1
	}
	return p
}
