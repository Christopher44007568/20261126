package main

import (
	"strings"
	"testing"
)

func mustLayer(t *testing.T, id, file string, strength int, body string) *Layer {
	t.Helper()
	src := `#usda 1.0
(
    layerId = "` + id + `"
    layerStrength = ` + formatFloat(float64(strength)) + `
)
` + body
	layer, err := ParseUSDA(id, file, []byte(src))
	if err != nil {
		t.Fatalf("parse %s: %v", id, err)
	}
	return layer
}

func TestStrongerLayerWinsRegardlessOfLoadOrder(t *testing.T) {
	weak := mustLayer(t, "weak", "weak.usda", 10, `
def Mesh "Ball" {
    color3f inputs:displayColor = (1, 0, 0)
}
`)
	strong := mustLayer(t, "strong", "strong.usda", 40, `
over Mesh "Ball" {
    color3f inputs:displayColor = (0, 0, 1)
}
`)
	for _, layers := range [][]*Layer{
		{weak, strong},
		{strong, weak},
	} {
		scene := Compose(layers)
		winner := scene.Winner("/Ball", "inputs:displayColor")
		if winner == nil || winner.LayerID != "strong" {
			t.Fatalf("winner = %#v, want strong", winner)
		}
		if winner.Value != "(0, 0, 1)" {
			t.Fatalf("winner value = %q", winner.Value)
		}
	}
}

func TestLocalOverKeepsChildPath(t *testing.T) {
	base := mustLayer(t, "base", "base.usda", 10, `
def Xform "Stage" {
    def Mesh "Panel" {
        color3f inputs:displayColor = (0.2, 0.2, 0.2)
        def Sphere "Knob" {
            color3f inputs:displayColor = (0.2, 0.2, 0.2)
        }
    }
}
`)
	over := mustLayer(t, "over", "over.usda", 20, `
over Xform "Stage" {
    over Mesh "Panel" {
        over Sphere "Knob" {
            color3f inputs:displayColor = (0, 1, 0)
        }
    }
}
`)
	scene := Compose([]*Layer{over, base})
	if scene.Prims["/Stage/Panel/Knob"] == nil {
		t.Fatal("missing /Stage/Panel/Knob")
	}
	if scene.Prims["/Stage/Panel/Knob/Knob"] != nil {
		t.Fatal("local over introduced a relocated child path")
	}
	winner := scene.Winner("/Stage/Panel/Knob", "inputs:displayColor")
	if winner == nil || winner.Value != "(0, 1, 0)" {
		t.Fatalf("child winner = %#v", winner)
	}
}

func TestEqualStrengthTieBreakIsDeterministic(t *testing.T) {
	a := mustLayer(t, "A-layer", "a.usda", 20, `
def Mesh "Ball" {
    color3f inputs:displayColor = (1, 0, 0)
}
`)
	b := mustLayer(t, "B-layer", "b.usda", 20, `
over Mesh "Ball" {
    color3f inputs:displayColor = (0, 1, 0)
}
`)
	one := Compose([]*Layer{a, b}).Winner("/Ball", "inputs:displayColor")
	two := Compose([]*Layer{b, a}).Winner("/Ball", "inputs:displayColor")
	if one.LayerID != "A-layer" || two.LayerID != "A-layer" {
		t.Fatalf("tie break changed with load order: %s vs %s", one.LayerID, two.LayerID)
	}
}

func TestPermutationOutputsAreByteIdentical(t *testing.T) {
	files, err := WriteSourceLayers(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	result, err := runPermutationCheck(files)
	if err != nil {
		t.Fatal(err)
	}
	if !result.allIdentical {
		t.Fatal("load order changed merged scene or PPM")
	}
	if string(result.forwardMerged) != string(result.reverseMerged) {
		t.Fatal("forward and reverse merged scenes differ")
	}
	if string(result.forwardPPM) != string(result.reversePPM) {
		t.Fatal("forward and reverse PPMs differ")
	}
}

func TestMergedSceneMarksWinningSource(t *testing.T) {
	files, err := WriteSourceLayers(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	layers, err := LoadLayerFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	merged := string(RenderMergedScene(Compose(layers)))
	if !strings.Contains(merged, "strongest opinion: L4-spot") {
		t.Fatal("merged scene does not identify the strongest opinion source")
	}
	if !strings.Contains(merged, "path=/Stage/PanelA/Knob") {
		t.Fatal("merged scene lost the nested child path")
	}
}
