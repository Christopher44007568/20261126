package internal

import "math"

const TextureSize = 512

// NewCheckerTexture builds a high-contrast checker with fine inner detail so
// aliasing / mip transitions are easy to inspect visually.
func NewCheckerTexture() *Texture {
	base := make([]RGBA, TextureSize*TextureSize)
	const cells = 16
	for y := 0; y < TextureSize; y++ {
		for x := 0; x < TextureSize; x++ {
			cx := x * cells / TextureSize
			cy := y * cells / TextureSize
			light := (cx+cy)%2 == 0
			// Subtle inset stripe gives the coarse mips something to average.
			inset := (x%(TextureSize/cells) < 2) || (y%(TextureSize/cells) < 2)
			var c RGBA
			switch {
			case light && inset:
				c = RGBA{0.95, 0.9, 0.8, 1}
			case light:
				c = RGBA{0.82, 0.78, 0.7, 1}
			case inset:
				c = RGBA{0.18, 0.2, 0.24, 1}
			default:
				c = RGBA{0.08, 0.09, 0.12, 1}
			}
			base[y*TextureSize+x] = c
		}
	}
	return NewTexture(base, TextureSize)
}

// NewSkyTexture builds an equirectangular sky with a gradient and a soft sun.
func NewSkyTexture() *Texture {
	base := make([]RGBA, TextureSize*TextureSize)
	sunU, sunV := 0.68, 0.32
	for y := 0; y < TextureSize; y++ {
		v := float64(y) / TextureSize
		for x := 0; x < TextureSize; x++ {
			u := float64(x) / TextureSize
			t := Clamp(1-v, 0, 1)
			horizon := RGBA{0.75, 0.8, 0.9, 1}
			zenith := RGBA{0.16, 0.32, 0.62, 1}
			c := RGBA{
				Lerp(horizon.R, zenith.R, t),
				Lerp(horizon.G, zenith.G, t),
				Lerp(horizon.B, zenith.B, t),
				1,
			}
			d := math.Hypot(angleDelta(u, sunU), v-sunV)
			sun := Clamp(1-d*7, 0, 1)
			sun = sun * sun
			c.R = Lerp(c.R, 1.0, sun*0.9)
			c.G = Lerp(c.G, 0.95, sun*0.9)
			c.B = Lerp(c.B, 0.85, sun*0.9)
			base[y*TextureSize+x] = c
		}
	}
	return NewTexture(base, TextureSize)
}

func angleDelta(a, b float64) float64 {
	d := a - b
	if d > 0.5 {
		d -= 1
	}
	if d < -0.5 {
		d += 1
	}
	return d
}

// GroundUV maps a world point on the y=height plane to tiling checker UVs.
// One UV tile spans TileWorld units.
const TileWorld = 2.0

func GroundUV(p Vec3) (u, v float64) {
	return p.X / TileWorld, p.Z / TileWorld
}

// GroundDerivs converts the world-space surface footprint into UV derivatives.
func GroundDerivs(dpx, dpy Vec3) (dudx, dvdx, dudy, dvdy float64) {
	inv := 1 / TileWorld
	return dpx.X * inv, dpx.Z * inv, dpy.X * inv, dpy.Z * inv
}

// SkyUV maps a direction to equirectangular coordinates.
func SkyUV(dir Vec3) (u, v float64) {
	d := dir.Normalize()
	u = 0.5 + math.Atan2(d.Z, d.X)/(2*math.Pi)
	v = 0.5 - math.Asin(Clamp(d.Y, -1, 1))/math.Pi
	return u, v
}

// SkyDerivs differentiates the equirectangular projection of the direction.
// The ray's direction differentials are angular offsets, giving an angular
// footprint on the sky dome.
func SkyDerivs(dir, ddirx, ddiry Vec3) (dudx, dvdx, dudy, dvdy float64) {
	d := dir.Normalize()
	one := func(dd Vec3) (du, dv float64) {
		xy := d.X*d.X + d.Z*d.Z
		if xy < 1e-10 {
			return 0, 0
		}
		du = (d.X*dd.Z - d.Z*dd.X) / (2 * math.Pi * xy)
		dv = -dd.Y / (math.Pi * math.Sqrt(math.Max(0, 1-d.Y*d.Y)))
		return du, dv
	}
	dudx, dvdx = one(ddirx)
	dudy, dvdy = one(ddiry)
	return
}
