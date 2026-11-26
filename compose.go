package main

import (
	"sort"
)

func Compose(layers []*Layer) *Scene {
	ordered := append([]*Layer(nil), layers...)
	sort.Slice(ordered, func(i, j int) bool {
		return compareLayer(ordered[i], ordered[j]) < 0
	})

	scene := &Scene{
		Layers: ordered,
		Prims:  make(map[string]*Prim),
	}
	for _, layer := range ordered {
		for _, spec := range layer.Specs {
			prim := scene.Prims[spec.Path]
			if prim == nil {
				prim = &Prim{
					Path:       spec.Path,
					Name:       spec.Name,
					Attributes: make(map[string]*AttributeComposition),
				}
				scene.Prims[spec.Path] = prim
				scene.Paths = append(scene.Paths, spec.Path)
			}
			prim.Specs = append(prim.Specs, spec)
			if spec.Specifier == SpecDef {
				prim.Defined = true
			}
		}
	}

	sort.Strings(scene.Paths)
	for _, prim := range scene.Prims {
		sort.SliceStable(prim.Specs, func(i, j int) bool {
			return compareSpec(prim.Specs[i], prim.Specs[j]) < 0
		})
		for _, spec := range prim.Specs {
			if prim.TypeName == "" && spec.TypeName != "" {
				prim.TypeName = spec.TypeName
			}
			for _, attr := range spec.Attributes {
				comp := prim.Attributes[attr.Name]
				if comp == nil {
					comp = &AttributeComposition{
						Name:     attr.Name,
						TypeName: attr.TypeName,
					}
					prim.Attributes[attr.Name] = comp
					prim.AttrNames = append(prim.AttrNames, attr.Name)
				}
				comp.Opinions = append(comp.Opinions, &Opinion{
					LayerID:   attr.Layer.ID,
					Strength:  attr.Layer.Strength,
					File:      attr.File,
					Line:      attr.Line,
					Value:     attr.Value,
					Color:     attr.Color,
					PrimPath:  spec.Path,
					Specifier: spec.Specifier,
				})
			}
		}
		sort.Strings(prim.AttrNames)
		for _, name := range prim.AttrNames {
			comp := prim.Attributes[name]
			sort.SliceStable(comp.Opinions, func(i, j int) bool {
				return compareOpinion(comp.Opinions[i], comp.Opinions[j]) < 0
			})
			if len(comp.Opinions) > 0 {
				comp.Winner = comp.Opinions[0]
			}
		}
	}
	return scene
}

func compareSpec(a, b *PrimSpec) int {
	if a.Layer.Strength != b.Layer.Strength {
		if a.Layer.Strength > b.Layer.Strength {
			return -1
		}
		return 1
	}
	if a.Layer.ID != b.Layer.ID {
		if a.Layer.ID < b.Layer.ID {
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

func (s *Scene) Attribute(path, name string) *AttributeComposition {
	prim := s.Prims[path]
	if prim == nil {
		return nil
	}
	return prim.Attributes[name]
}

func (s *Scene) Winner(path, name string) *Opinion {
	comp := s.Attribute(path, name)
	if comp == nil {
		return nil
	}
	return comp.Winner
}
