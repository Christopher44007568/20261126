package demo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"atlaspad/atlas"
)

func WriteMarkdownReport(path string, tight, safe *atlas.BuiltAtlas, tightReport, safeReport atlas.ValidationReport, u atlas.Utilization) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Atlas padding / rotation / mip validation\n\n")
	fmt.Fprintf(&b, "- Atlas: `%dx%d`, mip levels: `%d`\n", safe.Layout.Width, safe.Layout.Height, len(safe.Mips))
	fmt.Fprintf(&b, "- Pack padding: `%d` level-zero pixels per side\n", safe.Layout.PackPadding)
	fmt.Fprintf(&b, "- Rotated placements: `%d/%d`\n", rotatedCount(safe.Layout), len(safe.Layout.Placements))
	fmt.Fprintf(&b, "- Texture utilization: `%.2f%%`\n", u.TextureUtilization*100)
	fmt.Fprintf(&b, "- Cell utilization including padding: `%.2f%%`\n", u.CellUtilization*100)
	fmt.Fprintf(&b, "- Packing efficiency inside cells: `%.2f%%`\n\n", u.PackingEfficiency*100)

	fmt.Fprintf(&b, "## Runtime sampling\n\n")
	fmt.Fprintf(&b, "| Atlas | Samples | Violations | Max RGBA distance | Result |\n")
	fmt.Fprintf(&b, "|---|---:|---:|---:|---|\n")
	fmt.Fprintf(&b, "| tight/no padding | %d | %d | %.2f | %s |\n", tightReport.SampleCount, tightReport.ViolationCount, tightReport.MaxDistance, passLabel(tightReport.OK))
	fmt.Fprintf(&b, "| mip-expanded padding | %d | %d | %.2f | %s |\n\n", safeReport.SampleCount, safeReport.ViolationCount, safeReport.MaxDistance, passLabel(safeReport.OK))

	fmt.Fprintf(&b, "## Padding by mip\n\n")
	fmt.Fprintf(&b, "Padding policy starts at `%d` base texel(s). Required level-zero guard width doubles at every coarser mip; the packer reserves the maximum once.\n\n", safe.Layout.Policy.Padding)
	fmt.Fprintf(&b, "| Mip | Size | Required base px/side | Reserved pack px/side | Samples | Violations | Max error |\n")
	fmt.Fprintf(&b, "|---:|---|---:|---:|---:|---:|---:|\n")
	for i, stat := range safeReport.Levels {
		fmt.Fprintf(&b, "| %d | %dx%d | %d | %d | %d | %d | %.2f |\n",
			i, stat.Width, stat.Height, stat.RequiredPadding, stat.PackPadding,
			stat.Samples, stat.Violations, stat.MaxDistance)
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "## Placements and UVs\n\n")
	fmt.Fprintf(&b, "| ID | Rotated | Frame | Cell | UV u0,v0,u1,v1 |\n")
	fmt.Fprintf(&b, "|---|---:|---|---|---|\n")
	for _, p := range safe.Layout.Placements {
		u0, v0, u1, v1 := p.UVBounds(safe.Layout.Width, safe.Layout.Height)
		fmt.Fprintf(&b, "| `%s` | %v | `%d,%d %dx%d` | `%d,%d %dx%d` | `%.5f,%.5f,%.5f,%.5f` |\n",
			p.ID, p.Rotated, p.Frame.X, p.Frame.Y, p.Frame.W, p.Frame.H,
			p.Cell.X, p.Cell.Y, p.Cell.W, p.Cell.H, u0, v0, u1, v1)
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "## Output files\n\n")
	files := []string{
		"atlas_layout.png", "atlas_safe.png", "atlas_tight.png",
		"bleed_highlight.png", "bleed_highlight_safe.png", "bleed_comparison.png",
		"mip_padding.png", "utilization.png", "utilization.json",
		"validation_tight.json", "validation_safe.json", "layout.json", "result.json",
	}
	for _, f := range files {
		fmt.Fprintf(&b, "- `%s`\n", f)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func rotatedCount(l *atlas.Layout) int {
	n := 0
	for _, p := range l.Placements {
		if p.Rotated {
			n++
		}
	}
	return n
}

func passLabel(ok bool) string {
	if ok {
		return "PASS"
	}
	return "BLEED"
}
