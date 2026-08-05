package components

import (
	"math/cmplx"
	"qsim/qubits"
	"qsim/qubits/attributes"
	"qsim/utils"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// M and M2 must agree: the |k> branch remainder of the measurement gate (M)
// and the collapse gate's (M2) "R" output for the same forced outcome have to
// be identical states. This feeds both gates the same 3-qubit state and
// compares them for every measured qubit position and both outcomes.
func TestM1M2Agree(t *testing.T) {
	// Teleportation-style state: [a,b,b,a, a,-b,-b,a]/2 with a=0.8, b=0.6.
	amps := []complex64{0.4, 0.3, 0.3, 0.4, 0.4, -0.3, -0.3, 0.4}

	for pos := int32(0); pos < 3; pos++ {
		for k := int32(0); k <= 1; k++ {
			m1Rest := measureWithM1(t, amps, pos, k)
			m2Rest := measureWithM2(t, amps, pos, k)
			if len(m1Rest) != len(m2Rest) {
				t.Fatalf("pos=%d k=%d: remainder sizes differ (%d vs %d)", pos, k, len(m1Rest), len(m2Rest))
			}
			for j := range m1Rest {
				if cmplx.Abs(complex128(m1Rest[j]-m2Rest[j])) > 1e-4 {
					t.Fatalf("pos=%d k=%d j=%d: M gives %v, M2 gives %v", pos, k, j, m1Rest[j], m2Rest[j])
				}
			}
		}
	}
}

func measureWithM1(t *testing.T, amps []complex64, pos int32, branch int32) []complex64 {
	t.Helper()
	QD := makeSystemWithState(amps, pos)
	g := NewMeasurementGate(0, 0, 30, rl.White, "M")
	g.HookList[0].Connect(QD)
	g.MeasureOutput()
	qs := hookedSystem(t, g.OutPutHook[branch])
	return qs.Origin.Amptitude
}

func measureWithM2(t *testing.T, amps []complex64, pos int32, k int32) []complex64 {
	t.Helper()
	QD := makeSystemWithState(amps, pos)
	g := NewCollapseGate(0, 0, 30, rl.White, "M2")
	g.ForceMode = k + 1 // 1 = force 0, 2 = force 1
	g.HookList[0].Connect(QD)
	g.MeasureOutput()
	if g.Result != k {
		t.Fatalf("M2 realized %d, forced %d", g.Result, k)
	}
	qs := hookedSystem(t, g.OutPutHook[1])
	return qs.Origin.Amptitude
}

// makeSystemWithState builds a 3-qubit system with the given amplitudes and
// returns the determinator at logical position pos (determinators are built
// in ModifierID order, so list index == logical position).
func makeSystemWithState(amps []complex64, pos int32) *QubitDeterminator {
	mods := make([]int32, 3)
	for i := range mods {
		mods[i] = attributes.GenerateQubitModifierID()
	}
	cp := make([]complex64, len(amps))
	copy(cp, amps)
	qs := NewQubitsSystem(0, 0, 30, rl.White)
	qs.Assign(qubits.NewQubitStateManagerFrom(cp, mods))
	return qs.QubitDeterminatorList[pos]
}

func hookedSystem(t *testing.T, h *Hook) *QubitsSystem {
	t.Helper()
	if !h.IsHooked {
		t.Fatal("output hook not connected")
	}
	qs, ok := utils.GetObjectFromID(h.TargetID).(*QubitsSystem)
	if !ok || qs.Origin == nil {
		t.Fatal("output hook target is not a qubit system")
	}
	return qs
}
