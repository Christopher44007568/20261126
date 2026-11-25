package aliaslab

import "math"

func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		return 0
	}
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func skyFill(width, height int) *Image {
	img := NewImage(width, height)
	top := RGB{12, 29, 73}
	mid := RGB{58, 111, 177}
	horizon := RGB{231, 164, 105}
	for y := 0; y < height; y++ {
		v := float64(y) / float64(height-1)
		for x := 0; x < width; x++ {
			u := float64(x) / float64(width-1)
			var c RGB
			if v < 0.60 {
				c = mixRGB(top, mid, v/0.60)
			} else {
				c = mixRGB(mid, horizon, smoothstep(0.60, 1, v))
			}
			cool := math.Pow(1-u, 2) * (1 - v) * 0.20
			c = mixRGB(c, RGB{17, 28, 69}, cool)
			img.Set(x, y, c)
		}
	}
	return img
}

func sunGlow(inputs map[ResourceID]*Image) *Image {
	src := inputs["sky"]
	out := src.Clone()
	width, height := src.Width, src.Height
	aspect := float64(width) / float64(height)
	const cx, cy = 0.755, 0.255
	for y := 0; y < height; y++ {
		v := float64(y) / float64(height-1)
		for x := 0; x < width; x++ {
			u := float64(x) / float64(width-1)
			dist := math.Hypot((u-cx)*aspect, v-cy)
			c := src.At(x, y)
			if dist < 0.052 {
				core := mixRGB(RGB{255, 228, 151}, RGB{255, 251, 224}, 1-dist/0.052)
				c = mixRGB(c, core, 0.98)
			} else {
				glow := math.Exp(-dist / 0.085)
				halo := math.Exp(-dist / 0.31)
				c = addRGB(c, scaleRGB(RGB{176, 123, 51}, glow*0.62+halo*0.21))
			}
			out.Set(x, y, c)
		}
	}
	return out
}

type cloudPuff struct {
	cx, cy, rx, ry float64
}

func cloudMask(u, v float64) float64 {
	puffs := [...]cloudPuff{
		{0.12, 0.22, 0.15, 0.032},
		{0.29, 0.205, 0.19, 0.038},
		{0.57, 0.17, 0.17, 0.030},
		{0.80, 0.30, 0.21, 0.042},
		{0.23, 0.415, 0.23, 0.038},
		{0.61, 0.455, 0.27, 0.042},
		{0.90, 0.535, 0.15, 0.030},
	}
	best := 0.0
	for _, p := range puffs {
		dx := (u - p.cx) / p.rx
		dy := (v - p.cy) / p.ry
		d := math.Hypot(dx, dy)
		m := 1 - smoothstep(0.68, 1.06, d)
		if m > best {
			best = m
		}
	}
	if best <= 0 {
		return 0
	}
	detail := 0.86 + 0.14*math.Sin((u*19.0+v*11.0)*math.Pi)
	return clamp01(best * detail)
}

func cloudComposite(inputs map[ResourceID]*Image) *Image {
	sky := inputs["sky"]
	out := inputs["sun"].Clone()
	width, height := sky.Width, sky.Height
	aspect := float64(width) / float64(height)
	for y := 0; y < height; y++ {
		v := float64(y) / float64(height-1)
		for x := 0; x < width; x++ {
			u := float64(x) / float64(width-1)
			mask := cloudMask(u, v)
			if mask <= 0 {
				continue
			}
			cloud := mixRGB(RGB{236, 243, 251}, sky.At(x, y), 0.42)
			dist := math.Hypot((u-0.755)*aspect, v-0.255)
			sunWarm := math.Exp(-dist / 0.37)
			cloud = mixRGB(cloud, RGB{255, 213, 164}, sunWarm*0.55)
			cloud = mixRGB(cloud, RGB{82, 102, 137}, (1-v)*0.10)
			out.Set(x, y, mixRGB(out.At(x, y), cloud, mask*0.86))
		}
	}
	return out
}

func atmosphereLight(inputs map[ResourceID]*Image) *Image {
	src := inputs["clouds"]
	out := NewImage(src.Width, src.Height)
	aspect := float64(src.Width) / float64(src.Height)
	for y := 0; y < src.Height; y++ {
		v := float64(y) / float64(src.Height-1)
		for x := 0; x < src.Width; x++ {
			u := float64(x) / float64(src.Width-1)
			c := src.At(x, y)
			warm := math.Exp(-math.Hypot((u-0.78)*aspect, v-0.30) / 0.70)
			c = addRGB(c, scaleRGB(RGB{24, 11, 2}, warm))
			fog := smoothstep(0.62, 1.0, v) * 0.15
			c = mixRGB(c, RGB{128, 151, 183}, fog)
			ray := 1.0 + 0.10*(u-0.5) + 0.07*(0.5-v)
			out.Set(x, y, scaleRGB(c, ray))
		}
	}
	return out
}

func colorGrade(inputs map[ResourceID]*Image) *Image {
	src := inputs["lit"]
	out := NewImage(src.Width, src.Height)
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			c := src.At(x, y)
			lum := 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
			grade := func(ch uint8, tint float64) uint8 {
				v := float64(ch)
				v = lum + (v-lum)*1.14
				v = (v-128)*1.08 + 128 + tint
				return clamp8(int(v + 0.5))
			}
			shadow := (1 - clamp01(lum/128)) * 7
			highlight := clamp01((lum-128)/127) * 3
			out.Set(x, y, RGB{
				R: grade(c.R, highlight),
				G: grade(c.G, -shadow*0.15),
				B: grade(c.B, shadow),
			})
		}
	}
	return out
}

func vignette(inputs map[ResourceID]*Image) *Image {
	src := inputs["graded"]
	out := NewImage(src.Width, src.Height)
	for y := 0; y < src.Height; y++ {
		v := float64(y) / float64(src.Height-1)
		for x := 0; x < src.Width; x++ {
			u := float64(x) / float64(src.Width-1)
			d := math.Hypot((u-0.5)*1.55, (v-0.5)*1.12)
			factor := 1 - 0.36*smoothstep(0.34, 0.94, d)
			out.Set(x, y, scaleRGB(src.At(x, y), factor))
		}
	}
	return out
}
