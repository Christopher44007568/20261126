package main

import (
	"fmt"
	"math"
)

type BoundsMismatch struct {
	NodeID  int32  `json:"node_id"`
	WantMin Vec3   `json:"want_min"`
	WantMax Vec3   `json:"want_max"`
	GotMin  Vec3   `json:"got_min"`
	GotMax  Vec3   `json:"got_max"`
	Delta   Vec3   `json:"delta"`
	Reason  string `json:"reason"`
}

type StructureReport struct {
	NodeCount              int              `json:"node_count"`
	TriangleCount          int              `json:"triangle_count"`
	InstanceCount          int              `json:"instance_count"`
	OriginalRootAddress    string           `json:"original_root_address"`
	RelocatedRootAddress   string           `json:"relocated_root_address"`
	AddressesChanged       bool             `json:"addresses_changed"`
	NodeAddressChanges     int              `json:"node_address_changes"`
	TriangleOrderChanged   bool             `json:"triangle_order_changed"`
	OriginalOrderFirst     []int            `json:"original_order_first"`
	RelocatedOrderFirst    []int            `json:"relocated_order_first"`
	DegenerateTriangles    int              `json:"degenerate_triangles"`
	DeepInstancePathLength int              `json:"deep_instance_path_length"`
	BoundsMismatches       []BoundsMismatch `json:"bounds_mismatches"`
	TopologyMismatches     []string         `json:"topology_mismatches"`
	LeafOrderMismatches    []string         `json:"leaf_order_mismatches"`
	Checksum               string           `json:"checksum"`
	OK                     bool             `json:"ok"`
}

func ValidateStructure(scene *DemoScene, rt *RelocatedAS) StructureReport {
	origNodes := CollectPointerNodes(scene.BVH.Root)
	rep := StructureReport{
		NodeCount:            len(rt.Nodes),
		TriangleCount:        len(rt.Triangles),
		InstanceCount:        len(rt.Instances),
		OriginalRootAddress:  fmt.Sprintf("%p", scene.BVH.Root),
		RelocatedRootAddress: fmt.Sprintf("%p", rt.Root),
		Checksum:             fmt.Sprintf("%08x", rt.Checksum),
		BoundsMismatches:     []BoundsMismatch{},
		TopologyMismatches:   []string{},
		LeafOrderMismatches:  []string{},
	}
	rep.AddressesChanged = rep.OriginalRootAddress != rep.RelocatedRootAddress

	limit := minInt(12, len(scene.Triangles))
	rep.OriginalOrderFirst = make([]int, limit)
	for i := 0; i < limit; i++ {
		rep.OriginalOrderFirst[i] = i
	}
	rep.RelocatedOrderFirst = make([]int, limit)
	for i := 0; i < limit && i < len(rt.Triangles); i++ {
		rep.RelocatedOrderFirst[i] = rt.Triangles[i].OriginalIndex
	}
	rep.TriangleOrderChanged = !equalInts(rep.OriginalOrderFirst, rep.RelocatedOrderFirst)

	for _, tri := range scene.Triangles {
		if tri.Degenerate {
			rep.DegenerateTriangles++
		}
	}
	if leaf := FindInstance(scene.Root, deepLeafID); leaf != nil {
		rep.DeepInstancePathLength = leaf.Depth + 1
	}

	if len(origNodes) != len(rt.Nodes) {
		rep.TopologyMismatches = append(rep.TopologyMismatches,
			fmt.Sprintf("node count original=%d relocated=%d", len(origNodes), len(rt.Nodes)))
	}

	for i := 0; i < minInt(len(origNodes), len(rt.Nodes)); i++ {
		orig := origNodes[i]
		got := &rt.Nodes[i]
		if fmt.Sprintf("%p", orig) != fmt.Sprintf("%p", got) {
			rep.NodeAddressChanges++
		}
		if mm, bad := compareBounds(int32(i), orig.Bounds, got.Bounds); bad {
			rep.BoundsMismatches = append(rep.BoundsMismatches, mm)
		}

		wantLeft, wantRight := int32(-1), int32(-1)
		if orig.Left != nil {
			wantLeft = int32(orig.Left.SerializedID)
		}
		if orig.Right != nil {
			wantRight = int32(orig.Right.SerializedID)
		}
		if wantLeft != got.LeftID || wantRight != got.RightID {
			rep.TopologyMismatches = append(rep.TopologyMismatches,
				fmt.Sprintf("node %d child ids want=(%d,%d) got=(%d,%d)", i, wantLeft, wantRight, got.LeftID, got.RightID))
		}
		if orig.First != got.First || orig.Count != got.Count {
			rep.TopologyMismatches = append(rep.TopologyMismatches,
				fmt.Sprintf("node %d range want=(%d,%d) got=(%d,%d)", i, orig.First, orig.Count, got.First, got.Count))
		}

		if orig.Left == nil && orig.Right == nil {
			for k := 0; k < orig.Count; k++ {
				pos := orig.First + k
				wantOriginalTri := scene.BVH.Order[pos]
				storageIndex := int(rt.Order[pos])
				gotOriginalTri := rt.Triangles[storageIndex].OriginalIndex
				if wantOriginalTri != gotOriginalTri {
					rep.LeafOrderMismatches = append(rep.LeafOrderMismatches,
						fmt.Sprintf("node %d item %d want original triangle %d got %d", i, k, wantOriginalTri, gotOriginalTri))
				}
			}
		}
	}

	rep.OK = rep.AddressesChanged &&
		rep.TriangleOrderChanged &&
		len(rep.BoundsMismatches) == 0 &&
		len(rep.TopologyMismatches) == 0 &&
		len(rep.LeafOrderMismatches) == 0 &&
		rep.NodeAddressChanges == rep.NodeCount
	return rep
}

