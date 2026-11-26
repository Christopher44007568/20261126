package main

import "math"

type ChannelStat struct {
	Name      string
	MaxAbs    int
	DiffCount int
	SumSq     float64
}

func (s ChannelStat) rmse(n int) float64 {
	if n == 0 {
		return 0
	}
	return math.Sqrt(s.SumSq / float64(n))
}

func compareChannel(name string, ref, got []byte) ChannelStat {
	st := ChannelStat{Name: name}
	for i := range ref {
		d := int(ref[i]) - int(got[i])
		if d < 0 {
			d = -d
		}
		if d != 0 {
			st.DiffCount++
		}
		if d > st.MaxAbs {
			st.MaxAbs = d
		}
		st.SumSq += float64(d * d)
	}
	return st
}

func compareMaps(ref, got *Maps) []ChannelStat {
	stats := make([]ChannelStat, 0, 8)
	for c := 0; c < 3; c++ {
		names := []string{"ALBEDO[0]", "ALBEDO[1]", "ALBEDO[2]"}
		stats = append(stats, comparePlanar(names[c], ref.Albedo, got.Albedo, c, ref.W, ref.H))
	}
	for c := 0; c < 3; c++ {
		names := []string{"NORMAL[0]", "NORMAL[1]", "NORMAL[2]"}
		stats = append(stats, comparePlanar(names[c], ref.Normal, got.Normal, c, ref.W, ref.H))
	}
	stats = append(stats, compareChannel("ROUGH", ref.Roughness, got.Roughness))
	stats = append(stats, compareChannel("AO", ref.Occlusion, got.Occlusion))
	return stats
}

func comparePlanar(name string, ref, got []byte, channel, w, h int) ChannelStat {
	n := w * h
	a := make([]byte, n)
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		a[i] = ref[i*3+channel]
		b[i] = got[i*3+channel]
	}
	return compareChannel(name, a, b)
}

// maxNormalAngleDeg decodes both normal maps to unit vectors and returns the
// largest angular deviation in degrees. This is the metric that proves packed
// scalar data cannot rotate reconstructed normals.
func maxNormalAngleDeg(ref, got *Maps) float64 {
	maxCos := 1.0
	for i := 0; i < ref.W*ref.H; i++ {
		c := i * 3
		rx, ry, rz := normalVector(ref.Normal[c], ref.Normal[c+1], ref.Normal[c+2])
		gx, gy, gz := normalVector(got.Normal[c], got.Normal[c+1], got.Normal[c+2])
		dot := rx*gx + ry*gy + rz*gz
		if dot < maxCos {
			maxCos = dot
		}
	}
	if maxCos > 1 {
		maxCos = 1
	}
	if maxCos < -1 {
		maxCos = -1
	}
	return math.Acos(maxCos) * 180 / math.Pi
}

func normalVector(x, y, z byte) (float64, float64, float64) {
	vx := float64(x)/255*2 - 1
	vy := float64(y)/255*2 - 1
	vz := float64(z)/255*2 - 1
	return normalize3(vx, vy, vz)
}

func exact(stats []ChannelStat, names ...string) bool {
	for _, want := range names {
		for _, st := range stats {
			if st.Name == want && st.MaxAbs != 0 {
				return false
			}
		}
	}
	return true
}
