package vt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MetricPoint is one labeled camera/feedback state.
type MetricPoint struct {
	Stage               string  `json:"stage"`
	Frame               int     `json:"frame"`
	ExactPixels         int     `json:"exact_pixels"`
	FallbackPixels      int     `json:"fallback_pixels"`
	MissingPixels       int     `json:"missing_pixels"`
	PixelMissRate       float64 `json:"pixel_miss_rate"`
	FeedbackMissRate    float64 `json:"feedback_miss_rate"`
	ResidentPages       int     `json:"resident_pages"`
	CacheCapacity       int     `json:"cache_capacity"`
	InflightRequests    int     `json:"inflight_requests"`
	CumulativeEvictions int     `json:"cumulative_evictions"`
}

func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// WriteMetricsSVG draws miss rate and cache occupancy without a graphics SDK.
func WriteMetricsSVG(path string, points []MetricPoint) error {
	const width, height = 760, 380
	const left, top, right, bottom = 64, 28, 28, 58
	plotW := width - left - right
	plotH := height - top - bottom
	maxCache := 1
	for _, p := range points {
		if p.CacheCapacity > maxCache {
			maxCache = p.CacheCapacity
		}
	}

	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="760" height="380" viewBox="0 0 760 380" font-family="sans-serif">`)
	b.WriteString(`<rect width="760" height="380" fill="#0f172a"/>`)
	b.WriteString(`<text x="64" y="22" fill="#e2e8f0" font-size="16" font-weight="700">Virtual texture feedback and replacement</text>`)

	for tick := 0; tick <= 4; tick++ {
		y := top + plotH - int(float64(tick)/4*float64(plotH))
		b.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#334155" stroke-width="1"/>`, left, y, left+plotW, y))
		b.WriteString(fmt.Sprintf(`<text x="20" y="%d" fill="#94a3b8" font-size="12">%d%%</text>`, y+4, tick*25))
	}

	if len(points) > 1 {
		missPath := make([]string, 0, len(points))
		cachePath := make([]string, 0, len(points))
		for i, p := range points {
			x := left + int(float64(i)/float64(len(points)-1)*float64(plotW))
			my := top + plotH - int(p.PixelMissRate*float64(plotH))
			cy := top + plotH - int(float64(p.ResidentPages)/float64(maxCache)*float64(plotH))
			missPath = append(missPath, fmt.Sprintf("%d,%d", x, my))
			cachePath = append(cachePath, fmt.Sprintf("%d,%d", x, cy))
			b.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="5" fill="#ef4444"/>`, x, my))
			b.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="4" fill="#38bdf8"/>`, x, cy))
			b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="#cbd5e1" font-size="11" text-anchor="middle">%s</text>`, x, height-34, p.Stage))
			b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="#fca5a5" font-size="11" text-anchor="middle">%.0f%%</text>`, x, my-10, p.PixelMissRate*100))
		}
		b.WriteString(`<polyline points="` + strings.Join(missPath, " ") + `" fill="none" stroke="#ef4444" stroke-width="3"/>`)
		b.WriteString(`<polyline points="` + strings.Join(cachePath, " ") + `" fill="none" stroke="#38bdf8" stroke-width="3" stroke-dasharray="7 5"/>`)
	}
	b.WriteString(`<rect x="540" y="12" width="12" height="12" fill="#ef4444"/><text x="558" y="23" fill="#e2e8f0" font-size="12">exact-page miss</text>`)
	b.WriteString(`<rect x="660" y="12" width="12" height="12" fill="#38bdf8"/><text x="678" y="23" fill="#e2e8f0" font-size="12">resident/cap</text>`)
	b.WriteString(`<line x1="64" y1="322" x2="700" y2="322" stroke="#64748b" stroke-width="1"/>`)
	b.WriteString(`<text x="64" y="350" fill="#94a3b8" font-size="12">Move viewport → parent mip → partial async arrivals → all visible pages resident</text>`)
	b.WriteString(`</svg>`)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
