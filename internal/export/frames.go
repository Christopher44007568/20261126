package export

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
)

func SaveJPEG(path string, img image.Image, quality int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}

func ContactSheetFromFiles(path string, paths []string, columns, cellW, cellH int) error {
	if len(paths) == 0 {
		return nil
	}
	rows := (len(paths) + columns - 1) / columns
	sheet := image.NewRGBA(image.Rect(0, 0, columns*cellW, rows*cellH))
	draw.Draw(sheet, sheet.Bounds(), image.White, image.Point{}, draw.Src)

	for i, item := range paths {
		f, err := os.Open(item)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(f)
		_ = f.Close()
		if err != nil {
			return err
		}
		col := i % columns
		row := i / columns
		dst := image.Rect(col*cellW, row*cellH, (col+1)*cellW, (row+1)*cellH)
		thumb := halfSize(img)
		draw.Draw(sheet, dst, thumb, image.Point{}, draw.Over)
		borderColor := colorRGBA(210, 205, 195)
		hLine(sheet, dst.Min.X, dst.Max.X-1, dst.Min.Y, borderColor)
		hLine(sheet, dst.Min.X, dst.Max.X-1, dst.Max.Y-1, borderColor)
		vLine(sheet, dst.Min.X, dst.Min.Y, dst.Max.Y-1, borderColor)
		vLine(sheet, dst.Max.X-1, dst.Min.Y, dst.Max.Y-1, borderColor)
	}
	return SaveJPEG(path, sheet, 84)
}

func halfSize(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	w := bounds.Dx() / 2
	h := bounds.Dy() / 2
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, a uint32
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					rr, gg, bb, aa := src.At(bounds.Min.X+x*2+dx, bounds.Min.Y+y*2+dy).RGBA()
					r += rr
					g += gg
					b += bb
					a += aa
				}
			}
			out.SetRGBA(x, y, color.RGBA{
				R: uint8((r / 4) >> 8),
				G: uint8((g / 4) >> 8),
				B: uint8((b / 4) >> 8),
				A: uint8((a / 4) >> 8),
			})
		}
	}
	return out
}

func colorRGBA(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
