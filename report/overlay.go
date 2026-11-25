package report

import (
	"fmt"
	"image"
	"image/color"

	"autofocus/autofocus"
)

// FocusOverlay renders a locked frame with the ROI, a near/far focus scale,
// and a compact AF telemetry panel.
func FocusOverlay(frame Frame, cfg autofocus.Config, snap autofocus.Snapshot) *image.RGBA {
	const (
		canvasW = 1000
		canvasH = 640
		imgX    = 24
		imgY    = 64
		imgW    = 720
		imgH    = 540
	)
	img := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	fill(img, img.Bounds(), colorBackground)
	drawText(img, imgX, 20, "FOCUS POSITION OVERLAY", 2, colorText)
	drawText(img, 824, 23, "PURE GO / NO CAMERA SDK", 1, colorMuted)

	view := resize(frame.Image, imgW, imgH)
	copyAt(img, view, imgX, imgY)

	if !frame.ROI.Empty() {
		roi := scaleRect(frame.ROI, frame.Image.Bounds(), imgW, imgH).Add(image.Pt(imgX, imgY))
		stroke(img, roi, colorGreen, 3)
		drawText(img, roi.Min.X+5, roi.Min.Y+5, "AF ROI", 1, colorGreen)
	}

	hud := image.Rect(imgX+18, imgY+imgH-90, imgX+imgW-18, imgY+imgH-12)
	blendFill(img, hud, color.RGBA{R: 5, G: 9, B: 14, A: 255}, 206)
	drawText(img, hud.Min.X+12, hud.Min.Y+10, statusText(snap.State), 2, statusColor(snap.State))
	drawText(img, hud.Min.X+140, hud.Min.Y+10,
		fmt.Sprintf("P%02d  S%.3f", snap.Position, snap.LockScore), 2, colorText)
	bar := image.Rect(hud.Min.X+12, hud.Min.Y+47, hud.Max.X-12, hud.Min.Y+52)
	drawLine(img, bar.Min.X, bar.Min.Y, bar.Max.X, bar.Min.Y, colorGrid)
	drawText(img, bar.Min.X, bar.Min.Y+9, "NEAR", 1, colorMuted)
	drawText(img, bar.Max.X-textWidth("FAR", 1), bar.Min.Y+9, "FAR", 1, colorMuted)
	if cfg.MaxPosition > cfg.MinPosition {
		x := bar.Min.X + (snap.Position-cfg.MinPosition)*(bar.Dx()-1)/(cfg.MaxPosition-cfg.MinPosition)
		drawLine(img, x, bar.Min.Y-9, x, bar.Min.Y+9, colorCyan)
		fill(img, image.Rect(x-3, bar.Min.Y-13, x+4, bar.Min.Y-8), colorCyan)
	}

	panel := image.Rect(768, 64, 976, 604)
	fill(img, panel, colorPanel)
	stroke(img, panel, colorGrid, 1)
	drawText(img, panel.Min.X+16, panel.Min.Y+16, "AF TELEMETRY", 1, colorMuted)
	drawText(img, panel.Min.X+16, panel.Min.Y+38, "STATUS", 1, colorMuted)
	drawText(img, panel.Min.X+16, panel.Min.Y+54, statusText(snap.State), 2, statusColor(snap.State))
	drawText(img, panel.Min.X+16, panel.Min.Y+90, "MOTOR", 1, colorMuted)
	drawText(img, panel.Min.X+16, panel.Min.Y+106,
		fmt.Sprintf("%02d / %02d", snap.Position, cfg.MaxPosition), 2, colorText)
	drawText(img, panel.Min.X+16, panel.Min.Y+142, "BEST", 1, colorMuted)
	drawText(img, panel.Min.X+16, panel.Min.Y+158, fmt.Sprintf("%02d", snap.BestPosition), 2, colorText)
	drawText(img, panel.Min.X+16, panel.Min.Y+194, "SCORE", 1, colorMuted)
	drawText(img, panel.Min.X+16, panel.Min.Y+210, fmt.Sprintf("%.3f", snap.LockScore), 2, colorText)
	drawText(img, panel.Min.X+16, panel.Min.Y+246, "HYSTERESIS", 1, colorMuted)
	drawText(img, panel.Min.X+16, panel.Min.Y+262, fmt.Sprintf("%.0f PCT", cfg.LockHysteresis*100), 2, colorGreen)

	drawText(img, panel.Min.X+16, panel.Min.Y+312, "MOVES", 1, colorMuted)
	drawText(img, panel.Max.X-16-textWidth(fmt.Sprint(snap.Moves), 2), panel.Min.Y+312,
		fmt.Sprint(snap.Moves), 2, colorText)
	drawText(img, panel.Min.X+16, panel.Min.Y+348, "REVERSALS", 1, colorMuted)
	drawText(img, panel.Max.X-16-textWidth(fmt.Sprint(snap.Reversals), 2), panel.Min.Y+348,
		fmt.Sprint(snap.Reversals), 2, colorText)
	drawText(img, panel.Min.X+16, panel.Min.Y+384, "AFTER LOCK", 1, colorMuted)
	drawText(img, panel.Max.X-16-textWidth(fmt.Sprint(snap.MovesAfterLock), 2), panel.Min.Y+384,
		fmt.Sprint(snap.MovesAfterLock), 2, colorGreen)

	drawText(img, panel.Min.X+16, panel.Min.Y+434, "DETECTOR", 1, colorMuted)
	drawText(img, panel.Min.X+16, panel.Min.Y+450, "TENENGRAD", 2, colorText)
	drawText(img, panel.Min.X+16, panel.Min.Y+472, "SEARCH", 1, colorMuted)
	drawText(img, panel.Min.X+16, panel.Min.Y+488, "COARSE + FINE", 2, colorText)
	drawText(img, panel.Min.X+16, panel.Min.Y+510, "LOCK IS STICKY", 1, colorGreen)
	return img
}

func statusText(state autofocus.State) string {
	switch state {
	case autofocus.StateLocked:
		return "LOCKED"
	case autofocus.StateCoarse:
		return "COARSE"
	case autofocus.StateFine:
		return "FINE"
	case autofocus.StateLocking:
		return "CONFIRM"
	case autofocus.StateWaiting:
		return "WAITING"
	case autofocus.StateReacquire:
		return "REACQUIRE"
	default:
		return "AF"
	}
}

func statusColor(state autofocus.State) color.RGBA {
	if state == autofocus.StateLocked {
		return colorGreen
	}
	return colorOrange
}
