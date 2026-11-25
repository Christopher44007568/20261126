package main

import "sort"

const leafTriangles = 4

type PointerNode struct {
	Bounds       AABB
	Left         *PointerNode
	Right        *PointerNode
	First        int
	Count        int
	SerializedID uint32
}

type PointerBVH struct {
	Root  *PointerNode
	Order []int
}

func BuildPointerBVH(triangles []Triangle) *PointerBVH {
	order := make([]int, len(triangles))
	for i := range order {
		order[i] = i
	}
	if len(order) == 0 {
		return &PointerBVH{}
	}
	root := buildPointerNode(triangles, order, 0, len(order))
	return &PointerBVH{Root: root, Order: order}
}

func buildPointerNode(triangles []Triangle, order []int, lo, hi int) *PointerNode {
	node := &PointerNode{First: lo, Count: hi - lo}
	bounds := EmptyAABB()
	centroidBounds := EmptyAABB()
	for _, idx := range order[lo:hi] {
		tb := triangles[idx].Bounds()
		bounds = bounds.AddBox(tb)
		c := triangles[idx].V[0].Add(triangles[idx].V[1]).Add(triangles[idx].V[2]).Mul(1.0 / 3.0)
		centroidBounds = centroidBounds.AddPoint(c)
	}
	node.Bounds = bounds

	count := hi - lo
	if count <= leafTriangles {
		return node
	}

	extent := centroidBounds.Max.Sub(centroidBounds.Min)
	axis := 0
	if extent.Y > extent.X && extent.Y >= extent.Z {
		axis = 1
	} else if extent.Z > extent.X && extent.Z > extent.Y {
		axis = 2
	}

	sort.SliceStable(order[lo:hi], func(i, j int) bool {
		ci := triangleCentroid(triangles[order[lo+i]])
		cj := triangleCentroid(triangles[order[lo+j]])
		return rayAxis(ci, axis) < rayAxis(cj, axis)
	})

	mid := lo + count/2
	node.Left = buildPointerNode(triangles, order, lo, mid)
	node.Right = buildPointerNode(triangles, order, mid, hi)
	node.Count = 0
	return node
}

func triangleCentroid(t Triangle) Vec3 {
	return t.V[0].Add(t.V[1]).Add(t.V[2]).Mul(1.0 / 3.0)
}

func IntersectPointerBVH(bvh *PointerBVH, triangles []Triangle, ray Ray) (TriHit, bool) {
	if bvh == nil || bvh.Root == nil {
		return TriHit{}, false
	}

	var best TriHit
	found := false
	bestT := ray.TMax
	stack := []*PointerNode{bvh.Root}

	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if node == nil {
			continue
		}
		if _, _, ok := node.Bounds.Intersect(ray); !ok {
			continue
		}

		if node.Left == nil && node.Right == nil {
			for k := 0; k < node.Count; k++ {
				orderPos := node.First + k
				triIndex := bvh.Order[orderPos]
				queryRay := ray
				queryRay.TMax = bestT
				hit, ok := IntersectTriangle(queryRay, triangles[triIndex], triIndex)
				if ok && hit.T < bestT {
					best = hit
					bestT = hit.T
					found = true
				}
			}
			continue
		}

		if node.Left != nil {
			stack = append(stack, node.Left)
		}
		if node.Right != nil {
			stack = append(stack, node.Right)
		}
	}

	if found {
		best.T = bestT
	}
	return best, found
}

func CollectPointerNodes(root *PointerNode) []*PointerNode {
	var nodes []*PointerNode
	var walk func(*PointerNode)
	walk = func(node *PointerNode) {
		if node == nil {
			return
		}
		node.SerializedID = uint32(len(nodes))
		nodes = append(nodes, node)
		walk(node.Left)
		walk(node.Right)
	}
	walk(root)
	return nodes
}
