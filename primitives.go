package rayao

// NewCube returns twelve outward-wound triangles for an axis-aligned cube.
func NewCube(center Vec3, size float64, material int) []Triangle {
	h := size * 0.5
	return NewBox(center, Vec3{X: h, Y: h, Z: h}, material)
}

// NewBox returns twelve outward-wound triangles for an axis-aligned box.
func NewBox(center, halfExtents Vec3, material int) []Triangle {
	x0, x1 := center.X-halfExtents.X, center.X+halfExtents.X
	y0, y1 := center.Y-halfExtents.Y, center.Y+halfExtents.Y
	z0, z1 := center.Z-halfExtents.Z, center.Z+halfExtents.Z

	quads := [][4]Vec3{
		{{x1, y0, z0}, {x1, y1, z0}, {x1, y1, z1}, {x1, y0, z1}}, // +X
		{{x0, y0, z0}, {x0, y0, z1}, {x0, y1, z1}, {x0, y1, z0}}, // -X
		{{x0, y1, z0}, {x0, y1, z1}, {x1, y1, z1}, {x1, y1, z0}}, // +Y
		{{x0, y0, z0}, {x1, y0, z0}, {x1, y0, z1}, {x0, y0, z1}}, // -Y
		{{x0, y0, z1}, {x1, y0, z1}, {x1, y1, z1}, {x0, y1, z1}}, // +Z
		{{x0, y0, z0}, {x0, y1, z0}, {x1, y1, z0}, {x1, y0, z0}}, // -Z
	}

	triangles := make([]Triangle, 0, len(quads)*2)
	for _, q := range quads {
		triangles = append(triangles,
			NewTriangle(q[0], q[1], q[2], material),
			NewTriangle(q[0], q[2], q[3], material),
		)
	}
	return triangles
}
