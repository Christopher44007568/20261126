package aliaslab

import (
	"fmt"
	"sort"
)

type ResourceID string

type AccessKind uint8

const (
	Read AccessKind = iota
	Write
)

func (k AccessKind) String() string {
	if k == Read {
		return "R"
	}
	return "W"
}

type Resource struct {
	ID     ResourceID
	Width  int
	Height int
	Format string
}

type Access struct {
	Pass     int
	Resource ResourceID
	Kind     AccessKind
}

type Pass struct {
	ID     string
	Name   string
	Reads  []ResourceID
	Writes []ResourceID
	Apply  func(inputs map[ResourceID]*Image) *Image
}

// Graph is a deterministic sequence of full-screen image effects.
type Graph struct {
	Width     int
	Height    int
	Resources []Resource
	Passes    []Pass
	Accesses  []Access
}

func (g *Graph) Resource(id ResourceID) Resource {
	for _, r := range g.Resources {
		if r.ID == id {
			return r
		}
	}
	panic("unknown resource " + string(id))
}

func (g *Graph) ResourceIndex(id ResourceID) int {
	for i, r := range g.Resources {
		if r.ID == id {
			return i
		}
	}
	return -1
}

func NewGraph(width, height int, resources []Resource, passes []Pass) (*Graph, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid graph size %dx%d", width, height)
	}
	g := &Graph{Width: width, Height: height, Resources: resources, Passes: passes}
	seen := make(map[ResourceID]bool)
	for i := range g.Resources {
		r := &g.Resources[i]
		if r.ID == "" {
			return nil, fmt.Errorf("resource %d has empty id", i)
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("duplicate resource %q", r.ID)
		}
		seen[r.ID] = true
		if r.Width != width || r.Height != height {
			return nil, fmt.Errorf("resource %q is %dx%d, graph is %dx%d",
				r.ID, r.Width, r.Height, width, height)
		}
		if r.Format == "" {
			return nil, fmt.Errorf("resource %q has empty format", r.ID)
		}
	}

	written := make(map[ResourceID]bool)
	for passIndex := range g.Passes {
		p := &g.Passes[passIndex]
		if p.ID == "" || p.Name == "" {
			return nil, fmt.Errorf("pass %d has empty id or name", passIndex)
		}
		if p.Apply == nil {
			return nil, fmt.Errorf("pass %q has nil apply function", p.ID)
		}
		if len(p.Writes) != 1 {
			return nil, fmt.Errorf("pass %q must write exactly one resource", p.ID)
		}
		for _, id := range p.Reads {
			if !seen[id] {
				return nil, fmt.Errorf("pass %q reads unknown resource %q", p.ID, id)
			}
			g.Accesses = append(g.Accesses, Access{
				Pass: passIndex, Resource: id, Kind: Read,
			})
		}
		for _, id := range p.Writes {
			if !seen[id] {
				return nil, fmt.Errorf("pass %q writes unknown resource %q", p.ID, id)
			}
			written[id] = true
			g.Accesses = append(g.Accesses, Access{
				Pass: passIndex, Resource: id, Kind: Write,
			})
		}
	}
	for _, r := range g.Resources {
		if !written[r.ID] {
			return nil, fmt.Errorf("resource %q is never written", r.ID)
		}
	}
	return g, nil
}

// AccessLogText renders the complete read/write interval table.
func (g *Graph) AccessLogText(intervals []Interval) string {
	var b []byte
	b = append(b, []byte("pass  id            kind  resource\n")...)
	for _, a := range g.Accesses {
		kind := "write"
		if a.Kind == Read {
			kind = "read "
		}
		b = append(b, []byte(fmt.Sprintf("%4d  %-12s  %s   %s\n",
			a.Pass+1, g.Passes[a.Pass].ID, kind, a.Resource))...)
	}
	b = append(b, []byte("\nlifetime intervals (1-based pass numbers)\n")...)
	for _, in := range intervals {
		b = append(b, []byte(fmt.Sprintf(
			"%-8s actual[%d,%d] declared[%d,%d] reads=%v writes=%v\n",
			in.Resource, in.Actual.First+1, in.Actual.Last+1,
			in.Declared.First+1, in.Declared.Last+1,
			passNumbers(in.Actual.Reads), passNumbers(in.Actual.Writes)))...)
	}
	return string(b)
}

func sortPasses(passes []int) []int {
	out := append([]int(nil), passes...)
	sort.Ints(out)
	return out
}

func uniqueSorted(passes []int) []int {
	if len(passes) == 0 {
		return nil
	}
	out := sortPasses(passes)
	n := 1
	for i := 1; i < len(out); i++ {
		if out[i] != out[n-1] {
			out[n] = out[i]
			n++
		}
	}
	return out[:n]
}

func passNumbers(passes []int) []int {
	out := make([]int, len(passes))
	for i, p := range passes {
		out[i] = p + 1
	}
	return out
}
