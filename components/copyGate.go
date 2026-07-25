package components

import (
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	copyGateWidth  = 80
	copyGateHeight = 80
)

// CopyGate is a rectangular gate that takes a QubitSystem as input and emits a
// copy of it as a normal QubitSystem (determinators exposed). It behaves like
// a single-output gate but always mirrors the current input state into the
// output system.
type CopyGate struct {
	Circle
	ID      int32
	Label   string
	Tooltip string
	Color   rl.Color
	Width   float32
	Height  float32

	InHook  *Hook
	OutHook *Hook

	// CopyID tracks the last system produced so we don't respawn a copy every
	// frame while a previous one is still alive.
	CopyID int32
}

func NewCopyGate(x, y, radius float32, color rl.Color, label string) *CopyGate {
	w := float32(copyGateWidth)
	h := float32(copyGateHeight)
	cg := &CopyGate{
		Circle: *NewCircle(x, y, maxF(w, h)/2, color),
		Label:  label,
		Color:  color,
		Width:  w,
		Height: h,
	}
	cg.ID = utils.GenerateID(cg)
	cg.SetWeight(glob.GateWeight)
	cg.Tooltip = "Copy gate: creates a logical copy of the connected qubit system"

	halfW := w / 2
	halfH := h / 2
	off := utils.SnapToGrid(glob.QubitSystemCellWidth/2, config.SnapToGridInterval)

	cg.InHook = NewHook(x, y-halfH-off, glob.HookRadius, config.HookColor)
	cg.InHook.Label = "I"
	cg.InHook.AllowQubitSystem = true
	cg.InHook.Tooltip = "Copy input: connect a qubit system"

	cg.OutHook = NewOutputHook(x+halfW+off, y, glob.OutputHookRadius, config.OutputHookColor)
	cg.OutHook.Label = "O"
	cg.OutHook.AllowQubitSystem = true
	cg.OutHook.Tooltip = "Copy output: logical copy of the input"

	return cg
}

func (cg *CopyGate) GetID() int32 { return cg.ID }

func (cg *CopyGate) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, rl.Rectangle{
		X:      cg.Center.X - cg.Width/2,
		Y:      cg.Center.Y - cg.Height/2,
		Width:  cg.Width,
		Height: cg.Height,
	})
}

func (cg *CopyGate) inAnchor() rl.Vector2 {
	return rl.Vector2{X: cg.Center.X, Y: cg.Center.Y - cg.Height/2 - glob.QubitSystemCellWidth/2}
}

func (cg *CopyGate) outAnchor() rl.Vector2 {
	return rl.Vector2{X: cg.Center.X + cg.Width/2 + glob.QubitSystemCellWidth/2, Y: cg.Center.Y}
}

func (cg *CopyGate) pullToHook(h *Hook, anchor rl.Vector2) {
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

func (cg *CopyGate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// Drag the whole box.
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
		cg.InHook.Center = cg.InHook.Center.Add(delta)
		cg.OutHook.Center = cg.OutHook.Center.Add(delta)
		cg.ClearForce()
	} else {
		cg.DecayForce()
		cg.ApplyForce()
	}

	cg.pullToHook(cg.InHook, cg.inAnchor())
	cg.pullToHook(cg.OutHook, cg.outAnchor())

	cg.InHook.Update(worldMouse, holdingCursor, isCursorAvailable)
	cg.OutHook.Update(worldMouse, holdingCursor, isCursorAvailable)

	// Copy logic: mirror input state into the output system.
	if cg.InHook.IsHooked {
		target := utils.GetObjectFromID(cg.InHook.TargetID)
		qs, ok := target.(*QubitsSystem)
		if !ok || qs.Origin == nil {
			return
		}
		state := qubits.NewQubitStateManagerFrom([]complex64{}, qs.Origin.ModifierID)
		state.CopyFrom(qs.Origin)

		if cg.OutHook.IsHooked {
			outTarget := utils.GetObjectFromID(cg.OutHook.TargetID)
			if outQS, ok2 := outTarget.(*QubitsSystem); ok2 && outQS.Origin != nil {
				outQS.Origin.CopyFrom(state)
				cg.CopyID = outQS.ID
			}
		} else if cg.CopyID == 0 || utils.GetObjectFromID(cg.CopyID) == nil {
			// Spawn a fresh copy when there is no surviving copy.
			tmp := NewQubitsSystem(cg.OutHook.Center.X, cg.OutHook.Center.Y, glob.QubitSystemRadius, config.QubitSystemColor)
			tmp.IsLogical = true
			tmp.Assign(state)
			parent := cg.GetParent()
			if parent != nil {
				tmp.SetParent(parent)
				parent.PushComponent(tmp)
			}
			cg.OutHook.Connect(tmp)
			cg.CopyID = tmp.ID
		}
	} else {
		// Input removed: forget the tracked copy so a new one is produced on
		// the next connection. The existing copy remains independent.
		cg.CopyID = 0
	}
}

func (cg *CopyGate) Draw() {
	// Wire from box edge to the input hook.
	inEdge := utils.RectEdgePoint(cg.Center, cg.InHook.Center, cg.Width/2, cg.Height/2)
	if cg.InHook.IsHooked {
		target := utils.GetObjectFromID(cg.InHook.TargetID)
		if target == nil {
			cg.InHook.Disconnect()
		} else if t, ok := target.(Component); ok {
			inEdge := utils.RectEdgePoint(cg.Center, t.GetCircle().Center, cg.Width/2, cg.Height/2)
			DrawHookLink(inEdge, t, 4, config.HookColor)
		}
	} else {
		DrawWire(inEdge, cg.InHook.Center, 4, config.HookColor)
	}
	cg.InHook.Draw()

	// Wire from box edge to the output hook.
	outEdge := utils.RectEdgePoint(cg.Center, cg.OutHook.Center, cg.Width/2, cg.Height/2)
	if cg.OutHook.IsHooked {
		target := utils.GetObjectFromID(cg.OutHook.TargetID)
		if target == nil {
			cg.OutHook.Disconnect()
		} else if t, ok := target.(Component); ok {
			outEdge := utils.RectEdgePoint(cg.Center, t.GetCircle().Center, cg.Width/2, cg.Height/2)
			DrawHookLink(outEdge, t, 4, cg.Color)
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

func (cg *CopyGate) DrawGhost() {
	ghostColor := rl.Fade(cg.Color, 0.3)
	rect := rl.Rectangle{
		X:      cg.Center.X - cg.Width/2,
		Y:      cg.Center.Y - cg.Height/2,
		Width:  cg.Width,
		Height: cg.Height,
	}
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)

	inEdge := utils.RectEdgePoint(cg.Center, cg.InHook.Center, cg.Width/2, cg.Height/2)
	outEdge := utils.RectEdgePoint(cg.Center, cg.OutHook.Center, cg.Width/2, cg.Height/2)
	rl.DrawLineEx(inEdge, cg.InHook.Center, 4, ghostColor)
	rl.DrawLineEx(outEdge, cg.OutHook.Center, 4, ghostColor)
	cg.InHook.DrawGhost()
	cg.OutHook.DrawGhost()
}

func (cg *CopyGate) GetChildCircles() []*Circle {
	var children []*Circle
	if cg.InHook != nil && !cg.InHook.IsHooked {
		children = append(children, cg.InHook.GetCircle())
	}
	if cg.OutHook != nil && !cg.OutHook.IsHooked {
		children = append(children, cg.OutHook.GetCircle())
	}
	return children
}

func (cg *CopyGate) PostUpdate() {}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
