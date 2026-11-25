package aliaslab

import "fmt"

type RunResult struct {
	Outputs []*Image
	Final   *Image
}

// Run executes a plan against physical slots. Read inputs are snapshotted
// before the pass runs, while the write goes to the resource's declared slot.
// The snapshot is important: it isolates execution ordering from the liveness
// bug and lets stale aliases remain observable across pass boundaries.
func Run(g *Graph, plan Plan) (*RunResult, error) {
	slots := make([]*Image, plan.SlotCount)
	result := &RunResult{Outputs: make([]*Image, len(g.Passes))}
	for passIndex, pass := range g.Passes {
		inputs := make(map[ResourceID]*Image, len(pass.Reads))
		for _, id := range pass.Reads {
			slot := plan.SlotFor(id)
			if slot < 0 || slot >= len(slots) {
				return nil, fmt.Errorf("pass %q: resource %q has invalid slot %d",
					pass.ID, id, slot)
			}
			if slots[slot] == nil {
				return nil, fmt.Errorf("pass %q reads %q before it is produced",
					pass.ID, id)
			}
			inputs[id] = slots[slot].Clone()
		}
		out := pass.Apply(inputs)
		if out == nil {
			return nil, fmt.Errorf("pass %q returned a nil image", pass.ID)
		}
		if out.Width != g.Width || out.Height != g.Height || len(out.Pix) != g.Width*g.Height {
			return nil, fmt.Errorf("pass %q returned invalid image %dx%d",
				pass.ID, out.Width, out.Height)
		}
		slot := plan.SlotFor(pass.Writes[0])
		slots[slot] = out
		result.Outputs[passIndex] = out
		if passIndex == len(g.Passes)-1 {
			result.Final = out
		}
	}
	return result, nil
}

// FirstDivergingPass compares the resource produced by each pass. It is used
// to distinguish the first corrupted producer from downstream amplification.
func FirstDivergingPass(reference, candidate *RunResult) int {
	limit := len(reference.Outputs)
	if len(candidate.Outputs) < limit {
		limit = len(candidate.Outputs)
	}
	for i := 0; i < limit; i++ {
		if !reference.Outputs[i].Equal(candidate.Outputs[i]) {
			return i
		}
	}
	return -1
}
