package verify

import (
	"math"
	"math/cmplx"
	"sort"
	"strings"

	"qsim/components"
	"qsim/qubits"
	"qsim/windows"
)

// Extract loads a .qsim save and builds the verification circuit: the
// ordered op list, the QASM qubit mapping, and the expected final
// statevector replayed with the engine's own state primitives.
//
// Only the supported subset (see package doc) extracts; anything else on
// the quantum path is an error naming the component.
func Extract(data, name string) (*Circuit, error) {
	x := &extractor{
		systems:   map[int32]*components.QubitsSystem{},
		dets:      map[int32]*components.QubitDeterminator{},
		detSys:    map[int32]int32{},
		hooks:     map[int32]*components.Hook{},
		hookOwner: map[int32]components.Component{},
		managers:  map[int32]*qubits.QubitStateManager{},
		hookDet:   map[int32]int32{},
		exclusive: map[int32][]consumer{},

		measParent:   map[int32]int32{},
		measChildren: map[int32][]int32{},
		isInit:       map[int32]bool{},
		bits:         map[int32]*components.LogicalBit{},
		bitDrivers:   map[int32][]bitDriver{},
		meas:         map[int32]*measInfo{},
		skipMeas:     map[int32]bool{},
		spare:        map[int32]bool{},
		controls:     map[int32]ctlInfo{},
		compiled:     map[int32]bool{},
		ver:          map[int32]*qubits.QubitStateManager{},
		uses:         map[*qubits.QubitStateManager][]int32{},
		circuit:      &Circuit{Name: name},
		qIndex:       map[int32]int{},
	}
	if err := x.load(data); err != nil {
		return nil, err
	}
	if err := x.producers(); err != nil {
		return nil, err
	}
	if err := x.consumers(); err != nil {
		return nil, err
	}
	if err := x.checks(); err != nil {
		return nil, err
	}
	if err := x.assignQubits(); err != nil {
		return nil, err
	}
	if err := x.replay(); err != nil {
		return nil, err
	}
	return x.circuit, x.assemble()
}

type extractor struct {
	comps []components.Component

	systems   map[int32]*components.QubitsSystem
	dets      map[int32]*components.QubitDeterminator
	detSys    map[int32]int32 // determinator ID -> parent system ID
	hooks     map[int32]*components.Hook
	hookOwner map[int32]components.Component
	// hookDet maps input-hook ID -> determinator ID from the determinator
	// side. It is the authoritative read for gate inputs: it agrees with
	// the hook side on well-formed loads, and any disagreement is
	// reported as conflicting links instead of silently mis-replaying.
	hookDet map[int32]int32

	managers  map[int32]*qubits.QubitStateManager // system ID -> replay state
	exclusive map[int32][]consumer                // system ID -> exclusive consumers

	measParent   map[int32]int32   // meas output sys -> measurement input sys
	measChildren map[int32][]int32 // collapse comp ID -> output sys IDs
	isInit       map[int32]bool    // system ID -> source/standalone input

	bits       map[int32]*components.LogicalBit
	bitDrivers map[int32][]bitDriver
	meas       map[int32]*measInfo // collapse comp ID -> measurement record
	skipMeas   map[int32]bool      // measurement comp IDs with unconnected inputs
	spare      map[int32]bool      // comp IDs of unconnected spare gates
	controls   map[int32]ctlInfo   // controlled-gate comp ID -> resolved control
	compiled   map[int32]bool      // collapse IDs whose bit drove a control
	measComps  []measComp          // measurements with hooked inputs, in order
	ver        map[int32]*qubits.QubitStateManager
	uses       map[*qubits.QubitStateManager][]int32

	circuit *Circuit
	qIndex  map[int32]int // modifier ID -> QASM index
	phased  bool          // some gate matched up to a global phase
}

// bitDriver is one static driver of a LogicalBit: a measurement C output,
// a button constant, or a logic gate (unknown value).
type bitDriver struct {
	kind  string // "meas", "const", "logic"
	mid   int32  // collapse comp ID for meas
	value int32  // button value for const
}

// measInfo records an M2 collapse whose C-bit may drive controls.
type measInfo struct {
	sys   int32
	mod   int32
	bit   int32
	label string
}

// measComp pairs a measurement component with its input for end notes.
type measComp struct {
	id    int32
	label string
	sys   int32 // input system
	mod   int32 // measured modifier
	m4    bool  // oscillating: never exact
}

// ctlInfo is a resolved classical control: constant or compiled quantum
// control on the measured qubit (deferred measurement).
type ctlInfo struct {
	kind string // "const", "qctl"
	val  int32  // constant value
	mid  int32  // collapse comp ID for qctl
}

// load parses the save and indexes components, systems, determinators and
// hooks by ID. Hook TargetIDs were remapped to live objects by LoadState.
func (x *extractor) load(data string) error {
	var rw *windows.RenderWindow
	for _, w := range windows.LoadState(data) {
		if r, ok := w.(*windows.RenderWindow); ok {
			rw = r
			break
		}
	}
	if rw == nil {
		return errf("save contains no circuit window")
	}
	x.comps = rw.WComp
	hook := func(h *components.Hook, owner components.Component) {
		if h == nil {
			return
		}
		x.hooks[h.ID] = h
		x.hookOwner[h.ID] = owner
	}
	for _, c := range x.comps {
		switch t := c.(type) {
		case *components.Gate:
			for _, h := range t.HookList {
				hook(h, c)
			}
		case *components.SourceGate:
			hook(t.OutHook, c)
		case *components.CopyGate:
			hook(t.InHook, c)
			hook(t.OutHook, c)
		case *components.CollapseGate:
			for _, h := range t.HookList {
				hook(h, c)
			}
		case *components.ControlledGate:
			for _, h := range t.GetHooks() {
				hook(h, c)
			}
		case *components.ControlledUGate:
			for _, h := range t.GetHooks() {
				hook(h, c)
			}
		case *components.M4Gate:
			for _, h := range t.HookList {
				hook(h, c)
			}
		case *components.DecomposeGate:
			for _, h := range t.HookList {
				hook(h, c)
			}
		case *components.QubitsSystem:
			if t.Origin == nil {
				return errf("qubit system (id %d) has no state", t.ID)
			}
			x.systems[t.ID] = t
			for _, d := range t.QubitDeterminatorList {
				x.dets[d.ID] = d
				x.detSys[d.ID] = t.ID
				if d.HookID != 0 {
					x.hookDet[d.HookID] = d.ID
				}
			}
		case *components.LogicalBit:
			x.bits[t.ID] = t
		}
	}
	if len(x.systems) == 0 {
		return errf("save contains no qubit systems")
	}
	return nil
}

