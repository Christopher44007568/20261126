package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// Verify re-reads the emitted manifest/images and asserts the non-negotiable
// properties: hashes match, reference errors are recorded, and missing
// references produce visible error colour rather than an empty scene.
func Verify(m *Manifest, artDir string) []string {
	var fails []string

	if !m.Impact.OnlyReferencingPathsUpdate {
		fails = append(fails, "impact.onlyReferencingPathsUpdate must be true")
	}
	if m.Before.Live.Material.Error != "" || m.Before.Archive.Material.Error != "" {
		fails = append(fails, "before stage must have all references resolved")
	}
	if !strings.Contains(m.After.Live.Material.Error, "has no kind") {
		fails = append(fails, "after live material must report the kind contract error")
	}
	if m.After.Archive.Material.Error != "" {
		fails = append(fails, "independent archive copy must stay healthy after live edit")
	}
	if !strings.Contains(m.Missing.Live.Geometry.Error, "no such file or directory") {
		fails = append(fails, "missing geometry must record the absent-file error")
	}
	if last := m.Missing.Live.Geometry.Chain[len(m.Missing.Live.Geometry.Chain)-1]; last.Exists {
		fails = append(fails, "last geometry chain hop must be marked exists=false")
	}
	if m.Missing.Archive.Geometry.Error != "" {
		fails = append(fails, "archive geometry must remain resolved in missing stage")
	}

	expect := map[string]struct {
		info       ImageInfo
		redAtLeast int
		redAtMost  int
	}{
		"before.png":  {m.Before.Image, 0, 400},
		"after.png":   {m.After.Image, 3000, 0},
		"missing.png": {m.Missing.Image, 2500, 0},
	}

	for name, want := range expect {
		full := filepath.Join(artDir, name)
		body, err := os.ReadFile(full)
		if err != nil {
			fails = append(fails, fmt.Sprintf("%s unreadable: %v", name, err))
			continue
		}
		sum := sha256.Sum256(body)
		if got := hex.EncodeToString(sum[:]); got != want.info.SHA256 {
			fails = append(fails, name+" sha256 does not match manifest")
		}
		if int64(len(body)) != want.info.Bytes {
			fails = append(fails, name+" byte count does not match manifest")
		}
		f, err := os.Open(full)
		if err != nil {
			fails = append(fails, fmt.Sprintf("%s open: %v", name, err))
			continue
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			fails = append(fails, fmt.Sprintf("%s decode: %v", name, err))
			continue
		}
		if b := img.Bounds(); b.Dx() != boardW || b.Dy() != boardH {
			fails = append(fails, fmt.Sprintf("%s must be %dx%d", name, boardW, boardH))
		}
		red := countErrorRed(img, image.Rect(44, 150, 528, 456))
		fmt.Printf("pixel check %-12s error-red pixels=%d\n", name, red)
		if want.redAtLeast > 0 && red < want.redAtLeast {
			fails = append(fails, fmt.Sprintf("%s must show a visible error overlay (red=%d)", name, red))
		}
		if want.redAtMost > 0 && red > want.redAtMost {
			fails = append(fails, fmt.Sprintf("%s should not show error red (red=%d)", name, red))
		}
	}
	return fails
}

// countErrorRed counts strong crimson pixels within r.
func countErrorRed(img image.Image, r image.Rectangle) int {
	count := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			rr, gg, bb, _ := img.At(x, y).RGBA()
			R, G, B := int(rr>>8), int(gg>>8), int(bb>>8)
			if R > 170 && G < 95 && B < 120 && R-G > 90 {
				count++
			}
		}
	}
	return count
}
