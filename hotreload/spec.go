package hotreload

import "fmt"

// RGBWidth/RGBHeight define the hot-reloadable RGB payload resolution.
const (
	RGBWidth  = 64
	RGBHeight = 64
	RGBPixels = RGBWidth * RGBHeight
	RGBBytes  = RGBPixels * 4
)

// RGBPayload is a raw RGBA buffer carried by rgb.payload.
type RGBPayload struct {
	Revision int
	Pix      []byte
}

// Validate enforces the exact RGBA byte length so a truncated hot reload
// cannot be hidden behind a stale frame cache.
func (p RGBPayload) Validate() error {
	if len(p.Pix) != RGBBytes {
		return fmt.Errorf("RGBA payload length %d, want %d", len(p.Pix), RGBBytes)
	}
	return nil
}

// Shape identifies the geometry topology produced by geo.asset.
type Shape string

const (
	ShapeRectangle Shape = "rectangle"
	ShapeTriangle  Shape = "triangle"
)

// GeometrySpec is an immutable asset version held by a geometry layer.
type GeometrySpec struct {
	AssetID  string
	Revision int
	Shape    Shape
	Bounds   Rect
	// AcceptedMaterials is the material compatibility set of this asset.
	AcceptedMaterials []string
}

type Rect struct {
	X0, Y0, X1, Y1 int
}

func (r Rect) Width() int  { return r.X1 - r.X0 }
func (r Rect) Height() int { return r.Y1 - r.Y0 }

// MaterialKind describes a procedural shader bound to a surface node.
type MaterialKind string

const (
	MatStripes    MaterialKind = "mat.stripes"
	MatCheckAmber MaterialKind = "mat.check-amber"
	MatDanger     MaterialKind = "mat.danger"
)

// MaterialSpec is the texture/shader attached to a surface geometry.
type MaterialSpec struct {
	ID      MaterialKind
	Palette [2]RGB
}

// Compatible reports whether the material may bind the geometry. Geometry
// replacement drops incompatible old materials instead of reusing them.
func (m MaterialSpec) Compatible(g GeometrySpec) bool {
	for _, allowed := range g.AcceptedMaterials {
		if allowed == string(m.ID) {
			return true
		}
	}
	return false
}

// NodeError records a render failure without masking it with cache data.
type NodeError struct {
	NodeID  string `json:"node_id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e NodeError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

var (
	ColorBg          = RGB{18, 27, 48}
	ColorPanel       = RGB{13, 18, 32}
	ColorPanelLine   = RGB{31, 41, 66}
	ColorGrid        = RGB{20, 28, 49}
	ColorCyan        = RGB{126, 244, 255}
	ColorCyanSoft    = RGB{190, 246, 255}
	ColorWhite       = RGB{235, 245, 255}
	ColorYellow      = RGB{255, 214, 0}
	ColorMagenta     = RGB{255, 0, 214}
	ColorErrorBg     = RGB{84, 8, 28}
	ColorErrorLine   = RGB{255, 72, 96}
	ColorErrorBorder = RGB{255, 214, 0}
)