// producerHook resolves the output hook feeding a system from the union of
// surviving links: exactly one of the system side (HookID) and the hook
// side (TargetID) survives LoadState's order-dependent remap, so either may
// be zero. Both nonzero but disagreeing means genuine corruption.
func (x *extractor) producerHook(sys *components.QubitsSystem) (*components.Hook, error) {
	var viaSys *components.Hook
	if sys.HookID != 0 {
		h, ok := x.hooks[sys.HookID]
		if !ok {
			return nil, errf("qubit system (id %d) links to unknown hook %d", sys.ID, sys.HookID)
		}
		viaSys = h
	}
	var viaHook *components.Hook
	for _, h := range x.sortedHooks() {
		if h.TargetID == sys.ID {
			viaHook = h
			break
		}
	}
	if viaSys != nil && viaHook != nil && viaSys.ID != viaHook.ID {
		return nil, errf("qubit system (id %d) has conflicting producer links", sys.ID)
	}
	if viaSys != nil {
		return viaSys, nil
	}
	return viaHook, nil
}

func (x *extractor) sortedHooks() []*components.Hook {
	hs := make([]*components.Hook, 0, len(x.hooks))
	for _, h := range x.hooks {
		hs = append(hs, h)
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].ID < hs[j].ID })
	return hs
}

func (x *extractor) sortedSystems() []*components.QubitsSystem {
	ss := make([]*components.QubitsSystem, 0, len(x.systems))
	for _, s := range x.systems {
		ss = append(ss, s)
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].ID < ss[j].ID })
	return ss
}

// producers classifies how each hooked system gets its state and seeds the
// replay managers for source-fed and standalone inputs.
func (x *extractor) producers() error {
	for _, sys := range x.sortedSystems() {
		h, err := x.producerHook(sys)
		if err != nil {
			return err
		}
		if h == nil {
			if err := x.seedStandalone(sys); err != nil {
				return err
			}
			continue
		}
		owner, ok := x.hookOwner[h.ID]
		if !ok {
			return errf("qubit system (id %d) links to ownerless hook", sys.ID)
		}
		switch t := owner.(type) {
		case *components.SourceGate:
			if h != t.OutHook {
				return errf("qubit system (id %d) links to non-output source hook", sys.ID)
			}
			if err := x.seedSource(sys, t); err != nil {
				return err
			}
		case *components.Gate:
			if t.IsMeasurementGate {
				x.measChildren[t.ID] = append(x.measChildren[t.ID], sys.ID)
			} else if len(t.OutPutHook) == 0 || h != t.OutPutHook[0] {
				return errf("qubit system (id %d) links to non-output gate hook", sys.ID)
			}
		case *components.CopyGate:
			if h != t.OutHook {
				return errf("qubit system (id %d) links to non-output copy hook", sys.ID)
			}
		case *components.CollapseGate:
			x.measChildren[t.ID] = append(x.measChildren[t.ID], sys.ID)
		case *components.M4Gate:
			x.measChildren[t.ID] = append(x.measChildren[t.ID], sys.ID)
		case *components.ControlledGate, *components.ControlledUGate:
			// Handled as a replay job; control must be unhooked (checked there).
		default:
			return errf("qubit system (id %d) is produced by unsupported %T", sys.ID, owner)
		}
	}
	return nil
}

// seedStandalone seeds the replay from a detached input system (e.g. a
// clicked |0>/|1>/|+> state). Only single qubits are supported: larger
// standalone states cannot be expressed as QASM initializations.
func (x *extractor) seedStandalone(sys *components.QubitsSystem) error {
	o := sys.Origin
	if o.Size != 1 || len(o.Amptitude) != 2 {
		return errf("standalone %d-qubit system (id %d) is not supported; use 1-qubit inputs", o.Size, sys.ID)
	}
	amps := append([]complex64{}, o.Amptitude...)
	qubits.Normalize(amps)
	if amps[0] == 0 && amps[1] == 0 {
		return errf("standalone input system (id %d) is a zero vector", sys.ID)
	}
	x.managers[sys.ID] = qubits.NewQubitStateManagerFrom(amps, append([]int32{}, o.ModifierID...))
	x.isInit[sys.ID] = true
	return nil
}

// seedSource seeds the replay from a SourceGate, mirroring SourceGate.Update
// (normalize, then assert the configured amplitudes).
func (x *extractor) seedSource(sys *components.QubitsSystem, sg *components.SourceGate) error {
	if len(sg.Amplitude) != 2 {
		return errf("source %q has %d amplitudes, want 2", sg.Label, len(sg.Amplitude))
	}
	amps := append([]complex64{}, sg.Amplitude...)
	qubits.Normalize(amps)
	if amps[0] == 0 && amps[1] == 0 {
		return errf("source %q is a zero vector", sg.Label)
	}
	x.managers[sys.ID] = qubits.NewQubitStateManagerFrom(amps, []int32{sg.ModifierID})
	x.isInit[sys.ID] = true
	return nil
}

// inputSys resolves a gate input hook to its (system, modifier) via the
// determinator-side link map, which always survives LoadState. Both the
// consumer scan and the replay use it.
func (x *extractor) inputSys(h *components.Hook, what string) (int32, int32, error) {
	if h == nil {
		return 0, 0, errf("%s is unconnected", what)
	}
	detID, ok := x.hookDet[h.ID]
	if !ok {
		return 0, 0, errf("%s is unconnected", what)
	}
	d := x.dets[detID]
	sys, ok := x.detSys[d.ID]
	if !ok {
		return 0, 0, errf("%s links to an orphaned determinator", what)
	}
	if h.TargetID != 0 && h.TargetID != d.ID {
		return 0, 0, errf("%s has conflicting input links", what)
	}
	return sys, d.ModifierID, nil
}

// isSpare reports a gate with no quantum connections at all: a spare canvas
// part, not part of the circuit.
func (x *extractor) isSpare(c components.Component) bool {
	var hs []*components.Hook
	switch t := c.(type) {
	case *components.Gate:
		hs = t.HookList
	case *components.CopyGate:
		hs = []*components.Hook{t.InHook, t.OutHook}
	case *components.ControlledGate:
		hs = []*components.Hook{t.InQubit, t.OutHook}
	case *components.ControlledUGate:
		hs = append(append([]*components.Hook{}, t.QubitHooks...), t.OutHook)
	}
	for _, h := range hs {
		if x.hookLive(h) {
			return false
		}
	}
	return true
}

