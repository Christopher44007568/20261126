package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ValidationReport struct {
	GeneratedAt       string               `json:"generated_at"`
	AssetPath         string               `json:"asset_path"`
	OriginalPPM       string               `json:"original_ppm"`
	ReloadedPPM       string               `json:"reloaded_ppm"`
	DiffPPM           string               `json:"diff_ppm"`
	MismatchPPM       string               `json:"mismatch_ppm"`
	StructureOriginal string               `json:"structure_original_json"`
	StructureReloaded string               `json:"structure_reloaded_json"`
	Structure         StructureReport      `json:"structure"`
	Rays              RaySuiteReport       `json:"rays"`
	Diff              DiffStats            `json:"pixel_diff"`
	MismatchVisual    MismatchVisualReport `json:"mismatch_visual"`
	OK                bool                 `json:"ok"`
}

func main() {
	outDir := flag.String("out", "artifacts", "directory for AS binary, PPM images and reports")
	width := flag.Int("width", 640, "render width in pixels")
	height := flag.Int("height", 360, "render height in pixels")
	randomRays := flag.Int("random-rays", 512, "deterministic random equivalence rays")
	flag.Parse()

	if _, err := Generate(*outDir, *width, *height, *randomRays); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func Generate(outDir string, width, height, randomRays int) (*ValidationReport, error) {
	started := time.Now()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	if width < 16 || height < 9 {
		return nil, fmt.Errorf("image is too small: %dx%d", width, height)
	}

	scene := NewDemoScene()
	assetPath := filepath.Join(outDir, "accel.asb")
	if _, err := SaveAsset(assetPath, scene); err != nil {
		return nil, fmt.Errorf("save acceleration structure: %w", err)
	}

	rt, _, err := LoadAsset(assetPath)
	if err != nil {
		return nil, fmt.Errorf("load/relocate acceleration structure: %w", err)
	}

	structure := ValidateStructure(scene, rt)
	rays := VerifyRaySuite(scene, rt, randomRays)
	camera := NewCamera(width, height)

	originalRaw := RenderTrace(width, height, camera, func(ray Ray) Hit {
		return IntersectOriginalScene(scene, ray)
	})
	reloadedRaw := RenderTrace(width, height, camera, func(ray Ray) Hit {
		return IntersectRelocatedScene(rt, ray)
	})
	diffCanvas, diffStats := BuildDiff(originalRaw, reloadedRaw, 12)

	originalView := originalRaw.Clone()
	DrawOriginalBVHOverlay(scene, camera, originalView)
	reloadedView := reloadedRaw.Clone()
	DrawRelocatedBVHOverlay(rt, camera, reloadedView)
	DrawRoundTripBadge(originalView)
	DrawRoundTripBadge(reloadedView)
	mismatchCanvas, mismatchReport := BuildMismatchDiagnostic(reloadedRaw, rt, camera)

	paths := map[string]string{
		"original":   filepath.Join(outDir, "original.ppm"),
		"reloaded":   filepath.Join(outDir, "reloaded.ppm"),
		"diff":       filepath.Join(outDir, "diff.ppm"),
		"mismatch":   filepath.Join(outDir, "bounds_mismatch.ppm"),
		"origStruct": filepath.Join(outDir, "structure_original.json"),
		"relStruct":  filepath.Join(outDir, "structure_reloaded.json"),
		"json":       filepath.Join(outDir, "validation.json"),
		"text":       filepath.Join(outDir, "report.txt"),
	}
	for key, canvas := range map[string]*Canvas{
		"original": originalView,
		"reloaded": reloadedView,
		"diff":     diffCanvas,
		"mismatch": mismatchCanvas,
	} {
		if err := canvas.WritePPM(paths[key]); err != nil {
			return nil, err
		}
		fmt.Printf("wrote %s\n", paths[key])
	}

	originalDump := DumpOriginal(scene)
	reloadedDump := DumpRelocated(rt)
	if err := writeJSON(paths["origStruct"], originalDump); err != nil {
		return nil, err
	}
	if err := writeJSON(paths["relStruct"], reloadedDump); err != nil {
		return nil, err
	}

	report := &ValidationReport{
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		AssetPath:         assetPath,
		OriginalPPM:       paths["original"],
		ReloadedPPM:       paths["reloaded"],
		DiffPPM:           paths["diff"],
		MismatchPPM:       paths["mismatch"],
		StructureOriginal: paths["origStruct"],
		StructureReloaded: paths["relStruct"],
		Structure:         structure,
		Rays:              rays,
		Diff:              diffStats,
		MismatchVisual:    mismatchReport,
	}
	report.OK = structure.OK && rays.OK && diffStats.Identical &&
		mismatchReport.RedPixels > 0 && mismatchReport.GreenPixels > 0 && mismatchReport.YellowPixels > 0

	if err := writeJSON(paths["json"], report); err != nil {
		return nil, err
	}
	if err := os.WriteFile(paths["text"], []byte(renderTextReport(report, time.Since(started))), 0o644); err != nil {
		return nil, err
	}
	fmt.Printf("wrote %s\nwrote %s\n", paths["json"], paths["text"])

	if !report.OK {
		return report, fmt.Errorf("round-trip validation failed: structure=%v rays=%v identicalPixels=%v red/green/yellow=%d/%d/%d",
			structure.OK, rays.OK, diffStats.Identical,
			mismatchReport.RedPixels, mismatchReport.GreenPixels, mismatchReport.YellowPixels)
	}
	return report, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func renderTextReport(r *ValidationReport, elapsed time.Duration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "加速结构写盘 / 重定位 / 重新求交报告\n")
	fmt.Fprintf(&b, "====================================\n\n")
	fmt.Fprintf(&b, "总状态: %v\n", okText(r.OK))
	fmt.Fprintf(&b, "耗时: %s\n", elapsed.Round(time.Millisecond))
	fmt.Fprintf(&b, "二进制: %s (CRC32 %s)\n\n", r.AssetPath, r.Structure.Checksum)

	fmt.Fprintf(&b, "结构重定位\n")
	fmt.Fprintf(&b, "- 节点数: %d；实例数: %d；三角形数: %d\n", r.Structure.NodeCount, r.Structure.InstanceCount, r.Structure.TriangleCount)
	fmt.Fprintf(&b, "- 根节点地址: %s -> %s\n", r.Structure.OriginalRootAddress, r.Structure.RelocatedRootAddress)
	fmt.Fprintf(&b, "- 地址变化节点: %d/%d\n", r.Structure.NodeAddressChanges, r.Structure.NodeCount)
	fmt.Fprintf(&b, "- 三角形存储顺序已变化: %v；前 8 个重载原 ID: %v\n",
		r.Structure.TriangleOrderChanged, firstN(r.Structure.RelocatedOrderFirst, 8))
	fmt.Fprintf(&b, "- AABB 错位: %d；拓扑错误: %d；叶索引错误: %d\n",
		len(r.Structure.BoundsMismatches), len(r.Structure.TopologyMismatches), len(r.Structure.LeafOrderMismatches))
	fmt.Fprintf(&b, "- 退化三角形: %d；最深实例路径长度: %d\n\n", r.Structure.DegenerateTriangles, r.Structure.DeepInstancePathLength)

	fmt.Fprintf(&b, "重新求交\n")
	fmt.Fprintf(&b, "- 专项射线: %d，通过: %d，失败: %d\n", r.Rays.ProbeCount, r.Rays.ProbePassed, r.Rays.ProbeFailed)
	fmt.Fprintf(&b, "- 随机世界射线: %d，不一致: %d\n", r.Rays.RandomRayCount, r.Rays.RandomMismatches)
	fmt.Fprintf(&b, "- 最大 t 误差: %.3g；最大点误差: %.3g\n\n", r.Rays.MaxTError, r.Rays.MaxPointError)

	fmt.Fprintf(&b, "PPM 差分\n")
	fmt.Fprintf(&b, "- %s vs %s\n", r.OriginalPPM, r.ReloadedPPM)
	fmt.Fprintf(&b, "- 不同像素: %d/%d；最大亮度差: %.3g；完全一致: %v\n",
		r.Diff.DifferentPixels, r.Diff.PixelCount, r.Diff.MaxLuma, r.Diff.Identical)
	fmt.Fprintf(&b, "- 差分文件: %s\n\n", r.DiffPPM)

	fmt.Fprintf(&b, "错位包围盒颜色（%s）\n", r.MismatchPPM)
	fmt.Fprintf(&b, "- 绿色: %s\n", r.MismatchVisual.GreenMeaning)
	fmt.Fprintf(&b, "- 红色: %s\n", r.MismatchVisual.RedMeaning)
	fmt.Fprintf(&b, "- 黄色: %s\n", r.MismatchVisual.YellowMeaning)
	fmt.Fprintf(&b, "- 诊断盒数量: %d；红/绿/黄像素计数: %d/%d/%d\n\n",
		len(r.MismatchVisual.Boxes), r.MismatchVisual.RedPixels, r.MismatchVisual.GreenPixels, r.MismatchVisual.YellowPixels)

	if r.Rays.Failed > 0 {
		fmt.Fprintf(&b, "失败射线（最多 12 条）\n")
		shown := 0
		for _, cmp := range r.Rays.Comparisons {
			if cmp.OK {
				continue
			}
			fmt.Fprintf(&b, "- [%s] %s: %s\n", cmp.Mode, cmp.Name, cmp.Error)
			shown++
			if shown >= 12 {
				break
			}
		}
	}
	return b.String()
}

func okText(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func firstN(values []int, n int) []int {
	if len(values) <= n {
		return values
	}
	return values[:n]
}
