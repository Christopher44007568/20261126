package texresolver

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Root is one directory in the texture search order.
type Root struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Priority int    `json:"priority"`
}

// Candidate records every location that contains the requested filename.
type Candidate struct {
	RootID   string    `json:"rootId"`
	RootName string    `json:"rootLabel"`
	Kind     string    `json:"kind"`
	Priority int       `json:"priority"`
	Path     string    `json:"path"`
	Exists   bool      `json:"exists"`
	Size     int64     `json:"size,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

// Trace is the ordered outcome of probing one search root.
type Trace struct {
	RootID   string `json:"rootId"`
	RootName string `json:"rootLabel"`
	Kind     string `json:"kind"`
	Priority int    `json:"priority"`
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
}

// Resolution is the final decision for a single filename.
type Resolution struct {
	Filename    string      `json:"filename"`
	Found       bool        `json:"found"`
	Selected    *Candidate  `json:"selected,omitempty"`
	Existing    []Candidate `json:"existing"`
	Conflicts   []Candidate `json:"conflicts"`
	Trace       []Trace     `json:"trace"`
	Placeholder bool        `json:"placeholder"`
	Source      string      `json:"source"`
}

// Resolver applies a stable, ordered search across local roots.
type Resolver struct {
	Roots []Root
}

func (r Resolver) Resolve(filename string) (Resolution, error) {
	if filename == "" || filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\`) {
		return Resolution{}, errors.New("filename must be a single path element")
	}

	result := Resolution{
		Filename: filename,
		Trace:    make([]Trace, 0, len(r.Roots)),
		Existing: make([]Candidate, 0, 1),
	}

	for _, root := range r.Roots {
		full := filepath.Join(root.Path, filename)
		info, err := os.Stat(full)
		exists := err == nil && info.Mode().IsRegular()

		trace := Trace{
			RootID: root.ID, RootName: root.Label, Kind: root.Kind,
			Priority: root.Priority, Path: filepath.ToSlash(full), Exists: exists,
		}
		result.Trace = append(result.Trace, trace)

		if !exists {
			continue
		}
		candidate := Candidate{
			RootID: root.ID, RootName: root.Label, Kind: root.Kind,
			Priority: root.Priority, Path: filepath.ToSlash(full),
			Exists: true, Size: info.Size(), Modified: info.ModTime().UTC(),
		}
		result.Existing = append(result.Existing, candidate)
		if result.Selected == nil {
			pick := candidate
			result.Selected = &pick
			result.Found = true
			result.Source = candidate.Path
		}
	}

	if result.Found {
		for _, candidate := range result.Existing {
			if candidate.Path != result.Selected.Path {
				result.Conflicts = append(result.Conflicts, candidate)
			}
		}
		sort.SliceStable(result.Conflicts, func(i, j int) bool {
			return result.Conflicts[i].Priority < result.Conflicts[j].Priority
		})
	} else {
		result.Placeholder = true
		result.Source = "procedural://checkerboard"
	}

	return result, nil
}
