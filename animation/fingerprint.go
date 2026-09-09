package animation

import (
	"fmt"
	"hash/fnv"

	"qsim/components"
)

// lastCircuitHash remembers the circuit state the current steps were
// generated from; Sync regenerates when it changes.
var lastCircuitHash uint64

// hookLister is implemented by every component with hooks (gates,
// measurements, logic gates, ...).
type hookLister interface {
	GetHooks() []*components.Hook
}

// circuitHash hashes the computation-relevant state of a circuit: component
// types/IDs, gate matrices and labels, source amplitudes, qubit-system
// states, measurement settings, and every hook's wiring (targetIDs).
// Cosmetic movement (positions, drags, determinator drift) is intentionally
// excluded so rearranging the layout does not reset the animation.
func circuitHash(wComp []components.Component) uint64 {
	h := fnv.New64a()
	write := func(vals ...interface{}) {
		for _, v := range vals {
			fmt.Fprint(h, v)
			h.Write([]byte{0})
		}
	}
	for _, c := range wComp {
		switch v := c.(type) {
		case *components.Gate:
			write("Gate", v.ID, v.InputCount, v.IsMeasurementGate, v.Editable, v.Label)
			for _, row := range v.Operation {
				for _, e := range row {
					write(e)
				}
			}
		case *components.SourceGate:
			write("SourceGate", v.ID, v.ModifierID, v.Amplitude)
		case *components.QubitsSystem:
			write("QubitsSystem", v.ID, v.Probability)
			if v.Origin != nil {
				write(v.Origin.ModifierID, v.Origin.Amptitude)
			}
		case *components.CollapseGate:
			write("CollapseGate", v.ID, v.ForceMode, v.NormalSystem)
		case *components.M4Gate:
			write("M4Gate", v.ID, v.ForceMode, v.SwapInterval)
		case *components.ControlledGate:
			write("ControlledGate", v.ID, v.Operation)
		case *components.ControlledUGate:
			write("ControlledUGate", v.ID, v.InputCount, v.Editable, v.Label, v.Operation)
		case *components.CopyGate:
			write("CopyGate", v.ID, v.InHook.TargetID, v.OutHook.TargetID)
		case *components.LogicalBit:
			write("LogicalBit", v.ID, v.Value)
		case *components.LogicButton:
			write("LogicButton", v.ID)
		case *components.Light:
			write("Light", v.ID)
		default:
			write(c.GetID())
		}
		// Wiring: every hook and its target.
		if hl, ok := c.(hookLister); ok {
			for _, hk := range hl.GetHooks() {
				write("hook", hk.ID, hk.TargetID, hk.IsOutput, hk.AllowQubitSystem, hk.AllowLogicalBit)
			}
		}
	}
	return h.Sum64()
}

// Sync regenerates the animation steps whenever the circuit's
// computation-relevant state changed since the last call. Call it every
// frame with the circuit panel's components: editing a matrix, cycling a
// qubit, rewiring a hook or adding/removing a component restarts the
// animation from the beginning with the new circuit.
func Sync(wComp []components.Component) {
	h := circuitHash(wComp)
	if h == lastCircuitHash {
		return
	}
	lastCircuitHash = h
	Reset()
	Anim.Gates = GenerateSteps(wComp)
}
