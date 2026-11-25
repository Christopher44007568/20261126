package hotreload

// DiffThreshold is the inclusive per-pixel Euclidean RGB distance used to
// distinguish noise from real reload changes.
const DiffThreshold = 28

// ComputeDiff returns a highlighted frame and statistics for two renders.
// Unchanged pixels are desaturated; changed pixels become magenta with a
// yellow contour around the changed-pixel bounding box.
func ComputeDiff(before, after *Frame) (*Frame, DiffStats) {
	out := NewFrame(after.W, after.H)
	bounds := Bounds{Valid: false}
	changed := 0
	total := after.W * after.H

	mask := make([]bool, total)
	for y := 0; y < after.H; y++ {
		for x := 0; x < after.W; x++ {
			a := before.Get(x, y)
			b := after.Get(x, y)
			if colorDistance(a, b) > DiffThreshold {
				i := y*after.W + x
				mask[i] = true
				changed++
				if !bounds.Valid {
					bounds = Bounds{MinX: x, MinY: y, MaxX: x, MaxY: y, Valid: true}
				} else {
					if x < bounds.MinX {
						bounds.MinX = x
					}
					if x > bounds.MaxX {
						bounds.MaxX = x
					}
					if y < bounds.MinY {
						bounds.MinY = y
					}
					if y > bounds.MaxY {
						bounds.MaxY = y
					}
				}
			}
		}
	}

	for y := 0; y < after.H; y++ {
		for x := 0; x < after.W; x++ {
			i := y*after.W + x
			if mask[i] {
				out.Set(x, y, ColorMagenta)
				continue
			}
			c := after.Get(x, y)
			lum := (uint16(c.R)*299 + uint16(c.G)*587 + uint16(c.B)*114) / 1000
			g := uint8(lum)
			out.Set(x, y, RGB{g, g, g})
		}
	}

	if bounds.Valid {
		out.Rect(bounds.MinX, bounds.MinY, bounds.MaxX+1, bounds.MaxY+1, 1, ColorYellow)
	}

	stats := DiffStats{
		ChangedPixels: changed,
		TotalPixels:   total,
		Percent:       float64(changed) * 100 / float64(total),
		Bounds:        bounds,
		Threshold:     DiffThreshold,
	}
	return out, stats
}

func colorDistance(a, b RGB) float64 {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	sum := dr*dr + dg*dg + db*db
	// Integer square root keeps the renderer allocation-free.
	return float64(isqrt(sum))
}

func isqrt(v int) int {
	if v < 0 {
		return 0
	}
	x := v
	if x == 0 {
		return 0
	}
	guess := x
	for {
		next := (guess + x/guess) / 2
		if next >= guess {
			return guess
		}
		guess = next
	}
}
