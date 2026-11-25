package timeline

// KeyCoverage records whether an irregular sample was visible in a sheet.
type KeyCoverage struct {
	Index                int     `json:"index"`
	Time                 float64 `json:"time"`
	Name                 string  `json:"name"`
	Signature            bool    `json:"signature"`
	Reachable            bool    `json:"reachable"`
	ShownInFrame         bool    `json:"shown_in_frame"`
	SweptByInterpolation bool    `json:"swept_by_interpolation"`
	Skipped              bool    `json:"skipped"`
}

// SheetResult is a fully evaluated contact sheet and its key coverage audit.
type SheetResult struct {
	ID                   string        `json:"id"`
	CNTitle              string        `json:"cn_title"`
	Image                string        `json:"image,omitempty"`
	SVGImage             string        `json:"svg_image,omitempty"`
	Spec                 SheetSpec     `json:"spec"`
	Frames               []Frame       `json:"frames"`
	KeyCoverage          []KeyCoverage `json:"key_coverage"`
	SkippedSignatureKeys []string      `json:"skipped_signature_keys"`
}

// StaticNodeCheck proves a sample-less node stayed on its own bind pose.
type StaticNodeCheck struct {
	ID                        string `json:"id"`
	Kind                      string `json:"kind"`
	Status                    string `json:"status"`
	FramesChecked             int    `json:"frames_checked"`
	IdenticalToBindPose       bool   `json:"identical_to_bind_pose"`
	NeverReferencesSampleKeys bool   `json:"never_references_sample_keys"`
}

// StaticPolicy is the cross-sheet isolation invariant.
type StaticPolicy struct {
	Rule  string            `json:"rule"`
	Nodes []StaticNodeCheck `json:"nodes"`
}

// BuildSheet evaluates all frames and coverage for one sheet.
func (s *Scene) BuildSheet(spec SheetSpec) SheetResult {
	frames := s.BuildFrames(spec)
	hero, _ := s.NodeByID(s.PrimaryID)
	keys := make([]KeyCoverage, 0, len(hero.Samples))

	minSource, maxSource := frames[0].SourceTime, frames[len(frames)-1].SourceTime
	if minSource > maxSource {
		minSource, maxSource = maxSource, minSource
	}

	for i, sample := range hero.Samples {
		cov := KeyCoverage{
			Index:     i,
			Time:      sample.Time,
			Name:      sample.Name,
			Signature: sample.Signature,
			Reachable: sample.Time >= minSource && sample.Time <= maxSource,
		}
		for _, frame := range frames {
			primary, ok := frame.Node(s.PrimaryID)
			if !ok {
				continue
			}
			if primary.ShownKey == i {
				cov.ShownInFrame = true
			}
			if spec.ReadMode == ModeLinear && primary.Status == StatusIn &&
				(primary.Lo == i || primary.Hi == i) {
				cov.SweptByInterpolation = true
			}
		}
		cov.Skipped = !cov.ShownInFrame && !cov.SweptByInterpolation
		keys = append(keys, cov)
	}

	result := SheetResult{
		ID:                   spec.ID,
		CNTitle:              spec.CNTitle,
		Spec:                 spec,
		Frames:               frames,
		KeyCoverage:          keys,
		SkippedSignatureKeys: []string{},
	}
	for _, cov := range keys {
		if cov.Signature && cov.Skipped {
			result.SkippedSignatureKeys = append(result.SkippedSignatureKeys, cov.Name)
		}
	}
	return result
}

// BuildAllSheets evaluates every configured sheet and the static isolation
// invariant across the complete output frame count.
func (s *Scene) BuildAllSheets() ([]SheetResult, StaticPolicy) {
	results := make([]SheetResult, 0, len(s.Sheets))
	totalFrames := 0
	for _, spec := range s.Sheets {
		result := s.BuildSheet(spec)
		results = append(results, result)
		totalFrames += len(result.Frames)
	}

	policy := StaticPolicy{
		Rule: "nodes without time samples evaluate their own bind/rest pose at every time; neighbor tracks are never read",
	}
	for _, node := range s.Nodes {
		if len(node.Samples) != 0 {
			continue
		}
		check := StaticNodeCheck{
			ID:                        node.ID,
			Kind:                      node.Kind,
			Status:                    string(StatusStatic),
			FramesChecked:             totalFrames,
			IdenticalToBindPose:       true,
			NeverReferencesSampleKeys: true,
		}
		for _, sheet := range results {
			for _, frame := range sheet.Frames {
				got, ok := frame.Node(node.ID)
				if !ok || got.Status != StatusStatic || got.Pose != node.Bind ||
					got.Lo != -1 || got.Hi != -1 || got.ShownKey != -1 {
					check.IdenticalToBindPose = false
					check.NeverReferencesSampleKeys = false
				}
			}
		}
		policy.Nodes = append(policy.Nodes, check)
	}
	return results, policy
}
