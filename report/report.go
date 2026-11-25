// Package report builds dependency-free PNG diagnostics for the autofocus
// engine. It uses only the Go standard library's image primitives.
package report

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"autofocus/autofocus"
)

// Frame is one captured frame plus the AF decision for that frame.
type Frame struct {
	Image     image.Image
	Frame     int
	Position  int
	State     autofocus.State
	Sharpness float64
	ROI       image.Rectangle
	MoveTo    int
}

// Data is the complete diagnostic bundle for one acquisition run.
type Data struct {
	Config   autofocus.Config
	Snapshot autofocus.Snapshot
	Frames   []Frame
}

// TenFrames selects exactly ten frames evenly from the run, preserving the
// beginning, search phase, and final locked state.
func TenFrames(frames []Frame) []Frame {
	if len(frames) == 0 {
		return nil
	}
	out := make([]Frame, 10)
	if len(frames) == 1 {
		for i := range out {
			out[i] = frames[0]
		}
		return out
	}
	for i := range out {
		idx := i * (len(frames) - 1) / (len(out) - 1)
		out[i] = frames[idx]
	}
	return out
}

// ContactSheet renders ten annotated thumbnails in a 5x2 grid.
func ContactSheet(frames []Frame, cfg autofocus.Config) *image.RGBA {
	selected := TenFrames(frames)
	const (
		thumbW = 240
		thumbH = 180
		gap    = 12
		labelH = 32
		header = 48
	)
	width := 5*thumbW + 6*gap
	height := header + 2*(thumbH+labelH) + 3*gap
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fill(img, img.Bounds(), colorBackground)
	drawText(img, gap, 14, "TEN FRAME CONTACT SHEET", 2, colorText)
	drawText(img, width-textWidth("ROI=GREEN  POSITION BAR=BLUE", 1)-gap, 17,
		"ROI=GREEN  POSITION BAR=BLUE", 1, colorMuted)

	for i := 0; i < 10; i++ {
		col, row := i%5, i/5
		x := gap + col*(thumbW+gap)
		y := header + gap + row*(thumbH+labelH+gap)
		if i >= len(selected) {
			fill(img, image.Rect(x, y, x+thumbW, y+thumbH), colorPanel)
			continue
		}
		f := selected[i]
		thumb := resize(f.Image, thumbW, thumbH)
		fill(img, image.Rect(x, y, x+thumbW, y+thumbH), colorPanel)
		copyAt(img, thumb, x, y)
		if !f.ROI.Empty() {
			roi := scaleRect(f.ROI, f.Image.Bounds(), thumbW, thumbH)
			roi = roi.Add(image.Pt(x, y))
			stroke(img, roi, colorGreen, 2)
		}
		drawText(img, x+2, y+3, fmt.Sprintf("F%03d", f.Frame), 1, colorText)

		label := fmt.Sprintf("P%02d  S%.3f", f.Position, f.Sharpness)
		drawText(img, x+2, y+thumbH+5, label, 2, colorText)
		drawFocusBar(img, image.Rect(x+6, y+thumbH+labelH-5, x+thumbW-6, y+thumbH+labelH-3),
			cfg.MinPosition, cfg.MaxPosition, f.Position, colorCyan)
	}
	return img
}

func drawFocusBar(img *image.RGBA, r image.Rectangle, minPos, maxPos, pos int, c color.RGBA) {
	drawLine(img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y, colorGrid)
	if maxPos <= minPos {
		return
	}
	x := r.Min.X + (pos-minPos)*(r.Dx()-1)/(maxPos-minPos)
	drawLine(img, x, r.Min.Y-3, x, r.Min.Y+3, c)
	drawLine(img, x-2, r.Min.Y-5, x+2, r.Min.Y-5, c)
}

func copyAt(dst *image.RGBA, src *image.RGBA, x, y int) {
	b := src.Bounds()
	for yy := 0; yy < b.Dy(); yy++ {
		for xx := 0; xx < b.Dx(); xx++ {
			dst.SetRGBA(x+xx, y+yy, src.RGBAAt(b.Min.X+xx, b.Min.Y+yy))
		}
	}
}

func scaleRect(r image.Rectangle, src image.Rectangle, w, h int) image.Rectangle {
	if src.Empty() {
		return image.Rectangle{}
	}
	x0 := (r.Min.X - src.Min.X) * w / src.Dx()
	y0 := (r.Min.Y - src.Min.Y) * h / src.Dy()
	x1 := (r.Max.X - src.Min.X) * w / src.Dx()
	y1 := (r.Max.Y - src.Min.Y) * h / src.Dy()
	return image.Rect(x0, y0, x1, y1)
}

// SavePNG writes an image and creates the parent directory if needed.
func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

// CurvePoint is an aggregated sharpness measurement at one focus position.
type CurvePoint struct {
	Position  int
	Sharpness float64
	Phase     string
}

// CurvePoints sorts samples by motor position and keeps the strongest sample
// when confirmation repeats the same position.
func CurvePoints(samples []autofocus.Sample) []CurvePoint {
	byPos := make(map[int]autofocus.Sample)
	for _, s := range samples {
		old, ok := byPos[s.Position]
		if !ok || s.Sharpness > old.Sharpness {
			byPos[s.Position] = s
		}
	}
	out := make([]CurvePoint, 0, len(byPos))
	for pos, s := range byPos {
		out = append(out, CurvePoint{Position: pos, Sharpness: s.Sharpness, Phase: s.Phase})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out
}
