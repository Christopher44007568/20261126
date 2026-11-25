package main

import "testing"

func TestProvenanceResolvesEveryPixelToItsWinningOpinion(t *testing.T) {
	scene := demoScene()
	for _, mode := range RenderModes {
		t.Run(mode.ID, func(t *testing.T) {
			winners := resolveScene(scene, mode)
			report := buildProvenance(scene, mode, winners)
			if len(report.WinnerSlots) != scene.Height {
				t.Fatalf("rows = %d, want %d", len(report.WinnerSlots), scene.Height)
			}
			for y := 0; y < scene.Height; y++ {
				if len(report.WinnerSlots[y]) != scene.Width {
					t.Fatalf("row %d width = %d, want %d", y, len(report.WinnerSlots[y]), scene.Width)
				}
				for x := 0; x < scene.Width; x++ {
					winner := pixel(winners, scene, x, y)
					slot := report.WinnerSlots[y][x]
					if slot < 0 || slot >= len(report.Opinions) {
						t.Fatalf("pixel (%d,%d) slot %d outside opinion table", x, y, slot)
					}
					opinion := report.Opinions[slot]
					if opinion.ID != winner.Opinion.ID || opinion.Color != winner.Opinion.Material.Color.Hex() {
						t.Fatalf("pixel (%d,%d) traces to %s/%s, winner is %s/%s",
							x, y, opinion.ID, opinion.Color, winner.Opinion.ID, winner.Opinion.Material.Color.Hex())
					}
					if opinion.EffectiveRank != winner.Rank {
						t.Fatalf("pixel (%d,%d) rank = %d, opinion table says %d", x, y, winner.Rank, opinion.EffectiveRank)
					}
				}
			}
		})
	}
}

func TestForcedFallbackErrorsUseInheritedFinalOpinion(t *testing.T) {
	scene := demoScene()
	reference, _ := modeByID("after_override")
	mode, _ := modeByID("forced_fallback")
	winnersByMode := map[string][]PixelWinner{
		reference.ID: resolveScene(scene, reference),
		mode.ID:      resolveScene(scene, mode),
	}
	audit := auditRenders(reference, winnersByMode[reference.ID], []ModeSpec{mode}, winnersByMode)
	if err := validateAudit(audit); err != nil {
		t.Fatal(err)
	}
	if len(audit.Modes) != 1 || audit.Modes[0].WrongColorPixels == 0 {
		t.Fatalf("audit = %+v, want forced fallback errors", audit.Modes)
	}
	for _, group := range audit.Modes[0].Groups {
		if group.ActualLayer != LayerInherited {
			t.Fatalf("actual layer = %s, want inherited", group.ActualLayer)
		}
		if group.ExpectedLayer != LayerInstance {
			t.Fatalf("expected layer = %s, want instance", group.ExpectedLayer)
		}
		for _, index := range group.PixelIndices {
			actual := winnersByMode[mode.ID][index]
			expected := winnersByMode[reference.ID][index]
			if actual.Opinion.ID != group.ActualOpinionID || actual.Opinion.Material.Color.Hex() != group.ActualColor {
				t.Fatalf("pixel %d actual %s/%s, group says %s/%s", index,
					actual.Opinion.ID, actual.Opinion.Material.Color.Hex(), group.ActualOpinionID, group.ActualColor)
			}
			if expected.Opinion.ID != group.ExpectedOpinionID {
				t.Fatalf("pixel %d expected %s, group says %s", index, expected.Opinion.ID, group.ExpectedOpinionID)
			}
		}
	}
}
