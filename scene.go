package hdrlab

import "math"

const (
	SceneW = 384
	SceneH = 216
	// Margin is the alignment search radius; outputs are cropped by it.
	Margin = 5
	OutW   = SceneW - 2*Margin
	OutH   = SceneH - 2*Margin
	CarY   = 152
	RoadY  = 178
)

// Rect is a half-open integer rectangle.
type Rect struct {
	MinX, MinY, MaxX, MaxY int
}

// Billboard is the highlight-detail test target, in scene coordinates.
var Billboard = Rect{MinX: 220, MinY: 26, MaxX: 354, MaxY: 104}

// FrameSpec describes one bracketed exposure.
type FrameSpec struct {
	Name   string
	EV     float64
	DX, DY int // camera shift relative to the reference frame
	CarX   int // moving-object center x in this frame
}

func (s FrameSpec) Exposure() float64 { return math.Pow(2, s.EV) }

// BracketSpecs returns the synthetic bracket set: under/normal/over with
// simulated handheld shifts and a car that moves between frames.
func BracketSpecs() []FrameSpec {
	return []FrameSpec{
		{Name: "under", EV: -2, DX: 3, DY: -2, CarX: 110},
		{Name: "normal", EV: 0, DX: 0, DY: 0, CarX: 170},
		{Name: "over", EV: 2, DX: -3, DY: 2, CarX: 230},
	}
}

// rng is a tiny deterministic xorshift64* PRNG (stdlib only).
type rng struct{ s uint64 }

func (r *rng) next() uint64 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 7
	r.s ^= r.s << 17
	return r.s
}

func (r *rng) float() float64 { return float64(r.next()>>11) / float64(uint64(1)<<53) }

func (r *rng) intn(n int) int { return int(r.next() % uint64(n)) }

