// Command gen-qft is the second WORKED EXAMPLE of authoring a .qsim save
// programmatically. It builds a 3-qubit Quantum Fourier Transform circuit:
//
//	q0 ──H──╭R(π/2)──╭R(π/4)───────────────╭R(π/4)──>> (bit-reversed)
//	q1 ─────┘─────────┼────────── H ─╭R(π/2)┼─────────
//	q2 ───────────────┘                    ┘────── H ─
//
// Because the engine wires whole qubit systems from gate to gate (a gate's
// outputs are the merged system of all its inputs), the controlled rotations
// that couple an already-entangled qubit with a fresh one are written as
// 3-qubit 8×8 operations: CR(q0,q2), H(q1), CR(q1,q2) and H(q2) each take
// all three determinators of the running system as inputs.
//
// For the |000> input every controlled rotation sees a control bit of 0, so
// the state collapses to H⊗H⊗H|000> = the uniform superposition over all 8
// basis states (exactly QFT|0>). The generator asserts that analytically.
//
// Run it from the repository root:
//
//	go run ./cmd/examples/gen-qft
//
// then load saves/qft.qsim in the app. See docs/qsim-format.md.
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
)

// kron builds the n-qubit operation that applies the 2×2 op to bit position
// `bit` (0 = LSB) and identity to all other qubits. In the running 3-qubit
// system the modifier order is (q0, q1, q2), so q0 = bit 2, q1 = bit 1,
// q2 = bit 0.
func kron(n, bit int, op [][]complex64) [][]complex64 {
	size := 1 << n
	m := make([][]complex64, size)
	for i := range m {
		m[i] = make([]complex64, size)
	}
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if (i &^ (1 << bit)) == (j &^ (1 << bit)) {
				m[i][j] = op[(i>>bit)&1][(j>>bit)&1]
			}
		}
	}
	return m
}

// cpDiag builds the n-qubit diagonal controlled-phase matrix that multiplies
// index k by v wherever both the control and the target bit are set. The
// index maps are per gate: in the 3-qubit space q0 = bit 2, q1 = bit 1,
// q2 = bit 0.
func cpDiag(n int, entries map[int]complex64) [][]complex64 {
	size := 1 << n
	m := make([][]complex64, size)
	for i := 0; i < size; i++ {
		m[i] = make([]complex64, size)
		m[i][i] = 1
	}
	for idx, v := range entries {
		m[idx][idx] = v
	}
	return m
}

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

