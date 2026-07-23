package components

import (
	"math/cmplx"
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	compareGateWidth  = 140
	compareGateHeight = 100
)

// CompareGate takes two QubitSystems (logical or not) and produces a single
// LogicalBit: 0 if their states are not equal, 1 if they are equal.
type CompareGate struct {
	Circle
	ID      int32
	Label   string
	Tooltip string
	Color   rl.Color
	Width   float32
	Height  float32

	InA     *Hook
	InB     *Hook
	OutHook *Hook

	// OutputID tracks the spawned LogicalBit so we don't create duplicates.
	OutputID int32
}

func NewCompareGate(x, y, radius float32, color rl.Color, label string) *CompareGate {
	w := float32(compareGateWidth)
	h := float32(compareGateHeight)
	cg := &CompareGate{
		Circle:  *NewCircle(x, y, maxF(w, h)/2, color),
		Label:   label,
		Tooltip: "Compare gate: outputs a logical bit (1 if inputs are equal, 0 otherwise)",
		Color:   color,
		Width:   w,
		Height:  h,
	}
	cg.ID = utils.GenerateID(cg)
	cg.SetWeight(glob.GateWeight)

	halfW := w / 2
	halfH := h / 2
	off := utils.SnapToGrid(glob.QubitSystemCellWidth/2, config.SnapToGridInterval)

	cg.InA = NewHook(x, y-halfH-off, glob.HookRadius, config.HookColor)
	cg.InA.Label = "A"
	cg.InA.AllowQubitSystem = true
	cg.InA.Tooltip = "Compare input A: connect a qubit system"

	cg.InB = NewHook(x, y+halfH+off, glob.HookRadius, config.HookColor)
	cg.InB.Label = "B"
	cg.InB.AllowQubitSystem = true
	cg.InB.Tooltip = "Compare input B: connect a qubit system"

	cg.OutHook = NewOutputHook(x+halfW+off, y, glob.OutputHookRadius, config.OutputHookColor)
	cg.OutHook.Label = "O"
	cg.OutHook.AllowLogicalBit = true
	cg.OutHook.Tooltip = "Compare output: logical bit (1 = equal, 0 = not equal)"

	return cg
}

func (cg *CompareGate) GetID() int32 { return cg.ID }

func (cg *CompareGate) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, rl.Rectangle{
		X:      cg.Center.X - cg.Width/2,
		Y:      cg.Center.Y - cg.Height/2,
		Width:  cg.Width,
		Height: cg.Height,
	})
}

func (cg *CompareGate) anchorA() rl.Vector2 {
	return rl.Vector2{X: cg.Center.X, Y: cg.Center.Y - cg.Height/2 - glob.QubitSystemCellWidth/2}
}

func (cg *CompareGate) anchorB() rl.Vector2 {
	return rl.Vector2{X: cg.Center.X, Y: cg.Center.Y + cg.Height/2 + glob.QubitSystemCellWidth/2}
}

func (cg *CompareGate) anchorOut() rl.Vector2 {
	return rl.Vector2{X: cg.Center.X + cg.Width/2 + glob.QubitSystemCellWidth/2, Y: cg.Center.Y}
}

func (cg *CompareGate) pullToHook(h *Hook, anchor rl.Vector2) {
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
		cg.AddForce(dir.Scale(-1))
		return
	}
	h.AddForce(dir)
	cg.AddForce(dir.Scale(-1))
}

func (cg *CompareGate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if cg.CheckCollide(worldMouse) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) && cg.Tooltip != "" {
			glob.TooltipText = cg.Tooltip
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (cg.holdingCursor || *isCursorAvailable) {
			cg.dragging = true
			*isCursorAvailable = false
			cg.holdingCursor = true
			cg.offset = rl.Vector2Subtract(cg.Center, worldMouse)
			cg.VirtualCenter = cg.Center
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
		cg.InA.Center = cg.InA.Center.Add(delta)
		cg.InB.Center = cg.InB.Center.Add(delta)
		cg.OutHook.Center = cg.OutHook.Center.Add(delta)
		cg.ClearForce()
	} else {
		cg.DecayForce()
		cg.ApplyForce()
	}

	cg.pullToHook(cg.InA, cg.anchorA())
	cg.pullToHook(cg.InB, cg.anchorB())
	cg.pullToHook(cg.OutHook, cg.anchorOut())

	cg.InA.Update(worldMouse, holdingCursor, isCursorAvailable)
	cg.InB.Update(worldMouse, holdingCursor, isCursorAvailable)
	cg.OutHook.Update(worldMouse, holdingCursor, isCursorAvailable)

	cg.updateOutput()
}

