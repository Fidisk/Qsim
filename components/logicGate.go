package components

import (
	"qsim/config"
	glob "qsim/globals"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	logicButtonWidth  = 80
	logicButtonHeight = 60
	logicGateWidth    = 120
	logicGateHeight   = 80
	lightBodyRadius   = 40
)

// LogicButton is a classical bit source: no input, one logical-bit output.
// Clicking the body toggles the emitted value between 0 and 1.
type LogicButton struct {
	Circle
	ID      int32
	Tooltip string
	Color   rl.Color
	Width   float32
	Height  float32
	Value   int32

	// OutputID tracks the spawned LogicalBit so we don't create duplicates.
	OutputID int32
	OutHook  *Hook

	pressPos rl.Vector2
	hovered  bool
}

func NewLogicButton(x, y float32, color rl.Color) *LogicButton {
	w := float32(logicButtonWidth)
	h := float32(logicButtonHeight)
	lb := &LogicButton{
		Circle:  *NewCircle(x, y, maxF(w, h)/2, color),
		Tooltip: "Logic button: click to toggle the output bit between 0 and 1",
		Color:   color,
		Width:   w,
		Height:  h,
	}
	lb.ID = utils.GenerateID(lb)
	lb.SetWeight(glob.GateWeight)

	off := utils.SnapToGrid(glob.QubitSystemCellWidth/2, config.SnapToGridInterval)
	lb.OutHook = NewLogicalOutputHook(x+w/2+off, y)
	lb.OutHook.Label = "O"
	lb.OutHook.Tooltip = "Button output: logical bit (click the button to toggle)"
	return lb
}

func (lb *LogicButton) GetID() int32 { return lb.ID }

// GetHooks exposes the output hook to the zip helpers (hookOwner interface).
func (lb *LogicButton) GetHooks() []*Hook { return []*Hook{lb.OutHook} }

func (lb *LogicButton) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, rl.Rectangle{
		X:      lb.Center.X - lb.Width/2,
		Y:      lb.Center.Y - lb.Height/2,
		Width:  lb.Width,
		Height: lb.Height,
	})
}

func (lb *LogicButton) anchorOut() rl.Vector2 {
	return rl.Vector2{X: lb.Center.X + lb.Width/2 + glob.QubitSystemCellWidth/2, Y: lb.Center.Y}
}

func (lb *LogicButton) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		lb.IsFixed = !lb.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		lb.DestroyOutPut()
	case utils.IsMouseState(glob.MouseStateErase):
		lb.Destroy()
	default:
		lb.dragging = true
		*isCursorAvailable = false
		lb.holdingCursor = true
		lb.offset = rl.Vector2Subtract(lb.Center, worldMouse)
		lb.VirtualCenter = lb.Center
		lb.pressPos = worldMouse
	}
}

// Toggle flips the emitted bit; the change is pushed to the output in
// updateOutput on the same frame.
func (lb *LogicButton) Toggle() {
	if lb.Value == 0 {
		lb.Value = 1
	} else {
		lb.Value = 0
	}
}

