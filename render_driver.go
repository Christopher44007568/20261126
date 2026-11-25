package main

import (
	"fmt"
	"path/filepath"
)

type FrameResult struct {
	Params FrameParams
	Stats  Stats
}

// RenderFrame produces the beauty, shadow-visibility and color-transmission
// images. All three share the same any-hit occlusion query per surface point.
func RenderFrame(scene *Scene, w, h, aa int, st *Stats) (*Image, *Image, *Image) {
	beauty := NewImage(w, h)
	shadow := NewImage(w, h)
	trans := NewImage(w, h)
	cam := NewCamera(Vec3{0, 1.5, 5.6}, Vec3{0, 1.0, -2.0}, 55, float64(w)/float64(h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var col, vis, tint Vec3
			for sy := 0; sy < aa; sy++ {
				for sx := 0; sx < aa; sx++ {
					u := (float64(x)+(float64(sx)+0.5)/float64(aa))/float64(w)*2 - 1
					v := 1 - (float64(y)+(float64(sy)+0.5)/float64(aa))/float64(h)*2
					ray := cam.Ray(u, v)

					st.CameraRays++
					c, throughGlass, occ := scene.trace(ray, 0, st)
					if throughGlass {
						st.CameraThroughGlass++
					}
					col = col.Add(c)
					if occ == nil {
						vis = vis.Add(Vec3{1, 1, 1})
						tint = tint.Add(Vec3{1, 1, 1})
						continue
					}
					if occ.Opaque {
						continue // black in both diagnostic maps
					}
					l := Luminance(occ.Transmitted)
					vis = vis.Add(Vec3{l, l, l})
					tint = tint.Add(Clamp01(occ.Transmitted))
				}
			}
			n := float64(aa * aa)
			beauty.Set(x, y, col.Scale(1/n))
			shadow.Set(x, y, vis.Scale(1/n))
			trans.Set(x, y, tint.Scale(1/n))
		}
	}
	return beauty, shadow, trans
}

func (r FrameResult) CSVRow() string {
	p := r.Params
	s := r.Stats
	return fmt.Sprintf(
		"%d,%.4f,%.4f,%.3f,%.3f,%.3f,%t,%d,%d,%d,%d,%d,%d,%d,%d\n",
		p.Index, p.SlabThick, p.AbsorbScale,
		p.Tint.X, p.Tint.Y, p.Tint.Z, p.Clear,
		s.ShadowRays, s.OpaqueOccluded, s.GlassPenetrated,
		s.PenetratingRays, s.ClearRays, s.GlassCrossings,
		s.CameraRays, s.CameraThroughGlass,
	)
}

func CSVHeader() string {
	return "frame,slab_thickness,absorb_scale,tint_r,tint_g,tint_b,clear," +
		"shadow_rays,opaque_occluded,glass_penetrated,penetrating_rays," +
		"clear_rays,glass_crossings,camera_rays,camera_through_glass\n"
}

func framePath(outDir string, frame int, kind string) string {
	return filepath.Join(outDir, fmt.Sprintf("frame%02d_%s.ppm", frame, kind))
}
