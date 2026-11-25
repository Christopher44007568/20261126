package main

import "math"

type Ray struct {
	Origin Vec3
	Dir    Vec3
	Inv    Vec3
}

func NewRay(origin, dir Vec3) Ray {
	d := dir.Norm()
	return Ray{
		Origin: origin,
		Dir:    d,
		Inv:    V(1/d.X, 1/d.Y, 1/d.Z),
	}
}

type AABB struct {
	Min Vec3
	Max Vec3
}

func EmptyAABB() AABB {
	inf := math.Inf(1)
	return AABB{Min: V(inf, inf, inf), Max: V(-inf, -inf, -inf)}
}

func (b *AABB) Add(p Vec3) {
	b.Min = b.Min.Min(p)
	b.Max = b.Max.Max(p)
}

func (b AABB) Union(other AABB) AABB {
	return AABB{Min: b.Min.Min(other.Min), Max: b.Max.Max(other.Max)}
}

func (b AABB) Centroid() Vec3 {
	return b.Min.Add(b.Max).Mul(0.5)
}

func (b AABB) Intersect(r Ray, maxT float64) (float64, bool) {
	tMin, tMax := 0.0, maxT
	origin := [3]float64{r.Origin.X, r.Origin.Y, r.Origin.Z}
	inv := [3]float64{r.Inv.X, r.Inv.Y, r.Inv.Z}
	lo := [3]float64{b.Min.X, b.Min.Y, b.Min.Z}
	hi := [3]float64{b.Max.X, b.Max.Y, b.Max.Z}

	for axis := 0; axis < 3; axis++ {
		t1 := (lo[axis] - origin[axis]) * inv[axis]
		t2 := (hi[axis] - origin[axis]) * inv[axis]
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tMin = math.Max(tMin, t1)
		tMax = math.Min(tMax, t2)
		if tMax < tMin {
			return 0, false
		}
	}
	return tMin, true
}

func TransformAABB(b AABB, m Mat4) AABB {
	out := EmptyAABB()
	for _, x := range []float64{b.Min.X, b.Max.X} {
		for _, y := range []float64{b.Min.Y, b.Max.Y} {
			for _, z := range []float64{b.Min.Z, b.Max.Z} {
				out.Add(m.Point(V(x, y, z)))
			}
		}
	}
	return out
}

type Triangle struct {
	P [3]Vec3
	N [3]Vec3
}

func NewTriangle(a, b, c, na, nb, nc Vec3) Triangle {
	return Triangle{P: [3]Vec3{a, b, c}, N: [3]Vec3{na, nb, nc}}
}

func quad(a, b, c, d, normal Vec3) []Triangle {
	return []Triangle{
		NewTriangle(a, b, c, normal, normal, normal),
		NewTriangle(a, c, d, normal, normal, normal),
	}
}

func BoxMesh(center, half Vec3) []Triangle {
	x0, x1 := center.X-half.X, center.X+half.X
	y0, y1 := center.Y-half.Y, center.Y+half.Y
	z0, z1 := center.Z-half.Z, center.Z+half.Z

	var tris []Triangle
	tris = append(tris, quad(V(x1, y0, z0), V(x1, y1, z0), V(x1, y1, z1), V(x1, y0, z1), V(1, 0, 0))...)
	tris = append(tris, quad(V(x0, y0, z1), V(x0, y1, z1), V(x0, y1, z0), V(x0, y0, z0), V(-1, 0, 0))...)
	tris = append(tris, quad(V(x0, y1, z0), V(x0, y1, z1), V(x1, y1, z1), V(x1, y1, z0), V(0, 1, 0))...)
	tris = append(tris, quad(V(x0, y0, z1), V(x0, y0, z0), V(x1, y0, z0), V(x1, y0, z1), V(0, -1, 0))...)
	tris = append(tris, quad(V(x1, y0, z1), V(x1, y1, z1), V(x0, y1, z1), V(x0, y0, z1), V(0, 0, 1))...)
	tris = append(tris, quad(V(x0, y0, z0), V(x0, y1, z0), V(x1, y1, z0), V(x1, y0, z0), V(0, 0, -1))...)
	return tris
}

func SphereMesh(center Vec3, radius float64, slices, stacks int) []Triangle {
	point := func(stack, slice int) (Vec3, Vec3) {
		phi := -math.Pi/2 + math.Pi*float64(stack)/float64(stacks)
		theta := 2 * math.Pi * float64(slice) / float64(slices)
		cosPhi := math.Cos(phi)
		normal := V(cosPhi*math.Cos(theta), math.Sin(phi), cosPhi*math.Sin(theta))
		return center.Add(normal.Mul(radius)), normal
	}

	var tris []Triangle
	for stack := 0; stack < stacks; stack++ {
		for slice := 0; slice < slices; slice++ {
			p00, n00 := point(stack, slice)
			p10, n10 := point(stack+1, slice)
			p11, n11 := point(stack+1, slice+1)
			p01, n01 := point(stack, slice+1)
			tris = append(tris, NewTriangle(p00, p10, p11, n00, n10, n11))
			tris = append(tris, NewTriangle(p00, p11, p01, n00, n11, n01))
		}
	}
	return tris
}

func mergeTriangles(groups ...[]Triangle) []Triangle {
	var total int
	for _, group := range groups {
		total += len(group)
	}
	out := make([]Triangle, 0, total)
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}
