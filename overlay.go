package main

var boxEdges = [12][2]int{
	{0, 1}, {1, 3}, {3, 2}, {2, 0},
	{4, 5}, {5, 7}, {7, 6}, {6, 4},
	{0, 4}, {1, 5}, {2, 6}, {3, 7},
}

func AABBCorners(b AABB) [8]Vec3 {
	return [8]Vec3{
		V(b.Min.X, b.Min.Y, b.Min.Z),
		V(b.Max.X, b.Min.Y, b.Min.Z),
		V(b.Min.X, b.Max.Y, b.Min.Z),
		V(b.Max.X, b.Max.Y, b.Min.Z),
		V(b.Min.X, b.Min.Y, b.Max.Z),
		V(b.Max.X, b.Min.Y, b.Max.Z),
		V(b.Min.X, b.Max.Y, b.Max.Z),
		V(b.Max.X, b.Max.Y, b.Max.Z),
	}
}

func DrawProjectedBox(canvas *Canvas, cam Camera, corners [8]Vec3, color RGB, alpha float64) {
	var points [8]struct {
		x, y float64
		ok   bool
	}
	for i, corner := range corners {
		points[i].x, points[i].y, points[i].ok = cam.Project(corner)
	}
	for _, edge := range boxEdges {
		a := points[edge[0]]
		b := points[edge[1]]
		if !a.ok || !b.ok {
			continue
		}
		canvas.LineAlpha(int(a.x+0.5), int(a.y+0.5), int(b.x+0.5), int(b.y+0.5), color, alpha)
	}
}

func DrawTransformedAABB(canvas *Canvas, cam Camera, bounds AABB, world Mat4, color RGB, alpha float64) {
	localCorners := AABBCorners(bounds)
	var worldCorners [8]Vec3
	for i, corner := range localCorners {
		worldCorners[i] = matPoint(world, corner)
	}
	DrawProjectedBox(canvas, cam, worldCorners, color, alpha)
}

func DrawOriginalBVHOverlay(scene *DemoScene, cam Camera, canvas *Canvas) {
	var walkInstances func(inst *Instance)
	walkInstances = func(inst *Instance) {
		if inst.MeshID >= 0 {
			color := V(0.25, 0.95, 0.78)
			alpha := 0.20
			if inst.ID == deepLeafID {
				color = V(0.72, 0.52, 1.0)
				alpha = 0.34
			}
			var walkNodes func(node *PointerNode)
			walkNodes = func(node *PointerNode) {
				if node == nil {
					return
				}
				if node.Left == nil && node.Right == nil {
					DrawTransformedAABB(canvas, cam, node.Bounds, inst.World, color, alpha)
				}
				walkNodes(node.Left)
				walkNodes(node.Right)
			}
			walkNodes(scene.BVH.Root)
		}
		for _, child := range inst.Children {
			walkInstances(child)
		}
	}
	walkInstances(scene.Root)
}

func DrawRelocatedBVHOverlay(rt *RelocatedAS, cam Camera, canvas *Canvas) {
	var walkInstances func(inst *RuntimeInstance)
	walkInstances = func(inst *RuntimeInstance) {
		if inst.MeshID >= 0 {
			color := V(0.25, 0.95, 0.78)
			alpha := 0.20
			if inst.StableID == deepLeafID {
				color = V(0.72, 0.52, 1.0)
				alpha = 0.34
			}
			var walkNodes func(node *RuntimeNode)
			walkNodes = func(node *RuntimeNode) {
				if node == nil {
					return
				}
				if node.Left == nil && node.Right == nil {
					DrawTransformedAABB(canvas, cam, node.Bounds, inst.World, color, alpha)
				}
				walkNodes(node.Left)
				walkNodes(node.Right)
			}
			walkNodes(rt.Root)
		}
		for _, child := range inst.Children {
			walkInstances(child)
		}
	}
	walkInstances(rt.RootInstance)
}
