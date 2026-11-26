package visibilitybuffer

import (
	"errors"
	"fmt"
	"math"
)

const EmptyID uint32 = 0

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

type Vertex struct {
	Position Vec3
	Normal   Vec3
}

type Triangle struct {
	ID         uint32
	InstanceID uint32
	MaterialID uint32
	Vertices   [3]Vertex

	// Coverage is an optional alpha-mask value at each vertex. A nil triangle
	// is fully opaque. Values are interpolated at MSAA samples and compared
	// with AlphaThreshold.
	Coverage       *[3]float64
	AlphaThreshold float64
}

type VisibilitySample struct {
	TriangleID uint32
	Depth      float64
}

type VisibilityBuffer struct {
	Width       int
	Height      int
	SampleCount int
	Samples     []VisibilitySample
}

type Framebuffer struct {
	Visibility *VisibilityBuffer
	GeometryID []uint32
	InstanceID []uint32
	MaterialID []uint32
	Normal     []Vec3
	EdgeDefect []uint8
}

func NewVisibilityBuffer(width, height, sampleCount int) (*VisibilityBuffer, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("framebuffer dimensions must be positive")
	}
	if !validSampleCount(sampleCount) {
		return nil, fmt.Errorf("unsupported sample count %d: use 1, 4, or 8", sampleCount)
	}

	samples := make([]VisibilitySample, width*height*sampleCount)
	for i := range samples {
		samples[i].Depth = math.Inf(1)
	}

	return &VisibilityBuffer{
		Width:       width,
		Height:      height,
		SampleCount: sampleCount,
		Samples:     samples,
	}, nil
}

func validSampleCount(sampleCount int) bool {
	return sampleCount == 1 || sampleCount == 4 || sampleCount == 8
}
