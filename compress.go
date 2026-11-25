package main

import "math"

// CNode is the compressed layout: 12B quantized bounds + 4B packed meta.
//
// Meta bit 31 set   -> leaf:   bits 16..30 = count, bits 0..15 = first
//                              (triangles physically reordered, no index array)
// Meta bit 31 clear -> internal: bits 0..30 = left child, right = left+1
//                              (children allocated consecutively)
type CNode struct {
	QMin, QMax [3]uint16
	Meta       uint32
}

const CNodeBytes = 16
const leafFlag = uint32(1) << 31

type CBVH struct {
	Nodes          []CNode
	Tris           []Triangle // reordered so every leaf owns a contiguous run
	GridMin, GridMax Vec3     // quantization grid (root bounds, padded)
}

func (c *CBVH) dequant(q uint16, axis int) float32 {
	var g0, g1 float32
	switch axis {
	case 0:
		g0, g1 = c.GridMin.X, c.GridMax.X
	case 1:
		g0, g1 = c.GridMin.Y, c.GridMax.Y
	default:
		g0, g1 = c.GridMin.Z, c.GridMax.Z
	}
	return g0 + float32(q)/65535*(g1-g0)
}

func (c *CBVH) nodeBounds(n *CNode) (Vec3, Vec3) {
	mn := Vec3{c.dequant(n.QMin[0], 0), c.dequant(n.QMin[1], 1), c.dequant(n.QMin[2], 2)}
	mx := Vec3{c.dequant(n.QMax[0], 0), c.dequant(n.QMax[1], 1), c.dequant(n.QMax[2], 2)}
	return mn, mx
}

// Compress performs three steps:
//  1. node compression: AABBs quantized to 16-bit inside the BLAS grid,
//     conservatively (min floored, max ceiled) so traversal visits a
//     superset of the original nodes and results stay bit-identical;
//  2. leaf reordering: triangles physically reordered into leaf order,
//     eliminating the per-triangle index array entirely;
//  3. address remapping: nodes re-emitted so each internal node's children
//     occupy consecutive slots (right = left+1), packing both child
//     pointers into one 31-bit field.
func Compress(b *BVH) *CBVH {
	c := &CBVH{}
	root := b.Nodes[0]
	ext := vsub(root.Max, root.Min)
	pad := vadd(vscale(ext, 1e-3), Vec3{1e-4, 1e-4, 1e-4})
	c.GridMin = vsub(root.Min, pad)
	c.GridMax = vadd(root.Max, pad)

	// Address remapping: old node index -> new node index.
	oldToNew := make([]int32, len(b.Nodes))
	for i := range oldToNew {
		oldToNew[i] = -1
	}
	var order []int32 // new index -> old index
	alloc := func() int32 {
		order = append(order, -1)
		return int32(len(order)) - 1
	}
	oldToNew[0] = alloc()
	order[0] = 0
	var place func(old int32)
	place = func(old int32) {
		nd := b.Nodes[old]
		if nd.Count == 0 {
			l, r := alloc(), alloc() // consecutive by construction
			oldToNew[nd.Left], order[l] = l, nd.Left
			oldToNew[nd.Right], order[r] = r, nd.Right
			place(nd.Left)
			place(nd.Right)
		}
	}
	place(0)

	qmin := func(p Vec3) [3]uint16 {
		return [3]uint16{quantAxis(p.X, c.GridMin.X, c.GridMax.X, math.Floor),
			quantAxis(p.Y, c.GridMin.Y, c.GridMax.Y, math.Floor),
			quantAxis(p.Z, c.GridMin.Z, c.GridMax.Z, math.Floor)}
	}
	qmax := func(p Vec3) [3]uint16 {
		return [3]uint16{quantAxis(p.X, c.GridMin.X, c.GridMax.X, math.Ceil),
			quantAxis(p.Y, c.GridMin.Y, c.GridMax.Y, math.Ceil),
			quantAxis(p.Z, c.GridMin.Z, c.GridMax.Z, math.Ceil)}
	}

	c.Nodes = make([]CNode, len(order))
	c.Tris = make([]Triangle, 0, len(b.Tris))
	for newIdx, old := range order {
		nd := b.Nodes[old]
		cn := CNode{QMin: qmin(nd.Min), QMax: qmax(nd.Max)}
		if nd.Count == 0 {
			cn.Meta = uint32(oldToNew[nd.Left]) // right child implicit: +1
		} else {
			first := len(c.Tris)
			for k := nd.First; k < nd.First+nd.Count; k++ {
				c.Tris = append(c.Tris, b.Tris[b.Indices[k]])
			}
			cn.Meta = leafFlag | uint32(nd.Count)<<16 | uint32(first)
		}
		c.Nodes[newIdx] = cn
	}
	return c
}

func quantAxis(p, g0, g1 float32, round func(float64) float64) uint16 {
	t := float64(p-g0) / float64(g1-g0) * 65535
	v := round(t)
	if v < 0 {
		v = 0
	}
	if v > 65535 {
		v = 65535
	}
	return uint16(v)
}

// InvalidRef describes one broken reference found by Validate.
type InvalidRef struct {
	Node int
	Kind string
}

// Validate walks the compressed BVH and checks every reference: child
// indices in range, no revisits, quantized child bounds nested inside the
// parent, leaf ranges inside the triangle array, and full coverage of
// nodes and triangles exactly once.
func (c *CBVH) Validate() []InvalidRef {
	var bad []InvalidRef
	n := len(c.Nodes)
	visited := make([]bool, n)
	triUsed := make([]int32, len(c.Tris))
	var rec func(i int32, pmin, pmax [3]uint16, hasParent bool)
	rec = func(i int32, pmin, pmax [3]uint16, hasParent bool) {
		if i < 0 || int(i) >= n {
			bad = append(bad, InvalidRef{int(i), "child-index-out-of-range"})
			return
		}
		if visited[i] {
			bad = append(bad, InvalidRef{int(i), "node-revisited"})
			return
		}
		visited[i] = true
		nd := c.Nodes[i]
		if hasParent {
			for a := 0; a < 3; a++ {
				if nd.QMin[a] < pmin[a] || nd.QMax[a] > pmax[a] {
					bad = append(bad, InvalidRef{int(i), "bounds-escape-parent"})
					break
				}
			}
		}
		if nd.Meta&leafFlag != 0 {
			first := int32(nd.Meta & 0xffff)
			cnt := int32((nd.Meta >> 16) & 0x7fff)
			if cnt == 0 {
				bad = append(bad, InvalidRef{int(i), "empty-leaf"})
			}
			if first < 0 || first+cnt > int32(len(c.Tris)) {
				bad = append(bad, InvalidRef{int(i), "leaf-range-out-of-bounds"})
				return
			}
			for k := first; k < first+cnt; k++ {
				triUsed[k]++
			}
			return
		}
		l := int32(nd.Meta & 0x7fffffff)
		if l == i || l+1 == i {
			bad = append(bad, InvalidRef{int(i), "self-reference"})
		}
		rec(l, nd.QMin, nd.QMax, true)
		rec(l+1, nd.QMin, nd.QMax, true)
	}
	rec(0, [3]uint16{}, [3]uint16{}, false)
	for i, v := range visited {
		if !v {
			bad = append(bad, InvalidRef{i, "unreachable-node"})
		}
	}
	for k, cnt := range triUsed {
		if cnt != 1 {
			bad = append(bad, InvalidRef{k, "triangle-coverage"})
		}
	}
	return bad
}
