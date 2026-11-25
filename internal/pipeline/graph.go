package pipeline

import (
	"fmt"
	"sort"
	"strings"
)

// Graph 是一个有向图。边 u -> v 表示节点 u 必须在 v 之前执行。
type Graph struct {
	nodes []string
	succ  map[string]map[string]struct{}
	pred  map[string]map[string]struct{}
}

// NewGraph 创建以 nodes 为初始节点集合的图，节点顺序会被稳定保留。
func NewGraph(nodes []string) *Graph {
	g := &Graph{
		succ: make(map[string]map[string]struct{}, len(nodes)),
		pred: make(map[string]map[string]struct{}, len(nodes)),
	}
	for _, n := range nodes {
		if _, ok := g.succ[n]; ok {
			continue
		}
		g.nodes = append(g.nodes, n)
		g.succ[n] = map[string]struct{}{}
		g.pred[n] = map[string]struct{}{}
	}
	return g
}

// AddEdge 添加一条有向边，边的端点必须已存在，重复边会被忽略。
func (g *Graph) AddEdge(from, to string) error {
	if _, ok := g.succ[from]; !ok {
		return fmt.Errorf("graph: unknown node %q", from)
	}
	if _, ok := g.succ[to]; !ok {
		return fmt.Errorf("graph: unknown node %q", to)
	}
	g.succ[from][to] = struct{}{}
	g.pred[to][from] = struct{}{}
	return nil
}

func (g *Graph) sortedNeighbors(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// CycleError 携带检测到的循环及其完整路径。
type CycleError struct {
	Path []string // 完整循环路径，首尾节点相同，例如 A -> B -> A
}

func (e *CycleError) Error() string {
	return "graph: cycle detected: " + FormatPath(e.Path)
}

// FormatPath 把节点路径渲染成 A -> B -> C 的形式。
func FormatPath(path []string) string {
	return strings.Join(path, " -> ")
}

// CyclePath 使用三色 DFS 查找一个完整循环。
// 无循环时返回 nil；有循环时返回首尾相同的节点路径。
func (g *Graph) CyclePath() []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(g.nodes))
	var stack []string

	var visit func(node string) []string
	visit = func(node string) []string {
		color[node] = gray
		stack = append(stack, node)
		for _, next := range g.sortedNeighbors(g.succ[node]) {
			switch color[next] {
			case white:
				if cycle := visit(next); cycle != nil {
					return cycle
				}
			case gray:
				start := 0
				for stack[start] != next {
					start++
				}
				cycle := append([]string(nil), stack[start:]...)
				return append(cycle, next)
			}
		}
		stack = stack[:len(stack)-1]
		color[node] = black
		return nil
	}

	for _, node := range g.nodes {
		if color[node] == white {
			if cycle := visit(node); cycle != nil {
				return cycle
			}
		}
	}
	return nil
}

// TopoSort 用 Kahn 算法给出确定性的拓扑顺序（同层节点按名字排序）。
// 图中存在循环时返回 CycleError，且不返回任何部分顺序。
func (g *Graph) TopoSort() ([]string, error) {
	indegree := make(map[string]int, len(g.nodes))
	ready := make([]string, 0)
	for _, node := range g.nodes {
		indegree[node] = len(g.pred[node])
		if indegree[node] == 0 {
			ready = append(ready, node)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(g.nodes))
	for len(ready) > 0 {
		node := ready[0]
		ready = ready[1:]
		order = append(order, node)

		newlyReady := make([]string, 0)
		for _, next := range g.sortedNeighbors(g.succ[node]) {
			indegree[next]--
			if indegree[next] == 0 {
				newlyReady = append(newlyReady, next)
			}
		}
		sort.Strings(newlyReady)
		ready = append(ready, newlyReady...)
		sort.Strings(ready)
	}

	if len(order) != len(g.nodes) {
		return nil, &CycleError{Path: g.CyclePath()}
	}
	return order, nil
}

// String 渲染人类可读的邻接表形式。
func (g *Graph) String() string {
	var b strings.Builder
	for _, node := range g.nodes {
		nexts := g.sortedNeighbors(g.succ[node])
		if len(nexts) == 0 {
			b.WriteString(node)
			b.WriteString(" -> (none)\n")
			continue
		}
		b.WriteString(node)
		b.WriteString(" -> ")
		b.WriteString(strings.Join(nexts, ", "))
		b.WriteByte('\n')
	}
	return b.String()
}

// DOT 渲染 Graphviz DOT 形式，方便外部工具可视化。
func (g *Graph) DOT() string {
	var b strings.Builder
	b.WriteString("digraph pipeline {\n")
	for _, node := range g.nodes {
		b.WriteString(fmt.Sprintf("  %q;\n", node))
	}
	for _, node := range g.nodes {
		for _, next := range g.sortedNeighbors(g.succ[node]) {
			b.WriteString(fmt.Sprintf("  %q -> %q;\n", node, next))
		}
	}
	b.WriteString("}\n")
	return b.String()
}
