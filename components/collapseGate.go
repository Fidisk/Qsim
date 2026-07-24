package components

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand/v2"
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// CollapseGate is a measurement gate that realizes a single outcome instead
// of preparing both branches like Gate (NewMeasurementGate) does. It has one
// input (a qubit determinator) and two outputs:
//
//   - OutPutHook[0] ("C"): the measured qubit collapsed to |0> or |1>
//   - OutPutHook[1] ("R"): the remaining (n-1)-qubit state of the others,
//     conditioned on the same outcome
//
// The outcome is sampled from the state's probabilities, or forced by
// clicking the gate: ForceMode cycles random -> force 0 -> force 1.
type CollapseGate struct {
	Circle
	ID          int32
	Label       string
	Tooltip     string
	HookList    []*Hook
	OutPutHook  []*Hook
	InputCount  int32
	Measured     bool
	HasRemainder bool // false when the input was a single qubit (no "R" output)
	Result       int32 // the realized outcome: 0 or 1
	ForceMode   int32 // 0 = random, 1 = force 0, 2 = force 1
	OutcomeProbs [2]float64

	// NormalSystem (M3): output the collapsed full state as a normal qubit
	// system (with determinators) instead of a logical bit + logical
	// remainder.
	NormalSystem bool

	pressPos rl.Vector2 // mouse position at drag start, for click detection
}

func NewCollapseGate(x, y, radius float32, color rl.Color, label string) *CollapseGate {
	tmp := CollapseGate{
		Circle:     *NewCircle(x, y, radius, color),
		Label:      label,
		InputCount: 1,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight(glob.GateWeight)
	tmp.Tooltip = "Collapse measurement: outputs the measured qubit and remaining state; click to force 0 or 1"
	snappedDist := utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)
	newHook := NewHook(x-snappedDist, y, glob.HookRadius, config.HookColor)
	newHook.Label = "I"
	newHook.Tooltip = "Measure input: plug a qubit determinator"
	tmp.HookList = append(tmp.HookList, newHook)

	outCollapsed := NewLogicalOutputHook(x+snappedDist, y-glob.HookRadius)
	outCollapsed.Label = "C"
	outCollapsed.Tooltip = "Measured logical bit (|0> or |1>)"
	tmp.HookList = append(tmp.HookList, outCollapsed)
	tmp.OutPutHook = append(tmp.OutPutHook, outCollapsed)

	outRest := NewOutputHook(x+snappedDist, y+glob.HookRadius, glob.OutputHookRadius, config.OutputHookColor)
	outRest.Label = "R"
	outRest.AllowQubitSystem = true
	outRest.Tooltip = "Remaining state output (multi-qubit inputs only)"
	// Unused until a multi-qubit input is measured: the remainder hook only
	// exists when there actually is a remaining state.
	outRest.Hidden = true
	tmp.HookList = append(tmp.HookList, outRest)
	tmp.OutPutHook = append(tmp.OutPutHook, outRest)
	return &tmp
}

// NewCollapseGate3 creates the M3 measurement gate: like CollapseGate it
// realizes a single outcome, but its only output is the collapsed full state
// as a normal qubit system (determinators exposed, so the result can be used
// as a regular system), not a logical one.
func NewCollapseGate3(x, y, radius float32, color rl.Color, label string) *CollapseGate {
	tmp := CollapseGate{
		Circle:       *NewCircle(x, y, radius, color),
		Label:        label,
		InputCount:   1,
		NormalSystem: true,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight(glob.GateWeight)
	tmp.Tooltip = "Collapse measurement: outputs the collapsed qubit system; click to force 0 or 1"
	snappedDist := utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)
	in := NewHook(x-snappedDist, y, glob.HookRadius, config.HookColor)
	in.Label = "I"
	in.Tooltip = "Measure input: plug a qubit determinator"
	tmp.HookList = append(tmp.HookList, in)

	out := NewOutputHook(x+snappedDist, y, glob.OutputHookRadius, config.OutputHookColor)
	out.Label = "O"
	out.AllowQubitSystem = true
	out.Tooltip = "Collapsed system output (normal qubit system)"
	tmp.HookList = append(tmp.HookList, out)
	tmp.OutPutHook = append(tmp.OutPutHook, out)
	return &tmp
}

func (c *CollapseGate) GetID() int32 {
	return c.ID
}

// GetHooks exposes the hook list to the zip helpers (hookOwner interface).
func (c *CollapseGate) GetHooks() []*Hook {
	return c.HookList
}

