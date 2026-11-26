package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

const (
	overlayW      = 960
	overlayH      = 764
	overlayHeader = 104
	overlaySceneH = 600
)

func offsetQuad(quad []Vec2, dx, dy float64) []Vec2 {
	out := make([]Vec2, len(quad))
	for i, p := range quad {
		out[i] = Vec2{X: p.X + dx, Y: p.Y + dy}
	}
	return out
}

func drawLegendItem(dst *image.RGBA, x, y int, c color.RGBA, label string) {
	fillRect(dst, x, y, 13, 13, c)
	drawText(dst, x+21, y+3, label, 2, colWhite)
}

// BuildVersionOverlay composites the correct frame with both source quads and
// their version provenance. Cyan is the live camera, amber is the pinned
// historical camera, and magenta links corresponding corners to show the
// projection delta while the texture remains identical.
func BuildVersionOverlay(normal, stale RenderBinding, normalFrame *image.RGBA) (*image.RGBA, error) {
	dst := image.NewRGBA(image.Rect(0, 0, overlayW, overlayH))
	fillRect(dst, 0, 0, overlayW, overlayH, colInk)

	scene := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	draw.Draw(scene, scene.Bounds(), normalFrame, image.Point{}, draw.Src)
	dimImage(scene, 0.55)
	draw.Draw(dst, image.Rect(0, overlayHeader, canvasW, overlayHeader+canvasH), scene, image.Point{}, draw.Src)

	currentScreen, ok := projectQuad(normal.Camera.Matrix, materialWorldQuad)
	if !ok {
		return nil, fmt.Errorf("overlay: current projection failed")
	}
	pinnedScreen, ok := projectQuad(stale.Camera.Matrix, materialWorldQuad)
	if !ok {
		return nil, fmt.Errorf("overlay: stale projection failed")
	}
	current := offsetQuad(currentScreen, 0, overlayHeader)
	pinned := offsetQuad(pinnedScreen, 0, overlayHeader)

	fillConvexPolygon(dst, pinned, color.RGBA{R: 255, G: 178, B: 58, A: 46})
	fillConvexPolygon(dst, current, color.RGBA{R: 45, G: 212, B: 232, A: 34})
	drawQuadOutline(dst, current, 3, colCyan)
	drawDashedQuad(dst, pinned, 3, colAmber)

	for i := range current {
		drawDashedLine(dst, pinned[i], current[i], 2, 8, 6, colMagenta)
	}

	pinnedCenter := quadCenter(pinned)
	currentCenter := quadCenter(current)
	fillCircle(dst, int(pinnedCenter.X), int(pinnedCenter.Y), 6, colAmber)
	fillCircle(dst, int(currentCenter.X), int(currentCenter.Y), 6, colCyan)
	drawCircle(dst, int(pinnedCenter.X), int(pinnedCenter.Y), 15, 2, colAmber)
	drawCircle(dst, int(currentCenter.X), int(currentCenter.Y), 15, 2, colCyan)
	drawArrow(dst, pinnedCenter, currentCenter, 3, colMagenta)

	labelY := int(pinnedCenter.Y) - 66
	fillRect(dst, int(pinnedCenter.X)-128, labelY, 256, 28, color.RGBA{R: 48, G: 35, B: 12, A: 232})
	fillRect(dst, int(pinnedCenter.X)-128, labelY, 5, 28, colAmber)
	drawTextCentered(dst, int(pinnedCenter.X), labelY+7, fmt.Sprintf("PINNED CAM V%d", stale.Snapshot.CameraVersion), 2, colAmber)

	labelY = int(currentCenter.Y) + 26
	fillRect(dst, int(currentCenter.X)-124, labelY, 248, 28, color.RGBA{R: 9, G: 48, B: 54, A: 232})
	fillRect(dst, int(currentCenter.X)-124, labelY, 5, 28, colCyan)
	drawTextCentered(dst, int(currentCenter.X), labelY+7, fmt.Sprintf("CURRENT CAM V%d", normal.Snapshot.CameraVersion), 2, colCyan)

	mid := pinnedCenter.midpoint(currentCenter)
	drawTextCentered(dst, int(mid.X), int(mid.Y)-38, "PROJECTION DELTA", 2, colMagenta)

	fillRect(dst, 0, 0, overlayW, overlayHeader, color.RGBA{R: 10, G: 20, B: 34, A: 255})
	fillRect(dst, 0, overlayHeader-3, overlayW, 3, colPanelEdge)
	drawText(dst, 24, 18, "DESCRIPTOR VERSION SOURCE OVERLAY", 3, colWhite)
	drawText(dst, 24, 53, fmt.Sprintf("CURRENT  CAM V%d  TEX V%d  DESC V%d  FRESH", normal.Snapshot.CameraVersion, normal.Snapshot.TextureVersion, normal.Snapshot.DescriptorVersion), 2, colCyan)
	drawText(dst, 24, 76, fmt.Sprintf("PINNED   CAM V%d  TEX V%d  DESC V%d  STALE", stale.Snapshot.CameraVersion, stale.Snapshot.TextureVersion, stale.Snapshot.DescriptorVersion), 2, colAmber)
	drawText(dst, 690, 22, "SAME TEXTURE / WRONG MATRIX", 1, colMuted)
	drawText(dst, 690, 43, fmt.Sprintf("SNAPSHOT RETAINS CAM V%d", stale.Snapshot.CameraVersion), 1, colMuted)

	footerY := overlayHeader + overlaySceneH
	fillRect(dst, 0, footerY, overlayW, overlayH-footerY, color.RGBA{R: 10, G: 20, B: 34, A: 255})
	fillRect(dst, 0, footerY, overlayW, 2, colPanelEdge)
	drawLegendItem(dst, 24, footerY+13, colCyan, "CURRENT CAMERA")
	drawLegendItem(dst, 260, footerY+13, colAmber, "PINNED SNAPSHOT")
	drawLegendItem(dst, 500, footerY+13, colMagenta, "PROJECTION DELTA")
	drawLegendItem(dst, 748, footerY+13, colWhite, "TEX V1 SAME")
	drawText(dst, 24, footerY+37, "THE STALE FRAME USES THE CORRECT TEXTURE WITH ITS ORIGINAL CAMERA VERSION.", 1, colMuted)
	return dst, nil
}
