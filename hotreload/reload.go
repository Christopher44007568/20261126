package hotreload

// EdgeChange is one changed data-flow edge in a hot reload report.
type EdgeChange struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Kind   string `json:"kind"`
	Change string `json:"change"`
}

// DiffStats summarizes before/after frame differences.
type DiffStats struct {
	ChangedPixels int     `json:"changed_pixels"`
	TotalPixels   int     `json:"total_pixels"`
	Percent       float64 `json:"percent"`
	Bounds        Bounds  `json:"bounds"`
	Threshold     int     `json:"threshold"`
}

// Bounds is the bounding box of changed pixels.
type Bounds struct {
	MinX  int  `json:"min_x"`
	MinY  int  `json:"min_y"`
	MaxX  int  `json:"max_x"`
	MaxY  int  `json:"max_y"`
	Valid bool `json:"valid"`
}

// ReloadReport records changed edges, recomputation and retained caches.
type ReloadReport struct {
	OK                bool         `json:"ok"`
	Operation         string       `json:"operation"`
	Target            string       `json:"target"`
	Revision          int          `json:"revision"`
	ChangedEdges      []EdgeChange `json:"changed_edges"`
	RecomputedNodes   []string     `json:"recomputed_nodes"`
	CacheHits         []string     `json:"cache_hits"`
	CacheRetained     []string     `json:"cache_retained"`
	CacheInvalidated  []string     `json:"cache_invalidated"`
	DetachedMaterials []string     `json:"detached_materials,omitempty"`
	BoundMaterials    []string     `json:"bound_materials,omitempty"`
	Errors            []NodeError  `json:"errors,omitempty"`
	Diff              DiffStats    `json:"diff"`
}

// ReloadRGB hot-reloads only the RGB payload and recomputes the rgb.layer ->
// rgb.grade -> composite subtree. Geometry and surface caches are retained.
func (e *Engine) ReloadRGB(payload []byte) (*Frame, *Frame, *Frame, ReloadReport) {
	before := cloneFrame(e.Frame())

	source := e.payloadNode
	source.payload = &RGBPayload{Revision: e.revision + 1, Pix: payload}

	affected := e.subtree(source)
	edgeSet := map[[2]string]EdgeChange{
		{"rgb.payload", "rgb.layer"}: {
			From: "rgb.payload", To: "rgb.layer", Kind: "rgb-payload",
			Change: "payload-reloaded",
		},
		{"rgb.layer", "rgb.grade"}: {
			From: "rgb.layer", To: "rgb.grade", Kind: "rgb-process",
			Change: "rgb-subtree-refreshed",
		},
		{"rgb.grade", "composite"}: {
			From: "rgb.grade", To: "composite", Kind: "composite",
			Change: "composite-refreshed",
		},
	}

	e.revision++
	recomputed, retained := e.evaluateWithTracking(affected)
	after := e.Frame()
	diffFrame, stats := ComputeDiff(before, after)
	errs := e.collectErrors()

	report := ReloadReport{
		OK:               len(errs) == 0,
		Operation:        "reload-rgb",
		Target:           "rgb.layer",
		Revision:         e.revision,
		ChangedEdges:     orderedEdges(edgeSet, rgbEdgeOrder),
		RecomputedNodes:  recomputed,
		CacheHits:        retained,
		CacheRetained:    retained,
		CacheInvalidated: recomputed,
		Errors:           errs,
		Diff:             stats,
	}
	return before, after, diffFrame, report
}

var rgbEdgeOrder = [][2]string{
	{"rgb.payload", "rgb.layer"},
	{"rgb.layer", "rgb.grade"},
	{"rgb.grade", "composite"},
}

