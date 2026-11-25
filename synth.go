package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
)

type CheckerSpec struct {
	CornersX int
	CornersY int
	Square   float64
}

type RenderOptions struct {
	Width  int
	Height int
	SSAA   int
	Noise  float64
	Seed   int64
}

func checkerColor(w Vec3, square float64) color.RGBA {
	ix := int(math.Floor(w.X / square))
	iy := int(math.Floor(w.Y / square))
	parity := (ix+iy)%2 + 2
	if parity%2 == 0 {
		return color.RGBA{R: 242, G: 242, B: 238, A: 255}
	}
	return color.RGBA{R: 24, G: 27, B: 31, A: 255}
}

func planeIntersection(c Camera, pose Pose, px, py float64) (Vec3, bool) {
	ray := c.PixelToUndistortedRay(Vec2{X: px, Y: py})
	nx := pose.R[0][2]
	ny := pose.R[1][2]
	nz := pose.R[2][2]
	den := nx*ray.X + ny*ray.Y + nz
	if math.Abs(den) < 1e-9 {
		return Vec3{}, false
	}
	lambda := (nx*pose.T.X + ny*pose.T.Y + nz*pose.T.Z) / den
	if lambda <= 0.05 {
		return Vec3{}, false
	}
	xc := Vec3{X: lambda * ray.X, Y: lambda * ray.Y, Z: lambda}
	d := Vec3{X: xc.X - pose.T.X, Y: xc.Y - pose.T.Y, Z: xc.Z - pose.T.Z}
	return Vec3{
		X: pose.R[0][0]*d.X + pose.R[1][0]*d.Y + pose.R[2][0]*d.Z,
		Y: pose.R[0][1]*d.X + pose.R[1][1]*d.Y + pose.R[2][1]*d.Z,
		Z: pose.R[0][2]*d.X + pose.R[1][2]*d.Y + pose.R[2][2]*d.Z,
	}, true
}

func RenderCheckerboard(cam Camera, pose Pose, spec CheckerSpec, opt RenderOptions) *image.RGBA {
	if opt.Width == 0 {
		opt.Width = cam.Width
	}
	if opt.Height == 0 {
		opt.Height = cam.Height
	}
	if opt.SSAA < 1 {
		opt.SSAA = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, opt.Width, opt.Height))
	rng := rand.New(rand.NewSource(opt.Seed))
	boardMinX, boardMinY := -spec.Square, -spec.Square
	boardMaxX := float64(spec.CornersX)*spec.Square + spec.Square
	boardMaxY := float64(spec.CornersY)*spec.Square + spec.Square
	for y := 0; y < opt.Height; y++ {
		for x := 0; x < opt.Width; x++ {
			var r, g, b, count float64
			for sy := 0; sy < opt.SSAA; sy++ {
				for sx := 0; sx < opt.SSAA; sx++ {
					px := float64(x) + (float64(sx)+0.5)/float64(opt.SSAA)
					py := float64(y) + (float64(sy)+0.5)/float64(opt.SSAA)
					col := color.RGBA{R: 178, G: 184, B: 190, A: 255}
					if w, ok := planeIntersection(cam, pose, px, py); ok &&
						w.X >= boardMinX && w.X <= boardMaxX &&
						w.Y >= boardMinY && w.Y <= boardMaxY {
						col = checkerColor(w, spec.Square)
						shade := 1 - 0.08*((px-cam.Cx)*(px-cam.Cx)+(py-cam.Cy)*(py-cam.Cy))/
							(cam.Fx*cam.Fx+cam.Fy*cam.Fy)
						if shade < 0.82 {
							shade = 0.82
						}
						col.R = uint8(float64(col.R) * shade)
						col.G = uint8(float64(col.G) * shade)
						col.B = uint8(float64(col.B) * shade)
					}
					r += float64(col.R)
					g += float64(col.G)
					b += float64(col.B)
					count++
				}
			}
			nr := opt.Noise * rng.NormFloat64()
			ng := opt.Noise * rng.NormFloat64()
			nb := opt.Noise * rng.NormFloat64()
			img.SetRGBA(x, y, color.RGBA{
				R: clampUint8(r/count + nr),
				G: clampUint8(g/count + ng),
				B: clampUint8(b/count + nb),
				A: 255,
			})
		}
	}
	return img
}

func clampUint8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func ToGray(img *image.RGBA) []float64 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(x+b.Min.X, y+b.Min.Y)
			r, g, bl := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
			out[y*w+x] = 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)
		}
	}
	return out
}
