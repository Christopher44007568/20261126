package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"strings"
)

type Failure struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Value   float64 `json:"value"`
	Limit   float64 `json:"limit"`
}

type PatchMetrics struct {
	SourceRGB  RGB     `json:"sourceRgb"`
	TargetRGB  RGB     `json:"targetRgb"`
	SourceL    float64 `json:"sourceL"`
	TargetL    float64 `json:"targetL"`
	DeltaL     float64 `json:"deltaL"`
	DeltaE     float64 `json:"deltaE"`
	SourceClip float64 `json:"sourceClipRatio"`
	TargetClip float64 `json:"targetClipRatio"`
	NewClip    float64 `json:"newClipRatio"`
}

type PatchResult struct {
	ID              string       `json:"id"`
	Label           string       `json:"label"`
	Kind            string       `json:"kind"`
	Group           string       `json:"group,omitempty"`
	Order           int          `json:"order"`
	Rect            RectSpec     `json:"rect"`
	Metrics         PatchMetrics `json:"metrics"`
	DeltaELimit     float64      `json:"deltaELimit"`
	BrightnessLimit float64      `json:"brightnessLimit"`
	ClipLimit       float64      `json:"clipLimit"`
	Failures        []Failure    `json:"failures"`
	Pass            bool         `json:"pass"`
}

type MonotonicityViolation struct {
	Left      string  `json:"left"`
	Right     string  `json:"right"`
	LeftL     float64 `json:"leftL"`
	RightL    float64 `json:"rightL"`
	DeltaL    float64 `json:"deltaL"`
	Direction string  `json:"direction"`
	Limit     float64 `json:"limit"`
}

type GroupResult struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Direction  string                  `json:"direction"`
	PatchIDs   []string                `json:"patchIds"`
	Violations []MonotonicityViolation `json:"violations"`
	Pass       bool                    `json:"pass"`
}

type Summary struct {
	PatchCount    int      `json:"patchCount"`
	Passed        int      `json:"passed"`
	Failed        int      `json:"failed"`
	MeanDeltaE    float64  `json:"meanDeltaE"`
	MaxDeltaE     float64  `json:"maxDeltaE"`
	MeanDeltaL    float64  `json:"meanAbsDeltaL"`
	MaxDeltaL     float64  `json:"maxAbsDeltaL"`
	MeanNewClip   float64  `json:"meanNewClipRatio"`
	AveragePass   bool     `json:"averagePass"`
	LocalPass     bool     `json:"localPass"`
	Status        string   `json:"status"`
	LocalFailures []string `json:"localFailures"`
}

type Analysis struct {
	Source  image.Image
	Target  image.Image
	Patches []PatchResult
	Groups  []GroupResult
	Summary Summary
}

func analyze(manifest *Manifest, source, target image.Image) *Analysis {
	result := &Analysis{Source: source, Target: target}
	result.Patches = make([]PatchResult, 0, len(manifest.Patches))
	for _, spec := range manifest.Patches {
		pr := analyzePatch(spec, manifest.Thresholds, source, target)
		result.Patches = append(result.Patches, pr)
	}
	result.Groups = analyzeGroups(manifest, result.Patches)
	applyMonotonicityFailures(result.Patches, result.Groups, manifest)
	result.Summary = summarize(result.Patches, manifest.Thresholds)
	return result
}

func analyzePatch(spec PatchSpec, t Thresholds, source, target image.Image) PatchResult {
	metrics := measurePatch(source, target, spec.Rect.imageRect(), t.ClipHigh, t.ClipLow)
	cur := PatchResult{
		ID:       spec.ID,
		Label:    spec.Label,
		Kind:     spec.Kind,
		Group:    spec.Group,
		Order:    spec.Order,
		Rect:     spec.Rect,
		Metrics:  metrics,
		Failures: make([]Failure, 0),
	}
	cur.DeltaELimit, cur.BrightnessLimit, cur.ClipLimit = effectiveThresholds(spec.Kind, t)
	if metrics.DeltaE > cur.DeltaELimit {
		cur.Failures = append(cur.Failures, Failure{
			Code:    "delta_e",
			Message: fmt.Sprintf("CIEDE2000 %.2f exceeds %.2f", metrics.DeltaE, cur.DeltaELimit),
			Value:   metrics.DeltaE,
			Limit:   cur.DeltaELimit,
		})
	}
	if math.Abs(metrics.DeltaL) > cur.BrightnessLimit {
		cur.Failures = append(cur.Failures, Failure{
			Code:    "brightness",
			Message: fmt.Sprintf("|dL*| %.2f exceeds %.2f", math.Abs(metrics.DeltaL), cur.BrightnessLimit),
			Value:   math.Abs(metrics.DeltaL),
			Limit:   cur.BrightnessLimit,
		})
	}
	if metrics.NewClip > cur.ClipLimit {
		cur.Failures = append(cur.Failures, Failure{
			Code:    "gamut_clip",
			Message: fmt.Sprintf("new clipping %.2f%% exceeds %.2f%%", metrics.NewClip*100, cur.ClipLimit*100),
			Value:   metrics.NewClip,
			Limit:   cur.ClipLimit,
		})
	}
	cur.Pass = len(cur.Failures) == 0
	return cur
}

