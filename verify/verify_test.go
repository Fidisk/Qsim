package verify

import (
	"encoding/json"
	"math"
	"math/cmplx"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qsim/components"
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/qubits/attributes"
	"qsim/symbolic"
	"qsim/windows"
)

var (
	tVal     = symbolic.New(float32(1/math.Sqrt2), 0)
	hadamard = [][]symbolic.SymbolicValue{{tVal, tVal}, {tVal, tVal.Neg()}}
	cnot     = [][]symbolic.SymbolicValue{{symbolic.One(), symbolic.Zero(), symbolic.Zero(), symbolic.Zero()}, {symbolic.Zero(), symbolic.One(), symbolic.Zero(), symbolic.Zero()}, {symbolic.Zero(), symbolic.Zero(), symbolic.Zero(), symbolic.One()}, {symbolic.Zero(), symbolic.Zero(), symbolic.One(), symbolic.Zero()}}
	xGate    = [][]symbolic.SymbolicValue{{symbolic.Zero(), symbolic.One()}, {symbolic.One(), symbolic.Zero()}}
	zGate    = [][]symbolic.SymbolicValue{{symbolic.One(), symbolic.Zero()}, {symbolic.Zero(), symbolic.New(-1, 0)}}
)

func mkSystem(x, y float32, amps []symbolic.SymbolicValue, mods []int32) *components.QubitsSystem {
	qs := components.NewQubitsSystem(x, y, glob.QubitSystemRadius, config.QubitSystemColor)
	qs.Assign(qubits.NewQubitStateManagerFrom(append([]symbolic.SymbolicValue{}, amps...), append([]int32{}, mods...)))
	return qs
}

func mkGate(x, y float32, label string, op [][]symbolic.SymbolicValue, n int32) *components.Gate {
	return components.NewGate(x, y, glob.GateRadius, config.GateColor, label, op, n)
}

// bellSave builds H(q0); CX(q0,q1) headlessly and returns the save text.
func bellSave(t *testing.T) string {
	t.Helper()
	q0 := attributes.GenerateQubitModifierID()
	q1 := attributes.GenerateQubitModifierID()

	qsA := mkSystem(-2250, -100, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q0})
	qsB := mkSystem(-1450, 100, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q1})

	hA := mkGate(-1800, -100, "H", hadamard, 1)
	hA.HookList[0].Connect(qsA.QubitDeterminatorList[0])
	qsHA := mkSystem(hA.OutPutHook[0].Center.X, hA.OutPutHook[0].Center.Y,
		[]symbolic.SymbolicValue{tVal, tVal}, []int32{q0})
	hA.OutPutHook[0].Connect(qsHA)

	cx := mkGate(-1400, 0, "CX", cnot, 2)
	cx.HookList[0].Connect(qsHA.QubitDeterminatorList[0])
	cx.HookList[1].Connect(qsB.QubitDeterminatorList[0])
	qsBell := mkSystem(cx.OutPutHook[0].Center.X, cx.OutPutHook[0].Center.Y,
		[]symbolic.SymbolicValue{tVal, symbolic.Zero(), symbolic.Zero(), tVal}, []int32{q0, q1})
	cx.OutPutHook[0].Connect(qsBell)

	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	rw.Rename("Bell")
	for _, c := range []components.Component{qsA, qsB, hA, qsHA, cx, qsBell} {
		rw.PushComponent(c)
	}
	return rw.SaveState()
}

func fidelity(a []complex64, b []symbolic.SymbolicValue) float64 {
	if len(a) != len(b) {
		return -1
	}
	var dot complex128
	var na, nb float64
	for i := range a {
		av, bv := complex128(a[i]), b[i].ToComplex128()
		dot += cmplx.Conj(av) * bv
		na += real(av)*real(av) + imag(av)*imag(av)
		nb += real(bv)*real(bv) + imag(bv)*imag(bv)
	}
	if na == 0 || nb == 0 {
		return -1
	}
	return cmplx.Abs(dot) * cmplx.Abs(dot) / (na * nb)
}

