package hotreload

// InitialRGBPayload builds the cool blue diagonal baseline texture.
func InitialRGBPayload() []byte {
	pix := make([]byte, RGBBytes)
	for y := 0; y < RGBHeight; y++ {
		for x := 0; x < RGBWidth; x++ {
			i := (y*RGBWidth + x) * 4
			stripe := (x + y) % 11
			v := uint8(24 + stripe*17)
			pix[i] = uint8(8 + stripe*3)
			pix[i+1] = uint8(70 + int(v)/3)
			pix[i+2] = uint8(110 + int(v))
			pix[i+3] = 255
			if (x-RGBWidth/2)*(x-RGBWidth/2)+(y-RGBHeight/2)*(y-RGBHeight/2) < 14*14 {
				pix[i] = 24
				pix[i+1] = 220
				pix[i+2] = 200
			}
		}
	}
	return pix
}

// ReloadedRGBPayload builds the warm magenta/amber texture used after a
// successful RGB hot reload.
func ReloadedRGBPayload() []byte {
	pix := make([]byte, RGBBytes)
	for y := 0; y < RGBHeight; y++ {
		for x := 0; x < RGBWidth; x++ {
			i := (y*RGBWidth + x) * 4
			wave := (x*3 + y*5) % 17
			pix[i] = uint8(210 + wave%2*30)
			pix[i+1] = uint8(70 + wave*9)
			pix[i+2] = uint8(120 + wave*6)
			pix[i+3] = 255
			if (x-RGBWidth/2)*(x-RGBWidth/2)+(y-RGBHeight/2)*(y-RGBHeight/2) < 16*16 {
				pix[i] = 90
				pix[i+1] = 220
				pix[i+2] = 190
			}
		}
	}
	return pix
}

// BrokenRGBPayload is intentionally truncated to exercise error surfacing.
func BrokenRGBPayload() []byte {
	buf := make([]byte, 17)
	for i := range buf {
		buf[i] = byte(i)
	}
	return buf
}

// InitialGeometry is the rectangle accepted by the stripe material.
func InitialGeometry() *GeometrySpec {
	return &GeometrySpec{
		AssetID:           "asset.rect-01",
		Revision:          1,
		Shape:             ShapeRectangle,
		Bounds:            RightPaint,
		AcceptedMaterials: []string{string(MatStripes)},
	}
}

// ReplacementGeometry swaps in a triangle. The old stripe material is not
// accepted; check-amber is the compatible replacement.
func ReplacementGeometry() *GeometrySpec {
	return &GeometrySpec{
		AssetID:           "asset.tri-02",
		Revision:          2,
		Shape:             ShapeTriangle,
		Bounds:            RightPaint,
		AcceptedMaterials: []string{string(MatCheckAmber)},
	}
}

func renderBackground() *Frame {
	f := NewFrame(CanvasW, CanvasH)
	f.Fill(ColorBg)
	// Slightly darker header band.
	f.FillRect(0, 0, CanvasW, 34, RGB{14, 22, 40})
	// Sparse dotted grid.
	for y := 46; y < CanvasH; y += 24 {
		for x := 18; x < CanvasW; x += 32 {
			f.Set(x, y, ColorGrid)
		}
	}
	drawPanelChrome(f, LeftPanel)
	drawPanelChrome(f, RightPanel)
	return f
}

func drawPanelChrome(f *Frame, r Rect) {
	f.FillRect(r.X0, r.Y0, r.X1, r.Y1, ColorPanel)
	f.Rect(r.X0, r.Y0, r.X1, r.Y1, 2, ColorPanelLine)
}

func renderRGBLayer(payload *RGBPayload) (*Frame, *NodeError) {
	f := NewFrame(CanvasW, CanvasH)
	if err := payload.Validate(); err != nil {
		return f, &NodeError{
			NodeID:  "rgb.layer",
			Code:    "invalid-rgb-buffer",
			Message: err.Error(),
		}
	}
	// Nearest-neighbor upscale of the 64x64 RGBA payload into the left viewport.
	target := LeftPaint
	for y := target.Y0; y < target.Y1; y++ {
		sy := (y - target.Y0) * RGBHeight / target.Height()
		for x := target.X0; x < target.X1; x++ {
			sx := (x - target.X0) * RGBWidth / target.Width()
			src := (sy*RGBWidth + sx) * 4
			f.Set(x, y, RGB{payload.Pix[src], payload.Pix[src+1], payload.Pix[src+2]})
		}
	}
	return f, nil
}

func renderRGBGrade(src *Frame, nodeErr *NodeError) *Frame {
	f := NewFrame(CanvasW, CanvasH)
	if nodeErr != nil {
		// An upstream error becomes an explicit error card; stale pixels are
		// never reused as a successful frame.
		drawErrorCard(f, LeftPanel, "RGB BUFFER INVALID")
		return f
	}
	for y := LeftPaint.Y0; y < LeftPaint.Y1; y++ {
		for x := LeftPaint.X0; x < LeftPaint.X1; x++ {
			c := src.Get(x, y)
			// Horizontal scanline grade keeps the procedural texture deterministic.
			if (y-LeftPaint.Y0)%3 == 0 {
				c = RGB{c.R / 2, c.G / 2, c.B / 2}
			}
			f.Set(x, y, c)
		}
	}
	f.Rect(LeftPaint.X0, LeftPaint.Y0, LeftPaint.X1, LeftPaint.Y1, 1, ColorCyanSoft)
	return f
}

func drawErrorCard(f *Frame, r Rect, _ string) {
	f.FillRect(r.X0, r.Y0, r.X1, r.Y1, ColorErrorBg)
	f.DashedRect(r.X0+2, r.Y0+2, r.X1-2, r.Y1-2, 2, 6, 5, ColorErrorBorder)
	// Two diagonal bars form a failure cross without needing font rendering.
	f.Line(r.X0+12, r.Y0+12, r.X1-12, r.Y1-12, ColorErrorLine)
	f.Line(r.X1-12, r.Y0+12, r.X0+12, r.Y1-12, ColorErrorLine)
}

func renderGeoLayer(geo *GeometrySpec) *Frame {
	f := NewFrame(CanvasW, CanvasH)
	r := geo.Bounds
	switch geo.Shape {
	case ShapeRectangle:
		f.Rect(r.X0, r.Y0, r.X1, r.Y1, 2, ColorCyan)
		for _, p := range []Pt{{r.X0, r.Y0}, {r.X1 - 1, r.Y0}, {r.X0, r.Y1 - 1}, {r.X1 - 1, r.Y1 - 1}} {
			f.FillRect(p.X-3, p.Y-3, p.X+4, p.Y+4, ColorWhite)
		}
	case ShapeTriangle:
		a := Pt{X: (r.X0 + r.X1) / 2, Y: r.Y0}
		b := Pt{X: r.X0, Y: r.Y1 - 1}
		c := Pt{X: r.X1 - 1, Y: r.Y1 - 1}
		f.StrokeTriangle(a, b, c, ColorCyan)
		for _, p := range []Pt{a, b, c} {
			f.FillRect(p.X-3, p.Y-3, p.X+4, p.Y+4, ColorWhite)
		}
	}
	return f
}
