package main

import "math"

type gridKey [2]int

func growGrid(seedIdx, axisI, axisJ int, corners []Corner, spacing float64, maxKeys int) map[gridKey]int {
	grid := map[gridKey]int{{0, 0}: seedIdx, {1, 0}: axisI, {0, 1}: axisJ}
	used := map[int]bool{seedIdx: true, axisI: true, axisJ: true}
	v1 := sub2(corners[axisI].Pos, corners[seedIdx].Pos)
	v2 := sub2(corners[axisJ].Pos, corners[seedIdx].Pos)
	tol := 0.46 * spacing
	dirs := []gridKey{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for len(grid) < maxKeys {
		keys := make([]gridKey, 0, len(grid))
		for k := range grid {
			keys = append(keys, k)
		}
		sortGridKeys(keys)
		changed := false
		var pm *polyMap
		if len(grid) >= 6 {
			coords := make([]Vec2, 0, len(grid))
			points := make([]Vec2, 0, len(grid))
			for gk, ci := range grid {
				coords = append(coords, Vec2{X: float64(gk[0]), Y: float64(gk[1])})
				points = append(points, corners[ci].Pos)
			}
			pm, _ = fitPolyMap(coords, points)
		}
		for _, k := range keys {
			for _, d := range dirs {
				nk := gridKey{k[0] + d[0], k[1] + d[1]}
				if _, ok := grid[nk]; ok {
					continue
				}
				if math.Abs(float64(nk[0])) > 14 || math.Abs(float64(nk[1])) > 12 {
					continue
				}
				pred := predictGridPosition(pm, grid, corners, k, d, v1, v2)
				best, bestDist := -1, tol
				for ci, c := range corners {
					if used[ci] {
						continue
					}
					dist := math.Hypot(c.Pos.X-pred.X, c.Pos.Y-pred.Y)
					if dist < bestDist {
						bestDist = dist
						best = ci
					}
				}
				if best >= 0 {
					grid[nk] = best
					used[best] = true
					changed = true
					if len(grid) >= maxKeys {
						return grid
					}
				}
			}
		}
		if !changed {
			break
		}
	}
	return grid
}

func predictGridPosition(pm *polyMap, grid map[gridKey]int, corners []Corner, k, d gridKey, v1, v2 Vec2) Vec2 {
	if pm != nil {
		return pm.Map(Vec2{X: float64(k[0] + d[0]), Y: float64(k[1] + d[1])})
	}
	cur := corners[grid[k]].Pos
	behind := gridKey{k[0] - d[0], k[1] - d[1]}
	if bi, ok := grid[behind]; ok {
		bp := corners[bi].Pos
		return Vec2{X: 2*cur.X - bp.X, Y: 2*cur.Y - bp.Y}
	}
	switch d {
	case gridKey{1, 0}:
		return add2(cur, v1)
	case gridKey{-1, 0}:
		return sub2(cur, v1)
	case gridKey{0, 1}:
		return add2(cur, v2)
	default:
		return sub2(cur, v2)
	}
}

func sortGridKeys(keys []gridKey) {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && gridKeyLess(keys[j], keys[j-1]); j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
}

func gridKeyLess(a, b gridKey) bool {
	if a[1] != b[1] {
		return a[1] < b[1]
	}
	return a[0] < b[0]
}

func sub2(a, b Vec2) Vec2 { return Vec2{X: a.X - b.X, Y: a.Y - b.Y} }
func add2(a, b Vec2) Vec2 { return Vec2{X: a.X + b.X, Y: a.Y + b.Y} }
