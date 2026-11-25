package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const provenanceSchema = "materialviz/provenance/v1"

type OpinionRecord struct {
	Slot          int    `json:"slot"`
	ID            string `json:"id"`
	Layer         Layer  `json:"layer"`
	MaterialID    string `json:"material_id"`
	MaterialName  string `json:"material_name"`
	Color         string `json:"color"`
	Source        string `json:"source"`
	Rationale     string `json:"rationale"`
	Depth         int    `json:"depth"`
	Salience      int    `json:"salience"`
	Shape         string `json:"shape,omitempty"`
	EffectiveRank int    `json:"effective_rank"`
	Diagnostic    bool   `json:"diagnostic,omitempty"`
}

type Provenance struct {
	Schema           string          `json:"schema"`
	Mode             string          `json:"mode"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Cascade          string          `json:"cascade"`
	Width            int             `json:"width"`
	Height           int             `json:"height"`
	PixelLookup      string          `json:"pixel_lookup"`
	RankNote         string          `json:"rank_note"`
	Opinions         []OpinionRecord `json:"opinions"`
	WinnerSlots      [][]int         `json:"winner_slots"`
	UnresolvedPixels int             `json:"unresolved_pixels"`
}

func buildProvenance(scene Scene, mode ModeSpec, winners []PixelWinner) Provenance {
	unique := make(map[string]Opinion)
	rankByKey := make(map[string]int)
	for _, winner := range winners {
		key := opinionKey(winner.Opinion)
		unique[key] = winner.Opinion
		rankByKey[key] = winner.Rank
	}
	keys := sortedOpinionKeys(unique)
	slotByKey := make(map[string]int, len(keys))
	records := make([]OpinionRecord, 0, len(keys))
	for slot, key := range keys {
		opinion := unique[key]
		slotByKey[key] = slot
		shape := ""
		if opinion.Shape != nil {
			shape = opinion.Shape.Describe()
		}
		records = append(records, OpinionRecord{
			Slot:          slot,
			ID:            opinion.ID,
			Layer:         opinion.Layer,
			MaterialID:    opinion.Material.ID,
			MaterialName:  opinion.Material.Name,
			Color:         opinion.Material.Color.Hex(),
			Source:        opinion.Source,
			Rationale:     opinion.Rationale,
			Depth:         opinion.Depth,
			Salience:      opinion.Salience,
			Shape:         shape,
			EffectiveRank: rankByKey[key],
			Diagnostic:    opinion.Layer == LayerDiagnostic,
		})
	}

	rows := make([][]int, scene.Height)
	unresolved := 0
	for y := 0; y < scene.Height; y++ {
		rows[y] = make([]int, scene.Width)
		for x := 0; x < scene.Width; x++ {
			winner := pixel(winners, scene, x, y)
			rows[y][x] = slotByKey[opinionKey(winner.Opinion)]
			if winner.Diagnostic {
				unresolved++
			}
		}
	}
	rankNote := "Higher effective_rank wins; local=400+, instance=300+, inherited=200+ in the correct cascade."
	if mode.InheritedAboveInstance {
		rankNote = "Higher effective_rank wins; this negative-control mode intentionally gives inherited=300+ and instance=200+."
	} else if !mode.IncludeInherited {
		rankNote = "Higher effective_rank wins; inherited opinions are disabled, so local=400+, instance=300+, and unmatched pixels use rank -1 diagnostics."
	}

	return Provenance{
		Schema:           provenanceSchema,
		Mode:             mode.ID,
		Title:            mode.Title,
		Description:      mode.Description,
		Cascade:          mode.Cascade,
		Width:            scene.Width,
		Height:           scene.Height,
		PixelLookup:      "pixel at (x,y) is opinions[winner_slots[y][x]]",
		RankNote:         rankNote,
		Opinions:         records,
		WinnerSlots:      rows,
		UnresolvedPixels: unresolved,
	}
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
