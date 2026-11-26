package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"hdrlab"
)

func main() {
	outDir := "artifacts"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}
	save := func(name string, im *hdrlab.Img) {
		if err := hdrlab.SaveImage(filepath.Join(outDir, name), im); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", name)
	}

	// 1. Synthetic bracket set: under / normal / over with handheld shifts
	//    and a moving car (ground truth known by construction).
	specs := hdrlab.BracketSpecs()
	scene := hdrlab.RenderScene()
	frames := make([]*hdrlab.Img, len(specs))
	ts := make([]float64, len(specs))
	for i, s := range specs {
		frames[i] = hdrlab.RenderFrame(scene, s, uint64(1000+i))
		ts[i] = s.Exposure()
		save("bracket_"+s.Name, frames[i])
	}

	// 2. Align to the normal-exposure reference frame.
	const ref = 1
	shifts := make([]hdrlab.Shift, len(specs))
	shifts[ref] = hdrlab.Shift{Confidence: 1}
	for i := range specs {
		if i == ref {
			continue
		}
		shifts[i] = hdrlab.EstimateShift(frames[ref], frames[i], ts[ref], ts[i], hdrlab.Margin)
		fmt.Printf("align %-6s estimated dx=%+d dy=%+d (true %+d/%+d) score=%.4f confidence=%.3f\n",
			specs[i].Name, shifts[i].DX, shifts[i].DY, specs[i].DX, specs[i].DY,
			shifts[i].Score, shifts[i].Confidence)
	}

	// 3. HDR merge with hat weights + reference-anchored deghost.
	merged := hdrlab.MergeHDR(frames, ts, shifts, ref)
	preview := hdrlab.Tonemap(merged.HDR)
	save("hdr_preview", preview)

	// 4. Baselines: direct and aligned naive averages.
	direct := hdrlab.AverageFrames(frames, shifts, false)
	aligned := hdrlab.AverageFrames(frames, shifts, true)
	save("direct_average", direct)
	save("aligned_average", aligned)

	// 5. Metering on the reference frame.
	metering := hdrlab.ComputeMetering(frames[ref], 12, 8)
	save("hdr_metering", hdrlab.BuildMeteringImage(preview, metering))

	// 6. Alignment-error heat map + ghost mask, and the combined overlay.
	clusters := hdrlab.FindClusters(merged.Ghost, hdrlab.OutW, hdrlab.OutH, 100)
	scale := hdrlab.ResidualScale(merged.Residual)
	save("alignment_error", hdrlab.BuildAlignmentErrorImage(preview, merged.Residual, merged.Ghost, clusters, specs, shifts, scale))
	save("hdr_overlay", hdrlab.BuildOverlayImage(preview, merged.Residual, clusters, metering, scale))

	// 7. Comparison sheet.
	save("comparison_sheet", hdrlab.BuildComparisonSheet(frames, direct, aligned, preview))

	// 8. Metrics + acceptance + JSON report. The highlight-edge metric
	//    covers only the billboard's interior detail (hot square + dark
	//    bar), where naive averages clip but HDR keeps edges.
	bill := hdrlab.Rect{
		MinX: hdrlab.Billboard.MinX - hdrlab.Margin + 12,
		MinY: hdrlab.Billboard.MinY - hdrlab.Margin + 12,
		MaxX: hdrlab.Billboard.MaxX - hdrlab.Margin - 12,
		MaxY: hdrlab.Billboard.MaxY - hdrlab.Margin - 12,
	}
	eHDR := hdrlab.EdgeEnergy(preview, bill)
	eDirect := hdrlab.EdgeEnergy(direct, bill)
	eAligned := hdrlab.EdgeEnergy(aligned, bill)
	ratio := eHDR / ((eDirect + eAligned) / 2)
	carRow := hdrlab.CarY - hdrlab.Margin
	refCrop := hdrlab.AverageFrames(frames[ref:ref+1], shifts[ref:ref+1], false)
	const edgeThresh = 0.10
	nHDR := hdrlab.CountEdges(preview, carRow, edgeThresh)
	nRef := hdrlab.CountEdges(refCrop, carRow, edgeThresh)
	nDirect := hdrlab.CountEdges(direct, carRow, edgeThresh)
	maxErr := 0
	for i, s := range specs {
		if i == ref {
			continue
		}
		if d := absInt(shifts[i].DX - s.DX); d > maxErr {
			maxErr = d
		}
		if d := absInt(shifts[i].DY - s.DY); d > maxErr {
			maxErr = d
		}
	}
	met := hdrlab.MetricsForReport(eHDR, eDirect, eAligned, ratio, nHDR, nRef, nDirect, merged)
	acc := hdrlab.AcceptanceForReport(maxErr, ratio, nHDR, nRef)
	files := map[string]string{}
	for _, n := range []string{
		"bracket_under", "bracket_normal", "bracket_over",
		"direct_average", "aligned_average", "hdr_preview",
		"hdr_metering", "hdr_overlay", "alignment_error", "comparison_sheet",
	} {
		files[n+".ppm"] = filepath.Join(outDir, n+".ppm")
		files[n+".png"] = filepath.Join(outDir, n+".png")
	}
	data, err := hdrlab.BuildReport(outDir, specs, shifts, metering, met, acc, files)
	if err != nil {
		log.Fatal(err)
	}
	if err := hdrlab.WriteReport(outDir, data); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote hdr_report.json")
	fmt.Printf("highlight edge energy: hdr=%.4f direct=%.4f aligned=%.4f ratio=%.3f\n", eHDR, eDirect, eAligned, ratio)
	fmt.Printf("object edges row %d: hdr=%d reference=%d direct=%d\n", carRow, nHDR, nRef, nDirect)
	fmt.Printf("motion fraction=%.4f mean disagreement=%.4f max shift error=%dpx\n",
		merged.MotionFraction, merged.MeanDisagreement, maxErr)
	fmt.Printf("acceptance: %+v\n", acc)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
