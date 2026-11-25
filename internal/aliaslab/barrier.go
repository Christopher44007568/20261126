package aliaslab

import (
	"fmt"
	"strings"
)

type Barrier struct {
	Pass         int
	PassID       string
	Slot         int
	FromResource ResourceID
	FromAccess   AccessKind
	FromPass     int
	ToResource   ResourceID
	ToAccess     AccessKind
	Kind         string
	Reason       string
}

type slotAccess struct {
	resource ResourceID
	kind     AccessKind
	pass     int
}

// BuildBarriers walks the declared pass order and emits the transition needed
// whenever an access observes a previous producer, or a physical slot is
// rebound to another transient resource.
func BuildBarriers(g *Graph, plan Plan) []Barrier {
	states := make(map[int]slotAccess)
	var barriers []Barrier
	for _, access := range g.Accesses {
		slot := plan.SlotFor(access.Resource)
		prev, exists := states[slot]
		if exists {
			kind, reason := barrierKind(prev, access)
			if kind != "" {
				barriers = append(barriers, Barrier{
					Pass:         access.Pass,
					PassID:       g.Passes[access.Pass].ID,
					Slot:         slot,
					FromResource: prev.resource,
					FromAccess:   prev.kind,
					FromPass:     prev.pass,
					ToResource:   access.Resource,
					ToAccess:     access.Kind,
					Kind:         kind,
					Reason:       reason,
				})
			}
		}
		states[slot] = slotAccess{
			resource: access.Resource,
			kind:     access.Kind,
			pass:     access.Pass,
		}
	}
	return barriers
}

func barrierKind(prev slotAccess, next Access) (string, string) {
	alias := prev.resource != next.Resource
	from := fmt.Sprintf("%s %s@pass%d", prev.resource, prev.kind, prev.pass+1)
	to := fmt.Sprintf("%s %s@pass%d", next.Resource, next.Kind, next.Pass+1)

	var hazard string
	switch {
	case prev.kind == Write && next.Kind == Read:
		hazard = "RAW"
	case prev.kind == Read && next.Kind == Write:
		hazard = "WAR"
	case prev.kind == Write && next.Kind == Write:
		hazard = "WAW"
	case prev.kind == Read && next.Kind == Read:
		hazard = "RAR"
	}
	if alias {
		reason := fmt.Sprintf("rebind slot %s -> %s", from, to)
		if hazard == "RAW" {
			return "ALIAS_RAW", reason + " (new consumer sees prior tenant)"
		}
		if hazard == "WAR" {
			return "ALIAS_WAR", reason + " (old reader must finish before overwrite)"
		}
		if hazard == "WAW" {
			return "ALIAS_WAW", reason + " (overwrite prior tenant)"
		}
		return "ALIAS_RAR", reason + " (two tenants simultaneously live)"
	}
	if hazard == "RAW" {
		return "RAW", fmt.Sprintf("make prior write visible: %s -> %s", from, to)
	}
	if hazard == "WAR" {
		return "WAR", fmt.Sprintf("protect prior read: %s -> %s", from, to)
	}
	if hazard == "WAW" {
		return "WAW", fmt.Sprintf("order writes: %s -> %s", from, to)
	}
	return "", ""
}

func BarrierLogText(mode AliasMode, barriers []Barrier) string {
	var b strings.Builder
	fmt.Fprintf(&b, "mode=%s barriers=%d\n", mode, len(barriers))
	for _, barrier := range barriers {
		fmt.Fprintf(&b,
			"pass %02d %-10s slot %d  %-10s  %s[%s@%d] -> %s[%s@%d]\n",
			barrier.Pass+1, barrier.PassID, barrier.Slot, barrier.Kind,
			barrier.FromResource, barrier.FromAccess, barrier.FromPass+1,
			barrier.ToResource, barrier.ToAccess, barrier.Pass+1)
	}
	return b.String()
}