func writeTestdata(t *testing.T, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join("testdata", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExtractBell(t *testing.T) {
	c, err := Extract(bellSave(t), "Bell")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if c.NumQubits != 2 {
		t.Fatalf("NumQubits = %d, want 2", c.NumQubits)
	}
	if len(c.Ops) != 2 {
		t.Fatalf("Ops = %+v, want 2 ops", c.Ops)
	}
	if c.Ops[0].Gate != "h" || len(c.Ops[0].Qubits) != 1 || c.Ops[0].Qubits[0] != 0 {
		t.Fatalf("op0 = %+v, want h on [0]", c.Ops[0])
	}
	if c.Ops[1].Gate != "cx" || len(c.Ops[1].Qubits) != 2 || c.Ops[1].Qubits[0] != 0 || c.Ops[1].Qubits[1] != 1 {
		t.Fatalf("op1 = %+v, want cx on [0 1]", c.Ops[1])
	}
	want := []symbolic.SymbolicValue{tVal, symbolic.Zero(), symbolic.Zero(), tVal}
	if f := fidelity(c.Expected, want); f < 1-1e-9 {
		t.Fatalf("Expected = %v, fidelity to Bell = %v", c.Expected, f)
	}

	wantQASM := "OPENQASM 2.0;\n" +
		"include \"qelib1.inc\";\n" +
		"// verify-bundle: Bell (2 qubits)\n" +
		"qreg q[2];\n" +
		"h q[0]; // H\n" +
		"cx q[0],q[1]; // CX\n"
	if got := QASM(c); got != wantQASM {
		t.Fatalf("QASM mismatch:\n got:\n%s\nwant:\n%s", got, wantQASM)
	}

	files, err := WriteBundle(c)
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("WriteBundle returned %d files, want 2", len(files))
	}
	var sc sidecar
	if err := json.Unmarshal([]byte(files["bell.verify.json"]), &sc); err != nil {
		t.Fatalf("sidecar JSON: %v", err)
	}
	if sc.NQubits != 2 || len(sc.Expected) != 4 || len(sc.Ops) != 2 {
		t.Fatalf("sidecar = %+v", sc)
	}
	writeTestdata(t, files)
}

// flipTarget is a CNOT with the control on I1 (matrix LSB): it flips I0
// when I1 is 1. |01> -> |11>.
var flipTarget = [][]symbolic.SymbolicValue{{symbolic.One(), symbolic.Zero(), symbolic.Zero(), symbolic.Zero()}, {symbolic.Zero(), symbolic.Zero(), symbolic.Zero(), symbolic.One()}, {symbolic.Zero(), symbolic.Zero(), symbolic.One(), symbolic.Zero()}, {symbolic.Zero(), symbolic.One(), symbolic.Zero(), symbolic.Zero()}}

func TestExtractCustomUnitary(t *testing.T) {
	q0 := attributes.GenerateQubitModifierID()
	q1 := attributes.GenerateQubitModifierID()

	qsA := mkSystem(-2250, -100, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q0})
	qsB := mkSystem(-1450, 100, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q1})

	xB := mkGate(-1800, 100, "X", xGate, 1)
	xB.HookList[0].Connect(qsB.QubitDeterminatorList[0])
	qsBX := mkSystem(xB.OutPutHook[0].Center.X, xB.OutPutHook[0].Center.Y,
		[]symbolic.SymbolicValue{symbolic.Zero(), symbolic.One()}, []int32{q1})
	xB.OutPutHook[0].Connect(qsBX)

	cu := mkGate(-1400, 0, "U", flipTarget, 2)
	cu.HookList[0].Connect(qsA.QubitDeterminatorList[0])
	cu.HookList[1].Connect(qsBX.QubitDeterminatorList[0])
	qsOut := mkSystem(cu.OutPutHook[0].Center.X, cu.OutPutHook[0].Center.Y,
		[]symbolic.SymbolicValue{symbolic.Zero(), symbolic.Zero(), symbolic.Zero(), symbolic.One()}, []int32{q0, q1})
	cu.OutPutHook[0].Connect(qsOut)

	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	rw.Rename("Custom")
	for _, c := range []components.Component{qsA, qsB, xB, qsBX, cu, qsOut} {
		rw.PushComponent(c)
	}
	c, err := Extract(rw.SaveState(), "Custom")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(c.Ops) != 2 {
		t.Fatalf("Ops = %+v, want [x unitary]", c.Ops)
	}
	u := c.Ops[1]
	if u.Gate != "unitary" || len(u.Qubits) != 2 || u.Qubits[0] != 1 || u.Qubits[1] != 0 {
		t.Fatalf("custom op = %+v, want unitary on [1 0] (reversed input order)", u)
	}
	if f := fidelity(c.Expected, []symbolic.SymbolicValue{symbolic.Zero(), symbolic.Zero(), symbolic.Zero(), symbolic.One()}); f < 1-1e-9 {
		t.Fatalf("Expected = %v, fidelity to |11> = %v", c.Expected, f)
	}
	files, err := WriteBundle(c)
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	writeTestdata(t, files)
}

