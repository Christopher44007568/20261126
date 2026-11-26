package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ImageReport struct {
	Tier             Tier      `json:"tier"`
	File             string    `json:"file"`
	Generation       uint64    `json:"generation"`
	BudgetBytes      int       `json:"budget_bytes"`
	ResidentBytes    int       `json:"resident_bytes"`
	Nodes            int       `json:"nodes"`
	Triangles        int       `json:"triangles"`
	Fingerprint      string    `json:"fingerprint"`
	Diff             DiffStats `json:"diff_vs_full"`
	ErrorBorderDrawn bool      `json:"error_border_drawn"`
}

type StaleCheck struct {
	OldGeneration uint64 `json:"old_generation"`
	NewGeneration uint64 `json:"new_generation"`
	Rejected      bool   `json:"rejected"`
	Error         string `json:"error,omitempty"`
}

type RestoreReport struct {
	Generation           uint64 `json:"generation"`
	PixelIdenticalToFull bool   `json:"pixel_identical_to_full"`
	FullFingerprint      string `json:"full_fingerprint"`
	RestoredFingerprint  string `json:"restored_fingerprint"`
	RebuildCount         int    `json:"full_rebuild_count"`
}

type Report struct {
	GeneratedAtUTC string            `json:"generated_at_utc"`
	Width          int               `json:"width"`
	Height         int               `json:"height"`
	Detail         int               `json:"detail_segments"`
	LowDetail      int               `json:"low_detail_segments"`
	Budgets        BudgetPlan        `json:"budgets"`
	Outputs        []ImageReport     `json:"outputs"`
	StaleChecks    []StaleCheck      `json:"stale_checks"`
	Restore        RestoreReport     `json:"restore"`
	EvictionCount  int               `json:"eviction_count"`
	BuildCounts    map[Tier]int      `json:"build_counts"`
	Transitions    []Transition      `json:"transitions"`
	Legend         map[string]string `json:"legend"`
}