func effectiveThresholds(kind string, t Thresholds) (deltaE, brightness, clip float64) {
	deltaE = t.DeltaE
	brightness = t.Brightness
	clip = t.ClipRatio
	switch strings.ToLower(kind) {
	case "highlight":
		if t.HighlightDeltaE > 0 {
			deltaE = math.Min(deltaE, t.HighlightDeltaE)
		}
		if t.HighlightBrightness > 0 {
			brightness = math.Min(brightness, t.HighlightBrightness)
		}
		if t.HighlightClipRatio > 0 {
			clip = math.Min(clip, t.HighlightClipRatio)
		}
	case "skin":
		if t.SkinDeltaE > 0 {
			deltaE = math.Min(deltaE, t.SkinDeltaE)
		}
		if t.SkinBrightness > 0 {
			brightness = math.Min(brightness, t.SkinBrightness)
		}
	}
	return deltaE, brightness, clip
}

func measurePatch(source, target image.Image, rect image.Rectangle, clipHigh, clipLow int) PatchMetrics {
	rect = rect.Intersect(source.Bounds()).Intersect(target.Bounds())
	if rect.Empty() {
		return PatchMetrics{}
	}
	var sr, sg, sb, tr, tg, tb float64
	var sourceClip, targetClip, newClip int
	pixels := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			sc := rgbFromColor(source.At(x, y))
			tc := rgbFromColor(target.At(x, y))
			sr += srgbLinear(float64(sc.R) / 255)
			sg += srgbLinear(float64(sc.G) / 255)
			sb += srgbLinear(float64(sc.B) / 255)
			tr += srgbLinear(float64(tc.R) / 255)
			tg += srgbLinear(float64(tc.G) / 255)
			tb += srgbLinear(float64(tc.B) / 255)

			sClip := isClipped(sc, clipHigh, clipLow)
			tClip := isClipped(tc, clipHigh, clipLow)
			if sClip {
				sourceClip++
			}
			if tClip {
				targetClip++
				if !sClip {
					newClip++
				}
			}
			pixels++
		}
	}
	if pixels == 0 {
		return PatchMetrics{}
	}
	n := float64(pixels)
	meanSource := RGB{
		clampByte(linearToSRGB(sr/n) * 255),
		clampByte(linearToSRGB(sg/n) * 255),
		clampByte(linearToSRGB(sb/n) * 255),
	}
	meanTarget := RGB{
		clampByte(linearToSRGB(tr/n) * 255),
		clampByte(linearToSRGB(tg/n) * 255),
		clampByte(linearToSRGB(tb/n) * 255),
	}
	sl := meanSource.lab()
	tl := meanTarget.lab()
	return PatchMetrics{
		SourceRGB:  meanSource,
		TargetRGB:  meanTarget,
		SourceL:    sl.L,
		TargetL:    tl.L,
		DeltaL:     tl.L - sl.L,
		DeltaE:     deltaE2000(sl, tl),
		SourceClip: float64(sourceClip) / float64(pixels),
		TargetClip: float64(targetClip) / float64(pixels),
		NewClip:    float64(newClip) / float64(pixels),
	}
}

func isClipped(c RGB, high, low int) bool {
	return int(c.R) >= high || int(c.G) >= high || int(c.B) >= high ||
		int(c.R) <= low || int(c.G) <= low || int(c.B) <= low
}