// consumers records every quantum consumer of every system and registers
// classical bit drivers. Measurements never exclusively consume. Copy
// inputs only mirror state and may share their system with a gate.
func (x *extractor) consumers() error {
	inputSys := x.inputSys
	for _, c := range x.comps {
		switch t := c.(type) {
		case *components.Gate:
			if !t.IsMeasurementGate && x.isSpare(c) {
				x.spare[t.ID] = true
			}
		case *components.CopyGate:
			if x.isSpare(c) {
				x.spare[t.ID] = true
			}
		case *components.ControlledGate:
			if x.isSpare(c) {
				x.spare[t.ID] = true
			}
		case *components.ControlledUGate:
			if x.isSpare(c) {
				x.spare[t.ID] = true
			}
		}
	}
	for _, c := range x.comps {
		switch t := c.(type) {
		case *components.Gate:
			if x.spare[t.ID] {
				x.circuit.Notes = append(x.circuit.Notes, t.Label+": unconnected spare, skipped")
				continue
			}
			if t.IsMeasurementGate {
				h := hook0(t.HookList)
				if h == nil || !x.hookLive(h) {
					x.skipMeas[t.ID] = true
					x.circuit.Notes = append(x.circuit.Notes, "M1 "+t.Label+": input unconnected, skipped")
					continue
				}
				sys, mod, err := inputSys(h, "M1 "+t.Label+" input")
				if err != nil {
					return err
				}
				for _, child := range x.measChildren[t.ID] {
					x.measParent[child] = sys
				}
				x.measComps = append(x.measComps, measComp{t.ID, t.Label, sys, mod, false})
				continue
			}
			if int32(len(t.HookList)) < t.InputCount {
				return errf("gate %q is missing input hooks", t.Label)
			}
			for i := int32(0); i < t.InputCount; i++ {
				sys, _, err := inputSys(t.HookList[i], "gate "+t.Label+" input "+hookName(t.HookList[i], i))
				if err != nil {
					return err
				}
				x.exclusive[sys] = append(x.exclusive[sys], consumer{"gate " + t.Label, t.ID})
			}
		case *components.CopyGate:
			if x.spare[t.ID] {
				x.circuit.Notes = append(x.circuit.Notes, t.Label+": unconnected spare, skipped")
				continue
			}
			if t.InHook != nil && (t.InHook.IsHooked || t.InHook.TargetID != 0) {
				if _, ok := x.systems[t.InHook.TargetID]; !ok {
					if _, isDet := x.dets[t.InHook.TargetID]; isDet {
						return errf("copy gate %q input links to a determinator, want a system", t.Label)
					}
					return errf("copy gate %q input links to a non-system target", t.Label)
				}
			}
			// Read-only mirror: consumes nothing, may share its system.
			// Without an input it produces nothing (checked at replay).
		case *components.CollapseGate:
			if len(t.HookList) == 0 || !x.hookLive(t.HookList[0]) {
				x.skipMeas[t.ID] = true
				x.circuit.Notes = append(x.circuit.Notes, t.Label+": input unconnected, skipped")
				continue
			}
			sys, mod, err := inputSys(t.HookList[0], t.Label+" input")
			if err != nil {
				return err
			}
			for _, child := range x.measChildren[t.ID] {
				x.measParent[child] = sys
			}
			if len(t.OutPutHook) > 0 {
				if ch := t.OutPutHook[0]; ch != nil {
					if bit, ok := x.bits[ch.TargetID]; ok && !t.NormalSystem && ch.TargetID != 0 {
						x.meas[t.ID] = &measInfo{sys, mod, bit.ID, t.Label}
						x.bitDrivers[bit.ID] = append(x.bitDrivers[bit.ID], bitDriver{"meas", t.ID, 0})
					}
				}
			}
			x.measComps = append(x.measComps, measComp{t.ID, t.Label, sys, mod, false})
		case *components.ControlledGate:
			if x.spare[t.ID] {
				x.circuit.Notes = append(x.circuit.Notes, t.Label+": unconnected spare, skipped")
				continue
			}
			sys, _, err := inputSys(t.InQubit, "controlled gate "+t.Label+" qubit input")
			if err != nil {
				return err
			}
			x.exclusive[sys] = append(x.exclusive[sys], consumer{"controlled gate " + t.Label, t.ID})
		case *components.ControlledUGate:
			if x.spare[t.ID] {
				x.circuit.Notes = append(x.circuit.Notes, t.Label+": unconnected spare, skipped")
				continue
			}
			for i, h := range t.QubitHooks {
				sys, _, err := inputSys(h, "controlled-U "+t.Label+" input I"+itoa(i))
				if err != nil {
					return err
				}
				x.exclusive[sys] = append(x.exclusive[sys], consumer{"controlled-U " + t.Label, t.ID})
			}
		case *components.LogicButton:
			for _, bid := range []int32{hookTargetID(t.OutHook), t.OutputID} {
				if _, ok := x.bits[bid]; ok && bid != 0 {
					x.bitDrivers[bid] = append(x.bitDrivers[bid], bitDriver{"const", 0, t.Value})
				}
			}
		case *components.LogicGate:
			for _, bid := range []int32{hookTargetID(t.OutHook), t.OutputID} {
				if _, ok := x.bits[bid]; ok && bid != 0 {
					x.bitDrivers[bid] = append(x.bitDrivers[bid], bitDriver{"logic", 0, 0})
				}
			}
		case *components.M4Gate:
			// Read-only like a collapse (input continues, outputs strip
			// if terminal), but the C-bit oscillates: usable by nothing.
			if len(t.HookList) == 0 || !x.hookLive(t.HookList[0]) {
				x.skipMeas[t.ID] = true
				x.circuit.Notes = append(x.circuit.Notes, t.Label+": input unconnected, skipped")
				continue
			}
			sys, mod, err := inputSys(t.HookList[0], t.Label+" input")
			if err != nil {
				return err
			}
			for _, child := range x.measChildren[t.ID] {
				x.measParent[child] = sys
			}
			if len(t.HookList) > 1 {
				if ch := t.HookList[1]; ch != nil && ch.TargetID != 0 {
					if _, ok := x.bits[ch.TargetID]; ok {
						x.bitDrivers[ch.TargetID] = append(x.bitDrivers[ch.TargetID], bitDriver{"m4", 0, 0})
					}
				}
			}
			x.measComps = append(x.measComps, measComp{t.ID, t.Label, sys, mod, true})
		case *components.DecomposeGate:
			for _, h := range t.HookList {
				if x.hookLive(h) {
					return errf("decompose gate %q is connected but has no operation", t.Label)
				}
			}
		}
	}
	return x.resolveControls()
}

