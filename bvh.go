package main

import (
	"math"
	"sort"
)

const leafTriangleLimit = 4

type BVHNode struct {
	Bounds    AABB
	Left      int
	Right     int
	Triangles []int
}

type BVH struct {
	Name      string
	Nodes     []BVHNode
	Triangles []Triangle
}

type SimplifiedAccel struct {
	BVH *BVH
}

func NewSimplifiedAccel(triangles []Triangle) *SimplifiedAccel {
	return &SimplifiedAccel{BVH: NewBVH("conservative-median-bvh", triangles)}
}

func NewBVH(name string, triangles []Triangle) *BVH {
	b := &BVH{Name: name, Triangles: append([]Triangle(nil), triangles...)}
	if len(b.Triangles) == 0 {
		b.Nodes = []BVHNode{{Bounds: EmptyAABB(), Left: -1, Right: -1}}
		return b
	}

	ids := make([]int, len(b.Triangles))
	for i := range ids {
		ids[i] = i
	}
	b.Nodes = make([]BVHNode, 0, 2*len(ids))
	b.build(ids)
	return b
}

func (b *BVH) build(ids []int) int {
	bounds := EmptyAABB()
	centroidBounds := EmptyAABB()
	for _, id := range ids {
		triangleBounds := b.Triangles[id].Bounds()
		centroid := b.Triangles[id].Centroid()
		bounds = bounds.IncludeBox(triangleBounds)
		centroidBounds = centroidBounds.IncludePoint(centroid)
	}

	nodeIndex := len(b.Nodes)
	b.Nodes = append(b.Nodes, BVHNode{Bounds: bounds, Left: -1, Right: -1})

	if len(ids) <= leafTriangleLimit {
		b.Nodes[nodeIndex].Triangles = append([]int(nil), ids...)
		return nodeIndex
	}

	axis := centroidBounds.LargestAxis()
	sort.SliceStable(ids, func(i, j int) bool {
		left := b.Triangles[ids[i]].Centroid().Axis(axis)
		right := b.Triangles[ids[j]].Centroid().Axis(axis)
		if left != right {
			return left < right
		}
		return ids[i] < ids[j]
	})

	split := len(ids) / 2
	left := b.build(ids[:split])
	right := b.build(ids[split:])
	b.Nodes[nodeIndex].Left = left
	b.Nodes[nodeIndex].Right = right
	return nodeIndex
}

func (v Vec3) Axis(axis int) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}

func InfiniteRay(origin, direction Vec3) Ray {
	ray := MakeRay(origin, direction)
	ray.MaxT = math.Inf(1)
	return ray
}

func (b *BVH) Trace(ray Ray) TraceInfo {
	info := TraceInfo{}
	if len(b.Nodes) == 0 || !b.Nodes[0].Bounds.Valid() {
		return info
	}

	stack := []int{0}
	for len(stack) > 0 {
		nodeIndex := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := &b.Nodes[nodeIndex]

		if !node.Bounds.IntersectsRay(ray) {
			continue
		}

		if len(node.Triangles) > 0 {
			for _, id := range node.Triangles {
				info.Tests++
				if hit, ok := b.Triangles[id].Intersect(ray); ok {
					info.Hits = append(info.Hits, hit)
				}
			}
			continue
		}

		if node.Left >= 0 {
			stack = append(stack, node.Left)
		}
		if node.Right >= 0 {
			stack = append(stack, node.Right)
		}
	}

	sortedHits(info.Hits)
	if hit, ok := nearestHit(info.Hits); ok {
		info.Hit = hit
		info.OK = true
	}
	return info
}

func (a *SimplifiedAccel) Trace(ray Ray) TraceInfo {
	return a.BVH.Trace(ray)
}
