package main

import (
	"errors"
	"fmt"
)

type Tier string

const (
	TierFull       Tier = "full"
	TierCompressed Tier = "compressed"
	TierFallback   Tier = "low_poly_fallback"
)

const MaterialGPUBytes = 256

var ErrStaleHandle = errors.New("acceleration-structure handle is stale")
var ErrBudgetTooSmall = errors.New("budget is smaller than low-poly fallback")

// Package is the only resident GPU resource set for one quality tier.
// Switching tiers calls Evict before constructing the next package.
type Package struct {
	Tier       Tier
	Generation uint64
	Bytes      int
	Triangles  int
	Nodes      int

	BVH     *BVH
	QBVH    *QBVH
	Mesh    *Mesh
	QMesh   *QuantizedMesh
	Evicted bool
}

func NewFullPackage(mesh *Mesh, generation uint64) *Package {
	bvh := BuildBVH(mesh.Tris)
	bytes := bvh.ASBytes() + len(mesh.Tris)*TriangleBytes + MaterialGPUBytes
	return &Package{
		Tier: TierFull, Generation: generation, Bytes: bytes,
		Triangles: len(mesh.Tris), Nodes: len(bvh.Nodes),
		BVH: bvh, Mesh: mesh,
	}
}

func NewCompressedPackage(mesh *Mesh, generation uint64) (*Package, error) {
	qm := QuantizeMesh(mesh, QuantizedLevels)
	qbvh, err := BuildQBVH(qm)
	if err != nil {
		return nil, err
	}
	bytes := qbvh.ASBytes() + qm.Bytes() + MaterialGPUBytes
	return &Package{
		Tier: TierCompressed, Generation: generation, Bytes: bytes,
		Triangles: len(qm.Tris), Nodes: len(qbvh.Nodes),
		QBVH: qbvh, QMesh: qm,
	}, nil
}

func NewFallbackPackage(coarse *Mesh, generation uint64) *Package {
	bvh := BuildBVH(coarse.Tris)
	bytes := bvh.ASBytes() + len(coarse.Tris)*TriangleBytes + MaterialGPUBytes
	return &Package{
		Tier: TierFallback, Generation: generation, Bytes: bytes,
		Triangles: len(coarse.Tris), Nodes: len(bvh.Nodes),
		BVH: bvh, Mesh: coarse,
	}
}

func (p *Package) Render(camera Camera) (*RenderResult, error) {
	if p == nil || p.Evicted {
		return nil, ErrStaleHandle
	}
	switch p.Tier {
	case TierCompressed:
		if p.QBVH == nil {
			return nil, ErrStaleHandle
		}
		return RenderQBVH(p.QBVH, camera), nil
	default:
		if p.BVH == nil || p.Mesh == nil {
			return nil, ErrStaleHandle
		}
		return RenderBVH(p.BVH, p.Mesh.Materials, camera), nil
	}
}

// Evict releases all resident node and geometry references. The immutable
// source meshes remain owned by Manager and can be rebuilt after restoration.
func (p *Package) Evict() {
	if p == nil {
		return
	}
	p.BVH = nil
	p.QBVH = nil
	p.Mesh = nil
	p.QMesh = nil
	p.Evicted = true
}

type Budget struct {
	Tier          Tier    `json:"tier"`
	LimitBytes    int     `json:"limit_bytes"`
	LimitKiB      float64 `json:"limit_kib"`
	RequiredBytes int     `json:"required_bytes"`
	RequiredKiB   float64 `json:"required_kib"`
	SlackBytes    int     `json:"slack_bytes"`
}

type BudgetPlan struct {
	High   Budget `json:"high"`
	Medium Budget `json:"medium"`
	Low    Budget `json:"low"`
}

func makeBudget(tier Tier, limit, required int) Budget {
	return Budget{
		Tier: tier, LimitBytes: limit, LimitKiB: float64(limit) / 1024,
		RequiredBytes: required, RequiredKiB: float64(required) / 1024,
		SlackBytes: limit - required,
	}
}

// NewBudgetPlan chooses three nested limits from measured GPU layout sizes.
// The medium limit cannot hold the full package, and the low limit cannot
// hold the compressed package, so each tier is forced by the budget.
func NewBudgetPlan(detailed, coarse *Mesh) (*BudgetPlan, error) {
	full := NewFullPackage(detailed, 0)
	compressed, err := NewCompressedPackage(detailed, 0)
	if err != nil {
		return nil, err
	}
	fallback := NewFallbackPackage(coarse, 0)

	highLimit := full.Bytes + 512
	mediumGap := full.Bytes - compressed.Bytes
	if mediumGap <= 1 {
		return nil, fmt.Errorf("compressed package (%d) must be smaller than full (%d)", compressed.Bytes, full.Bytes)
	}
	mediumLimit := compressed.Bytes + mediumGap/8
	if mediumLimit >= full.Bytes {
		mediumLimit = full.Bytes - 1
	}

	lowGap := compressed.Bytes - fallback.Bytes
	if lowGap <= 1 {
		return nil, fmt.Errorf("fallback package (%d) must be smaller than compressed (%d)", fallback.Bytes, compressed.Bytes)
	}
	lowLimit := fallback.Bytes + lowGap/8
	if lowLimit >= compressed.Bytes {
		lowLimit = compressed.Bytes - 1
	}

	return &BudgetPlan{
		High:   makeBudget(TierFull, highLimit, full.Bytes),
		Medium: makeBudget(TierCompressed, mediumLimit, compressed.Bytes),
		Low:    makeBudget(TierFallback, lowLimit, fallback.Bytes),
	}, nil
}

func (p BudgetPlan) TierForLimit(limit int) Tier {
	switch {
	case limit >= p.High.RequiredBytes:
		return TierFull
	case limit >= p.Medium.RequiredBytes:
		return TierCompressed
	default:
		return TierFallback
	}
}

func (p BudgetPlan) LimitForTier(tier Tier) int {
	switch tier {
	case TierFull:
		return p.High.LimitBytes
	case TierCompressed:
		return p.Medium.LimitBytes
	default:
		return p.Low.LimitBytes
	}
}
