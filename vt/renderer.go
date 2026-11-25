package vt

import (
	"image"
	"image/color"
)

type SampleStatus uint8

const (
	StatusExact SampleStatus = iota
	StatusAncestor
	StatusMissing
)

// PixelInfo retains the routing decision for verification and mask output.
type PixelInfo struct {
	U             float64
	V             float64
	Requested     PageID
	Resolved      PageID
	Slot          int
	FallbackDepth int
	Status        SampleStatus
}

type FrameMetrics struct {
	Width          int
	Height         int
	ExactPixels    int
	FallbackPixels int
	MissingPixels  int
	ExactHitRate   float64
	MissRate       float64
}

// Frame contains the camera image, masks and per-pixel routing diagnostics.
type Frame struct {
	Image      *image.RGBA
	MissMask   *image.Gray
	StatusMask *image.RGBA
	Pixels     []PixelInfo
	Metrics    FrameMetrics
}

// Render software-rasterizes the camera view using one table snapshot.
func Render(cam Camera, info Info, table *Table, width, height int) *Frame {
	if width < 2 {
		width = 2
	}
	if height < 2 {
		height = 2
	}
	snapshot := table.Snapshot()
	aspect := float64(width) / float64(height)
	u0, v0, u1, v1 := cam.Bounds(aspect)
	duDX := (u1 - u0) / float64(width)
	dvDY := (v1 - v0) / float64(height)
	mip := Mip(info, duDX, dvDY)

	frame := &Frame{
		Image:      image.NewRGBA(image.Rect(0, 0, width, height)),
		MissMask:   image.NewGray(image.Rect(0, 0, width, height)),
		StatusMask: image.NewRGBA(image.Rect(0, 0, width, height)),
		Pixels:     make([]PixelInfo, width*height),
	}
	used := make(map[PageID]struct{})

	for y := 0; y < height; y++ {
		v := v0 + (float64(y)+0.5)/float64(height)*(v1-v0)
		for x := 0; x < width; x++ {
			u := u0 + (float64(x)+0.5)/float64(width)*(u1-u0)
			uu, vv := ClampUV(u, v)
			requested := info.PageAtUV(uu, vv, mip)
			resolved, entry, depth, ok := Resolve(snapshot, info, requested)
			idx := y*width + x
			infoPixel := PixelInfo{
				U: uu, V: vv, Requested: requested, Resolved: resolved,
				FallbackDepth: depth,
			}

			var raster color.RGBA
			missGray := uint8(255)
			statusColor := color.RGBA{R: 220, G: 48, B: 48, A: 255}
			if !ok {
				infoPixel.Status = StatusMissing
				frame.Metrics.MissingPixels++
				raster = color.RGBA{R: 24, G: 20, B: 28, A: 255}
			} else {
				infoPixel.Slot = entry.Slot
				used[resolved] = struct{}{}
				dim := float64(info.BaseSize >> resolved.Mip)
				localX := uu*dim - 0.5 - float64(resolved.X*info.PageSize)
				localY := vv*dim - 0.5 - float64(resolved.Y*info.PageSize)
				raster = sampleBilinear(entry.Image, localX, localY)
				if depth == 0 {
					infoPixel.Status = StatusExact
					missGray = 0
					statusColor = color.RGBA{R: 38, G: 170, B: 92, A: 255}
					frame.Metrics.ExactPixels++
				} else {
					infoPixel.Status = StatusAncestor
					statusColor = color.RGBA{R: 245, G: 178, B: 46, A: 255}
					frame.Metrics.FallbackPixels++
				}
			}

			frame.Pixels[idx] = infoPixel
			frame.Image.SetRGBA(x, y, raster)
			frame.MissMask.SetGray(x, y, color.Gray{Y: missGray})
			frame.StatusMask.SetRGBA(x, y, statusColor)
		}
	}

	usedIDs := make([]PageID, 0, len(used))
	for id := range used {
		usedIDs = append(usedIDs, id)
	}
	table.Touch(usedIDs)

	total := float64(width * height)
	frame.Metrics.Width = width
	frame.Metrics.Height = height
	frame.Metrics.ExactHitRate = float64(frame.Metrics.ExactPixels) / total
	frame.Metrics.MissRate = float64(frame.Metrics.FallbackPixels+frame.Metrics.MissingPixels) / total
	return frame
}
