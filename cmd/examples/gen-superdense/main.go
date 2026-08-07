// Command gen-superdense is a standalone EXAMPLE that shows how to author a
// .qsim save programmatically. Instead of hand-writing JSON it builds the
// circuit out of the engine's own components (sources, gates, systems,
// collapse measurements, logical bits and lights), wires them with the hook
// Connect/ConnectInfo APIs, assigns the exact intermediate quantum states
// with qubits.QubitStateManager, and finally serializes the render window
// with RenderWindow.SaveState().
//
// The generated file (saves/superdense.qsim) implements superdense coding:
// two sources |0>_A and |0>_B form a Bell pair (H then CX), Alice encodes
// the message "11" on her qubit (Z gate then X gate, both acting on qubit 0
// only), Bob applies CX and H, and two M2 collapse gates measure both qubits
// into logical bits that drive lights.
//
// Run it from the repository root:
//
//	go run ./cmd/examples/gen-superdense
//
// then load saves/superdense.qsim in the app. The file is a single JSON line
// describing one RenderWindow; see docs/qsim-format.md for the format.
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"

	"qsim/components"
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/qubits/attributes"
	"qsim/utils"
	"qsim/windows"
)

var h = complex64(complex(float32(1/math.Sqrt(2)), 0)) // 1/√2

var (
	// 1-qubit Hadamard.
	hadamard = [][]complex64{{h, h}, {h, -h}}
	// CNOT with control = qubit 0 (first column) and target = qubit 1.
	cnot = [][]complex64{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 0, 1},
		{0, 0, 1, 0},
	}
	// Pauli Z acting on qubit 0 only (Z ⊗ I on |q0,q1>).
	zOnQ0 = [][]complex64{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, -1, 0},
		{0, 0, 0, -1},
	}
	// Pauli X acting on qubit 0 only (X ⊗ I on |q0,q1>).
	xOnQ0 = [][]complex64{
		{0, 0, 1, 0},
		{0, 0, 0, 1},
		{1, 0, 0, 0},
		{0, 1, 0, 0},
	}
	// Hadamard acting on qubit 0 only (H ⊗ I on |q0,q1>).
	hOnQ0 = [][]complex64{
		{h, 0, h, 0},
		{0, h, 0, h},
		{h, 0, -h, 0},
		{0, h, 0, -h},
	}
)

// gate returns a new Gate component at (x,y) with the given label, operation
// matrix and input count. Hooks are positioned by the constructor at
// x±GateToHookDist on the snap grid.
func gate(x, y float32, label string, op [][]complex64, inputCount int32) *components.Gate {
	return components.NewGate(x, y, glob.GateRadius, config.GateColor, label, op, inputCount)
}

// system returns a new QubitsSystem assigned the given state.
func system(x, y float32, state *qubits.QubitStateManager) *components.QubitsSystem {
	qs := components.NewQubitsSystem(x, y, glob.QubitSystemRadius, config.QubitSystemColor)
	qs.Assign(state)
	return qs
}

// cloneState makes a fresh QubitStateManager copy of src (new ID, same
// amplitudes and modifier order) so each stage owns its own origin.
func cloneState(src *qubits.QubitStateManager) *qubits.QubitStateManager {
	return qubits.NewQubitStateManagerFrom(
		append([]complex64{}, src.Amptitude...),
		append([]int32{}, src.ModifierID...),
	)
}

// oneQubit simulates a 1-qubit gate exactly like the engine: copy, then
// Multiply with the 2×2 matrix.
func oneQubit(src *qubits.QubitStateManager, op [][]complex64) *qubits.QubitStateManager {
	out := cloneState(src)
	out.Multiply(op, 1)
	return out
}

// twoQubit simulates a 2-qubit gate exactly like the engine: merge both
// single-qubit states (first operand becomes the high bit), then Multiply
// with the 4×4 matrix.
func twoQubit(a, b *qubits.QubitStateManager, op [][]complex64) *qubits.QubitStateManager {
	out := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	out.Merge(a)
	out.Merge(b)
	out.Multiply(op, 2)
	return out
}

// twoQubitOnState applies a 4×4 matrix to an existing 2-qubit state.
func twoQubitOnState(src *qubits.QubitStateManager, op [][]complex64) *qubits.QubitStateManager {
	out := cloneState(src)
	out.Multiply(op, 2)
	return out
}

