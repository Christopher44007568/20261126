package shadow

// ReceiverY is the height of the lit ground plane that receives shadows.
const ReceiverY = 0.0

// Object heights: every sheet and the cube float above the receiver.
const (
	sheetY   = 3.0
	cubeY    = 2.0
	cubeHalf = 1.2
)

// CubeTopY is the height of the closed cube's top face.
const CubeTopY = cubeY + cubeHalf

// BuildScene assembles one scene reused by all three shadow maps.
//
// Layout (top view, x/z in receiver units):
//
//	ss_down : single-sided sheet, front face points down toward the receiver.
//	ss_up   : single-sided sheet, back face points down toward the receiver.
//	ds_sheet: double-sided sheet, both faces are shading-relevant.
//	cube    : closed, translucent solid; entry and exit merge into one event.
func BuildScene() *Scene {
	ssDown := &Material{Name: "ss_down", DoubleSided: false, Transmittance: 0}
	ssUp := &Material{Name: "ss_up", DoubleSided: false, Transmittance: 0}
	dsSheet := &Material{Name: "ds_sheet", DoubleSided: true, Transmittance: 0}
	cube := &Material{Name: "cube", DoubleSided: true, Transmittance: 0.6, Closure: true}

	scene := &Scene{Materials: []*Material{ssDown, ssUp, dsSheet, cube}}

	// Horizontal sheets. NewQuad's winding gives normal -Cross(u, v):
	// u=+x, v=+z -> normal -y (front face down); v=-z gives normal +y.
	downU, downV := Vec3{2.4, 0, 0}, Vec3{0, 0, 2.4}
	upU, upV := Vec3{2.4, 0, 0}, Vec3{0, 0, -2.4}
	// The double-sided sheet faces away from the receiver so that back-face
	// culling lets light through while the double-sided flags still block it.
	sideU, sideV := Vec3{2.4, 0, 0}, Vec3{0, 0, -2.4}

	scene.Add(NewQuad(ssDown, Vec3{-4.5, sheetY, -1.5}, downU, downV)...)
	scene.Add(NewQuad(ssUp, Vec3{-4.5, sheetY, 2.5}, upU, upV)...)
	scene.Add(NewQuad(dsSheet, Vec3{0, sheetY, -1.5}, sideU, sideV)...)
	scene.Add(NewCube(cube, Vec3{4.5, cubeY, 0.5}, cubeHalf)...)

	return scene
}

// LightDir is the shadow-ray direction: from a receiver point toward the sun.
// Rays travel upward, so a downward-facing front side is hit front-on.
func LightDir() Vec3 {
	return Vec3{0.35, 1, 0.2}.Normalize()
}

// ShadowRay builds an offset ray from a receiver point toward the light.
func ShadowRay(point Vec3) Ray {
	dir := LightDir()
	origin := point.Add(Vec3{0, 1e-6, 0})
	return Ray{Origin: origin, Direction: dir}
}

// PointUnderObject returns a receiver point whose shadow ray reaches the
// object point (at height y) through its center, i.e. the shadow's footprint.
func PointUnderObject(objectPoint Vec3) Vec3 {
	dir := LightDir()
	distance := (objectPoint.Y - ReceiverY) / dir.Y
	return objectPoint.Sub(dir.Scale(distance))
}
