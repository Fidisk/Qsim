package components

import (
	"qsim/qubits"
	"qsim/qubits/attributes"
	"qsim/utils"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// stubWindow mimics RenderWindow's WComp bookkeeping (GetElement +
// DeleteChildWithID) well enough to exercise component Destroy() chains.
type stubWindow struct {
	elems []Component
}

func (s *stubWindow) GetElement() []Component { return s.elems }
func (s *stubWindow) PushComponent(v ...Component) {
	for _, c := range v {
		c.SetParent(s)
		s.elems = append(s.elems, c)
	}
}
func (s *stubWindow) DeleteChildWithID(id ...int32) {
	for _, want := range id {
		for i, c := range s.elems {
			if c.GetID() == want {
				s.elems = append(s.elems[:i], s.elems[i+1:]...)
				utils.DeleteObjectWithID(want)
				return
			}
		}
	}
}

func mkTwoQD() *QubitsSystem {
	mods := make([]int32, 3)
	for i := range mods {
		mods[i] = attributes.GenerateQubitModifierID()
	}
	qs := NewQubitsSystem(0, 0, 30, rl.White)
	qs.Assign(qubits.NewQubitStateManagerFrom(
		[]complex64{0.4, 0.3, 0.3, 0.4, 0.4, -0.3, -0.3, 0.4}, mods))
	return qs
}

// A measured collapse gate (M2/M3/M4) must be deletable without panicking,
// both before and after measurement, with the input determinator attached.
func TestDeleteCollapseGate(t *testing.T) {
	ctors := []struct {
		name string
		fn   func() *CollapseGate
	}{
		{"M2", func() *CollapseGate { return NewCollapseGate(0, 0, 30, rl.White, "M2") }},
		{"M3", func() *CollapseGate { return NewCollapseGate3(0, 0, 30, rl.White, "M3") }},
		{"M4", func() *CollapseGate {
			m := NewM4Gate(0, 0, 30, rl.White)
			return &m.CollapseGate
		}},
	}

	for _, tc := range ctors {
		t.Run(tc.name+"_measured_then_delete", func(t *testing.T) {
			win := &stubWindow{}
			qd := mkTwoQD()
			win.PushComponent(qd)
			g := tc.fn()
			win.PushComponent(g)
			g.HookList[0].Connect(qd.QubitDeterminatorList[0])
			g.MeasureOutput()
			g.Destroy() // the crash site
		})
		t.Run(tc.name+"_delete_before_measure", func(t *testing.T) {
			win := &stubWindow{}
			qd := mkTwoQD()
			win.PushComponent(qd)
			g := tc.fn()
			win.PushComponent(g)
			g.HookList[0].Connect(qd.QubitDeterminatorList[0])
			g.Destroy()
		})
	}
}

// Re-measuring an unchanged input state must reuse the realized outcome
// instead of drawing a new random one. That is what made M2/M3 flicker when a
// source refreshes identical amplitudes every frame.
func TestCollapseGateReMeasureStable(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func() *CollapseGate
	}{
		{"M2", func() *CollapseGate { return NewCollapseGate(0, 0, 30, rl.White, "M2") }},
		{"M3", func() *CollapseGate { return NewCollapseGate3(0, 0, 30, rl.White, "M3") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mods := []int32{attributes.GenerateQubitModifierID()}
			for i := 0; i < 50; i++ {
				win := &stubWindow{}
				qs := NewQubitsSystem(0, 0, 30, rl.White)
				qs.Assign(qubits.NewQubitStateManagerFrom(
					[]complex64{0.7, 0.7}, mods))
				win.PushComponent(qs)
				g := tc.fn()
				win.PushComponent(g)
				g.HookList[0].Connect(qs.QubitDeterminatorList[0])

				g.MeasureOutput()
				first := g.Result
				g.Measured = false
				g.MeasureOutput() // re-measure, same state
				if g.Result != first {
					t.Fatalf("result changed on re-measure of identical state: %d -> %d", first, g.Result)
				}
			}
		})
	}
}

// A force click clears the reuse fingerprint; forcing then overrides the
// previously realized random outcome.
func TestCollapseGateForceOverridesReuse(t *testing.T) {
	mods := []int32{attributes.GenerateQubitModifierID()}
	for i := 0; i < 50; i++ {
		win := &stubWindow{}
		qs := NewQubitsSystem(0, 0, 30, rl.White)
		qs.Assign(qubits.NewQubitStateManagerFrom(
			[]complex64{0.7, 0.7}, mods))
		win.PushComponent(qs)
		g := NewCollapseGate(0, 0, 30, rl.White, "M2")
		win.PushComponent(g)
		g.HookList[0].Connect(qs.QubitDeterminatorList[0])

		g.MeasureOutput() // random outcome, fingerprint stored
		g.ForceMode = 1
		g.Measured = false
		g.measuredHash = false // what CycleForce does
		g.MeasureOutput()
		if g.Result != 0 {
			t.Fatalf("force 0 was not honored after a random measure: result = %d", g.Result)
		}
	}
}

// M4 must keep the input qubit hooked and alive: measuring must not remove the
// determinator from its parent system, and left-click forcing is disabled.
func TestM4DoesNotConsumeInput(t *testing.T) {
	win := &stubWindow{}
	qd := mkTwoQD()
	win.PushComponent(qd)
	m := NewM4Gate(0, 0, 30, rl.White)
	win.PushComponent(m)
	in := m.HookList[0]
	in.Connect(qd.QubitDeterminatorList[0])

	m.MeasureOutput()

	if !in.IsHooked {
		t.Fatal("M4 input hook was unhooked after measurement")
	}
	if utils.GetObjectFromID(qd.QubitDeterminatorList[0].ID) == nil {
		t.Fatal("input determinator was killed after measurement")
	}
	if !m.DisableForceClick {
		t.Fatal("M4 must have DisableForceClick set")
	}
}
