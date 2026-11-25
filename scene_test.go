package main

import "testing"

// A ray crossing a slab from outside must report exactly one entry followed
// by one exit event, with the exit at the true far-face distance.
func TestSlabEntryExitOrder(t *testing.T) {
	glass := &Glass{ID: 1, IOR: 1.5}
	slab := &Slab{Min: Vec3{-1, -1, -1}, Max: Vec3{1, 1, 1}, G: glass}
	scene := &Scene{Primitives: []Primitive{slab}}

	r := Ray{Vec3{0, 0, 5}, Vec3{0, 0, -1}}
	hits := scene.AllHits(r)
	if len(hits) != 2 {
		t.Fatalf("want 2 slab events, got %d: %+v", len(hits), hits)
	}
	if !hits[0].Entering || hits[1].Entering {
		t.Fatalf("event order wrong: %+v", hits)
	}
	if !approx(hits[0].T, 4) || !approx(hits[1].T, 6) {
		t.Fatalf("event distances wrong: %+v", hits)
	}

	// A ray starting inside must only produce the exit event.
	inside := Ray{Vec3{0, 0, 0}, Vec3{0, 0, -1}}
	events := scene.AllHits(inside)
	if len(events) != 1 || events[0].Entering {
		t.Fatalf("inside-ray must see exit only, got %+v", events)
	}
}

// Coincident opaque and glass surfaces resolve to the opaque hit first, so a
// glass boundary can never mask a hard occluder.
func TestTieBreakOpaqueWins(t *testing.T) {
	glass := &Glass{ID: 1, IOR: 1.5}
	scene := &Scene{
		Primitives: []Primitive{
			&Slab{Min: Vec3{-1, -1, -2}, Max: Vec3{1, 1, -1}, G: glass},
			&Sphere{Center: Vec3{0, 0, -3}, Radius: 2, Albedo: Vec3{0.5, 0.5, 0.5}},
		},
	}
	first, ok := scene.NearestHit(Ray{Vec3{0, 0, 4}, Vec3{0, 0, -1}})
	if !ok {
		t.Fatal("expected a hit")
	}
	if first.Glass != nil {
		t.Fatal("opaque coincident surface must win the nearest-hit tie")
	}
}
