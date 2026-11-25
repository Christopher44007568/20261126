package main

import (
	"errors"
	"image"
	"math"
	"sort"
)

var errInsufficientPoints = errors.New("insufficient points for polynomial grid map")

type DetectedPoint struct {
	ID    int     `json:"id"`
	I     int     `json:"i"`
	J     int     `json:"j"`
	Pos   Vec2    `json:"pos"`
	Found bool    `json:"found"`
	Score float64 `json:"score,omitempty"`
}

type Detection struct {
	Width          int             `json:"width"`
	Height         int             `json:"height"`
	Spec           CheckerSpec     `json:"spec"`
	Points         []DetectedPoint `json:"points"`
	RawCornerCount int             `json:"raw_corner_count"`
	Spacing        float64         `json:"spacing_px"`
	FoundCount     int             `json:"found_count"`
	Complete       bool            `json:"complete"`
	EdgeComplete   bool            `json:"edge_complete"`
	Usable         bool            `json:"usable_for_fit"`
	RMSE           float64         `json:"fit_rmse_px"`
}

func (d Detection) Observed() []Vec2 {
	out := make([]Vec2, 0, d.FoundCount)
	for _, p := range d.Points {
		if p.Found {
			out = append(out, p.Pos)
		}
	}
	return out
}

func (d Detection) FoundByID() map[int]Vec2 {
	out := make(map[int]Vec2, d.FoundCount)
	for _, p := range d.Points {
		if p.Found {
			out[p.ID] = p.Pos
		}
	}
	return out
}

func (d Detection) ObjectPoints() []Vec3 {
	out := make([]Vec3, 0, d.FoundCount)
	for _, p := range d.Points {
		if p.Found {
			out = append(out, Vec3{X: float64(p.I) * d.Spec.Square, Y: float64(p.J) * d.Spec.Square})
		}
	}
	return out
}

func DetectCheckerboard(img *image.RGBA, spec CheckerSpec) Detection {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	result := Detection{Width: w, Height: h, Spec: spec}
	n := spec.CornersX * spec.CornersY
	result.Points = make([]DetectedPoint, n)
	for j := 0; j < spec.CornersY; j++ {
		for i := 0; i < spec.CornersX; i++ {
			id := j*spec.CornersX + i
			result.Points[id] = DetectedPoint{ID: id, I: i, J: j}
		}
	}
	corners, gray := detectHarrisCorners(img)
	result.RawCornerCount = len(corners)
	spacing := nearestSpacing(corners)
	result.Spacing = spacing
	if len(corners) < 6 || spacing <= 0 {
		return result
	}
	corners = filterInnerCheckerCorners(corners, gray, b.Dx(), b.Dy(), spacing)
	spacing = nearestSpacing(corners)
	result.Spacing = spacing

	best := alignBestGrid(corners, spec, spacing)
	if best == nil {
		return result
	}
	result.Points = best.points
	canonicalizeGridChirality(&result, spec)
	result.FoundCount = best.found
	result.RMSE = best.rmse
	result.Complete = best.found == n
	result.EdgeComplete = borderComplete(best.points, spec)
	result.Usable = result.Complete && result.RMSE < spacing*0.35
	return result
}

type alignedGrid struct {
	points []DetectedPoint
	found  int
	rmse   float64
}

func nearestSpacing(corners []Corner) float64 {
	if len(corners) < 2 {
		return 0
	}
	var ds []float64
	for i, c := range corners {
		best := math.Inf(1)
		for j, q := range corners {
			if i == j {
				continue
			}
			best = math.Min(best, math.Hypot(c.Pos.X-q.Pos.X, c.Pos.Y-q.Pos.Y))
		}
		if math.IsInf(best, 1) == false {
			ds = append(ds, best)
		}
	}
	sort.Float64s(ds)
	if len(ds) == 0 {
		return 0
	}
	return ds[len(ds)/2]
}