// hookTargetID returns h.TargetID or 0 for a nil hook.
func hookTargetID(h *components.Hook) int32 {
	if h == nil {
		return 0
	}
	return h.TargetID
}

// compID returns the component ID for gate-like components.
func (x *extractor) compID(c components.Component) int32 {
	switch t := c.(type) {
	case *components.Gate:
		return t.ID
	case *components.CopyGate:
		return t.ID
	case *components.ControlledGate:
		return t.ID
	case *components.ControlledUGate:
		return t.ID
	}
	return 0
}

// resolveControls decides every hooked classical control: unhooked or
// non-bit targets read 0 (mirroring inputBit/controlValue), undriven bits
// read their stored value, button-driven bits read the button value, and
// measurement-driven bits compile to quantum controls.
func (x *extractor) resolveControls() error {
	for _, c := range x.comps {
		var h *components.Hook
		var what string
		var gid int32
		switch t := c.(type) {
		case *components.ControlledGate:
			if x.spare[t.ID] {
				continue
			}
			h, what, gid = t.InControl, "controlled gate "+t.Label, t.ID
		case *components.ControlledUGate:
			if x.spare[t.ID] {
				continue
			}
			h, what, gid = t.InControl, "controlled-U "+t.Label, t.ID
		default:
			continue
		}
		if h == nil || !h.IsHooked {
			x.controls[gid] = ctlInfo{"const", 0, 0}
			continue
		}
		bit, ok := x.bits[h.TargetID]
		if !ok {
			x.controls[gid] = ctlInfo{"const", 0, 0}
			x.circuit.Notes = append(x.circuit.Notes, what+": control targets a non-bit, reads 0")
			continue
		}
		drvs := x.bitDrivers[bit.ID]
		if len(drvs) == 0 {
			x.controls[gid] = ctlInfo{"const", bit.Value, 0}
			x.circuit.Notes = append(x.circuit.Notes, what+": undriven control bit reads stored value "+itoa(int(bit.Value)))
			continue
		}
		kinds := map[string]bool{}
		vals := map[int32]bool{}
		for _, d := range drvs {
			kinds[d.kind] = true
			if d.kind == "const" {
				vals[d.value] = true
			}
		}
		switch {
		case len(kinds) == 1 && kinds["const"] && len(vals) == 1:
			for v := range vals {
				x.controls[gid] = ctlInfo{"const", v, 0}
			}
		case len(kinds) == 1 && kinds["meas"] && len(drvs) == 1:
			mid := drvs[0].mid
			if _, err := x.controlCarrier(mid, what, gid); err != nil {
				return err
			}
			x.controls[gid] = ctlInfo{"qctl", 0, mid}
			x.compiled[mid] = true
		case len(kinds) == 1 && kinds["m4"]:
			return errf("%s control bit is driven by an oscillating (M4) measurement and has no static value", what)
		default:
			return errf("%s control bit value is not statically known (only measurement-compiled, button-driven, or stored bits verify)", what)
		}
	}
	return nil
}

// controlCarrier returns the produced system holding a measurement's qubit,
// walked transparent through measurement outputs. It must be free of
// unitary consumers (other than the controlled gate itself) so its stored
// state is time-independent.
func (x *extractor) controlCarrier(mid int32, what string, self int32) (int32, error) {
	m, ok := x.meas[mid]
	if !ok {
		return 0, errf("%s references an unknown measurement", what)
	}
	sys := m.sys
	seen := map[int32]bool{}
	for {
		parent, ok := x.measParent[sys]
		if !ok {
			break
		}
		if seen[sys] {
			return 0, errf("%s: measurement chain cycle", what)
		}
		seen[sys] = true
		sys = parent
	}
	for _, c := range x.exclusive[sys] {
		if c.gate != self {
			return 0, errf("%s: control qubit carrier system (id %d) also feeds %s; deferred measurement needs it measurement-only", what, sys, c.desc)
		}
	}
	return sys, nil
}

// hookLive reports whether a hook carries a quantum link on any surviving
// side (hook flags/target or the determinator-side map).
func (x *extractor) hookLive(h *components.Hook) bool {
	if h == nil {
		return false
	}
	if h.IsHooked || h.TargetID != 0 {
		return true
	}
	_, ok := x.hookDet[h.ID]
	return ok
}

func hook0(hs []*components.Hook) *components.Hook {
	if len(hs) == 0 {
		return nil
	}
	return hs[0]
}

func hookName(h *components.Hook, i int32) string {
	if h != nil && h.Label != "" {
		return h.Label
	}
	return "I" + itoa(int(i))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [8]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}

