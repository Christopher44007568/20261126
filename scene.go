package rayao

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
)

const SceneSchemaVersion = 1

// Material is a minimal Lambertian reflectance description.
type Material struct {
	Name  string
	Color Vec3
}

// Scene is an immutable, versioned triangle set. Construct it with NewScene
// and do not mutate its slices after renderers or queries start using it.
type Scene struct {
	Name        string
	Schema      int
	Version     string
	ContentHash string
	Materials   []Material
	Triangles   []Triangle
}

// NewScene copies its inputs and assigns a deterministic content version.
func NewScene(name string, triangles []Triangle, materials []Material) *Scene {
	tris := append([]Triangle(nil), triangles...)
	mats := append([]Material(nil), materials...)
	hash := hashScene(name, tris, mats)
	return &Scene{
		Name:        name,
		Schema:      SceneSchemaVersion,
		Version:     hash[:16],
		ContentHash: hash,
		Materials:   mats,
		Triangles:   tris,
	}
}

// Snapshot returns an independent copy carrying the same version. It lets a
// render pass pin the exact scene revision used for all three output maps.
func (s *Scene) Snapshot() *Scene {
	if s == nil {
		return nil
	}
	cp := *s
	cp.Triangles = append([]Triangle(nil), s.Triangles...)
	cp.Materials = append([]Material(nil), s.Materials...)
	return &cp
}

func hashScene(name string, triangles []Triangle, materials []Material) string {
	h := sha256.New()
	var buf [8]byte
	writeFloat := func(v float64) {
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v))
		_, _ = h.Write(buf[:])
	}
	writeInt := func(v int) {
		binary.LittleEndian.PutUint64(buf[:], uint64(v))
		_, _ = h.Write(buf[:])
	}

	writeInt(SceneSchemaVersion)
	writeInt(len(name))
	_, _ = h.Write([]byte(name))
	writeInt(len(materials))
	for _, m := range materials {
		writeInt(len(m.Name))
		_, _ = h.Write([]byte(m.Name))
		writeFloat(m.Color.X)
		writeFloat(m.Color.Y)
		writeFloat(m.Color.Z)
	}
	writeInt(len(triangles))
	for _, t := range triangles {
		for _, p := range t.Vertices {
			writeFloat(p.X)
			writeFloat(p.Y)
			writeFloat(p.Z)
		}
		writeFloat(t.Normal.X)
		writeFloat(t.Normal.Y)
		writeFloat(t.Normal.Z)
		writeInt(t.Material)
	}
	return hex.EncodeToString(h.Sum(nil))
}
