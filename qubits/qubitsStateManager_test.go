package qubits

import (
	"testing"

	"qsim/symbolic"
)

// Mirror of ControlledGate.updateOutput: bring the target qubit to logical
// position 0, apply the 1-qubit op, swap back.
func TestSwapMultiplySwapBack(t *testing.T) {
	// logical0 = |1>, logical1 = |0>  -> index 0b10 = 2
	qsm := NewQubitStateManagerFrom([]symbolic.SymbolicValue{symbolic.Zero(), symbolic.Zero(), symbolic.One(), symbolic.Zero()}, []int32{100, 200})

	// apply X to logical position 1 (modifier 200)
	pos := qsm.FindID(200)
	if pos != 1 {
		t.Fatalf("FindID = %d, want 1", pos)
	}
	qsm.SwapColumn(0, pos)
	qsm.Multiply([][]symbolic.SymbolicValue{{symbolic.Zero(), symbolic.One()}, {symbolic.One(), symbolic.Zero()}}, 1)
	qsm.SwapColumn(0, pos)

	// expect logical0 = |1>, logical1 = |1> -> index 0b11 = 3
	want := []symbolic.SymbolicValue{symbolic.Zero(), symbolic.Zero(), symbolic.Zero(), symbolic.One()}
	for i := range want {
		if qsm.Amptitude[i] != want[i] {
			t.Fatalf("amps = %v, want %v", qsm.Amptitude, want)
		}
	}
	// modifier order must be restored
	if qsm.ModifierID[0] != 100 || qsm.ModifierID[1] != 200 {
		t.Fatalf("modifiers = %v, want [100 200]", qsm.ModifierID)
	}
}

// Same op on logical position 0 (no effective swap): X on logical0 of |10>
// gives |00> -> index 0.
func TestSwapMultiplyPositionZero(t *testing.T) {
	qsm := NewQubitStateManagerFrom([]symbolic.SymbolicValue{symbolic.Zero(), symbolic.Zero(), symbolic.One(), symbolic.Zero()}, []int32{100, 200})
	pos := qsm.FindID(100)
	qsm.SwapColumn(0, pos)
	qsm.Multiply([][]symbolic.SymbolicValue{{symbolic.Zero(), symbolic.One()}, {symbolic.One(), symbolic.Zero()}}, 1)
	qsm.SwapColumn(0, pos)
	if !qsm.Amptitude[0].IsOne() {
		t.Fatalf("amps = %v, want [1 0 0 0]", qsm.Amptitude)
	}
}
