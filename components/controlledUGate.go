package components

import (
	"fmt"
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/unitary"
	"qsim/utils"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	controlledUGateWidth  = 120
	controlledUGateHeight = 90
)

// ControlledUGate is a universal gate whose control is a LogicalBit instead
// of a qubit: the edited matrix is applied only while the control bit is 1,
// otherwise the inputs pass through unchanged. It has k qubit inputs (I0..),
// one logical bit control input (C, defaults to 0 when unhooked), and one
// qubit-system output. The output system has the same size as the input
// union: the control bit is classical, so it never expands the state.
type ControlledUGate struct {
	Circle
	ID      int32
	Label   string
	Tooltip string
	Color   rl.Color
	Width   float32
	Height  float32

	InputCount int32
	Operation  [][]complex64
	Editable   bool

	QubitHooks []*Hook
	InControl  *Hook
	OutHook    *Hook

	// inline matrix editing state (see arbGate.go)
	editing     bool
	editErr     string
	cursorBlink float32
	cursorShow  bool
	hovered     bool
	renaming    bool
	renameStr   string
	editingCell bool
	cellRow     int32
	cellCol     int32
	cellBuffer  string
	sizeEditing bool
	sizeStr     string
	notice      string
	noticeTimer float32
}

func NewControlledUGate(x, y float32, color rl.Color) *ControlledUGate {
	w := float32(controlledUGateWidth)
	h := float32(controlledUGateHeight)
	cg := &ControlledUGate{
		Circle:     *NewCircle(x, y, maxF(w, h)/2, color),
		Label:      "cU",
		Color:      color,
		Width:      w,
		Height:     h,
		InputCount: 1,
		Operation:  [][]complex64{{1, 0}, {0, 1}},
		Editable:   true,
		Tooltip:    "Bit-controlled universal gate: applies the operation only while the control bit is 1; right-click to edit the matrix",
	}
	cg.ID = utils.GenerateID(cg)
	cg.SetWeight(glob.GateWeight)

	dist := cg.hookDist()
	for i := 0; i < int(cg.InputCount); i++ {
		hook := NewHook(x-dist, y+cg.qubitOffset(int32(i)), glob.HookRadius, config.HookColor)
		hook.Label = "I" + strconv.Itoa(i)
		hook.Tooltip = "CU input: plug a qubit determinator"
		cg.QubitHooks = append(cg.QubitHooks, hook)
	}

	cg.InControl = NewLogicalHook(x-dist, y+cg.controlOffset())
	cg.InControl.Label = "C"
	cg.InControl.Tooltip = "Control input: connect a logical bit (defaults to 0)"

	cg.OutHook = NewOutputHook(x+dist, y, glob.OutputHookRadius, config.OutputHookColor)
	cg.OutHook.Label = "O"
	cg.OutHook.AllowQubitSystem = true
	cg.OutHook.Tooltip = "CU output: the transformed qubit system (same size as the inputs)"
	return cg
}

func (c *ControlledUGate) GetID() int32 { return c.ID }

// GetHooks exposes the hooks to the zip helpers (hookOwner interface), in
// serialization order: qubit inputs I0.., then the control, then the output.
func (c *ControlledUGate) GetHooks() []*Hook {
	hooks := make([]*Hook, 0, len(c.QubitHooks)+2)
	hooks = append(hooks, c.QubitHooks...)
	hooks = append(hooks, c.InControl, c.OutHook)
	return hooks
}

// qubitOffset is the vertical offset of input i relative to the body center:
// the k inputs stack in 50px steps centered on the body.
func (c *ControlledUGate) qubitOffset(i int32) float32 {
	return (float32(i) - float32(c.InputCount-1)/2) * 50
}

// controlOffset places the control hook one 50px step below the input stack.
func (c *ControlledUGate) controlOffset() float32 {
	return (float32(c.InputCount) + 1) / 2 * 50
}

// hookDist is the horizontal distance from the body center to the hooks
// (SnapToGrid(GateToHookDist) = 300px).
func (c *ControlledUGate) hookDist() float32 {
	return utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)
}

func (c *ControlledUGate) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, rl.Rectangle{
		X:      c.Center.X - c.Width/2,
		Y:      c.Center.Y - c.Height/2,
		Width:  c.Width,
		Height: c.Height,
	})
}

func (c *ControlledUGate) anchorQubit(i int32) rl.Vector2 {
	return rl.Vector2{X: c.Center.X - c.hookDist(), Y: c.Center.Y + c.qubitOffset(i)}
}

