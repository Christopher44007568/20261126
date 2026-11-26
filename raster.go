package visibilitybuffer

import (
	"errors"
	"math"
)

var sampleOffsets = map[int][][2]float64{
	1: {
		{0.5, 0.5},
	},
	4: {
		{0.375, 0.125},
		{0.875, 0.375},
		{0.125, 0.625},
		{0.625, 0.875},
	},
	8: {
		{0.5625, 0.3125},
		{0.4375, 0.6875},
		{0.8125, 0.5625},
		{0.3125, 0.1875},
		{0.1875, 0.8125},
		{0.0625, 0.4375},
		{0.6875, 0.9375},
		{0.9375, 0.0625},
	},
}

func Render(triangles []Triangle, width, height, sampleCount int) (*VisibilityBuffer, error) {
	buffer, err := NewVisibilityBuffer(width, height, sampleCount)
	if err != nil {
		return nil, err
	}

	for triangleIndex := range triangles {
		if err := validateTriangle(triangles[triangleIndex]); err != nil {
			return nil, err
		}
		rasterizeTriangle(buffer, &triangles[triangleIndex])
	}

	return buffer, nil
}

func validateTriangle(triangle Triangle) error {
	if triangle.ID == EmptyID {
		return errors.New("triangle ID 0 is reserved for empty samples")
	}

	for _, vertex := range triangle.Vertices {
		if !finiteVec3(vertex.Position) {
			return errors.New("triangle positions must be finite")
		}
		if !finiteVec3(vertex.Normal) || vertex.Normal == (Vec3{}) {
			return errors.New("triangle normals must be finite and non-zero")
		}
	}

	if triangle.Coverage != nil {
		for _, coverage := range triangle.Coverage {
			if math.IsNaN(coverage) || coverage < 0 || coverage > 1 {
				return errors.New("coverage values must be in [0, 1]")
			}
		}
		if math.IsNaN(triangle.AlphaThreshold) || triangle.AlphaThreshold < 0 || triangle.AlphaThreshold > 1 {
			return errors.New("alpha threshold must be in [0, 1]")
		}
	}

	return nil
}

func finiteVec3(value Vec3) bool {
	return !math.IsNaN(value.X) && !math.IsInf(value.X, 0) &&
		!math.IsNaN(value.Y) && !math.IsInf(value.Y, 0) &&
		!math.IsNaN(value.Z) && !math.IsInf(value.Z, 0)
}

func rasterizeTriangle(buffer *VisibilityBuffer, triangle *Triangle) {
	minX, maxX := bounds(triangle.Vertices[0].Position.X, triangle.Vertices[1].Position.X, triangle.Vertices[2].Position.X)
	minY, maxY := bounds(triangle.Vertices[0].Position.Y, triangle.Vertices[1].Position.Y, triangle.Vertices[2].Position.Y)

	startX := maxInt(0, int(math.Floor(minX)))
	endX := minInt(buffer.Width-1, int(math.Ceil(maxX))-1)
	startY := maxInt(0, int(math.Floor(minY)))
	endY := minInt(buffer.Height-1, int(math.Ceil(maxY))-1)

	if startX > endX || startY > endY {
		return
	}

	denominator := (triangle.Vertices[1].Position.Y-triangle.Vertices[2].Position.Y)*
		(triangle.Vertices[0].Position.X-triangle.Vertices[2].Position.X) +
		(triangle.Vertices[2].Position.X-triangle.Vertices[1].Position.X)*
			(triangle.Vertices[0].Position.Y-triangle.Vertices[2].Position.Y)
	if math.Abs(denominator) < 1e-12 {
		return
	}

	offsets := sampleOffsets[buffer.SampleCount]
	for pixelY := startY; pixelY <= endY; pixelY++ {
		for pixelX := startX; pixelX <= endX; pixelX++ {
			for sampleIndex, offset := range offsets {
				sampleX := float64(pixelX) + offset[0]
				sampleY := float64(pixelY) + offset[1]
				lambda0, lambda1, lambda2, inside := barycentric(triangle, denominator, sampleX, sampleY)
				if !inside {
					continue
				}

				depth := interpolateDepth(triangle, lambda0, lambda1, lambda2)
				if math.IsNaN(depth) || math.IsInf(depth, 0) {
					continue
				}
				if triangle.Coverage != nil {
					coverage := interpolateCoverage(triangle, lambda0, lambda1, lambda2)
					if coverage < triangle.AlphaThreshold {
						continue
					}
				}

				index := ((pixelY*buffer.Width+pixelX)*buffer.SampleCount + sampleIndex)
				current := &buffer.Samples[index]
				if depth < current.Depth || (depth == current.Depth && triangle.ID < current.TriangleID) {
					current.TriangleID = triangle.ID
					current.Depth = depth
				}
			}
		}
	}
}

func barycentric(triangle *Triangle, denominator, x, y float64) (float64, float64, float64, bool) {
	v0 := triangle.Vertices[0].Position
	v1 := triangle.Vertices[1].Position
	v2 := triangle.Vertices[2].Position

	lambda0 := ((v1.Y-v2.Y)*(x-v2.X) + (v2.X-v1.X)*(y-v2.Y)) / denominator
	lambda1 := ((v2.Y-v0.Y)*(x-v2.X) + (v0.X-v2.X)*(y-v2.Y)) / denominator
	lambda2 := 1 - lambda0 - lambda1

	const epsilon = 1e-10
	return lambda0, lambda1, lambda2, lambda0 >= -epsilon && lambda1 >= -epsilon && lambda2 >= -epsilon
}

func interpolateDepth(triangle *Triangle, lambda0, lambda1, lambda2 float64) float64 {
	return triangle.Vertices[0].Position.Z*lambda0 +
		triangle.Vertices[1].Position.Z*lambda1 +
		triangle.Vertices[2].Position.Z*lambda2
}

func interpolateCoverage(triangle *Triangle, lambda0, lambda1, lambda2 float64) float64 {
	return triangle.Coverage[0]*lambda0 +
		triangle.Coverage[1]*lambda1 +
		triangle.Coverage[2]*lambda2
}

func bounds(a, b, c float64) (float64, float64) {
	return math.Min(a, math.Min(b, c)), math.Max(a, math.Max(b, c))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
