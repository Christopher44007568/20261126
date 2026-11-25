package hotreload

func (e *Engine) evaluateAll() {
	for _, n := range e.order {
		e.evaluate(n)
	}
}

func (e *Engine) evaluate(n *Node) {
	if !n.HasCache() {
		return
	}
	key := e.cacheKey(n)
	if n.cache != nil && n.cache.Key == key {
		// Retained cache: content-addressed input is unchanged.
		return
	}

	entry := &CacheEntry{Key: key}
	switch n.Kind {
	case KindBackground:
		entry.Frame = renderBackground()
	case KindRGBLayer:
		entry.Frame, entry.Err = renderRGBLayer(n.Inputs[0].payload)
	case KindRGBGrade:
		entry.Frame = renderRGBGrade(n.Inputs[0].cache.Frame, n.Inputs[0].cache.Err)
	case KindGeoLayer:
		geo := *n.Inputs[0].geometry
		entry.Geo = &geo
		entry.Frame = renderGeoLayer(entry.Geo)
	case KindSurface:
		geo := n.Inputs[0].cache.Geo
		entry.Geo = geo
		entry.Frame = renderSurface(geo, n.material)
	case KindOutline:
		entry.Frame = renderOutline(n.Inputs[0].cache.Geo)
	case KindComposite:
		bg := n.input("background").cache.Frame
		grade := n.input("rgb.grade").cache.Frame
		surf := n.input("surface").cache.Frame
		outlineF := n.input("outline").cache.Frame
		geoF := n.input("geo.layer").cache.Frame
		entry.Frame = renderComposite(
			bg, grade, surf, outlineF, geoF,
		)
	}

	// Replace atomically: an error frame is a new result and never restores
	// an older successful frame through a later cache lookup.
	n.cache = entry
}

func (e *Engine) cacheKey(n *Node) uint64 {
	switch n.Kind {
	case KindBackground:
		return keyHash(1, "background")
	case KindRGBLayer:
		p := n.Inputs[0].payload
		return keyHash(2, len(p.Pix), p.Pix)
	case KindRGBGrade:
		return keyHash(3, n.Inputs[0].cache.Key, n.Inputs[0].cache.Err != nil)
	case KindGeoLayer:
		g := n.Inputs[0].geometry
		return keyHash(4, g.AssetID, g.Revision, string(g.Shape),
			g.Bounds.X0, g.Bounds.Y0, g.Bounds.X1, g.Bounds.Y1)
	case KindSurface:
		g := n.Inputs[0].cache.Geo
		var matKey any
		if n.material != nil {
			matKey = string(n.material.ID)
		}
		return keyHash(5, n.Inputs[0].cache.Key, matKey, g.AssetID)
	case KindOutline:
		return keyHash(6, n.Inputs[0].cache.Key, "outline")
	case KindComposite:
		parts := []any{uint64(7)}
		for _, in := range n.Inputs {
			parts = append(parts, in.cache.Key)
		}
		return keyHash(parts[0].(uint64), parts[1:]...)
	default:
		return 0
	}
}

// collectErrors walks renderable nodes after evaluation.
func (e *Engine) collectErrors() []NodeError {
	var errs []NodeError
	for _, n := range e.Nodes() {
		if n.cache != nil && n.cache.Err != nil {
			errs = append(errs, *n.cache.Err)
		}
	}
	return errs
}
