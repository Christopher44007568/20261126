package main

import "math"

type Camera struct {
	Width      int
	Height     int
	ViewHeight float64
	EyeZ       float64
}

func DefaultCamera(width, height int) Camera {
	if width <= 0 {
		width = 480
	}
	if height <= 0 {
		height = 360
	}
	return Camera{Width: width, Height: height, ViewHeight: 3.75, EyeZ: 6}
}

func (c Camera) Ray(x, y int) Ray {
	aspect := float64(c.Width) / float64(c.Height)
	viewWidth := c.ViewHeight * aspect
	px := (float64(x)+0.5)/float64(c.Width)*viewWidth - viewWidth/2
	py := c.ViewHeight/2 - (float64(y)+0.5)/float64(c.Height)*c.ViewHeight
	return Ray{Origin: Vec3{X: px, Y: py, Z: c.EyeZ}, Dir: Vec3{X: 0, Y: 0, Z: -1}}
}

type RenderResult struct {
	Image *Image
	Mask  []bool
	Hits  int
}

func RenderBVH(bvh *BVH, materials []Material, camera Camera) *RenderResult {
	result := &RenderResult{Image: NewImage(camera.Width, camera.Height), Mask: make([]bool, camera.Width*camera.Height)}
	for y := 0; y < camera.Height; y++ {
		for x := 0; x < camera.Width; x++ {
			i := y*camera.Width + x
			ray := camera.Ray(x, y)
			hit, ok := bvh.Intersect(ray)
			if !ok {
				result.Image.Pixels[i] = backgroundPixel(y, camera.Height)
				continue
			}
			tri := bvh.Tris[hit.TriIndex]
			result.Mask[i] = true
			result.Hits++
			result.Image.Pixels[i] = shadeTriangle(tri, ray, hit, materials)
		}
	}
	return result
}

func RenderQBVH(bvh *QBVH, camera Camera) *RenderResult {
	result := &RenderResult{Image: NewImage(camera.Width, camera.Height), Mask: make([]bool, camera.Width*camera.Height)}
	materials := bvh.Mesh.Materials
	for y := 0; y < camera.Height; y++ {
		for x := 0; x < camera.Width; x++ {
			i := y*camera.Width + x
			ray := camera.Ray(x, y)
			hit, ok := bvh.Intersect(ray)
			if !ok {
				result.Image.Pixels[i] = backgroundPixel(y, camera.Height)
				continue
			}
			tri := bvh.Mesh.Triangle(hit.TriIndex)
			result.Mask[i] = true
			result.Hits++
			result.Image.Pixels[i] = shadeTriangle(tri, ray, hit, materials)
		}
	}
	return result
}

func backgroundPixel(y, height int) RGB {
	t := float64(y) / float64(maxInt(height-1, 1))
	return mixRGB(colorBackgroundTop, colorBackgroundBottom, t)
}

func shadeTriangle(tri Triangle, ray Ray, hit Hit, materials []Material) RGB {
	mat := Material{Base: RGB{220, 180, 90}, Sheen: 0.1}
	if tri.Material >= 0 && tri.Material < len(materials) {
		mat = materials[tri.Material]
	}
	n := triangleNormal(tri)
	point := Vec3{
		X: ray.Origin.X + ray.Dir.X*hit.T,
		Y: ray.Origin.Y + ray.Dir.Y*hit.T,
		Z: ray.Origin.Z + ray.Dir.Z*hit.T,
	}
	view := vNorm(vSub(ray.Origin, point))
	if vDot(n, view) < 0 {
		n = vScale(n, -1)
	}

	light := vNorm(Vec3{X: -0.38, Y: -0.48, Z: 0.79})
	diffuse := math.Max(0, vDot(n, light))
	intensity := 0.34 + 0.72*diffuse

	half := vNorm(vAdd(light, view))
	specular := math.Pow(math.Max(0, vDot(n, half)), 34) * 185 * mat.Sheen

	return RGB{
		R: clampByte(float64(mat.Base.R)*intensity + specular),
		G: clampByte(float64(mat.Base.G)*intensity + specular),
		B: clampByte(float64(mat.Base.B)*intensity + specular),
	}
}

func mixRGB(a, b RGB, t float64) RGB {
	return RGB{
		R: clampByte(float64(a.R)*(1-t) + float64(b.R)*t),
		G: clampByte(float64(a.G)*(1-t) + float64(b.G)*t),
		B: clampByte(float64(a.B)*(1-t) + float64(b.B)*t),
	}
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(math.Round(v))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