func (lb *LogicButton) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	lb.hovered = lb.CheckCollide(worldMouse) && !lb.dragging
	if lb.CheckCollide(worldMouse) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) && lb.Tooltip != "" {
			glob.TooltipText = lb.Tooltip
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (lb.holdingCursor || *isCursorAvailable) {
			lb.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && lb.holdingCursor {
		if utils.IsMouseState(glob.MouseStateNormal) && utils.Dist(lb.pressPos, worldMouse) < 5 {
			// plain click (no drag): toggle the output value
			lb.Toggle()
		}
		lb.dragging = false
		lb.holdingCursor = false
		*isCursorAvailable = true
		lb.Center = rl.Vector2{
			X: utils.SnapToGrid(lb.VirtualCenter.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(lb.VirtualCenter.Y, config.SnapToGridInterval),
		}
		lb.VirtualCenter = lb.Center
		lb.ClearForce()
	}
	if lb.dragging {
		raw := rl.Vector2Add(worldMouse, lb.offset)
		oldVC := lb.VirtualCenter
		lb.VirtualCenter = rl.Vector2Lerp(lb.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		delta := lb.VirtualCenter.Subtract(oldVC)
		lb.OutHook.Center = lb.OutHook.Center.Add(delta)
		lb.ClearForce()
	} else {
		lb.DecayForce()
		lb.ApplyForce()
	}

	pullHookToAnchor(&lb.Circle, lb.OutHook, lb.anchorOut())
	lb.OutHook.Update(worldMouse, holdingCursor, isCursorAvailable)

	lb.updateOutput()
}

// updateOutput writes the current value into the hooked LogicalBit, or spawns
// one on the output hook if there is none.
func (lb *LogicButton) updateOutput() {
	if lb.OutHook.IsHooked {
		out := utils.GetObjectFromID(lb.OutHook.TargetID)
		if bit, ok := out.(*LogicalBit); ok {
			bit.SetValue(lb.Value)
			lb.OutputID = bit.ID
			return
		}
	}
	// The tracked bit may have been dragged onto another hook: keep driving
	// it so the button still controls it, and only respawn once it is gone.
	if lb.OutputID != 0 {
		if bit, ok := utils.GetObjectFromID(lb.OutputID).(*LogicalBit); ok {
			bit.SetValue(lb.Value)
			return
		}
		lb.OutputID = 0
	}
	if lb.OutputID == 0 || utils.GetObjectFromID(lb.OutputID) == nil {
		bit := NewLogicalBit(lb.OutHook.Center.X, lb.OutHook.Center.Y, glob.QubitSystemRadius/2, lb.Value)
		parent := lb.GetParent()
		if parent != nil {
			bit.SetParent(parent)
			parent.PushComponent(bit)
		}
		lb.OutHook.Connect(bit)
		lb.OutputID = bit.ID
	}
}

func (lb *LogicButton) DestroyOutPut() {
	lb.OutHook.DisconnectAndKill()
	lb.OutputID = 0
}

func (lb *LogicButton) Destroy() {
	lb.DestroyOutPut()
	if parent := lb.GetParent(); parent != nil {
		parent.DeleteChildWithID(lb.ID)
	}
}

func (lb *LogicButton) Draw() {
	drawHookWire(lb.Center, lb.OutHook, lb.Width/2, lb.Height/2, lb.Color)
	lb.OutHook.Draw()

	rect := rl.Rectangle{
		X:      lb.Center.X - lb.Width/2,
		Y:      lb.Center.Y - lb.Height/2,
		Width:  lb.Width,
		Height: lb.Height,
	}
	rl.DrawRectangleRec(rect, config.ColorBg)
	thickness := float32(4.0)
	if lb.IsFixed {
		thickness = 5.0
	}
	borderColor := lb.Color
	if lb.hovered {
		borderColor = lighten(lb.Color, 0.4)
	}
	rl.DrawRectangleLinesEx(rect, thickness, borderColor)

	fontSize := int32(lb.Height / 2)
	text := "0"
	if lb.Value == 1 {
		text = "1"
	}
	textWidth := rl.MeasureText(text, fontSize)
	rl.DrawText(text, int32(lb.Center.X)-textWidth/2, int32(lb.Center.Y)-fontSize/2, fontSize, borderColor)
}

func (lb *LogicButton) DrawGhost() {
	ghostColor := rl.Fade(lb.Color, 0.3)
	rect := rl.Rectangle{
		X:      lb.Center.X - lb.Width/2,
		Y:      lb.Center.Y - lb.Height/2,
		Width:  lb.Width,
		Height: lb.Height,
	}
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)
	outEdge := utils.RectEdgePoint(lb.Center, lb.OutHook.Center, lb.Width/2, lb.Height/2)
	rl.DrawLineEx(outEdge, lb.OutHook.Center, 4, ghostColor)
	lb.OutHook.DrawGhost()
}

func (lb *LogicButton) GetChildCircles() []*Circle {
	if lb.OutHook == nil || lb.OutHook.IsHooked {
		return nil
	}
	return []*Circle{lb.OutHook.GetCircle()}
}

func (lb *LogicButton) PostUpdate() {}

// Logic gate kinds.
const (
	LogicNot int32 = iota
	LogicAnd
	LogicOr
)

// LogicGate is a classical NOT/AND/OR gate over LogicalBit inputs, producing
// a LogicalBit output. It only computes while every required input is hooked.
type LogicGate struct {
	Circle
	ID      int32
	Kind    int32
	Label   string
	Tooltip string
	Color   rl.Color
	Width   float32
	Height  float32

	InA     *Hook
	InB     *Hook // nil for NOT
	OutHook *Hook

	// OutputID tracks the spawned LogicalBit so we don't create duplicates.
	OutputID int32

	hovered bool
}

func NewLogicGate(x, y float32, color rl.Color, kind int32) *LogicGate {
	w := float32(logicGateWidth)
	h := float32(logicGateHeight)
	lg := &LogicGate{
		Circle: *NewCircle(x, y, maxF(w, h)/2, color),
		Kind:   kind,
		Color:  color,
		Width:  w,
		Height: h,
	}
	switch kind {
	case LogicAnd:
		lg.Label = "AND"
		lg.Tooltip = "AND gate: outputs 1 when both input bits are 1"
	case LogicOr:
		lg.Label = "OR"
		lg.Tooltip = "OR gate: outputs 1 when at least one input bit is 1"
	default:
		lg.Label = "NOT"
		lg.Tooltip = "NOT gate: outputs the inverse of the input bit"
	}
	lg.ID = utils.GenerateID(lg)
	lg.SetWeight(glob.GateWeight)

	off := utils.SnapToGrid(glob.QubitSystemCellWidth/2, config.SnapToGridInterval)

	inAY := y
	inB := false
	if kind != LogicNot {
		inAY = y - h/4
		inB = true
	}
	lg.InA = NewLogicalHook(x-w/2-off, inAY)
	lg.InA.Label = "A"
	lg.InA.Tooltip = "Logic input A: connect a logical bit"

	if inB {
		lg.InB = NewLogicalHook(x-w/2-off, y+h/4)
		lg.InB.Label = "B"
		lg.InB.Tooltip = "Logic input B: connect a logical bit"
	}

	lg.OutHook = NewLogicalOutputHook(x+w/2+off, y)
	lg.OutHook.Label = "O"
	lg.OutHook.Tooltip = "Logic output: logical bit"
	return lg
}

func (lg *LogicGate) GetID() int32 { return lg.ID }

// GetHooks exposes the input/output hooks to the zip helpers (hookOwner interface).
func (lg *LogicGate) GetHooks() []*Hook {
	hooks := []*Hook{lg.InA, lg.OutHook}
	if lg.InB != nil {
		hooks = append(hooks, lg.InB)
	}
	return hooks
}

func (lg *LogicGate) inputs() []*Hook {
	if lg.InB != nil {
		return []*Hook{lg.InA, lg.InB}
	}
	return []*Hook{lg.InA}
}

func (lg *LogicGate) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, rl.Rectangle{
		X:      lg.Center.X - lg.Width/2,
		Y:      lg.Center.Y - lg.Height/2,
		Width:  lg.Width,
		Height: lg.Height,
	})
}

