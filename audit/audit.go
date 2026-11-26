// Package audit resolves every reference, material binding and camera target,
// classifies failures, and records the concrete old namespace path so a
// string-only edit that skipped parent/child updates can be traced.
package audit

import (
	"nsmigrate/migrate"
	"nsmigrate/paths"
	"nsmigrate/scene"
)

// Status is the resolved state of one arc.
type Status string

const (
	StatusOK      Status = "ok"
	StatusBroken  Status = "broken"
	StatusDefault Status = "default_retained"
)

// Finding is one audited link with full old/new path provenance.
type Finding struct {
	LinkID          string         `json:"link_id"`
	Kind            scene.LinkKind `json:"kind"`
	RelName         string         `json:"rel"`
	OwnerID         string         `json:"owner_id"`
	OwnerPath       string         `json:"owner_path"`
	Target          string         `json:"target"`
	OldPath         string         `json:"old_path"`
	Resolved        bool           `json:"resolved"`
	ResolvedPrimID  string         `json:"resolved_prim_id,omitempty"`
	Status          Status         `json:"status"`
	StalePath       string         `json:"stale_parent_child_path,omitempty"`
	DefaultMaterial string         `json:"default_material,omitempty"`
	Hint            string         `json:"hint,omitempty"`
	Expect          scene.Expect   `json:"expect"`
	AsExpected      bool           `json:"as_expected"`
}

// DefaultValue marks a built-in value retained because an arc was unresolved.
type DefaultValue struct {
	LinkID    string    `json:"link_id"`
	OwnerID   string    `json:"owner_id"`
	OwnerPath string    `json:"owner_path"`
	Property  string    `json:"property"`
	ValueName string    `json:"value_name"`
	ValueRGB  scene.RGB `json:"value_rgb"`
	OldPath   string    `json:"old_path"`
	Because   string    `json:"because"`
}

// Report is the full audit result for one rendered state.
type Report struct {
	Mode        string                        `json:"mode"`
	Mapping     scene.Mapping                 `json:"mapping"`
	Findings    []Finding                     `json:"findings"`
	Defaults    []DefaultValue                `json:"defaults"`
	ParentChild []migrate.ParentChildMismatch `json:"parent_child_mismatches"`
	Counts      map[string]int                `json:"counts"`
	InvariantOK bool                          `json:"invariant_ok"`
}

func expectedStatus(e scene.Expect) Status {
	switch e {
	case scene.ExpectBroken:
		return StatusBroken
	case scene.ExpectDefault:
		return StatusDefault
	default:
		return StatusOK
	}
}

// Audit resolves all links in s. mode labels the state ("before", "correct",
// "string-only"). The mapping is used only to invert targets when looking for a
// stale old parent/child path.
func Audit(mode string, s *scene.Stage, m scene.Mapping) *Report {
	r := &Report{Mode: mode, Mapping: m, Counts: map[string]int{}, InvariantOK: true}
	r.ParentChild = migrate.CheckParentChild(s)
	if len(r.ParentChild) > 0 {
		r.InvariantOK = false
	}

	for _, link := range s.Links {
		owner := s.PrimByID(link.OwnerID)
		ownerPath := ""
		if owner != nil {
			ownerPath = owner.Path
		}

		f := Finding{
			LinkID:    link.ID,
			Kind:      link.Kind,
			RelName:   link.RelName,
			OwnerID:   link.OwnerID,
			OwnerPath: ownerPath,
			Target:    link.Target,
			OldPath:   link.OldTarget,
			Expect:    link.Expect,
		}

		if target := s.PrimByPath(link.Target); target != nil {
			f.Resolved = true
			f.ResolvedPrimID = target.ID
			f.Status = StatusOK
		} else {
			if link.Kind == scene.KindMaterialBinding {
				f.Status = StatusDefault
				f.DefaultMaterial = scene.DefaultMaterialName
				f.Hint = "binding unresolved; renderer retains the built-in default material"
				r.Defaults = append(r.Defaults, DefaultValue{
					LinkID: link.ID, OwnerID: link.OwnerID, OwnerPath: ownerPath,
					Property: "material:binding", ValueName: scene.DefaultMaterialName,
					ValueRGB: scene.DefaultMaterialColor, OldPath: link.OldTarget,
					Because: link.Target,
				})
			} else {
				f.Status = StatusBroken
				f.Hint = "target path does not resolve to any prim"
			}

			// When a string replacement moved the arc but the node stayed at its
			// old parent/child path, recover that concrete stale path.
			if oldCandidate, inv := paths.RewritePrefix(m.NewRoot, m.OldRoot, link.Target); inv {
				if stale := s.PrimByPath(oldCandidate); stale != nil {
					f.StalePath = oldCandidate
					f.Hint = "arc string was replaced, but the prim still lives at the old " +
						"parent/child path; re-root the whole subtree"
				}
			}
		}

		f.AsExpected = f.Status == expectedStatus(link.Expect)
		if !f.AsExpected {
			r.InvariantOK = false
		}
		r.Findings = append(r.Findings, f)
		r.Counts[string(f.Status)]++
		r.Counts[string(link.Kind)]++
	}
	return r
}

// FindingsFor returns findings of one link kind.
func (r *Report) FindingsFor(kind scene.LinkKind) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}
