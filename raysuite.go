package main

import (
	"fmt"
	"math/rand"
)

type NamedRay struct {
	Group string
	Name  string
	Ray   Ray
}

type RayProbeMismatch struct {
	Group       string       `json:"group"`
	Name        string       `json:"name"`
	Kind        MismatchKind `json:"kind"`
	RefID       int          `json:"ref_id"`
	SimpleID    int          `json:"simple_id"`
	RefCount    int          `json:"ref_count"`
	SimpleCount int          `json:"simple_count"`
	RefT        float64      `json:"ref_t"`
	SimpleT     float64      `json:"simple_t"`
}

type RayProbeReport struct {
	Group      string             `json:"group"`
	Rays       int                `json:"rays"`
	Pass       bool               `json:"pass"`
	Mismatches []RayProbeMismatch `json:"mismatches,omitempty"`
}

func probeRay(from, to Vec3) Ray {
	return InfiniteRay(from, to.Sub(from))
}

func EdgeProbeRays(group string, scene *Scene) []NamedRay {
	origin := V(0.15, 5.4, 9.4)
	if group == "large_triangles" {
		origin = V(0, 3.0, 12)
	}
	probes := make([]NamedRay, 0, len(scene.Triangles)*18)

	for _, tri := range scene.Triangles {
		vertices := tri.Vertices()
		centroid := tri.A.Add(tri.B).Add(tri.C).Scale(1 / 3.0)
		edges := [][2]Vec3{{tri.A, tri.B}, {tri.B, tri.C}, {tri.C, tri.A}}

		for edgeIndex, edge := range edges {
			mid := edge[0].Lerp(edge[1], 0.5)
			inward := centroid.Sub(mid)
			probes = append(probes,
				NamedRay{Group: group, Name: fmt.Sprintf("%s/t%d/e%d/exact", tri.Name, tri.ID, edgeIndex), Ray: probeRay(origin, mid)},
				NamedRay{Group: group, Name: fmt.Sprintf("%s/t%d/e%d/inside-near", tri.Name, tri.ID, edgeIndex), Ray: probeRay(origin, mid.Add(inward.Scale(1e-5)))},
				NamedRay{Group: group, Name: fmt.Sprintf("%s/t%d/e%d/outside-near", tri.Name, tri.ID, edgeIndex), Ray: probeRay(origin, mid.Sub(inward.Scale(1e-5)))},
				NamedRay{Group: group, Name: fmt.Sprintf("%s/t%d/e%d/inside-tight", tri.Name, tri.ID, edgeIndex), Ray: probeRay(origin, mid.Add(inward.Scale(1e-7)))},
			)
		}

		for vertexIndex, vertex := range vertices {
			probes = append(probes, NamedRay{
				Group: group,
				Name:  fmt.Sprintf("%s/t%d/v%d", tri.Name, tri.ID, vertexIndex),
				Ray:   probeRay(origin, vertex),
			})
		}
	}
	return probes
}

func ParallelProbeRays(group string, scene *Scene) []NamedRay {
	probes := make([]NamedRay, 0, len(scene.Triangles)*12)
	for _, tri := range scene.Triangles {
		e1 := tri.B.Sub(tri.A)
		e2 := tri.C.Sub(tri.A)
		normal := e1.Cross(e2).Normalize()
		centroid := tri.A.Add(tri.B).Add(tri.C).Scale(1 / 3.0)
		directions := []Vec3{e1, e2, e1.Add(e2), e1.Scale(-1)}
		offsets := []float64{-0.03, 0, 0.03}

		for dirIndex, direction := range directions {
			for _, offset := range offsets {
				rayOrigin := centroid.Add(normal.Scale(offset))
				probes = append(probes, NamedRay{
					Group: group,
					Name:  fmt.Sprintf("%s/t%d/d%d/o%g", tri.Name, tri.ID, dirIndex, offset),
					Ray:   InfiniteRay(rayOrigin, direction),
				})
			}

			start := tri.A.Sub(direction.Normalize().Scale(3)).Add(normal.Scale(0.015))
			probes = append(probes, NamedRay{
				Group: group,
				Name:  fmt.Sprintf("%s/t%d/d%d/offset-start", tri.Name, tri.ID, dirIndex),
				Ray:   InfiniteRay(start, direction),
			})
		}
	}
	return probes
}

func RandomProbeRays(group string, scene *Scene, count int, seed int64) []NamedRay {
	rng := rand.New(rand.NewSource(seed))
	bounds := SceneBounds(scene)
	probes := make([]NamedRay, 0, count)
	for i := 0; i < count; i++ {
		origin := V(
			bounds.Min.X+rng.Float64()*(bounds.Max.X-bounds.Min.X),
			bounds.Min.Y+rng.Float64()*(bounds.Max.Y-bounds.Min.Y),
			bounds.Max.Z+2+rng.Float64()*8,
		)
		target := V(
			bounds.Min.X+rng.Float64()*(bounds.Max.X-bounds.Min.X),
			bounds.Min.Y+rng.Float64()*(bounds.Max.Y-bounds.Min.Y),
			bounds.Min.Z+rng.Float64()*(bounds.Max.Z-bounds.Min.Z),
		)
		probes = append(probes, NamedRay{
			Group: group,
			Name:  fmt.Sprintf("random/%05d", i),
			Ray:   probeRay(origin, target),
		})
	}
	return probes
}

func SceneBounds(scene *Scene) AABB {
	bounds := EmptyAABB()
	for _, tri := range scene.Triangles {
		bounds = bounds.IncludePoint(tri.A).IncludePoint(tri.B).IncludePoint(tri.C)
	}
	if !bounds.Valid() {
		bounds = AABB{Min: V(-4, 0, -6), Max: V(4, 4, 8)}
	}
	return bounds
}

func ValidateRays(scene *Scene, tracer Tracer, probes []NamedRay) RayProbeReport {
	report := RayProbeReport{Pass: true}
	if len(probes) > 0 {
		report.Group = probes[0].Group
	}
	report.Rays = len(probes)

	for _, probe := range probes {
		ref := TraceReference(scene, probe.Ray)
		simple := tracer.Trace(probe.Ray)
		kind := classifyTrace(ref, simple)
		if kind == KindMatch {
			continue
		}

		refID, refT := -1, 0.0
		if ref.OK {
			refID, refT = ref.Hit.ID, ref.Hit.T
		}
		simpleID, simpleT := -1, 0.0
		if simple.OK {
			simpleID, simpleT = simple.Hit.ID, simple.Hit.T
		}
		report.Pass = false
		report.Mismatches = append(report.Mismatches, RayProbeMismatch{
			Group: probe.Group, Name: probe.Name, Kind: kind,
			RefID: refID, SimpleID: simpleID,
			RefCount: len(ref.Hits), SimpleCount: len(simple.Hits),
			RefT: refT, SimpleT: simpleT,
		})
	}
	return report
}

func DefaultProbeSuite(scene *Scene, randomCount int) []NamedRay {
	group := scene.Name
	probes := EdgeProbeRays(group, scene)
	probes = append(probes, ParallelProbeRays(group, scene)...)
	probes = append(probes, RandomProbeRays(group, scene, randomCount, 0x5EED1234)...)
	return probes
}
