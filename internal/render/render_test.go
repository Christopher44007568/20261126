package render

import (
	"bytes"
	"encoding/xml"
	"testing"

	"instancelab/internal/lab"
)

func TestAllDiagramsRender(t *testing.T) {
	doc := lab.BuildScenario()
	proto, err := doc.Prototype("proto-beacon-card")
	if err != nil {
		t.Fatal(err)
	}
	sharedFault, err := lab.SimulateSharedMapFault("I-037", "body", "#FF2E93")
	if err != nil {
		t.Fatal(err)
	}
	writebackFault, err := lab.SimulateWritebackFault("I-004", "body", "#FF2E93")
	if err != nil {
		t.Fatal(err)
	}
	isolation, err := lab.CheckColorIsolation(lab.BuildScenario(), "I-007", "body", "#FF2E93")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		fn   func(*bytes.Buffer) error
	}{
		{"prototype", func(b *bytes.Buffer) error { return WritePrototypeDiagram(b, proto) }},
		{"overview", func(b *bytes.Buffer) error { return WriteInstancesOverview(b, doc) }},
		{"shared", func(b *bytes.Buffer) error { return WriteSharedColorError(b, doc, sharedFault) }},
		{"writeback", func(b *bytes.Buffer) error { return WriteWritebackError(b, doc, writebackFault) }},
		{"fixed", func(b *bytes.Buffer) error { return WriteFixedIsolation(b, doc, isolation) }},
		{"geometry", func(b *bytes.Buffer) error {
			return WriteBoundsOutlineSync(b, lab.BuildScenario(), "I-001", 24, 18)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tc.fn(&buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			if buf.Len() == 0 {
				t.Fatal("empty svg output")
			}
			var v any
			if err := xml.Unmarshal(buf.Bytes(), &v); err != nil {
				t.Fatalf("invalid xml: %v", err)
			}
		})
	}
}