// gauss approximates a standard normal via 4 uniform samples.
func (r *rng) gauss() float64 {
	return (r.float() + r.float() + r.float() + r.float() - 2) * 1.7320508
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// RenderScene builds the static linear-radiance scene (values may exceed 1):
// dusk sky with sun glow, a bright billboard with internal detail, a city
// skyline with lit windows, and a road. The moving car is drawn per frame.
func RenderScene() *Img {
	im := NewImg(SceneW, SceneH)
	r := &rng{s: 0x9E3779B97F4A7C15}

	for y := 0; y < RoadY; y++ {
		t := math.Pow(float64(y)/RoadY, 1.25)
		for x := 0; x < SceneW; x++ {
			rr := lerp(0.05, 0.78, t)
			gg := lerp(0.07, 0.45, t)
			bb := lerp(0.18, 0.26, t)
			hf := 0.82 + 0.35*float64(x)/SceneW
			dx := float64(x-295) / 55
			dy := float64(y-12) / 34
			glow := 1.35 * math.Exp(-(dx*dx + dy*dy))
			im.Set(x, y, rr*hf+glow*1.15, gg*hf+glow*0.78, bb*hf+glow*0.45)
		}
	}

	for y := RoadY; y < SceneH; y++ {
		for x := 0; x < SceneW; x++ {
			v := 0.085 + 0.01*math.Sin(float64(x)*0.3)
			im.Set(x, y, v, v, v*1.05)
		}
	}
	im.FillRect(0, RoadY, SceneW, RoadY+5, 0.16, 0.16, 0.17)
	for x := 0; x < SceneW; x++ {
		if x%36 < 18 {
			for y := 194; y < 199; y++ {
				im.Set(x, y, 0.5, 0.5, 0.47)
			}
		}
	}

	for x := 2; x < SceneW-10; {
		bw := 22 + r.intn(20)
		bh := 42 + r.intn(62)
		top := RoadY - bh
		shade := 0.75 + 0.5*r.float()
		br, bg, bb := 0.075*shade, 0.09*shade, 0.14*shade
		x1 := x + bw
		if x1 > SceneW {
			x1 = SceneW
		}
		im.FillRect(x, top, x1, RoadY, br, bg, bb)
		for wy := top + 6; wy < RoadY-6; wy += 9 {
			for wx := x + 4; wx < x1-5; wx += 8 {
				if r.float() < 0.32 {
					lv := 0.5 + 0.5*r.float()
					im.FillRect(wx, wy, wx+4, wy+4, 0.85*lv, 0.62*lv, 0.28*lv)
				} else {
					im.FillRect(wx, wy, wx+4, wy+4, 0.03, 0.045, 0.08)
				}
			}
		}
		x += bw + 4 + r.intn(8)
	}

	drawBillboard(im)
	return im
}

// drawBillboard renders the bright panel with internal highlight detail:
// horizontal dark slats (period 7 so the +-5px alignment search sees no
// alias), a hot inner square, and a dark vertical bar.
func drawBillboard(im *Img) {
	im.FillRect(250, 104, 256, 150, 0.09, 0.09, 0.10)
	im.FillRect(320, 104, 326, 150, 0.09, 0.09, 0.10)
	b := Billboard
	im.FillRect(b.MinX, b.MinY, b.MaxX, b.MaxY, 0.05, 0.05, 0.06)
	for y := b.MinY + 4; y < b.MaxY-4; y++ {
		v := 1.5 + 0.2*(1-float64(y-b.MinY-4)/float64(b.MaxY-b.MinY-8))
		if y%7 < 2 {
			v = 0.30 // dark slat
		}
		for x := b.MinX + 4; x < b.MaxX-4; x++ {
			im.Set(x, y, v, v*0.96, v*0.88)
		}
	}
	im.FillRect(268, 44, 298, 74, 3.4, 3.3, 3.1)
	im.FillRect(312, 34, 328, 94, 0.25, 0.25, 0.28)
}

// drawCar paints the moving object (red car) into a radiance image.
func drawCar(im *Img, cx, cy int) {
	for y := cy - 14; y <= cy+14; y++ {
		for x := cx - 25; x <= cx+25; x++ {
			if x < 0 || y < 0 || x >= im.W || y >= im.H {
				continue
			}
			dxb := float64(x-cx) / 22
			dyb := float64(y-(cy+2)) / 10
			inBody := dxb*dxb+dyb*dyb <= 1
			dxc := float64(x-cx+2) / 13
			dyc := float64(y-(cy-5)) / 7
			inCabin := dxc*dxc+dyc*dyc <= 1 && y < cy+1
			if inBody {
				im.Set(x, y, 0.55, 0.07, 0.05)
			}
			if inCabin {
				im.Set(x, y, 0.12, 0.16, 0.24)
			}
		}
	}
	im.FillEllipse(cx-12, cy+9, 4, 4, 0.02, 0.02, 0.03)
	im.FillEllipse(cx+12, cy+9, 4, 4, 0.02, 0.02, 0.03)
}

// RenderFrame renders one bracket: shifted scene + car, exposure, clamp,
// gamma encode (like a camera JPEG), plus a little sensor noise.
func RenderFrame(scene *Img, spec FrameSpec, seed uint64) *Img {
	rad := NewImg(SceneW, SceneH)
	for y := 0; y < SceneH; y++ {
		for x := 0; x < SceneW; x++ {
			r, g, b := scene.GetClamped(x+spec.DX, y+spec.DY)
			rad.Set(x, y, r, g, b)
		}
	}
	drawCar(rad, spec.CarX, CarY)

	t := spec.Exposure()
	rn := &rng{s: seed}
	out := NewImg(SceneW, SceneH)
	for i := 0; i < len(out.Pix); i++ {
		v := Clamp01(rad.Pix[i] * t)
		v = math.Pow(v, 1.0/2.2)
		v += rn.gauss() * 0.008
		out.Pix[i] = Clamp01(v)
	}
	return out
}
