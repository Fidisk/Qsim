package components

import (
	"fmt"
	"math"
	"math/cmplx"
	"qsim/config"
	"strconv"
	"strings"

	glob "qsim/globals"
	"qsim/qubits"
	"qsim/qubits/attributes"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// SourceGate creates a QubitSystem of size 1 (2 amplitudes: |0⟩ and |1⟩) with user‑defined values.
type SourceGate struct {
	Circle
	ID         int32
	Label      string
	Color      rl.Color
	OutHook    *Hook
	Amplitude  []complex64 // index 0 = |0⟩, index 1 = |1⟩
	ModifierID int32

	modifierGenerated bool

	// dragging state
	dragging      bool
	holdingCursor bool
	offset        rl.Vector2

	// inline editing – four real numbers
	editing     bool
	editField   int // 0=Re0, 1=Im0, 2=Re1, 3=Im1
	real0Str    string
	imag0Str    string
	real1Str    string
	imag1Str    string
	cursorBlink float32
	cursorShow  bool

	// table dimensions (cached)
	tableWidth  float32
	tableHeight float32
}

func (sg *SourceGate) getModifierID() int32 {
	if !sg.modifierGenerated {
		sg.ModifierID = attributes.GenerateQubitModifierID()
		sg.modifierGenerated = true
	}
	return sg.ModifierID
}

// layout constants
const (
	sgRowCount    = 2
	sgRowHeight   = 24
	sgPaddingTop  = 8
	sgPaddingLeft = 8
	sgCol1Width   = 50
	sgCol2Width   = 100
	sgCol3Width   = 100
	sgTableWidth  = sgCol1Width + sgCol2Width + sgCol3Width + 3*sgPaddingLeft // approx
	sgTableHeight = 2*sgRowCount*sgRowHeight + 2*sgPaddingTop                 // fixed for 2 rows
)

// NewSourceGate creates a source gate.
func NewSourceGate(x, y, radius float32, color rl.Color, label string, amp []complex64) *SourceGate {
	if len(amp) != 2 {
		amp = []complex64{complex(1, 0), complex(0, 0)}
	}
	sg := &SourceGate{
		Circle:    *NewCircle(x, y, radius, color),
		Label:     label,
		Color:     color,
		Amplitude: amp,
	}
	sg.ID = utils.GenerateID(sg)
	sg.SetWeight(glob.GateWeight)
	// Output hook on the right edge of the table
	sg.OutHook = NewOutputHook(utils.SnapToGrid(x+sgTableWidth/2+glob.OutputHookRadius, config.SnapToGridInterval), y, glob.OutputHookRadius, config.OutputHookColor)
	sg.OutHook.Label = "O"
	sg.OutHook.AllowQubitSystem = true
	sg.OutHook.IsOutput = true
	sg.OutHook.Tooltip = "Source output: emits the configured qubit state"
	// Cache table size
	sg.tableWidth = sgTableWidth
	sg.tableHeight = sgTableHeight
	return sg
}

func NewSourceGateWithID(x, y, radius float32, color rl.Color, label string, amp []complex64, modID int32) *SourceGate {
	if len(amp) != 2 {
		amp = []complex64{complex(1, 0), complex(0, 0)}
	}
	sg := &SourceGate{
		Circle:            *NewCircle(x, y, radius, color),
		Label:             label,
		Color:             color,
		Amplitude:         amp,
		ModifierID:        modID,
		modifierGenerated: true,
	}
	sg.ID = utils.GenerateID(sg)
	sg.SetWeight(glob.GateWeight)
	// Output hook on the right edge of the table
	sg.OutHook = NewOutputHook(utils.SnapToGrid(x+sgTableWidth/2+glob.OutputHookRadius, config.SnapToGridInterval), y, glob.OutputHookRadius, config.OutputHookColor)
	sg.OutHook.Label = "O"
	sg.OutHook.AllowQubitSystem = true
	sg.OutHook.IsOutput = true
	sg.OutHook.Tooltip = "Source output: emits the configured qubit state"
	// Cache table size
	sg.tableWidth = sgTableWidth
	sg.tableHeight = sgTableHeight
	return sg
}

func (sg *SourceGate) GetID() int32       { return sg.ID }
func (sg *SourceGate) GetCircle() *Circle { return &sg.Circle }

// getTableRect returns the rectangle occupied by the table.
func (sg *SourceGate) getTableRect() rl.Rectangle {
	return rl.Rectangle{
		X:      sg.Center.X - sg.tableWidth/2,
		Y:      sg.Center.Y - sg.tableHeight/2,
		Width:  sg.tableWidth,
		Height: sg.tableHeight,
	}
}

// CheckCollide now uses the rectangle.
func (sg *SourceGate) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, sg.getTableRect())
}

