package visibilitybuffer

import (
	"math"
	"testing"
)

func TestRenderKeepsNearestTriangleRegardlessOfOrder(t *testing.T) {
	near := screenTriangle(7, 0.4, Vec3{X: 0, Y: 0, Z: 1})
	far := screenTriangle(9, 0.8, Vec3{X: 0, Y: 0, Z: -1})

	buffer, err := Render([]Triangle{far, near}, 4, 4, 1)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	for pixelIndex := 0; pixelIndex < len(buffer.Samples); pixelIndex++ {
		if got := buffer.Samples[pixelIndex].TriangleID; got != near.ID {
			t.Fatalf("sample %d ID = %d, want %d", pixelIndex, got, near.ID)
		}
		if got := buffer.Samples[pixelIndex].Depth; math.Abs(got-0.4) > 1e-12 {
			t.Fatalf("sample %d depth = %v, want 0.4", pixelIndex, got)
		}
	}
}

func TestBuildFramebufferReconstructsRepeatedMaterialAndNormal(t *testing.T) {
	near := screenTriangleWithMaterial(7, 0.2, 3, Vec3{X: 1, Y: 0, Z: 0})
	far := screenTriangleWithMaterial(9, 0.9, 3, Vec3{X: 0, Y: 1, Z: 0})

	buffer, err := Render([]Triangle{far, near}, 4, 4, 1)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	framebuffer, err := BuildFramebuffer(buffer, []Triangle{near, far})
	if err != nil {
		t.Fatalf("BuildFramebuffer() error: %v", err)
	}

	for pixelIndex := range framebuffer.GeometryID {
		if framebuffer.GeometryID[pixelIndex] != 7 {
			t.Fatalf("geometry ID = %d, want 7", framebuffer.GeometryID[pixelIndex])
		}
		if framebuffer.MaterialID[pixelIndex] != 3 {
			t.Fatalf("material ID = %d, want repeated material 3", framebuffer.MaterialID[pixelIndex])
		}
		if !approxVec(framebuffer.Normal[pixelIndex], Vec3{X: 1, Y: 0, Z: 0}) {
			t.Fatalf("normal = %+v, want interpolated nearest normal", framebuffer.Normal[pixelIndex])
		}
	}
}

func TestTransparentLeafUsesAlphaCoverageAndDepth(t *testing.T) {
	leaf := screenTriangle(11, 0.1, Vec3{X: 0, Y: 0, Z: 1})
	leaf.Coverage = &[3]float64{0.1, 0.1, 0.1}
	leaf.AlphaThreshold = 0.5
	far := screenTriangle(12, 0.9, Vec3{X: 0, Y: 1, Z: 0})

	buffer, err := Render([]Triangle{leaf, far}, 4, 4, 4)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	for sampleIndex, sample := range buffer.Samples {
		if sample.TriangleID != far.ID {
			t.Fatalf("sample %d ID = %d, want alpha-tested leaf rejected and far ID %d", sampleIndex, sample.TriangleID, far.ID)
		}
	}
}

func TestPartialTransparentLeafKeepsNearestVisibleSample(t *testing.T) {
	leaf := screenTriangle(13, 0.1, Vec3{X: 0, Y: 0, Z: 1})
	leaf.InstanceID = 55
	leaf.Coverage = &[3]float64{0.9, 0.9, 0.9}
	leaf.AlphaThreshold = 0.5
	far := screenTriangle(14, 0.9, Vec3{X: 0, Y: 1, Z: 0})

	buffer, err := Render([]Triangle{far, leaf}, 4, 4, 4)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	framebuffer, err := BuildFramebuffer(buffer, []Triangle{leaf, far})
	if err != nil {
		t.Fatalf("BuildFramebuffer() error: %v", err)
	}

	for pixelIndex := range framebuffer.GeometryID {
		if framebuffer.GeometryID[pixelIndex] != leaf.ID {
			t.Fatalf("geometry ID = %d, want visible leaf %d", framebuffer.GeometryID[pixelIndex], leaf.ID)
		}
		if framebuffer.InstanceID[pixelIndex] != leaf.InstanceID {
			t.Fatalf("instance ID = %d, want leaf instance %d", framebuffer.InstanceID[pixelIndex], leaf.InstanceID)
		}
		if framebuffer.MaterialID[pixelIndex] != leaf.MaterialID {
			t.Fatalf("material ID = %d, want leaf material %d", framebuffer.MaterialID[pixelIndex], leaf.MaterialID)
		}
	}
}

func TestRenderRejectsNonFiniteDepth(t *testing.T) {
	triangle := screenTriangle(41, 0.1, Vec3{X: 0, Y: 0, Z: 1})
	triangle.Vertices[1].Position.Z = math.NaN()

	if _, err := Render([]Triangle{triangle}, 2, 2, 1); err == nil {
		t.Fatal("Render() expected error for NaN depth")
	}
}