func (lg *LogicGate) anchorA() rl.Vector2 {
	y := lg.Center.Y
	if lg.InB != nil {
		y -= lg.Height / 4
	}
	return rl.Vector2{X: lg.Center.X - lg.Width/2 - glob.QubitSystemCellWidth/2, Y: y}
}

func (lg *LogicGate) anchorB() rl.Vector2 {
	return rl.Vector2{X: lg.Center.X - lg.Width/2 - glob.QubitSystemCellWidth/2, Y: lg.Center.Y + lg.Height/4}
}

func (lg *LogicGate) anchorOut() rl.Vector2 {
	return rl.Vector2{X: lg.Center.X + lg.Width/2 + glob.QubitSystemCellWidth/2, Y: lg.Center.Y}
}

func (lg *LogicGate) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		lg.IsFixed = !lg.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		lg.DestroyOutPut()
		for _, h := range lg.inputs() {
			h.Disconnect()
		}
	case utils.IsMouseState(glob.MouseStateErase):
		lg.Destroy()
	default:
		lg.dragging = true
		*isCursorAvailable = false
		lg.holdingCursor = true
		lg.offset = rl.Vector2Subtract(lg.Center, worldMouse)
		lg.VirtualCenter = lg.Center
	}
}

func (lg *LogicGate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	lg.hovered = lg.CheckCollide(worldMouse) && !lg.dragging
	if lg.CheckCollide(worldMouse) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) && lg.Tooltip != "" {
			glob.TooltipText = lg.Tooltip
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (lg.holdingCursor || *isCursorAvailable) {
			lg.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && lg.holdingCursor {
		lg.dragging = false
		lg.holdingCursor = false
		*isCursorAvailable = true
		lg.Center = rl.Vector2{
			X: utils.SnapToGrid(lg.VirtualCenter.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(lg.VirtualCenter.Y, config.SnapToGridInterval),
		}
		lg.VirtualCenter = lg.Center
		lg.ClearForce()
	}
	if lg.dragging {
		raw := rl.Vector2Add(worldMouse, lg.offset)
		oldVC := lg.VirtualCenter
		lg.VirtualCenter = rl.Vector2Lerp(lg.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		delta := lg.VirtualCenter.Subtract(oldVC)
		for _, h := range lg.GetHooks() {
			h.Center = h.Center.Add(delta)
		}
		lg.ClearForce()
	} else {
		lg.DecayForce()
		lg.ApplyForce()
	}

	pullHookToAnchor(&lg.Circle, lg.InA, lg.anchorA())
	if lg.InB != nil {
		pullHookToAnchor(&lg.Circle, lg.InB, lg.anchorB())
	}
	pullHookToAnchor(&lg.Circle, lg.OutHook, lg.anchorOut())

	for _, h := range lg.GetHooks() {
		h.Update(worldMouse, holdingCursor, isCursorAvailable)
	}

	lg.updateOutput()
}

// inputBit returns the LogicalBit hooked to h, or nil.
func inputBit(h *Hook) *LogicalBit {
	if !h.IsHooked {
		return nil
	}
	bit, _ := utils.GetObjectFromID(h.TargetID).(*LogicalBit)
	return bit
}

func (lg *LogicGate) updateOutput() {
	a := inputBit(lg.InA)
	var b *LogicalBit
	if lg.InB != nil {
		b = inputBit(lg.InB)
	}
	// The gate only computes while every required input is a logical bit;
	// otherwise the stale output is removed, like CollapseGate.
	if a == nil || (lg.InB != nil && b == nil) {
		lg.DestroyOutPut()
		return
	}

	value := int32(0)
	switch lg.Kind {
	case LogicAnd:
		if a.Value == 1 && b.Value == 1 {
			value = 1
		}
	case LogicOr:
		if a.Value == 1 || b.Value == 1 {
			value = 1
		}
	default: // LogicNot
		if a.Value == 0 {
			value = 1
		}
	}

	if lg.OutHook.IsHooked {
		out := utils.GetObjectFromID(lg.OutHook.TargetID)
		if bit, ok := out.(*LogicalBit); ok {
			bit.SetValue(value)
			lg.OutputID = bit.ID
			return
		}
	}
	// The tracked bit may have been dragged onto another hook: keep driving
	// it so the gate still controls it, and only respawn once it is gone.
	if lg.OutputID != 0 {
		if bit, ok := utils.GetObjectFromID(lg.OutputID).(*LogicalBit); ok {
			bit.SetValue(value)
			return
		}
		lg.OutputID = 0
	}
	if lg.OutputID == 0 || utils.GetObjectFromID(lg.OutputID) == nil {
		bit := NewLogicalBit(lg.OutHook.Center.X, lg.OutHook.Center.Y, glob.QubitSystemRadius/2, value)
		parent := lg.GetParent()
		if parent != nil {
			bit.SetParent(parent)
			parent.PushComponent(bit)
		}
		lg.OutHook.Connect(bit)
		lg.OutputID = bit.ID
	}
}

func (lg *LogicGate) DestroyOutPut() {
	lg.OutHook.DisconnectAndKill()
	lg.OutputID = 0
}

func (lg *LogicGate) Destroy() {
	lg.DestroyOutPut()
	for _, h := range lg.inputs() {
		h.DisconnectAndKill()
	}
	if parent := lg.GetParent(); parent != nil {
		parent.DeleteChildWithID(lg.ID)
	}
}

func (lg *LogicGate) Draw() {
	drawHookWire(lg.Center, lg.InA, lg.Width/2, lg.Height/2, lg.Color)
	if lg.InB != nil {
		drawHookWire(lg.Center, lg.InB, lg.Width/2, lg.Height/2, lg.Color)
	}
	drawHookWire(lg.Center, lg.OutHook, lg.Width/2, lg.Height/2, lg.Color)
	for _, h := range lg.GetHooks() {
		h.Draw()
	}

	rect := rl.Rectangle{
		X:      lg.Center.X - lg.Width/2,
		Y:      lg.Center.Y - lg.Height/2,
		Width:  lg.Width,
		Height: lg.Height,
	}
	rl.DrawRectangleRec(rect, config.ColorBg)
	thickness := float32(4.0)
	if lg.IsFixed {
		thickness = 5.0
	}
	borderColor := lg.Color
	if lg.hovered {
		borderColor = lighten(lg.Color, 0.4)
	}
	rl.DrawRectangleLinesEx(rect, thickness, borderColor)

	fontSize := int32(lg.Height / 3)
	if lg.Label != "" {
		textWidth := rl.MeasureText(lg.Label, fontSize)
		rl.DrawText(lg.Label, int32(lg.Center.X)-textWidth/2, int32(lg.Center.Y)-fontSize/2, fontSize, borderColor)
	}
}

func (lg *LogicGate) DrawGhost() {
	ghostColor := rl.Fade(lg.Color, 0.3)
	rect := rl.Rectangle{
		X:      lg.Center.X - lg.Width/2,
		Y:      lg.Center.Y - lg.Height/2,
		Width:  lg.Width,
		Height: lg.Height,
	}
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)
	for _, h := range lg.GetHooks() {
		edge := utils.RectEdgePoint(lg.Center, h.Center, lg.Width/2, lg.Height/2)
		rl.DrawLineEx(edge, h.Center, 4, ghostColor)
		h.DrawGhost()
	}
}

func (lg *LogicGate) GetChildCircles() []*Circle {
	var children []*Circle
	for _, h := range lg.GetHooks() {
		if h != nil && !h.IsHooked {
			children = append(children, h.GetCircle())
		}
	}
	return children
}

func (lg *LogicGate) PostUpdate() {}

// Light is a read-only logical-bit indicator: it shines while the bit hooked
// to its input is 1.
type Light struct {
	Circle
	ID      int32
	Tooltip string
	Color   rl.Color

	InHook *Hook

	hovered bool
}

func NewLight(x, y float32, color rl.Color) *Light {
	l := &Light{
		Circle:  *NewCircle(x, y, lightBodyRadius, color),
		Tooltip: "Light: shines while the connected logical bit is 1",
		Color:   color,
	}
	l.ID = utils.GenerateID(l)
	l.SetWeight(glob.GateWeight)

	off := utils.SnapToGrid(glob.QubitSystemCellWidth/2, config.SnapToGridInterval)
	l.InHook = NewLogicalHook(x-lightBodyRadius-off, y)
	l.InHook.Label = "I"
	l.InHook.Tooltip = "Light input: connect a logical bit"
	return l
}

func (l *Light) GetID() int32 { return l.ID }

// GetHooks exposes the input hook to the zip helpers (hookOwner interface).
func (l *Light) GetHooks() []*Hook { return []*Hook{l.InHook} }

func (l *Light) anchorIn() rl.Vector2 {
	return rl.Vector2{X: l.Center.X - l.Radius - glob.QubitSystemCellWidth/2, Y: l.Center.Y}
}

// Lit reports whether the hooked input bit is currently 1.
func (l *Light) Lit() bool {
	bit := inputBit(l.InHook)
	return bit != nil && bit.Value == 1
}

func (l *Light) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		l.IsFixed = !l.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		l.InHook.Disconnect()
	case utils.IsMouseState(glob.MouseStateErase):
		l.Destroy()
	default:
		l.dragging = true
		*isCursorAvailable = false
		l.holdingCursor = true
		l.offset = rl.Vector2Subtract(l.Center, worldMouse)
		l.VirtualCenter = l.Center
	}
}