// checks enforces single-exclusive-consumer wiring. Measurement outputs
// may feed downstream transparently (deferred measurement); fan-out of one
// state object is caught during replay by use counting.
func (x *extractor) checks() error {
	var ids []int32
	for sys := range x.exclusive {
		ids = append(ids, sys)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, sys := range ids {
		who := x.exclusive[sys]
		gates := map[int32]string{}
		for _, c := range who {
			gates[c.gate] = c.desc
		}
		if len(gates) > 1 {
			var names []string
			for _, n := range gates {
				names = append(names, n)
			}
			sort.Strings(names)
			return errf("qubit system (id %d) feeds %d gates (%s); one system feeds one gate", sys, len(gates), strings.Join(names, ", "))
		}
	}
	return nil
}

// distinctGates counts the distinct gates exclusively consuming a system.
func (x *extractor) distinctGates(sys int32) int {
	seen := map[int32]bool{}
	for _, c := range x.exclusive[sys] {
		seen[c.gate] = true
	}
	return len(seen)
}

// assignQubits maps every source modifier to a QASM index in save order and
// emits the initialization ops.
func (x *extractor) assignQubits() error {
	var sources []*components.QubitsSystem
	for _, c := range x.comps {
		if sys, ok := c.(*components.QubitsSystem); ok && x.isInit[sys.ID] {
			sources = append(sources, sys)
		}
	}
	if len(sources) == 0 {
		return errf("no source or standalone input systems found")
	}
	for _, sys := range sources {
		m := x.managers[sys.ID]
		// Init managers are always single-qubit (seedStandalone/seedSource
		// enforce it), so ModifierID has one entry and Amptitude two.
		if len(m.ModifierID) != 1 || len(m.Amptitude) != 2 {
			return errf("input system (id %d) is not a single qubit", sys.ID)
		}
		mod := m.ModifierID[0]
		if _, dup := x.qIndex[mod]; dup {
			return errf("modifier %d is initialized twice", mod)
		}
		x.qIndex[mod] = len(x.circuit.Modifiers)
		x.circuit.Modifiers = append(x.circuit.Modifiers, mod)
		x.circuit.Init = append(x.circuit.Init, [2]complex128{complex128(m.Amptitude[0]), complex128(m.Amptitude[1])})
		x.ver[mod] = m
	}
	x.circuit.NumQubits = len(x.circuit.Modifiers)
	for qi, amps := range x.circuit.Init {
		a, b := amps[0], amps[1]
		switch {
		case close1(a, 1) && close1(b, 0):
			// |0>: Qiskit default, no op.
		case close1(a, 0) && (close1(b, 1) || close1(b, -1)):
			x.circuit.Ops = append(x.circuit.Ops, Op{Gate: "x", Qubits: []int{qi}, Label: "init|1>"})
		default:
			x.circuit.Ops = append(x.circuit.Ops, Op{Gate: "prep", Qubits: []int{qi}, State: []complex128{a, b}, Label: "init"})
		}
	}
	return nil
}

func close1(v complex128, want float64) bool {
	return cmplx.Abs(v-complex(want, 0)) < 1e-6
}

// consumer is one exclusive use of a system by a gate. A system feeding
// several inputs of the same gate counts once, mirroring the engine.
type consumer struct {
	desc string
	gate int32
}

type jobKind int

const (
	jobUnitary jobKind = iota
	jobCopy
	jobControlled
)

type jobInput struct {
	sys int32
	mod int32
}

type job struct {
	id      int32
	desc    string
	kind    jobKind
	in      []jobInput
	out     int32
	op      [][]complex64
	n       int32
	label   string
	ctl     ctlInfo // resolved control for jobControlled
	carrier int32   // produced carrier system for qctl, else 0
	waits   []int32 // chain predecessors (same-carrier compiled controls)
	ckind   int32   // ControlledGate kind, or -1 for ControlledUGate/unitary
}

// resolveMgr returns the manager holding sys's qubits, transparent through
// measurement outputs (deferred measurement: remainders/branches see the
// uncollapsed input state). ok=false means not replayed yet.
func (x *extractor) resolveMgr(sys int32) (mgr *qubits.QubitStateManager, ok bool, err error) {
	seen := map[int32]bool{}
	for {
		parent, isMeas := x.measParent[sys]
		if !isMeas {
			break
		}
		if seen[sys] {
			return nil, false, errf("measurement chain cycle at system %d", sys)
		}
		seen[sys] = true
		sys = parent
	}
	mgr, ok = x.managers[sys]
	return mgr, ok, nil
}

// use records one gate's use of a state object; the same object feeding two
// different gates is fan-out of one state (e.g. both M1 branches consumed)
// and is not verifiable.
func (x *extractor) use(mgr *qubits.QubitStateManager, gid int32, what string) error {
	users := x.uses[mgr]
	for _, u := range users {
		if u == gid {
			return nil
		}
	}
	if len(users) > 0 {
		return errf("%s: one state feeds two gates (fan-out of a collapsed or remainder state is not verifiable)", what)
	}
	x.uses[mgr] = append(users, gid)
	return nil
}

// controlledMatrix builds the (1+n)-qubit controlled version of mat with
// the control as MSB: diag(I, U).
func controlledMatrix(mat [][]complex64) [][]complex64 {
	n := len(mat)
	out := make([][]complex64, 2*n)
	for i := range out {
		out[i] = make([]complex64, 2*n)
	}
	for i := 0; i < n; i++ {
		out[i][i] = 1
		for j := 0; j < n; j++ {
			out[n+i][n+j] = mat[i][j]
		}
	}
	return out
}

// ctrl1Q returns the single-qubit operation for a ControlledGate kind
// (0/1/2 = X/Y/Z).
func ctrl1Q(kind int32) ([][]complex64, string) {
	i := complex64(complex(0, 1))
	switch kind {
	case components.CtrlY:
		return [][]complex64{{0, -i}, {i, 0}}, "y"
	case components.CtrlZ:
		return [][]complex64{{1, 0}, {0, -1}}, "z"
	default:
		return [][]complex64{{0, 1}, {1, 0}}, "x"
	}
}

// ctrl1QName names the 1q operation of a ControlledGate kind.
func ctrl1QName(kind int32) string {
	_, name := ctrl1Q(kind)
	return name
}

// replay processes producers in dependency order, mirroring
// Gate.CalculateOutPut (merge in hook order with dedup, swap leading,
// multiply), and emits the op list.
func (x *extractor) replay() error {
	var jobs []*job
	outputOf := func(h *components.Hook, what string) (int32, error) {
		if h != nil && h.TargetID != 0 {
			if _, ok := x.systems[h.TargetID]; !ok {
				return 0, errf("%s output links to a non-system target", what)
			}
			return h.TargetID, nil
		}
		// Hook side dropped by the loader: find the system claiming this
		// hook as its producer.
		if h != nil {
			for _, sys := range x.sortedSystems() {
				ph, err := x.producerHook(sys)
				if err != nil {
					return 0, err
				}
				if ph != nil && ph.ID == h.ID {
					return sys.ID, nil
				}
			}
		}
		return 0, errf("%s output is unconnected", what)
	}
	gateInputs := func(hs []*components.Hook, n int32, what string) ([]jobInput, error) {
		if int32(len(hs)) < n {
			return nil, errf("%s is missing input hooks", what)
		}
		var in []jobInput
		for i := int32(0); i < n; i++ {
			sys, mod, err := x.inputSys(hs[i], what+" input "+hookName(hs[i], i))
			if err != nil {
				return nil, err
			}
			in = append(in, jobInput{sys: sys, mod: mod})
		}
		return in, nil
	}
	for _, c := range x.comps {
		switch t := c.(type) {
		case *components.Gate:
			if t.IsMeasurementGate || x.spare[t.ID] {
				continue
			}
			in, err := gateInputs(t.HookList, t.InputCount, "gate "+t.Label)
			if err != nil {
				return err
			}
			out, err := outputOf(hookOut(t.OutPutHook), "gate "+t.Label)
			if err != nil {
				return err
			}
			jobs = append(jobs, &job{id: t.ID, desc: "gate " + t.Label, kind: jobUnitary, in: in, out: out, op: t.Operation, n: t.InputCount, label: t.Label})
		case *components.CopyGate:
			if x.spare[t.ID] {
				continue
			}
			// Read-only mirror: consumes nothing, may share its system.
			// Without an input it produces nothing (mirroring the engine).
			in := []jobInput{}
			if t.InHook != nil && (t.InHook.IsHooked || t.InHook.TargetID != 0) {
				sys, ok := x.systems[t.InHook.TargetID]
				if !ok {
					return errf("copy gate %q input links to a non-system target", t.Label)
				}
				in = []jobInput{{sys: sys.ID}}
			}
			out, err := outputOf(t.OutHook, "copy gate "+t.Label)
			if err != nil {
				return err
			}
			jobs = append(jobs, &job{id: t.ID, desc: "copy " + t.Label, kind: jobCopy, in: in, out: out, label: t.Label})
		case *components.ControlledGate:
			if x.spare[t.ID] {
				continue
			}
			in, err := gateInputs([]*components.Hook{t.InQubit}, 1, "controlled gate "+t.Label)
			if err != nil {
				return err
			}
			out, err := outputOf(t.OutHook, "controlled gate "+t.Label)
			if err != nil {
				return err
			}
			ctl, ok := x.controls[t.ID]
			if !ok {
				return errf("controlled gate %q has an unresolved control", t.Label)
			}
			jobs = append(jobs, &job{id: t.ID, desc: "controlled gate " + t.Label, kind: jobControlled, in: in, out: out, op: t.Operation, n: 1, label: t.Label, ctl: ctl, ckind: t.Kind})
		case *components.ControlledUGate:
			if x.spare[t.ID] {
				continue
			}
			in, err := gateInputs(t.QubitHooks, t.InputCount, "controlled-U "+t.Label)
			if err != nil {
				return err
			}
			out, err := outputOf(t.OutHook, "controlled-U "+t.Label)
			if err != nil {
				return err
			}
			ctl, ok := x.controls[t.ID]
			if !ok {
				return errf("controlled-U %q has an unresolved control", t.Label)
			}
			jobs = append(jobs, &job{id: t.ID, desc: "controlled-U " + t.Label, kind: jobControlled, in: in, out: out, op: t.Operation, n: t.InputCount, label: t.Label, ctl: ctl, ckind: -1})
		}
	}
	// Same-carrier compiled controls form chains in ID order (sequential
	// quantum controls on one qubit commute, so any order verifies; the
	// chain only serializes replay).
	carriers := map[int32][]*job{}
	for _, j := range jobs {
		if j.kind == jobControlled && j.ctl.kind == "qctl" {
			if sys, err := x.controlCarrier(j.ctl.mid, j.desc, j.id); err != nil {
				return err
			} else {
				j.carrier = sys
				carriers[sys] = append(carriers[sys], j)
			}
		}
	}
	for _, chain := range carriers {
		sort.Slice(chain, func(i, k int) bool { return chain[i].id < chain[k].id })
		for i := 1; i < len(chain); i++ {
			chain[i].waits = append(chain[i].waits, chain[i-1].id)
		}
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].id < jobs[j].id })
	ready := func(j *job, doneIDs map[int32]bool) bool {
		for _, w := range j.waits {
			if !doneIDs[w] {
				return false
			}
		}
		for _, in := range j.in {
			if _, ok, _ := x.resolveMgr(in.sys); !ok {
				return false
			}
		}
		if j.kind == jobControlled && j.ctl.kind == "qctl" {
			if _, ok := x.managers[j.carrier]; !ok {
				return false
			}
		}
		return true
	}
	done := map[*job]bool{}
	doneIDs := map[int32]bool{}
	for len(done) < len(jobs) {
		progress := false
		for _, j := range jobs {
			if done[j] || !ready(j, doneIDs) {
				continue
			}
			if err := x.run(j); err != nil {
				return err
			}
			done[j] = true
			doneIDs[j.id] = true
			progress = true
		}
		if !progress {
			return errf("circuit has a dependency cycle near %s", jobsPending(jobs, done))
		}
	}
	x.measureNotes()
	return nil
}

