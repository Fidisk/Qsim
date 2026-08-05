package components

import (
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	controlledGateWidth  = 120
	controlledGateHeight = 90
)

// Controlled-gate kinds: which single-qubit operation is applied while the
// control bit is 1.
const (
	CtrlX int32 = iota
	CtrlY
	CtrlZ
)

// ControlledGate is a CX/CY/CZ-style gate whose control is a LogicalBit
// instead of a qubit. It has one qubit input (a determinator), one logical
// bit control input, and one qubit-system output. The output system has the
// same size as the input system: the control bit is classical, so it never
// expands the state. An unhooked control reads as 0 (identity passthrough),
// so disconnecting the bit never errors or tears down the output.
type ControlledGate struct {
	Circle
	ID      int32
	Kind    int32
	Label   string
	Tooltip string
	Color   rl.Color
	Width   float32
	Height  float32

	Operation [][]complex64

	InQubit   *Hook
	InControl *Hook
	OutHook   *Hook

	hovered bool
}

func NewControlledGate(x, y float32, color rl.Color, kind int32) *ControlledGate {
	w := float32(controlledGateWidth)
	h := float32(controlledGateHeight)
	cg := &ControlledGate{
		Circle: *NewCircle(x, y, maxF(w, h)/2, color),
		Kind:   kind,
		Color:  color,
		Width:  w,
		Height: h,
	}
	i := complex64(complex(0, 1))
	switch kind {
	case CtrlY:
		cg.Label = "cY"
		cg.Operation = [][]complex64{{0, -i}, {i, 0}}
		cg.Tooltip = "Bit-controlled Y: applies Y to the qubit while the control bit is 1"
	case CtrlZ:
		cg.Label = "cZ"
		cg.Operation = [][]complex64{{1, 0}, {0, -1}}
		cg.Tooltip = "Bit-controlled Z: applies Z to the qubit while the control bit is 1"
	default:
		cg.Label = "cX"
		cg.Operation = [][]complex64{{0, 1}, {1, 0}}
		cg.Tooltip = "Bit-controlled X: applies X to the qubit while the control bit is 1"
	}
	cg.ID = utils.GenerateID(cg)
	cg.SetWeight(glob.GateWeight)

	off := utils.SnapToGrid(glob.QubitSystemCellWidth/2, config.SnapToGridInterval)

	cg.InQubit = NewHook(x-w/2-off, y-h/4, glob.HookRadius, config.HookColor)
	cg.InQubit.Label = "Q"
	cg.InQubit.Tooltip = "Controlled input: plug a qubit determinator"

	cg.InControl = NewLogicalHook(x-w/2-off, y+h/4)
	cg.InControl.Label = "C"
	cg.InControl.Tooltip = "Control input: connect a logical bit (defaults to 0)"

	cg.OutHook = NewOutputHook(x+w/2+off, y, glob.OutputHookRadius, config.OutputHookColor)
	cg.OutHook.Label = "O"
	cg.OutHook.AllowQubitSystem = true
	cg.OutHook.Tooltip = "Controlled output: the transformed qubit system (same size as the input)"
	return cg
}

func (cg *ControlledGate) GetID() int32 { return cg.ID }

// GetHooks exposes the hooks to the zip helpers (hookOwner interface).
func (cg *ControlledGate) GetHooks() []*Hook {
	return []*Hook{cg.InQubit, cg.InControl, cg.OutHook}
}

func (cg *ControlledGate) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, rl.Rectangle{
		X:      cg.Center.X - cg.Width/2,
		Y:      cg.Center.Y - cg.Height/2,
		Width:  cg.Width,
		Height: cg.Height,
	})
}

func (cg *ControlledGate) anchorQubit() rl.Vector2 {
	return rl.Vector2{X: cg.Center.X - cg.Width/2 - glob.QubitSystemCellWidth/2, Y: cg.Center.Y - cg.Height/4}
}

