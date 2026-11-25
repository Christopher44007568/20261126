package main

import (
	"fmt"
)

type FirstMismatch struct {
	Found               bool       `json:"found"`
	X                   int        `json:"x"`
	Y                   int        `json:"y"`
	ObjectID            ObjectID   `json:"object_id"`
	ObjectName          string     `json:"object_name"`
	ExpectedHitGroupID  HitGroupID `json:"expected_hit_group_id"`
	ExpectedHitGroup    string     `json:"expected_hit_group"`
	ActualHitGroupID    HitGroupID `json:"actual_hit_group_id"`
	ActualHitGroup      string     `json:"actual_hit_group"`
	ExpectedMaterialID  MaterialID `json:"expected_material_id"`
	ActualMaterialID    MaterialID `json:"actual_material_id"`
	Difference          []string   `json:"difference"`
	ExpectedShadedColor string     `json:"expected_shaded_color"`
	ActualShadedColor   string     `json:"actual_shaded_color"`
}

// FindFirstMismatch scans in image order. A mismatch in identity maps or in the
// final color is an error; the identity report always uses the reference hit.
func FindFirstMismatch(scene *Scene, reference, broken *RenderResult) (FirstMismatch, error) {
	if reference.Width != broken.Width || reference.Height != broken.Height {
		return FirstMismatch{}, fmt.Errorf(
			"render sizes differ: %dx%d vs %dx%d",
			reference.Width, reference.Height, broken.Width, broken.Height,
		)
	}
	for y := 0; y < reference.Height; y++ {
		for x := 0; x < reference.Width; x++ {
			want := reference.At(x, y)
			got := broken.At(x, y)
			difference := pixelDifferences(want, got)
			if len(difference) == 0 {
				continue
			}
			objectName := "none"
			if object, ok := scene.Object(want.ObjectID); ok {
				objectName = object.Name
			}
			wantGroup, _ := scene.Catalog.Group(want.HitGroupID)
			gotGroup, _ := scene.Catalog.Group(got.HitGroupID)
			return FirstMismatch{
				Found: true, X: x, Y: y,
				ObjectID: want.ObjectID, ObjectName: objectName,
				ExpectedHitGroupID: want.HitGroupID, ExpectedHitGroup: wantGroup.Name,
				ActualHitGroupID: got.HitGroupID, ActualHitGroup: gotGroup.Name,
				ExpectedMaterialID: want.MaterialID, ActualMaterialID: got.MaterialID,
				Difference:          difference,
				ExpectedShadedColor: colorHex(want.Color),
				ActualShadedColor:   colorHex(got.Color),
			}, nil
		}
	}
	return FirstMismatch{Found: false, ObjectName: "none"}, nil
}

func pixelDifferences(a, b PixelSample) []string {
	difference := make([]string, 0, 4)
	if a.ObjectID != b.ObjectID {
		difference = append(difference, "object_id")
	}
	if a.HitGroupID != b.HitGroupID {
		difference = append(difference, "hit_group")
	}
	if a.MaterialID != b.MaterialID {
		difference = append(difference, "material_index")
	}
	if a.Color.Sub(b.Color).LengthSquared() > 1e-20 {
		difference = append(difference, "shaded_color")
	}
	return difference
}

func colorHex(value Vec3) string {
	rgba := linearToRGBA(value)
	return fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B)
}

func (m FirstMismatch) Summary() string {
	if !m.Found {
		return "no pixel mismatch found"
	}
	return fmt.Sprintf(
		"first mismatch at (%d,%d): object #%d %q, expected hit group #%d %q, got #%d %q (%v)",
		m.X, m.Y, m.ObjectID, m.ObjectName,
		m.ExpectedHitGroupID, m.ExpectedHitGroup,
		m.ActualHitGroupID, m.ActualHitGroup, m.Difference,
	)
}
