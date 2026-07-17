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

	return gates
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
