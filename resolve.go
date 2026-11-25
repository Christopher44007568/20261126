package main

import (
	"fmt"
	"sort"
)

type ModeSpec struct {
	ID                     string
	Title                  string
	ImageFile              string
	ProvenanceFile         string
	Cascade                string
	IncludeInherited       bool
	InheritedAboveInstance bool
	Description            string
}

var RenderModes = []ModeSpec{
	{
		ID:               "before_inheritance",
		Title:            "Before inheritance",
		ImageFile:        "01_before_inheritance.png",
		ProvenanceFile:   "before_inheritance.provenance.json",
		Cascade:          "local > instance > unresolved",
		IncludeInherited: false,
		Description:      "The chain is disabled to expose pixels that only an inherited opinion could paint.",
	},
	{
		ID:               "after_override",
		Title:            "After override",
		ImageFile:        "02_after_override.png",
		ProvenanceFile:   "after_override.provenance.json",
		Cascade:          "local > instance > inherited",
		IncludeInherited: true,
		Description:      "The correct cascade. The instance material beats every inherited class material.",
	},
	{
		ID:                     "forced_fallback",
		Title:                  "Forced fallback (negative control)",
		ImageFile:              "03_forced_fallback.png",
		ProvenanceFile:         "forced_fallback.provenance.json",
		Cascade:                "local > inherited > instance",
		IncludeInherited:       true,
		InheritedAboveInstance: true,
		Description:            "A deliberate strength inversion used to prove regression errors are attributable.",
	},
}

func modeByID(id string) (ModeSpec, bool) {
	for _, mode := range RenderModes {
		if mode.ID == id {
			return mode, true
		}
	}
	return ModeSpec{}, false
}

// inheritanceDepths walks the declared instance -> parent -> root chain and
// records each class's distance from the instance. The resolver uses these
// depths through the inherited opinions, so nearest-ancestor precedence comes
// from the chain rather than from an unrelated numeric priority.
func inheritanceDepths(chain []ClassNode, instanceOf string) map[string]int {
	byID := make(map[string]ClassNode, len(chain))
	for _, node := range chain {
		byID[node.ID] = node
	}
	depths := make(map[string]int, len(chain))
	visited := make(map[string]bool, len(chain))
	for class, depth := instanceOf, 0; class != "" && !visited[class]; class, depth = byID[class].Parent, depth+1 {
		visited[class] = true
		depths[class] = depth
	}
	return depths
}

func maxInheritedDepth(scene Scene) int {
	maxDepth := 0
	for _, opinion := range scene.Opinions {
		if opinion.Layer == LayerInherited && opinion.Depth > maxDepth {
			maxDepth = opinion.Depth
		}
	}
	return maxDepth
}

// effectiveRank encodes the cascade as numbers so tests and provenance can
// expose why a winner won. Layer separation is wider than any salience used by
// this scene, which keeps local > instance > inherited unambiguous.
func effectiveRank(opinion Opinion, mode ModeSpec, maxDepth int) (int, bool) {
	switch opinion.Layer {
	case LayerLocal:
		return 400 + opinion.Salience, true
	case LayerInstance:
		if mode.InheritedAboveInstance {
			return 200 + opinion.Salience, true
		}
		return 300 + opinion.Salience, true
	case LayerInherited:
		if !mode.IncludeInherited {
			return 0, false
		}
		base := 200
		if mode.InheritedAboveInstance {
			base = 300
		}
		return base + (maxDepth - opinion.Depth) + opinion.Salience, true
	default:
		return 0, false
	}
}

func unresolvedOpinion(x, y int) Opinion {
	color := RGB{86, 34, 62}
	parity := "even"
	if (x+y)&1 == 1 {
		color = RGB{112, 43, 79}
		parity = "odd"
	}
	return Opinion{
		ID:    "diagnostic:unresolved-" + parity,
		Layer: LayerDiagnostic,
		Material: Material{
			ID:    "diagnostic.unresolved",
			Name:  "Unresolved diagnostic",
			Color: color,
		},
		Depth:     -1,
		Source:    "renderer",
		Rationale: "no local, instance, or enabled inherited opinion covers this pixel",
	}
}

type PixelWinner struct {
	X          int
	Y          int
	Opinion    Opinion
	Rank       int
	Diagnostic bool
}

func resolvePixel(scene Scene, mode ModeSpec, maxDepth, x, y int) PixelWinner {
	var best Opinion
	bestRank := -1
	found := false
	for _, opinion := range scene.Opinions {
		if !opinion.Shape.Contains(x, y) {
			continue
		}
		rank, eligible := effectiveRank(opinion, mode, maxDepth)
		if !eligible {
			continue
		}
		if !found || rank > bestRank || (rank == bestRank && opinion.ID < best.ID) {
			best = opinion
			bestRank = rank
			found = true
		}
	}
	if !found {
		diagnostic := unresolvedOpinion(x, y)
		return PixelWinner{X: x, Y: y, Opinion: diagnostic, Rank: -1, Diagnostic: true}
	}
	return PixelWinner{X: x, Y: y, Opinion: best, Rank: bestRank}
}

func resolveScene(scene Scene, mode ModeSpec) []PixelWinner {
	maxDepth := maxInheritedDepth(scene)
	winners := make([]PixelWinner, 0, scene.Width*scene.Height)
	for y := 0; y < scene.Height; y++ {
		for x := 0; x < scene.Width; x++ {
			winners = append(winners, resolvePixel(scene, mode, maxDepth, x, y))
		}
	}
	return winners
}

func pixel(winners []PixelWinner, scene Scene, x, y int) PixelWinner {
	return winners[y*scene.Width+x]
}

func opinionKey(opinion Opinion) string {
	return fmt.Sprintf("%s|%s|%s|%s", opinion.ID, opinion.Layer, opinion.Material.ID, opinion.Material.Color.Hex())
}

func sortedOpinionKeys(opinions map[string]Opinion) []string {
	keys := make([]string, 0, len(opinions))
	for key := range opinions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
