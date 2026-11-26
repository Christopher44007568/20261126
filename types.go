package main

import (
	"fmt"
	"sort"
)

// Specifier is the non-destructive prim statement kind used by the USDA subset.
type Specifier string

const (
	SpecDef   Specifier = "def"
	SpecOver  Specifier = "over"
	SpecClass Specifier = "class"
)

type Color3 struct {
	R float64
	G float64
	B float64
}

func (c Color3) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", channel(c.R), channel(c.G), channel(c.B))
}

func channel(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

type Layer struct {
	ID       string
	File     string
	Strength int
	Specs    []*PrimSpec
}

type PrimSpec struct {
	Path       string
	Name       string
	TypeName   string
	Specifier  Specifier
	Layer      *Layer
	File       string
	Line       int
	Attributes []*AttributeSpec
}

type AttributeSpec struct {
	Name     string
	TypeName string
	Value    string
	Color    Color3
	Line     int
	Spec     *PrimSpec
	Layer    *Layer
	File     string
}

type Prim struct {
	Path       string
	Name       string
	TypeName   string
	Defined    bool
	Specs      []*PrimSpec
	Attributes map[string]*AttributeComposition
	AttrNames  []string
}

type AttributeComposition struct {
	Name     string
	TypeName string
	Opinions []*Opinion
	Winner   *Opinion
}

type Opinion struct {
	LayerID   string
	Strength  int
	File      string
	Line      int
	Value     string
	Color     Color3
	PrimPath  string
	Specifier Specifier
}

type Scene struct {
	Layers []*Layer
	Prims  map[string]*Prim
	Paths  []string
}

func (s *Scene) SortedPrims() []*Prim {
	out := make([]*Prim, 0, len(s.Paths))
	for _, path := range s.Paths {
		out = append(out, s.Prims[path])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func compareLayer(a, b *Layer) int {
	if a.Strength != b.Strength {
		if a.Strength > b.Strength {
			return -1
		}
		return 1
	}
	if a.ID != b.ID {
		if a.ID < b.ID {
			return -1
		}
		return 1
	}
	return 0
}

func compareOpinion(a, b *Opinion) int {
	if a.Strength != b.Strength {
		if a.Strength > b.Strength {
			return -1
		}
		return 1
	}
	if a.LayerID != b.LayerID {
		if a.LayerID < b.LayerID {
			return -1
		}
		return 1
	}
	if a.Line != b.Line {
		if a.Line > b.Line {
			return -1
		}
		return 1
	}
	if a.File < b.File {
		return -1
	}
	if a.File > b.File {
		return 1
	}
	return 0
}