func (c *ControlledUGate) anchorControl() rl.Vector2 {
	return rl.Vector2{X: c.Center.X - c.hookDist(), Y: c.Center.Y + c.controlOffset()}
}

func (c *ControlledUGate) anchorOut() rl.Vector2 {
	return rl.Vector2{X: c.Center.X + c.hookDist(), Y: c.Center.Y}
}

func (c *ControlledUGate) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		c.IsFixed = !c.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		c.DestroyOutPut()
		for _, h := range c.QubitHooks {
			h.Disconnect()
		}
		c.InControl.Disconnect()
	case utils.IsMouseState(glob.MouseStateErase):
		c.Destroy()
	default:
		// The most recently clicked gate is the Duplicate button's source.
		LastSelectedGate = c
		c.dragging = true
		*isCursorAvailable = false
		c.holdingCursor = true
		c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		c.VirtualCenter = c.Center
	}
}

func (c *ControlledUGate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if c.noticeTimer > 0 {
		c.noticeTimer -= rl.GetFrameTime()
		if c.noticeTimer <= 0 {
			c.notice = ""
		}
	}
	if c.editing {
		c.processEditing(worldMouse, isCursorAvailable)
		return
	}
	if c.renaming {
		c.processRename(isCursorAvailable)
		return
	}
	c.hovered = c.CheckCollide(worldMouse) && !c.dragging
	if c.CheckCollide(worldMouse) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) && c.Tooltip != "" {
			glob.TooltipText = c.Tooltip
		}
		if c.Editable && utils.IsMouseState(glob.MouseStateNormal) &&
			rl.IsMouseButtonPressed(rl.MouseButtonRight) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			// right-click: rename the gate inline (like a universal gate)
			c.renaming = true
			c.renameStr = c.Label
			c.holdingCursor = true
			*isCursorAvailable = false
			return
		}
		if c.Editable && utils.IsMouseState(glob.MouseStateNormal) &&
			rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) &&
			c.DoubleClicked(worldMouse) {
			// double-click: edit the qubit count and the matrix inline
			c.editing = true
			c.editErr = ""
			c.holdingCursor = true
			*isCursorAvailable = false
			return
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			c.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		c.dragging = false
		c.holdingCursor = false
		*isCursorAvailable = true
		c.Center = rl.Vector2{
			X: utils.SnapToGrid(c.VirtualCenter.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(c.VirtualCenter.Y, config.SnapToGridInterval),
		}
		c.VirtualCenter = c.Center
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
		for _, h := range c.GetHooks() {
			h.Center = h.Center.Add(delta)
		}
		c.ClearForce()
	} else {
		c.DecayForce()
		c.ApplyForce()
	}

	for i := range c.QubitHooks {
		pullHookToAnchor(&c.Circle, c.QubitHooks[i], c.anchorQubit(int32(i)))
	}
	pullHookToAnchor(&c.Circle, c.InControl, c.anchorControl())
	pullHookToAnchor(&c.Circle, c.OutHook, c.anchorOut())

	for _, h := range c.GetHooks() {
		h.Update(worldMouse, holdingCursor, isCursorAvailable)
	}

	c.updateOutput()
}

// controlValue reads the control bit; an unhooked control counts as 0.
func (c *ControlledUGate) controlValue() int32 {
	if bit := inputBit(c.InControl); bit != nil {
		return bit.Value
	}
	return 0
}

