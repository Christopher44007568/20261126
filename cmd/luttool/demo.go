package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"luttool/lut"
	"luttool/render"
)

// nonUniformLUT samples a per-channel gamma curve over a shifted,
// per-channel domain, so misreading DOMAIN_MIN/MAX visibly shifts
// the result.
func nonUniformLUT(n int) *lut.LUT {
	min := [3]float64{0.10, 0.00, 0.20}
	max := [3]float64{0.90, 1.00, 0.60}
	l := lut.Build(n, min, max, func(r, g, b float64) (float64, float64, float64) {
		return math.Pow(r, 1.5), math.Pow(g, 1.2), math.Pow(b, 0.8)
	})
	l.Title = "nonuniform-domain gamma"
	return l
}

// runDemo renders the three comparison groups and the interpolation
// slice grids into dir, and writes the underlying .cube files so the
// same inputs can be replayed through `luttool apply`.
func runDemo(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cubeDir := filepath.Join(dir, "cubes")
	if err := os.MkdirAll(cubeDir, 0o755); err != nil {
		return err
	}

	identity := lut.Identity(17)
	identity.Title = "identity"
	swap := lut.SwapRB(17)
	swap.Title = "swap R<->B"
	nonuni := nonUniformLUT(17)

	for name, l := range map[string]*lut.LUT{
		"identity.cube":   identity,
		"swap_rb.cube":    swap,
		"nonuniform.cube": nonuni,
	} {
		f, err := os.Create(filepath.Join(cubeDir, name))
		if err != nil {
			return err
		}
		if err := l.Write(f); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	chart := render.TestImage(512, 256)

	// Group 1: identity LUT. Correct application must match the
	// source; the amplified difference map proves it.
	idMapped := render.Apply(chart, identity.Apply)
	g1 := render.HStack(6, chart, idMapped, render.Diff(chart, idMapped, 8))

	// Group 2: channel-swap LUT. Correct parse swaps R and B; the
	// wrong storage-order assumption produces a different image.
	swapOK := render.Apply(chart, swap.Apply)
	swapBad := render.Apply(chart, swap.ApplyWrongOrder)
	g2 := render.HStack(6, chart, swapOK, swapBad)

	// Group 3: non-uniform domain LUT. Correct domain mapping vs
	// the classic mistake of assuming a 0..1 unit scale.
	nuOK := render.Apply(chart, nonuni.Apply)
	nuBad := render.Apply(chart, nonuni.ApplyNoDomain)
	g3 := render.HStack(6, chart, nuOK, nuBad)

	groups := render.VStack(10, g1, g2, g3)
	if err := writePNGAtomic(filepath.Join(dir, "compare_groups.png"), groups); err != nil {
		return err
	}

	// Interpolation slices: lattice slices per blue node plus one
	// fractional-blue slice through the trilinear core, per LUT.
	slices := render.VStack(10,
		render.SliceGrid(identity, 48, 128),
		render.SliceGrid(swap, 48, 128),
		render.SliceGrid(nonuni, 48, 128),
	)
	if err := writePNGAtomic(filepath.Join(dir, "interp_slices.png"), slices); err != nil {
		return err
	}

	fmt.Printf("wrote %s/{compare_groups.png, interp_slices.png} and %s/*.cube\n", dir, cubeDir)
	return nil
}
