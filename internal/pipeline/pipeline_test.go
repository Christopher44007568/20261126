package pipeline

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func producerPass(name, res string) *Pass {
	return &Pass{
		Name:   name,
		Writes: []string{res},
		Fn: func(frame *Frame) error {
			frame.Ensure(res, 1)
			return nil
		},
	}
}

func consumerPass(name, in, out string) *Pass {
	return &Pass{
		Name:   name,
		Reads:  []string{in},
		Writes: []string{out},
		Fn: func(frame *Frame) error {
			if _, err := frame.Get(in); err != nil {
				return err
			}
			frame.Ensure(out, 1)
			return nil
		},
	}
}

func TestPlanAndRunOrder(t *testing.T) {
	p := New("test", 4, 3)
	for _, pass := range []*Pass{
		producerPass("geometry", ResDepth),
		consumerPass("shadow", ResDepth, ResShadow),
	} {
		if err := p.Add(pass); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := p.Plan()
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	want := []string{"geometry", "shadow"}
	if got := plan.Order(); got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("order = %v, want %v", got, want)
	}
	frame := p.NewFrame()
	if err := plan.Run(frame); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := frame.Get(ResShadow); err != nil {
		t.Fatalf("shadow buffer missing: %v", err)
	}
}

func TestRejectsBufferWithoutProducer(t *testing.T) {
	p := New("test", 2, 2)
	if err := p.Add(consumerPass("shadow", ResDepth, ResShadow)); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Plan(); err == nil ||
		!strings.Contains(err.Error(), "no pass produces") {
		t.Fatalf("want missing-producer error, got %v", err)
	}
}

func TestRejectsDuplicateWriter(t *testing.T) {
	p := New("test", 2, 2)
	if err := p.Add(producerPass("a", ResDepth)); err != nil {
		t.Fatal(err)
	}
	if err := p.Add(producerPass("b", ResDepth)); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Plan(); err == nil ||
		!strings.Contains(err.Error(), "written by both") {
		t.Fatalf("want duplicate-writer error, got %v", err)
	}
}

func TestRejectsCycleAndDoesNotExecute(t *testing.T) {
	executed := map[string]bool{}
	p := New("test", 2, 2)
	must := func(pass *Pass) {
		if err := p.Add(pass); err != nil {
			t.Fatal(err)
		}
	}
	must(&Pass{
		Name:   "A",
		Reads:  []string{"b"},
		Writes: []string{"a"},
		Fn:     func(*Frame) error { executed["A"] = true; return nil },
	})
	must(&Pass{
		Name:   "B",
		Reads:  []string{"a"},
		Writes: []string{"b"},
		Fn:     func(*Frame) error { executed["B"] = true; return nil },
	})

	_, err := p.Plan()
	var cycleErr *CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("want CycleError, got %v", err)
	}
	if len(executed) != 0 {
		t.Fatalf("no pass may execute on a cyclic graph, ran %v", executed)
	}
}

func TestRunStopsOnPassFailure(t *testing.T) {
	p := New("test", 2, 2)
	must := func(pass *Pass) {
		if err := p.Add(pass); err != nil {
			t.Fatal(err)
		}
	}
	must(producerPass("ok", "a"))
	must(&Pass{
		Name:   "boom",
		Reads:  []string{"a"},
		Writes: []string{"b"},
		Fn:     func(*Frame) error { return errors.New("boom") },
	})
	plan, err := p.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Run(p.NewFrame()); err == nil {
		t.Fatal("Run must propagate pass failure")
	}
}

func TestPPMRoundTrip(t *testing.T) {
	buf := &Buffer{
		Name:     "c",
		Width:    2,
		Height:   2,
		Channels: 3,
		Data: []float64{
			1, 0, 0, 0, 1, 0,
			0, 0, 1, 1, 1, 1,
		},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "out.ppm")
	if err := WritePPM(path, buf); err != nil {
		t.Fatalf("WritePPM: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "P6\n2 2\n255\n") {
		t.Fatalf("bad PPM header: %q", data[:12])
	}
	payload := data[len("P6\n2 2\n255\n"):]
	want := []byte{255, 0, 0, 0, 255, 0, 0, 0, 255, 255, 255, 255}
	if string(payload) != string(want) {
		t.Fatalf("payload = %v, want %v", payload, want)
	}
}

func TestEncodePPMRejectsSingleChannel(t *testing.T) {
	buf := &Buffer{Name: "d", Width: 1, Height: 1, Channels: 1, Data: []float64{0.5}}
	if _, err := EncodePPM(buf); err == nil {
		t.Fatal("single-channel buffer must not be encodable as PPM")
	}
}
