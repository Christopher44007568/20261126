package main

import "sort"

const LeafSize = 4

// Node is the uncompressed layout: 24B bounds + 16B references = 40 bytes.
// Internal: Count == 0, Left/Right are node indices.
// Leaf: Count > 0, First/Count reference the Indices array.
type Node struct {
	Min, Max     Vec3
	Left, Right  int32
	First, Count int32
}

const NodeBytes = 40

type BVH struct {
	Nodes   []Node
	Indices []int32
	Tris    []Triangle
}

func triBounds(t *Triangle) (Vec3, Vec3) {
	mn := vmin(t.V0, vmin(t.V1, t.V2))
	mx := vmax(t.V0, vmax(t.V1, t.V2))
	return mn, mx
}

func centroid(t *Triangle) Vec3 {
	return vscale(vadd(t.V0, vadd(t.V1, t.V2)), 1.0/3.0)
}

// BuildBVH is a deterministic median-split builder: ties in the sort are
// broken by triangle ID so the same input always yields the same tree.
func BuildBVH(tris []Triangle) *BVH {
	b := &BVH{Tris: tris}
	n := len(tris)
	b.Indices = make([]int32, n)
	cents := make([]Vec3, n)
	for i := range tris {
		b.Indices[i] = int32(i)
		cents[i] = centroid(&tris[i])
	}
	axisVal := func(c Vec3, a int) float32 {
		switch a {
		case 0:
			return c.X
		case 1:
			return c.Y
		}
		return c.Z
	}
	var build func(start, end int32) int32
	build = func(start, end int32) int32 {
		idx := int32(len(b.Nodes))
		b.Nodes = append(b.Nodes, Node{})
		mn, mx := triBounds(&tris[b.Indices[start]])
		cmn, cmx := cents[b.Indices[start]], cents[b.Indices[start]]
		for k := start + 1; k < end; k++ {
			t := &tris[b.Indices[k]]
			tmn, tmx := triBounds(t)
			mn, mx = vmin(mn, tmn), vmax(mx, tmx)
			c := cents[b.Indices[k]]
			cmn, cmx = vmin(cmn, c), vmax(cmx, c)
		}
		count := end - start
		ext := vsub(cmx, cmn)
		axis := 0
		if ext.Y > ext.X && ext.Y >= ext.Z {
			axis = 1
		} else if ext.Z > ext.X && ext.Z >= ext.Y {
			axis = 2
		}
		if count <= LeafSize || axisVal(cmx, axis) == axisVal(cmn, axis) {
			b.Nodes[idx] = Node{Min: mn, Max: mx, Left: -1, Right: -1, First: start, Count: count}
			return idx
		}
		sub := b.Indices[start:end]
		sort.Slice(sub, func(i, j int) bool {
			a, c := sub[i], sub[j]
			va, vc := axisVal(cents[a], axis), axisVal(cents[c], axis)
			if va != vc {
				return va < vc
			}
			return tris[a].ID < tris[c].ID
		})
		mid := start + count/2
		l := build(start, mid)
		r := build(mid, end)
		b.Nodes[idx] = Node{Min: mn, Max: mx, Left: l, Right: r, Count: 0}
		return idx
	}
	build(0, int32(n))
	return b
}

// BuildTLAS builds a top-level BVH over per-BLAS world-space bounds.
// Leaves reference exactly one BLAS (First = BLAS index, Count = 1).
func BuildTLAS(boxes [][2]Vec3) []Node {
	n := len(boxes)
	idx := make([]int32, n)
	cents := make([]Vec3, n)
	for i := range boxes {
		idx[i] = int32(i)
		cents[i] = vscale(vadd(boxes[i][0], boxes[i][1]), 0.5)
	}
	nodes := make([]Node, 0, 2*n)
	var build func(start, end int32) int32
	build = func(start, end int32) int32 {
		ni := int32(len(nodes))
		nodes = append(nodes, Node{})
		mn, mx := boxes[idx[start]][0], boxes[idx[start]][1]
		cmn, cmx := cents[idx[start]], cents[idx[start]]
		for k := start + 1; k < end; k++ {
			mn, mx = vmin(mn, boxes[idx[k]][0]), vmax(mx, boxes[idx[k]][1])
			cmn, cmx = vmin(cmn, cents[idx[k]]), vmax(cmx, cents[idx[k]])
		}
		count := end - start
		if count == 1 {
			nodes[ni] = Node{Min: mn, Max: mx, Left: -1, Right: -1, First: idx[start], Count: 1}
			return ni
		}
		ext := vsub(cmx, cmn)
		axis := 0
		if ext.Y > ext.X && ext.Y >= ext.Z {
			axis = 1
		} else if ext.Z > ext.X && ext.Z >= ext.Y {
			axis = 2
		}
		sub := idx[start:end]
		sort.Slice(sub, func(i, j int) bool {
			var va, vc float32
			switch axis {
			case 0:
				va, vc = cents[sub[i]].X, cents[sub[j]].X
			case 1:
				va, vc = cents[sub[i]].Y, cents[sub[j]].Y
			default:
				va, vc = cents[sub[i]].Z, cents[sub[j]].Z
			}
			if va != vc {
				return va < vc
			}
			return sub[i] < sub[j]
		})
		mid := start + count/2
		l := build(start, mid)
		r := build(mid, end)
		nodes[ni] = Node{Min: mn, Max: mx, Left: l, Right: r, Count: 0}
		return ni
	}
	build(0, int32(n))
	return nodes
}