// updateOutput merges the hooked input systems in hook order (each system
// once), re-orders them to the matrix's basis order, and applies the edited
// operation only while the control bit is 1. While the bit is 0 the merged
// state passes through unchanged: no qubits are added or removed.
func (c *ControlledUGate) updateOutput() {
	for _, h := range c.QubitHooks {
		if !h.IsHooked {
			c.DestroyOutPut()
			return
		}
	}

	QSM := []*qubits.QubitStateManager{}
	Idx := []int32{}
	prob := 1.0
	IsIN := func(id int32) bool {
		for _, d := range QSM {
			if d.ID == id {
				return true
			}
		}
		return false
	}
	for _, d := range c.QubitHooks {
		QD, ok := utils.GetObjectFromID(d.TargetID).(*QubitDeterminator)
		if !ok {
			return
		}
		qidObject := QD.GetQubitParent()
		if qidObject == nil {
			return
		}
		qid := qidObject.Origin.ID
		if IsIN(qid) {
			Idx = append(Idx, QD.ModifierID)
		} else {
			prob *= qidObject.Probability
			tmp := *QD.GetQubitParent().Origin
			QSM = append(QSM, &tmp)
			Idx = append(Idx, QD.ModifierID)
		}
	}

	result := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	for i := range QSM {
		result.Merge(QSM[i])
	}
	for i := range Idx {
		result.SwapColumn(int32(i), result.FindID(Idx[i]))
	}
	if c.controlValue() == 1 {
		result.Multiply(c.Operation, c.InputCount)
	}

	if c.OutHook.IsHooked {
		tmp := utils.GetObjectFromID(c.OutHook.TargetID)
		if qs, ok := tmp.(*QubitsSystem); ok && qs.Origin != nil {
			qs.CopyFromState(result)
			qs.Probability = prob
			return
		}
	}

	tmp := NewQubitsSystem(c.Center.X, c.Center.Y, glob.QubitSystemRadius, config.QubitSystemColor)
	tmp.Probability = prob
	tmp.Assign(result)
	parent := c.GetParent()
	if parent != nil {
		tmp.SetParent(parent)
		parent.PushComponent(tmp)
	}
	c.OutHook.Connect(tmp)
	tmp.ZipDeterminatorsToHooks()
}

func (c *ControlledUGate) DestroyOutPut() {
	c.OutHook.DisconnectAndKill()
}

func (c *ControlledUGate) Destroy() {
	c.DestroyOutPut()
	for _, h := range c.QubitHooks {
		h.Disconnect()
	}
	c.InControl.Disconnect()
	if parent := c.GetParent(); parent != nil {
		parent.DeleteChildWithID(c.ID)
	}
}

func (c *ControlledUGate) Draw() {
	for _, h := range c.QubitHooks {
		drawHookWire(c.Center, h, c.Width/2, c.Height/2, c.Color)
	}
	drawHookWire(c.Center, c.InControl, c.Width/2, c.Height/2, c.Color)
	drawHookWire(c.Center, c.OutHook, c.Width/2, c.Height/2, c.Color)
	for _, h := range c.GetHooks() {
		h.Draw()
	}

	rect := rl.Rectangle{
		X:      c.Center.X - c.Width/2,
		Y:      c.Center.Y - c.Height/2,
		Width:  c.Width,
		Height: c.Height,
	}
	rl.DrawRectangleRounded(rect, 0.15, 6, config.ColorBg)
	thickness := float32(4.0)
	if c.IsFixed {
		thickness = 5.0
	}
	borderColor := c.Color
	if c.hovered {
		borderColor = lighten(c.Color, 0.4)
	}
	rl.DrawRectangleRoundedLinesEx(rect, 0.15, 6, thickness, borderColor)

	// Control dot: fills while the control bit is 1.
	dot := rl.Vector2{X: rect.X + 12, Y: c.Center.Y + c.Height/4}
	if c.controlValue() == 1 {
		rl.DrawCircleV(dot, 6, rl.Yellow)
	} else {
		rl.DrawCircleLines(int32(dot.X), int32(dot.Y), 6, rl.Fade(borderColor, 0.7))
	}

	if c.Label != "" {
		fontSize := int32(c.Height / 3)
		textWidth := rl.MeasureText(c.Label, fontSize)
		rl.DrawText(c.Label, int32(c.Center.X)-textWidth/2, int32(c.Center.Y)-fontSize/2, fontSize, borderColor)
	}

	if c.Editable && c.hovered && !c.editing && !c.renaming {
		c.DrawValuePanel()
	}

	if c.renaming {
		fontSize := int32(16)
		label := "Name: " + c.renameStr
		w := rl.MeasureText(label, fontSize) + 12
		if w < 120 {
			w = 120
		}
		hint := "Enter: apply + edit matrix   Esc: cancel"
		hintW := rl.MeasureText(hint, 10) + 12
		if hintW > w {
			w = hintW
		}
		box := rl.NewRectangle(c.Center.X-float32(w)/2, c.Center.Y-c.Height/2-44, float32(w), 40)
		rl.DrawRectangleRounded(box, 0.2, 4, rl.NewColor(30, 30, 30, 255))
		rl.DrawRectangleRoundedLinesEx(box, 0.2, 4, 1.5, rl.SkyBlue)
		rl.DrawText(hint, int32(box.X+6), int32(box.Y+3), 10, rl.Gray)
		rl.DrawText(label, int32(box.X+6), int32(box.Y+18), fontSize, rl.White)
		if c.cursorShow {
			tw := rl.MeasureText(label, fontSize)
			rl.DrawText("|", int32(box.X+6)+tw, int32(box.Y+18), fontSize, rl.SkyBlue)
		}
	}

	if c.editing {
		c.DrawEditPanel()
	}

	if c.notice != "" {
		fontSize := int32(12)
		tw := rl.MeasureText(c.notice, fontSize)
		nx := c.Center.X - float32(tw)/2 - 8
		ny := c.Center.Y - c.Height/2 - 26
		rl.DrawRectangleRec(rl.Rectangle{X: nx, Y: ny, Width: float32(tw) + 16, Height: 22}, rl.NewColor(30, 30, 30, 235))
		rl.DrawRectangleLinesEx(rl.Rectangle{X: nx, Y: ny, Width: float32(tw) + 16, Height: 22}, 1.5, rl.Lime)
		rl.DrawText(c.notice, int32(c.Center.X)-tw/2, int32(ny+5), fontSize, rl.Lime)
	}
}

