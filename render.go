package main

import (
	"fmt"
	"math"
)

const (
	frameW = 640
	frameH = 480
)

var (
	colorBackground = rgb(15, 23, 42)
	colorPanel      = rgb(30, 41, 59)
	colorViewport   = rgb(10, 18, 33)
	colorInk        = rgb(226, 232, 240)
	colorMuted      = rgb(148, 163, 184)
	colorExact      = rgb(34, 197, 94)
	colorLoading    = rgb(245, 158, 11)
	colorCoarse     = rgb(56, 189, 248)
	colorRoot       = rgb(129, 140, 248)
)

type FrameMetrics struct {
	Strategy       Strategy `json:"strategy"`
	Detail         string   `json:"detail"`
	IdealLOD       int      `json:"idealLod"`
	MeshPixels     int      `json:"meshPixels"`
	ExactPixels    int      `json:"exactPixels"`
	LoadingPixels  int      `json:"loadingPixels"`
	CoarsePixels   int      `json:"coarsePixels"`
	RootPixels     int      `json:"rootPixels"`
	BytesLoaded    int      `json:"bytesLoaded"`
	MeanError      float64  `json:"meanScreenErrorPx"`
	TargetCoverage float64  `json:"targetCoveragePct"`
}

type RenderedFrame struct {
	Image   *Image
	Metrics FrameMetrics
	Mesh    []bool
}

func mainViewport() Viewport {
	return Viewport{X: 28, Y: 76, Size: 360}
}

func RenderFrame(loader *Loader, cam Camera, detail string) RenderedFrame {
	vp := mainViewport()
	im := NewImage(frameW, frameH, colorBackground)
	mesh := make([]bool, frameW*frameH)
	metrics := FrameMetrics{
		Strategy:    loader.Kind,
		Detail:      detail,
		IdealLOD:    idealLOD(cam),
		BytesLoaded: loader.BytesLoaded,
	}

	drawChrome(im, loader.Kind, detail, cam)
	im.FillRect(vp.X-4, vp.Y-4, vp.Size+8, vp.Size+8, rgb(51, 65, 85))
	im.FillRect(vp.X, vp.Y, vp.Size, vp.Size, colorViewport)

	for py := vp.Y; py < vp.Y+vp.Size; py++ {
		for px := vp.X; px < vp.X+vp.Size; px++ {
			uv := cam.UVAt(px, py, vp)
			if uv.U < 0 || uv.U > 1 || uv.V < 0 || uv.V > 1 {
				continue
			}
			info := loader.Inspect(uv, cam)
			count := 1 << info.ResolvedLevel
			tx := tileIndex(uv.U, count)
			ty := tileIndex(uv.V, count)
			localU := uv.U*float64(count) - float64(tx)
			localV := uv.V*float64(count) - float64(ty)
			base := info.Tile.sample(localU, localV)
			im.Set(px, py, base)
			applyStatusTint(im, px, py, info.Fallback)
			if info.Loading {
				im.Blend(px, py, colorLoading.withAlpha(22))
				if (px+py)%14 < 6 {
					im.Blend(px, py, colorLoading.withAlpha(116))
				}
			}

			mesh[py*frameW+px] = true
			metrics.MeshPixels++
			metrics.MeanError += info.ErrorPixels
			if info.Loading {
				metrics.LoadingPixels++
			}
			switch info.Fallback {
			case StatusExact:
				metrics.ExactPixels++
			case StatusCoarse:
				metrics.CoarsePixels++
			case StatusRoot:
				metrics.RootPixels++
			}
		}
	}

	drawNodeBounds(im, loader, cam, vp)
	im.FrameRect(vp.X, vp.Y, vp.Size, vp.Size, colorInk)
	finalizeMetrics(&metrics)
	drawDashboard(im, metrics, cam)
	drawFooter(im)

	return RenderedFrame{Image: im, Metrics: metrics, Mesh: mesh}
}

func applyStatusTint(im *Image, x, y int, status LoadStatus) {
	switch status {
	case StatusExact:
		im.Blend(x, y, colorExact.withAlpha(38))
	case StatusLoading:
		im.Blend(x, y, colorLoading.withAlpha(62))
	case StatusCoarse:
		im.Blend(x, y, colorCoarse.withAlpha(62))
	case StatusRoot:
		im.Blend(x, y, colorRoot.withAlpha(70))
	}
}

func finalizeMetrics(m *FrameMetrics) {
	if m.MeshPixels > 0 {
		m.MeanError /= float64(m.MeshPixels)
		m.TargetCoverage = float64(m.ExactPixels) / float64(m.MeshPixels) * 100
	}
}