func main() {
	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	rw.Rename("QFT (3 qubits)")

	// --- quantum states -------------------------------------------------
	//
	// The engine's Merge(first, second) puts `first` on the high bits, so in
	// a {q0,q1,q2} system the amplitude index is i = q0<<2 | q1<<1 | q2.
	// Register real modifier IDs: Assign resolves them against the attribute
	// manager while building the qubit visuals.
	q0 := attributes.GenerateQubitModifierID()
	q1 := attributes.GenerateQubitModifierID()
	q2 := attributes.GenerateQubitModifierID()

	zero := []complex64{1, 0}

	stQ0 := qubits.NewQubitStateManagerFrom(zero, []int32{q0}) // |0>_A
	stQ1 := qubits.NewQubitStateManagerFrom(zero, []int32{q1}) // |0>_B
	stQ2 := qubits.NewQubitStateManagerFrom(zero, []int32{q2}) // |0>_C

	// Stage 1: H on q0.
	stH := cloneState(stQ0)
	stH.Multiply(hadamard, 1)

	// Stage 2: CR(π/2) on (q0,q1). 2-qubit diagonal: e^{iπ/2} at |11>.
	cr2 := cpDiag(2, map[int]complex64{3: 1i})
	stCR2 := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	stCR2.Merge(stH)  // q0 = high bit
	stCR2.Merge(stQ1) // q1 = low bit
	stCR2.Multiply(cr2, 2)

	// Stage 3: CR(π/4) control q0 target q2. Fires at |101> and |111>
	// (bits 2 and 0 set), i.e. indices 5 and 7.
	crQ0Q2 := cpDiag(3, map[int]complex64{
		5: complex64(complex(float32(math.Cos(math.Pi/4)), float32(math.Sin(math.Pi/4)))),
		7: complex64(complex(float32(math.Cos(math.Pi/4)), float32(math.Sin(math.Pi/4)))),
	})
	stCR4 := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	stCR4.Merge(stCR2)
	stCR4.Merge(stQ2)
	stCR4.Multiply(crQ0Q2, 3)

	// Stage 4: H on q1 (bit 1) over the whole 3-qubit system.
	stHQ1 := cloneState(stCR4)
	stHQ1.Multiply(kron(3, 1, hadamard), 3)

	// Stage 5: CR(π/2) control q1 target q2. Fires at |011> and |111>
	// (bits 1 and 0 set), i.e. indices 3 and 7.
	crQ1Q2 := cpDiag(3, map[int]complex64{3: 1i, 7: 1i})
	stCR2b := cloneState(stHQ1)
	stCR2b.Multiply(crQ1Q2, 3)

	// Stage 6: H on q2 (bit 0).
	stHQ2 := cloneState(stCR2b)
	stHQ2.Multiply(kron(3, 0, hadamard), 3)

	// The decoded state must be the uniform superposition: every amplitude
	// equals 1/√8 (0.353553...).
	ok := true
	for _, a := range stHQ2.Amptitude {
		if math.Abs(float64(real(a))-1/math.Sqrt2/math.Sqrt2/math.Sqrt2) > 1e-6 ||
			math.Abs(float64(imag(a))) > 1e-6 {
			ok = false
		}
	}
	if !ok {
		fmt.Println("warning: QFT|0> is not the uniform superposition")
		for i, a := range stHQ2.Amptitude {
			fmt.Printf("  %d: %+.5f%+.5fi\n", i, real(a), imag(a))
		}
	} else {
		fmt.Println("QFT check: final state is the uniform superposition over 8 basis states")
	}

	// --- components -----------------------------------------------------
	var comps []components.Component

	// Sources.
	sA := components.NewSourceGateWithID(-1500, -150, 90, config.GateColor, "|0> A", zero, q0)
	sB := components.NewSourceGateWithID(-1500, 0, 90, config.GateColor, "|0> B", zero, q1)
	sC := components.NewSourceGateWithID(-1500, 150, 90, config.GateColor, "|0> C", zero, q2)
	comps = append(comps, sA, sB, sC)

	qsA := system(sA.OutHook.Center.X, sA.OutHook.Center.Y, stQ0)
	qsB := system(sB.OutHook.Center.X, sB.OutHook.Center.Y, stQ1)
	qsC := system(sC.OutHook.Center.X, sC.OutHook.Center.Y, stQ2)
	sA.OutHook.Connect(qsA)
	sA.OutHook.ConnectInfo(qsA)
	sB.OutHook.Connect(qsB)
	sB.OutHook.ConnectInfo(qsB)
	sC.OutHook.Connect(qsC)
	sC.OutHook.ConnectInfo(qsC)
	comps = append(comps, qsA, qsB, qsC)

	// Stage 1: H on q0 only.
	hA := gate(-1050, -150, "H", hadamard, 1)
	hA.HookList[0].Connect(qsA.QubitDeterminatorList[0])
	qsH := system(hA.OutPutHook[0].Center.X, hA.OutPutHook[0].Center.Y, stH)
	hA.OutPutHook[0].Connect(qsH)
	comps = append(comps, hA, qsH)

	// Stage 2: CR(π/2) on (q0,q1), 2 inputs.
	cr2g := gate(-800, -75, "R(π/2)", cr2, 2)
	cr2g.HookList[0].Connect(qsH.QubitDeterminatorList[0]) // q0
	cr2g.HookList[1].Connect(qsB.QubitDeterminatorList[0]) // q1
	qsCR2 := system(cr2g.OutPutHook[0].Center.X, cr2g.OutPutHook[0].Center.Y, stCR2)
	cr2g.OutPutHook[0].Connect(qsCR2)
	comps = append(comps, cr2g, qsCR2)

	// Stage 3: CR(π/4) control q0 target q2 — the first 3-qubit gate. All
	// three wires of the running system become inputs.
	cr4g := gate(-500, 0, "R(π/4)", crQ0Q2, 3)
	cr4g.HookList[0].Connect(qsCR2.QubitDeterminatorList[0]) // q0
	cr4g.HookList[1].Connect(qsCR2.QubitDeterminatorList[1]) // q1
	cr4g.HookList[2].Connect(qsC.QubitDeterminatorList[0])   // q2
	qsCR4 := system(cr4g.OutPutHook[0].Center.X, cr4g.OutPutHook[0].Center.Y, stCR4)
	cr4g.OutPutHook[0].Connect(qsCR4)
	comps = append(comps, cr4g, qsCR4)

	// Stage 4: H on q1 over the 3-qubit system.
	hq1 := gate(-200, 0, "H", kron(3, 1, hadamard), 3)
	hq1.HookList[0].Connect(qsCR4.QubitDeterminatorList[0])
	hq1.HookList[1].Connect(qsCR4.QubitDeterminatorList[1])
	hq1.HookList[2].Connect(qsCR4.QubitDeterminatorList[2])
	qsHQ1 := system(hq1.OutPutHook[0].Center.X, hq1.OutPutHook[0].Center.Y, stHQ1)
	hq1.OutPutHook[0].Connect(qsHQ1)
	comps = append(comps, hq1, qsHQ1)

	// Stage 5: CR(π/2) control q1 target q2.
	cr2bg := gate(100, 0, "R(π/2)", crQ1Q2, 3)
	cr2bg.HookList[0].Connect(qsHQ1.QubitDeterminatorList[0])
	cr2bg.HookList[1].Connect(qsHQ1.QubitDeterminatorList[1])
	cr2bg.HookList[2].Connect(qsHQ1.QubitDeterminatorList[2])
	qsCR2b := system(cr2bg.OutPutHook[0].Center.X, cr2bg.OutPutHook[0].Center.Y, stCR2b)
	cr2bg.OutPutHook[0].Connect(qsCR2b)
	comps = append(comps, cr2bg, qsCR2b)

	// Stage 6: H on q2.
	hq2 := gate(400, 0, "H", kron(3, 0, hadamard), 3)
	hq2.HookList[0].Connect(qsCR2b.QubitDeterminatorList[0])
	hq2.HookList[1].Connect(qsCR2b.QubitDeterminatorList[1])
	hq2.HookList[2].Connect(qsCR2b.QubitDeterminatorList[2])
	qsHQ2 := system(hq2.OutPutHook[0].Center.X, hq2.OutPutHook[0].Center.Y, stHQ2)
	hq2.OutPutHook[0].Connect(qsHQ2)
	comps = append(comps, hq2, qsHQ2)

	// Stage captions.
	label := func(x, y float32, text string) {
		comps = append(comps, components.NewLabel(x, y, text, 18, rl.White))
	}
	label(-1450, 260, "3-qubit QFT, input |000>: H, CR(π/2), CR(π/4), H, CR(π/2), H")
	label(-1450, 300, "controlled rotations never fire on |0> → uniform superposition (bit-reversed output, no trailing SWAP)")

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
	outPath := filepath.Join("saves", "qft.qsim")
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
	check := func(label string, hk *components.Hook) {
		if hk.IsHooked && utils.GetObjectFromID(hk.TargetID) == nil {
			fmt.Fprintf(os.Stderr, "%s hook dangling: target %d\n", label, hk.TargetID)
			os.Exit(1)
		}
	}
	for _, c := range rlw.WComp {
		switch v := c.(type) {
		case *components.SourceGate:
			check("source", v.OutHook)
		case *components.Gate:
			for _, hk := range v.HookList {
				check("gate "+v.Label, hk)
			}
		}
	}
	fmt.Println("round-trip check: ok, all hooks resolve")
}