func (c *ControlledUGate) DrawGhost() {
	ghostColor := rl.Fade(c.Color, 0.3)
	rect := rl.Rectangle{
		X:      c.Center.X - c.Width/2,
		Y:      c.Center.Y - c.Height/2,
		Width:  c.Width,
		Height: c.Height,
	}
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)
	for _, h := range c.GetHooks() {
		edge := utils.RectEdgePoint(c.Center, h.Center, c.Width/2, c.Height/2)
		rl.DrawLineEx(edge, h.Center, 4, ghostColor)
		h.DrawGhost()
	}
}

func (c *ControlledUGate) GetChildCircles() []*Circle {
	var children []*Circle
	for _, h := range c.GetHooks() {
		if h != nil && !h.IsHooked {
			children = append(children, h.GetCircle())
		}
	}
	return children
}

func (c *ControlledUGate) PostUpdate() {}

// processEditing handles the inline table editor: the same name-and-matrix
// table shown on hover, now interactive. Click a cell to type its value
// ("0.7" or "0.7,0.3"), use the +/- buttons to change the qubit count;
// Enter closes the editor and auto-applies the matrix (raylib's default exit
// key is Escape, so it must not be the way out of the editor).
func (c *ControlledUGate) processEditing(worldMouse rl.Vector2, isCursorAvailable *bool) {
	c.cursorBlink += rl.GetFrameTime()
	if c.cursorBlink > 0.5 {
		c.cursorShow = !c.cursorShow
		c.cursorBlink = 0
	}

	if c.editingCell {
		key := rl.GetCharPressed()
		for key > 0 {
			if strings.ContainsRune("0123456789.,-+eEi ", key) && len(c.cellBuffer) < 32 {
				c.cellBuffer += string(key)
			}
			key = rl.GetCharPressed()
		}
		if rl.IsKeyPressed(rl.KeyBackspace) && len(c.cellBuffer) > 0 {
			c.cellBuffer = c.cellBuffer[:len(c.cellBuffer)-1]
		}
		// Arrow keys commit the current value and move to the neighbour cell.
		if rl.IsKeyPressed(rl.KeyUp) || rl.IsKeyPressed(rl.KeyDown) ||
			rl.IsKeyPressed(rl.KeyLeft) || rl.IsKeyPressed(rl.KeyRight) {
			if v, err := parseMatrixEntry(c.cellBuffer); err != nil {
				c.editErr = err.Error()
				return
			} else {
				c.Operation[c.cellRow][c.cellCol] = v
				c.editErr = ""
				c.notice = ""
			}
			size := int32(1) << c.InputCount
			switch {
			case rl.IsKeyPressed(rl.KeyUp) && c.cellRow > 0:
				c.cellRow--
			case rl.IsKeyPressed(rl.KeyDown) && c.cellRow < size-1:
				c.cellRow++
			case rl.IsKeyPressed(rl.KeyLeft) && c.cellCol > 0:
				c.cellCol--
			case rl.IsKeyPressed(rl.KeyRight) && c.cellCol < size-1:
				c.cellCol++
			}
			c.cellBuffer = formatCellValue(c.Operation[c.cellRow][c.cellCol])
			return
		}
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
			v, err := parseMatrixEntry(c.cellBuffer)
			if err != nil {
				c.editErr = err.Error()
				return
			}
			c.Operation[c.cellRow][c.cellCol] = v
			c.editErr = ""
			c.notice = ""
			// Enter closes the editor: auto-apply the matrix (projecting to
			// the nearest unitary when needed) and leave edit mode.
			c.finalizeEdit()
			c.editing = false
			c.holdingCursor = false
			*isCursorAvailable = true
			return
		}
		if rl.IsKeyPressed(rl.KeyEscape) {
			c.editingCell = false
			return
		}
		// Clicking a different cell commits the current edit and moves the
		// editor there; Escape cancels the pending edit.
		// Arrow keys move the cell selection; typing a value character starts
		// editing the selected cell.
		size := int32(1) << c.InputCount
		if c.cellRow >= size {
			c.cellRow = size - 1
		}
		if c.cellCol >= size {
			c.cellCol = size - 1
		}
		selKey := rl.GetCharPressed()
		for selKey > 0 {
			if strings.ContainsRune("0123456789.,-+eEi ", selKey) {
				c.cellBuffer = string(rune(selKey))
				c.editingCell = true
			}
			selKey = rl.GetCharPressed()
		}
		if rl.IsKeyPressed(rl.KeyUp) && c.cellRow > 0 {
			c.cellRow--
			c.cellBuffer = formatCellValue(c.Operation[c.cellRow][c.cellCol])
			return
		}
		if rl.IsKeyPressed(rl.KeyDown) && c.cellRow < size-1 {
			c.cellRow++
			c.cellBuffer = formatCellValue(c.Operation[c.cellRow][c.cellCol])
			return
		}
		if rl.IsKeyPressed(rl.KeyLeft) && c.cellCol > 0 {
			c.cellCol--
			c.cellBuffer = formatCellValue(c.Operation[c.cellRow][c.cellCol])
			return
		}
		if rl.IsKeyPressed(rl.KeyRight) && c.cellCol < size-1 {
			c.cellCol++
			c.cellBuffer = formatCellValue(c.Operation[c.cellRow][c.cellCol])
			return
		}

		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && (c.holdingCursor || *isCursorAvailable) {
			x, _, _, _, cell, gridY, _ := c.editPanelGeom()
			size := int32(1) << c.InputCount
			gridX := x + 6
			if worldMouse.Y >= gridY && worldMouse.Y < gridY+float32(size)*cell &&
				worldMouse.X >= gridX && worldMouse.X < gridX+float32(size)*cell {
				row := int32((worldMouse.Y - gridY) / cell)
				col := int32((worldMouse.X - gridX) / cell)
				if row >= 0 && row < size && col >= 0 && col < size &&
					(row != c.cellRow || col != c.cellCol) {
					v, err := parseMatrixEntry(c.cellBuffer)
					if err != nil {
						c.editErr = err.Error()
						return
					}
					c.Operation[c.cellRow][c.cellCol] = v
					c.editErr = ""
					c.notice = ""
					c.cellRow, c.cellCol = row, col
					c.cellBuffer = formatCellValue(c.Operation[row][col])
				}
			}
		}
		return
	}

	x, y, w, _, cell, gridY, _ := c.editPanelGeom()
	size := int32(1) << c.InputCount
	sizeRowY := y + 4 + 18 + 4
	sizeField := rl.Rectangle{X: x + 8 + float32(rl.MeasureText("qubits: ", 14)), Y: sizeRowY - 3, Width: 28, Height: 20}
	minusRect := rl.Rectangle{X: x + w - 54, Y: sizeRowY, Width: 22, Height: 20}
	plusRect := rl.Rectangle{X: x + w - 28, Y: sizeRowY, Width: 22, Height: 20}

	// The size field is clickable while editing: type 1-4 and press Enter to
	// change the qubit count directly (Enter also closes the editor).
	if c.sizeEditing {
		key := rl.GetCharPressed()
		for key > 0 {
			if key >= '0' && key <= '9' && len(c.sizeStr) < 3 {
				c.sizeStr += string(rune(key))
			}
			key = rl.GetCharPressed()
		}
		if rl.IsKeyPressed(rl.KeyBackspace) && len(c.sizeStr) > 0 {
			c.sizeStr = c.sizeStr[:len(c.sizeStr)-1]
		}
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
			if v, err := strconv.Atoi(c.sizeStr); err == nil && v >= 1 && v <= maxGateQubits {
				c.reconfigureOperation(resizeGateOperation(c.Operation, int32(v)), int32(v))
				c.notice = ""
			}
			c.sizeEditing = false
		}
		if rl.IsKeyPressed(rl.KeyEscape) {
			c.sizeEditing = false
		}
		return // typing in the size field: ignore mouse until done
	}

	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && (c.holdingCursor || *isCursorAvailable) {
		switch {
		case rl.CheckCollisionPointRec(worldMouse, plusRect) && c.InputCount < maxGateQubits:
			c.reconfigureOperation(resizeGateOperation(c.Operation, c.InputCount+1), c.InputCount+1)
			c.notice = ""
			return
		case rl.CheckCollisionPointRec(worldMouse, minusRect) && c.InputCount > 1:
			c.reconfigureOperation(resizeGateOperation(c.Operation, c.InputCount-1), c.InputCount-1)
			if !unitary.IsUnitary(c.Operation, unitary.Tolerance) {
				c.Operation = unitary.NearestUnitary(c.Operation)
				c.notice = "shrunk matrix projected to nearest unitary"
				c.noticeTimer = 4
			}
			return
		case rl.CheckCollisionPointRec(worldMouse, sizeField):
			c.sizeEditing = true
			c.sizeStr = strconv.Itoa(int(c.InputCount))
			return
		}
		gridX := x + 6
		if worldMouse.Y >= gridY && worldMouse.Y < gridY+float32(size)*cell &&
			worldMouse.X >= gridX && worldMouse.X < gridX+float32(size)*cell {
			row := int32((worldMouse.Y - gridY) / cell)
			col := int32((worldMouse.X - gridX) / cell)
			if row >= 0 && row < size && col >= 0 && col < size {
				c.cellRow, c.cellCol = row, col
				c.cellBuffer = formatCellValue(c.Operation[row][col])
				c.editingCell = true
			}
		}
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		c.finalizeEdit()
		c.editing = false
		c.holdingCursor = false
		*isCursorAvailable = true
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		c.finalizeEdit()
		c.editing = false
		c.holdingCursor = false
		*isCursorAvailable = true
		return
	}
}