func main() {
	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	rw.Rename("Superdense coding")

	// --- quantum states -------------------------------------------------
	//
	// Qubit 0 lives in column 0 (high bit), qubit 1 in column 1, matching
	// the engine's Merge order: result index i = q0<<1 | q1.
	// Register real modifier IDs: Assign resolves them against the attribute
	// manager while building the qubit visuals.
	q0 := attributes.GenerateQubitModifierID()
	q1 := attributes.GenerateQubitModifierID()

	zero := []complex64{1, 0}

	stA0 := qubits.NewQubitStateManagerFrom(zero, []int32{q0}) // |0>_A
	stB0 := qubits.NewQubitStateManagerFrom(zero, []int32{q1}) // |0>_B
	stHA := oneQubit(stA0, hadamard)                           // H|0>_A
	stBell := twoQubit(stHA, stB0, cnot)                       // (|00>+|11>)/√2
	stZ := twoQubitOnState(stBell, zOnQ0)                      // Alice: Z
	stX := twoQubitOnState(stZ, xOnQ0)                         // Alice: X → "11"
	stCXB := twoQubitOnState(stX, cnot)                        // Bob: CX
	stHB := twoQubitOnState(stCXB, hOnQ0)                      // Bob: H

	// The decoded 2-qubit state must be exactly |11> = index 3.
	best, bestIdx := float32(0), int32(-1)
	for i, a := range stHB.Amptitude {
		p := real(a)*real(a) + imag(a)*imag(a)
		if p > best {
			best, bestIdx = p, int32(i)
		}
	}
	if bestIdx != 3 || best < 0.99 {
		fmt.Printf("warning: decode state is not |11> (best index %d, p=%.3f)\n", bestIdx, best)
	} else {
		fmt.Println("decode check: final state is |11> (message 11)")
	}

	// --- components -----------------------------------------------------
	var comps []components.Component

	// Sources.
	// NewSourceGateWithID pins the emitted qubit's modifier ID; the plain
	// NewSourceGate only generates one lazily on the first Update, which
	// would leave the wrong ID in the save.
	sA := components.NewSourceGateWithID(-1500, -150, 90, config.GateColor, "|0> A", zero, q0)
	sB := components.NewSourceGateWithID(-1500, 150, 90, config.GateColor, "|0> B", zero, q1)
	comps = append(comps, sA, sB)

	// Source output systems.
	qsA := system(sA.OutHook.Center.X, sA.OutHook.Center.Y, stA0)
	qsB := system(sB.OutHook.Center.X, sB.OutHook.Center.Y, stB0)
	sA.OutHook.Connect(qsA)
	sA.OutHook.ConnectInfo(qsA)
	sB.OutHook.Connect(qsB)
	sB.OutHook.ConnectInfo(qsB)
	comps = append(comps, qsA, qsB)

	// Bell preparation: H on qubit 0, then CX.
	hA := gate(-1050, -150, "H", hadamard, 1)
	hA.HookList[0].Connect(qsA.QubitDeterminatorList[0])
	qsHA := system(hA.OutPutHook[0].Center.X, hA.OutPutHook[0].Center.Y, stHA)
	hA.OutPutHook[0].Connect(qsHA)
	comps = append(comps, hA, qsHA)

	cxPrep := gate(-750, 0, "CX", cnot, 2)
	cxPrep.HookList[0].Connect(qsHA.QubitDeterminatorList[0])
	cxPrep.HookList[1].Connect(qsB.QubitDeterminatorList[0])
	qsBell := system(cxPrep.OutPutHook[0].Center.X, cxPrep.OutPutHook[0].Center.Y, stBell)
	cxPrep.OutPutHook[0].Connect(qsBell)
	comps = append(comps, cxPrep, qsBell)

	// Alice encodes "11": Z then X, both on qubit 0 only.
	msgZ := gate(-300, 0, "Z", zOnQ0, 2)
	msgZ.HookList[0].Connect(qsBell.QubitDeterminatorList[0])
	msgZ.HookList[1].Connect(qsBell.QubitDeterminatorList[1])
	qsZ := system(msgZ.OutPutHook[0].Center.X, msgZ.OutPutHook[0].Center.Y, stZ)
	msgZ.OutPutHook[0].Connect(qsZ)
	comps = append(comps, msgZ, qsZ)

	msgX := gate(150, 0, "X", xOnQ0, 2)
	msgX.HookList[0].Connect(qsZ.QubitDeterminatorList[0])
	msgX.HookList[1].Connect(qsZ.QubitDeterminatorList[1])
	qsX := system(msgX.OutPutHook[0].Center.X, msgX.OutPutHook[0].Center.Y, stX)
	msgX.OutPutHook[0].Connect(qsX)
	comps = append(comps, msgX, qsX)

	// Bob decodes: CX then H on qubit 0.
	cxBob := gate(600, 0, "CX", cnot, 2)
	cxBob.HookList[0].Connect(qsX.QubitDeterminatorList[0])
	cxBob.HookList[1].Connect(qsX.QubitDeterminatorList[1])
	qsCXB := system(cxBob.OutPutHook[0].Center.X, cxBob.OutPutHook[0].Center.Y, stCXB)
	cxBob.OutPutHook[0].Connect(qsCXB)
	comps = append(comps, cxBob, qsCXB)

	hBob := gate(900, 0, "H", hOnQ0, 2)
	hBob.HookList[0].Connect(qsCXB.QubitDeterminatorList[0])
	hBob.HookList[1].Connect(qsCXB.QubitDeterminatorList[1])
	qsHB := system(hBob.OutPutHook[0].Center.X, hBob.OutPutHook[0].Center.Y, stHB)
	hBob.OutPutHook[0].Connect(qsHB)
	comps = append(comps, hBob, qsHB)

	// Measurements: M2 collapse gates, one per qubit. ForceMode 2 forces
	// the outcome |1>, matching the decoded |11> message so both lights
	// stay lit.
	measure := func(x, y float32, det *components.QubitDeterminator) (*components.CollapseGate, *components.LogicalBit) {
		m := components.NewCollapseGate(x, y, glob.GateRadius, config.GateColor, "M2")
		m.ForceMode = 2
		m.HookList[0].Connect(det)
		bit := components.NewLogicalBit(m.OutPutHook[0].Center.X, m.OutPutHook[0].Center.Y, glob.QubitSystemRadius/2, 1)
		m.OutPutHook[0].Connect(bit)
		comps = append(comps, m, bit)
		return m, bit
	}

	_, bitA := measure(1200, -150, qsHB.QubitDeterminatorList[0])
	_, bitB := measure(1200, 150, qsHB.QubitDeterminatorList[1])

	// Lights display the measured bits.
	lightCol := rl.NewColor(241, 121, 0, 255)
	lightA := components.NewLight(1550, -150, lightCol)
	lightA.InHook.Connect(bitA)
	lightB := components.NewLight(1900, 150, lightCol)
	lightB.InHook.Connect(bitB)
	comps = append(comps, lightA, lightB)

	// Stage captions.
	label := func(x, y float32, text string) {
		comps = append(comps, components.NewLabel(x, y, text, 18, rl.White))
	}
	label(-1300, -330, "Alice: prepare Bell pair (H, CX)")
	label(-250, 260, "Alice: encode message 11 (Z, X)")
	label(650, 260, "Bob: decode (CX, H)")
	label(1300, -330, "Measure & display (M2 -> bits -> lights)")

	rw.PushComponent(comps...)

	// Frame the whole circuit: point the camera at the centroid.
	var cx, cy float32
	var n int
	for _, c := range comps {
		cx += c.GetCircle().Center.X
		cy += c.GetCircle().Center.Y
		n++
	}
	if n > 0 {
		rw.Camera.Target = rl.NewVector2(cx/float32(n), cy/float32(n))
	}

	// --- serialize ------------------------------------------------------
	data := rw.SaveState()
	outPath := filepath.Join("saves", "superdense.qsim")
	if err := os.MkdirAll("saves", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, []byte(data), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes, %d components)\n", outPath, len(data), len(comps))

	// --- verify: the file must round-trip through the real loader --------
	loaded := windows.LoadState(data)
	if len(loaded) != 1 {
		fmt.Fprintf(os.Stderr, "LoadState returned %d windows, want 1\n", len(loaded))
		os.Exit(1)
	}
	rlw, ok := loaded[0].(*windows.RenderWindow)
	if !ok {
		fmt.Fprintln(os.Stderr, "loaded window is not a *RenderWindow")
		os.Exit(1)
	}
	if len(rlw.WComp) != len(comps) {
		fmt.Fprintf(os.Stderr, "loaded %d components, want %d\n", len(rlw.WComp), len(comps))
		os.Exit(1)
	}
	// Every hook that claims to be hooked must resolve to a live object.
	for _, c := range rlw.WComp {
		if g, ok := c.(*components.Gate); ok {
			for _, hk := range g.HookList {
				if hk.IsHooked && utils.GetObjectFromID(hk.TargetID) == nil {
					fmt.Fprintf(os.Stderr, "gate %s hook %s dangling: target %d\n", g.Label, hk.Label, hk.TargetID)
					os.Exit(1)
				}
			}
		}
		if m, ok := c.(*components.CollapseGate); ok {
			for _, hk := range m.HookList {
				if hk.IsHooked && utils.GetObjectFromID(hk.TargetID) == nil {
					fmt.Fprintf(os.Stderr, "collapse %s hook %s dangling: target %d\n", m.Label, hk.Label, hk.TargetID)
					os.Exit(1)
				}
			}
		}
	}
	fmt.Println("round-trip check: ok, all hooks resolve")
}