// measureNotes records one note per measurement: compiled controls,
// transparent downstream remainders, or stripped terminal readouts.
// Deterministic outcomes verify exactly; random ones only at protocol level.
func (x *extractor) measureNotes() {
	for _, mc := range x.measComps {
		det := ""
		if !mc.m4 {
			if p0, ok := x.outcomeProb(mc.sys, mc.mod); ok && (p0 > 1-1e-9 || p0 < 1e-9) {
				k := int32(0)
				if p0 < 1e-9 {
					k = 1
				}
				det = "deterministic outcome " + itoa(int(k)) + ", exact; "
			}
		}
		var base string
		switch {
		case x.compiled[mc.id]:
			base = "control(s) compiled to quantum control (deferred measurement)"
		default:
			fed := false
			for _, child := range x.measChildren[mc.id] {
				if x.distinctGates(child) > 0 {
					fed = true
				}
			}
			if fed {
				base = "remainder feeds downstream transparently (deferred measurement)"
			} else {
				base = "terminal readout stripped, pre-measurement state compared"
			}
		}
		tail := ""
		if det == "" {
			tail = "; outcome statistics not verified"
		}
		x.circuit.Notes = append(x.circuit.Notes, mc.label+": "+det+base+tail)
	}
}

// outcomeProb returns P(measuring mod == 0) from the replayed input state.
func (x *extractor) outcomeProb(sys, mod int32) (float64, bool) {
	mgr, ok, err := x.resolveMgr(sys)
	if err != nil || !ok || mgr == nil {
		return 0, false
	}
	pos := -1
	for i, v := range mgr.ModifierID {
		if v == mod {
			pos = i
			break
		}
	}
	if pos < 0 {
		return 0, false
	}
	var p0 float64
	bit := mgr.Size - 1 - int32(pos)
	for i, a := range mgr.Amptitude {
		if (int32(i)>>bit)&1 == 0 {
			p0 += float64(real(a)*real(a) + imag(a)*imag(a))
		}
	}
	return p0, true
}

func jobsPending(jobs []*job, done map[*job]bool) string {
	for _, j := range jobs {
		if !done[j] {
			return j.desc
		}
	}
	return "?"
}

func hookOut(hs []*components.Hook) *components.Hook { return hook0(hs) }

