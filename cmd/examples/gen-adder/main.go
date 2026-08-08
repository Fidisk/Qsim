// Command gen-adder is a worked example of authoring a .qsim save
// programmatically: a 4-bit quantum ripple-carry adder, 7 + 5 = 12
// (saves/adder.qsim). The simple 2-bit version lives in gen-adder2.
//
//	carry chain c0..c4, A register a0..a3, B register b0..b3
//	(sum lands in the B register, final carry in c4)
//
// Each bit i is ONE 4-input gate whose 16×16 operation (a real permutation,
// hence unitary) maps, in the basis |c_{i+1} a_i b_i c_i>:
//
//	c_{i+1} -> majority(a_i,b_i,c_i)  (XORed onto the ancilla's input value,
//	                                  which is always 0, to stay bijective)
//	b_i     -> s_i = a_i XOR b_i XOR c_i        (the sum bit)
//	a_i, c_i -> unchanged
//
// The gate's input hooks are ordered (c_{i+1}, a_i, b_i, c_i) — fresh
// ancilla first — so the engine's merge keeps the carry-out c_{i+1} as the
// LEADING modifier of the merged system, and every gate's inputs are the
// leading modifiers of its merged input (required: the engine's SwapColumn
// only behaves correctly for inputs that already lead the merged state).
//
// Layout: a qubit system's drawn state grid has 2^size cells and grows
// exponentially with the qubit count, so a 13-qubit system would be drawn
// 12800x6400 world units and cover the whole circuit. To keep every system
// small, each bit ends with a chain of M2 collapse measurements (a_i, b_i,
// c_i): the remainder output (R) of each M2 is the input of the next, so
// the running system never exceeds 4 qubits (400x400 grid) and the dead
// qubits become logical bits driving the lights — the full readout.
//
// Because the inputs are basis states every intermediate state is a basis
// state. The generator mirrors the engine's merge/SwapColumn/Multiply and
// M2 remainder math exactly, asserts the adder against all 2^(2n) input
// pairs, and styles the layout per docs/qsim-style.md.
//
// Run it from the repository root:
//
//	go run ./cmd/examples/gen-adder
//
// then load saves/adder.qsim in the app. All annotations are TextBoxes
// (see docs/qsim-style.md).
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

// adderGateOp builds the 16×16 permutation for one adder bit. Basis
// |I0 I1 I2 I3> = |c_out a b c_in>; the target qubits are c_out (bit 3,
// gets majority ⊕ input) and b (bit 1, gets the sum). The map is a
// bijection, so the matrix is a valid unitary.
func adderGateOp() [][]complex64 {
	m := make([][]complex64, 16)
	for i := range m {
		m[i] = make([]complex64, 16)
	}
	for x := 0; x < 16; x++ {
		cino := (x >> 3) & 1
		a := (x >> 2) & 1
		b := (x >> 1) & 1
		cin := x & 1
		maj := (a & b) ^ (a & cin) ^ (b & cin)
		s := a ^ b ^ cin
		y := ((maj ^ cino) << 3) | (a << 2) | (s << 1) | cin
		m[y][x] = 1
	}
	return m
}

func gate(x, y float32, label string, op [][]complex64, inputCount int32) *components.Gate {
	return components.NewGate(x, y, glob.GateRadius, config.GateColor, label, op, inputCount)
}

func system(x, y float32, state *qubits.QubitStateManager) *components.QubitsSystem {
	qs := components.NewQubitsSystem(x, y, glob.QubitSystemRadius, config.QubitSystemColor)
	qs.Assign(state)
	return qs
}

func basisState(value int) []complex64 {
	if value == 1 {
		return []complex64{0, 1}
	}
	return []complex64{1, 0}
}

func detOf(qs *components.QubitsSystem, modID int32) *components.QubitDeterminator {
	for _, d := range qs.QubitDeterminatorList {
		if d.ModifierID == modID {
			return d
		}
	}
	return nil
}

