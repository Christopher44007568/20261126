// Package viz composes the before/after audit posters: a top-down scene map
// with link arcs and red annotations plus a side ledger quoting concrete old
// paths. Every pure-red rectangle is recorded as a Box so red_boxes.json can
// describe exactly what is marked.
package viz

import (
	"fmt"

	"nsmigrate/audit"
	"nsmigrate/raster"
	"nsmigrate/scene"
)

// Box is one red annotation rectangle in image pixel coordinates.
type Box struct {
	ID       string `json:"id"`
	Image    string `json:"image,omitempty"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	W        int    `json:"w"`
	H        int    `json:"h"`
	Category string `json:"category"` // broken_link | retained_default
	Part     string `json:"part,omitempty"`
	LinkID   string `json:"link_id,omitempty"`
	Target   string `json:"target,omitempty"`
	OldPath  string `json:"old_path,omitempty"`
	Label    string `json:"label,omitempty"`
}

// Options configures one poster.
type Options struct {
	ImageName string
	Mode      string // before | correct | string-only
	Title     string
	Subtitle  string
	Stage     *scene.Stage
	Report    *audit.Report
	Mapping   scene.Mapping
}

// Poster is the rendered canvas plus its red boxes.
type Poster struct {
	Canvas *raster.Canvas
	Boxes  []Box
	Width  int
	Height int
}

const (
	winW       = 1760
	margin     = 28
	headerH    = 118
	contentY   = 132
	sceneW     = 1040
	panelGap   = 20
	sideX      = margin + sceneW + panelGap // 1088
	sideW      = winW - sideX - margin      // 644
	cardW      = sideW - 32                 // 612
	cardX      = sideX + 16
	cardInnerX = cardX + 16
	cardInnerW = cardW - 32 // 580
)

var palette = struct {
	page, panel, border, floor, grid, ink, muted, faint raster.Color
	ref, mat, cam, ok, okBg, bad, badBg, warnBg         raster.Color
}{
	page:   raster.Hex(0xDDE4EE),
	panel:  raster.Hex(0xFFFFFF),
	border: raster.Hex(0xD3DCE8),
	floor:  raster.Hex(0xFAFBFE),
	grid:   raster.Hex(0xE7EDF5),
	ink:    raster.Hex(0x1F2A3D),
	muted:  raster.Hex(0x5C6B82),
	faint:  raster.Hex(0x8A97AB),
	ref:    raster.Hex(0x2563EB),
	mat:    raster.Hex(0xD98306),
	cam:    raster.Hex(0x0E93B5),
	ok:     raster.Hex(0x15803D),
	okBg:   raster.Hex(0xE4F5EB),
	bad:    raster.Hex(0xB42318),
	badBg:  raster.Hex(0xFDECEC),
	warnBg: raster.Hex(0xFEF4D6),
}

func cvt(c scene.RGB) raster.Color { return raster.Color{R: c.R, G: c.G, B: c.B} }

// redBox draws a pure-red frame (with a white halo for contrast) and records
// it. The recorded rectangle is the red frame outer bounds.
func (p *Poster) redBox(x, y, w, h int, b Box, halo bool) {
	if halo {
		p.Canvas.FrameRect(x-4, y-4, w+8, h+8, 7, raster.Color{R: 255, G: 255, B: 255})
	}
	p.Canvas.FrameRect(x, y, w, h, 4, raster.Red)
	b.X, b.Y, b.W, b.H = x, y, w, h
	if b.ID == "" {
		b.ID = fmt.Sprintf("RBOX-%02d", len(p.Boxes)+1)
	}
	p.Boxes = append(p.Boxes, b)
}