func (cg *ControlledGate) anchorControl() rl.Vector2 {
	return rl.Vector2{X: cg.Center.X - cg.Width/2 - glob.QubitSystemCellWidth/2, Y: cg.Center.Y + cg.Height/4}
}

func (cg *ControlledGate) anchorOut() rl.Vector2 {
	return rl.Vector2{X: cg.Center.X + cg.Width/2 + glob.QubitSystemCellWidth/2, Y: cg.Center.Y}
}

func (cg *ControlledGate) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		cg.IsFixed = !cg.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		cg.DestroyOutPut()
		cg.InQubit.Disconnect()
		cg.InControl.Disconnect()
	case utils.IsMouseState(glob.MouseStateErase):
		cg.Destroy()
	default:
		cg.dragging = true
		*isCursorAvailable = false
		cg.holdingCursor = true
		cg.offset = rl.Vector2Subtract(cg.Center, worldMouse)
		cg.VirtualCenter = cg.Center
	}
}

func (cg *ControlledGate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	cg.hovered = cg.CheckCollide(worldMouse) && !cg.dragging
	if cg.CheckCollide(worldMouse) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) && cg.Tooltip != "" {
			glob.TooltipText = cg.Tooltip
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (cg.holdingCursor || *isCursorAvailable) {
			cg.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && cg.holdingCursor {
		cg.dragging = false
		cg.holdingCursor = false
		*isCursorAvailable = true
		cg.Center = rl.Vector2{
			X: utils.SnapToGrid(cg.VirtualCenter.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(cg.VirtualCenter.Y, config.SnapToGridInterval),
		}
		cg.VirtualCenter = cg.Center
		cg.ClearForce()
	}
	if cg.dragging {
		raw := rl.Vector2Add(worldMouse, cg.offset)
		oldVC := cg.VirtualCenter
		cg.VirtualCenter = rl.Vector2Lerp(cg.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		delta := cg.VirtualCenter.Subtract(oldVC)
		for _, h := range cg.GetHooks() {
			h.Center = h.Center.Add(delta)
		}
		cg.ClearForce()
	} else {
		cg.DecayForce()
		cg.ApplyForce()
	}

	pullHookToAnchor(&cg.Circle, cg.InQubit, cg.anchorQubit())
	pullHookToAnchor(&cg.Circle, cg.InControl, cg.anchorControl())
	pullHookToAnchor(&cg.Circle, cg.OutHook, cg.anchorOut())

	for _, h := range cg.GetHooks() {
		h.Update(worldMouse, holdingCursor, isCursorAvailable)
	}

	cg.updateOutput()
}

// controlValue reads the control bit; an unhooked control counts as 0.
func (cg *ControlledGate) controlValue() int32 {
	if bit := inputBit(cg.InControl); bit != nil {
		return bit.Value
	}
	return 0
}

// updateOutput applies the single-qubit operation to the hooked qubit when
// the control bit is 1. The system is transformed in place size-wise: no
// qubits are added or removed.
func (cg *ControlledGate) updateOutput() {
	if !cg.InQubit.IsHooked {
		cg.DestroyOutPut()
		return
	}
	QD, ok := utils.GetObjectFromID(cg.InQubit.TargetID).(*QubitDeterminator)
	if !ok || QD == nil {
		return
	}
	qp := QD.GetQubitParent()
	if qp == nil || qp.Origin == nil {
		return
	}
	src := qp.Origin

	amps := make([]complex64, len(src.Amptitude))
	copy(amps, src.Amptitude)
	mods := make([]int32, len(src.ModifierID))
	copy(mods, src.ModifierID)
	result := qubits.NewQubitStateManagerFrom(amps, mods)

	if cg.controlValue() == 1 {
		pos := result.FindID(QD.ModifierID)
		if pos < 0 {
			return
		}
		// Bring the target qubit to the front, apply the 1-qubit operation,
		// then restore the original column order.
		result.SwapColumn(0, pos)
		result.Multiply(cg.Operation, 1)
		result.SwapColumn(0, pos)
	}

	if cg.OutHook.IsHooked {
		tmp := utils.GetObjectFromID(cg.OutHook.TargetID)
		if qs, ok := tmp.(*QubitsSystem); ok && qs.Origin != nil {
			qs.Origin.CopyFrom(result)
			return
		}
	}

	tmp := NewQubitsSystem(cg.Center.X, cg.Center.Y, glob.QubitSystemRadius, config.QubitSystemColor)
	tmp.Assign(result)
	parent := cg.GetParent()
	if parent != nil {
		tmp.SetParent(parent)
		parent.PushComponent(tmp)
	}
	cg.OutHook.Connect(tmp)
	tmp.ZipDeterminatorsToHooks()
}

func (cg *ControlledGate) DestroyOutPut() {
	cg.OutHook.DisconnectAndKill()
}

func (cg *ControlledGate) Destroy() {
	cg.DestroyOutPut()
	cg.InQubit.Disconnect()
	cg.InControl.Disconnect()
	if parent := cg.GetParent(); parent != nil {
		parent.DeleteChildWithID(cg.ID)
	}
}

func (cg *ControlledGate) Draw() {
	drawHookWire(cg.Center, cg.InQubit, cg.Width/2, cg.Height/2, config.HookColor)
	drawHookWire(cg.Center, cg.InControl, cg.Width/2, cg.Height/2, cg.Color)
	drawHookWire(cg.Center, cg.OutHook, cg.Width/2, cg.Height/2, cg.Color)
	for _, h := range cg.GetHooks() {
		h.Draw()
	}

	rect := rl.Rectangle{
		X:      cg.Center.X - cg.Width/2,
		Y:      cg.Center.Y - cg.Height/2,
		Width:  cg.Width,
		Height: cg.Height,
	}
	rl.DrawRectangleRounded(rect, 0.15, 6, config.ColorBg)
	thickness := float32(4.0)
	if cg.IsFixed {
		thickness = 5.0
	}
	borderColor := cg.Color
	if cg.hovered {
		borderColor = lighten(cg.Color, 0.4)
	}
	rl.DrawRectangleRoundedLinesEx(rect, 0.15, 6, thickness, borderColor)

	// Control dot: fills while the control bit is 1.
	dot := rl.Vector2{X: rect.X + 12, Y: cg.Center.Y + cg.Height/4}
	if cg.controlValue() == 1 {
		rl.DrawCircleV(dot, 6, rl.Yellow)
	} else {
		rl.DrawCircleLines(int32(dot.X), int32(dot.Y), 6, rl.Fade(borderColor, 0.7))
	}

	if cg.Label != "" {
		fontSize := int32(cg.Height / 3)
		textWidth := rl.MeasureText(cg.Label, fontSize)
		rl.DrawText(cg.Label, int32(cg.Center.X)-textWidth/2, int32(cg.Center.Y)-fontSize/2, fontSize, borderColor)
	}
}

func (cg *ControlledGate) DrawGhost() {
	ghostColor := rl.Fade(cg.Color, 0.3)
	rect := rl.Rectangle{
		X:      cg.Center.X - cg.Width/2,
		Y:      cg.Center.Y - cg.Height/2,
		Width:  cg.Width,
		Height: cg.Height,
	}
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)
	for _, h := range cg.GetHooks() {
		edge := utils.RectEdgePoint(cg.Center, h.Center, cg.Width/2, cg.Height/2)
		rl.DrawLineEx(edge, h.Center, 4, ghostColor)
		h.DrawGhost()
	}
}

func (cg *ControlledGate) GetChildCircles() []*Circle {
	var children []*Circle
	for _, h := range cg.GetHooks() {
		if h != nil && !h.IsHooked {
			children = append(children, h.GetCircle())
		}
	}
	return children
}

func (cg *ControlledGate) PostUpdate() {}
