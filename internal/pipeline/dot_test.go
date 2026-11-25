package pipeline

import (
	"strings"
	"testing"
)

func TestDOTRendersAllEdges(t *testing.T) {
	g := NewGraph([]string{"geometry", "shadow", "color", "lighting"})
	for _, edge := range [][2]string{
		{"geometry", "shadow"},
		{"shadow", "lighting"},
		{"color", "lighting"},
	} {
		if err := g.AddEdge(edge[0], edge[1]); err != nil {
			t.Fatal(err)
		}
	}
	dot := g.DOT()
	for _, want := range []string{
		"digraph pipeline {",
		`"geometry" -> "shadow";`,
		`"shadow" -> "lighting";`,
		`"color" -> "lighting";`,
		"}",
	} {
		if !strings.Contains(dot, want) {
			t.Fatalf("DOT missing %q:\n%s", want, dot)
		}
	}
}
