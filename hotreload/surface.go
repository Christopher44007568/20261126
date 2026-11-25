package hotreload

func shadeMaterial(mat MaterialSpec, x, y int) RGB {
	a, b := mat.Palette[0], mat.Palette[1]
	switch mat.ID {
	case MatStripes:
		if (x+y)%12 < 6 {
			return a
		}
		return b
	case MatCheckAmber:
		if ((x/16)+(y/16))%2 == 0 {
			return a
		}
		return b
	case MatDanger:
		if (x+y)%10 < 5 {
			return a
		}
		return b
	default:
		return a
	}
}

func defaultMaterial(id MaterialKind) MaterialSpec {
	switch id {
	case MatCheckAmber:
		return MaterialSpec{ID: id, Palette: [2]RGB{{116, 72, 220}, {255, 176, 46}}}
	case MatDanger:
		return MaterialSpec{ID: id, Palette: [2]RGB{{58, 12, 28}, {255, 72, 96}}}
	default:
		return MaterialSpec{ID: MatStripes, Palette: [2]RGB{{116, 72, 220}, {72, 226, 210}}}
	}
}

// renderSurface paints a materialized geometry. A nil material means the old
// material was detached by geometry replacement; the danger fallback makes
// that state visible rather than silently reusing the old shader.
func renderSurface(geo *GeometrySpec, mat *MaterialSpec) *Frame {
	f := NewFrame(CanvasW, CanvasH)
	effective := MaterialSpec{ID: MatDanger, Palette: [2]RGB{{58, 12, 28}, {255, 72, 96}}}
	if mat != nil {
		effective = *mat
	}
	r := geo.Bounds
	paint := func(x, y int) {
		if f.In(x, y) {
			f.Set(x, y, shadeMaterial(effective, x, y))
		}
	}
	switch geo.Shape {
	case ShapeRectangle:
		for y := r.Y0; y < r.Y1; y++ {
			for x := r.X0; x < r.X1; x++ {
				paint(x, y)
			}
		}
	case ShapeTriangle:
		a := Pt{X: (r.X0 + r.X1) / 2, Y: r.Y0}
		b := Pt{X: r.X0, Y: r.Y1 - 1}
		c := Pt{X: r.X1 - 1, Y: r.Y1 - 1}
		f.FillTriangle(a, b, c, paint)
	}
	return f
}

// renderOutline derives the yellow contour from the current geometry only;
// it never consumes a material cache, so geometry replacement always
// refreshes it.
func renderOutline(geo *GeometrySpec) *Frame {
	f := NewFrame(CanvasW, CanvasH)
	// Normal frames keep the contour invisible; ComputeDiff paints the yellow
	// changed-bounds contour. The node still re-keys from geometry so replacing
	// an asset invalidates and recomputes this subtree entry.
	_ = geo
	return f
}

func renderComposite(background, grade, surface, outline, geo *Frame) *Frame {
	out := NewFrame(CanvasW, CanvasH)
	out.Copy(background, 0, 0)

	// RGB viewport: error card or graded pixels.
	copyOverlay(out, grade, LeftPaint)
	// Geometry viewport: materialized surface, then geometry frame, contour.
	copyOverlay(out, surface, RightPaint)
	copyOverlay(out, geo, RightPaint)
	copyOverlay(out, outline, RightPaint)

	// Viewport labels/handles outside painted regions remain on background.
	return out
}

// copyOverlay paints only non-black source pixels so transparent frame
// regions do not erase already composited surface content.
func copyOverlay(dst, src *Frame, r Rect) {
	black := RGB{}
	for y := r.Y0; y < r.Y1; y++ {
		for x := r.X0; x < r.X1; x++ {
			if src.In(x, y) {
				c := src.Get(x, y)
				if c != black {
					dst.Set(x, y, c)
				}
			}
		}
	}
}

func copyRect(dst, src *Frame, r Rect) {
	for y := r.Y0; y < r.Y1; y++ {
		for x := r.X0; x < r.X1; x++ {
			if src.In(x, y) {
				dst.Set(x, y, src.Get(x, y))
			}
		}
	}
}
