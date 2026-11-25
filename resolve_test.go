package main

import "testing"

func TestInheritanceChainDefinesDepths(t *testing.T) {
	scene := demoScene()
	depths := inheritanceDepths(scene.Chain, scene.InstanceOf)
	want := map[string]int{"vessel": 0, "alloy": 1, "stone": 2}
	for class, depth := range want {
		if got := depths[class]; got != depth {
			t.Fatalf("class %s depth = %d, want %d", class, got, depth)
		}
	}
}

func TestCorrectCascadeUsesInstanceAboveInherited(t *testing.T) {
	scene := demoScene()
	mode, _ := modeByID("after_override")
	winners := resolveScene(scene, mode)

	cases := []struct {
		name string
		x, y int
		want string
	}{
		{name: "outer margin falls to root", x: 5, y: 5, want: "inherited:stone.field"},
		{name: "panel uses nearest ancestor", x: 20, y: 20, want: "inherited:alloy.panel"},
		{name: "instance beats both inherited opinions", x: 80, y: 120, want: "instance:vessel.ember"},
		{name: "halo beats instance", x: 92, y: 60, want: "local:halo"},
		{name: "core beats halo", x: 92, y: 86, want: "local:core"},
		{name: "spark beats instance", x: 160, y: 65, want: "local:spark"},
		{name: "slit beats instance", x: 140, y: 100, want: "local:slit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pixel(winners, scene, tc.x, tc.y).Opinion.ID
			if got != tc.want {
				t.Fatalf("pixel (%d,%d) winner = %q, want %q", tc.x, tc.y, got, tc.want)
			}
		})
	}

	instance := pixel(winners, scene, 80, 120)
	if instance.Rank != 300 {
		t.Fatalf("instance rank = %d, want 300", instance.Rank)
	}
	panel := pixel(winners, scene, 20, 20)
	if panel.Rank != 201 {
		t.Fatalf("nearest inherited rank = %d, want 201", panel.Rank)
	}
}

func TestBeforeInheritanceLeavesDiagnostics(t *testing.T) {
	scene := demoScene()
	mode, _ := modeByID("before_inheritance")
	winners := resolveScene(scene, mode)

	outer := pixel(winners, scene, 5, 5)
	if !outer.Diagnostic || outer.Opinion.Layer != LayerDiagnostic {
		t.Fatalf("outer pixel = %+v, want a diagnostic winner", outer.Opinion)
	}
	body := pixel(winners, scene, 80, 120)
	if body.Opinion.ID != "instance:vessel.ember" {
		t.Fatalf("body pixel winner = %q, want instance", body.Opinion.ID)
	}
}

func TestForcedFallbackInvertsOnlyInstanceAgainstInherited(t *testing.T) {
	scene := demoScene()
	mode, _ := modeByID("forced_fallback")
	winners := resolveScene(scene, mode)

	body := pixel(winners, scene, 80, 120)
	if body.Opinion.ID != "inherited:alloy.panel" {
		t.Fatalf("body pixel winner = %q, want inherited panel", body.Opinion.ID)
	}
	if body.Rank != 301 {
		t.Fatalf("forced inherited rank = %d, want 301", body.Rank)
	}
	core := pixel(winners, scene, 92, 86)
	if core.Opinion.ID != "local:core" {
		t.Fatalf("local core = %q, want local:core to remain strongest", core.Opinion.ID)
	}
}
