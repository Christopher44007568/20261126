package aliaslab

import (
	"fmt"
	"sort"
	"strings"
)

type LifetimePolicy uint8

const (
	// ReadWriteLifetime is the correct policy: a transient dies after its last
	// read or write.
	ReadWriteLifetime LifetimePolicy = iota
	// WritesOnlyLifetime is deliberately wrong. It treats a resource as dead
	// after its last producer write and ignores later consumers.
	WritesOnlyLifetime
)

func (p LifetimePolicy) String() string {
	if p == WritesOnlyLifetime {
		return "writes-only (fault injected)"
	}
	return "last-read-or-write"
}

type Lifetime struct {
	First  int
	Last   int
	Reads  []int
	Writes []int
}

type Interval struct {
	Resource     ResourceID
	Actual       Lifetime
	Declared     Lifetime
	Policy       LifetimePolicy
	ExposedReads []int
}

type AliasMode string

const (
	ModeNoReuse AliasMode = "no_reuse"
	ModeValid   AliasMode = "valid_reuse"
	ModeInvalid AliasMode = "error_reuse"
)

type ResourceAllocation struct {
	Resource   ResourceID
	Slot       int
	FirstPass  int
	LastPass   int
	SharedWith []ResourceID
}

type SlotUser struct {
	Resource    ResourceID
	FirstPass   int
	LastPass    int
	ActualFirst int
	ActualLast  int
}

type SlotAllocation struct {
	Slot  int
	Users []SlotUser
}

type Violation struct {
	Kind     string
	Pass     int
	Resource ResourceID
	Message  string
}

type Plan struct {
	Mode        AliasMode
	Policy      LifetimePolicy
	Intervals   []Interval
	Allocations []ResourceAllocation
	Slots       []SlotAllocation
	SlotCount   int
	ReuseCount  int
	Violations  []Violation
	slotByID    map[ResourceID]int
}

func (p Plan) SlotFor(id ResourceID) int {
	return p.slotByID[id]
}

func AnalyzeIntervals(g *Graph, policy LifetimePolicy) []Interval {
	byID := make(map[ResourceID]*Interval, len(g.Resources))
	order := make([]ResourceID, 0, len(g.Resources))
	for _, r := range g.Resources {
		byID[r.ID] = &Interval{Resource: r.ID, Policy: policy}
		order = append(order, r.ID)
	}
	for _, a := range g.Accesses {
		in := byID[a.Resource]
		if a.Kind == Read {
			in.Actual.Reads = append(in.Actual.Reads, a.Pass)
		} else {
			in.Actual.Writes = append(in.Actual.Writes, a.Pass)
		}
	}
	out := make([]Interval, 0, len(order))
	for _, id := range order {
		in := byID[id]
		in.Actual.Reads = uniqueSorted(in.Actual.Reads)
		in.Actual.Writes = uniqueSorted(in.Actual.Writes)
		in.Actual.First, in.Actual.Last = span(in.Actual)
		if policy == WritesOnlyLifetime {
			in.Declared = Lifetime{
				First:  in.Actual.Writes[0],
				Last:   in.Actual.Writes[len(in.Actual.Writes)-1],
				Writes: in.Actual.Writes,
			}
			for _, pass := range in.Actual.Reads {
				if pass < in.Declared.First || pass > in.Declared.Last {
					in.ExposedReads = append(in.ExposedReads, pass)
				}
			}
		} else {
			in.Declared = in.Actual
		}
		out = append(out, *in)
	}
	return out
}

func span(l Lifetime) (int, int) {
	first, last := -1, -1
	for _, p := range l.Reads {
		if first < 0 || p < first {
			first = p
		}
		if p > last {
			last = p
		}
	}
	for _, p := range l.Writes {
		if first < 0 || p < first {
			first = p
		}
		if p > last {
			last = p
		}
	}
	return first, last
}