func screenXForU(u float64, cam Camera, vp Viewport) int {
	return int(math.Round(vp.CenterX() + (u-cam.Center.U)*cam.Scale))
}

func screenYForV(v float64, cam Camera, vp Viewport) int {
	return int(math.Round(vp.CenterY() + (v-cam.Center.V)*cam.Scale))
}

func drawNodeBounds(im *Image, loader *Loader, cam Camera, vp Viewport) {
	u0, v0, u1, v1 := visibleUVBounds(cam, vp)
	level := idealLOD(cam)
	for _, id := range nodesInBounds(level, u0, v0, u1, v1) {
		n := float64(levelCount(level))
		x0 := screenXForU(float64(id.X)/n, cam, vp)
		y0 := screenYForV(float64(id.Y)/n, cam, vp)
		x1 := screenXForU(float64(id.X+1)/n, cam, vp)
		y1 := screenYForV(float64(id.Y+1)/n, cam, vp)
		status := loader.Inspect(nodeCenter(id), cam).Status()
		c := statusColor(status)
		drawClippedRect(im, x0, y0, x1-x0, y1-y0, vp, c)
	}
}

func statusColor(status LoadStatus) Color {
	switch status {
	case StatusExact:
		return colorExact
	case StatusLoading:
		return colorLoading
	case StatusCoarse:
		return colorCoarse
	default:
		return colorRoot
	}
}

func drawClippedRect(im *Image, x, y, w, h int, vp Viewport, c Color) {
	x = maxInt(x, vp.X)
	y = maxInt(y, vp.Y)
	x2 := minInt(x+w, vp.X+vp.Size-1)
	y2 := minInt(y+h, vp.Y+vp.Size-1)
	if x2 <= x || y2 <= y {
		return
	}
	for xx := x; xx <= x2; xx++ {
		im.Set(xx, y, c)
		im.Set(xx, y2, c)
	}
	for yy := y; yy <= y2; yy++ {
		im.Set(x, yy, c)
		im.Set(x2, yy, c)
	}
}

func drawChrome(im *Image, strategy Strategy, detail string, cam Camera) {
	title := strategyTitle(strategy)
	drawText(im, 28, 22, 3, title, colorInk)
	drawText(im, 30, 54, 2, detail+" DETAIL", colorAccentForDetail(detail))
	im.FillRect(410, 76, 202, 360, colorPanel)
	im.FrameRect(410, 76, 202, 360, rgb(71, 85, 105))
	drawText(im, 426, 94, 2, "LOAD STATUS", colorInk)
}

func strategyTitle(s Strategy) string {
	switch s {
	case StrategyFull:
		return "FULL PAYLOAD"
	case StrategyChunk:
		return "FIXED CHUNKS"
	default:
		return "SCREEN ERROR"
	}
}

func colorAccentForDetail(detail string) Color {
	switch detail {
	case "LOW":
		return colorRoot
	case "MEDIUM":
		return colorCoarse
	default:
		return colorExact
	}
}

func drawDashboard(im *Image, m FrameMetrics, cam Camera) {
	x := 426
	y := 126
	lines := []string{
		fmt.Sprintf("IDEAL LOD %d", m.IdealLOD),
		fmt.Sprintf("BYTES %d KB", m.BytesLoaded/1024),
		fmt.Sprintf("READY %d%%", int(m.TargetCoverage+0.5)),
		fmt.Sprintf("SSE %.2f PX", m.MeanError),
		fmt.Sprintf("ZOOM %.0f", cam.Scale),
	}
	for _, line := range lines {
		drawText(im, x, y, 2, line, colorMuted)
		y += 24
	}

	legend := []struct {
		text   string
		status LoadStatus
	}{
		{"EXACT TILE", StatusExact},
		{"LOADING", StatusLoading},
		{"COARSE PARENT", StatusCoarse},
		{"ROOT PARENT", StatusRoot},
	}
	y = 258
	for _, item := range legend {
		c := statusColor(item.status)
		im.FillRect(x, y-2, 16, 14, c)
		im.FrameRect(x, y-2, 16, 14, colorInk)
		drawText(im, x+24, y, 2, item.text, colorInk)
		y += 30
	}

	drawText(im, x, 392, 1, "TINT: RESIDENCY STATE", colorMuted)
	drawText(im, x, 404, 1, "SAME UV MESH EVERY FRAME", colorMuted)
}

func drawFooter(im *Image) {
	scale := 2
	lines := []string{
		"PARENT FALLBACK KEEPS UV AND COLOR STABLE",
		"NO BLANK / NO BLACK-WHITE FLASH",
	}
	for i, msg := range lines {
		w := textWidth(msg, scale)
		x := (frameW - w) / 2
		drawText(im, x, 444+i*18, scale, msg, colorInk)
	}
}
