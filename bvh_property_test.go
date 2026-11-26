package main

import (
	"math/rand"
	"testing"
)

func TestExtremeScaleTriangleProbes(t *testing.T) {
	const scale = 1e6
	scene := &Scene{Triangles: []Triangle{{
		ID: 0,
		A:  V(-scale, -scale, 0),
		B:  V(scale, -scale, 0),
		C:  V(0, scale, 0),
	}}}
	accel := NewSimplifiedAccel(scene.Triangles)

	probes := []NamedRay{
		{Group: "extreme", Name: "center", Ray: probeRay(V(0, 0, 10), V(0, 0, 0))},
		{Group: "extreme", Name: "edge-ab", Ray: probeRay(V(0, 0, 10), scene.Triangles[0].A.Lerp(scene.Triangles[0].B, .5))},
		{Group: "extreme", Name: "edge-bc", Ray: probeRay(V(0, 0, 10), scene.Triangles[0].B.Lerp(scene.Triangles[0].C, .5))},
		{Group: "extreme", Name: "edge-ca", Ray: probeRay(V(0, 0, 10), scene.Triangles[0].C.Lerp(scene.Triangles[0].A, .5))},
		{Group: "extreme", Name: "vertex-a", Ray: probeRay(V(0, 0, 10), scene.Triangles[0].A)},
		{Group: "extreme", Name: "vertex-b", Ray: probeRay(V(0, 0, 10), scene.Triangles[0].B)},
		{Group: "extreme", Name: "vertex-c", Ray: probeRay(V(0, 0, 10), scene.Triangles[0].C)},
	}
	probes = append(probes, ParallelProbeRays("extreme", scene)...)
	probes = append(probes, RandomProbeRays("extreme", scene, 512, 424242)...)
	report := ValidateRays(scene, accel, probes)
	assertRayReportPass(t, report)
}

func TestRandomScenesBVHMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(0xBADC0DE))
	for iteration := 0; iteration < 24; iteration++ {
		count := 1 + rng.Intn(42)
		triangles := make([]Triangle, 0, count)
		for id := 0; id < count; id++ {
			center := V((rng.Float64()*2-1)*5, (rng.Float64()*2-1)*4, (rng.Float64()*2-1)*6)
			size := 0.25 + rng.Float64()*3.5
			triangles = append(triangles, Triangle{
				ID: id,
				A:  center.Add(V((rng.Float64()*2-1)*size, (rng.Float64()*2-1)*size, (rng.Float64()*2-1)*size)),
				B:  center.Add(V((rng.Float64()*2-1)*size, (rng.Float64()*2-1)*size, (rng.Float64()*2-1)*size)),
				C:  center.Add(V((rng.Float64()*2-1)*size, (rng.Float64()*2-1)*size, (rng.Float64()*2-1)*size)),
			})
		}
		scene := &Scene{Name: "random", Triangles: triangles}
		accel := NewSimplifiedAccel(triangles)

		for rayIndex := 0; rayIndex < 240; rayIndex++ {
			origin := V((rng.Float64()*2-1)*9, (rng.Float64()*2-1)*7, 4+rng.Float64()*10)
			target := V((rng.Float64()*2-1)*8, (rng.Float64()*2-1)*6, (rng.Float64()*2-1)*10)
			ray := probeRay(origin, target)
			ref := TraceReference(scene, ray)
			simple := accel.Trace(ray)
			if kind := classifyTrace(ref, simple); kind != KindMatch {
				t.Fatalf("iter=%d ray=%d kind=%s ref=%+v simple=%+v origin=%v target=%v",
					iteration, rayIndex, kind, ref.Hit, simple.Hit, origin, target)
			}
		}
	}
}

func TestOddPixelDimensionsStayStrict(t *testing.T) {
	for _, dims := range [][2]int{{17, 11}, {31, 19}, {64, 63}} {
		scene := GrazingScene(dims[0], dims[1])
		result := RenderSuite(scene, NewSimplifiedAccel(scene.Triangles))
		assertSuitePass(t, result)
	}
}

func TestBVHLeafIDsAreUniqueAndComplete(t *testing.T) {
	for _, scene := range AllScenes(48, 30) {
		t.Run(scene.Name, func(t *testing.T) {
			accel := NewSimplifiedAccel(scene.Triangles)
			seen := make(map[int]int, len(scene.Triangles))
			var walk func(int)
			walk = func(nodeIndex int) {
				node := accel.BVH.Nodes[nodeIndex]
				if len(node.Triangles) > 0 {
					for _, id := range node.Triangles {
						seen[id]++
					}
					return
				}
				if node.Left >= 0 {
					walk(node.Left)
				}
				if node.Right >= 0 {
					walk(node.Right)
				}
			}
			walk(0)
			if len(seen) != len(scene.Triangles) {
				t.Fatalf("BVH contains %d unique triangles, want %d", len(seen), len(scene.Triangles))
			}
			for id, count := range seen {
				if count != 1 {
					t.Fatalf("triangle %d appears %d times", id, count)
				}
			}
		})
	}
}
