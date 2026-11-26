// Package migrate performs namespace re-rooting.
//
// MigrateCorrect walks every prim and every arc and rewrites whole path
// prefixes, so descendant parent/child paths and all relationship targets move
// together. MigrateStringOnly reproduces the warned-about edit: arc strings are
// text-replaced and the namespace root string is renamed, but descendant
// parent/child paths are left on the old namespace. The audit package then
// quotes the concrete old path of every dangling arc.
package migrate

import (
	"strings"

	"nsmigrate/paths"
	"nsmigrate/scene"
)

// Clone returns a deep copy so before/after stages are independent.
func Clone(in *scene.Stage) *scene.Stage {
	out := &scene.Stage{Name: in.Name}
	for _, p := range in.Prims {
		cp := *p
		out.Prims = append(out.Prims, &cp)
	}
	for _, l := range in.Links {
		cl := *l
		out.Links = append(out.Links, &cl)
	}
	return out
}

// CorrectReport summarizes the path-aware migration.
type CorrectReport struct {
	Mapping          scene.Mapping `json:"mapping"`
	RewrittenPrims   int           `json:"rewritten_prims"`
	RewrittenLinks   int           `json:"rewritten_links"`
	ParentChildClean bool          `json:"parent_child_clean"`
}

// MigrateCorrect re-roots the namespace subtree while keeping parent/child
// paths and every reference/material/camera arc consistent.
func MigrateCorrect(in *scene.Stage, m scene.Mapping) (*scene.Stage, CorrectReport) {
	s := Clone(in)
	rep := CorrectReport{Mapping: m, ParentChildClean: true}

	for _, p := range s.Prims {
		if next, ok := paths.RewritePrefix(m.OldRoot, m.NewRoot, p.Path); ok {
			p.Path = next
			rep.RewrittenPrims++
		}
	}
	for _, l := range s.Links {
		if next, ok := paths.RewritePrefix(m.OldRoot, m.NewRoot, l.Target); ok {
			l.Target = next
			rep.RewrittenLinks++
		}
	}
	rep.ParentChildClean = len(CheckParentChild(s)) == 0
	return s, rep
}

// ParentChildMismatch describes a node whose absolute path no longer agrees
// with its parent's path + its own base name.
type ParentChildMismatch struct {
	PrimID       string `json:"prim_id"`
	Path         string `json:"path"`
	ParentID     string `json:"parent_id"`
	ExpectedPath string `json:"expected_path"`
}

// CheckParentChild verifies child.Path == parent.Path + "/" + Base(child.Path).
func CheckParentChild(s *scene.Stage) []ParentChildMismatch {
	var bad []ParentChildMismatch
	byID := map[string]*scene.Prim{}
	for _, p := range s.Prims {
		byID[p.ID] = p
	}
	for _, p := range s.Prims {
		if p.ParentID == "" {
			continue
		}
		parent := byID[p.ParentID]
		if parent == nil {
			bad = append(bad, ParentChildMismatch{PrimID: p.ID, Path: p.Path, ParentID: p.ParentID, ExpectedPath: "<missing parent>"})
			continue
		}
		want := paths.Join(parent.Path, paths.Base(p.Path))
		if p.Path != want {
			bad = append(bad, ParentChildMismatch{
				PrimID: p.ID, Path: p.Path, ParentID: p.ParentID, ExpectedPath: want,
			})
		}
	}
	return bad
}

// MigrateStringOnly reproduces the incomplete text edit described in the
// feedback:
//
//   - relationship target strings get a literal strings.ReplaceAll,
//   - only the namespace root node string is renamed,
//   - descendant prim paths (the parent/child chain) are NOT re-rooted.
//
// The returned stage is intentionally inconsistent and must be diagnosed with
// audit.Audit.
func MigrateStringOnly(in *scene.Stage, m scene.Mapping) *scene.Stage {
	s := Clone(in)
	for _, l := range s.Links {
		l.Target = strings.ReplaceAll(l.Target, m.OldRoot, m.NewRoot)
	}
	for _, p := range s.Prims {
		if p.ID == "root" || p.Path == m.OldRoot {
			p.Path = m.NewRoot
		}
	}
	return s
}
