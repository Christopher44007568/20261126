package main

import (
	"fmt"
	"math/rand"
)

func VerifyRaySuite(scene *DemoScene, rt *RelocatedAS, randomCount int) RaySuiteReport {
	report := RaySuiteReport{
		GoldSurfaceID: SurfaceGold,
		Comparisons:   []RayComparison{},
	}

	origLeaf := FindInstance(scene.Root, deepLeafID)
	runtimeLeaf := FindRuntimeInstance(rt.RootInstance, deepLeafID)

	for _, probe := range scene.Probes {
		report.Comparisons = append(report.Comparisons, verifyLocalProbe(scene, rt, probe))
		if origLeaf != nil && runtimeLeaf != nil {
			report.Comparisons = append(report.Comparisons, verifyDeepProbe(scene, rt, probe, origLeaf))
		}
	}

	rng := rand.New(rand.NewSource(0x434F5245))
	for i := 0; i < randomCount; i++ {
		origin := V(-3.4+6.8*rng.Float64(), -1.5+3.0*rng.Float64(), 5.5+rng.Float64())
		target := V(-3.0+6.0*rng.Float64(), -1.25+2.5*rng.Float64(), -0.25+0.5*rng.Float64())
		ray := NewRay(origin, target.Sub(origin))
		a := IntersectOriginalScene(scene, ray)
		b := IntersectRelocatedScene(rt, ray)
		cmp := compareWorldRay(fmt.Sprintf("random_%04d", i), "random_world", ray, a, b, nil, false)
		report.Comparisons = append(report.Comparisons, cmp)
	}

	for _, cmp := range report.Comparisons {
		if cmp.Mode == "random_world" {
			report.RandomRayCount++
			if !cmp.OK {
				report.RandomMismatches++
			}
		} else {
			report.ProbeCount++
			if cmp.OK {
				report.ProbePassed++
			} else {
				report.ProbeFailed++
			}
		}
		if cmp.OK {
			report.Passed++
		} else {
			report.Failed++
		}
		if cmp.PointError > report.MaxPointError {
			report.MaxPointError = cmp.PointError
		}
		if cmp.TError > report.MaxTError {
			report.MaxTError = cmp.TError
		}
	}
	report.OK = report.Failed == 0
	return report
}

func verifyLocalProbe(scene *DemoScene, rt *RelocatedAS, probe Probe) RayComparison {
	oh, oOK := IntersectPointerBVH(scene.BVH, scene.Triangles, probe.LocalRay)
	rh, rOK := intersectRuntimeNode(rt.Root, rt, probe.LocalRay)

	a := Hit{OK: oOK}
	b := Hit{OK: rOK}
	expected := &probe.ExpectedLocal
	if oOK {
		tri := scene.Triangles[oh.Triangle]
		a = Hit{
			OK:               true,
			T:                oh.T,
			Point:            oh.Point,
			Normal:           oh.Normal,
			SurfaceID:        tri.SurfaceID,
			OriginalTriangle: tri.OriginalIndex,
			StorageTriangle:  oh.Triangle,
		}
	}
	if rOK {
		storageTri := rh.Triangle
		tri := rt.Triangles[storageTri]
		b = Hit{
			OK:               true,
			T:                rh.T,
			Point:            rh.Point,
			Normal:           rh.Normal,
			SurfaceID:        tri.SurfaceID,
			OriginalTriangle: tri.OriginalIndex,
			StorageTriangle:  storageTri,
		}
	}
	return compareWorldRay(probe.Name, "direct_bvh", probe.LocalRay, a, b, expected, true)
}

func verifyDeepProbe(scene *DemoScene, rt *RelocatedAS, probe Probe, leaf *Instance) RayComparison {
	worldOrigin := matPoint(leaf.World, probe.LocalRay.Origin)
	worldDir := matDir(leaf.World, probe.LocalRay.Dir).Normalized()
	worldRay := NewRay(worldOrigin, worldDir)
	expected := matPoint(leaf.World, probe.ExpectedLocal)

	a := IntersectOriginalScene(scene, worldRay)
	b := IntersectRelocatedScene(rt, worldRay)
	return compareWorldRay(probe.Name+"/deep_15_levels", "deep_instance", worldRay, a, b, &expected, true)
}

func FindRuntimeInstance(root *RuntimeInstance, stableID uint32) *RuntimeInstance {
	if root == nil {
		return nil
	}
	if root.StableID == stableID {
		return root
	}
	for _, child := range root.Children {
		if found := FindRuntimeInstance(child, stableID); found != nil {
			return found
		}
	}
	return nil
}