func compareBounds(id int32, want, got AABB) (BoundsMismatch, bool) {
	const eps = 1e-12
	bad := !vecClose(want.Min, got.Min, eps) || !vecClose(want.Max, got.Max, eps)
	if !bad {
		return BoundsMismatch{}, false
	}
	return BoundsMismatch{
		NodeID:  id,
		WantMin: want.Min, WantMax: want.Max,
		GotMin: got.Min, GotMax: got.Max,
		Delta:  got.Center().Sub(want.Center()),
		Reason: "AABB differs after relocation",
	}, true
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type RayComparison struct {
	Name                     string   `json:"name"`
	Mode                     string   `json:"mode"`
	OK                       bool     `json:"ok"`
	Hit                      bool     `json:"hit"`
	SurfaceID                uint64   `json:"surface_id"`
	OriginalTriangle         int      `json:"original_triangle"`
	ReloadedOriginalTriangle int      `json:"reloaded_original_triangle"`
	TOriginal                float64  `json:"t_original"`
	TReloaded                float64  `json:"t_reloaded"`
	TError                   float64  `json:"t_error"`
	PointError               float64  `json:"point_error"`
	NormalError              float64  `json:"normal_error"`
	ExpectedError            float64  `json:"expected_error,omitempty"`
	InstancePath             []uint32 `json:"instance_path,omitempty"`
	Error                    string   `json:"error,omitempty"`
}

type RaySuiteReport struct {
	ProbeCount       int             `json:"probe_count"`
	ProbePassed      int             `json:"probe_passed"`
	ProbeFailed      int             `json:"probe_failed"`
	Passed           int             `json:"passed"`
	Failed           int             `json:"failed"`
	GoldSurfaceID    uint64          `json:"gold_surface_id"`
	RandomRayCount   int             `json:"random_ray_count"`
	RandomMismatches int             `json:"random_mismatches"`
	MaxPointError    float64         `json:"max_point_error"`
	MaxTError        float64         `json:"max_t_error"`
	Comparisons      []RayComparison `json:"comparisons"`
	OK               bool            `json:"ok"`
}

func compareWorldRay(name, mode string, ray Ray, a, b Hit, expectedWorld *Vec3, requireGold bool) RayComparison {
	cmp := RayComparison{
		Name:         name,
		Mode:         mode,
		Hit:          a.OK,
		OK:           true,
		InstancePath: append([]uint32{}, b.InstancePath...),
	}
	if a.OK != b.OK {
		cmp.OK = false
		cmp.Error = fmt.Sprintf("hit presence original=%v reloaded=%v", a.OK, b.OK)
		return cmp
	}
	if !a.OK {
		return cmp
	}

	cmp.SurfaceID = b.SurfaceID
	cmp.OriginalTriangle = a.OriginalTriangle
	cmp.ReloadedOriginalTriangle = b.OriginalTriangle
	cmp.TOriginal = a.T
	cmp.TReloaded = b.T
	tErr := math.Abs(a.T - b.T)
	cmp.TError = tErr
	cmp.PointError = a.Point.Sub(b.Point).Len()
	cmp.NormalError = a.Normal.Sub(b.Normal).Len()
	if tErr > 1e-9 || cmp.PointError > 1e-8 || cmp.NormalError > 1e-8 {
		cmp.OK = false
		cmp.Error = fmt.Sprintf("intersection drift t=%.3g point=%.3g normal=%.3g", tErr, cmp.PointError, cmp.NormalError)
	}
	if a.SurfaceID != b.SurfaceID || a.OriginalTriangle != b.OriginalTriangle {
		cmp.OK = false
		cmp.Error = fmt.Sprintf("identity mismatch surface=(%d,%d) tri=(%d,%d)",
			a.SurfaceID, b.SurfaceID, a.OriginalTriangle, b.OriginalTriangle)
	}
	if expectedWorld != nil {
		cmp.ExpectedError = b.Point.Sub(*expectedWorld).Len()
		if cmp.ExpectedError > 2e-6 {
			cmp.OK = false
			cmp.Error = fmt.Sprintf("missed expected surface point by %.3g", cmp.ExpectedError)
		}
	}
	if requireGold && b.SurfaceID != SurfaceGold {
		cmp.OK = false
		cmp.Error = fmt.Sprintf("landed on surface %d, want gold %d", b.SurfaceID, SurfaceGold)
	}
	return cmp
}