func BuildPlan(g *Graph, mode AliasMode) (Plan, error) {
	policy := ReadWriteLifetime
	if mode == ModeInvalid {
		policy = WritesOnlyLifetime
	}
	intervals := AnalyzeIntervals(g, policy)
	plan := Plan{
		Mode:      mode,
		Policy:    policy,
		Intervals: intervals,
		slotByID:  make(map[ResourceID]int),
	}

	type slotState struct {
		last int
		uses []SlotUser
	}
	var states []slotState
	for _, in := range intervals {
		slot := -1
		if mode == ModeNoReuse {
			slot = len(states)
			states = append(states, slotState{last: -1})
		} else {
			for j := range states {
				if states[j].last < in.Declared.First {
					slot = j
					break
				}
			}
			if slot < 0 {
				slot = len(states)
				states = append(states, slotState{last: -1})
			}
		}
		states[slot].last = in.Declared.Last
		states[slot].uses = append(states[slot].uses, SlotUser{
			Resource:    in.Resource,
			FirstPass:   in.Declared.First,
			LastPass:    in.Declared.Last,
			ActualFirst: in.Actual.First,
			ActualLast:  in.Actual.Last,
		})
		plan.slotByID[in.Resource] = slot
	}

	plan.SlotCount = len(states)
	plan.ReuseCount = len(intervals) - plan.SlotCount
	for slot, state := range states {
		plan.Slots = append(plan.Slots, SlotAllocation{
			Slot:  slot,
			Users: append([]SlotUser(nil), state.uses...),
		})
	}
	for _, in := range intervals {
		alloc := ResourceAllocation{
			Resource:  in.Resource,
			Slot:      plan.slotByID[in.Resource],
			FirstPass: in.Declared.First,
			LastPass:  in.Declared.Last,
		}
		for _, user := range states[alloc.Slot].uses {
			if user.Resource != in.Resource {
				alloc.SharedWith = append(alloc.SharedWith, user.Resource)
			}
		}
		plan.Allocations = append(plan.Allocations, alloc)
	}
	if err := validatePlan(&plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func validatePlan(plan *Plan) error {
	for _, in := range plan.Intervals {
		for _, pass := range in.ExposedReads {
			plan.Violations = append(plan.Violations, Violation{
				Kind:     "read_after_lifetime",
				Pass:     pass,
				Resource: in.Resource,
				Message: fmt.Sprintf(
					"%s read in pass %d, outside declared lifetime [%d,%d]",
					in.Resource, pass+1, in.Declared.First+1, in.Declared.Last+1),
			})
		}
	}
	for slotIndex, slot := range plan.Slots {
		for i := 0; i < len(slot.Users); i++ {
			for j := i + 1; j < len(slot.Users); j++ {
				a, b := slot.Users[i], slot.Users[j]
				if intervalsOverlap(a, b) {
					plan.Violations = append(plan.Violations, Violation{
						Kind:     "overlapping_alias",
						Pass:     maxInt(a.FirstPass, b.FirstPass),
						Resource: a.Resource,
						Message: fmt.Sprintf(
							"slot %d aliases %s [%d,%d] with %s [%d,%d]; their actual lifetimes overlap",
							slotIndex, a.Resource, a.ActualFirst+1, a.ActualLast+1,
							b.Resource, b.ActualFirst+1, b.ActualLast+1),
					})
				}
			}
		}
	}
	sort.SliceStable(plan.Violations, func(i, j int) bool {
		if plan.Violations[i].Pass != plan.Violations[j].Pass {
			return plan.Violations[i].Pass < plan.Violations[j].Pass
		}
		return plan.Violations[i].Resource < plan.Violations[j].Resource
	})
	if plan.Mode == ModeValid && len(plan.Violations) > 0 {
		return fmt.Errorf("valid plan has %d lifetime violations", len(plan.Violations))
	}
	return nil
}

func intervalsOverlap(a, b SlotUser) bool {
	return a.ActualFirst <= b.ActualLast && b.ActualFirst <= a.ActualLast
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (p Plan) AliasLogText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "mode=%s policy=%s slots=%d resources=%d reused=%d\n",
		p.Mode, p.Policy, p.SlotCount, len(p.Intervals), p.ReuseCount)
	for _, slot := range p.Slots {
		fmt.Fprintf(&b, "slot %d:", slot.Slot)
		for _, u := range slot.Users {
			fmt.Fprintf(&b, " %s[%d..%d]", u.Resource, u.FirstPass+1, u.LastPass+1)
		}
		b.WriteByte('\n')
	}
	if len(p.Violations) > 0 {
		b.WriteString("violations:\n")
		for _, v := range p.Violations {
			fmt.Fprintf(&b, "  pass %d %s: %s\n", v.Pass+1, v.Kind, v.Message)
		}
	} else {
		b.WriteString("violations: none\n")
	}
	return b.String()
}
