package animation

import (
	"math"
	"testing"

	"qsim/components"
)

// SetTime/CurrentTime round-trip over a fake 3-gate animation. Measurement
// gates have no demo, so every sub-step lasts StepDuration:
// gate0: 1 input  = 1+1+1 sub-steps ->  6s
// gate1: 2 inputs = 2+1+1 sub-steps ->  8s
// gate2: 0 inputs = 0+1+1 sub-steps ->  4s   (total 18s)
func TestSetTimeRoundTrip(t *testing.T) {
	Reset()
	defer Reset()

	mk := func(n int) GateAnim {
		return GateAnim{
			Inputs: make([]InputDot, n),
			Gate:   &components.Gate{IsMeasurementGate: true},
		}
	}
	Anim.Gates = []GateAnim{mk(1), mk(2), mk(0)}
	Anim.State = StateIdle

	if got := TotalDuration(); got != 18 {
		t.Fatalf("TotalDuration = %v, want 18", got)
	}

	type want struct {
		idx, sub int
		progress float64
	}
	cases := []struct {
		t    float64
		want want
	}{
		{0, want{0, 0, 0}},
		{3, want{0, 1, 1}},      // mid second input of gate0
		{6, want{1, 0, 0}},      // exact gate boundary -> start of gate1
		{10, want{1, 2, 0}},     // exact compute-step boundary of gate1
		{11.5, want{1, 2, 1.5}}, // mid compute step of gate1
		{15, want{2, 0, 1}},     // output... compute step of gate2
		{-5, want{0, 0, 0}},     // clamped to start
	}
	for _, c := range cases {
		SetTime(c.t)
		if Anim.CurrentIdx != c.want.idx || Anim.SubIdx != c.want.sub {
			t.Fatalf("SetTime(%v) -> (%d,%d), want (%d,%d)", c.t, Anim.CurrentIdx, Anim.SubIdx, c.want.idx, c.want.sub)
		}
		if math.Abs(Anim.Progress-c.want.progress) > 1e-9 {
			t.Fatalf("SetTime(%v) progress = %v, want %v", c.t, Anim.Progress, c.want.progress)
		}
		if math.Abs(CurrentTime()-max(c.t, 0)) > 1e-9 {
			t.Fatalf("CurrentTime after SetTime(%v) = %v", c.t, CurrentTime())
		}
		if Anim.State != StatePaused {
			t.Fatalf("SetTime(%v) should leave the animation paused, got %v", c.t, Anim.State)
		}
	}

	// Scrubbing to/past the end finishes the animation.
	SetTime(18)
	if !IsFinished() || Anim.State != StateIdle {
		t.Fatalf("SetTime(18) should finish the animation, idx=%d state=%v", Anim.CurrentIdx, Anim.State)
	}
	SetTime(100)
	if !IsFinished() {
		t.Fatalf("SetTime(100) should stay finished")
	}
}