// measureOut mirrors CollapseGate.emitOutcome for basis states: it returns
// the measured value of the qubit and the conditioned remainder state (the
// input state with that qubit removed). Deterministic for basis states.
func measureOut(st *qubits.QubitStateManager, id int32) (value int, rest *qubits.QubitStateManager) {
	pos := st.FindID(id)
	bitPos := st.Size - 1 - pos
	idx := 0
	for i, a := range st.Amptitude {
		if a != 0 {
			idx = i
			break
		}
	}
	value = (idx >> bitPos) & 1
	newMods := append([]int32{}, st.ModifierID[:pos]...)
	newMods = append(newMods, st.ModifierID[pos+1:]...)
	hi := (idx >> (bitPos + 1)) << bitPos
	lo := idx & ((1 << bitPos) - 1)
	amps := make([]complex64, 1<<(st.Size-1))
	amps[hi|lo] = 1
	rest = qubits.NewQubitStateManagerFrom(amps, newMods)
	return
}

func main() {
	const bits = 4

	// Demo operands with a visible carry chain: 7 + 5 = 12.
	A, B := 7, 5 // 0111 + 0101 = 1100

	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	rw.Rename(fmt.Sprintf("%d-bit quantum adder: %d + %d = %d", bits, A, B, A+B))

	// --- registers -------------------------------------------------------
	// Per-bit system order is (c_{i+1}, a_i, b_i, c_i). Lanes are grouped by
	// bit, MSB on top, 200px pitch.
	newID := attributes.GenerateQubitModifierID
	aID := make([]int32, bits)
	bID := make([]int32, bits)
	cID := make([]int32, bits+1)
	for i := range cID {
		cID[i] = newID()
	}
	for i := 0; i < bits; i++ {
		aID[i] = newID()
		bID[i] = newID()
	}

	// Lane order (unique qubits): bit 0 (a0, b0, c0), carry c1, bit 1
	// (a1, b1), carry c2, ... final carry c_n.
	var lanes []int32
	for i := 0; i < bits; i++ {
		lanes = append(lanes, aID[i], bID[i], cID[i])
		if i < bits-1 {
			lanes = append(lanes, cID[i+1])
		}
	}
	lanes = append(lanes, cID[bits])

	// --- components ------------------------------------------------------
	var comps []components.Component
	// All canvas text lives in a TextBox (docs/qsim-style.md §1).
	text := func(x, y float32, s string, size int32, width float32) {
		tb := components.NewTextBox(x, y, width, float32(size)+16, size)
		tb.Text = s
		comps = append(comps, tb)
	}

	// Sources + per-qubit running systems.
	sysOf := map[int32]*components.QubitsSystem{} // modID -> current system
	states := map[int32]*qubits.QubitStateManager{}

	addSource := func(x, y float32, name string, bit int, id int32) {
		s := components.NewSourceGateWithID(x, y, 90, config.GateColor, name, basisState(bit), id)
		comps = append(comps, s)
		st := qubits.NewQubitStateManagerFrom(basisState(bit), []int32{id})
		states[id] = st
		qs := system(s.OutHook.Center.X, s.OutHook.Center.Y, st)
		s.OutHook.Connect(qs)
		s.OutHook.ConnectInfo(qs)
		comps = append(comps, qs)
		sysOf[id] = qs
	}

	laneOf := map[int32]float32{}
	topY := float32(-2400)
	for i, id := range lanes {
		laneOf[id] = topY + float32(i)*200
	}
	srcX := float32(-2000)
	// One source per qubit, in the lane order.
	for _, id := range lanes {
		val := 0
		name := fmt.Sprintf("Q%d", id)
		for i := 0; i < bits; i++ {
			switch id {
			case aID[i]:
				val, name = (A>>i)&1, fmt.Sprintf("A%d = %d", i, (A>>i)&1)
			case bID[i]:
				val, name = (B>>i)&1, fmt.Sprintf("B%d = %d", i, (B>>i)&1)
			case cID[i]:
				val, name = 0, fmt.Sprintf("C%d = 0", i)
			}
		}
		if id == cID[bits] {
			name = fmt.Sprintf("C%d = 0", bits)
		}
		addSource(srcX, laneOf[id], name, val, id)
	}

	// Register captions on the left of the sources.
	text(-2400, topY-100, fmt.Sprintf("carry chain c0..c%d", bits), 18, 260)
	text(-2400, laneOf[aID[bits-1]]+100, "A register (left), B register -> sum (right)", 18, 420)
	text(-2400, laneOf[cID[0]]-100, "input register bits (LSB at the bottom)", 18, 360)

	// Title + protocol explanation.
	text(-2000, -2700, fmt.Sprintf("%d-bit quantum ripple-carry adder: %d + %d = %d", bits, A, B, A+B), 24, 620)
	expl := components.NewTextBox(-2000, -2600, 1000, 60, 14)
	expl.Text = "One 4-input gate per bit computes the sum bit s_i = a_i XOR b_i XOR c_i into the B register and the carry-out c_{i+1} = majority(a_i,b_i,c_i). M2 measurements after each bit read a_i, b_i and c_i off the chain, keeping the running system small (a qubit system's drawn grid doubles per qubit)."
	comps = append(comps, expl)

	// --- gates + measurement chain ---------------------------------------
	// Mirror the engine's Gate.CalculateOutPut: merge the input parent
	// systems in hook order (deduplicated by system), SwapColumn inputs to
	// the top, Multiply, and keep the merged state as the output system.
	apply := func(ids []int32, op [][]complex64, x, y float32, name string) {
		var parts []*qubits.QubitStateManager
		for _, id := range ids {
			st := states[id]
			dup := false
			for _, p := range parts {
				if p == st {
					dup = true
					break
				}
			}
			if !dup {
				parts = append(parts, st)
			}
		}
		merged := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
		for _, p := range parts {
			merged.Merge(p)
		}
		for i, id := range ids {
			merged.SwapColumn(int32(i), merged.FindID(id))
		}
		merged.Multiply(op, int32(len(ids)))

		g := gate(x, y, name, op, int32(len(ids)))
		for j, id := range ids {
			g.HookList[j].Connect(detOf(sysOf[id], id))
		}
		qs := system(g.OutPutHook[0].Center.X, g.OutPutHook[0].Center.Y, merged)
		g.OutPutHook[0].Connect(qs)
		comps = append(comps, g, qs)

		for _, id := range merged.ModifierID {
			states[id] = merged
			sysOf[id] = qs
		}
	}

	// measure emits an M2 + logical bit + light for one qubit, and wires
	// the remainder (R) as the next system in the chain.
	lightCol := rl.NewColor(241, 121, 0, 255)
	measure := func(x float32, id int32, expected int) {
		val, rest := measureOut(states[id], id)
		if val != expected {
			fmt.Printf("ERROR: measured bit %d = %d, want %d\n", id, val, expected)
			os.Exit(1)
		}
		m := components.NewCollapseGate(x, laneOf[id], glob.GateRadius, config.GateColor, "M2")
		m.ForceMode = int32(val) + 1
		m.HookList[0].Connect(detOf(sysOf[id], id))
		bit := components.NewLogicalBit(m.OutPutHook[0].Center.X, m.OutPutHook[0].Center.Y, 15, int32(val))
		m.OutPutHook[0].Connect(bit)
		light := components.NewLight(x+550, laneOf[id], lightCol)
		light.InHook.Connect(bit)
		comps = append(comps, m, bit, light)
		if len(rest.ModifierID) > 0 {
			rq := system(m.OutPutHook[1].Center.X, m.OutPutHook[1].Center.Y, rest)
			m.OutPutHook[1].Connect(rq)
			comps = append(comps, rq)
			for _, rid := range rest.ModifierID {
				states[rid] = rest
				sysOf[rid] = rq
			}
		}
	}

	op := adderGateOp()
	const bitGap float32 = 2100 // ADD gate + 3 M2 columns + remainder room
	for i := 0; i < bits; i++ {
		x0 := float32(-1200) + float32(i)*bitGap
		apply([]int32{cID[i+1], aID[i], bID[i], cID[i]}, op, x0, laneOf[cID[i+1]], fmt.Sprintf("ADD bit %d", i))
		// Step header above the block.
		text(x0+600, -2700+float32(i)*60+20,
			fmt.Sprintf("Bit %d: s%d = A%d XOR B%d XOR C%d, carry out -> C%d", i, i, i, i, i, i+1),
			16, 520)
		// Separator after the block (except the last).
		if i < bits-1 {
			sep := components.NewLineDraw(x0+bitGap-450, -2640+float32(i)*60)
			sep.End = rl.Vector2{X: x0 + bitGap - 450, Y: laneOf[cID[i+1]] + 200}
			sep.LineColor = rl.NewColor(90, 90, 90, 255)
			sep.Placing = false
			comps = append(comps, sep)
		}
		// Read the three dead qubits of this bit off the chain. c_i is the
		// carry-IN of bit i (i.e. the carry-out of bit i-1, 0 for bit 0).
		mx := x0 + 450
		measure(mx, aID[i], (A>>i)&1)
		mx += 600
		measure(mx, bID[i], ((A+B)>>i)&1)
		mx += 600
		carryIn := 0
		if i > 0 {
			carryIn = carryBit(bits, A, B, i-1)
		}
		measure(mx, cID[i], carryIn)
	}
	// Final carry-out.
	mx := float32(-1200) + float32(bits)*bitGap + 450
	measure(mx, cID[bits], (A+B)>>bits)
	text(mx, -2700, fmt.Sprintf("Sum (B register, MSB first) and carry-out C%d -> lights", bits), 16, 520)

	// Universal-gate-style caption: the ADD operation is one distinct
	// 16×16 type, so it is explained once (see docs/qsim-style.md §4).
	text(-1200, laneOf[cID[1]]+320,
		"ADD gate (16x16, one type): in basis |C_out A B C_in> it maps (C_out,A,B,C_in) -> (C_out XOR majority(A,B,C_in), A, A XOR B XOR C_in, C_in); C_in is always 0 here",
		14, 1200)

	rw.PushComponent(comps...)

	// Frame the circuit: centroid + zoom that fits the bounds.
	var cx, cy float32
	minX, maxX := float32(1e9), float32(-1e9)
	minY, maxY := float32(1e9), float32(-1e9)
	n := 0
	for _, c := range comps {
		cc := c.GetCircle().Center
		cx += cc.X
		cy += cc.Y
		minX = float32(math.Min(float64(minX), float64(cc.X)))
		maxX = float32(math.Max(float64(maxX), float64(cc.X)))
		minY = float32(math.Min(float64(minY), float64(cc.Y)))
		maxY = float32(math.Max(float64(maxY), float64(cc.Y)))
		n++
	}
	if n > 0 {
		rw.Camera.Target = rl.NewVector2(cx/float32(n), cy/float32(n))
		zx := 1600 / (maxX - minX + 300)
		zy := 900 / (maxY - minY + 200)
		rw.Camera.Zoom = float32(math.Min(float64(zx), float64(zy))) * 0.9
		if rw.Camera.Zoom > 1 {
			rw.Camera.Zoom = 1
		}
	}

	// --- verify the adder logic on all inputs ----------------------------
	ok := true
	for a := 0; a < 1<<bits; a++ {
		for b := 0; b < 1<<bits; b++ {
			sum, carry := adderBits(bits, a, b)
			if sum != (a+b)&((1<<bits)-1) || carry != (a+b)>>bits {
				ok = false
				fmt.Printf("adder fail: %d + %d -> sum %d carry %d (want %d, %d)\n", a, b, sum, carry, (a+b)&((1<<bits)-1), (a+b)>>bits)
			}
		}
	}
	if !ok {
		fmt.Println("ERROR: ripple-carry adder truth table is wrong")
		os.Exit(1)
	}
	fmt.Printf("adder check: %d + %d = %d; all %d (A,B) pairs verified\n", A, B, A+B, 1<<(2*bits))

	// The serialized states must all be basis states (single 1 each) and
	// every system must stay small (the drawn grid of a 13-qubit system
	// would cover the whole circuit).
	for _, st := range states {
		ones := 0
		for _, a := range st.Amptitude {
			if a != 0 {
				ones++
			}
		}
		if ones != 1 {
			fmt.Println("ERROR: intermediate state is not a basis state")
			os.Exit(1)
		}
		if st.Size > 4 {
			fmt.Printf("ERROR: running system grew to %d qubits; the drawn grid would overlap the circuit\n", st.Size)
			os.Exit(1)
		}
	}

	// --- serialize -------------------------------------------------------
	data := rw.SaveState()
	outName := fmt.Sprintf("adder%d.qsim", bits)
	if bits == 4 {
		outName = "adder.qsim"
	}
	outPath := filepath.Join("saves", outName)
	if err := os.MkdirAll("saves", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, []byte(data), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes, %d components)\n", outPath, len(data), len(comps))

	// --- round-trip through the real loader ------------------------------
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
	checkHooks := func(what string, hooks []*components.Hook) {
		for _, hk := range hooks {
			if hk.IsHooked && utils.GetObjectFromID(hk.TargetID) == nil {
				fmt.Fprintf(os.Stderr, "%s hook %s dangling: target %d\n", what, hk.Label, hk.TargetID)
				os.Exit(1)
			}
		}
	}
	for _, c := range rlw.WComp {
		switch v := c.(type) {
		case *components.SourceGate:
			checkHooks("source", []*components.Hook{v.OutHook})
		case *components.Gate:
			checkHooks("gate "+v.Label, v.HookList)
		case *components.CollapseGate:
			checkHooks("collapse "+v.Label, v.HookList)
		case *components.Light:
			checkHooks("light", []*components.Hook{v.InHook})
		}
	}
	fmt.Println("round-trip check: ok, all hooks resolve")
}