// finalizeEdit auto-applies the edited matrix on editor close: if it is not
// unitary within tolerance it is projected to the nearest unitary and a
// transient notice is queued.
func (c *ControlledUGate) finalizeEdit() {
	if !unitary.IsUnitary(c.Operation, unitary.Tolerance) {
		dev := unitary.Error(c.Operation)
		c.Operation = unitary.NearestUnitary(c.Operation)
		c.notice = fmt.Sprintf("projected to nearest unitary (dev was %.3g)", dev)
		c.noticeTimer = 4
	}
}

// editPanelGeom returns the geometry shared by processEditing and
// DrawEditPanel: the panel rect, the cell size, the grid font, the grid
// top-left Y and the panel height.
func (c *ControlledUGate) editPanelGeom() (x, y, w, h, cell, gridY float32, font int32) {
	size := int32(1) << c.InputCount
	cell, font = gateCellSize(size)
	gridW := float32(size) * cell
	w = gridW + 12
	if labelW := float32(rl.MeasureText(c.Label, 18)) + 12; labelW > w {
		w = labelW
	}
	nameTop := float32(4)
	nameH := float32(18)
	sizeTop := nameTop + nameH + 4
	sizeH := float32(22)
	x = c.Center.X - w/2
	y = c.Center.Y - c.Height/2 - (sizeTop + sizeH + gridW + 6) - 8
	gridY = y + sizeTop + sizeH
	h = gridY - y + gridW + 6
	return
}

