// Command hdrexport renders a synthetic linear-light HDR scene containing
// highlights above SDR white, exports it as a floating-point PPM (PFM) and
// an 8-bit sRGB PNG preview with explicit color metadata, annotates a
// luminance scale and conversion notes, and verifies a PFM reload round trip.
// It uses only the Go standard library.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"hdrexport/internal/hdr"
	"hdrexport/internal/pfm"
	"hdrexport/internal/pngx"
	"hdrexport/internal/preview"
	"hdrexport/internal/reload"
	"hdrexport/internal/scene"
)

func main() {
	outDir := flag.String("out", "output", "output directory")
	width := flag.Int("w", 640, "scene width in pixels")
	height := flag.Int("h", 400, "scene height in pixels")
	exposure := flag.Float64("exposure", 1.0, "linear exposure scale before tone mapping")
	flag.Parse()

	if err := run(*outDir, *width, *height, *exposure); err != nil {
		fmt.Fprintln(os.Stderr, "hdrexport:", err)
		os.Exit(1)
	}
}

func run(outDir string, w, h int, exposure float64) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	fmt.Println(">> building linear-light HDR scene (float32, values may exceed 1.0)")
	original := scene.Build(w, h)
	stats := original.Compute()
	printStats(stats)

	pfmPath := filepath.Join(outDir, "scene.pfm")
	if err := writePFM(pfmPath, original); err != nil {
		return fmt.Errorf("write pfm: %w", err)
	}
	fmt.Println(">> exported float PPM:", pfmPath)

	ldr := preview.RenderLDR(original, exposure)
	peakCode := preview.MaxCodeOfScene(stats, exposure)
	annotated := preview.AnnotatedImage(ldr, stats, exposure, peakCode)

	pngPath := filepath.Join(outDir, "scene_preview.png")
	meta := pngx.SRGBMeta(preview.PNGNotes(stats, exposure, peakCode))
	if err := writePNG(pngPath, annotated.Pix, annotated.W, annotated.H, meta); err != nil {
		return fmt.Errorf("write png: %w", err)
	}
	fmt.Println(">> exported 8-bit sRGB PNG preview:", pngPath)

	fmt.Println(">> reloading PFM from disk and comparing before/after")
	reloaded, err := readPFM(pfmPath)
	if err != nil {
		return fmt.Errorf("reload pfm: %w", err)
	}
	metrics, beforeLDR, afterLDR := reload.Compare(original, reloaded, exposure)
	sheet := reload.BuildSheet(beforeLDR, afterLDR, w, h, metrics)
	sheetPath := filepath.Join(outDir, "reload_compare.png")
	if err := writePNG(sheetPath, sheet.Pix, sheet.W, sheet.H, pngx.SRGBMeta(nil)); err != nil {
		return fmt.Errorf("write compare sheet: %w", err)
	}
	fmt.Println(">> exported reload comparison:", sheetPath)

	reportPath := filepath.Join(outDir, "conversion_report.txt")
	report := buildReport(stats, exposure, peakCode, metrics, pfmPath, pngPath, sheetPath)
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		return err
	}
	fmt.Println(">> wrote conversion notes:", reportPath)
	fmt.Print(report)
	return nil
}

func writePFM(path string, m *hdr.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pfm.Encode(f, m)
}

func readPFM(path string) (*hdr.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return pfm.Decode(f)
}

func writePNG(path string, rgb []byte, w, h int, meta pngx.Meta) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pngx.Encode(f, rgb, w, h, 3, meta)
}

func printStats(s hdr.Stats) {
	fmt.Printf("   pixels=%d  peak lum=%.4f  peak channel=%.4f  mean lum=%.4f  over-white=%d\n",
		s.TotalPx, s.MaxLum, s.MaxChannel, s.MeanLum, s.OverWhite)
}
