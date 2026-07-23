package config

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// These are vars (not consts) so the dev bar on the web build can tweak them
// at runtime via window.qsimSetConfig(key, value).
var MaxZoom float32 = 10.0
var MinZoom float32 = 0.1

var QubitSystemStatePause bool = true

var SnapToGridInterval float32 = 100.0
var PhysicsEnabled bool = false

// Gate-computation demo visualization.

// QubitColors is the demo color palette. In the compute demo each input
// system gets one palette color shared by all its qubits (indexed by source
// order); qColor falls back to per-modifier coloring when the source is
// unknown.
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

var ComputeDotArriveDur = 1.0  // seconds for the input dots to fly into the gate
var ComputeMergeDur = 1.6      // source columns merging into one Dirac column
var ComputeReorderDur = 1.4    // gate qubits moving to the top of the column
var ComputeGateAppearDur = 1.0 // gate matrix fade-in
var ComputeIterBaseDur = 1.5   // seconds for the first iteration
var ComputeIterDecay = 0.7     // per-iteration speed-up factor
var ComputeIterFloorDur = 0.15 // fastest per-iteration duration
var ComputeIterCap int32 = 64   // max visualized iterations before fast-forward
var ComputeCollapseDur = 1.6    // sum columns collapsing into the result column
var ComputeMaxRows int32 = 8    // visible rows per column before "..." is used
