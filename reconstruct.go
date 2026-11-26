package visibilitybuffer

import (
	"errors"
	"fmt"
	"math"
)

func BuildFramebuffer(buffer *VisibilityBuffer, triangles []Triangle) (*Framebuffer, error) {
	if buffer == nil {
		return nil, errors.New("visibility buffer is required")
	}

	lookup, err := buildTriangleLookup(buffer, triangles)
	if err != nil {
		return nil, err
	}

	pixelCount := buffer.Width * buffer.Height
	framebuffer := &Framebuffer{
		Visibility: buffer,
		GeometryID: make([]uint32, pixelCount),
		InstanceID: make([]uint32, pixelCount),
		MaterialID: make([]uint32, pixelCount),
		Normal:     make([]Vec3, pixelCount),
		EdgeDefect: make([]uint8, pixelCount),
	}

	offsets := sampleOffsets[buffer.SampleCount]
	for pixelY := 0; pixelY < buffer.Height; pixelY++ {
		for pixelX := 0; pixelX < buffer.Width; pixelX++ {
			pixelIndex := pixelY*buffer.Width + pixelX
			bestSample := resolvePixel(buffer, pixelX, pixelY)
			if bestSample.TriangleID == EmptyID {
				framebuffer.EdgeDefect[pixelIndex] = edgeDefect(buffer, pixelX, pixelY)
				continue
			}

			triangle := lookup[bestSample.TriangleID]
			framebuffer.GeometryID[pixelIndex] = bestSample.TriangleID
			framebuffer.InstanceID[pixelIndex] = triangle.InstanceID
			framebuffer.MaterialID[pixelIndex] = triangle.MaterialID
			framebuffer.Normal[pixelIndex] = recoverNormal(
				triangle,
				float64(pixelX)+offsets[bestSample.sampleIndex][0],
				float64(pixelY)+offsets[bestSample.sampleIndex][1],
			)
			framebuffer.EdgeDefect[pixelIndex] = edgeDefect(buffer, pixelX, pixelY)
		}
	}

	return framebuffer, nil
}

type resolvedSample struct {
	TriangleID  uint32
	Depth       float64
	sampleIndex int
}

func resolvePixel(buffer *VisibilityBuffer, pixelX, pixelY int) resolvedSample {
	base := (pixelY*buffer.Width + pixelX) * buffer.SampleCount
	best := resolvedSample{Depth: math.Inf(1), sampleIndex: -1}

	for sampleIndex := 0; sampleIndex < buffer.SampleCount; sampleIndex++ {
		sample := buffer.Samples[base+sampleIndex]
		if sample.TriangleID == EmptyID {
			continue
		}
		if sampleIndex == 0 || sample.Depth < best.Depth ||
			(sample.Depth == best.Depth && sample.TriangleID < best.TriangleID) {
			best = resolvedSample{
				TriangleID:  sample.TriangleID,
				Depth:       sample.Depth,
				sampleIndex: sampleIndex,
			}
		}
	}

	return best
}

func recoverNormal(triangle *Triangle, x, y float64) Vec3 {
	denominator := (triangle.Vertices[1].Position.Y-triangle.Vertices[2].Position.Y)*
		(triangle.Vertices[0].Position.X-triangle.Vertices[2].Position.X) +
		(triangle.Vertices[2].Position.X-triangle.Vertices[1].Position.X)*
			(triangle.Vertices[0].Position.Y-triangle.Vertices[2].Position.Y)
	lambda0, lambda1, lambda2, inside := barycentric(triangle, denominator, x, y)
	if !inside {
		return Vec3{}
	}

	normal := Vec3{
		X: triangle.Vertices[0].Normal.X*lambda0 +
			triangle.Vertices[1].Normal.X*lambda1 +
			triangle.Vertices[2].Normal.X*lambda2,
		Y: triangle.Vertices[0].Normal.Y*lambda0 +
			triangle.Vertices[1].Normal.Y*lambda1 +
			triangle.Vertices[2].Normal.Y*lambda2,
		Z: triangle.Vertices[0].Normal.Z*lambda0 +
			triangle.Vertices[1].Normal.Z*lambda1 +
			triangle.Vertices[2].Normal.Z*lambda2,
	}

	length := math.Sqrt(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)
	if length == 0 {
		return Vec3{}
	}

	return Vec3{X: normal.X / length, Y: normal.Y / length, Z: normal.Z / length}
}

func edgeDefect(buffer *VisibilityBuffer, pixelX, pixelY int) uint8 {
	if buffer.SampleCount == 1 {
		return 0
	}

	base := (pixelY*buffer.Width + pixelX) * buffer.SampleCount
	firstID := buffer.Samples[base].TriangleID
	for sampleIndex := 1; sampleIndex < buffer.SampleCount; sampleIndex++ {
		if buffer.Samples[base+sampleIndex].TriangleID != firstID {
			return 1
		}
	}

	return 0
}

func buildTriangleLookup(buffer *VisibilityBuffer, triangles []Triangle) (map[uint32]*Triangle, error) {
	lookup := make(map[uint32]*Triangle, len(triangles))
	for triangleIndex := range triangles {
		triangle := &triangles[triangleIndex]
		if triangle.ID == EmptyID {
			return nil, errors.New("triangle ID 0 is reserved for empty samples")
		}
		if _, exists := lookup[triangle.ID]; exists {
			return nil, fmt.Errorf("duplicate triangle ID %d", triangle.ID)
		}
		lookup[triangle.ID] = triangle
	}

	for _, sample := range buffer.Samples {
		if sample.TriangleID != EmptyID {
			if _, exists := lookup[sample.TriangleID]; !exists {
				return nil, fmt.Errorf("visibility buffer references unknown triangle ID %d", sample.TriangleID)
			}
		}
	}

	return lookup, nil
}
