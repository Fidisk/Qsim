package animation

import (
	"qsim/config"
)

// gateDurations returns the duration of each sub-step of a gate's animation:
// one per input dot, then the compute step (dot arrival + demo), then the
// output step. It must stay in sync with stepDuration in render.go.
func gateDurations(ga *GateAnim) []float64 {
	durs := make([]float64, 0, len(ga.Inputs)+2)
	for range ga.Inputs {
		durs = append(durs, StepDuration)
	}
	ensureDemo(ga)
	if ga.Demo != nil {
		durs = append(durs, float64(config.ComputeDotArriveDur)+ga.Demo.Total)
	} else {
		durs = append(durs, StepDuration)
	}
	durs = append(durs, StepDuration)
	return durs
}

// GateDuration returns the total length of one gate's animation in seconds.
func GateDuration(ga *GateAnim) float64 {
	t := 0.0
	for _, d := range gateDurations(ga) {
		t += d
	}
	return t
}

// TotalDuration returns the full animation length in seconds.
func TotalDuration() float64 {
	t := 0.0
	for i := range Anim.Gates {
		t += GateDuration(&Anim.Gates[i])
	}
	return t
}

// CurrentTime returns the absolute playback position in seconds.
func CurrentTime() float64 {
	t := 0.0
	for i := 0; i < Anim.CurrentIdx && i < len(Anim.Gates); i++ {
		t += GateDuration(&Anim.Gates[i])
	}
	ga := CurrentAnim()
	if ga != nil {
		durs := gateDurations(ga)
		for s := 0; s < Anim.SubIdx && s < len(durs); s++ {
			t += durs[s]
		}
		t += Anim.Progress
	}
	return t
}

// SetTime scrubs the animation to absolute time t seconds, mapping it back
// to (CurrentIdx, SubIdx, Progress). Scrubbing to the end finishes the
// animation; scrubbing anywhere else leaves it paused.
func SetTime(t float64) {
	if len(Anim.Gates) == 0 {
		return
	}
	if t < 0 {
		t = 0
	}
	if t >= TotalDuration() {
		Anim.CurrentIdx = len(Anim.Gates)
		Anim.SubIdx = 0
		Anim.Progress = 0
		Anim.State = StateIdle
		return
	}

	idx := 0
	for idx < len(Anim.Gates)-1 {
		gd := GateDuration(&Anim.Gates[idx])
		if t < gd {
			break
		}
		t -= gd
		idx++
	}
	Anim.CurrentIdx = idx

	durs := gateDurations(&Anim.Gates[idx])
	sub := 0
	for sub < len(durs)-1 && t >= durs[sub] {
		t -= durs[sub]
		sub++
	}
	Anim.SubIdx = sub
	Anim.Progress = t
	if Anim.State == StateIdle {
		Anim.State = StatePaused
	}
}
