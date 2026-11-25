package pipeline

import (
	"errors"
	"reflect"
	"testing"
)

func TestTopoSortDiamond(t *testing.T) {
	g := NewGraph([]string{"A", "B", "C", "D"})
	for _, edge := range [][2]string{
		{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"},
	} {
		if err := g.AddEdge(edge[0], edge[1]); err != nil {
			t.Fatalf("AddEdge: %v", err)
		}
	}
	order, err := g.TopoSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pos := map[string]int{}
	for i, name := range order {
		pos[name] = i
	}
	if !(pos["A"] < pos["B"] && pos["B"] < pos["D"] &&
		pos["A"] < pos["C"] && pos["C"] < pos["D"]) {
		t.Fatalf("topological order violated: %v", order)
	}
	// 同层节点按名字排序，结果确定可复现。
	want := []string{"A", "B", "C", "D"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestCyclePathComplete(t *testing.T) {
	g := NewGraph([]string{"A", "B", "C"})
	for _, edge := range [][2]string{
		{"A", "B"}, {"B", "C"}, {"C", "A"},
	} {
		if err := g.AddEdge(edge[0], edge[1]); err != nil {
			t.Fatalf("AddEdge: %v", err)
		}
	}
	_, err := g.TopoSort()
	var cycleErr *CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("want CycleError, got %v", err)
	}
	first, last := cycleErr.Path[0], cycleErr.Path[len(cycleErr.Path)-1]
	if first != last {
		t.Fatalf("cycle path must be closed: %v", cycleErr.Path)
	}
	if got := FormatPath(cycleErr.Path); got != "A -> B -> C -> A" {
		t.Fatalf("path = %q", got)
	}
}

func TestSelfLoopIsCycle(t *testing.T) {
	g := NewGraph([]string{"A"})
	if err := g.AddEdge("A", "A"); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if g.CyclePath() == nil {
		t.Fatal("self loop must be detected as a cycle")
	}
}

func TestAddEdgeRejectsUnknownNode(t *testing.T) {
	g := NewGraph([]string{"A"})
	if err := g.AddEdge("A", "X"); err == nil {
		t.Fatal("edge to unknown node must fail")
	}
}

func TestDuplicateEdgeIgnored(t *testing.T) {
	g := NewGraph([]string{"A", "B"})
	if err := g.AddEdge("A", "B"); err != nil {
		t.Fatal(err)
	}
	if err := g.AddEdge("A", "B"); err != nil {
		t.Fatalf("duplicate edge should be ignored, got %v", err)
	}
	order, err := g.TopoSort()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 {
		t.Fatalf("order = %v", order)
	}
}
