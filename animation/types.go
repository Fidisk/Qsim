package animation

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"qsim/components"
)

type State int

const (
	StateIdle    State = iota
	StatePlaying
	StatePaused
)

type InputDot struct {
	StartPos rl.Vector2
	HookPos  rl.Vector2
}

type GateAnim struct {
	Inputs    []InputDot
	OutputPos rl.Vector2
	Gate      *components.Gate

	// Demo is the precomputed gate-computation visualization. It is built
	// lazily when the animation reaches the compute sub-step; DemoReady
	// records that the build was attempted (Demo may still be nil, e.g. for
	// measurement gates, in which case the legacy compute box is shown).
	Demo      *ComputeDemo
	DemoReady bool
}

type AnimState struct {
	Gates      []GateAnim
	CurrentIdx int
	SubIdx     int
	Progress   float64
	State      State
}

var Anim = AnimState{
	State: StateIdle,
}

const DotRadius float32 = 10
const StepDuration = 2.0

func Reset() {
	Anim.Gates = nil
	Anim.CurrentIdx = 0
	Anim.SubIdx = 0
	Anim.Progress = 0
	Anim.State = StateIdle
}

func IsActive() bool {
	return (Anim.State == StatePlaying || Anim.State == StatePaused) && len(Anim.Gates) > 0
}

func IsFinished() bool {
	return Anim.CurrentIdx >= len(Anim.Gates)
}
