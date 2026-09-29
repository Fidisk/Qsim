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
					write(e.ToComplex64())
				}
			}
		case *components.SourceGate:
			c64src := make([]complex64, len(v.Amplitude))
			for i, a := range v.Amplitude {
				c64src[i] = a.ToComplex64()
			}
			write("SourceGate", v.ID, v.ModifierID, c64src)
		case *components.QubitsSystem:
			write("QubitsSystem", v.ID, v.Probability)
			if v.Origin != nil {
				// Amptitude is []symbolic.SymbolicValue: hash its complex64
				// values so the fingerprint stays stable across the migration.
				c64 := make([]complex64, len(v.Origin.Amptitude))
				for i, a := range v.Origin.Amptitude {
					c64[i] = a.ToComplex64()
				}
				write(v.Origin.ModifierID, c64)
			}
		case *components.CollapseGate:
			write("CollapseGate", v.ID, v.ForceMode, v.NormalSystem)
		case *components.M4Gate:
			write("M4Gate", v.ID, v.ForceMode, v.SwapInterval)
		case *components.ControlledGate:
			c64cg := make([][]complex64, len(v.Operation))
			for i, row := range v.Operation {
				c64cg[i] = make([]complex64, len(row))
				for j, e := range row {
					c64cg[i][j] = e.ToComplex64()
				}
			}
			write("ControlledGate", v.ID, c64cg)
		case *components.ControlledUGate:
			c64cu := make([][]complex64, len(v.Operation))
			for i, row := range v.Operation {
				c64cu[i] = make([]complex64, len(row))
				for j, e := range row {
					c64cu[i][j] = e.ToComplex64()
				}
			}
			write("ControlledUGate", v.ID, v.InputCount, v.Editable, v.Label, c64cu)
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