// run executes one replay job and emits its op.
func (x *extractor) run(j *job) error {
	// Resolve inputs transparently through measurement outputs; record
	// object uses for fan-out detection.
	inputs := make([]*qubits.QubitStateManager, len(j.in))
	mods := make([]int32, len(j.in))
	for i, in := range j.in {
		mgr, ok, err := x.resolveMgr(in.sys)
		if err != nil {
			return err
		}
		if !ok || mgr == nil {
			return errf("%s input has no replayed state", j.desc)
		}
		inputs[i] = mgr
		mods[i] = in.mod
	}
	if j.kind == jobCopy {
		if len(inputs) == 0 {
			if x.distinctGates(j.out) > 0 {
				return errf("copy %q has no input but its output feeds computation; the state is undetermined", j.label)
			}
			x.circuit.Notes = append(x.circuit.Notes, "copy "+j.label+": no input, output ignored")
			return nil
		}
		out := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
		out.CopyFrom(inputs[0])
		x.managers[j.out] = out
		x.circuit.Notes = append(x.circuit.Notes, "copy "+j.label+": state carried through")
		x.circuit.Ops = append(x.circuit.Ops, Op{Gate: "copy", Label: j.label})
		return nil
	}
	uniqOf := func() []*qubits.QubitStateManager {
		var uniq []*qubits.QubitStateManager
		seen := map[*qubits.QubitStateManager]bool{}
		for _, m := range inputs {
			if !seen[m] {
				seen[m] = true
				uniq = append(uniq, m)
			}
		}
		return uniq
	}
	qubitsOf := func(ms []int32) ([]int, error) {
		q := make([]int, len(ms))
		for i, mod := range ms {
			qi, ok := x.qIndex[mod]
			if !ok {
				return nil, errf("%s input modifier %d has no QASM qubit", j.desc, mod)
			}
			q[i] = qi
		}
		return q, nil
	}
	trackVer := func(res *qubits.QubitStateManager) {
		x.managers[j.out] = res
		for _, mod := range res.ModifierID {
			x.ver[mod] = res
		}
	}

	if j.kind == jobUnitary {
		if len(j.op) == 0 {
			return errf("%s has an empty operation matrix", j.desc)
		}
		uniq := uniqOf()
		for _, m := range uniq {
			if err := x.use(m, j.id, j.desc); err != nil {
				return err
			}
		}
		res, err := replayGate(inputs, mods, j.op, j.n)
		if err != nil {
			return err
		}
		trackVer(res)
		name, phased := classify(j.op)
		if phased {
			x.phased = true
		}
		if name == "id" {
			x.circuit.Notes = append(x.circuit.Notes, j.desc+": identity operation, no QASM emitted")
			return nil
		}
		q, err := qubitsOf(mods)
		if err != nil {
			return err
		}
		x.circuit.Ops = append(x.circuit.Ops, x.unitaryOp(name, j.op, q, j.label))
		return nil
	}

	// jobControlled with a resolved control.
	switch j.ctl.kind {
	case "const":
		if j.ctl.val == 0 {
			// Merge-only passthrough (nil op skips the multiply).
			res, err := replayGate(inputs, mods, nil, j.n)
			if err != nil {
				return err
			}
			trackVer(res)
			x.circuit.Notes = append(x.circuit.Notes, j.desc+": control 0, passed through")
			x.circuit.Ops = append(x.circuit.Ops, Op{Gate: "identity", Label: j.label})
			return nil
		}
		var mat [][]complex64
		var name string
		if j.ckind >= 0 {
			mat, name = ctrl1Q(j.ckind)
		} else {
			mat = j.op
			var phased bool
			name, phased = classify(mat)
			if phased {
				x.phased = true
			}
		}
		uniq := uniqOf()
		for _, m := range uniq {
			if err := x.use(m, j.id, j.desc); err != nil {
				return err
			}
		}
		res, err := replayGate(inputs, mods, mat, j.n)
		if err != nil {
			return err
		}
		trackVer(res)
		q, err := qubitsOf(mods)
		if err != nil {
			return err
		}
		x.circuit.Ops = append(x.circuit.Ops, x.unitaryOp(name, mat, q, j.label))
		return nil
	default: // qctl: quantum control via deferred measurement
		mi, ok := x.meas[j.ctl.mid]
		if !ok {
			return errf("%s references an unknown measurement", j.desc)
		}
		cmod := mi.mod
		// Current holder from the version map (chains sequential compiled
		// controls); the static carrier restriction was checked up front.
		holder, ok := x.ver[cmod]
		if !ok || holder == nil {
			return errf("%s: control qubit has no replayed state", j.desc)
		}
		all := append([]*qubits.QubitStateManager{holder}, inputs...)
		uniq := []*qubits.QubitStateManager{}
		seen := map[*qubits.QubitStateManager]bool{}
		allmods := append([]int32{cmod}, mods...)
		for _, m := range all {
			if !seen[m] {
				seen[m] = true
				uniq = append(uniq, m)
			}
		}
		for _, m := range uniq {
			if err := x.use(m, j.id, j.desc+" (control/target)"); err != nil {
				return err
			}
		}
		var mat [][]complex64
		var name string
		var qs []int
		if j.ckind >= 0 {
			mat, _ = ctrl1Q(j.ckind)
			// Emit the quantum gate name (cx/cy/cz), not the 1q name.
			name = map[string]string{"x": "cx", "y": "cy", "z": "cz"}[ctrl1QName(j.ckind)]
			qc, err := qubitsOf([]int32{cmod})
			if err != nil {
				return err
			}
			qt, err := qubitsOf(mods)
			if err != nil {
				return err
			}
			qs = append(qc, qt...)
		} else {
			mat = j.op
			name = "unitary"
			qt, err := qubitsOf(mods)
			if err != nil {
				return err
			}
			qc, err := qubitsOf([]int32{cmod})
			if err != nil {
				return err
			}
			for l, r := 0, len(qt)-1; l < r; l, r = l+1, r-1 {
				qt[l], qt[r] = qt[r], qt[l]
			}
			qs = append(qt, qc...)
		}
		cu := controlledMatrix(mat)
		res, err := replayGate(all, allmods, cu, int32(len(allmods)))
		if err != nil {
			return err
		}
		trackVer(res)
		o := Op{Gate: name, Qubits: qs, Label: j.label}
		if name == "unitary" {
			o.Matrix = to128(cu)
			for _, e := range x.circuit.Ops {
				if e.Gate == "unitary" {
					o.Custom++
				}
			}
		}
		x.circuit.Ops = append(x.circuit.Ops, o)
		return nil
	}
}

