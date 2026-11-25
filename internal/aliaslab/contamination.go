package aliaslab

type Bounds struct {
	MinX int `json:"min_x"`
	MinY int `json:"min_y"`
	MaxX int `json:"max_x"`
	MaxY int `json:"max_y"`
}

func (b Bounds) Width() int  { return b.MaxX - b.MinX + 1 }
func (b Bounds) Height() int { return b.MaxY - b.MinY + 1 }

type Contamination struct {
	FirstX       int    `json:"first_x"`
	FirstY       int    `json:"first_y"`
	Bounds       Bounds `json:"bounds"`
	RegionPixels int    `json:"region_pixels"`
	TotalPixels  int    `json:"total_pixels"`
}

// FindFirstContamination performs a row-major scan and then a 4-connected
// flood fill from the first differing pixel. That component is the first
// polluted region in scan order.
func FindFirstContamination(reference, candidate *Image) (Contamination, bool) {
	if reference.Width != candidate.Width || reference.Height != candidate.Height {
		return Contamination{}, true
	}
	width, height := reference.Width, reference.Height
	first := -1
	total := 0
	for i := range reference.Pix {
		if reference.Pix[i] != candidate.Pix[i] {
			total++
			if first < 0 {
				first = i
			}
		}
	}
	if first < 0 {
		return Contamination{}, false
	}

	x0, y0 := first%width, first/width
	bounds := Bounds{MinX: x0, MinY: y0, MaxX: x0, MaxY: y0}
	seen := make([]bool, len(reference.Pix))
	queue := make([]int, 0, 128)
	queue = append(queue, first)
	seen[first] = true
	count := 0
	for head := 0; head < len(queue); head++ {
		index := queue[head]
		if reference.Pix[index] == candidate.Pix[index] {
			continue
		}
		count++
		x, y := index%width, index/width
		if x < bounds.MinX {
			bounds.MinX = x
		}
		if y < bounds.MinY {
			bounds.MinY = y
		}
		if x > bounds.MaxX {
			bounds.MaxX = x
		}
		if y > bounds.MaxY {
			bounds.MaxY = y
		}
		add := func(nx, ny int) {
			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				return
			}
			next := ny*width + nx
			if seen[next] || reference.Pix[next] == candidate.Pix[next] {
				return
			}
			seen[next] = true
			queue = append(queue, next)
		}
		add(x-1, y)
		add(x+1, y)
		add(x, y-1)
		add(x, y+1)
	}
	return Contamination{
		FirstX:       x0,
		FirstY:       y0,
		Bounds:       bounds,
		RegionPixels: count,
		TotalPixels:  total,
	}, true
}

// HighlightFirstRegion draws a black/yellow double frame plus a white/red
// crosshair. The erroneous pixels inside remain visible.
func HighlightFirstRegion(img *Image, c Contamination) {
	outer := Bounds{
		MinX: c.Bounds.MinX - 5,
		MinY: c.Bounds.MinY - 5,
		MaxX: c.Bounds.MaxX + 5,
		MaxY: c.Bounds.MaxY + 5,
	}
	drawFrame(img, outer, 5, RGB{5, 8, 14})
	inner := Bounds{
		MinX: c.Bounds.MinX - 2,
		MinY: c.Bounds.MinY - 2,
		MaxX: c.Bounds.MaxX + 2,
		MaxY: c.Bounds.MaxY + 2,
	}
	drawFrame(img, inner, 3, RGB{255, 212, 59})

	for d := -5; d <= 5; d++ {
		setIfInside(img, c.FirstX+d, c.FirstY, RGB{255, 255, 255})
		setIfInside(img, c.FirstX, c.FirstY+d, RGB{255, 255, 255})
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			setIfInside(img, c.FirstX+dx, c.FirstY+dy, RGB{230, 35, 35})
		}
	}
}

func drawFrame(img *Image, b Bounds, thickness int, color RGB) {
	for t := 0; t < thickness; t++ {
		x0, y0 := b.MinX+t, b.MinY+t
		x1, y1 := b.MaxX-t, b.MaxY-t
		if x0 > x1 || y0 > y1 {
			return
		}
		for x := x0; x <= x1; x++ {
			setIfInside(img, x, y0, color)
			setIfInside(img, x, y1, color)
		}
		for y := y0; y <= y1; y++ {
			setIfInside(img, x0, y, color)
			setIfInside(img, x1, y, color)
		}
	}
}

func setIfInside(img *Image, x, y int, c RGB) {
	if x >= 0 && x < img.Width && y >= 0 && y < img.Height {
		img.Set(x, y, c)
	}
}
