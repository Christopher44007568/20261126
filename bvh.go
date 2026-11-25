package main

import "sort"

type BVHNode struct {
	Bounds       AABB
	BaseBounds   AABB
	BaseCentroid Vec2
	Left         *BVHNode
	Right        *BVHNode
	Primitives   []int
}

func (n *BVHNode) IsLeaf() bool {
	return n.Left == nil && n.Right == nil
}

type BVH struct {
	Root     *BVHNode
	LeafSize int
}

func BuildBVH(mesh Mesh, leafSize int) *BVH {
	if leafSize < 1 {
		leafSize = 1
	}
	indices := make([]int, len(mesh.Triangles))
	for i := range indices {
		indices[i] = i
	}
	return &BVH{Root: buildNode(mesh, indices, leafSize), LeafSize: leafSize}
}

func buildNode(mesh Mesh, indices []int, leafSize int) *BVHNode {
	bounds := EmptyAABB()
	centroidSum := Vec2{}
	for _, i := range indices {
		bounds = bounds.Union(mesh.TriangleAABB(i))
		centroidSum = centroidSum.Add(mesh.Triangles[i].Centroid())
	}
	count := float64(len(indices))
	centroid := centroidSum.Scale(1 / count)
	node := &BVHNode{
		Bounds:       bounds,
		BaseBounds:   bounds,
		BaseCentroid: centroid,
	}
	if len(indices) <= leafSize {
		node.Primitives = append([]int(nil), indices...)
		return node
	}

	size := bounds.Size()
	axis := 0
	if size.Y > size.X {
		axis = 1
	}
	sort.Slice(indices, func(i, j int) bool {
		a := mesh.Triangles[indices[i]].Centroid()
		b := mesh.Triangles[indices[j]].Centroid()
		if axis == 0 {
			return a.X < b.X
		}
		return a.Y < b.Y
	})
	mid := len(indices) / 2
	node.Left = buildNode(mesh, indices[:mid], leafSize)
	node.Right = buildNode(mesh, indices[mid:], leafSize)
	node.Bounds = node.Left.Bounds.Union(node.Right.Bounds)
	node.BaseBounds = node.Bounds
	return node
}

// RefitFast keeps each leaf's bind-pose box and translates it by the leaf
// centroid delta. It is cheap, but rotation or large soft-body deformation
// can leave geometry outside a translated leaf box.
func (b *BVH) RefitFast(mesh Mesh) {
	refitNodeFast(b.Root, mesh)
}

func refitNodeFast(node *BVHNode, mesh Mesh) {
	if node == nil {
		return
	}
	if node.IsLeaf() {
		if len(node.Primitives) == 0 {
			node.Bounds = EmptyAABB()
			return
		}
		centroid := Vec2{}
		for _, i := range node.Primitives {
			centroid = centroid.Add(mesh.Triangles[i].Centroid())
		}
		centroid = centroid.Scale(1 / float64(len(node.Primitives)))
		delta := centroid.Sub(node.BaseCentroid)
		node.Bounds = node.BaseBounds.Translated(delta)
		return
	}
	refitNodeFast(node.Left, mesh)
	refitNodeFast(node.Right, mesh)
	node.Bounds = node.Left.Bounds.Union(node.Right.Bounds)
}

type BoundsMetrics struct {
	MaxLeafEscape float64
	RootEscape    float64
	MaxAreaGrowth float64
	MeanLeafArea  float64
}

type BoundsBudget struct {
	MaxLeafEscape float64
	MaxRootEscape float64
	MaxAreaGrowth float64
}

func DefaultBoundsBudget() BoundsBudget {
	return BoundsBudget{
		MaxLeafEscape: 0.05,
		MaxRootEscape: 0.02,
		MaxAreaGrowth: 2.0,
	}
}

func (b BoundsBudget) Exceeded(m BoundsMetrics) bool {
	return m.MaxLeafEscape > b.MaxLeafEscape ||
		m.RootEscape > b.MaxRootEscape ||
		m.MaxAreaGrowth > b.MaxAreaGrowth
}

func (b *BVH) AssessBounds(mesh Mesh) BoundsMetrics {
	metrics := BoundsMetrics{}
	leafCount, maxEscape, maxGrowth, totalArea := assessNode(b.Root, mesh)
	metrics.MaxLeafEscape = maxEscape
	metrics.MaxAreaGrowth = maxGrowth
	if leafCount > 0 {
		metrics.MeanLeafArea = totalArea / float64(leafCount)
	}
	if b.Root != nil {
		metrics.RootEscape = uncoveredFraction(mesh.Bounds(), b.Root.Bounds)
	}
	return metrics
}

func assessNode(node *BVHNode, mesh Mesh) (leafCount int, maxEscape, maxGrowth, totalArea float64) {
	if node == nil {
		return 0, 0, 0, 0
	}
	if !node.IsLeaf() {
		leftCount, leftEscape, leftGrowth, leftArea := assessNode(node.Left, mesh)
		rightCount, rightEscape, rightGrowth, rightArea := assessNode(node.Right, mesh)
		return leftCount + rightCount,
			max64(leftEscape, rightEscape),
			max64(leftGrowth, rightGrowth),
			leftArea + rightArea
	}
	exact := EmptyAABB()
	for _, i := range node.Primitives {
		exact = exact.Union(mesh.TriangleAABB(i))
	}
	if exact.IsEmpty() {
		return 0, 0, 0, 0
	}
	escape := uncoveredFraction(exact, node.Bounds)
	area := exact.Area()
	growth := 0.0
	if node.Bounds.Area() > 1e-12 {
		growth = area/node.Bounds.Area() - 1
	} else if area > 1e-12 {
		growth = 1e12
	}
	return 1, escape, growth, area
}

func uncoveredFraction(expected, actual AABB) float64 {
	if expected.IsEmpty() {
		return 0
	}
	expectedArea := expected.Area()
	if expectedArea <= 1e-12 {
		if actual.ContainsPoint(expected.Min) && actual.ContainsPoint(expected.Max) {
			return 0
		}
		return 1
	}
	covered := expected.Intersection(actual).Area()
	return mathMax(0, expectedArea-covered) / expectedArea
}

func (b *BVH) IntersectPoint(mesh Mesh, point Vec2) (int, bool) {
	if b.Root == nil || !b.Root.Bounds.ContainsPoint(point) {
		return -1, false
	}
	stack := []*BVHNode{b.Root}
	best := -1
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !n.Bounds.ContainsPoint(point) {
			continue
		}
		if n.IsLeaf() {
			for _, i := range n.Primitives {
				if i > best && mesh.Triangles[i].Contains(point) {
					best = i
				}
			}
			continue
		}
		stack = append(stack, n.Left, n.Right)
	}
	return best, best >= 0
}

func BruteIntersect(mesh Mesh, point Vec2) (int, bool) {
	best := -1
	for i := range mesh.Triangles {
		if i > best && mesh.Triangles[i].Contains(point) {
			best = i
		}
	}
	return best, best >= 0
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