func TestBuildFramebufferRejectsUnknownGeometryID(t *testing.T) {
	buffer, err := NewVisibilityBuffer(1, 1, 1)
	if err != nil {
		t.Fatalf("NewVisibilityBuffer() error: %v", err)
	}
	buffer.Samples[0] = VisibilitySample{TriangleID: 99, Depth: 0.2}

	if _, err := BuildFramebuffer(buffer, nil); err == nil {
		t.Fatal("BuildFramebuffer() expected error for unknown geometry ID")
	}
}

func TestInstanceOverlapUsesDepthWhenTriangleOrderIsReversed(t *testing.T) {
	near := screenTriangleWithMaterial(31, 0.2, 40, Vec3{X: 1, Y: 0, Z: 0})
	near.InstanceID = 501
	far := screenTriangleWithMaterial(32, 0.8, 41, Vec3{X: 0, Y: 1, Z: 0})
	far.InstanceID = 909

	buffer, err := Render([]Triangle{near, far}, 4, 4, 4)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	framebuffer, err := BuildFramebuffer(buffer, []Triangle{far, near})
	if err != nil {
		t.Fatalf("BuildFramebuffer() error: %v", err)
	}

	for _, geometryID := range framebuffer.GeometryID {
		if geometryID != near.ID {
			t.Fatalf("geometry ID = %d, want nearest instance triangle %d", geometryID, near.ID)
		}
	}
}

func TestMSAAEdgeDefectMarksOnlyMixedCoveragePixels(t *testing.T) {
	triangle := Triangle{
		ID: 21,
		Vertices: [3]Vertex{
			{Position: Vec3{X: 0, Y: -1, Z: 0.3}, Normal: Vec3{X: 0, Y: 0, Z: 1}},
			{Position: Vec3{X: 2, Y: 0, Z: 0.3}, Normal: Vec3{X: 0, Y: 0, Z: 1}},
			{Position: Vec3{X: 0, Y: 1, Z: 0.3}, Normal: Vec3{X: 0, Y: 0, Z: 1}},
		},
	}

	buffer, err := Render([]Triangle{triangle}, 2, 2, 4)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	framebuffer, err := BuildFramebuffer(buffer, []Triangle{triangle})
	if err != nil {
		t.Fatalf("BuildFramebuffer() error: %v", err)
	}

	edgePixels := 0
	fullPixels := 0
	for pixelIndex, defect := range framebuffer.EdgeDefect {
		switch defect {
		case 0:
			fullPixels++
		case 1:
			edgePixels++
			if framebuffer.GeometryID[pixelIndex] != triangle.ID {
				t.Fatalf("edge pixel %d did not resolve covered geometry", pixelIndex)
			}
		default:
			t.Fatalf("edge defect = %d, want 0 or 1", defect)
		}
	}

	if edgePixels == 0 {
		t.Fatal("expected at least one mixed MSAA edge pixel")
	}
	if fullPixels == 0 {
		t.Fatal("expected at least one fully covered non-edge pixel")
	}
}

func TestEightSampleMSAAProducesEdgeDefects(t *testing.T) {
	triangle := Triangle{
		ID: 22,
		Vertices: [3]Vertex{
			{Position: Vec3{X: 0, Y: -1, Z: 0.3}, Normal: Vec3{X: 0, Y: 0, Z: 1}},
			{Position: Vec3{X: 2, Y: 0, Z: 0.3}, Normal: Vec3{X: 0, Y: 0, Z: 1}},
			{Position: Vec3{X: 0, Y: 1, Z: 0.3}, Normal: Vec3{X: 0, Y: 0, Z: 1}},
		},
	}

	buffer, err := Render([]Triangle{triangle}, 2, 2, 8)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if len(buffer.Samples) != 32 {
		t.Fatalf("sample count = %d, want 32", len(buffer.Samples))
	}

	framebuffer, err := BuildFramebuffer(buffer, []Triangle{triangle})
	if err != nil {
		t.Fatalf("BuildFramebuffer() error: %v", err)
	}

	edgePixels := 0
	for _, defect := range framebuffer.EdgeDefect {
		if defect == 1 {
			edgePixels++
		}
	}
	if edgePixels == 0 {
		t.Fatal("expected mixed 8x MSAA edge pixels")
	}
}

func screenTriangle(id uint32, depth float64, normal Vec3) Triangle {
	return screenTriangleWithMaterial(id, depth, 1, normal)
}

func screenTriangleWithMaterial(id uint32, depth float64, materialID uint32, normal Vec3) Triangle {
	return Triangle{
		ID:         id,
		InstanceID: id + 100,
		MaterialID: materialID,
		Vertices: [3]Vertex{
			{Position: Vec3{X: -1, Y: -1, Z: depth}, Normal: normal},
			{Position: Vec3{X: 10, Y: -1, Z: depth}, Normal: normal},
			{Position: Vec3{X: -1, Y: 10, Z: depth}, Normal: normal},
		},
	}
}

func approxVec(got, want Vec3) bool {
	const epsilon = 1e-12
	return math.Abs(got.X-want.X) < epsilon &&
		math.Abs(got.Y-want.Y) < epsilon &&
		math.Abs(got.Z-want.Z) < epsilon
}
