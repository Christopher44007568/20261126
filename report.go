package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type opinionJSON struct {
	Layer    string `json:"layer"`
	Strength int    `json:"strength"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Value    string `json:"value"`
	Winner   bool   `json:"winner"`
}

type attributeJSON struct {
	Name               string        `json:"name"`
	TypeName           string        `json:"typeName"`
	WinnerLayer        string        `json:"winnerLayer"`
	WinnerPrimPath     string        `json:"winnerPrimPath"`
	OpinionsStrongWeak []opinionJSON `json:"opinionsStrongToWeak"`
}

type primJSON struct {
	Path       string          `json:"path"`
	TypeName   string          `json:"typeName"`
	Defined    bool            `json:"defined"`
	Attributes []attributeJSON `json:"attributes"`
}

type sourcesJSON struct {
	Rule                      string     `json:"rule"`
	TieBreak                  string     `json:"tieBreak"`
	StrengthOrderStrongToWeak []string   `json:"strengthOrderStrongToWeak"`
	Prims                     []primJSON `json:"prims"`
}

func RenderOpinionJSON(scene *Scene) ([]byte, error) {
	doc := sourcesJSON{
		Rule:     "higher declared layerStrength wins; load order is ignored",
		TieBreak: "equal strength -> lexicographically smaller layerId; same layer -> later source line",
	}
	for _, layer := range scene.Layers {
		doc.StrengthOrderStrongToWeak = append(doc.StrengthOrderStrongToWeak,
			fmt.Sprintf("%s(strength=%d)", layer.ID, layer.Strength))
	}
	for _, prim := range scene.SortedPrims() {
		entry := primJSON{
			Path:     prim.Path,
			TypeName: prim.TypeName,
			Defined:  prim.Defined,
		}
		for _, name := range prim.AttrNames {
			comp := prim.Attributes[name]
			attr := attributeJSON{
				Name:           comp.Name,
				TypeName:       comp.TypeName,
				WinnerLayer:    comp.Winner.LayerID,
				WinnerPrimPath: comp.Winner.PrimPath,
			}
			for i, opinion := range comp.Opinions {
				attr.OpinionsStrongWeak = append(attr.OpinionsStrongWeak, opinionJSON{
					Layer:    opinion.LayerID,
					Strength: opinion.Strength,
					File:     opinion.File,
					Line:     opinion.Line,
					Value:    opinion.Value,
					Winner:   i == 0,
				})
			}
			entry.Attributes = append(entry.Attributes, attr)
		}
		doc.Prims = append(doc.Prims, entry)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func RenderOpinionText(scene *Scene) []byte {
	var b strings.Builder
	b.WriteString("STRONG-OPINION SOURCES (winner per attribute)\n")
	b.WriteString("============================================\n")
	for _, prim := range scene.SortedPrims() {
		for _, name := range prim.AttrNames {
			comp := prim.Attributes[name]
			fmt.Fprintf(&b, "%-28s %-24s -> %-9s s=%-3d (%d opinion(s))  %s:%d\n",
				prim.Path, name, comp.Winner.LayerID, comp.Winner.Strength,
				len(comp.Opinions), comp.Winner.File, comp.Winner.Line)
		}
	}
	return []byte(b.String())
}