// DrawEditPanel draws the right-click edit table: the name on top, the
// qubit-count +/- buttons, then the operation matrix as the same grid used by
// the hover preview — the clicked cell is edited inline.
func (c *ControlledUGate) DrawEditPanel() {
	if c.InputCount < 1 {
		return
	}
	x, y, w, h, cell, gridY, font := c.editPanelGeom()
	size := int32(1) << c.InputCount
	gridX := x + 6

	rl.DrawRectangleRec(rl.Rectangle{X: x, Y: y, Width: w, Height: h}, rl.NewColor(30, 30, 30, 235))
	borderColor := c.Color
	if !unitary.IsUnitary(c.Operation, unitary.Tolerance) {
		borderColor = rl.Orange
	}
	rl.DrawRectangleLinesEx(rl.Rectangle{X: x, Y: y, Width: w, Height: h}, 1.5, borderColor)

	labelW := rl.MeasureText(c.Label, 18)
	rl.DrawText(c.Label, int32(x+(w-float32(labelW))/2), int32(y+4), 18, c.Color)

	sizeRowY := y + 4 + 18 + 4
	rl.DrawText("qubits: ", int32(x+8), int32(sizeRowY), 14, rl.White)

	// Clickable size field: type 1-4 directly (see processEditing).
	sizeField := rl.Rectangle{X: x + 8 + float32(rl.MeasureText("qubits: ", 14)), Y: sizeRowY - 3, Width: 28, Height: 20}
	rl.DrawRectangleRec(sizeField, rl.NewColor(45, 45, 45, 255))
	rl.DrawRectangleLinesEx(sizeField, 1.5, rl.SkyBlue)
	snum := strconv.Itoa(int(c.InputCount))
	if c.sizeEditing {
		snum = c.sizeStr
	}
	sw := rl.MeasureText(snum, 14)
	rl.DrawText(snum, int32(sizeField.X)+int32(sizeField.Width)/2-sw/2, int32(sizeField.Y)+3, 14, rl.White)
	if c.sizeEditing && c.cursorShow {
		rl.DrawText("|", int32(sizeField.X)+int32(sizeField.Width)/2+sw/2+1, int32(sizeField.Y)+3, 14, rl.SkyBlue)
	}

	minusRect := rl.Rectangle{X: x + w - 54, Y: sizeRowY, Width: 22, Height: 20}
	plusRect := rl.Rectangle{X: x + w - 28, Y: sizeRowY, Width: 22, Height: 20}
	for btnName, btnRect := range map[string]rl.Rectangle{"-": minusRect, "+": plusRect} {
		rl.DrawRectangleRec(btnRect, rl.NewColor(60, 60, 60, 255))
		rl.DrawRectangleLinesEx(btnRect, 1.5, rl.SkyBlue)
		bw := rl.MeasureText(btnName, 16)
		rl.DrawText(btnName, int32(btnRect.X)+int32(btnRect.Width)/2-bw/2, int32(btnRect.Y)+2, 16, rl.White)
	}

	for r := int32(0); r < size; r++ {
		for col := int32(0); col < size; col++ {
			cx := gridX + float32(col)*cell + cell/2
			cy := gridY + float32(r)*cell + cell/2
			entry := formatGateEntry(c.Operation[r][col])
			if c.editingCell && r == c.cellRow && col == c.cellCol {
				entry = c.cellBuffer
				rl.DrawRectangleRec(rl.Rectangle{X: gridX + float32(col)*cell, Y: gridY + float32(r)*cell, Width: cell, Height: cell}, rl.NewColor(50, 60, 90, 255))
				tw := rl.MeasureText(entry, font)
				rl.DrawText(entry, int32(cx)-tw/2, int32(cy)-font/2, font, rl.White)
				if c.cursorShow {
					rl.DrawText("|", int32(cx)+tw/2, int32(cy)-font/2, font, rl.SkyBlue)
				}
				continue
			}
			tw := rl.MeasureText(entry, font)
			rl.DrawText(entry, int32(cx)-tw/2, int32(cy)-font/2, font, rl.White)
		}
	}

	// Grid lines for readability
	for i := int32(0); i <= size; i++ {
		gx := gridX + float32(i)*cell
		gy := gridY
		rl.DrawLineV(rl.Vector2{X: gx, Y: gy}, rl.Vector2{X: gx, Y: gy + float32(size)*cell}, rl.Gray)
		rl.DrawLineV(rl.Vector2{X: gridX, Y: gy + float32(i)*cell}, rl.Vector2{X: gridX + float32(size)*cell, Y: gy + float32(i)*cell}, rl.Gray)
	}

	// Selected cell outline (arrow-key navigation target).
	if !c.editingCell && c.cellRow >= 0 && c.cellRow < size && c.cellCol >= 0 && c.cellCol < size {
		rl.DrawRectangleLinesEx(rl.Rectangle{
			X: gridX + float32(c.cellCol)*cell, Y: gridY + float32(c.cellRow)*cell,
			Width: cell, Height: cell,
		}, 2, rl.SkyBlue)
	}

	if c.editErr != "" {
		rl.DrawText(c.editErr, int32(x+6), int32(y+h-18), 12, rl.Orange)
	}
}

