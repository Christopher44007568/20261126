package main

import "math"

func IntersectOriginalScene(scene *DemoScene, ray Ray) Hit {
	if scene == nil || scene.Root == nil || scene.BVH == nil {
		return Hit{}
	}
	return intersectOriginalInstance(scene.Root, scene, ray, nil)
}

func intersectOriginalInstance(inst *Instance, scene *DemoScene, ray Ray, parentPath []uint32) Hit {
	path := append(append([]uint32{}, parentPath...), inst.ID)
	var best Hit
	bestT := ray.TMax

	if inst.MeshID >= 0 && int(inst.MeshID) == 0 {
		localRay := TransformRayToLocal(ray, inst.WorldInv)
		localRay.TMax = math.Inf(1)
		hit, ok := IntersectPointerBVH(scene.BVH, scene.Triangles, localRay)
		if ok {
			tri := scene.Triangles[hit.Triangle]
			worldHit := LocalHitToWorld(ray, hit, tri, inst.World)
			if worldHit.T >= ray.TMin && worldHit.T <= bestT {
				worldHit.InstancePath = append([]uint32{}, path...)
				worldHit.Depth = len(path)
				worldHit.StorageTriangle = hit.Triangle
				best = worldHit
				bestT = worldHit.T
			}
		}
	}

	for _, child := range inst.Children {
		childRay := ray
		childRay.TMax = bestT
		childHit := intersectOriginalInstance(child, scene, childRay, path)
		if childHit.OK && childHit.T < bestT {
			best = childHit
			bestT = childHit.T
		}
	}
	return best
}
