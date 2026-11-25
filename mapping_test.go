package main

import (
	"testing"
)

func TestCorrectMappingCoversScene(t *testing.T) {
	scene := NewScene()
	mapping := CorrectMapping()
	if err := mapping.Validate(scene); err != nil {
		t.Fatalf("correct mapping invalid: %v", err)
	}
	if len(mapping) != len(scene.Objects) {
		t.Fatalf("mapping has %d entries for %d objects", len(mapping), len(scene.Objects))
	}
	checks := map[ObjectID]HitGroupID{
		ObjectStoneWall:  GroupStone,
		ObjectGlassPanel: GroupGlass,
		ObjectAreaLight:  GroupLight,
	}
	for objectID, want := range checks {
		if got := mapping[objectID]; got != want {
			t.Errorf("object %d maps to hit group %d, want %d", objectID, got, want)
		}
	}
	stone, _ := scene.Catalog.MaterialForGroup(GroupStone)
	glass, _ := scene.Catalog.MaterialForGroup(GroupGlass)
	if stone.ID == glass.ID {
		t.Fatal("stone and glass must have distinct material indices")
	}
}

func TestSingleRemapChangesOnlyStoneWall(t *testing.T) {
	scene := NewScene()
	correct := CorrectMapping()
	broken, err := RemapOne(correct, ObjectStoneWall, GroupGlass)
	if err != nil {
		t.Fatal(err)
	}
	if err := broken.Validate(scene); err != nil {
		t.Fatalf("broken mapping should still be structurally valid: %v", err)
	}
	changes := MappingChanges(scene, correct, broken)
	if len(changes) != 1 {
		t.Fatalf("got %d mapping changes, want exactly 1", len(changes))
	}
	change := changes[0]
	if change.ObjectID != ObjectStoneWall || change.CorrectGroup != GroupStone || change.BrokenGroup != GroupGlass {
		t.Fatalf("unexpected change: %+v", change)
	}
	if correct[ObjectStoneWall] != GroupStone {
		t.Fatal("CorrectMapping was mutated")
	}
}

func TestFirstMismatchLocatesObjectAndHitGroup(t *testing.T) {
	scene := NewScene()
	correct := CorrectMapping()
	broken, err := RemapOne(correct, ObjectStoneWall, GroupGlass)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := Render(scene, correct, 192, 120)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := Render(scene, broken, 192, 120)
	if err != nil {
		t.Fatal(err)
	}
	first, err := FindFirstMismatch(scene, reference, actual)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Found {
		t.Fatal("expected a visible mapping error")
	}
	if first.ObjectID != ObjectStoneWall || first.ObjectName != "back_stone_wall" {
		t.Fatalf("first error object = %d %q, want stone wall", first.ObjectID, first.ObjectName)
	}
	if first.ExpectedHitGroupID != GroupStone || first.ExpectedHitGroup != "stone" {
		t.Fatalf("expected hit group = %d %q", first.ExpectedHitGroupID, first.ExpectedHitGroup)
	}
	if first.ActualHitGroupID != GroupGlass || first.ActualHitGroup != "glass" {
		t.Fatalf("actual hit group = %d %q", first.ActualHitGroupID, first.ActualHitGroup)
	}
	if first.ExpectedMaterialID != MaterialStone || first.ActualMaterialID != MaterialGlass {
		t.Fatalf("material mismatch was not reported: %+v", first)
	}

	for y := 0; y <= first.Y; y++ {
		limit := reference.Width
		if y == first.Y {
			limit = first.X
		}
		for x := 0; x < limit; x++ {
			if diff := pixelDifferences(reference.At(x, y), actual.At(x, y)); len(diff) != 0 {
				t.Fatalf("pixel (%d,%d) differs before reported first error (%d,%d): %v",
					x, y, first.X, first.Y, diff)
			}
		}
	}
}

func TestLightHitGroupDrivesIllumination(t *testing.T) {
	scene := NewScene()
	correct := CorrectMapping()
	lights := gatherLights(scene, correct)
	if len(lights) != 1 || lights[0].ObjectID != ObjectAreaLight {
		t.Fatalf("correct mapping should expose exactly the area light: %+v", lights)
	}
	broken, err := RemapOne(correct, ObjectAreaLight, GroupStone)
	if err != nil {
		t.Fatal(err)
	}
	if lights := gatherLights(scene, broken); len(lights) != 0 {
		t.Fatalf("light geometry remapped to stone still illuminates: %+v", lights)
	}
}
