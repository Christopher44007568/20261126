package main

type OmitTracer struct {
	Scene   *Scene
	Omitted map[int]bool
}

func (o OmitTracer) Trace(ray Ray) TraceInfo {
	info := TraceInfo{Tests: len(o.Scene.Triangles)}
	for i := range o.Scene.Triangles {
		if o.Omitted[i] {
			continue
		}
		if hit, ok := o.Scene.Triangles[i].Intersect(ray); ok {
			info.Hits = append(info.Hits, hit)
		}
	}
	sortedHits(info.Hits)
	if hit, ok := nearestHit(info.Hits); ok {
		info.Hit = hit
		info.OK = true
	}
	return info
}
