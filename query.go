package rayao

import "math"

const (
	// ParallelEpsilon rejects determinants too close to zero for stable
	// barycentric coordinates.
	ParallelEpsilon = 1e-12
	// BarycentricEpsilon makes shared edges and exact vertices watertight.
	BarycentricEpsilon = 1e-10
	// IntervalEpsilon keeps an exact zero-distance start usable after the
	// few ulps introduced by Möller–Trumbore arithmetic.
	IntervalEpsilon = 1e-9
)

// RayQuery is the common input for closest-hit and any-hit traversal.
type RayQuery struct {
	Ray Ray
	// IgnorePrimitive is normally the primary-hit triangle used to spawn a
	// shadow/AO ray. Use -1 to test every primitive, including at t=0.
	IgnorePrimitive int
}

// DefaultRayQuery tests all primitives in the ray's interval.
func DefaultRayQuery(ray Ray) RayQuery {
	return RayQuery{Ray: ray, IgnorePrimitive: -1}
}

// QueryResult contains both hit policies produced by one deterministic scan.
type QueryResult struct {
	SceneVersion string
	Closest      Hit
	HasClosest   bool
	FirstAny     Hit
	HasAny       bool

	Checked             int
	Accepted            int
	IgnoredPrimitive    int
	DuplicateCandidates int
	InvalidCandidates   int
}

// Query evaluates all scene triangles and records closest and first-any hits.
func (s *Scene) Query(q RayQuery) QueryResult {
	return s.QueryCandidates(q, nil)
}

// QueryCandidates is Query with an explicit traversal candidate list. A BVH may
// pass the same primitive more than once when overlapping nodes are visited;
// the query layer treats that as one primitive for this ray.
func (s *Scene) QueryCandidates(q RayQuery, candidateIDs []int) QueryResult {
	if s == nil {
		return QueryResult{}
	}
	return s.scan(q, candidateIDs, true)
}

// ClosestHit returns the minimum-T accepted intersection.
func (s *Scene) ClosestHit(q RayQuery) (Hit, bool) {
	r := s.Query(q)
	return r.Closest, r.HasClosest
}

// AnyHit returns the first accepted intersection in deterministic scene order.
// It is intended for shadow and visibility tests where distance is irrelevant.
func (s *Scene) AnyHit(q RayQuery) (Hit, bool) {
	if s == nil {
		return Hit{}, false
	}
	r := s.scan(q, nil, false)
	return r.FirstAny, r.HasAny
}

func (s *Scene) scan(q RayQuery, candidateIDs []int, all bool) QueryResult {
	result := QueryResult{}
	if s == nil {
		return result
	}
	result.SceneVersion = s.Version
	if !q.Ray.Finite() {
		return result
	}

	if candidateIDs == nil {
		for id := range s.Triangles {
			if s.evaluateCandidate(q, id, &result) && !all {
				return result
			}
		}
		return result
	}

	seen := make(map[int]struct{}, len(candidateIDs))
	for _, id := range candidateIDs {
		if _, duplicate := seen[id]; duplicate {
			result.DuplicateCandidates++
			continue
		}
		seen[id] = struct{}{}

		if s.evaluateCandidate(q, id, &result) && !all {
			return result
		}
	}
	return result
}

func (s *Scene) evaluateCandidate(q RayQuery, id int, result *QueryResult) bool {
	if id < 0 || id >= len(s.Triangles) {
		result.InvalidCandidates++
		return false
	}
	if id == q.IgnorePrimitive {
		result.IgnoredPrimitive++
		return false
	}

	result.Checked++
	hit, ok := intersectTriangle(q.Ray, s.Triangles[id], id)
	if !ok {
		return false
	}

	result.Accepted++
	if !result.HasAny {
		result.FirstAny = hit
		result.HasAny = true
	}
	if !result.HasClosest || hit.T < result.Closest.T ||
		(hit.T == result.Closest.T && hit.Primitive < result.Closest.Primitive) {
		result.Closest = hit
		result.HasClosest = true
	}
	return true
}

func intersectTriangle(r Ray, tri Triangle, id int) (Hit, bool) {
	a := tri.Vertices[0]
	b := tri.Vertices[1]
	c := tri.Vertices[2]
	e1 := b.Sub(a)
	e2 := c.Sub(a)
	pVec := r.Direction.Cross(e2)
	det := e1.Dot(pVec)
	if math.Abs(det) < ParallelEpsilon {
		return Hit{}, false
	}

	invDet := 1.0 / det
	tVec := r.Origin.Sub(a)
	u := tVec.Dot(pVec) * invDet
	if u < -BarycentricEpsilon || u > 1.0+BarycentricEpsilon {
		return Hit{}, false
	}

	qVec := tVec.Cross(e1)
	v := r.Direction.Dot(qVec) * invDet
	if v < -BarycentricEpsilon || u+v > 1.0+BarycentricEpsilon {
		return Hit{}, false
	}

	distance := e2.Dot(qVec) * invDet
	if distance < r.MinT-IntervalEpsilon || distance > r.MaxT+IntervalEpsilon {
		return Hit{}, false
	}
	if distance < r.MinT {
		distance = r.MinT
	}
	if distance > r.MaxT {
		distance = r.MaxT
	}

	hit := tri.hitForRay(r, id, distance, clamp01(u), clamp01(v))
	return hit, true
}
