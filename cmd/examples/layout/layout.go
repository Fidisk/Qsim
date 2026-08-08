// Package layout provides shared geometry helpers for the save generators.
//
// A QubitsSystem draws 2^size cells as a grid of 2^ceil(size/2) columns by
// 2^floor(size/2) rows, each cell 100px, centered on the system. A system
// connected to a component's output hook snaps its center onto that hook
// (300px right of the component body). Use these helpers so the grid of
// every produced system clears the next component: nothing may sit inside a
// system grid, or it gets hidden behind it.
package layout

// HookDist is the snapped gate-to-hook distance: input hooks sit at x-300,
// the output hook at x+300, and remainder hooks at x+300 on their gate.
const HookDist float32 = 300

const cell = float32(100)

// GridW returns the drawn width of a system's state grid for n qubits.
func GridW(n int) float32 {
	cols := 1 << ((n + 1) / 2)
	return float32(cols) * cell
}

// GridH returns the drawn height of a system's state grid for n qubits.
func GridH(n int) float32 {
	rows := 1 << (n / 2)
	return float32(rows) * cell
}

// GridBounds returns the (x, y, w, h) rectangle of the grid of a system
// centered at (cx, cy) with n qubits.
func GridBounds(cx, cy float32, n int) (float32, float32, float32, float32) {
	w, h := GridW(n), GridH(n)
	return cx - w/2, cy - h/2, w, h
}

// AfterOutput returns the X at which the next component can start so the
// state grid of a system with n qubits — centered on the output hook at
// gateX+HookDist — clears it by the given margin.
func AfterOutput(gateX float32, n int, margin float32) float32 {
	return gateX + HookDist + GridW(n)/2 + margin
}
