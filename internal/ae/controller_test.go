package ae

import (
	"math"
	"testing"
)

func TestFastSmoothProfileConvergesWithoutOvershoot(t *testing.T) {
	controller := NewController(profileByID("fast_smooth"))
	const initial = 1.24
	const target = -0.98
	controller.Reset(initial)

	maxOvershoot := 0.0
	for frame := 0; frame < 300; frame++ {
		ev := controller.Update(target, 1.0/30.0)
		maxOvershoot = math.Max(maxOvershoot, target-ev)
	}

	if maxOvershoot > 0.035 {
		t.Fatalf("fast_smooth overshoot = %.3f EV, want <= 0.035", maxOvershoot)
	}
	if math.Abs(controller.EV-target) > 0.001 {
		t.Fatalf("final EV = %.6f, target %.6f", controller.EV, target)
	}
}

func TestRapidProfileOvershootsAndRings(t *testing.T) {
	controller := NewController(profileByID("rapid_breathing"))
	const target = -0.98
	controller.Reset(1.24)

	maxOvershoot := 0.0
	crossings := 0
	side := 1
	for frame := 0; frame < 240; frame++ {
		ev := controller.Update(target, 1.0/30.0)
		maxOvershoot = math.Max(maxOvershoot, target-ev)
		deviation := ev - target
		if deviation > 0.02 && side < 0 {
			crossings++
			side = 1
		} else if deviation < -0.02 && side > 0 {
			crossings++
			side = -1
		}
	}

	if maxOvershoot < 0.20 {
		t.Fatalf("rapid max overshoot = %.3f EV, want >= 0.20", maxOvershoot)
	}
	if crossings < 2 {
		t.Fatalf("rapid significant crossings = %d, want at least 2", crossings)
	}
}

func TestResponseOrderingEarlyInTransition(t *testing.T) {
	target := -0.98
	errors := map[string]float64{}
	for _, id := range []string{"rapid_breathing", "fast_smooth", "gentle_slow"} {
		controller := NewController(profileByID(id))
		controller.Reset(1.24)
		for frame := 0; frame < 34; frame++ {
			controller.Update(target, 1.0/30.0)
		}
		errors[id] = math.Abs(controller.EV - target)
	}
	if errors["fast_smooth"] >= errors["gentle_slow"] {
		t.Fatalf("fast error %.3f should be below slow error %.3f", errors["fast_smooth"], errors["gentle_slow"])
	}
}

func profileByID(id string) Profile {
	for _, profile := range Profiles {
		if profile.ID == id {
			return profile
		}
	}
	panic("unknown profile " + id)
}