// DrawValuePanel draws the hover info box above the gate: the name on top and
// the operation matrix as a grid of cells below, scaled to the space
// available.
func (c *ControlledUGate) DrawValuePanel() {
	if c.InputCount < 1 {
		return
	}
	size := int32(1) << c.InputCount

	// Choose a cell size that keeps the whole grid readable; bigger matrices
	// get smaller cells.
	var cell float32
	var font int32
	switch size {
	case 2:
		cell, font = 54, 20
	case 4:
		cell, font = 44, 16
	case 8:
		cell, font = 30, 12
	default:
		cell, font = 22, 9
	}

	label := c.Label
	fontSize := int32(20)
	labelW := rl.MeasureText(label, fontSize)
	panelW := float32(size)*cell + 12
	panelH := float32(size)*cell + 4 + float32(fontSize) + 8
	if float32(labelW)+12 > panelW {
		panelW = float32(labelW) + 12
	}

	x := c.Center.X - panelW/2
	y := c.Center.Y - c.Height/2 - panelH - 8

	rl.DrawRectangleRec(rl.Rectangle{X: x, Y: y, Width: panelW, Height: panelH}, rl.NewColor(30, 30, 30, 235))
	rl.DrawRectangleLinesEx(rl.Rectangle{X: x, Y: y, Width: panelW, Height: panelH}, 1.5, c.Color)

	rl.DrawText(label, int32(x+(panelW-float32(labelW))/2), int32(y+4), fontSize, c.Color)

	for r := int32(0); r < size; r++ {
		for col := int32(0); col < size; col++ {
			cx := x + 6 + float32(col)*cell + cell/2
			cy := y + 6 + float32(fontSize) + 4 + float32(r)*cell + cell/2
			entry := formatGateEntry(c.Operation[r][col])
			tw := rl.MeasureText(entry, font)
			rl.DrawText(entry, int32(cx)-tw/2, int32(cy)-font/2, font, rl.White)
		}
	}

	// Grid lines for readability
	for i := int32(0); i <= size; i++ {
		gx := x + 6 + float32(i)*cell
		gy := y + 6 + float32(fontSize) + 4
		rl.DrawLineV(rl.Vector2{X: gx, Y: gy}, rl.Vector2{X: gx, Y: gy + float32(size)*cell}, rl.Gray)
		rl.DrawLineV(rl.Vector2{X: x + 6, Y: gy + float32(i)*cell}, rl.Vector2{X: x + 6 + float32(size)*cell, Y: gy + float32(i)*cell}, rl.Gray)
	}
}