func (c *CollapseGate) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		c.IsFixed = !c.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		c.DestroyOutPut()
		for _, d := range c.HookList {
			d.Disconnect()
		}
	case utils.IsMouseState(glob.MouseStateErase):
		c.Destroy()
	default:
		c.dragging = true
		*isCursorAvailable = false
		c.holdingCursor = true
		c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		c.VirtualCenter = c.Center
		c.pressPos = worldMouse
	}
}

// CycleForce advances the force-measure setting (random -> 0 -> 1) and
// re-measures if an input is attached.
func (c *CollapseGate) CycleForce() {
	c.ForceMode = (c.ForceMode + 1) % 3
	c.DestroyOutPut()
}

func (c *CollapseGate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) && c.Tooltip != "" {
			glob.TooltipText = c.Tooltip
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			c.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		if utils.IsMouseState(glob.MouseStateNormal) && utils.Dist(c.pressPos, worldMouse) < 5 {
			// plain click (no drag): cycle the force-measure setting
			c.CycleForce()
		}
		c.dragging = false
		c.holdingCursor = false
		*isCursorAvailable = true
		c.Center = c.VirtualCenter
		c.ClearForce()
	}
	if c.dragging {
		raw := rl.Vector2Add(worldMouse, c.offset)
		oldVC := c.VirtualCenter
		c.VirtualCenter = rl.Vector2Lerp(c.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		delta := c.VirtualCenter.Subtract(oldVC)
		for _, d := range c.HookList {
			d.Center = d.Center.Add(delta)
		}
		c.ClearForce()
	} else {
		c.DecayForce()
		c.ApplyForce()
	}

	cnt := 0

	for _, d := range c.HookList {
		if d.Hidden {
			continue
		}
		c.pullToHook(d)

		d.Update(worldMouse, holdingCursor, isCursorAvailable)

		if d.IsHooked && !d.IsOutput {
			cnt++
		}
	}

	if cnt == int(c.InputCount) {
		if !c.Measured {
			c.MeasureOutput()
		}
	} else if c.NormalSystem && c.Measured {
		// M3: drop the collapsed system when the input is removed.
		c.DestroyOutPut()
	} else if len(c.OutPutHook) > 1 && (c.OutPutHook[0].IsHooked || c.OutPutHook[1].IsHooked) && cnt != int(c.InputCount) {
		c.DestroyOutPut()
	}
}

// MeasureOutput realizes a single measurement outcome for the hooked qubit
// and produces the collapsed qubit and the conditioned remainder state.
func (c *CollapseGate) MeasureOutput() {
	var input *Hook
	for _, h := range c.HookList {
		if !h.IsOutput {
			input = h
			break
		}
	}
	if input == nil || !input.IsHooked {
		return
	}
	target := utils.GetObjectFromID(input.TargetID)
	QD, ok := target.(*QubitDeterminator)
	if !ok {
		return
	}
	qp := QD.GetQubitParent()
	if qp == nil || qp.Origin == nil {
		return
	}
	QSM := qp.Origin

	var pos int32
	for i, d := range QSM.ModifierID {
		if QD.ModifierID == d {
			pos = int32(i)
			break
		}
	}

	// p(measuring 0)
	var l complex64
	n := QSM.Size - 1
	for i, d := range QSM.Amptitude {
		if ((i >> (n - pos)) & 1) == 0 {
			l += complex(real(d)*real(d)+imag(d)*imag(d), 0)
		}
	}
	c.OutcomeProbs[0] = cmplx.Abs(complex128(l))
	c.OutcomeProbs[1] = cmplx.Abs(complex128(complex(1, 0) - l))

	// realize one outcome: sampled, or forced by the click setting
	var k int32
	switch c.ForceMode {
	case 1:
		k = 0
	case 2:
		k = 1
	default:
		if rand.Float64() < c.OutcomeProbs[0] {
			k = 0
		} else {
			k = 1
		}
	}
	c.Result = k
	p := c.OutcomeProbs[k]

	if c.NormalSystem {
		// M3: spit the collapsed measured qubit out as its own normal
		// single-qubit system (|0> or |1> on the realized outcome), so it
		// exposes its determinator and can be used as a regular system.
		amps := make([]complex64, 2)
		amps[k] = 1
		state := qubits.NewQubitStateManagerFrom(amps, []int32{QSM.ModifierID[pos]})
		c.spawnOrUpdateNormal(0, state)
		c.Measured = true
		return
	}

	// --- the remaining (n-1)-qubit state, conditioned on the same outcome ---
	restMods := make([]int32, 0, QSM.Size-1)
	for i, d := range QSM.ModifierID {
		if int32(i) != pos {
			restMods = append(restMods, d)
		}
	}
	shift := QSM.Size - 1 - pos
	restSize := int32(1) << (QSM.Size - 1)
	restAmps := make([]complex64, restSize)
	if p > 0 {
		inv := complex64(cmplx.Sqrt(complex128(complex(1/p, 0))))
		for j := int32(0); j < restSize; j++ {
			high := (j >> (shift + 1)) << (shift + 1)
			low := j & ((1 << shift) - 1)
			idx := high | (k << shift) | low
			restAmps[j] = QSM.Amptitude[idx] * inv
		}
	}
	rest := qubits.NewQubitStateManagerFrom(restAmps, restMods)

	c.spawnOrUpdateLogicalBit(0, int(k))
	if QSM.Size > 1 {
		c.HasRemainder = true
		c.OutPutHook[1].Hidden = false
		c.spawnOrUpdate(1, rest)
	} else {
		// Single-qubit input: there is no remaining state, so only the
		// collapsed qubit shows. Hide the remainder hook and clear any stale
		// remainder from a previous multi-qubit measurement.
		c.HasRemainder = false
		c.OutPutHook[1].Hidden = true
		c.OutPutHook[1].DisconnectAndKill()
	}
	c.Measured = true
}

