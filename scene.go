package main

import (
	"fmt"
	"math"
)

const deepLeafID = uint32(99)

type Probe struct {
	Name          string
	LocalRay      Ray
	ExpectedLocal Vec3
	Bary          Vec3
	Grazing       bool
}

type DemoScene struct {
	Triangles []Triangle
	BVH       *PointerBVH
	Root      *Instance
	GoldA     Vec3
	GoldB     Vec3
	GoldC     Vec3
	Probes    []Probe
}

func NewDemoScene() *DemoScene {
	scene := &DemoScene{}
	tris := make([]Triangle, 0, 160)
	add := func(a, b, c Vec3, color RGB, surface uint64) {
		tris = append(tris, NewTriangle(a, b, c, color, surface, len(tris)))
	}

	// Explicitly degenerate baffles. They participate in BVH bounds but can
	// never produce a hit; rays through them continue to the gold surface.
	add(V(-0.22, -0.08, 0.19), V(-0.22, -0.08, 0.19), V(-0.22, -0.08, 0.19),
		V(0.9, 0.2, 0.2), 2001)
	add(V(0.16, 0.08, 0.19), V(0.16, 0.08, 0.19), V(0.27, 0.13, 0.19),
		V(0.95, 0.35, 0.2), 2002)
	add(V(-0.52, 0.22, 0.17), V(0.0, 0.22, 0.17), V(0.52, 0.22, 0.17),
		V(1.0, 0.55, 0.15), 2003)

	const nx, ny = 12, 7
	x0, x1 := -1.55, 1.55
	y0, y1 := -0.92, 0.92
	for iy := 0; iy < ny; iy++ {
		for ix := 0; ix < nx; ix++ {
			xa := x0 + (x1-x0)*float64(ix)/nx
			xb := x0 + (x1-x0)*float64(ix+1)/nx
			ya := y0 + (y1-y0)*float64(iy)/ny
			yb := y0 + (y1-y0)*float64(iy+1)/ny
			z := 0.02 * math.Sin(float64(ix)*0.9+float64(iy)*1.7)
			check := (ix + iy) % 2
			color := V(0.10+0.05*float64(check), 0.34+0.08*float64(iy%3), 0.52+0.08*float64(ix%4))
			if ix == 0 || iy == 0 || ix == nx-1 || iy == ny-1 {
				color = V(0.16, 0.48, 0.58)
			}
			add(V(xa, ya, z), V(xb, ya, z), V(xb, yb, z), color, 3000+uint64(iy*nx+ix))
			add(V(xa, ya, z), V(xb, yb, z), V(xa, yb, z), color.Mul(0.82), 3000+uint64(iy*nx+ix))
		}
	}

	scene.GoldA = V(-0.43, -0.31, 0.115)
	scene.GoldB = V(0.43, -0.31, 0.115)
	scene.GoldC = V(0.43, 0.31, 0.115)
	goldD := V(-0.43, 0.31, 0.115)
	goldColor := V(1.0, 0.68, 0.16)
	add(scene.GoldA, scene.GoldB, scene.GoldC, goldColor, SurfaceGold)
	add(scene.GoldA, scene.GoldC, goldD, goldColor.Mul(0.9), SurfaceGold)

	add(V(-1.72, -1.02, -0.04), V(-1.58, -1.02, -0.04), V(-1.65, -0.82, 0.02),
		V(0.5, 0.25, 0.75), 4001)
	add(V(1.58, 0.82, -0.03), V(1.76, 0.86, -0.03), V(1.66, 1.03, 0.03),
		V(0.35, 0.75, 0.85), 4002)
	add(V(-0.1, -1.12, 0.0), V(0.0, -1.12, 0.0), V(-0.05, -0.98, 0.05),
		V(0.8, 0.35, 0.55), 4003)

	tris = append(tris, NewTriangle(V(1.8, -1.05, 0), V(1.8, -1.05, 0), V(1.95, -0.92, 0),
		V(0.9, 0.2, 0.2), 2004, len(tris)))

	scene.Triangles = tris
	for i := range scene.Triangles {
		scene.Triangles[i].OriginalIndex = i
	}
	scene.BVH = BuildPointerBVH(scene.Triangles)
	scene.Root = buildInstanceTree()
	scene.Root.UpdateWorld(IdentityMat4(), 0)
	scene.Probes = buildProbes(scene)
	return scene
}

func buildInstanceTree() *Instance {
	root := NewGroup(0, "world", IdentityMat4())
	root.Add(NewMeshInstance(1, "main_panel", 0, TRS(V(-1.12, 0, 0), 0, V(1, 1, 1))))

	base := NewGroup(2, "deep_base", TRS(V(1.48, -0.08, 0), 0.07, V(1, 1, 1)))
	root.Add(base)
	cur := base
	for depth := 1; depth <= 12; depth++ {
		angle := 0.028 + 0.006*math.Sin(float64(depth))
		shift := V(0.041, 0.012*math.Sin(float64(depth)*1.71), 0)
		tr := TRS(shift, angle, V(0.982, 0.982, 1))
		child := NewGroup(uint32(10+depth), fmt.Sprintf("relocation_chain_%02d", depth), tr)
		cur.Add(child)
		cur = child
	}
	leaf := NewMeshInstance(deepLeafID, "deep_gold_surface", 0,
		TRS(V(0.43, 0, 0), -0.16, V(0.34, 0.34, 0.34)))
	cur.Add(leaf)
	return root
}

