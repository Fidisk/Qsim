package animation

import (
	"fmt"
	"qsim/components"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func Update() {
	if Anim.State != StatePlaying {
		return
	}
	if IsFinished() || CurrentAnim() == nil {
		return
	}

	dt := float64(rl.GetFrameTime())
	Anim.Progress += dt
	if Anim.Progress >= StepDuration {
		Anim.Progress = 0
		Anim.SubIdx++

		ga := CurrentAnim()
		// sub-steps: n inputs arriving + 1 all-to-gate + 1 output
		if Anim.SubIdx > len(ga.Inputs)+1 {
			Anim.SubIdx = 0
			Anim.CurrentIdx++
			if IsFinished() {
				Anim.State = StateIdle
			}
		}
	}
}

func CurrentAnim() *GateAnim {
	if Anim.CurrentIdx < len(Anim.Gates) {
		return &Anim.Gates[Anim.CurrentIdx]
	}
	return nil
}

func Draw() {
	if !IsActive() || IsFinished() {
		return
	}
	ga := CurrentAnim()
	if ga == nil {
		return
	}

	n := len(ga.Inputs)
	if n == 0 {
		return
	}

	t := float32(Anim.Progress / StepDuration)
	gateCenter := ga.Gate.GetCenter()
	gateRadius := ga.Gate.GetCircle().Radius

	isOutputPhase := Anim.SubIdx > n
	isAllToGatePhase := Anim.SubIdx == n
	inputMovingIdx := Anim.SubIdx // the input index currently moving to its hook (when < n)

	// Draw compute label overlay
	if isAllToGatePhase || isOutputPhase {
		opac := float32(0.8)
		col := rl.Fade(rl.Gold, opac)
		rect := rl.Rectangle{
			X:      gateCenter.X - gateRadius,
			Y:      gateCenter.Y - gateRadius - 40,
			Width:  gateRadius * 2,
			Height: 30,
		}
		rl.DrawRectangleRec(rect, col)
		rl.DrawRectangleLinesEx(rect, 2, rl.Fade(rl.Gold, opac+0.2))
		label := fmt.Sprintf("Compute: %s", ga.Gate.Label)
		textW := rl.MeasureText(label, 14)
		rl.DrawText(label, int32(rect.X+rect.Width/2)-textW/2, int32(rect.Y+5), 14, rl.Fade(rl.White, opac))
	}

	// Draw input dots
	for i := 0; i < n; i++ {
		var pos rl.Vector2

		if isOutputPhase {
			continue
		} else if isAllToGatePhase {
			// All dots move from their hooks to the gate center together
			p := rl.Vector2Lerp(ga.Inputs[i].HookPos, gateCenter, t)
			pos = p
		} else if i < inputMovingIdx {
			// Already arrived – stay at hook
			pos = ga.Inputs[i].HookPos
		} else if i == inputMovingIdx {
			// Currently moving to hook
			pos = rl.Vector2Lerp(ga.Inputs[i].StartPos, ga.Inputs[i].HookPos, t)
		} else {
			continue
		}

		rl.DrawCircleV(pos, DotRadius, rl.Yellow)
		rl.DrawCircleLinesV(pos, DotRadius, rl.Orange)
	}

	// Draw output dot
	if isOutputPhase {
		outT := float32(Anim.Progress / StepDuration)
		edge := utils.RectEdgePoint(gateCenter, ga.OutputPos, gateRadius, gateRadius)
		pos := rl.Vector2Lerp(edge, ga.OutputPos, outT)
		rl.DrawCircleV(pos, DotRadius, rl.Yellow)
		rl.DrawCircleLinesV(pos, DotRadius, rl.Orange)
	}
}

func DrawStaticLabels(wComp []components.Component) {
}