func alignBestGrid(corners []Corner, spec CheckerSpec, spacing float64) *alignedGrid {
	maxKeys := (spec.CornersX + 3) * (spec.CornersY + 3)
	seedLimit := min(len(corners), 70)
	var best *alignedGrid
	for si := 0; si < seedLimit; si++ {
		type neighbor struct {
			idx int
			v   Vec2
		}
		var ns []neighbor
		for j, c := range corners {
			if j == si {
				continue
			}
			v := sub2(c.Pos, corners[si].Pos)
			d := math.Hypot(v.X, v.Y)
			if d >= spacing*0.48 && d <= spacing*1.65 {
				ns = append(ns, neighbor{idx: j, v: v})
			}
		}
		sort.Slice(ns, func(a, b int) bool {
			return math.Hypot(ns[a].v.X, ns[a].v.Y) < math.Hypot(ns[b].v.X, ns[b].v.Y)
		})
		if len(ns) > 9 {
			ns = ns[:9]
		}
		for a := 0; a < len(ns); a++ {
			for b := a + 1; b < len(ns); b++ {
				la := math.Hypot(ns[a].v.X, ns[a].v.Y)
				lb := math.Hypot(ns[b].v.X, ns[b].v.Y)
				dot := ns[a].v.X*ns[b].v.X + ns[a].v.Y*ns[b].v.Y
				if math.Abs(dot)/(la*lb) > 0.72 {
					continue
				}
				grid := growGrid(si, ns[a].idx, ns[b].idx, corners, spacing, maxKeys)
				if len(grid) < min(12, spec.CornersX*spec.CornersY/2) {
					continue
				}
				ag := alignGrownGrid(grid, corners, spec, spacing)
				if ag != nil && (best == nil || ag.found > best.found ||
					(ag.found == best.found && ag.rmse < best.rmse)) {
					best = ag
				}
				if best != nil && best.found == spec.CornersX*spec.CornersY && best.rmse < 2.5 {
					return best
				}
			}
		}
	}
	return best
}

var gridMatrices = [][2][2]int{
	{{1, 0}, {0, 1}},
	{{0, -1}, {1, 0}},
	{{-1, 0}, {0, -1}},
	{{0, 1}, {-1, 0}},
	{{0, 1}, {1, 0}},
	{{0, -1}, {-1, 0}},
	{{1, 0}, {0, -1}},
	{{-1, 0}, {0, 1}},
}

func alignGrownGrid(grid map[gridKey]int, corners []Corner, spec CheckerSpec, spacing float64) *alignedGrid {
	gminI, gmaxI := math.MaxInt, math.MinInt
	gminJ, gmaxJ := math.MaxInt, math.MinInt
	for k := range grid {
		gminI, gmaxI = min(gminI, k[0]), max(gmaxI, k[0])
		gminJ, gmaxJ = min(gminJ, k[1]), max(gmaxJ, k[1])
	}
	var best *alignedGrid
	for _, mat := range gridMatrices {
		tminI, tmaxI, tminJ, tmaxJ := math.MaxInt, math.MinInt, math.MaxInt, math.MinInt
		for j := 0; j < spec.CornersY; j++ {
			for i := 0; i < spec.CornersX; i++ {
				ti := mat[0][0]*i + mat[0][1]*j
				tj := mat[1][0]*i + mat[1][1]*j
				tminI, tmaxI = min(tminI, ti), max(tmaxI, ti)
				tminJ, tmaxJ = min(tminJ, tj), max(tmaxJ, tj)
			}
		}
		widthI, widthJ := tmaxI-tminI, tmaxJ-tminJ
		type offset struct{ i, j, overlap int }
		var offsets []offset
		bestOverlap := 0
		for offI := gminI - 2; offI <= gmaxI+2-widthI; offI++ {
			for offJ := gminJ - 2; offJ <= gmaxJ+2-widthJ; offJ++ {
				tx := offI - tminI
				ty := offJ - tminJ
				overlap := 0
				for j := 0; j < spec.CornersY; j++ {
					for i := 0; i < spec.CornersX; i++ {
						gi := mat[0][0]*i + mat[0][1]*j + tx
						gj := mat[1][0]*i + mat[1][1]*j + ty
						if _, ok := grid[gridKey{gi, gj}]; ok {
							overlap++
						}
					}
				}
				if overlap >= min(8, spec.CornersX*spec.CornersY/2) {
					offsets = append(offsets, offset{offI, offJ, overlap})
					bestOverlap = max(bestOverlap, overlap)
				}
			}
		}
		for _, off := range offsets {
			if off.overlap < bestOverlap-3 {
				continue
			}
			tx := off.i - tminI
			ty := off.j - tminJ
			ag := evaluateAlignment(mat, tx, ty, grid, corners, spec, spacing)
			if ag != nil && (best == nil || ag.found > best.found ||
				(ag.found == best.found && ag.rmse < best.rmse)) {
				best = ag
			}
		}
	}
	return best
}