// unitaryOp builds an Op for a (possibly custom) unitary on qiskit-ordered
// qubits q, reversing multi-qubit lists to little-endian.
func (x *extractor) unitaryOp(name string, mat [][]complex64, q []int, label string) Op {
	o := Op{Gate: name, Qubits: q, Label: label}
	if name == "unitary" {
		o.Matrix = to128(mat)
		for l, r := 0, len(o.Qubits)-1; l < r; l, r = l+1, r-1 {
			o.Qubits[l], o.Qubits[r] = o.Qubits[r], o.Qubits[l]
		}
		for _, e := range x.circuit.Ops {
			if e.Gate == "unitary" {
				o.Custom++
			}
		}
	}
	return o
}

// replayGate mirrors Gate.CalculateOutPut: merge the unique input states in
// hook order, swap the gate qubits to the front, and multiply by op (nil
// op skips the multiply, for unhooked controlled-U passthrough).
func replayGate(inputs []*qubits.QubitStateManager, mods []int32, op [][]complex64, n int32) (*qubits.QubitStateManager, error) {
	var uniq []*qubits.QubitStateManager
	seen := map[*qubits.QubitStateManager]bool{}
	for _, m := range inputs {
		if m == nil {
			return nil, errf("gate input has no replayed state")
		}
		if !seen[m] {
			seen[m] = true
			uniq = append(uniq, m)
		}
	}
	result := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	for _, m := range uniq {
		result.Merge(m)
	}
	for i, mod := range mods {
		col := result.FindID(mod)
		if col < 0 {
			return nil, errf("gate input modifier %d not found after merge", mod)
		}
		result.SwapColumn(int32(i), col)
	}
	if op != nil {
		result.Multiply(op, n)
	}
	return result, nil
}

// NOTE: replayGate takes op == nil to mean merge-and-swap only (no
// multiply), used for unhooked controlled-U and constant-0 passthrough.

// assemble builds Expected in Qiskit order (index bit i = QASM qubit i)
// from the version map: every source modifier has exactly one current
// holder by construction (SSA style, see ver).
func (x *extractor) assemble() error {
	type carrier struct {
		m   *qubits.QubitStateManager
		pos int // position of the modifier in m
	}
	owners := map[int32]carrier{}
	for _, mod := range x.circuit.Modifiers {
		m, ok := x.ver[mod]
		if !ok || m == nil {
			return errf("lost track of qubit modifier %d during replay", mod)
		}
		pos := -1
		for i, v := range m.ModifierID {
			if v == mod {
				pos = i
				break
			}
		}
		if pos < 0 {
			return errf("lost track of qubit modifier %d during replay", mod)
		}
		owners[mod] = carrier{m: m, pos: pos}
	}
	n := x.circuit.NumQubits
	exp := make([]complex64, 1<<n)
	// Per-carrier product assembly: carriers hold disjoint qubit sets whose
	// tensor product is the full state.
	for full := range exp {
		var amp complex64 = 1
		seen := map[*qubits.QubitStateManager]int{}
		for qi, mod := range x.circuit.Modifiers {
			c := owners[mod]
			bit := (full >> qi) & 1
			seen[c.m] |= bit << (c.m.Size - 1 - int32(c.pos))
		}
		for m, idx := range seen {
			amp *= m.Amptitude[idx]
		}
		exp[full] = amp
	}
	if x.phased {
		x.circuit.Notes = append(x.circuit.Notes, "one or more gates matched a standard gate up to global phase")
	}
	x.circuit.Expected = exp
	return nil
}

func to128(op [][]complex64) [][]complex128 {
	out := make([][]complex128, len(op))
	for i, row := range op {
		out[i] = make([]complex128, len(row))
		for j, v := range row {
			out[i][j] = complex128(v)
		}
	}
	return out
}

// classify names a gate matrix: a standard QASM gate when it matches (up to
// a global phase, which the fidelity check ignores), "id" for identity, or
// "unitary" for anything else (emitted via the sidecar).
func classify(op [][]complex64) (name string, phased bool) {
	n := len(op)
	if n == 0 || len(op[0]) != n {
		return "unitary", false
	}
	s := int(math.Round(math.Log2(float64(n))))
	if 1<<s != n || s < 1 || s > 3 {
		return "unitary", false
	}
	refs := map[string][][]complex128{}
	if s == 1 {
		t := 1 / math.Sqrt2
		refs["h"] = [][]complex128{{complex(t, 0), complex(t, 0)}, {complex(t, 0), complex(-t, 0)}}
		refs["x"] = [][]complex128{{0, 1}, {1, 0}}
		refs["y"] = [][]complex128{{0, complex(0, -1)}, {complex(0, 1), 0}}
		refs["z"] = [][]complex128{{1, 0}, {0, -1}}
		refs["id"] = [][]complex128{{1, 0}, {0, 1}}
	} else if s == 2 {
		refs["cx"] = [][]complex128{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}
		refs["cy"] = [][]complex128{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, complex(0, -1)}, {0, 0, complex(0, 1), 0}}
		refs["cz"] = [][]complex128{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1}}
		refs["id"] = identity4()
	}
	for g, ref := range refs {
		if matchUpToPhase(op, ref) {
			return g, !matchExact(op, ref)
		}
	}
	return "unitary", false
}

func identity4() [][]complex128 {
	m := make([][]complex128, 4)
	for i := range m {
		m[i] = make([]complex128, 4)
		m[i][i] = 1
	}
	return m
}

func matchExact(op [][]complex64, ref [][]complex128) bool {
	for i := range ref {
		for j := range ref[i] {
			if cmplx.Abs(complex128(op[i][j])-ref[i][j]) > 1e-6 {
				return false
			}
		}
	}
	return true
}

// matchUpToPhase reports op ~= e^{i*phi} * ref for some phase phi,
// determined by the largest reference element.
func matchUpToPhase(op [][]complex64, ref [][]complex128) bool {
	bi, bj := 0, 0
	best := 0.0
	for i := range ref {
		for j := range ref[i] {
			if m := cmplx.Abs(ref[i][j]); m > best {
				best, bi, bj = m, i, j
			}
		}
	}
	if best == 0 {
		return false
	}
	phase := complex128(op[bi][bj]) / ref[bi][bj]
	if cmplx.Abs(phase) == 0 {
		return false
	}
	phase /= complex(cmplx.Abs(phase), 0)
	for i := range ref {
		for j := range ref[i] {
			if cmplx.Abs(complex128(op[i][j])-phase*ref[i][j]) > 1e-6 {
				return false
			}
		}
	}
	return true
}
