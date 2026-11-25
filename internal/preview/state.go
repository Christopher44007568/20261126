package preview

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"dds-preview/internal/dds"
)

// Previewer owns the last successfully parsed texture. Load only commits a
// new texture after Parse has validated every row of every face and mip.
type Previewer struct {
	current       *dds.Texture
	currentSource string
}

func New() *Previewer {
	return &Previewer{}
}

func (p *Previewer) Current() (*dds.Texture, string) {
	if p.current == nil {
		return nil, ""
	}
	return p.current, p.currentSource
}

// Load parses data and commits it only when the complete texture is valid.
// On failure the previous texture remains current.
func (p *Previewer) Load(data []byte, source string) *Result {
	tex, err := dds.Parse(data)
	if err != nil {
		res := &Result{OK: false, Source: source, Errors: dds.FieldErrors(err)}
		if len(res.Errors) == 0 {
			res.Errors = []dds.FieldError{{Field: "source", Message: err.Error()}}
		}
		p.fillPrevious(res)
		return res
	}
	p.current = tex
	p.currentSource = source
	return &Result{
		OK: true, Source: source, Current: ptr(tex.Summary()),
		Errors: []dds.FieldError{},
	}
}

// LoadFile reads a file and applies the same all-or-nothing commit rule.
func (p *Previewer) LoadFile(path string) *Result {
	data, err := os.ReadFile(path)
	if err != nil {
		res := &Result{OK: false, Source: path}
		field := "source"
		if errors.Is(err, fs.ErrNotExist) {
			field = "source.path"
		}
		res.Errors = []dds.FieldError{{
			Field: field, Offset: -1, Message: fmt.Sprintf("cannot read DDS: %v", err),
		}}
		p.fillPrevious(res)
		return res
	}
	return p.Load(data, path)
}

func (p *Previewer) fillPrevious(res *Result) {
	if p.current != nil {
		res.KeptPrevious = true
		res.Current = ptr(p.current.Summary())
		res.PreviousSource = p.currentSource
	}
}

// Result describes one load attempt. Current always refers to a complete,
// valid texture, including when OK is false and KeptPrevious is true.
type Result struct {
	OK             bool                `json:"ok"`
	Source         string              `json:"source"`
	PreviousSource string              `json:"previousSource,omitempty"`
	KeptPrevious   bool                `json:"keptPreviousTexture"`
	Current        *dds.TextureSummary `json:"currentTexture,omitempty"`
	Errors         []dds.FieldError    `json:"errors"`
	Artifacts      []Artifact          `json:"artifacts,omitempty"`
	OutputError    string              `json:"outputError,omitempty"`
}

type Artifact struct {
	Kind   string `json:"kind"`
	Face   int    `json:"face"`
	Level  int    `json:"level"`
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func ptr[T any](v T) *T {
	return &v
}