func TestExtractSourcePrep(t *testing.T) {
	// SourceGate emitting |+> followed by H: exercises source-seeded input
	// (prep op) and must decode back to |0>.
	sg := components.NewSourceGateWithID(-2250, -100, glob.GateRadius, config.GateColor, "S",
		[]symbolic.SymbolicValue{tVal, tVal}, attributes.GenerateQubitModifierID())
	qsS := mkSystem(sg.OutHook.Center.X, sg.OutHook.Center.Y,
		[]symbolic.SymbolicValue{tVal, tVal}, []int32{sg.ModifierID})
	sg.OutHook.Connect(qsS)

	hA := mkGate(-1800, -100, "H", hadamard, 1)
	hA.HookList[0].Connect(qsS.QubitDeterminatorList[0])
	qsH := mkSystem(hA.OutPutHook[0].Center.X, hA.OutPutHook[0].Center.Y,
		[]symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{sg.ModifierID})
	hA.OutPutHook[0].Connect(qsH)

	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	rw.Rename("Prep")
	for _, c := range []components.Component{sg, qsS, hA, qsH} {
		rw.PushComponent(c)
	}
	c, err := Extract(rw.SaveState(), "Prep")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(c.Ops) != 2 || c.Ops[0].Gate != "prep" || c.Ops[1].Gate != "h" {
		t.Fatalf("Ops = %+v, want [prep h]", c.Ops)
	}
	if f := fidelity(c.Expected, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}); f < 1-1e-9 {
		t.Fatalf("Expected = %v, fidelity to |0> = %v", c.Expected, f)
	}
	files, err := WriteBundle(c)
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	writeTestdata(t, files)
}

func TestExtractTeleporSave(t *testing.T) {
	// The real teleport save: M2-driven corrections compile via deferred
	// measurement, the shared copy is tolerated, spares skipped.
	raw, err := os.ReadFile(filepath.Join("..", "saves", "Telepor.qsim"))
	if err != nil {
		t.Fatalf("read save: %v", err)
	}
	c, err := Extract(string(raw), "Telepor")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if c.NumQubits != 4 {
		t.Fatalf("NumQubits = %d, want 4", c.NumQubits)
	}
	var norm float64
	for _, a := range c.Expected {
		norm += float64(real(a)*real(a) + imag(a)*imag(a))
	}
	if math.Abs(norm-1) > 1e-6 {
		t.Fatalf("Expected norm = %v, want 1", norm)
	}
	compiled := 0
	for _, n := range c.Notes {
		if strings.Contains(n, "deferred measurement") {
			compiled++
		}
	}
	if compiled < 2 {
		t.Fatalf("want >=2 deferred-measurement notes, got %v", c.Notes)
	}
}