// processRename handles the inline rename editor (right-click on the gate),
// mirroring the universal gate's rename: type to edit, Enter applies and
// opens the size/matrix editor, Escape cancels.
func (c *ControlledUGate) processRename(isCursorAvailable *bool) {
	c.cursorBlink += rl.GetFrameTime()
	if c.cursorBlink > 0.5 {
		c.cursorShow = !c.cursorShow
		c.cursorBlink = 0
	}

	key := rl.GetCharPressed()
	for key > 0 {
		if key >= 32 && key <= 125 && len(c.renameStr) < 24 {
			c.renameStr += string(rune(key))
		}
		key = rl.GetCharPressed()
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(c.renameStr) > 0 {
		c.renameStr = c.renameStr[:len(c.renameStr)-1]
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		if c.renameStr != "" {
			c.Label = c.renameStr
		}
		c.renaming = false
		// Continue the right-click flow: open the size/matrix table editor so one
		// right-click can rename, resize and set the matrix values.
		c.editing = true
		c.editErr = ""
		c.holdingCursor = true
		*isCursorAvailable = false
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		c.renaming = false
		c.holdingCursor = false
		*isCursorAvailable = true
	}
}

// reconfigureOperation swaps the gate's operation and qubit count, rebuilding
// the qubit input hooks. The control and output hooks are kept; the stale
// output system is destroyed since its size may no longer match.
func (c *ControlledUGate) reconfigureOperation(op [][]complex64, inputCount int32) {
	c.DestroyOutPut()
	for _, h := range c.QubitHooks {
		h.Disconnect()
	}

	c.Operation = op
	c.InputCount = inputCount
	c.QubitHooks = nil

	dist := c.hookDist()
	for i := 0; i < int(inputCount); i++ {
		newHook := NewHook(c.Center.X-dist, c.Center.Y+c.qubitOffset(int32(i)), glob.HookRadius, config.HookColor)
		newHook.Label = "I" + strconv.Itoa(i)
		newHook.Tooltip = "CU input: plug a qubit determinator"
		c.QubitHooks = append(c.QubitHooks, newHook)
	}
	c.InControl.Center = rl.Vector2{X: c.Center.X - dist, Y: c.Center.Y + c.controlOffset()}
}
