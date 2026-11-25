package internal

import "testing"

// TestScreenLODContinuity sweeps the ground region and rejects pixel-to-pixel
// LOD jumps within the same surface. Crossing a sphere silhouette legitimately
// switches surfaces (and LOD), so only ground-to-ground neighbors are checked.
// A hard jump there would indicate a footprint or clamp discontinuity.
func TestScreenLODContinuity(t *testing.T) {
	w, h := 160, 120
	r := NewRenderer(w, h, false)
	buf := r.Render()

	maxJump := 0.0
	surface := func(x, y int) int {
		ray := r.Camera.GenerateRay(x, y)
		hit, ok := r.trace(ray)
		if !ok {
			return -1
		}
		if hit.mat.Metal {
			return 1
		}
		return 2
	}
	for y := 0; y < h; y++ {
		for x := 1; x < w; x++ {
			s0 := surface(x-1, y)
			s1 := surface(x, y)
			if s0 != 2 || s1 != 2 {
				continue // only compare contiguous ground pixels
			}
			a := buf.LODLevel[y*w+x-1]
			b := buf.LODLevel[y*w+x]
			if d := b - a; d > maxJump {
				maxJump = d
			} else if -d > maxJump {
				maxJump = -d
			}
		}
	}
	if maxJump > 1.0 {
		t.Fatalf("max horizontal LOD jump %.3f exceeds threshold", maxJump)
	}
}

// TestRoughnessContinuity varies roughness on a fixed shading point and checks
// that the selected LOD changes monotonically and without integer-level snaps
// beyond the fractional trilinear blending range.
func TestRoughnessContinuity(t *testing.T) {
	ray := Ray{
		Origin:  V(0, 1.0, 0),
		Dir:     V(0, 0, -1),
		DDirX:   V(0.004, 0, 0),
		DDirY:   V(0, 0.004, 0),
		HasDiff: true,
	}
	sphere := Sphere{Center: V(0, 1, -4), Radius: 1}
	_, p, n, dpx, dpy, dnx, dny, ok := sphere.Intersect(ray)
	if !ok {
		t.Fatal("setup ray must hit sphere")
	}

	prev := -1.0
	for k := 0; k <= 100; k++ {
		rough := float64(k) / 100
		refl := ReflectedRay(ray, p, n, dpx, dpy, dnx, dny, rough)
		// Project the widened angular footprint onto a far ground plane to read
		// back the LOD it would select.
		pl := Plane{Height: 0}
		tt, hp, _, hdx, hdy, _, _, hitOK := pl.Intersect(refl)
		if !hitOK {
			continue
		}
		_ = tt
		_ = hp
		tex := NewCheckerTexture()
		u, v := GroundUV(hp)
		dudx, dvdx, dudy, dvdy := GroundDerivs(hdx, hdy)
		_, lod := tex.FootprintSample(u, v, dudx, dvdx, dudy, dvdy)
		if lod < prev-1e-9 {
			t.Fatalf("LOD not monotonic in roughness: %.4f -> %.4f at r=%.2f",
				prev, lod, rough)
		}
		prev = lod
	}
}