func (sg *SourceGate) pullToHook() {
	d := sg.OutHook
	// Hook distance from the right edge of the table
	targetDist := utils.SnapToGrid(sg.tableWidth/2+glob.GateToHookDist, config.SnapToGridInterval)
	val := utils.Dist(sg.Center, d.Center) - targetDist
	if math.Abs(float64(val)) <= float64(glob.GateToHookGraceDist) {
		return
	}
	val = float32(math.Max(float64(val), float64(-100)))
	val = float32(math.Min(float64(val), float64(100)))
	dir := sg.Center.Subtract(d.Center).Normalize().Scale(val * glob.GateToHookPullCoeff)

	if d.IsHooked {
		target := utils.GetObjectFromID(d.TargetID)
		if target == nil {
			d.Disconnect()
			return
		}
		if !config.PhysicsEnabled {
			return
		}
		t := target.(Component)
		t.AddForce(dir)
		sg.AddForce(dir.Scale(-1))
	} else if config.PhysicsEnabled {
		d.AddForce(dir)
		sg.AddForce(dir.Scale(-1))
	}
}

func (sg *SourceGate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if sg.editing {
		sg.processEditing(isCursorAvailable)
		return
	}

	if sg.CheckCollide(worldMouse) {
		if rl.IsMouseButtonPressed(rl.MouseButtonRight) && holdingCursor && (sg.holdingCursor || *isCursorAvailable) {
			// Enter edit mode
			sg.editing = true
			sg.real0Str = fmt.Sprintf("%.4f", real(sg.Amplitude[0]))
			sg.imag0Str = fmt.Sprintf("%.4f", imag(sg.Amplitude[0]))
			sg.real1Str = fmt.Sprintf("%.4f", real(sg.Amplitude[1]))
			sg.imag1Str = fmt.Sprintf("%.4f", imag(sg.Amplitude[1]))
			sg.editField = 0
			*isCursorAvailable = false
			sg.holdingCursor = true
			return
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (sg.holdingCursor || *isCursorAvailable) {
			sg.dragging = true
			*isCursorAvailable = false
			sg.holdingCursor = true
			sg.offset = rl.Vector2Subtract(sg.Center, worldMouse)
			sg.VirtualCenter = sg.Center
		}
	}

	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && sg.holdingCursor {
		sg.dragging = false
		sg.holdingCursor = false
		*isCursorAvailable = true
		sg.Center = sg.VirtualCenter
		sg.ClearForce()
	}

	if sg.dragging {
		raw := rl.Vector2Add(worldMouse, sg.offset)
		oldVC := sg.VirtualCenter
		sg.VirtualCenter = rl.Vector2Lerp(sg.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		delta := sg.VirtualCenter.Subtract(oldVC)
		sg.OutHook.Center = sg.OutHook.Center.Add(delta)
		sg.ClearForce()
	} else {
		sg.DecayForce()
		sg.ApplyForce()
	}

	sg.pullToHook()
	sg.OutHook.Update(worldMouse, holdingCursor, isCursorAvailable)

	if sg.OutHook.IsHooked {
		targetObj := utils.GetObjectFromID(sg.OutHook.TargetID)
		if targetObj == nil {
			return
		}
		target, ok := targetObj.(*QubitsSystem)
		if !ok {
			return
		}
		state := qubits.NewQubitStateManagerFrom(sg.Amplitude, []int32{sg.getModifierID()})
		if target.Origin != nil {
			target.Origin.CopyFrom(state)
		} else {
			target.Assign(state)
		}
	} else {
		state := qubits.NewQubitStateManagerFrom(sg.Amplitude, []int32{sg.getModifierID()})

		tmp := NewQubitsSystem(sg.Center.X, sg.Center.Y, glob.QubitSystemRadius, config.QubitSystemColor)
		tmp.Assign(state)
		parent := sg.GetParent()
		if parent != nil {
			tmp.SetParent(parent)
			parent.PushComponent(tmp)
		}

		sg.OutHook.Connect(tmp)
		tmp.ZipDeterminatorsToHooks()
	}
}

func (sg *SourceGate) processEditing(isCursorAvailable *bool) {
	sg.cursorBlink += rl.GetFrameTime()
	if sg.cursorBlink > 0.5 {
		sg.cursorShow = !sg.cursorShow
		sg.cursorBlink = 0
	}

	if rl.IsKeyPressed(rl.KeyTab) {
		sg.editField = (sg.editField + 1) % 4
	}

	key := rl.GetCharPressed()
	for key > 0 {
		c := string(key)
		if strings.ContainsRune("0123456789.-", key) {
			switch sg.editField {
			case 0:
				sg.real0Str += c
			case 1:
				sg.imag0Str += c
			case 2:
				sg.real1Str += c
			case 3:
				sg.imag1Str += c
			}
		}
		key = rl.GetCharPressed()
	}

	if rl.IsKeyPressed(rl.KeyBackspace) {
		switch sg.editField {
		case 0:
			if len(sg.real0Str) > 0 {
				sg.real0Str = sg.real0Str[:len(sg.real0Str)-1]
			}
		case 1:
			if len(sg.imag0Str) > 0 {
				sg.imag0Str = sg.imag0Str[:len(sg.imag0Str)-1]
			}
		case 2:
			if len(sg.real1Str) > 0 {
				sg.real1Str = sg.real1Str[:len(sg.real1Str)-1]
			}
		case 3:
			if len(sg.imag1Str) > 0 {
				sg.imag1Str = sg.imag1Str[:len(sg.imag1Str)-1]
			}
		}
	}

	if rl.IsKeyPressed(rl.KeyEnter) {
		r0, _ := strconv.ParseFloat(sg.real0Str, 64)
		i0, _ := strconv.ParseFloat(sg.imag0Str, 64)
		r1, _ := strconv.ParseFloat(sg.real1Str, 64)
		i1, _ := strconv.ParseFloat(sg.imag1Str, 64)
		sg.Amplitude[0] = complex64(complex(r0, i0))
		sg.Amplitude[1] = complex64(complex(r1, i1))
		sg.editing = false
		sg.holdingCursor = false
		*isCursorAvailable = true
	}

	if rl.IsKeyPressed(rl.KeyEscape) {
		sg.editing = false
		sg.holdingCursor = false
		*isCursorAvailable = true
	}
}

func (sg *SourceGate) Draw() {
	// --- Connection line to output hook ---
	tableRect := sg.getTableRect()
	edge := utils.RectEdgePoint(sg.Center, sg.OutHook.Center, tableRect.Width/2, tableRect.Height/2)
	if sg.OutHook.IsHooked {
		target := utils.GetObjectFromID(sg.OutHook.TargetID)
		if target == nil {
			sg.OutHook.Disconnect()
			return
		}
		t, ok := target.(Component)
		if !ok {
			sg.OutHook.Disconnect()
			return
		}
		edge = utils.RectEdgePoint(sg.Center, t.GetCircle().Center, tableRect.Width/2, tableRect.Height/2)
		DrawHookLink(edge, t, 4, sg.Color)
	} else {
		DrawWire(edge, sg.OutHook.Center, 4, sg.Color)
	}
	sg.OutHook.Draw()

	// --- Table rectangle ---
	rect := sg.getTableRect()
	rl.DrawRectangleRounded(rect, 0.06, 6, config.ColorBg)
	thickness := float32(4.0)
	if sg.IsFixed {
		thickness = 5.0
	}
	rl.DrawRectangleRoundedLinesEx(rect, 0.06, 6, thickness, sg.Color)

	// Column dividers
	col1Right := rect.X + sgPaddingLeft + sgCol1Width
	col2Right := col1Right + sgPaddingLeft + sgCol2Width
	rl.DrawLineEx(rl.Vector2{X: col1Right, Y: rect.Y}, rl.Vector2{X: col1Right, Y: rect.Y + rect.Height}, 4, sg.Color)
	rl.DrawLineEx(rl.Vector2{X: col2Right, Y: rect.Y}, rl.Vector2{X: col2Right, Y: rect.Y + rect.Height}, 4, sg.Color)

	fontSize := int32(14)

	// Row data
	labels := []string{"|0>", "|1>"}
	for i := 0; i < 2; i++ {
		rowY := rect.Y + sgPaddingTop + float32(i)*sgRowHeight

		// Column 1 – basis label
		lbl := labels[i]
		lblWidth := rl.MeasureText(lbl, fontSize)
		lblX := int32(col1Right - sgPaddingLeft - float32(lblWidth))
		lblY := int32(rowY + (sgRowHeight-float32(fontSize))/2)
		rl.DrawText(lbl, lblX, lblY, fontSize, rl.White)

		// Column 2 – probability bar
		amp := complex128(sg.Amplitude[i])
		prob := float32(cmplx.Abs(amp))
		prob = prob * prob // probability (0..1)
		barX := col1Right + sgPaddingLeft
		barW := col2Right - barX - sgPaddingLeft
		barH := sgRowHeight * float32(0.5)
		barY := rowY + (sgRowHeight-barH)/float32(2.0)
		// Background bar
		rl.DrawRectangleRounded(rl.NewRectangle(barX, barY, barW, barH), barH/2, 8, rl.LightGray)
		fillW := barW * clampF(prob, 0, 1)
		if fillW > 0 {
			rl.DrawRectangleRounded(rl.NewRectangle(barX, barY, fillW, barH), barH/2, 8, progressColor(prob))
		}

		// Column 3 – complex amplitude
		complexStr := formatComplex(amp)
		cTextW := rl.MeasureText(complexStr, fontSize)
		remainingWidth := (rect.X + rect.Width) - col2Right - 2*sgPaddingLeft
		cTextX := int32(col2Right + sgPaddingLeft + (remainingWidth-float32(cTextW))/2)
		rl.DrawText(complexStr, cTextX, lblY, fontSize, rl.White)

		// Row separator
		if i == 0 {
			rl.DrawLineEx(
				rl.Vector2{X: rect.X, Y: rowY + sgRowHeight},
				rl.Vector2{X: rect.X + rect.Width, Y: rowY + sgRowHeight},
				4, sg.Color,
			)
		}
	}

	// --- Editing overlay (shown on top if active) ---
	if sg.editing {
		const editFontSize = 12
		// Position above the table
		yBase := rect.Y - 40
		row1Y := yBase
		row2Y := yBase + 20

		rl.DrawText("|0> Re:", int32(rect.X)+5, int32(row1Y), editFontSize, rl.White)
		rl.DrawText(sg.real0Str, int32(rect.X)+70, int32(row1Y), editFontSize, rl.White)
		rl.DrawText("Im:", int32(rect.X)+130, int32(row1Y), editFontSize, rl.White)
		rl.DrawText(sg.imag0Str, int32(rect.X)+160, int32(row1Y), editFontSize, rl.White)

		rl.DrawText("|1> Re:", int32(rect.X)+5, int32(row2Y), editFontSize, rl.White)
		rl.DrawText(sg.real1Str, int32(rect.X)+70, int32(row2Y), editFontSize, rl.White)
		rl.DrawText("Im:", int32(rect.X)+130, int32(row2Y), editFontSize, rl.White)
		rl.DrawText(sg.imag1Str, int32(rect.X)+160, int32(row2Y), editFontSize, rl.White)

		if sg.cursorShow {
			var cx, cy int32
			switch sg.editField {
			case 0:
				cx = int32(rect.X) + 70 + rl.MeasureText(sg.real0Str, editFontSize)
				cy = int32(row1Y)
			case 1:
				cx = int32(rect.X) + 160 + rl.MeasureText(sg.imag0Str, editFontSize)
				cy = int32(row1Y)
			case 2:
				cx = int32(rect.X) + 70 + rl.MeasureText(sg.real1Str, editFontSize)
				cy = int32(row2Y)
			case 3:
				cx = int32(rect.X) + 160 + rl.MeasureText(sg.imag1Str, editFontSize)
				cy = int32(row2Y)
			}
			rl.DrawText("|", cx, cy, editFontSize, rl.Red)
		}
	}
}

func (sg *SourceGate) DrawGhost() {
	ghostColor := rl.Fade(sg.Color, 0.3)
	rect := sg.getTableRect()
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)
	sg.OutHook.DrawGhost()
}

func (sg *SourceGate) GetChildCircles() []*Circle {
	if sg.OutHook == nil || sg.OutHook.IsHooked {
		return nil
	}
	return []*Circle{sg.OutHook.GetCircle()}
}

func (sg *SourceGate) PostUpdate() {}
