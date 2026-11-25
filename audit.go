package main

import (
	"fmt"
	"sort"
)

const auditSchema = "materialviz/audit/v1"

type MismatchGroup struct {
	ActualOpinionID   string `json:"actual_opinion_id"`
	ActualLayer       Layer  `json:"actual_layer"`
	ActualMaterialID  string `json:"actual_material_id"`
	ActualColor       string `json:"actual_color"`
	ActualSource      string `json:"actual_source"`
	ActualRationale   string `json:"actual_rationale"`
	ExpectedOpinionID string `json:"expected_opinion_id"`
	ExpectedLayer     Layer  `json:"expected_layer"`
	ExpectedColor     string `json:"expected_color"`
	ExpectedSource    string `json:"expected_source"`
	PixelCount        int    `json:"pixel_count"`
	PixelIndices      []int  `json:"pixel_indices"`
}

type ModeAudit struct {
	Mode                            string          `json:"mode"`
	ImageFile                       string          `json:"image_file"`
	ProvenanceFile                  string          `json:"provenance_file"`
	WrongColorPixels                int             `json:"wrong_color_pixels"`
	SameColorDifferentOpinionPixels int             `json:"same_color_different_opinion_pixels"`
	Groups                          []MismatchGroup `json:"groups"`
}

type Audit struct {
	Schema              string      `json:"schema"`
	ReferenceMode       string      `json:"reference_mode"`
	ReferenceImage      string      `json:"reference_image"`
	ReferenceProvenance string      `json:"reference_provenance"`
	Definition          string      `json:"definition"`
	PixelIndexFormula   string      `json:"pixel_index_formula"`
	Modes               []ModeAudit `json:"modes"`
}

func auditRenders(reference ModeSpec, referenceWinners []PixelWinner, modes []ModeSpec, winnersByMode map[string][]PixelWinner) Audit {
	audit := Audit{
		Schema:              auditSchema,
		ReferenceMode:       reference.ID,
		ReferenceImage:      reference.ImageFile,
		ReferenceProvenance: reference.ProvenanceFile,
		Definition:          "A pixel is wrong when its delivered color differs from after_override. Every group names the actual final opinion that painted it.",
		PixelIndexFormula:   "index = y*width + x; x = index%width; y = index/width",
	}
	for _, mode := range modes {
		actualWinners := winnersByMode[mode.ID]
		modeAudit := ModeAudit{
			Mode:           mode.ID,
			ImageFile:      mode.ImageFile,
			ProvenanceFile: mode.ProvenanceFile,
			Groups:         []MismatchGroup{},
		}
		groups := make(map[string]*MismatchGroup)
		for index := range referenceWinners {
			expected := referenceWinners[index]
			actual := actualWinners[index]
			if actual.Opinion.Material.Color == expected.Opinion.Material.Color {
				if actual.Opinion.ID != expected.Opinion.ID {
					modeAudit.SameColorDifferentOpinionPixels++
				}
				continue
			}
			modeAudit.WrongColorPixels++
			key := actual.Opinion.ID + "\x00" + actual.Opinion.Material.Color.Hex() + "\x00" +
				expected.Opinion.ID + "\x00" + expected.Opinion.Material.Color.Hex()
			group, ok := groups[key]
			if !ok {
				group = &MismatchGroup{
					ActualOpinionID:   actual.Opinion.ID,
					ActualLayer:       actual.Opinion.Layer,
					ActualMaterialID:  actual.Opinion.Material.ID,
					ActualColor:       actual.Opinion.Material.Color.Hex(),
					ActualSource:      actual.Opinion.Source,
					ActualRationale:   actual.Opinion.Rationale,
					ExpectedOpinionID: expected.Opinion.ID,
					ExpectedLayer:     expected.Opinion.Layer,
					ExpectedColor:     expected.Opinion.Material.Color.Hex(),
					ExpectedSource:    expected.Opinion.Source,
				}
				groups[key] = group
			}
			group.PixelIndices = append(group.PixelIndices, index)
			group.PixelCount = len(group.PixelIndices)
		}
		groupKeys := make([]string, 0, len(groups))
		for key := range groups {
			groupKeys = append(groupKeys, key)
		}
		sort.Strings(groupKeys)
		for _, key := range groupKeys {
			modeAudit.Groups = append(modeAudit.Groups, *groups[key])
		}
		audit.Modes = append(audit.Modes, modeAudit)
	}
	return audit
}

func validateAudit(audit Audit) error {
	for _, mode := range audit.Modes {
		total := 0
		for _, group := range mode.Groups {
			if group.PixelCount != len(group.PixelIndices) {
				return fmt.Errorf("%s: group count does not match pixel indices", mode.Mode)
			}
			total += group.PixelCount
		}
		if total != mode.WrongColorPixels {
			return fmt.Errorf("%s: wrong-color total %d does not match groups %d", mode.Mode, mode.WrongColorPixels, total)
		}
	}
	return nil
}
