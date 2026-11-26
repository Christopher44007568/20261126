package main

import (
	"fmt"
	"math"
)

const (
	QuantizedLevels = 72
	QTriangleBytes  = 12 // 9 position bytes + material/flags in padded word
	QIndexBytes     = 2
	QNodeBytes      = 16 // 6 bound bytes + packed references/count/padding
)

type QTriangle struct {
	V   [3][3]uint8
	Mat uint8
}

type QuantizedMesh struct {
	Name      string
	Tris      []QTriangle
	Min       Vec3
	Max       Vec3
	Levels    int
	Materials []Material
}

func QuantizeMesh(mesh *Mesh, levels int) *QuantizedMesh {
	if levels < 2 || levels > 256 {
		levels = QuantizedLevels
	}
	bounds := mesh.Bounds()
	qm := &QuantizedMesh{
		Name:      mesh.Name + "-quantized",
		Min:       bounds.Min,
		Max:       bounds.Max,
		Levels:    levels,
		Materials: mesh.Materials,
		Tris:      make([]QTriangle, len(mesh.Tris)),
	}
	for i, tri := range mesh.Tris {
		qm.Tris[i] = QTriangle{
			V: [3][3]uint8{
				qm.quantizeVertex(tri.A, math.Round),
				qm.quantizeVertex(tri.B, math.Round),
				qm.quantizeVertex(tri.C, math.Round),
			},
			Mat: uint8(tri.Material),
		}
	}
	return qm
}

func (qm *QuantizedMesh) quantizeVertex(v Vec3, round func(float64) float64) [3]uint8 {
	return [3]uint8{
		qm.quantizeAxis(v.X, 0, round),
		qm.quantizeAxis(v.Y, 1, round),
		qm.quantizeAxis(v.Z, 2, round),
	}
}

func (qm *QuantizedMesh) span(axis int) (float64, float64) {
	switch axis {
	case 0:
		return qm.Min.X, qm.Max.X
	case 1:
		return qm.Min.Y, qm.Max.Y
	default:
		return qm.Min.Z, qm.Max.Z
	}
}

func (qm *QuantizedMesh) quantizeAxis(v float64, axis int, round func(float64) float64) uint8 {
	lo, hi := qm.span(axis)
	if hi-lo < 1e-12 {
		return 0
	}
	scaled := (v - lo) / (hi - lo) * float64(qm.Levels-1)
	code := int(round(scaled))
	if code < 0 {
		code = 0
	}
	if code >= qm.Levels {
		code = qm.Levels - 1
	}
	return uint8(code)
}

func (qm *QuantizedMesh) dequantizeCode(code uint8, axis int) float64 {
	lo, hi := qm.span(axis)
	return lo + float64(code)/float64(qm.Levels-1)*(hi-lo)
}

func (qm *QuantizedMesh) Vertex(v [3]uint8) Vec3 {
	return Vec3{
		X: qm.dequantizeCode(v[0], 0),
		Y: qm.dequantizeCode(v[1], 1),
		Z: qm.dequantizeCode(v[2], 2),
	}
}

func (qm *QuantizedMesh) Triangle(index int) Triangle {
	q := qm.Tris[index]
	return Triangle{
		A:        qm.Vertex(q.V[0]),
		B:        qm.Vertex(q.V[1]),
		C:        qm.Vertex(q.V[2]),
		ID:       index,
		Material: int(q.Mat),
	}
}

func (qm *QuantizedMesh) Dequantize() []Triangle {
	out := make([]Triangle, len(qm.Tris))
	for i := range qm.Tris {
		out[i] = qm.Triangle(i)
	}
	return out
}

func (qm *QuantizedMesh) Bytes() int { return len(qm.Tris) * QTriangleBytes }

type QNode struct {
	QMin  [3]uint8
	QMax  [3]uint8
	First uint16
	Left  int16
	Right int16
	Count uint8
	_     byte
}

type QBVH struct {
	Nodes   []QNode
	Indices []uint16
	Mesh    *QuantizedMesh
}

func BuildQBVH(qm *QuantizedMesh) (*QBVH, error) {
	work := qm.Dequantize()
	full := BuildBVH(work)
	if len(full.Nodes) > math.MaxInt16 {
		return nil, fmt.Errorf("compressed BVH has %d nodes; int16 child references are full", len(full.Nodes))
	}
	if len(full.Indices) > int(^uint16(0)) {
		return nil, fmt.Errorf("compressed BVH has %d indices; uint16 references are full", len(full.Indices))
	}

	q := &QBVH{Mesh: qm, Nodes: make([]QNode, len(full.Nodes)), Indices: make([]uint16, len(full.Indices))}
	for i, idx := range full.Indices {
		q.Indices[i] = uint16(idx)
	}
	for i, node := range full.Nodes {
		qn := QNode{
			QMin: [3]uint8{
				qm.quantizeAxis(node.Min.X, 0, math.Floor),
				qm.quantizeAxis(node.Min.Y, 1, math.Floor),
				qm.quantizeAxis(node.Min.Z, 2, math.Floor),
			},
			QMax: [3]uint8{
				qm.quantizeAxis(node.Max.X, 0, math.Ceil),
				qm.quantizeAxis(node.Max.Y, 1, math.Ceil),
				qm.quantizeAxis(node.Max.Z, 2, math.Ceil),
			},
		}
		if node.IsLeaf() {
			if node.Count > int(^uint8(0)) {
				return nil, fmt.Errorf("leaf contains %d triangles", node.Count)
			}
			qn.First = uint16(node.First)
			qn.Count = uint8(node.Count)
			qn.Left, qn.Right = -1, -1
		} else {
			qn.Left = int16(node.Left)
			qn.Right = int16(node.Right)
		}
		q.Nodes[i] = qn
	}
	return q, nil
}

func (q *QBVH) nodeAABB(node QNode) AABB {
	return AABB{Min: q.Mesh.Vertex(node.QMin), Max: q.Mesh.Vertex(node.QMax)}
}

func (q *QBVH) Intersect(ray Ray) (Hit, bool) {
	if len(q.Nodes) == 0 {
		return Hit{}, false
	}
	stack := []int{0}
	var best Hit
	found := false
	for len(stack) > 0 {
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := q.Nodes[idx]
		if !q.nodeAABB(node).Hit(ray.Origin, ray.Dir) {
			continue
		}
		if node.Count > 0 {
			start := int(node.First)
			end := start + int(node.Count)
			for k := start; k < end; k++ {
				triIndex := int(q.Indices[k])
				hit, ok := q.Mesh.Triangle(triIndex).Intersect(ray)
				if ok && (!found || hit.T < best.T) {
					best, found = hit, true
				}
			}
			continue
		}
		stack = append(stack, int(node.Left), int(node.Right))
	}
	return best, found
}

func (q *QBVH) ASBytes() int {
	return len(q.Nodes)*QNodeBytes + len(q.Indices)*QIndexBytes
}
