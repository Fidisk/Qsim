// Package verify emits static verification bundles from .qsim saves so a
// circuit can be cross-checked against an external simulator (Qiskit).
//
// The package is a pure library: it never runs external programs. It reads a
// save, replays the unitary pipeline with the engine's own state primitives
// (Merge/SwapColumn/Multiply), and writes two files per circuit:
//
//	<name>.qasm        OpenQASM 2.0 rendering (standard gates; custom
//	                   unitaries and non-|0> preparations are comments
//	                   pointing at the sidecar)
//	<name>.verify.json sidecar: full ordered op list with matrices, the
//	                   expected final statevector (Qiskit order, qubit 0 =
//	                   LSB), and notes about anything stripped on export
//
// verify/qiskit_check.py (standalone, run manually) rebuilds the circuit
// from the sidecar, simulates it, and asserts fidelity against Expected.
// File layout is the interface: Go never invokes Python and Python never
// invokes Go.
//
// Supported canvas subset (v1): SourceGate and standalone 1-qubit inputs,
// unitary Gate matrices of any size (incl. editable universal gates),
// CopyGate, and classically-controlled gates whose control is unhooked
// (identity passthrough). Terminal M1/M2/M3 measurements are stripped (the
// pre-measurement state is compared) and recorded in Notes. Anything else on
// the quantum path (mid-circuit measurement outputs feeding gates,
// driven classical controls, M4, DecomposeGate, multi-qubit standalone
// inputs) is a hard error with a message naming the component.
//
// Bit ordering: Qsim stores ModifierID[0] as the most-significant bit.
// QASM qubit i maps to exactly one canvas modifier (Circuit.Modifiers[i]),
// and Qiskit index bit i is qubit i, so Expected lines up elementwise with
// Qiskit's statevector. For "unitary" ops, Op.Qubits is least-significant
// first (reversed gate-input order): UnitaryGate(matrix, qubits) treats the
// LAST listed qubit as the most significant matrix bit, so the gate input
// I0 (matrix MSB) goes last.
package verify

import "fmt"

// Op is one time-ordered operation. Gate is one of "x" (init |1> flip),
// "prep" (non-|0>/|1> initial state, carries State), "h", "x", "y", "z",
// "cx", "cy", "cz", or "unitary" (carries Matrix, Qubits least-significant
// first, i.e. reversed gate-input order). Custom counts the unitary index
// for QASM comments.
type Op struct {
	Gate   string         `json:"gate"`
	Qubits []int          `json:"qubits"`
	State  []complex128   `json:"state,omitempty"`
	Matrix [][]complex128 `json:"matrix,omitempty"`
	Label  string         `json:"label"`
	Custom int            `json:"custom,omitempty"`
}

// Circuit is the extracted verification model of one save.
type Circuit struct {
	Name      string          `json:"name"`
	NumQubits int             `json:"n_qubits"`
	Modifiers []int32         `json:"-"`
	Init      [][2]complex128 `json:"-"`
	Ops       []Op            `json:"ops"`
	Expected  []complex64     `json:"expected"`
	Notes     []string        `json:"notes"`
}

// errf builds an extraction error naming the offending component.
func errf(format string, args ...interface{}) error {
	return fmt.Errorf("verify: "+format, args...)
}