func (cg *CompareGate) updateOutput() {
	if !cg.InA.IsHooked || !cg.InB.IsHooked {
		return
	}
	aTarget := utils.GetObjectFromID(cg.InA.TargetID)
	bTarget := utils.GetObjectFromID(cg.InB.TargetID)
	qa, aOK := aTarget.(*QubitsSystem)
	qb, bOK := bTarget.(*QubitsSystem)
	if !aOK || !bOK || qa.Origin == nil || qb.Origin == nil {
		return
	}

	value := int32(0)
	if compareStatesEqual(qa.Origin, qb.Origin) {
		value = 1
	}

	if cg.OutHook.IsHooked {
		out := utils.GetObjectFromID(cg.OutHook.TargetID)
		if lb, ok := out.(*LogicalBit); ok {
			lb.SetValue(value)
			cg.OutputID = lb.ID
			return
		}
	}

	if !cg.OutHook.IsHooked {
		// The tracked bit was dragged onto another hook: forget it so a fresh
		// bit spawns on the output.
		cg.OutputID = 0
	}

	if cg.OutputID == 0 || utils.GetObjectFromID(cg.OutputID) == nil {
		tmp := NewLogicalBit(cg.OutHook.Center.X, cg.OutHook.Center.Y, glob.QubitSystemRadius, value)
		parent := cg.GetParent()
		if parent != nil {
			tmp.SetParent(parent)
			parent.PushComponent(tmp)
		}
		cg.OutHook.Connect(tmp)
		cg.OutputID = tmp.ID
	}
}

func (cg *CompareGate) Draw() {
	// Input A wire
	aEdge := utils.RectEdgePoint(cg.Center, cg.InA.Center, cg.Width/2, cg.Height/2)
	if cg.InA.IsHooked {
		target := utils.GetObjectFromID(cg.InA.TargetID)
		if target == nil {
			cg.InA.Disconnect()
		} else if t, ok := target.(Component); ok {
			end := rl.Vector2Add(aEdge, rl.Vector2Scale(
				rl.Vector2Normalize(rl.Vector2Subtract(cg.InA.Center, aEdge)),
				utils.Dist(aEdge, t.GetCircle().Center)-t.GetCircle().Radius,
			))
			DrawWire(aEdge, end, 4, config.HookColor)
		}
	} else {
		DrawWire(aEdge, cg.InA.Center, 4, config.HookColor)
	}
	cg.InA.Draw()

	// Input B wire
	bEdge := utils.RectEdgePoint(cg.Center, cg.InB.Center, cg.Width/2, cg.Height/2)
	if cg.InB.IsHooked {
		target := utils.GetObjectFromID(cg.InB.TargetID)
		if target == nil {
			cg.InB.Disconnect()
		} else if t, ok := target.(Component); ok {
			end := rl.Vector2Add(bEdge, rl.Vector2Scale(
				rl.Vector2Normalize(rl.Vector2Subtract(cg.InB.Center, bEdge)),
				utils.Dist(bEdge, t.GetCircle().Center)-t.GetCircle().Radius,
			))
			DrawWire(bEdge, end, 4, config.HookColor)
		}
	} else {
		DrawWire(bEdge, cg.InB.Center, 4, config.HookColor)
	}
	cg.InB.Draw()

	// Output wire (logical bit => double line)
	outEdge := utils.RectEdgePoint(cg.Center, cg.OutHook.Center, cg.Width/2, cg.Height/2)
	if cg.OutHook.IsHooked {
		target := utils.GetObjectFromID(cg.OutHook.TargetID)
		if target == nil {
			cg.OutHook.Disconnect()
		} else if t, ok := target.(Component); ok {
			end := rl.Vector2Add(outEdge, rl.Vector2Scale(
				rl.Vector2Normalize(rl.Vector2Subtract(cg.OutHook.Center, outEdge)),
				utils.Dist(outEdge, t.GetCircle().Center)-t.GetCircle().Radius,
			))
			DrawLogicalBitWire(outEdge, end, 4, cg.Color)
		}
	} else {
		DrawWire(outEdge, cg.OutHook.Center, 4, cg.Color)
	}
	cg.OutHook.Draw()

	rect := rl.Rectangle{
		X:      cg.Center.X - cg.Width/2,
		Y:      cg.Center.Y - cg.Height/2,
		Width:  cg.Width,
		Height: cg.Height,
	}
	rl.DrawRectangleRec(rect, config.ColorBg)
	thickness := float32(4.0)
	if cg.IsFixed {
		thickness = 5.0
	}
	rl.DrawRectangleLinesEx(rect, thickness, cg.Color)

	fontSize := int32(cg.Height / 2)
	if cg.Label != "" {
		textWidth := rl.MeasureText(cg.Label, fontSize)
		textX := int32(cg.Center.X) - textWidth/2
		textY := int32(cg.Center.Y) - fontSize/2
		rl.DrawText(cg.Label, textX, textY, fontSize, cg.Color)
	}
}

