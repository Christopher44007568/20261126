package main

import "sort"

const (
	DefaultLeafSize = 4
	FullNodeBytes   = 40 // 6 float32 bounds + 4 int32 references
	IndexBytes      = 4
	TriangleBytes   = 40 // 9 float32 positions + 4 byte material/flags padding
)

type Node struct {
	Min, Max     Vec3
	Left, Right  int
	First, Count int
}

func (n Node) IsLeaf() bool { return n.Count > 0 }

type BVH struct {
	Nodes   []Node
	Indices []int
	Tris    []Triangle
}

func BuildBVH(tris []Triangle) *BVH {
	bvh := &BVH{Tris: append([]Triangle(nil), tris...)}
	n := len(bvh.Tris)
	bvh.Indices = make([]int, n)
	centers := make([]Vec3, n)
	for i := range bvh.Tris {
		bvh.Indices[i] = i
		centers[i] = triCentroid(bvh.Tris[i])
	}

	var build func(start, end int) int
	build = func(start, end int) int {
		nodeIndex := len(bvh.Nodes)
		bvh.Nodes = append(bvh.Nodes, Node{})

		firstTri := bvh.Tris[bvh.Indices[start]]
		bounds := triangleAABB(firstTri)
		centerBounds := AABB{Min: centers[bvh.Indices[start]], Max: centers[bvh.Indices[start]]}
		for k := start + 1; k < end; k++ {
			idx := bvh.Indices[k]
			bounds = boundsUnion(bounds, triangleAABB(bvh.Tris[idx]))
			centerBounds.Min = vMin(centerBounds.Min, centers[idx])
			centerBounds.Max = vMax(centerBounds.Max, centers[idx])
		}

		count := end - start
		if count <= DefaultLeafSize {
			bvh.Nodes[nodeIndex] = Node{
				Min: bounds.Min, Max: bounds.Max,
				Left: -1, Right: -1, First: start, Count: count,
			}
			return nodeIndex
		}

		extent := vSub(centerBounds.Max, centerBounds.Min)
		axis := 0
		if extent.Y > extent.X && extent.Y >= extent.Z {
			axis = 1
		} else if extent.Z > extent.X && extent.Z >= extent.Y {
			axis = 2
		}
		if axisExtent(extent, axis) < 1e-9 {
			bvh.Nodes[nodeIndex] = Node{
				Min: bounds.Min, Max: bounds.Max,
				Left: -1, Right: -1, First: start, Count: count,
			}
			return nodeIndex
		}

		sub := bvh.Indices[start:end]
		sort.Slice(sub, func(i, j int) bool {
			ai, aj := sub[i], sub[j]
			vi := axisValue(centers[ai], axis)
			vj := axisValue(centers[aj], axis)
			if vi != vj {
				return vi < vj
			}
			return bvh.Tris[ai].ID < bvh.Tris[aj].ID
		})

		mid := start + count/2
		left := build(start, mid)
		right := build(mid, end)
		bvh.Nodes[nodeIndex] = Node{
			Min: bounds.Min, Max: bounds.Max,
			Left: left, Right: right, First: -1, Count: 0,
		}
		return nodeIndex
	}

	if n > 0 {
		build(0, n)
	}
	return bvh
}

func triCentroid(t Triangle) Vec3 {
	return vScale(vAdd(vAdd(t.A, t.B), t.C), 1.0/3.0)
}

func axisValue(v Vec3, axis int) float64 {
	switch axis {
	case 1:
		return v.Y
	case 2:
		return v.Z
	default:
		return v.X
	}
}

func axisExtent(v Vec3, axis int) float64 { return axisValue(v, axis) }

func (b *BVH) Intersect(ray Ray) (Hit, bool) {
	if len(b.Nodes) == 0 {
		return Hit{}, false
	}
	stack := make([]int, 0, 32)
	stack = append(stack, 0)
	var best Hit
	found := false

	for len(stack) > 0 {
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := b.Nodes[idx]
		box := AABB{Min: node.Min, Max: node.Max}
		if !box.Hit(ray.Origin, ray.Dir) {
			continue
		}
		if node.IsLeaf() {
			for k := node.First; k < node.First+node.Count; k++ {
				triIndex := b.Indices[k]
				hit, ok := b.Tris[triIndex].Intersect(ray)
				if ok && (!found || hit.T < best.T) {
					best, found = hit, true
				}
			}
			continue
		}
		stack = append(stack, node.Left, node.Right)
	}
	return best, found
}

func BruteIntersect(tris []Triangle, ray Ray) (Hit, bool) {
	var best Hit
	found := false
	for i := range tris {
		hit, ok := tris[i].Intersect(ray)
		if ok && (!found || hit.T < best.T) {
			best, found = hit, true
		}
	}
	return best, found
}

func (b *BVH) ASBytes() int {
	return len(b.Nodes)*FullNodeBytes + len(b.Indices)*IndexBytes
}