func TestExtractSuperdense(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "saves", "superdense.qsim"))
	if err != nil {
		t.Fatalf("read save: %v", err)
	}
	c, err := Extract(string(raw), "Superdense")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if c.NumQubits != 2 {
		t.Fatalf("NumQubits = %d, want 2", c.NumQubits)
	}
	stripped := false
	for _, n := range c.Notes {
		if strings.Contains(n, "M2") {
			stripped = true
		}
	}
	if !stripped {
		t.Fatalf("expected stripped-measurement notes, got %v", c.Notes)
	}
	// The generator encodes "11": the pre-measurement state must be |11>.
	if f := fidelity(c.Expected, []symbolic.SymbolicValue{symbolic.Zero(), symbolic.Zero(), symbolic.Zero(), symbolic.One()}); f < 1-1e-9 {
		t.Fatalf("Expected = %v, fidelity to |11> = %v", c.Expected, f)
	}
	files, err := WriteBundle(c)
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	writeTestdata(t, files)
}

func TestExtractBranchFanoutErrors(t *testing.T) {
	// Both M1 branches feeding different gates: fan-out of one state.
	q0 := attributes.GenerateQubitModifierID()
	qs := mkSystem(0, 0, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q0})
	m1 := components.NewMeasurementGate(300, 0, glob.GateRadius, config.GateColor, "M1")
	m1.HookList[0].Connect(qs.QubitDeterminatorList[0])
	b0 := mkSystem(600, -100, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q0})
	m1.OutPutHook[0].Connect(b0)
	b1 := mkSystem(600, 100, []symbolic.SymbolicValue{symbolic.Zero(), symbolic.One()}, []int32{q0})
	m1.OutPutHook[1].Connect(b1)
	g0 := mkGate(900, -100, "X", xGate, 1)
	g0.HookList[0].Connect(b0.QubitDeterminatorList[0])
	out0 := mkSystem(1200, -100, []symbolic.SymbolicValue{symbolic.Zero(), symbolic.One()}, []int32{q0})
	g0.OutPutHook[0].Connect(out0)
	g1 := mkGate(900, 100, "Z", zGate, 1)
	g1.HookList[0].Connect(b1.QubitDeterminatorList[0])
	out1 := mkSystem(1200, 100, []symbolic.SymbolicValue{symbolic.Zero(), symbolic.One()}, []int32{q0})
	g1.OutPutHook[0].Connect(out1)

	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	for _, c := range []components.Component{qs, m1, b0, b1, g0, out0, g1, out1} {
		rw.PushComponent(c)
	}
	_, err := Extract(rw.SaveState(), "Branch")
	if err == nil || !strings.Contains(err.Error(), "fan-out") {
		t.Fatalf("err = %v, want fan-out error", err)
	}
}

func TestExtractStoredControl(t *testing.T) {
	// Undriven control bit with stored value 1: applies X like the engine.
	q0 := attributes.GenerateQubitModifierID()
	qs := mkSystem(0, 0, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q0})
	bit := components.NewLogicalBit(300, -200, glob.QubitSystemRadius/2, 1)
	cg := components.NewControlledGate(300, 0, config.GateColor, components.CtrlX)
	cg.InQubit.Connect(qs.QubitDeterminatorList[0])
	cg.InControl.Connect(bit)
	out := mkSystem(600, 0, []symbolic.SymbolicValue{symbolic.Zero(), symbolic.One()}, []int32{q0})
	cg.OutHook.Connect(out)

	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	for _, c := range []components.Component{qs, bit, cg, out} {
		rw.PushComponent(c)
	}
	c, err := Extract(rw.SaveState(), "Stored")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if f := fidelity(c.Expected, []symbolic.SymbolicValue{symbolic.Zero(), symbolic.One()}); f < 1-1e-9 {
		t.Fatalf("Expected = %v, fidelity to |1> = %v", c.Expected, f)
	}
}