func baryPoint(a, b, c Vec3, wa, wb, wc float64) Vec3 {
	return a.Mul(wa).Add(b.Mul(wb)).Add(c.Mul(wc))
}

func buildProbes(scene *DemoScene) []Probe {
	a, b, c := scene.GoldA, scene.GoldB, scene.GoldC
	type target struct {
		name string
		p    Vec3
		bary Vec3
	}
	targets := []target{
		{"center", baryPoint(a, b, c, 1/3.0, 1/3.0, 1/3.0), V(1/3.0, 1/3.0, 1/3.0)},
		{"edge_ab_exact", baryPoint(a, b, c, 0.5, 0.5, 0), V(0, 0.5, 0.5)},
		{"edge_bc_exact", baryPoint(a, b, c, 0, 0.5, 0.5), V(0.5, 0, 0.5)},
		{"edge_ca_exact", baryPoint(a, b, c, 0.5, 0, 0.5), V(0.5, 0.5, 0)},
		{"near_a_edge", baryPoint(a, b, c, 0.985, 0.0075, 0.0075), V(0.985, 0.0075, 0.0075)},
		{"near_b_edge", baryPoint(a, b, c, 0.008, 0.984, 0.008), V(0.008, 0.984, 0.008)},
		{"near_c_edge", baryPoint(a, b, c, 0.008, 0.008, 0.984), V(0.008, 0.008, 0.984)},
		{"interior_ab", baryPoint(a, b, c, 0.02, 0.49, 0.49), V(0.02, 0.49, 0.49)},
		{"interior_bc", baryPoint(a, b, c, 0.49, 0.02, 0.49), V(0.49, 0.02, 0.49)},
		{"interior_ca", baryPoint(a, b, c, 0.49, 0.49, 0.02), V(0.49, 0.49, 0.02)},
	}
	angles := []Vec3{
		V(0.78, 0.28, -0.075),
		V(-0.58, 0.62, -0.068),
		V(0.18, -0.91, -0.061),
		V(-0.82, -0.31, -0.083),
	}

	probes := make([]Probe, 0, len(targets)*len(angles)+2)
	for ti, tg := range targets {
		for ai, rawDir := range angles {
			dir := rawDir.Normalized()
			distance := 0.55 + 0.08*float64((ti+ai)%3)
			origin := tg.p.Sub(dir.Mul(distance))
			probes = append(probes, Probe{
				Name:          fmt.Sprintf("%s/graze_%d", tg.name, ai),
				LocalRay:      Ray{Origin: origin, Dir: dir, TMin: 1e-7, TMax: math.Inf(1)},
				ExpectedLocal: tg.p,
				Bary:          tg.bary,
				Grazing:       true,
			})
		}
	}

	for i, p := range []Vec3{
		baryPoint(a, b, c, 0.2, 0.3, 0.5),
		baryPoint(a, b, c, 0.6, 0.2, 0.2),
	} {
		dir := V(0.04*float64(i+1), -0.02, -1).Normalized()
		probes = append(probes, Probe{
			Name:          fmt.Sprintf("near_vertical_%d", i),
			LocalRay:      Ray{Origin: p.Sub(dir.Mul(1.2)), Dir: dir, TMin: 1e-7, TMax: math.Inf(1)},
			ExpectedLocal: p,
			Bary:          V(0, 0, 0),
		})
	}

	degenerateGateways := []struct {
		name   string
		gate   Vec3
		target Vec3
	}{
		{"point", V(-0.22, -0.08, 0.19), baryPoint(a, b, c, 0.55, 0.25, 0.20)},
		{"duplicate_vertex", V(0.16, 0.08, 0.19), baryPoint(a, b, c, 0.20, 0.45, 0.35)},
		{"collinear_edge", V(-0.52, 0.22, 0.17), baryPoint(a, b, c, 0.12, 0.18, 0.70)},
	}
	for i, gateway := range degenerateGateways {
		dir := gateway.target.Sub(gateway.gate).Normalized()
		origin := gateway.gate.Sub(dir.Mul(0.09))
		probes = append(probes, Probe{
			Name:          fmt.Sprintf("degenerate_gateway_%s", gateway.name),
			LocalRay:      Ray{Origin: origin, Dir: dir, TMin: 1e-7, TMax: math.Inf(1)},
			ExpectedLocal: gateway.target,
			Bary:          V(float64(i), 0, 0),
		})
	}
	return probes
}

func FindInstance(root *Instance, id uint32) *Instance {
	if root == nil {
		return nil
	}
	if root.ID == id {
		return root
	}
	for _, child := range root.Children {
		if found := FindInstance(child, id); found != nil {
			return found
		}
	}
	return nil
}
