package shadow

import (
	"image"
	"image/png"
	"os"
)

// WritePNG encodes an image to path using only the standard library.
func WritePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

// TagCount summarizes how many receiver pixels carry each diagnostic tag.
type TagCount struct {
	ExtraOccluder   int
	IgnoredBackface int
	ClosureExit     int
}

// CountTags re-renders conceptually by scanning pixel evaluations directly.
func (s *Scene) CountTags(flag RayFlag) TagCount {
	var counts TagCount
	for py := 0; py < ImageSize; py++ {
		for px := 0; px < ImageSize; px++ {
			eval := s.Evaluate(ShadowRay(receiverPoint(px, py)), flag)
			switch eval.Tag {
			case TagExtraOccluder:
				counts.ExtraOccluder++
			case TagIgnoredBackface:
				counts.IgnoredBackface++
			case TagClosureExit:
				counts.ClosureExit++
			}
		}
	}
	return counts
}
