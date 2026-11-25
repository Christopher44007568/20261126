package main

import (
	"math"
	"sort"
)

type BLASNode struct {
	Bounds          AABB
	Left, Right     int
	First, TriCount int
}

// BLAS is immutable after NewBLAS returns. Character instances share these
// meshes across all four frames.
type BLAS struct {
	Name   string
	Tris   []Triangle
	Order  []int
	Nodes  []BLASNode
	Root   int
	Bounds AABB
}

type triBuild struct {
	index    int
	bounds   AABB
	centroid Vec3
}

func NewBLAS(name string, tris []Triangle) *BLAS {
	b := &BLAS{Name: name, Tris: tris, Bounds: EmptyAABB()}
	items := make([]triBuild, len(tris))
	order := make([]int, len(tris))
	for i, tri := range tris {
		bounds := EmptyAABB()
		bounds.Add(tri.P[0])
		bounds.Add(tri.P[1])
		bounds.Add(tri.P[2])
		b.Bounds = b.Bounds.Union(bounds)
		items[i] = triBuild{index: i, bounds: bounds, centroid: bounds.Centroid()}
		order[i] = i
	}

	var build func(start, end int) int
	build = func(start, end int) int {
		bounds := EmptyAABB()
		for i := start; i < end; i++ {
			bounds = bounds.Union(items[order[i]].bounds)
		}
		nodeID := len(b.Nodes)
		b.Nodes = append(b.Nodes, BLASNode{
			Bounds:   bounds,
			First:    start,
			TriCount: end - start,
		})
		if end-start <= 4 {
			return nodeID
		}

		centerBounds := EmptyAABB()
		for i := start; i < end; i++ {
			centerBounds.Add(items[order[i]].centroid)
		}
		extent := V(
			centerBounds.Max.X-centerBounds.Min.X,
			centerBounds.Max.Y-centerBounds.Min.Y,
			centerBounds.Max.Z-centerBounds.Min.Z,
		)
		axis := 0
		if extent.Y > extent.X && extent.Y >= extent.Z {
			axis = 1
		} else if extent.Z > extent.X && extent.Z > extent.Y {
			axis = 2
		}

		sort.SliceStable(order[start:end], func(i, j int) bool {
			ci := items[order[start+i]].centroid
			cj := items[order[start+j]].centroid
			switch axis {
			case 1:
				return ci.Y < cj.Y
			case 2:
				return ci.Z < cj.Z
			default:
				return ci.X < cj.X
			}
		})

		mid := (start + end) / 2
		left := build(start, mid)
		right := build(mid, end)
		b.Nodes[nodeID].Left = left
		b.Nodes[nodeID].Right = right
		b.Nodes[nodeID].TriCount = 0
		return nodeID
	}

	b.Root = build(0, len(tris))
	b.Order = order
	return b
}

func (b *BLAS) Intersect(r Ray, maxT float64) (t float64, triIndex int, u, v float64, ok bool) {
	bestT := maxT
	stack := make([]int, 0, 32)
	stack = append(stack, b.Root)

	for len(stack) > 0 {
		nodeID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := &b.Nodes[nodeID]
		if _, hit := node.Bounds.Intersect(r, bestT); !hit {
			continue
		}

		if node.TriCount > 0 {
			for at := node.First; at < node.First+node.TriCount; at++ {
				idx := b.Order[at]
				tri := &b.Tris[idx]
				hitT, hitU, hitV, hit := intersectTriangle(r, tri)
				if hit && hitT > 1e-4 && hitT < bestT {
					bestT = hitT
					triIndex = idx
					u, v = hitU, hitV
					ok = true
				}
			}
			continue
		}

		stack = append(stack, node.Left, node.Right)
	}
	return bestT, triIndex, u, v, ok
}

func intersectTriangle(r Ray, tri *Triangle) (t, u, v float64, ok bool) {
	a, b, c := tri.P[0], tri.P[1], tri.P[2]
	e1 := b.Sub(a)
	e2 := c.Sub(a)
	pv := r.Dir.Cross(e2)
	det := e1.Dot(pv)
	if math.Abs(det) < 1e-10 {
		return 0, 0, 0, false
	}
	invDet := 1 / det
	tv := r.Origin.Sub(a)
	u = tv.Dot(pv) * invDet
	if u < 0 || u > 1 {
		return 0, 0, 0, false
	}
	qv := tv.Cross(e1)
	v = r.Dir.Dot(qv) * invDet
	if v < 0 || u+v > 1 {
		return 0, 0, 0, false
	}
	t = e2.Dot(qv) * invDet
	return t, u, v, t > 1e-5
}

func triangleNormal(tri *Triangle, u, v float64) Vec3 {
	w := 1 - u - v
	n := tri.N[0].Mul(w).Add(tri.N[1].Mul(u)).Add(tri.N[2].Mul(v))
	if n.Len() < 1e-8 {
		return tri.P[1].Sub(tri.P[0]).Cross(tri.P[2].Sub(tri.P[0])).Norm()
	}
	return n.Norm()
}