func TestExtractLogicControlErrors(t *testing.T) {
	// Logic-driven control bit: value not statically known.
	q0 := attributes.GenerateQubitModifierID()
	qs := mkSystem(0, 0, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q0})
	bitIn := components.NewLogicalBit(0, -400, glob.QubitSystemRadius/2, 1)
	bitOut := components.NewLogicalBit(300, -200, glob.QubitSystemRadius/2, 0)
	lg := components.NewLogicGate(150, -300, config.GateColor, components.LogicNot)
	lg.InA.Connect(bitIn)
	lg.OutHook.Connect(bitOut)
	cg := components.NewControlledGate(300, 0, config.GateColor, components.CtrlX)
	cg.InQubit.Connect(qs.QubitDeterminatorList[0])
	cg.InControl.Connect(bitOut)
	out := mkSystem(600, 0, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q0})
	cg.OutHook.Connect(out)

	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	for _, c := range []components.Component{qs, bitIn, bitOut, lg, cg, out} {
		rw.PushComponent(c)
	}
	_, err := Extract(rw.SaveState(), "Driven")
	if err == nil || !strings.Contains(err.Error(), "not statically known") {
		t.Fatalf("err = %v, want statically-known control error", err)
	}
}

func TestExtractTeleportDriven(t *testing.T) {
	// Bell pair, M2 on qubit 0 driving cX on qubit 1: deferred-measurement
	// compile gives CX(q0 -> q1) on Bell = |+>|0>.
	q0 := attributes.GenerateQubitModifierID()
	q1 := attributes.GenerateQubitModifierID()

	qsA := mkSystem(-2250, -100, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q0})
	qsB := mkSystem(-1450, 100, []symbolic.SymbolicValue{symbolic.One(), symbolic.Zero()}, []int32{q1})

	hA := mkGate(-1800, -100, "H", hadamard, 1)
	hA.HookList[0].Connect(qsA.QubitDeterminatorList[0])
	qsHA := mkSystem(hA.OutPutHook[0].Center.X, hA.OutPutHook[0].Center.Y,
		[]symbolic.SymbolicValue{tVal, tVal}, []int32{q0})
	hA.OutPutHook[0].Connect(qsHA)

	cx := mkGate(-1400, 0, "CX", cnot, 2)
	cx.HookList[0].Connect(qsHA.QubitDeterminatorList[0])
	cx.HookList[1].Connect(qsB.QubitDeterminatorList[0])
	qsBell := mkSystem(cx.OutPutHook[0].Center.X, cx.OutPutHook[0].Center.Y,
		[]symbolic.SymbolicValue{tVal, symbolic.Zero(), symbolic.Zero(), tVal}, []int32{q0, q1})
	cx.OutPutHook[0].Connect(qsBell)

	m2 := components.NewCollapseGate(-1000, 0, glob.GateRadius, config.GateColor, "M2")
	m2.HookList[0].Connect(qsBell.QubitDeterminatorList[0])
	bit := components.NewLogicalBit(-700, -100, glob.QubitSystemRadius/2, 0)
	m2.OutPutHook[0].Connect(bit)

	cx2 := components.NewControlledGate(-400, 100, config.GateColor, components.CtrlX)
	cx2.InQubit.Connect(qsBell.QubitDeterminatorList[1])
	cx2.InControl.Connect(bit)
	qsOut := mkSystem(-100, 100, []symbolic.SymbolicValue{tVal, symbolic.Zero(), tVal, symbolic.Zero()}, []int32{q0, q1})
	cx2.OutHook.Connect(qsOut)

	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	rw.Rename("TeleportDriven")
	for _, c := range []components.Component{qsA, qsB, hA, qsHA, cx, qsBell, m2, bit, cx2, qsOut} {
		rw.PushComponent(c)
	}
	c, err := Extract(rw.SaveState(), "TeleportDriven")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// CX(control q0) on (|00> + |11>)/sqrt2 = (|00> + |01>)/sqrt2:
	// the control ends in |+>, the target in |0>.
	if f := fidelity(c.Expected, []symbolic.SymbolicValue{tVal, tVal, symbolic.Zero(), symbolic.Zero()}); f < 1-1e-9 {
		t.Fatalf("Expected = %v, fidelity to want = %v", c.Expected, f)
	}
	found := false
	for _, o := range c.Ops {
		if o.Gate == "cx" && len(o.Qubits) == 2 && o.Qubits[0] == 0 && o.Qubits[1] == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("Ops = %+v, want compiled cx on [0 1]", c.Ops)
	}
	files, err := WriteBundle(c)
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	writeTestdata(t, files)
}
