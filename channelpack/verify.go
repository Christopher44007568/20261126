package main

import "strings"

type LayoutReport struct {
	Name         string
	Stats        []ChannelStat
	AlbedoExact  bool
	NormalExact  bool
	MaxAngle     float64
	RoughExact   bool
	AOExact      bool
	Independence bool
}

func verifyLayout(l *Layout, src *Maps) LayoutReport {
	texs := l.Encode(src)
	got := l.Decode(texs, src.W, src.H)
	stats := compareMaps(src, got)
	return LayoutReport{
		Name:         l.Name,
		Stats:        stats,
		AlbedoExact:  exact(stats, "ALBEDO[0]", "ALBEDO[1]", "ALBEDO[2]"),
		NormalExact:  exact(stats, "NORMAL[0]", "NORMAL[1]", "NORMAL[2]"),
		MaxAngle:     maxNormalAngleDeg(src, got),
		RoughExact:   exact(stats, "ROUGH"),
		AOExact:      exact(stats, "AO"),
		Independence: scalarIndependence(l, src),
	}
}

// scalarIndependence proves scalar packing/quantization cannot alter normal
// or albedo decoding. It perturbs only roughness and occlusion, re-encodes,
// and requires the decoded albedo and all three normal bytes to be identical.
func scalarIndependence(l *Layout, src *Maps) bool {
	base := l.Decode(l.Encode(src), src.W, src.H)
	perturbed := *src
	perturbed.Roughness = append([]byte(nil), src.Roughness...)
	perturbed.Occlusion = append([]byte(nil), src.Occlusion...)
	for i := range perturbed.Roughness {
		perturbed.Roughness[i] = byte((int(src.Roughness[i]) + 73) % 256)
		perturbed.Occlusion[i] = byte((int(src.Occlusion[i]) + 151) % 256)
	}
	got := l.Decode(l.Encode(&perturbed), src.W, src.H)
	for i := range base.Albedo {
		if base.Albedo[i] != got.Albedo[i] || base.Normal[i] != got.Normal[i] {
			return false
		}
	}
	return true
}

func formatReport(reports []LayoutReport, misalignedDiff int, total int, w, h, sphereSize int) string {
	var b strings.Builder
	b.WriteString("CHANNEL PACKING ROUND-TRIP VERIFICATION\n")
	b.WriteString("source maps: " + itoa(w) + "x" + itoa(h) + ", sphere: " + itoa(sphereSize) + "px\n\n")
	for _, r := range reports {
		b.WriteString(strings.ToUpper(r.Name) + " ")
		b.WriteString(" ALBEDO_EXACT=" + boolStr(r.AlbedoExact))
		b.WriteString("  NORMAL_EXACT=" + boolStr(r.NormalExact))
		b.WriteString("  MAX_ANGLE=" + ftoa4(r.MaxAngle) + "DEG")
		b.WriteString(" ROUGH_EXACT=" + boolStr(r.RoughExact))
		b.WriteString("  AO_EXACT=" + boolStr(r.AOExact))
		b.WriteString("  INDEPENDENCE=" + boolStr(r.Independence) + "\n")
		for _, st := range r.Stats {
			b.WriteString("  " + st.Name + " max_abs=" + itoa(st.MaxAbs) +
				" diffs=" + itoa(st.DiffCount) +
				" rmse=" + ftoa4(st.rmse(w*h)) + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("MISALIGNED DECODE: " + itoa(misalignedDiff) + "/" + itoa(total) +
		" albedo bytes differ (expected, demonstrates slot-binding bug)\n")
	return b.String()
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func ftoa4(v float64) string {
	scaled := v * 10000
	whole := int(scaled)
	if scaled-float64(whole) >= 0.5 {
		whole++
	}
	intPart := whole / 10000
	frac := whole % 10000
	digits := itoa(frac)
	for len(digits) < 4 {
		digits = "0" + digits
	}
	return itoa(intPart) + "." + digits
}

func countAlbedoDifferences(ref, got *Maps) int {
	diff := 0
	for i := range ref.Albedo {
		if ref.Albedo[i] != got.Albedo[i] {
			diff++
		}
	}
	return diff
}
