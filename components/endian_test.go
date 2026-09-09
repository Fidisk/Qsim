package components

import (
	"math"
	"testing"

	"qsim/qubits"
	"qsim/qubits/attributes"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var endianH = complex64(complex(float32(0.70710678), 0))

// State |10> + |01> over sqrt(2): ModifierID[0] is documented as the MSB,
// so index 2 (|10>) means m0=1,m1=0 and index 1 (|01>) means m0=0,m1=1.
func TestMeasurementBitOrder(t *testing.T) {
	win := &stubWindow{}
	m0 := attributes.GenerateQubitModifierID()
	m1 := attributes.GenerateQubitModifierID()
	qs := NewQubitsSystem(-400, 0, 30, rl.White)
	qs.Assign(qubits.NewQubitStateManagerFrom(
		[]complex64{0, endianH, endianH, 0}, []int32{m0, m1}))
	win.PushComponent(qs)

	prob := func(vals [2]float64) bool {
		return math.Abs(vals[0]-0.5) > 1e-5 || math.Abs(vals[1]-0.5) > 1e-5
	}
	amp := func(s *QubitsSystem, i int) float64 {
		return math.Abs(float64(real(s.Origin.Amptitude[i])))
	}

	// Measure m0 (ModifierID[0], the MSB). Correct semantics: p(0) = |01|^2.
	m := NewCollapseGate(0, 0, 30, rl.Lime, "M2")
	win.PushComponent(m)
	m.HookList[0].Connect(qs.QubitDeterminatorList[0])
	m.MeasureOutput()
	if prob([2]float64{m.OutcomeProbs[0], m.OutcomeProbs[1]}) {
		t.Fatalf("m0 probs = [%v %v], want [0.5 0.5]", m.OutcomeProbs[0], m.OutcomeProbs[1])
	}
	// Force m0=|0>: the remainder must be |1>_m1 (from the |01> branch).
	m.ForceMode = 1
	m.Measured = false
	m.MeasureOutput()
	rest := utils.GetObjectFromID(m.OutPutHook[1].TargetID).(*QubitsSystem)
	if amp(rest, 1) < 0.99 {
		t.Fatalf("m0=|0> remainder = %v, want |1>_m1", rest.Origin.Amptitude)
	}
	// Force m0=|1>: remainder must be |0>_m1 (from the |10> branch).
	m.ForceMode = 2
	m.Measured = false
	m.MeasureOutput()
	if amp(rest, 0) < 0.99 {
		t.Fatalf("m0=|1> remainder = %v, want |0>_m1", rest.Origin.Amptitude)
	}

	// Measure m1 (ModifierID[1], the LSB).
	m2 := NewCollapseGate(200, 0, 30, rl.Lime, "M2")
	win.PushComponent(m2)
	m2.HookList[0].Connect(qs.QubitDeterminatorList[1])
	m2.MeasureOutput()
	if prob([2]float64{m2.OutcomeProbs[0], m2.OutcomeProbs[1]}) {
		t.Fatalf("m1 probs = [%v %v], want [0.5 0.5]", m2.OutcomeProbs[0], m2.OutcomeProbs[1])
	}
	// Force m1=|0>: the remainder must be |1>_m0 (from the |10> branch).
	m2.ForceMode = 1
	m2.Measured = false
	m2.MeasureOutput()
	rest2 := utils.GetObjectFromID(m2.OutPutHook[1].TargetID).(*QubitsSystem)
	if amp(rest2, 1) < 0.99 {
		t.Fatalf("m1=|0> remainder = %v, want |1>_m0", rest2.Origin.Amptitude)
	}
}

// The M1 branch gate must put the |0> outcome on output 0 with the correct
// conditioned remainder.
func TestM1BranchBitOrder(t *testing.T) {
	win := &stubWindow{}
	m0 := attributes.GenerateQubitModifierID()
	m1 := attributes.GenerateQubitModifierID()
	qs := NewQubitsSystem(-400, 0, 30, rl.White)
	qs.Assign(qubits.NewQubitStateManagerFrom(
		[]complex64{0, endianH, endianH, 0}, []int32{m0, m1}))
	win.PushComponent(qs)

	g := NewMeasurementGate(0, 0, 30, rl.Lime, "M")
	win.PushComponent(g)
	g.HookList[0].Connect(qs.QubitDeterminatorList[0]) // measure MSB m0
	g.MeasureOutput()
	if math.Abs(g.OutcomeProbs[0]-0.5) > 1e-5 || math.Abs(g.OutcomeProbs[1]-0.5) > 1e-5 {
		t.Fatalf("M1 probs = [%v %v], want [0.5 0.5]", g.OutcomeProbs[0], g.OutcomeProbs[1])
	}
	branch0 := utils.GetObjectFromID(g.OutPutHook[0].TargetID).(*QubitsSystem)
	if math.Abs(float64(real(branch0.Origin.Amptitude[1]))) < 0.99 {
		t.Fatalf("|0> branch remainder = %v, want |1>_m1", branch0.Origin.Amptitude)
	}
	branch1 := utils.GetObjectFromID(g.OutPutHook[1].TargetID).(*QubitsSystem)
	if math.Abs(float64(real(branch1.Origin.Amptitude[0]))) < 0.99 {
		t.Fatalf("|1> branch remainder = %v, want |0>_m1", branch1.Origin.Amptitude)
	}
}
