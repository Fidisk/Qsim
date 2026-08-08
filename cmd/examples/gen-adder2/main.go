// Command gen-adder2 is a worked example of authoring a .qsim save
// programmatically: the simplest possible 2-bit quantum adder, 2 + 1 = 3.
//
//	6 qubits: A register (a1 a0), B register (b1 b0 -> sum), carries c1 c2
//	One 6-input gate (64x64 permutation, hence unitary) computes the whole
//	addition in the basis |a1 a0 b1 b0 c1 c2>:
//
//	    s0  = a0 XOR b0                 (sum bit 0 -> b0)
//	    c1' = c1 XOR (a0 AND b0)        (carry into bit 1; ancilla is 0)
//	    s1  = a1 XOR b1 XOR c1'         (sum bit 1 -> b1)
//	    c2' = c2 XOR majority(a1,b1,c1') (carry-out; ancilla is 0)
//
// Then a chain of three M2 measurements reads s1, s0 and the carry-out off
// the system (each M2's remainder output feeds the next), so every running
// system stays small and each gate is the only gate its system feeds (see
// docs/qsim-style.md and the one-gate-per-system rule). All annotations are
// TextBoxes.
//
// Run it from the repository root:
//
//	go run ./cmd/examples/gen-adder2
//
// then load saves/adder2.qsim in the app.
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

// adder2Op builds the 64×64 permutation for the whole 2-bit addition.
// The map is a bijection (the carry ancillas are XORed, not overwritten),
// so the matrix is a valid unitary.
func adder2Op() [][]complex64 {
	m := make([][]complex64, 64)
	for i := range m {
		m[i] = make([]complex64, 64)
	}
	for x := 0; x < 64; x++ {
		a1 := (x >> 5) & 1
		a0 := (x >> 4) & 1
		b1 := (x >> 3) & 1
		b0 := (x >> 2) & 1
		c1 := (x >> 1) & 1
		c2 := x & 1
		s0 := a0 ^ b0
		c1o := c1 ^ (a0 & b0)
		s1 := a1 ^ b1 ^ c1o
		c2o := c2 ^ ((a1 & b1) ^ (a1 & c1o) ^ (b1 & c1o))
		y := (a1 << 5) | (a0 << 4) | (s1 << 3) | (s0 << 2) | (c1o << 1) | c2o
		m[y][x] = 1
	}
	return m
}

