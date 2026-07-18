package animation

import (
	"qsim/components"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func GenerateSteps(wComp []components.Component) []GateAnim {
	var gates []GateAnim

	for _, c := range wComp {
		g, ok := c.(*components.Gate)
		if !ok {
			continue
		}
		if !allInputsConnected(g) {
			continue
		}

		var anim GateAnim
		anim.Gate = g

		for _, h := range g.HookList {
			if h.IsOutput || !h.IsHooked {
				continue
			}
			target := utils.GetObjectFromID(h.TargetID)
			if target == nil {
				continue
			}

			var startPos rl.Vector2
			if det, ok := target.(*components.QubitDeterminator); ok {
				if qp := det.GetQubitParent(); qp != nil {
					startPos = qp.GetCenter()
				} else {
					startPos = det.GetCenter()
				}
			} else if qs, ok := target.(*components.QubitsSystem); ok {
				startPos = qs.GetCenter()
			} else {
				continue
			}

			if rl.Vector2Distance(startPos, h.GetCenter()) < 5 {
				startPos = rl.Vector2Add(h.GetCenter(), rl.Vector2{X: -100, Y: 0})
			}

			anim.Inputs = append(anim.Inputs, InputDot{
				StartPos: startPos,
				HookPos:  h.GetCenter(),
			})
		}

		if len(anim.Inputs) == 0 {
			continue
		}

		outPos := g.GetCenter()
		if len(g.OutPutHook) > 0 && g.OutPutHook[0].IsHooked {
			target := utils.GetObjectFromID(g.OutPutHook[0].TargetID)
			if qs, ok := target.(*components.QubitsSystem); ok {
				outPos = qs.GetCenter()
			}
		}
		anim.OutputPos = outPos

		gates = append(gates, anim)
	}

	return orderGates(gates)
}

// inputSystemID returns the ID of the QubitsSystem feeding a hooked input,
// whether the hook target is a QubitDeterminator or a QubitsSystem directly.
func inputSystemID(targetID int32) int32 {
	switch t := utils.GetObjectFromID(targetID).(type) {
	case *components.QubitDeterminator:
		if qp := t.GetQubitParent(); qp != nil {
			return qp.ID
		}
	case *components.QubitsSystem:
		return t.ID
	}
	return 0
}

// orderGates sorts gate animations in data-flow order: gates fed by a
// QubitsSystem that is not the output of another gate run first, then each
// consumer gate follows the gate that produced its input system.
func orderGates(gates []GateAnim) []GateAnim {
	// Map each output QubitsSystem to the gate that produces it
	producedBy := map[int32]int{}
	for i, ga := range gates {
		for _, oh := range ga.Gate.OutPutHook {
			if !oh.IsHooked {
				continue
			}
			if _, ok := utils.GetObjectFromID(oh.TargetID).(*components.QubitsSystem); ok {
				producedBy[oh.TargetID] = i
			}
		}
	}

	// Edge producer -> consumer when the consumer reads the producer's system
	inDegree := make([]int, len(gates))
	adj := make([][]int, len(gates))
	for j, ga := range gates {
		seen := map[int]bool{}
		for _, h := range ga.Gate.HookList {
			if h.IsOutput || !h.IsHooked {
				continue
			}
			src, ok := producedBy[inputSystemID(h.TargetID)]
			if !ok || src == j || seen[src] {
				continue
			}
			seen[src] = true
			adj[src] = append(adj[src], j)
			inDegree[j]++
		}
	}

	// Kahn's algorithm; among ready gates, keep the original panel order
	var ready []int
	for i := range gates {
		if inDegree[i] == 0 {
			ready = append(ready, i)
		}
	}
	order := make([]int, 0, len(gates))
	for len(ready) > 0 {
		i := ready[0]
		ready = ready[1:]
		order = append(order, i)
		for _, j := range adj[i] {
			inDegree[j]--
			if inDegree[j] == 0 {
				k := len(ready)
				for k > 0 && ready[k-1] > j {
					k--
				}
				ready = append(ready, 0)
				copy(ready[k+1:], ready[k:])
				ready[k] = j
			}
		}
	}

	// Cyclic leftovers (feedback wiring), keep them at the end in panel order
	inOrder := make([]bool, len(gates))
	for _, i := range order {
		inOrder[i] = true
	}
	for i := range gates {
		if !inOrder[i] {
			order = append(order, i)
		}
	}

	sorted := make([]GateAnim, 0, len(gates))
	for _, i := range order {
		sorted = append(sorted, gates[i])
	}
	return sorted
}

func allInputsConnected(g *components.Gate) bool {
	cnt := 0
	for _, h := range g.HookList {
		if !h.IsOutput && h.IsHooked {
			cnt++
		}
	}
	return cnt == int(g.InputCount)
}

func NeedsRegenerate(wComp []components.Component) bool {
	return len(Anim.Gates) == 0 && Anim.State == StateIdle
}