// spawnOrUpdate writes the state into the system hooked to the given output
// hook, or spawns a new system on it.
func (c *CollapseGate) spawnOrUpdate(hookIdx int, state *qubits.QubitStateManager) {
	hook := c.OutPutHook[hookIdx]
	if hook.IsHooked {
		tmp := utils.GetObjectFromID(hook.TargetID)
		if qs, ok := tmp.(*QubitsSystem); ok && qs.Origin != nil {
			qs.Origin.CopyFrom(state)
			return
		}
	}
	tmp := NewQubitsSystem(c.Center.X, c.Center.Y, glob.QubitSystemRadius, config.QubitSystemColor)
	tmp.IsLogical = true
	tmp.Assign(state)
	parent := c.GetParent()
	if parent != nil {
		tmp.SetParent(parent)
		parent.PushComponent(tmp)
	}
	hook.Connect(tmp)
	tmp.ZipDeterminatorsToHooks()
}

// spawnOrUpdateNormal is spawnOrUpdate for M3: the collapsed state goes into
// a normal qubit system (no IsLogical), so it exposes its determinators and
// can be used as a regular system.
func (c *CollapseGate) spawnOrUpdateNormal(hookIdx int, state *qubits.QubitStateManager) {
	hook := c.OutPutHook[hookIdx]
	if hook.IsHooked {
		tmp := utils.GetObjectFromID(hook.TargetID)
		if qs, ok := tmp.(*QubitsSystem); ok && qs.Origin != nil {
			qs.Origin.CopyFrom(state)
			return
		}
	}
	tmp := NewQubitsSystem(c.Center.X, c.Center.Y, glob.QubitSystemRadius, config.QubitSystemColor)
	tmp.Assign(state)
	parent := c.GetParent()
	if parent != nil {
		tmp.SetParent(parent)
		parent.PushComponent(tmp)
	}
	hook.Connect(tmp)
	tmp.ZipDeterminatorsToHooks()
}

func (c *CollapseGate) spawnOrUpdateLogicalBit(hookIdx int, value int) {
	hook := c.OutPutHook[hookIdx]
	if hook.IsHooked {
		tmp := utils.GetObjectFromID(hook.TargetID)
		if lb, ok := tmp.(*LogicalBit); ok {
			lb.SetValue(int32(value))
			return
		}
	}
	tmp := NewLogicalBit(c.Center.X, c.Center.Y, glob.QubitSystemRadius/2, int32(value))
	parent := c.GetParent()
	if parent != nil {
		tmp.SetParent(parent)
		parent.PushComponent(tmp)
	}
	hook.Connect(tmp)
}

func (c *CollapseGate) DestroyOutPut() {
	c.Measured = false
	for _, d := range c.OutPutHook {
		d.DisconnectAndKill()
	}
}

func (c *CollapseGate) Destroy() {
	c.DestroyOutPut()
	for _, d := range c.HookList {
		d.DisconnectAndKill()
	}
	c.GetParent().DeleteChildWithID(c.ID)
}

func (c *CollapseGate) pullToHook(d *Hook) {
	val := utils.Dist(c.Center, d.Center) - utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)

	if math.Abs(float64(val)) <= float64(glob.GateToHookGraceDist) {
		return
	}

	val = float32(math.Max(float64(val), float64(-100)))
	val = float32(math.Min(float64(val), float64(100)))

	tmp := c.Center.Subtract(d.Center).Normalize().Scale(val * glob.GateToHookPullCoeff)

	if d.IsHooked {
		tmp2 := utils.GetObjectFromID(d.TargetID)
		if tmp2 == nil {
			d.Disconnect()
			return
		}
		if !config.PhysicsEnabled {
			return
		}
		t, ok := tmp2.(Component)
		if !ok {
			d.Disconnect()
			return
		}
		t.AddForce(tmp)
		c.AddForce(tmp.Scale(-1))
		return
	}

	if !config.PhysicsEnabled {
		return
	}

	d.AddForce(tmp)
	c.AddForce(tmp.Scale(-1))
}

