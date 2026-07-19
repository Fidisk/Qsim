package config

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

const MaxZoom float32 = 10.0
const MinZoom float32 = 0.1

const QubitSystemStatePause bool = true

const SnapToGridInterval float32 = 100.0
const PhysicsEnabled bool = false

// Gate-computation demo visualization.

// QubitColors assigns each qubit a color by its modifier ID
// (modifier ID modulo the number of colors).
var QubitColors = []rl.Color{
	rl.SkyBlue,
	rl.Orange,
	rl.Lime,
	rl.Pink,
	rl.Yellow,
	rl.Purple,
	rl.Red,
	rl.Green,
}

const ComputeDotArriveDur = 0.8  // seconds for the input dots to fly into the gate
const ComputeMergeDur = 1.2      // source columns merging into one Dirac column
const ComputeReorderDur = 1.0    // gate qubits moving to the top of the column
const ComputeGateAppearDur = 0.8 // gate matrix fade-in
const ComputeIterBaseDur = 0.9   // seconds for the first iteration
const ComputeIterDecay = 0.55    // per-iteration speed-up factor
const ComputeIterFloorDur = 0.06 // fastest per-iteration duration
const ComputeIterCap = 64        // max visualized iterations before fast-forward
const ComputeCollapseDur = 1.2   // sum columns collapsing into the result column
const ComputeMaxRows = 8         // visible rows per column before "..." is used