func analyzeGroups(manifest *Manifest, patches []PatchResult) []GroupResult {
	groups := make([]GroupResult, 0)
	for _, spec := range manifest.Groups {
		dir := strings.ToLower(spec.Direction)
		if dir == "" {
			dir = "auto"
		}
		if dir == "none" {
			continue
		}
		ids := make([]string, 0)
		var members []int
		for i := range patches {
			if patches[i].Group == spec.ID {
				ids = append(ids, patches[i].ID)
				members = append(members, i)
			}
		}
		if len(members) < 2 {
			continue
		}
		sort.SliceStable(members, func(i, j int) bool {
			left, right := patches[members[i]], patches[members[j]]
			if left.Order != right.Order {
				return left.Order < right.Order
			}
			return members[i] < members[j]
		})
		if dir == "auto" {
			dir = inferDirection(patches[members[0]], patches[members[len(members)-1]], manifest.Thresholds.MonotonicityDeltaL)
		}
		gr := GroupResult{
			ID:         spec.ID,
			Name:       spec.Name,
			Direction:  dir,
			PatchIDs:   ids,
			Violations: make([]MonotonicityViolation, 0),
			Pass:       true,
		}
		if gr.Name == "" {
			gr.Name = spec.ID
		}
		if dir == "none" {
			groups = append(groups, gr)
			continue
		}
		limit := manifest.Thresholds.MonotonicityDeltaL
		for i := 1; i < len(members); i++ {
			left := patches[members[i-1]]
			right := patches[members[i]]
			delta := right.Metrics.TargetL - left.Metrics.TargetL
			bad := false
			switch dir {
			case "increasing":
				bad = delta < -limit
			case "decreasing":
				bad = delta > limit
			}
			if bad {
				gr.Violations = append(gr.Violations, MonotonicityViolation{
					Left:      left.ID,
					Right:     right.ID,
					LeftL:     left.Metrics.TargetL,
					RightL:    right.Metrics.TargetL,
					DeltaL:    delta,
					Direction: dir,
					Limit:     limit,
				})
				gr.Pass = false
			}
		}
		groups = append(groups, gr)
	}
	return groups
}

func inferDirection(first, last PatchResult, limit float64) string {
	delta := last.Metrics.SourceL - first.Metrics.SourceL
	switch {
	case delta > limit:
		return "increasing"
	case delta < -limit:
		return "decreasing"
	default:
		return "none"
	}
}

func applyMonotonicityFailures(patches []PatchResult, groups []GroupResult, manifest *Manifest) {
	for _, gr := range groups {
		if gr.Pass {
			continue
		}
		flagged := make(map[string]bool)
		for _, v := range gr.Violations {
			flagged[v.Left] = true
			flagged[v.Right] = true
		}
		for i := range patches {
			if !flagged[patches[i].ID] {
				continue
			}
			patches[i].Failures = append(patches[i].Failures, Failure{
				Code:    "monotonicity",
				Message: fmt.Sprintf("group %q is not %s", gr.ID, gr.Direction),
				Value:   math.Abs(patches[i].Metrics.TargetL - patches[i].Metrics.SourceL),
				Limit:   manifest.Thresholds.MonotonicityDeltaL,
			})
			patches[i].Pass = false
		}
	}
}

func summarize(patches []PatchResult, t Thresholds) Summary {
	s := Summary{PatchCount: len(patches), LocalFailures: make([]string, 0)}
	var sumE, sumL, sumClip float64
	for _, p := range patches {
		sumE += p.Metrics.DeltaE
		sumL += math.Abs(p.Metrics.DeltaL)
		sumClip += p.Metrics.NewClip
		if p.Metrics.DeltaE > s.MaxDeltaE {
			s.MaxDeltaE = p.Metrics.DeltaE
		}
		if abs := math.Abs(p.Metrics.DeltaL); abs > s.MaxDeltaL {
			s.MaxDeltaL = abs
		}
		if p.Pass {
			s.Passed++
		} else {
			s.Failed++
			codes := make([]string, 0, len(p.Failures))
			for _, f := range p.Failures {
				codes = append(codes, f.Code)
			}
			s.LocalFailures = append(s.LocalFailures, fmt.Sprintf("%s (%s)", p.ID, strings.Join(codes, ", ")))
		}
	}
	if len(patches) > 0 {
		n := float64(len(patches))
		s.MeanDeltaE = sumE / n
		s.MeanDeltaL = sumL / n
		s.MeanNewClip = sumClip / n
	}
	s.AveragePass = s.MeanDeltaE <= t.DeltaE && s.MeanDeltaL <= t.Brightness
	s.LocalPass = s.Failed == 0
	if s.LocalPass {
		s.Status = "pass"
	} else {
		s.Status = "fail"
	}
	return s
}
