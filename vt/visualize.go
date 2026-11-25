package vt

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

func WritePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		return err
	}
	return file.Sync()
}

// RenderResidencyMap draws every mip page. Bright cells are resident; dark
// cells are absent. White outlines are exact pages requested by UV feedback.
func RenderResidencyMap(info Info, resident map[PageID]SnapshotEntry, requested map[PageID]int, cell int) *image.RGBA {
	if cell < 8 {
		cell = 8
	}
	pad := 18
	gap := 14
	width := info.PagesPerSide*cell + pad*2
	height := pad*2 + (info.PagesPerSide*(cell+gap) - gap) +
		(info.PagesPerSide/2*(cell+gap) - gap) +
		(info.PagesPerSide/4*(cell+gap) - gap) + cell
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{R: 17, G: 24, B: 39, A: 255}}, image.Point{}, draw.Src)

	y := pad
	for mip := 0; mip <= info.MaxMip; mip++ {
		n := info.Count(mip)
		grid := n * cell
		x := pad
		for py := 0; py < n; py++ {
			for px := 0; px < n; px++ {
				id := PageID{Mip: mip, X: px, Y: py}
				r := image.Rect(x+px*cell, y+py*cell, x+(px+1)*cell, y+(py+1)*cell)
				fill := color.RGBA{R: 31, G: 41, B: 55, A: 255}
				if entry, ok := resident[id]; ok {
					fill = slotColor(entry.Slot)
				}
				draw.Draw(canvas, r, &image.Uniform{C: fill}, image.Point{}, draw.Src)
				border := color.RGBA{R: 75, G: 85, B: 99, A: 255}
				drawRectOutline(canvas, r, border)
				if _, asked := requested[id]; asked && mip == requestedMip(requested) {
					drawRectOutlineThick(canvas, r.Inset(1), color.RGBA{R: 255, G: 255, B: 255, A: 255}, 2)
				}
				if _, ok := resident[id]; ok {
					drawRectOutline(canvas, r.Inset(3), color.RGBA{R: 15, G: 23, B: 42, A: 180})
				}
			}
		}
		drawRectOutline(canvas, image.Rect(x-1, y-1, x+grid+1, y+n*cell+1),
			color.RGBA{R: 148, G: 163, B: 184, A: 255})
		y += n*cell + gap
	}
	return canvas
}

func requestedMip(requested map[PageID]int) int {
	for id := range requested {
		return id.Mip
	}
	return 0
}

// RenderPhysicalSlots shows independent physical cache slots with dark gutters.
func RenderPhysicalSlots(table *Table) *image.RGBA {
	capacity := table.Capacity()
	snapshot := table.Snapshot()
	bySlot := make(map[int]SnapshotEntry, capacity)
	for _, entry := range snapshot {
		bySlot[entry.Slot] = entry
	}
	columns := 4
	if capacity < columns {
		columns = capacity
	}
	rows := (capacity + columns - 1) / columns
	gutter := 8
	cell := DefaultInfo().PageSize
	width := columns*(cell+gutter) + gutter
	height := rows*(cell+gutter) + gutter
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{R: 3, G: 7, B: 18, A: 255}}, image.Point{}, draw.Src)

	for slot := 0; slot < capacity; slot++ {
		col := slot % columns
		row := slot / columns
		origin := image.Pt(gutter+col*(cell+gutter), gutter+row*(cell+gutter))
		r := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(cell, cell))}
		draw.Draw(canvas, r, &image.Uniform{C: color.RGBA{R: 17, G: 24, B: 39, A: 255}}, image.Point{}, draw.Src)
		if entry, ok := bySlot[slot]; ok && entry.Image != nil {
			draw.Draw(canvas, r, entry.Image, image.Point{}, draw.Src)
			drawRectOutlineThick(canvas, r, slotColor(slot), 3)
		} else {
			drawRectOutline(canvas, r, color.RGBA{R: 55, G: 65, B: 81, A: 255})
		}
	}
	return canvas
}

// RenderFeedback visualizes the coarse UV feedback buffer by requested page.
func RenderFeedback(fb *FeedbackBuffer, scale int) *image.RGBA {
	if scale < 1 {
		scale = 1
	}
	canvas := image.NewRGBA(image.Rect(0, 0, fb.Width*scale, fb.Height*scale))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{R: 10, G: 12, B: 20, A: 255}}, image.Point{}, draw.Src)
	for _, sample := range fb.Samples {
		c := pageColor(sample.Page)
		r := image.Rect(sample.BufferX*scale, sample.BufferY*scale,
			(sample.BufferX+1)*scale, (sample.BufferY+1)*scale)
		draw.Draw(canvas, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
	}
	return canvas
}

func drawRectOutline(img *image.RGBA, r image.Rectangle, c color.Color) {
	for x := r.Min.X; x < r.Max.X; x++ {
		img.Set(x, r.Min.Y, c)
		img.Set(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		img.Set(r.Min.X, y, c)
		img.Set(r.Max.X-1, y, c)
	}
}

func drawRectOutlineThick(img *image.RGBA, r image.Rectangle, c color.Color, thickness int) {
	for t := 0; t < thickness; t++ {
		drawRectOutline(img, image.Rect(r.Min.X+t, r.Min.Y+t, r.Max.X-t, r.Max.Y-t), c)
	}
}

func slotColor(slot int) color.RGBA {
	hue := float64((slot*47 + 12) % 360)
	return hsl(hue, 0.62, 0.52)
}

func pageColor(id PageID) color.RGBA {
	hue := float64((id.Mip*97 + id.X*31 + id.Y*53) % 360)
	return hsl(hue, 0.58, 0.58)
}

func hsl(h, s, l float64) color.RGBA {
	c := (1 - absFloat(2*l-1)) * s
	hp := h / 60
	x := c * (1 - absFloat(float64(int(hp)%2)-1))
	var r, g, b float64
	switch int(hp) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := l - c/2
	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