func main() {
	outDir := flag.String("out", "artifacts", "directory for PPM images and report.json")
	width := flag.Int("width", 480, "render width in pixels")
	height := flag.Int("height", 360, "render height in pixels")
	detail := flag.Int("detail", 80, "angular segments for full-quality crest")
	lowDetail := flag.Int("low-detail", 16, "angular segments for low-poly fallback")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}

	camera := DefaultCamera(*width, *height)
	detailed := NewCrest(*detail, "full-detail-crest")
	coarse := NewCrest(*lowDetail, "low-poly-proxy")

	plan, err := NewBudgetPlan(detailed, coarse)
	if err != nil {
		log.Fatalf("budget plan: %v", err)
	}
	manager := NewManager(detailed, coarse, *plan)

	fullHandle, err := manager.SetBudget(plan.High.LimitBytes, "initial high VRAM budget")
	if err != nil {
		log.Fatalf("select full tier: %v", err)
	}
	fullResult, err := manager.RenderCurrent(camera)
	if err != nil {
		log.Fatalf("render full: %v", err)
	}

	compressedHandle, err := manager.SetBudget(plan.Medium.LimitBytes, "medium budget: evict full BVH and use quantized nodes")
	if err != nil {
		log.Fatalf("select compressed tier: %v", err)
	}
	staleAfterCompress := checkStale(manager, fullHandle, compressedHandle, camera)
	compressedResult, err := manager.RenderCurrent(camera)
	if err != nil {
		log.Fatalf("render compressed: %v", err)
	}
	compressedDiff := AnnotateSilhouetteErrors(compressedResult.Image, fullResult.Mask, compressedResult.Mask, 2)
	DrawTierFrame(compressedResult.Image, RGB{255, 176, 64}, 5)

	fallbackHandle, err := manager.SetBudget(plan.Low.LimitBytes, "low budget: evict compressed AS and high-detail mesh")
	if err != nil {
		log.Fatalf("select fallback tier: %v", err)
	}
	staleAfterFallback := checkStale(manager, compressedHandle, fallbackHandle, camera)
	fallbackResult, err := manager.RenderCurrent(camera)
	if err != nil {
		log.Fatalf("render fallback: %v", err)
	}
	fallbackDiff := AnnotateSilhouetteErrors(fallbackResult.Image, fullResult.Mask, fallbackResult.Mask, 3)
	DrawTierFrame(fallbackResult.Image, RGB{255, 83, 89}, 6)

	restoredHandle, err := manager.RestoreFull("VRAM budget restored: rebuild, do not reuse old nodes")
	if err != nil {
		log.Fatalf("restore full: %v", err)
	}
	staleAfterRestore := checkStale(manager, fallbackHandle, restoredHandle, camera)
	restoredResult, err := manager.RenderCurrent(camera)
	if err != nil {
		log.Fatalf("render restored: %v", err)
	}
	identical := imagesIdentical(fullResult.Image, restoredResult.Image)
	if !identical {
		log.Fatalf("restored image does not match full-quality reference")
	}

	fullForWrite := CloneImage(fullResult.Image)
	DrawTierFrame(fullForWrite, RGB{82, 220, 116}, 5)
	restoredForWrite := CloneImage(restoredResult.Image)
	DrawTierFrame(restoredForWrite, RGB{82, 220, 116}, 5)

	outputs := []struct {
		name   string
		img    *Image
		handle *Handle
		diff   DiffStats
		border bool
	}{
		{"full.ppm", fullForWrite, fullHandle, DiffStats{}, false},
		{"compressed.ppm", compressedResult.Image, compressedHandle, compressedDiff, compressedDiff.BorderPixels > 0},
		{"fallback.ppm", fallbackResult.Image, fallbackHandle, fallbackDiff, fallbackDiff.BorderPixels > 0},
		{"restored.ppm", restoredForWrite, restoredHandle, DiffStats{}, false},
	}

	var imageReports []ImageReport
	for _, out := range outputs {
		path := filepath.Join(*outDir, out.name)
		if err := WritePPM(path, out.img); err != nil {
			log.Fatalf("write %s: %v", path, err)
		}
		pkg := out.handle.pkg
		imageReports = append(imageReports, ImageReport{
			Tier: out.handle.Tier, File: path, Generation: out.handle.Generation,
			BudgetBytes: out.handle.BudgetBytes, ResidentBytes: pkg.Bytes,
			Nodes: pkg.Nodes, Triangles: pkg.Triangles,
			Fingerprint: fmt.Sprintf("%016x", Fingerprint(out.img)),
			Diff:        out.diff, ErrorBorderDrawn: out.border,
		})
	}

	report := Report{
		GeneratedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Width:          *width, Height: *height, Detail: *detail, LowDetail: *lowDetail,
		Budgets: *plan, Outputs: imageReports,
		StaleChecks: []StaleCheck{staleAfterCompress, staleAfterFallback, staleAfterRestore},
		Restore: RestoreReport{
			Generation: restoredHandle.Generation, PixelIdenticalToFull: identical,
			FullFingerprint:     fmt.Sprintf("%016x", Fingerprint(fullResult.Image)),
			RestoredFingerprint: fmt.Sprintf("%016x", Fingerprint(restoredResult.Image)),
			RebuildCount:        manager.BuildCounts[TierFull],
		},
		EvictionCount: manager.EvictionCount, BuildCounts: manager.BuildCounts,
		Transitions: manager.Transitions,
		Legend: map[string]string{
			"green_frame": "full budget / rebuilt clear image",
			"amber_frame": "compressed acceleration structure",
			"red_frame":   "low-poly fallback",
			"magenta":     "silhouette coverage boundary affected by downgrade",
			"yellow_core": "exact degraded/reference contour edge",
		},
	}
	reportPath := filepath.Join(*outDir, "report.json")
	if err := writeJSON(reportPath, report); err != nil {
		log.Fatalf("write report: %v", err)
	}

	fmt.Printf("wrote 4 PPM artifacts (3 required tiers + restored check) and %s\n", reportPath)
	for _, imgReport := range imageReports {
		fmt.Printf("%-18s tier=%-18s gen=%d resident=%d/%d bytes diff=%d border=%d\n",
			filepath.Base(imgReport.File), imgReport.Tier, imgReport.Generation,
			imgReport.ResidentBytes, imgReport.BudgetBytes,
			imgReport.Diff.DifferentPixels, imgReport.Diff.BorderPixels)
	}
	fmt.Printf("restored identical=%v stale_checks=%d evictions=%d\n",
		identical, len(report.StaleChecks), manager.EvictionCount)
}

func checkStale(manager *Manager, old *Handle, current *Handle, camera Camera) StaleCheck {
	_, err := manager.Render(old, camera)
	check := StaleCheck{OldGeneration: old.Generation, NewGeneration: current.Generation}
	if err != nil && errors.Is(err, ErrStaleHandle) {
		check.Rejected = true
		check.Error = err.Error()
	} else if err != nil {
		check.Error = err.Error()
	}
	return check
}

func CloneImage(src *Image) *Image {
	dst := NewImage(src.Width, src.Height)
	copy(dst.Pixels, src.Pixels)
	return dst
}

func imagesIdentical(a, b *Image) bool {
	if a.Width != b.Width || a.Height != b.Height || len(a.Pixels) != len(b.Pixels) {
		return false
	}
	for i := range a.Pixels {
		if a.Pixels[i] != b.Pixels[i] {
			return false
		}
	}
	return true
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
