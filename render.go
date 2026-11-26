package main

import "fmt"

const (
	FrameW = 480
	FrameH = 360
)

type RenderOptions struct {
	HighlightPath string
}

type Renderer struct{}

func NewRenderer() *Renderer { return &Renderer{} }

// Render 每次分配新画布，并在绘制前验证所有绑定。
// 因此 name-only 的伪切换不可能悄悄产出一帧。
func (r *Renderer) Render(stage *Stage, opts RenderOptions) (*Image, error) {
	if err := stage.ValidateBindings(); err != nil {
		return nil, fmt.Errorf("invalid variant bindings: %w", err)
	}
	if opts.HighlightPath != "" {
		if _, err := stage.Resolve(opts.HighlightPath); err != nil {
			return nil, err
		}
	}

	im := NewImage(FrameW, FrameH)
	for _, node := range stage.orderedNodes {
		if len(node.Slots) == 0 {
			continue
		}
		binding := node.Bindings["surface"]
		if !binding.Resolved() || binding.Material == nil {
			return nil, fmt.Errorf("%s has unresolved surface binding", node.Path)
		}
		material := *binding.Material
		im.FillShape(node.Shape, material.Fill)
		drawMaterialPattern(im, node, material)
	}

	if opts.HighlightPath != "" {
		resolution, _ := stage.Resolve(opts.HighlightPath)
		drawSelectionHighlight(im, stage, resolution)
	}
	return im, nil
}

func drawSelectionHighlight(im *Image, stage *Stage, resolution Resolution) {
	amber := hexRGB("#FFB000")
	cyan := hexRGB("#35E0FF")
	white := hexRGB("#F8FAFC")

	for _, step := range resolution.Steps {
		node, err := stage.Node(step.Path)
		if err != nil || node.Shape.Rect.W == 0 || node.Shape.Rect.H == 0 {
			continue
		}
		switch step.Path {
		case resolution.TargetPath:
			im.DrawCornerBrackets(node.Shape.Rect, 5, 3, cyan)
		case resolution.SourcePath:
			im.FrameRect(node.Shape.Rect, 2, amber)
		default:
			im.FrameRect(node.Shape.Rect, 1, white)
		}
	}

	if source, err := stage.Node(resolution.SourcePath); err == nil && source.Shape.Rect.W > 0 {
		label := "AUTH OVR: " + string(resolution.Variant)
		drawLabelPill(im, source.Shape.Rect.X, maxInt(8, source.Shape.Rect.Y-18), 1, label, amber, RGB{R: 20, G: 16, B: 8})
	}
}

func (im *Image) DrawCornerBrackets(r Rect, arm, thickness int, c RGB) {
	im.FillRect(Rect{X: r.X, Y: r.Y, W: arm, H: thickness}, c)
	im.FillRect(Rect{X: r.X, Y: r.Y, W: thickness, H: arm}, c)

	im.FillRect(Rect{X: r.X + r.W - arm, Y: r.Y, W: arm, H: thickness}, c)
	im.FillRect(Rect{X: r.X + r.W - thickness, Y: r.Y, W: thickness, H: arm}, c)

	im.FillRect(Rect{X: r.X, Y: r.Y + r.H - thickness, W: arm, H: thickness}, c)
	im.FillRect(Rect{X: r.X, Y: r.Y + r.H - arm, W: thickness, H: arm}, c)

	im.FillRect(Rect{X: r.X + r.W - arm, Y: r.Y + r.H - thickness, W: arm, H: thickness}, c)
	im.FillRect(Rect{X: r.X + r.W - thickness, Y: r.Y + r.H - arm, W: thickness, H: arm}, c)
}

func drawLabelPill(im *Image, x, y, scale int, text string, fg, bg RGB) {
	width := TextWidth(text, scale) + 10*scale
	height := 9*scale + 6
	r := Rect{X: x, Y: y, W: width, H: height}
	im.BlendRect(r, bg, 220)
	im.FrameRect(r, 1, fg)
	im.DrawText(x+5*scale, y+3, scale, text, fg)
}

func deterministicHash(x, y int) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}
