package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	outDir := "out"
	must(os.MkdirAll(filepath.Join(outDir, "frames"), 0o755))
	var report strings.Builder
	logf := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		fmt.Println(line)
		report.WriteString(line + "\n")
	}

	// Scene: forest split into many independent BLAS.
	soups := buildForest(220, 7)
	scene := &Scene{}
	totalTris := 0
	for _, soup := range soups {
		scene.BLAS = append(scene.BLAS, BuildBVH(soup))
		totalTris += len(soup)
	}
	boxes := make([][2]Vec3, len(scene.BLAS))
	for i, b := range scene.BLAS {
		boxes[i] = [2]Vec3{b.Nodes[0].Min, b.Nodes[0].Max}
	}
	scene.TLAS = BuildTLAS(boxes)

	// Memory before compression.
	var nodeBytes, idxBytes int64
	totalNodes := 0
	for _, b := range scene.BLAS {
		nodeBytes += int64(len(b.Nodes)) * NodeBytes
		idxBytes += int64(len(b.Indices)) * 4
		totalNodes += len(b.Nodes)
	}
	logf("=== Forest BLAS compression report ===")
	logf("BLAS count: %d (1 ground + %d trees), triangles: %d, BVH nodes: %d",
		len(scene.BLAS), len(scene.BLAS)-1, totalTris, totalNodes)
	logf("[before] node memory: %d B (%d x %dB), index memory: %d B (%d x 4B), total: %d B",
		nodeBytes, totalNodes, NodeBytes, idxBytes, totalTris, nodeBytes+idxBytes)

	// Compress every BLAS and validate all remapped references.
	scene.CBLAS = make([]*CBVH, len(scene.BLAS))
	var cNodeBytes int64
	invalid := 0
	for i, b := range scene.BLAS {
		cb := Compress(b)
		scene.CBLAS[i] = cb
		cNodeBytes += int64(len(cb.Nodes))*CNodeBytes + 24
		bad := cb.Validate()
		invalid += len(bad)
		for _, r := range bad[:min(3, len(bad))] {
			logf("  BLAS %d invalid ref: node %d %s", i, r.Node, r.Kind)
		}
	}
	logf("[after]  node memory: %d B (%d x %dB + grid), index memory: 0 B (triangles reordered), total: %d B",
		cNodeBytes, totalNodes, CNodeBytes, cNodeBytes)
	saved := 100 * (1 - float64(cNodeBytes)/float64(nodeBytes+idxBytes))
	logf("saved: %.1f%% of node+index memory", saved)
	logf("validation: %d invalid references across all compressed BLAS", invalid)

	// Corruption self-test: prove the checker catches bad references.
	probe := *scene.CBLAS[1]
	probe.Nodes = append([]CNode(nil), scene.CBLAS[1].Nodes...)
	for i := range probe.Nodes {
		if probe.Nodes[i].Meta&leafFlag != 0 {
			probe.Nodes[i].Meta = leafFlag | 1<<16 | 0xffff
			break
		}
	}
	probe.Nodes[len(probe.Nodes)/2].Meta = 0x7ffffff0
	caught := len(probe.Validate())
	logf("corruption self-test: injected 2 faults, checker caught %d", caught)

	// Beauty render: before vs after compression.
	cam := Camera{Pos: Vec3{16.2, 9.5, 25.2}, Look: Vec3{0, 2.5, 0}, Up: Vec3{0, 1, 0}, FovYDeg: 55}
	before := Render(scene, cam, 480, 360, false, false)
	after := Render(scene, cam, 480, 360, true, false)
	diff, maxd := DiffImage(before, after, 20)
	idMismatch := 0
	for i := range before.IDs {
		if before.IDs[i] != after.IDs[i] {
			idMismatch++
		}
	}
	invImg := Render(scene, cam, 480, 360, true, true)
	invHL, invPx := HighlightInvalid(invImg)
	must(before.SavePPM(filepath.Join(outDir, "before.ppm")))
	must(after.SavePPM(filepath.Join(outDir, "after.ppm")))
	must(diff.SavePPM(filepath.Join(outDir, "diff.ppm")))
	must(invHL.SavePPM(filepath.Join(outDir, "invalid.ppm")))
	logf("beauty 480x360: max|diff|=%g (x20 amplified in diff.ppm), triangle-ID mismatches=%d, invalid-ref pixels=%d",
		maxd, idMismatch, invPx)

	// Highlight demo: render with the corrupted BLAS to show red flagging.
	demo := *scene
	demo.CBLAS = append([]*CBVH(nil), scene.CBLAS...)
	demo.CBLAS[1] = &probe
	demoImg := Render(&demo, cam, 480, 360, true, true)
	demoHL, demoPx := HighlightInvalid(demoImg)
	must(demoHL.SavePPM(filepath.Join(outDir, "invalid_demo.ppm")))
	logf("invalid highlight demo: %d pixels flagged red in invalid_demo.ppm", demoPx)

	// Continuous camera path: compression must not introduce flicker.
	const frames = 12
	const fw, fh = 288, 216
	logf("camera path: %d frames %dx%d", frames, fw, fh)
	var prevIDs []int32
	allExact := true
	for f := 0; f < frames; f++ {
		ang := 0.6 + 0.8*float64(f)/(frames-1)
		pos := Vec3{30 * float32(math.Cos(ang)), 9 + 1.2*float32(math.Sin(3*ang)), 30 * float32(math.Sin(ang))}
		fcam := Camera{Pos: pos, Look: Vec3{0, 2.5, 0}, Up: Vec3{0, 1, 0}, FovYDeg: 55}
		fa := Render(scene, fcam, fw, fh, true, false)
		fb := Render(scene, fcam, fw, fh, false, false)
		mism := 0
		for i := range fa.IDs {
			if fa.IDs[i] != fb.IDs[i] || fa.Pix[i] != fb.Pix[i] {
				mism++
			}
		}
		if mism != 0 {
			allExact = false
		}
		temporal := 0
		if prevIDs != nil {
			for i := range fa.IDs {
				if fa.IDs[i] != prevIDs[i] {
					temporal++
				}
			}
		}
		prevIDs = fa.IDs
		must(fa.SavePPM(filepath.Join(outDir, "frames", fmt.Sprintf("frame_%02d.ppm", f))))
		logf("  frame %02d: compressed-vs-uncompressed mismatches=%d, temporal ID changes=%d (%.1f%%)",
			f, mism, temporal, 100*float64(temporal)/float64(fw*fh))
	}
	if allExact {
		logf("flicker check: PASS - every frame bit-identical, compression introduces zero flicker")
	} else {
		logf("flicker check: FAIL - compressed output diverges from reference")
	}
	must(os.WriteFile(filepath.Join(outDir, "report.txt"), []byte(report.String()), 0o644))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