func (cg *CompareGate) DrawGhost() {
	ghostColor := rl.Fade(cg.Color, 0.3)
	rect := rl.Rectangle{
		X:      cg.Center.X - cg.Width/2,
		Y:      cg.Center.Y - cg.Height/2,
		Width:  cg.Width,
		Height: cg.Height,
	}
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)

	aEdge := utils.RectEdgePoint(cg.Center, cg.InA.Center, cg.Width/2, cg.Height/2)
	bEdge := utils.RectEdgePoint(cg.Center, cg.InB.Center, cg.Width/2, cg.Height/2)
	outEdge := utils.RectEdgePoint(cg.Center, cg.OutHook.Center, cg.Width/2, cg.Height/2)
	rl.DrawLineEx(aEdge, cg.InA.Center, 4, ghostColor)
	rl.DrawLineEx(bEdge, cg.InB.Center, 4, ghostColor)
	rl.DrawLineEx(outEdge, cg.OutHook.Center, 4, ghostColor)
	cg.InA.DrawGhost()
	cg.InB.DrawGhost()
	cg.OutHook.DrawGhost()
}

func (cg *CompareGate) GetChildCircles() []*Circle {
	var children []*Circle
	if cg.InA != nil && !cg.InA.IsHooked {
		children = append(children, cg.InA.GetCircle())
	}
	if cg.InB != nil && !cg.InB.IsHooked {
		children = append(children, cg.InB.GetCircle())
	}
	if cg.OutHook != nil && !cg.OutHook.IsHooked {
		children = append(children, cg.OutHook.GetCircle())
	}
	return children
}

// GetHooks exposes the input/output hooks to the zip helpers (hookOwner interface).
func (cg *CompareGate) GetHooks() []*Hook {
	return []*Hook{cg.InA, cg.InB, cg.OutHook}
}

func (cg *CompareGate) PostUpdate() {}

func compareStatesEqual(a, b *qubits.QubitStateManager) bool {
	if a.Size != b.Size || len(a.Amptitude) != len(b.Amptitude) {
		return false
	}
	if len(a.ModifierID) != len(b.ModifierID) {
		return false
	}
	for i, id := range a.ModifierID {
		if id != b.ModifierID[i] {
			return false
		}
	}
	eps := float64(1e-4)
	for i, av := range a.Amptitude {
		bv := b.Amptitude[i]
		if cmplx.Abs(complex128(av-bv)) > eps {
			return false
		}
	}
	return true
}