func evaluateAlignment(mat [2][2]int, tx, ty int, grid map[gridKey]int, corners []Corner, spec CheckerSpec, spacing float64) *alignedGrid {
	var coords []Vec2
	var points []Vec2
	type mapped struct {
		id  int
		pos Vec2
	}
	var mappedKnown []mapped
	for j := 0; j < spec.CornersY; j++ {
		for i := 0; i < spec.CornersX; i++ {
			gi := mat[0][0]*i + mat[0][1]*j + tx
			gj := mat[1][0]*i + mat[1][1]*j + ty
			if ci, ok := grid[gridKey{gi, gj}]; ok {
				q := Vec2{X: float64(i), Y: float64(j)}
				p := corners[ci].Pos
				coords = append(coords, q)
				points = append(points, p)
				mappedKnown = append(mappedKnown, mapped{id: j*spec.CornersX + i, pos: p})
			}
		}
	}
	if len(coords) < min(8, spec.CornersX*spec.CornersY/2) {
		return nil
	}
	pm, err := fitPolyMap(coords, points)
	if err != nil {
		return nil
	}
	type pair struct {
		id   int
		ci   int
		dist float64
		pos  Vec2
	}
	var pairs []pair
	matchTol := spacing * 0.38
	for j := 0; j < spec.CornersY; j++ {
		for i := 0; i < spec.CornersX; i++ {
			id := j*spec.CornersX + i
			pred := pm.Map(Vec2{X: float64(i), Y: float64(j)})
			for ci, c := range corners {
				d := math.Hypot(c.Pos.X-pred.X, c.Pos.Y-pred.Y)
				if d <= matchTol {
					pairs = append(pairs, pair{id: id, ci: ci, dist: d, pos: c.Pos})
				}
			}
		}
	}
	sort.Slice(pairs, func(a, b int) bool { return pairs[a].dist < pairs[b].dist })
	usedCorner := map[int]bool{}
	gotID := map[int]Vec2{}
	var sum2 float64
	for _, p := range pairs {
		if usedCorner[p.ci] {
			continue
		}
		if _, exists := gotID[p.id]; exists {
			continue
		}
		usedCorner[p.ci] = true
		gotID[p.id] = p.pos
		sum2 += p.dist * p.dist
	}
	if len(gotID) < min(12, spec.CornersX*spec.CornersY/2) {
		return nil
	}
	out := make([]DetectedPoint, spec.CornersX*spec.CornersY)
	for j := 0; j < spec.CornersY; j++ {
		for i := 0; i < spec.CornersX; i++ {
			id := j*spec.CornersX + i
			dp := DetectedPoint{ID: id, I: i, J: j}
			if pos, ok := gotID[id]; ok {
				dp.Found = true
				dp.Pos = pos
			}
			out[id] = dp
		}
	}
	return &alignedGrid{points: out, found: len(gotID), rmse: math.Sqrt(sum2 / float64(len(gotID)))}
}

func borderComplete(points []DetectedPoint, spec CheckerSpec) bool {
	for _, p := range points {
		border := p.I == 0 || p.J == 0 || p.I == spec.CornersX-1 || p.J == spec.CornersY-1
		if border && !p.Found {
			return false
		}
	}
	return true
}

func canonicalizeGridChirality(d *Detection, spec CheckerSpec) {
	var viSum, vjSum Vec2
	var ni, nj int
	for j := 0; j < spec.CornersY; j++ {
		a := d.Points[j*spec.CornersX]
		b := d.Points[j*spec.CornersX+spec.CornersX-1]
		if a.Found && b.Found {
			viSum.X += b.Pos.X - a.Pos.X
			viSum.Y += b.Pos.Y - a.Pos.Y
			ni++
		}
	}
	for i := 0; i < spec.CornersX; i++ {
		a := d.Points[i]
		b := d.Points[(spec.CornersY-1)*spec.CornersX+i]
		if a.Found && b.Found {
			vjSum.X += b.Pos.X - a.Pos.X
			vjSum.Y += b.Pos.Y - a.Pos.Y
			nj++
		}
	}
	if ni == 0 || nj == 0 {
		return
	}
	vi := Vec2{X: viSum.X / float64(ni), Y: viSum.Y / float64(ni)}
	vj := Vec2{X: vjSum.X / float64(nj), Y: vjSum.Y / float64(nj)}
	cross := vi.X*vj.Y - vi.Y*vj.X
	if cross >= 0 {
		return
	}
	next := make([]DetectedPoint, len(d.Points))
	for _, p := range d.Points {
		newJ := spec.CornersY - 1 - p.J
		p.J = newJ
		p.ID = newJ*spec.CornersX + p.I
		next[p.ID] = p
	}
	d.Points = next
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