func (l *Light) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	l.hovered = rl.CheckCollisionPointCircle(worldMouse, l.Center, l.Radius) && !l.dragging
	if rl.CheckCollisionPointCircle(worldMouse, l.Center, l.Radius) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) && l.Tooltip != "" {
			glob.TooltipText = l.Tooltip
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (l.holdingCursor || *isCursorAvailable) {
			l.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && l.holdingCursor {
		l.dragging = false
		l.holdingCursor = false
		*isCursorAvailable = true
		l.Center = rl.Vector2{
			X: utils.SnapToGrid(l.VirtualCenter.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(l.VirtualCenter.Y, config.SnapToGridInterval),
		}
		l.VirtualCenter = l.Center
		l.ClearForce()
	}
	if l.dragging {
		raw := rl.Vector2Add(worldMouse, l.offset)
		oldVC := l.VirtualCenter
		l.VirtualCenter = rl.Vector2Lerp(l.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		delta := l.VirtualCenter.Subtract(oldVC)
		l.InHook.Center = l.InHook.Center.Add(delta)
		l.ClearForce()
	} else {
		l.DecayForce()
		l.ApplyForce()
	}

	pullHookToAnchor(&l.Circle, l.InHook, l.anchorIn())
	l.InHook.Update(worldMouse, holdingCursor, isCursorAvailable)
}

func (l *Light) Destroy() {
	l.InHook.DisconnectAndKill()
	if parent := l.GetParent(); parent != nil {
		parent.DeleteChildWithID(l.ID)
	}
}

func (l *Light) Draw() {
	if l.InHook.IsHooked {
		target := utils.GetObjectFromID(l.InHook.TargetID)
		if target == nil {
			l.InHook.Disconnect()
		} else if t, ok := target.(Component); ok {
			// The wire leaves the light on the side facing the bit and points
			// at its center, no matter where the hook was left.
			edge := rl.Vector2Add(l.Center, rl.Vector2Scale(
				rl.Vector2Normalize(rl.Vector2Subtract(t.GetCircle().Center, l.Center)),
				l.Radius,
			))
			DrawHookLink(edge, t, 4, l.Color)
		}
	} else {
		edge := rl.Vector2Add(l.Center, rl.Vector2Scale(
			rl.Vector2Normalize(rl.Vector2Subtract(l.InHook.Center, l.Center)),
			l.Radius,
		))
		DrawWire(edge, l.InHook.Center, 4, l.Color)
	}
	l.InHook.Draw()

	if l.Lit() {
		// Glow: wide faded halo behind the bright core.
		rl.DrawCircleV(l.Center, l.Radius+16, rl.Fade(rl.Yellow, 0.15))
		rl.DrawCircleV(l.Center, l.Radius+8, rl.Fade(rl.Yellow, 0.3))
		rl.DrawCircleV(l.Center, l.Radius, rl.Yellow)
	} else {
		rl.DrawCircleV(l.Center, l.Radius, rl.Fade(config.ColorBg, 0.9))
	}
	borderColor := l.Color
	if l.hovered {
		borderColor = lighten(l.Color, 0.4)
	}
	rl.DrawCircleLines(int32(l.Center.X), int32(l.Center.Y), l.Radius, borderColor)
	if l.IsFixed {
		rl.DrawCircleLines(int32(l.Center.X), int32(l.Center.Y), l.Radius-1, borderColor)
	}
}

func (l *Light) DrawGhost() {
	ghostColor := rl.Fade(l.Color, 0.3)
	rl.DrawCircleLines(int32(l.Center.X), int32(l.Center.Y), l.Radius, ghostColor)
	edge := rl.Vector2Add(l.Center, rl.Vector2Scale(
		rl.Vector2Normalize(rl.Vector2Subtract(l.InHook.Center, l.Center)),
		l.Radius,
	))
	rl.DrawLineEx(edge, l.InHook.Center, 4, ghostColor)
	l.InHook.DrawGhost()
}

func (l *Light) GetChildCircles() []*Circle {
	if l.InHook == nil || l.InHook.IsHooked {
		return nil
	}
	return []*Circle{l.InHook.GetCircle()}
}

func (l *Light) PostUpdate() {}

// pullHookToAnchor springs a hook toward its anchored position relative to the
// owner body, dragging a hooked target along (CompareGate-style).
func pullHookToAnchor(owner *Circle, h *Hook, anchor rl.Vector2) {
	if !config.PhysicsEnabled {
		return
	}
	disp := anchor.Subtract(h.Center)
	dist := disp.Length()
	if dist <= glob.GateToHookGraceDist {
		return
	}
	val := float32(minFloat64(float64(dist), 100))
	dir := disp.Normalize().Scale(val * glob.GateToHookPullCoeff)

	if h.IsHooked {
		target := utils.GetObjectFromID(h.TargetID)
		if target == nil {
			h.Disconnect()
			return
		}
		t, ok := target.(Component)
		if !ok {
			h.Disconnect()
			return
		}
		t.AddForce(dir)
		owner.AddForce(dir.Scale(-1))
		return
	}
	h.AddForce(dir)
	owner.AddForce(dir.Scale(-1))
}

// drawHookWire draws the wire between a rectangular body and one of its hooks.
// When hooked, the wire leaves the body on the side facing the target and
// stops at the target's outline (DrawHookLink); the double-line style is used
// when the target is a logical carrier.
func drawHookWire(center rl.Vector2, h *Hook, halfW, halfH float32, color rl.Color) {
	if h.IsHooked {
		target := utils.GetObjectFromID(h.TargetID)
		if target == nil {
			h.Disconnect()
			return
		}
		t, ok := target.(Component)
		if !ok {
			h.Disconnect()
			return
		}
		edge := utils.RectEdgePoint(center, t.GetCircle().Center, halfW, halfH)
		DrawHookLink(edge, t, 4, color)
		return
	}
	edge := utils.RectEdgePoint(center, h.Center, halfW, halfH)
	DrawWire(edge, h.Center, 4, color)
}
