package hotreload

import "hash/fnv"

// Canvas dimensions shared by the demo scene and PNG outputs.
const (
	CanvasW = 480
	CanvasH = 300
)

// Panel layout of the two viewports painted by the composite node.
var (
	LeftPanel  = Rect{X0: 28, Y0: 58, X1: 232, Y1: 246}
	LeftInner  = Rect{X0: 38, Y0: 68, X1: 222, Y1: 236}
	RightPanel = Rect{X0: 255, Y0: 58, X1: 455, Y1: 246}
	RightPaint = Rect{X0: 260, Y0: 62, X1: 452, Y1: 244}
	LeftPaint  = Rect{X0: 30, Y0: 60, X1: 230, Y1: 244}
)

type NodeKind string

const (
	KindSource     NodeKind = "source"
	KindBackground NodeKind = "background"
	KindRGBLayer   NodeKind = "rgb-layer"
	KindRGBGrade   NodeKind = "rgb-grade"
	KindGeoLayer   NodeKind = "geo-layer"
	KindSurface    NodeKind = "surface"
	KindOutline    NodeKind = "outline"
	KindComposite  NodeKind = "composite"
)

// Node is one vertex of the render graph. Source nodes carry mutable
// payloads; renderable nodes own a content-addressed cache entry.
type Node struct {
	ID      string
	Kind    NodeKind
	Inputs  []*Node
	EdgesIn []*Edge

	payload  *RGBPayload
	geometry *GeometrySpec
	material *MaterialSpec

	cache *CacheEntry
}

// Edge is a typed data-flow edge reported when its downstream subtree changes.
type Edge struct {
	From *Node
	To   *Node
	Kind string
}

// CacheEntry binds a deterministic key to a rendered frame and any node
// error. A key mismatch always forces recomputation, so a hit can never mask
// a failure produced by newly reloaded input.
type CacheEntry struct {
	Key   uint64
	Frame *Frame
	Err   *NodeError
	Geo   *GeometrySpec
}

func (n *Node) HasCache() bool {
	switch n.Kind {
	case KindBackground, KindRGBLayer, KindRGBGrade, KindGeoLayer,
		KindSurface, KindOutline, KindComposite:
		return true
	default:
		return false
	}
}

func (n *Node) input(id string) *Node {
	for _, in := range n.Inputs {
		if in.ID == id {
			return in
		}
	}
	return nil
}

// Engine owns the graph and drives hot-reload evaluations.
type Engine struct {
	revision int
	nodes    map[string]*Node
	order    []*Node
	edges    []*Edge

	payloadNode *Node
	geoNode     *Node
	matNode     *Node
	composite   *Node
}

func keyHash(seed uint64, parts ...any) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	writeUint := func(v uint64) {
		for i := 0; i < 8; i++ {
			buf[i] = byte(v >> (i * 8))
		}
		_, _ = h.Write(buf[:])
	}
	writeUint(seed)
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			_, _ = h.Write([]byte(v))
		case int:
			writeUint(uint64(v))
		case uint64:
			writeUint(v)
		case []byte:
			_, _ = h.Write(v)
		case bool:
			if v {
				_, _ = h.Write([]byte{1})
			} else {
				_, _ = h.Write([]byte{0})
			}
		case nil:
			_, _ = h.Write([]byte("nil"))
		}
	}
	return h.Sum64()
}

func (e *Engine) addNode(id string, kind NodeKind, inputs ...*Node) *Node {
	n := &Node{ID: id, Kind: kind, Inputs: inputs}
	e.nodes[id] = n
	e.order = append(e.order, n)
	for _, in := range inputs {
		edge := &Edge{From: in, To: n}
		e.edges = append(e.edges, edge)
		in.EdgesIn = append(in.EdgesIn, edge)
		n.EdgesIn = append(n.EdgesIn, edge)
	}
	return n
}

// NewEngine constructs the reference scene:
// rgb.payload -> rgb.layer -> rgb.grade ─┐
// geo.asset -> geo.layer -> surface/outline -> composite
// material -> surface; background -> composite.
func NewEngine() *Engine {
	e := &Engine{revision: 1, nodes: map[string]*Node{}}

	background := e.addNode("background", KindBackground)

	payload := e.addNode("rgb.payload", KindSource)
	rgbLayer := e.addNode("rgb.layer", KindRGBLayer, payload)
	rgbGrade := e.addNode("rgb.grade", KindRGBGrade, rgbLayer)

	asset := e.addNode("geo.asset", KindSource)
	geoLayer := e.addNode("geo.layer", KindGeoLayer, asset)
	material := e.addNode("material", KindSource)
	surface := e.addNode("surface", KindSurface, geoLayer, material)
	outline := e.addNode("outline", KindOutline, geoLayer)

	composite := e.addNode("composite", KindComposite,
		background, rgbGrade, surface, outline, geoLayer)

	edgeKinds := map[[2]string]string{
		{"rgb.payload", "rgb.layer"}: "rgb-payload",
		{"rgb.layer", "rgb.grade"}:   "rgb-process",
		{"rgb.grade", "composite"}:   "composite",
		{"geo.asset", "geo.layer"}:   "geometry-asset",
		{"geo.layer", "surface"}:     "geometry",
		{"geo.layer", "outline"}:     "outline",
		{"material", "surface"}:      "material",
		{"geo.layer", "composite"}:   "composite",
		{"surface", "composite"}:     "composite",
		{"outline", "composite"}:     "composite",
		{"background", "composite"}:  "composite",
	}
	for _, edge := range e.edges {
		edge.Kind = edgeKinds[[2]string{edge.From.ID, edge.To.ID}]
	}

	e.payloadNode = payload
	e.geoNode = asset
	e.matNode = material
	e.composite = composite

	payload.payload = &RGBPayload{Revision: 1, Pix: InitialRGBPayload()}
	asset.geometry = InitialGeometry()
	material.material = &MaterialSpec{
		ID:      MatStripes,
		Palette: [2]RGB{{116, 72, 220}, {72, 226, 210}},
	}
	surface.material = material.material

	e.evaluateAll()
	return e
}

// Frame returns the current composite output.
func (e *Engine) Frame() *Frame {
	return e.composite.cache.Frame
}

// Nodes returns renderable nodes in deterministic topological order.
func (e *Engine) Nodes() []*Node {
	out := make([]*Node, 0, len(e.order))
	for _, n := range e.order {
		if n.HasCache() {
			out = append(out, n)
		}
	}
	return out
}