func (c *CollapseGate) Draw() {
	for _, d := range c.HookList {
		if d.Hidden {
			continue
		}
		start := utils.RectEdgePoint(c.Center, d.Center, c.Radius, c.Radius)
		if d.IsHooked {
			tmp := utils.GetObjectFromID(d.TargetID)
			if tmp == nil {
				d.Disconnect()
				continue
			}
			t, ok := tmp.(Component)
			if !ok {
				d.Disconnect()
				continue
			}

			start = utils.RectEdgePoint(c.Center, t.GetCircle().Center, c.Radius, c.Radius)
			DrawHookLink(start, t, 4, c.Color)
		} else {
			DrawWire(start, d.Center, 4, c.Color)
		}
		d.Draw()
	}

	// Outcome probability on each output line: both outputs came from the
	// same realized outcome. The remainder line only shows when the input
	// had more than one qubit.
	for i, oh := range c.OutPutHook {
		if i == 1 && !c.HasRemainder {
			continue
		}
		mid := rl.Vector2Scale(rl.Vector2Add(c.Center, oh.Center), 0.5)
		label := fmt.Sprintf("|%d>, probability: %.2f", c.Result, c.OutcomeProbs[c.Result])
		textWidth := rl.MeasureText(label, 14)
		bg := rl.NewRectangle(mid.X-float32(textWidth)/2-4, mid.Y-22, float32(textWidth)+8, 18)
		rl.DrawRectangleRec(bg, rl.Fade(rl.Black, 0.6))
		rl.DrawText(label, int32(mid.X)-textWidth/2, int32(mid.Y)-20, 14, rl.White)
	}

	rect := rl.Rectangle{
		X:      c.Center.X - c.Radius,
		Y:      c.Center.Y - c.Radius,
		Width:  c.Radius * 2,
		Height: c.Radius * 2,
	}
	rl.DrawRectangleRounded(rect, 0.15, 6, config.ColorBg)
	thickness := float32(4.0)
	if c.IsFixed {
		thickness = 5.0
	}
	rl.DrawRectangleRoundedLinesEx(rect, 0.15, 6, thickness, c.Color)

	if c.Label != "" {
		fontSize := int32(c.Radius / 2)
		textWidth := rl.MeasureText(c.Label, fontSize)
		textX := int32(c.Center.X) - textWidth/2
		textY := int32(c.Center.Y) - fontSize/2
		rl.DrawText(c.Label, textX, textY, fontSize, c.Color)
	}

	// Force-measure indicator: R (random), 0, or 1 (forced)
	mode := []string{"R", "0", "1"}[c.ForceMode]
	rl.DrawText(mode, int32(c.Center.X+c.Radius-14), int32(c.Center.Y-c.Radius+4), 14, rl.Gold)
}

func (c *CollapseGate) DrawGhost() {
	ghostColor := rl.Fade(c.Color, 0.3)
	ghostBg := rl.Fade(config.ColorBg, 0.3)

	rect := rl.Rectangle{
		X:      c.Center.X - c.Radius,
		Y:      c.Center.Y - c.Radius,
		Width:  c.Radius * 2,
		Height: c.Radius * 2,
	}
	rl.DrawRectangleRec(rect, ghostBg)
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)

	if c.Label != "" {
		fontSize := int32(c.Radius / 2)
		textWidth := rl.MeasureText(c.Label, fontSize)
		textX := int32(c.Center.X) - textWidth/2
		textY := int32(c.Center.Y) - fontSize/2
		rl.DrawText(c.Label, textX, textY, fontSize, ghostColor)
	}

	for _, d := range c.HookList {
		start := utils.RectEdgePoint(c.Center, d.Center, c.Radius, c.Radius)
		rl.DrawLineEx(start, d.Center, 4, ghostColor)
		d.DrawGhost()
	}
}

func (c *CollapseGate) GetChildCircles() []*Circle {
	var children []*Circle
	for _, h := range c.HookList {
		if !h.IsHooked {
			children = append(children, h.GetCircle())
		}
	}
	return children
}

func (c *CollapseGate) PostUpdate() {
	for i := range c.HookList {
		for j := i + 1; j < len(c.HookList); j++ {
			c.HookList[i].GetCircle().AntiGravity(c.HookList[j].GetCircle())
		}
	}
}