func gate(x, y float32, op [][]complex64) *components.Gate {
	return components.NewGate(x, y, glob.GateRadius, config.GateColor, "ADD 2+2", op, 6)
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

// textBox creates a backgrounded annotation box (all canvas text must live
// in a TextBox — see docs/qsim-style.md §1).
func textBox(x, y float32, text string, size int32) *components.TextBox {
	tb := components.NewTextBox(x, y, float32(rl.MeasureText(text, size))+24, float32(size)+16, size)
	tb.Text = text
	return tb
}

// measureOut mirrors CollapseGate.emitOutcome for basis states: the measured
// value of the qubit plus the conditioned remainder state.
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
	rw := windows.NewRenderWindow(0, 0, 1600, 900)
	rw.Rename("2-bit quantum adder: 2 + 1 = 3")

	A, B := 2, 1 // 10 + 01 = 11

	// --- registers -------------------------------------------------------
	// State order (gate input order): a1 a0 b1 b0 c1 c2 (MSB first).
	// Lanes follow that order, MSB on top, 200px pitch.
	newID := attributes.GenerateQubitModifierID
	a1, a0 := newID(), newID()
	b1, b0 := newID(), newID()
	c1, c2 := newID(), newID()
	mods := []int32{a1, a0, b1, b0, c1, c2}

	var comps []components.Component
	laneOf := map[int32]float32{}
	topY := float32(-2400)
	for i, id := range mods {
		laneOf[id] = topY + float32(i)*200
	}

	// Sources.
	sysOf := map[int32]*components.QubitsSystem{}
	states := map[int32]*qubits.QubitStateManager{}
	srcX := float32(-2000)
	for _, id := range mods {
		val := 0
		name := ""
		switch id {
		case a1:
			val, name = (A>>1)&1, fmt.Sprintf("A1 = %d", (A>>1)&1)
		case a0:
			val, name = A&1, fmt.Sprintf("A0 = %d", A&1)
		case b1:
			val, name = (B>>1)&1, fmt.Sprintf("B1 = %d", (B>>1)&1)
		case b0:
			val, name = B&1, fmt.Sprintf("B0 = %d", B&1)
		case c1:
			val, name = 0, "C1 = 0"
		case c2:
			val, name = 0, "C2 = 0"
		}
		s := components.NewSourceGateWithID(srcX, laneOf[id], 90, config.GateColor, name, basisState(val), id)
		comps = append(comps, s)
		st := qubits.NewQubitStateManagerFrom(basisState(val), []int32{id})
		states[id] = st
		qs := system(s.OutHook.Center.X, s.OutHook.Center.Y, st)
		s.OutHook.Connect(qs)
		s.OutHook.ConnectInfo(qs)
		comps = append(comps, qs)
		sysOf[id] = qs
	}

	// Register captions.
	comps = append(comps, textBox(-2400, topY-120, fmt.Sprintf("carry chain c1 c2 |  A register (left)  B register -> sum (right)"), 18))

	// Title + explanation.
	comps = append(comps, textBox(-1900, -2750, "2-bit quantum adder: 2 + 1 = 3", 24))
	expl := textBox(-1900, -2630, "One 6-qubit gate computes the whole addition in the basis |A1 A0 B1 B0 C1 C2>: sum bits land in B (S0 = A0 XOR B0, S1 = A1 XOR B1 XOR C1'), carry chain C1' = A0 AND B0, C2' = majority(A1,B1,C1').", 14)
	expl.Width = 1100
	comps = append(comps, expl)

	// --- the gate --------------------------------------------------------
	// Mirror the engine's Gate.CalculateOutPut: merge inputs in hook order,
	// SwapColumn to the top, Multiply.
	op := adder2Op()
	g := gate(-700, laneOf[b0], op)
	for j, id := range mods {
		g.HookList[j].Connect(detOf(sysOf[id], id))
	}
	merged := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	for _, id := range mods {
		merged.Merge(states[id])
	}
	for i, id := range mods {
		merged.SwapColumn(int32(i), merged.FindID(id))
	}
	merged.Multiply(op, 6)
	qs := system(g.OutPutHook[0].Center.X, g.OutPutHook[0].Center.Y, merged)
	g.OutPutHook[0].Connect(qs)
	comps = append(comps, g, qs)
	for _, id := range merged.ModifierID {
		states[id] = merged
		sysOf[id] = qs
	}

	// Step header + gate caption (one distinct 64x64 type, annotated once).
	comps = append(comps, textBox(-700, -2650, "Step 1 - the ADD gate computes the whole 2-bit addition", 16))
	cap := textBox(-700, laneOf[c2]+320, "ADD gate (64x64, one type): |A1 A0 B1 B0 C1 C2> -> |A1 A0 S1 S0 C1' C2'> with S_i = A_i XOR B_i XOR C_i, C1' = A0 AND B0, C2' = majority(A1,B1,C1')", 14)
	cap.Width = 1200
	comps = append(comps, cap)

	// --- readout: measure s1, s0, carry via a chained M2 ---------------
	// Each M2's remainder output feeds the next, keeping the running
	// system small and giving every gate its own system.
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
		light := components.NewLight(x+600, laneOf[id], lightCol)
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
	comps = append(comps, textBox(400, -2650, "Step 2 - measure the sum bits and the carry-out (lights)", 16))
	measure(400, b1, ((A+B)>>1)&1) // s1
	measure(1200, b0, (A+B)&1)     // s0
	measure(2000, c2, (A+B)>>2)    // carry-out

	rw.PushComponent(comps...)

	// Frame the circuit.
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

	// --- verify the adder logic on all 16 inputs -------------------------
	ok := true
	for a := 0; a < 4; a++ {
		for b := 0; b < 4; b++ {
			sum, carry := adder2Bits(a, b)
			if sum != (a+b)&3 || carry != (a+b)>>2 {
				ok = false
				fmt.Printf("adder fail: %d + %d -> sum %d carry %d (want %d, %d)\n", a, b, sum, carry, (a+b)&3, (a+b)>>2)
			}
		}
	}
	if !ok {
		fmt.Println("ERROR: 2-bit adder truth table is wrong")
		os.Exit(1)
	}
	fmt.Println("adder check: 2 + 1 = 3; all 16 (A,B) pairs verified")

	// The serialized states must all be basis states.
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
	}

	// --- serialize -------------------------------------------------------
	data := rw.SaveState()
	outPath := filepath.Join("saves", "adder2.qsim")
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

// adder2Bits applies the 2-bit adder truth tables and returns sum and carry.
func adder2Bits(a, b int) (sum, carry int) {
	a1, a0 := (a>>1)&1, a&1
	b1, b0 := (b>>1)&1, b&1
	s0 := a0 ^ b0
	c1o := a0 & b0
	s1 := a1 ^ b1 ^ c1o
	c2o := (a1 & b1) ^ (a1 & c1o) ^ (b1 & c1o)
	return s1<<1 | s0, c2o
}