// ReplaceGeometry swaps a geometry asset, refreshes surface and outline, and
// detaches incompatible old materials so stale shaders cannot bind new meshes.
func (e *Engine) ReplaceGeometry(geo *GeometrySpec, bindFallback bool) (
	*Frame, *Frame, *Frame, ReloadReport,
) {
	before := cloneFrame(e.Frame())

	source := e.geoNode
	oldMat := e.matNode.material
	source.geometry = geo

	var detached, bound []string
	surface := e.node("surface")
	if oldMat != nil && !oldMat.Compatible(*geo) {
		detached = append(detached, string(oldMat.ID))
		surface.material = nil
		if bindFallback {
			fallback := defaultMaterial(MatCheckAmber)
			surface.material = &fallback
			bound = append(bound, string(MatCheckAmber))
		}
	}

	affected := e.subtree(source)
	edgeSet := map[[2]string]EdgeChange{}
	if len(detached) > 0 {
		edgeSet[[2]string{"material", "surface"}] = EdgeChange{
			From: "material", To: "surface", Kind: "material",
			Change: "detached-from-new-geometry:" + detached[0],
		}
	}
	for _, key := range geoEdgeOrder {
		edgeSet[key] = geoEdgeChange(key)
	}

	e.revision++
	recomputed, retained := e.evaluateWithTracking(affected)
	after := e.Frame()
	diffFrame, stats := ComputeDiff(before, after)
	errs := e.collectErrors()

	report := ReloadReport{
		OK:                len(errs) == 0,
		Operation:         "replace-geometry",
		Target:            "geo.layer",
		Revision:          e.revision,
		ChangedEdges:      orderedEdges(edgeSet, edgeOrderWithMaterial(len(detached) > 0)),
		RecomputedNodes:   recomputed,
		CacheHits:         retained,
		CacheRetained:     retained,
		CacheInvalidated:  recomputed,
		DetachedMaterials: detached,
		BoundMaterials:    bound,
		Errors:            errs,
		Diff:              stats,
	}
	return before, after, diffFrame, report
}

var geoEdgeOrder = [][2]string{
	{"geo.asset", "geo.layer"},
	{"geo.layer", "surface"},
	{"geo.layer", "outline"},
	{"geo.layer", "composite"},
	{"surface", "composite"},
	{"outline", "composite"},
}

func edgeOrderWithMaterial(hasMaterial bool) [][2]string {
	if !hasMaterial {
		return geoEdgeOrder
	}
	order := make([][2]string, 0, len(geoEdgeOrder)+1)
	order = append(order, [2]string{"material", "surface"})
	return append(order, geoEdgeOrder...)
}

func geoEdgeChange(key [2]string) EdgeChange {
	change := "composite-refreshed"
	switch key {
	case [2]string{"geo.asset", "geo.layer"}:
		change = "geometry-replaced"
	case [2]string{"geo.layer", "surface"}:
		change = "geometry-cache-invalidated"
	case [2]string{"geo.layer", "outline"}:
		change = "outline-refreshed"
	}
	return EdgeChange{From: key[0], To: key[1], Kind: edgeKind(key), Change: change}
}

func edgeKind(key [2]string) string {
	switch key {
	case [2]string{"geo.asset", "geo.layer"}:
		return "geometry-asset"
	case [2]string{"geo.layer", "surface"}:
		return "geometry"
	case [2]string{"geo.layer", "outline"}:
		return "outline"
	case [2]string{"material", "surface"}:
		return "material"
	default:
		return "composite"
	}
}

func orderedEdges(set map[[2]string]EdgeChange, order [][2]string) []EdgeChange {
	out := make([]EdgeChange, 0, len(set))
	for _, key := range order {
		if edge, ok := set[key]; ok {
			out = append(out, edge)
		}
	}
	return out
}

func (e *Engine) node(id string) *Node {
	return e.nodes[id]
}

// subtree returns every renderable node reachable from a source, including
// composite nodes reached through any changed branch.
func (e *Engine) subtree(root *Node) map[string]bool {
	seen := map[string]bool{}
	queue := []*Node{root}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, edge := range cur.EdgesIn {
			if edge.From != cur {
				continue
			}
			next := edge.To
			if next.HasCache() {
				seen[next.ID] = true
			}
			queue = append(queue, next)
		}
	}
	return seen
}

// evaluateWithTracking re-runs the whole topological evaluation. Nodes whose
// content-addressed key changes are recomputed; unchanged nodes retain caches
// and are reported as hits.
func (e *Engine) evaluateWithTracking(affected map[string]bool) ([]string, []string) {
	var recomputed, retained []string
	for _, n := range e.Nodes() {
		oldKey := uint64(0)
		if n.cache != nil {
			oldKey = n.cache.Key
		}
		newKey := e.cacheKey(n)
		if oldKey == newKey {
			retained = append(retained, n.ID)
			continue
		}
		if !affected[n.ID] {
			// Content-addressed caches guarantee that only the reachable
			// subtree can change; fail loudly if that invariant breaks.
			panic("hotreload: node outside changed subtree changed key: " + n.ID)
		}
		recomputed = append(recomputed, n.ID)
		e.evaluate(n)
	}
	return recomputed, retained
}

func cloneFrame(f *Frame) *Frame {
	pix := make([]byte, len(f.Pix))
	copy(pix, f.Pix)
	return &Frame{W: f.W, H: f.H, Pix: pix}
}
