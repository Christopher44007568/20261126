package main

import "math"

const (
	IDFloor = iota
	IDPelvis
	IDTorso
	IDHead
	IDVisor
	IDLeftArm
	IDRightArm
	IDLeftLeg
	IDRightLeg
)

const (
	StaleID  = IDLeftArm
	MissedID = IDRightLeg
)

type Scene struct {
	Width  int
	Height int
	Camera Camera
	Light  Vec3
	TLAS   *TLAS
	BLAS   []*BLAS
	Frame  int

	leftLeg  []float64
	rightLeg []float64
	leftArm  []float64
	rightArm []float64
	bob      []float64
}

func deg(v float64) float64 { return v * math.Pi / 180 }

func hexColor(hex uint32) Vec3 {
	r := float64((hex>>16)&0xff) / 255
	g := float64((hex>>8)&0xff) / 255
	b := float64(hex&0xff) / 255
	return V(
		srgbToLinear(r),
		srgbToLinear(g),
		srgbToLinear(b),
	)
}

func NewScene(width, height int) *Scene {
	angles := struct {
		leftLeg  []float64
		rightLeg []float64
		leftArm  []float64
		rightArm []float64
		bob      []float64
	}{
		leftLeg:  []float64{28, 7, -28, -7},
		rightLeg: []float64{-28, -7, 28, 7},
		leftArm:  []float64{0, -72, -28, 72},
		rightArm: []float64{24, 6, -24, -6},
		bob:      []float64{0, 0.055, 0, 0.055},
	}

	floorBLAS := NewBLAS("floor", BoxMesh(V(0, -0.06, 0), V(16, 0.06, 16)))
	pelvisBLAS := NewBLAS("pelvis", BoxMesh(V(0, 0, 0), V(0.38, 0.19, 0.23)))
	torsoMesh := mergeTriangles(
		BoxMesh(V(0, 0, 0), V(0.44, 0.43, 0.24)),
		BoxMesh(V(0, 0.43, 0), V(0.15, 0.12, 0.16)),
		BoxMesh(V(0, 0.10, 0.245), V(0.28, 0.16, 0.025)),
	)
	torsoBLAS := NewBLAS("torso", torsoMesh)
	headMesh := mergeTriangles(
		SphereMesh(V(0, 0, 0), 0.32, 20, 12),
		SphereMesh(V(0, 0.39, 0), 0.055, 10, 6),
	)
	headBLAS := NewBLAS("head", headMesh)
	visorBLAS := NewBLAS("visor", BoxMesh(V(0, 0.02, 0.30), V(0.235, 0.085, 0.045)))
	armMesh := mergeTriangles(
		BoxMesh(V(0, -0.34, 0), V(0.135, 0.37, 0.15)),
		SphereMesh(V(0, -0.78, 0), 0.16, 12, 8),
	)
	armBLAS := NewBLAS("arm", armMesh)
	legMesh := mergeTriangles(
		BoxMesh(V(0, -0.43, 0), V(0.17, 0.45, 0.18)),
		BoxMesh(V(0, -0.96, 0.13), V(0.18, 0.11, 0.31)),
	)
	legBLAS := NewBLAS("leg", legMesh)

	scene := &Scene{
		Width:    width,
		Height:   height,
		Camera:   NewCamera(V(4.1, 2.55, 6.4), V(0, 1.35, 0), V(0, 1, 0), 38, float64(width)/float64(height), width, height),
		Light:    V(0.48, 0.76, 0.44).Norm(),
		BLAS:     []*BLAS{floorBLAS, pelvisBLAS, torsoBLAS, headBLAS, visorBLAS, armBLAS, legBLAS},
		leftLeg:  angles.leftLeg,
		rightLeg: angles.rightLeg,
		leftArm:  angles.leftArm,
		rightArm: angles.rightArm,
		bob:      angles.bob,
	}

	parts := []struct {
		name   string
		blas   *BLAS
		color  uint32
		status InstanceStatus
	}{
		{"floor", floorBLAS, 0xd9e1ec, StatusNormal},
		{"pelvis", pelvisBLAS, 0x1d4ed8, StatusNormal},
		{"torso", torsoBLAS, 0x2563eb, StatusNormal},
		{"head", headBLAS, 0xffd7a8, StatusNormal},
		{"visor", visorBLAS, 0x101827, StatusNormal},
		{"left_arm", armBLAS, 0x38bdf8, StatusStaleBounds},
		{"right_arm", armBLAS, 0x38bdf8, StatusNormal},
		{"left_leg", legBLAS, 0x1e40af, StatusNormal},
		{"right_leg", legBLAS, 0x1e3a8a, StatusMissedUpdate},
	}

	instances := make([]*Instance, len(parts))
	for id, part := range parts {
		instances[id] = NewInstance(
			id, part.name, part.blas,
			scene.poseMatrix(id, 0),
			hexColor(part.color), part.status,
		)
		instances[id].LastFrame = 0
	}
	scene.TLAS = BuildTLAS(instances)
	return scene
}

func (s *Scene) poseMatrix(id, frame int) Mat4 {
	bob := s.bob[frame]
	switch id {
	case IDFloor:
		return Identity()
	case IDPelvis:
		return Translate(V(0, 1.13+bob, 0))
	case IDTorso:
		return Translate(V(0, 1.61+bob, 0))
	case IDHead:
		return Translate(V(0, 2.25+bob, 0))
	case IDVisor:
		return Translate(V(0, 2.25+bob, 0))
	case IDLeftArm:
		return Mul(
			Translate(V(0.59, 1.91+bob, 0)),
			RotX(deg(s.leftArm[frame])),
		)
	case IDRightArm:
		return Mul(
			Translate(V(-0.59, 1.91+bob, 0)),
			RotX(deg(s.rightArm[frame])),
		)
	case IDLeftLeg:
		return Mul(
			Translate(V(0.23, 1.08+bob, 0)),
			RotX(deg(s.leftLeg[frame])),
		)
	case IDRightLeg:
		return Mul(
			Translate(V(-0.23, 1.08+bob, 0)),
			RotX(deg(s.rightLeg[frame])),
		)
	default:
		return Identity()
	}
}

// Advance updates the persistent TLAS for one new pose. BLAS objects are not
// rebuilt; only instance matrices, leaf bounds, and node bounds change.
func (s *Scene) Advance(frame int) {
	s.Frame = frame
	for id := IDFloor + 1; id <= IDRightLeg; id++ {
		switch id {
		case MissedID:
			s.TLAS.SkipUpdate(id)
		case StaleID:
			s.TLAS.UpdateTransformWithStaleBounds(id, s.poseMatrix(id, frame), frame)
		default:
			s.TLAS.UpdateTransform(id, s.poseMatrix(id, frame), frame)
		}
	}
	s.TLAS.Refit()
}