// carryBit returns the carry-out of bit i (c_{i+1}) when adding a and b.
func carryBit(n, a, b, i int) int {
	ab := make([]int, n)
	bb := make([]int, n)
	for k := 0; k < n; k++ {
		ab[k] = (a >> k) & 1
		bb[k] = (b >> k) & 1
	}
	cb := make([]int, n+1)
	for k := 0; k <= i; k++ {
		maj := (ab[k] & bb[k]) ^ (ab[k] & cb[k]) ^ (bb[k] & cb[k])
		cb[k+1] ^= maj
	}
	return cb[i+1]
}

// adderBits applies the adder truth tables to the classical bits of a and b
// and returns the n-bit sum and the carry-out.
func adderBits(n, a, b int) (sum, carry int) {
	ab := make([]int, n)
	bb := make([]int, n)
	for i := 0; i < n; i++ {
		ab[i] = (a >> i) & 1
		bb[i] = (b >> i) & 1
	}
	cb := make([]int, n+1)
	for i := 0; i < n; i++ {
		maj := (ab[i] & bb[i]) ^ (ab[i] & cb[i]) ^ (bb[i] & cb[i])
		cb[i+1] ^= maj
		bb[i] = ab[i] ^ bb[i] ^ cb[i]
	}
	for i := n - 1; i >= 0; i-- {
		sum = sum<<1 | bb[i]
	}
	return sum, cb[n]
}
