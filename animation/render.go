package animation

import (
	"fmt"
	"qsim/components"
	"qsim/config"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func Update() {
	if Anim.State != StatePlaying {
		return
	}

	dt := float64(rl.GetFrameTime())

	if IsFinished() {
		Anim.EndPause -= dt
		if Anim.EndPause <= 0 {
			Anim.State = StateIdle
		}
		return
	}
	if Anim.EndPause > 0 {
		Anim.EndPause -= dt
		if Anim.EndPause <= 0 {
			Anim.CurrentIdx = len(Anim.Gates)
		}
		return
	}
	if CurrentAnim() == nil {
		return
	}

	Anim.Progress += dt
	if Anim.Progress >= stepDuration() {
		Anim.Progress = 0
		Anim.SubIdx++

		ga := CurrentAnim()
		// sub-steps: n inputs arriving + 1 all-to-gate + 1 output
		if Anim.SubIdx > len(ga.Inputs)+1 {
			Anim.SubIdx = 0
			Anim.CurrentIdx++
			if IsFinished() {
				// Hold the last gate's final frame for a beat before finishing.
				Anim.EndPause = EndPauseDur
				Anim.CurrentIdx = len(Anim.Gates) - 1
				Anim.SubIdx = len(ga.Inputs) + 1
				Anim.Progress = stepDuration()
			}
		}
	}
}

// ensureDemo builds the gate's computation demo the first time the compute
// sub-step is reached. Demo stays nil for gates without a matrix input.
func ensureDemo(ga *GateAnim) {
	if ga == nil || ga.DemoReady {
		return
	}
	ga.Demo = BuildComputeDemo(ga.Gate)
	ga.DemoReady = true
}

// stepDuration returns how long the current sub-step takes. The compute
// sub-step lasts long enough for the input dots to arrive plus the whole
// computation demo; everything else uses StepDuration.
func stepDuration() float64 {
	ga := CurrentAnim()
	if ga == nil {
		return StepDuration
	}
	if Anim.SubIdx == len(ga.Inputs) {
		ensureDemo(ga)
		if ga.Demo != nil {
			return float64(config.ComputeDotArriveDur) + ga.Demo.Total
		}
	}
	return StepDuration
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

	if isAllToGatePhase {
		ensureDemo(ga)
	}

	// Legacy "Compute:" box, only when there is no demo to show
	// (e.g. measurement gates have no unitary matrix).
	if ga.Demo == nil && (isAllToGatePhase || isOutputPhase) {
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
		} else if isAllToGatePhase && ga.Demo != nil {
			// Dots fly into the gate, then the computation demo takes over
			if Anim.Progress >= float64(config.ComputeDotArriveDur) {
				continue
			}
			dotT := float32(Anim.Progress / float64(config.ComputeDotArriveDur))
			pos = rl.Vector2Lerp(ga.Inputs[i].HookPos, gateCenter, dotT)
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
		rl.DrawCircleLines(int32(pos.X), int32(pos.Y), DotRadius, rl.Orange)
	}

	// The computation demo replaces the old yellow box
	if isAllToGatePhase && ga.Demo != nil && Anim.Progress >= float64(config.ComputeDotArriveDur) {
		DrawCompute(ga, Anim.Progress-float64(config.ComputeDotArriveDur))
	}

	// Draw output dot
	if isOutputPhase {
		outT := float32(Anim.Progress / StepDuration)
		edge := utils.RectEdgePoint(gateCenter, ga.OutputPos, gateRadius, gateRadius)
		pos := rl.Vector2Lerp(edge, ga.OutputPos, outT)
		rl.DrawCircleV(pos, DotRadius, rl.Yellow)
		rl.DrawCircleLines(int32(pos.X), int32(pos.Y), DotRadius, rl.Orange)
	}
}

func DrawStaticLabels(wComp []components.Component) {
}
