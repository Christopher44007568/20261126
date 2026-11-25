package main

import (
	"math"
	"math/rand"
)

// Triangle carries everything shading needs so traversal never touches
// auxiliary arrays. ID is globally unique and used for flicker analysis.
type Triangle struct {
	V0, V1, V2 Vec3
	N, C       Vec3
	ID         int32
}

// Intersect is a deterministic Moller-Trumbore test.
func (t *Triangle) Intersect(r Ray) (float32, bool) {
	e1 := vsub(t.V1, t.V0)
	e2 := vsub(t.V2, t.V0)
	p := vcross(r.Dir, e2)
	det := vdot(e1, p)
	if det > -1e-8 && det < 1e-8 {
		return 0, false
	}
	inv := 1 / det
	tv := vsub(r.Orig, t.V0)
	u := vdot(tv, p) * inv
	if u < 0 || u > 1 {
		return 0, false
	}
	q := vcross(tv, e1)
	v := vdot(r.Dir, q) * inv
	if v < 0 || u+v > 1 {
		return 0, false
	}
	tt := vdot(e2, q) * inv
	if tt < 1e-5 {
		return 0, false
	}
	return tt, true
}

func makeTri(a, b, c Vec3, col Vec3, id int32) Triangle {
	n := vnorm(vcross(vsub(b, a), vsub(c, a)))
	return Triangle{V0: a, V1: b, V2: c, N: n, C: col, ID: id}
}

// groundHeight is an analytic height field so trees can be planted exactly
// on the terrain (no floating/sunk trunks).
func groundHeight(x, z float32) float32 {
	return 0.5*float32(math.Sin(float64(x*0.12)))*float32(math.Cos(float64(z*0.10))) +
		0.15*float32(math.Sin(float64(x*0.31+z*0.17)))
}

func makeGround(idBase int32) []Triangle {
	const n = 24
	const ext = float32(90)
	tris := make([]Triangle, 0, n*n*2)
	id := idBase
	c0 := Vec3{0.30, 0.34, 0.18}
	c1 := Vec3{0.26, 0.30, 0.16}
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			x0 := -ext/2 + ext*float32(i)/n
			x1 := -ext/2 + ext*float32(i+1)/n
			z0 := -ext/2 + ext*float32(j)/n
			z1 := -ext/2 + ext*float32(j+1)/n
			v00 := Vec3{x0, groundHeight(x0, z0), z0}
			v10 := Vec3{x1, groundHeight(x1, z0), z0}
			v01 := Vec3{x0, groundHeight(x0, z1), z1}
			v11 := Vec3{x1, groundHeight(x1, z1), z1}
			col := c0
			if (i+j)&1 == 1 {
				col = c1
			}
			tris = append(tris, makeTri(v00, v10, v11, col, id))
			id++
			tris = append(tris, makeTri(v00, v11, v01, col, id))
			id++
		}
	}
	return tris
}

// makeTree builds a pine: tapered trunk plus three foliage cones.
// Every tree becomes one independent BLAS.
func makeTree(rng *rand.Rand, x, z, scale float32, idBase int32) []Triangle {
	tris := make([]Triangle, 0, 64)
	id := idBase
	add := func(a, b, c Vec3, col Vec3) {
		tris = append(tris, makeTri(a, b, c, col, id))
		id++
	}
	y0 := groundHeight(x, z)

	const sides = 6
	trunkH := 1.0 * scale
	rB, rT := 0.14*scale, 0.09*scale
	trunkCol := Vec3{0.32, 0.21, 0.11}
	ring := func(y, r float32) []Vec3 {
		pts := make([]Vec3, sides)
		for i := 0; i < sides; i++ {
			a := float64(i) / sides * 2 * math.Pi
			pts[i] = Vec3{x + r*float32(math.Cos(a)), y, z + r*float32(math.Sin(a))}
		}
		return pts
	}
	bot := ring(y0, rB)
	top := ring(y0+trunkH, rT)
	for i := 0; i < sides; i++ {
		j := (i + 1) % sides
		add(bot[i], bot[j], top[j], trunkCol)
		add(bot[i], top[j], top[i], trunkCol)
	}

	jit := func(d float32) float32 { return d * (rng.Float32()*2 - 1) }
	fcol := Vec3{0.14 + jit(0.03), 0.34 + jit(0.06), 0.10 + jit(0.03)}
	const csides = 8
	for lvl := 0; lvl < 3; lvl++ {
		baseY := y0 + (0.45+0.55*float32(lvl))*scale
		baseR := (1.05 - 0.28*float32(lvl)) * scale
		apex := Vec3{x, baseY + 1.05*scale, z}
		cen := Vec3{x, baseY, z}
		pts := make([]Vec3, csides)
		for i := 0; i < csides; i++ {
			a := float64(i)/csides*2*math.Pi + float64(lvl)*0.3
			pts[i] = Vec3{x + baseR*float32(math.Cos(a)), baseY, z + baseR*float32(math.Sin(a))}
		}
		for i := 0; i < csides; i++ {
			j := (i + 1) % csides
			add(pts[i], pts[j], apex, fcol)
			add(cen, pts[j], pts[i], fcol)
		}
	}
	return tris
}

// buildForest returns one triangle slice per BLAS: ground first, then trees.
func buildForest(numTrees int, seed int64) [][]Triangle {
	rng := rand.New(rand.NewSource(seed))
	out := make([][]Triangle, 0, numTrees+1)
	out = append(out, makeGround(0))
	for i := 0; i < numTrees; i++ {
		ang := rng.Float32() * 2 * math.Pi
		rad := 8 + rng.Float32()*34
		x := rad * float32(math.Cos(float64(ang)))
		z := rad * float32(math.Sin(float64(ang)))
		scale := 0.7 + rng.Float32()*0.9
		out = append(out, makeTree(rng, x, z, scale, int32((i+1)*100000)))
	}
	return out
}
