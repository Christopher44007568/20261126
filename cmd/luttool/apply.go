package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"luttool/lut"
	"luttool/render"
)

// loadLUT parses path and rewrites parse errors to carry the file
// name together with the 1-based line number.
func loadLUT(path string) (*lut.LUT, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	l, err := lut.Parse(f)
	if err != nil {
		var pe *lut.Error
		if errors.As(err, &pe) && pe.Line > 0 {
			return nil, fmt.Errorf("%s:%d: %s", path, pe.Line, pe.Msg)
		}
		return nil, fmt.Errorf("%s: %s", path, pe.Msg)
	}
	return l, nil
}

// writePNGAtomic writes img to a temp file next to path and renames
// it, so a failed run never clobbers the last valid output.
func writePNGAtomic(path string, img image.Image) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".luttool-*.png")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := png.Encode(tmp, img); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// runApply renders the test chart mapped through the LUT in src,
// stacked above the interpolation slice grid, and writes dst. On any
// parse failure dst is left untouched (the previous valid image is
// kept) and the error names the offending line.
func runApply(src, dst string) error {
	l, err := loadLUT(src)
	if err != nil {
		return err
	}
	chart := render.TestImage(512, 256)
	mapped := render.Apply(chart, l.Apply)
	top := render.HStack(6, chart, mapped)
	slices := render.SliceGrid(l, 64, 160)
	sheet := render.VStack(8, top, slices)
	if err := writePNGAtomic(dst, sheet); err != nil {
		return err
	}
	fmt.Printf("ok: %s -> %s (size %d, domain %v..%v)\n", src, dst, l.Size, l.Min, l.Max)
	return nil
}
