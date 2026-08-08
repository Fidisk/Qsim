package components

import (
	"fmt"
	"qsim/config"
	glob "qsim/globals"
	"qsim/unitary"
	"qsim/utils"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// NewArbGate creates an editable gate: right-clicking the body opens an
// inline editor where the user sets the qubit count and the full operation
// matrix. It spawns as the 1-qubit identity labeled "U".
func NewArbGate(x, y, radius float32, color rl.Color) *Gate {
	g := NewGate(x, y, radius, color, "U", [][]complex64{{1, 0}, {0, 1}}, 1)
	g.Editable = true
	g.Tooltip = "Arbitrary gate: right-click to rename, then edit size and the matrix table; non-unitary matrices are auto-fixed on close; hover to preview"
	return g
}

// ParseMatrixSpec parses the inline matrix spec:
//
//	n ; row ; row ; ...      (exactly 2^n rows)
//
// where each row holds 2^n whitespace-separated entries and each entry is
// "re" or "re,im". Example (1-qubit X):  "1; 0 1; 1 0".
func ParseMatrixSpec(spec string) (int32, [][]complex64, error) {
	parts := strings.Split(spec, ";")
	if len(parts) < 2 {
		return 0, nil, fmt.Errorf("expected 'n; row; row; ...'")
	}
	n64, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil || n64 < 1 || n64 > 4 {
		return 0, nil, fmt.Errorf("qubit count must be 1-4")
	}
	n := int32(n64)
	size := 1 << n
	rows := parts[1:]
	if len(rows) != size {
		return 0, nil, fmt.Errorf("expected %d rows, got %d", size, len(rows))
	}
	op := make([][]complex64, size)
	for i, row := range rows {
		fields := strings.Fields(row)
		if len(fields) != size {
			return 0, nil, fmt.Errorf("row %d: expected %d entries, got %d", i, size, len(fields))
		}
		op[i] = make([]complex64, size)
		for j, f := range fields {
			v, err := parseMatrixEntry(f)
			if err != nil {
				return 0, nil, fmt.Errorf("row %d entry %d: %v", i, j, err)
			}
			op[i][j] = v
		}
	}
	return n, op, nil
}

// parseMatrixEntry parses a single matrix entry in either form:
//
//	"re" or "re,im"          (comma form:  0.7  or  0.7,0.3)
//	"a+bi" / "a-bi" / "i"    (math form:   0.7+0.3i, 1-i, i, -i, 2i)
//
// Exponent signs are respected ("1e-3+2i" splits at the plus, not the e-).
func parseMatrixEntry(s string) (complex64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty entry")
	}

	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		if len(parts) != 2 {
			return 0, fmt.Errorf("bad entry %q (want re, re,im or a+bi)", s)
		}
		re, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
		im, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
		return complex64(complex(re, im)), nil
	}

	hasImag := strings.HasSuffix(s, "i")
	body := s
	if hasImag {
		body = s[:len(s)-1]
		switch body {
		case "":
			return complex64(1i), nil // "i"
		case "+":
			return complex64(1i), nil
		case "-":
			return complex64(-1i), nil // "-i"
		}
		// Split real/imag at the last sign that is not an exponent sign.
		split := -1
		for i := len(body) - 1; i > 0; i-- {
			c := body[i]
			if (c == '+' || c == '-') && body[i-1] != 'e' && body[i-1] != 'E' {
				split = i
				break
			}
		}
		if split == -1 {
			im, err := strconv.ParseFloat(body, 64) // pure imaginary: "2i"
			if err != nil {
				return 0, fmt.Errorf("bad entry %q", s)
			}
			return complex64(complex(0, im)), nil
		}
		re, err := strconv.ParseFloat(body[:split], 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
		if split == len(body)-1 {
			// Trailing sign means the imaginary coefficient is 1: "1-i".
			im := float64(1)
			if body[split] == '-' {
				im = -1
			}
			return complex64(complex(re, im)), nil
		}
		im, err := strconv.ParseFloat(body[split:], 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
		return complex64(complex(re, im)), nil
	}

	re, err := strconv.ParseFloat(s, 64) // pure real: "1", "-0.5", "1e-3"
	if err != nil {
		return 0, fmt.Errorf("bad entry %q (want re, re,im or a+bi)", s)
	}
	return complex64(complex(re, 0)), nil
}

// formatMatrixSpec renders the gate's current matrix in the edit-buffer
// format, so right-clicking starts from the existing operation.
func formatMatrixSpec(n int32, op [][]complex64) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(int(n)))
	for _, row := range op {
		b.WriteString("; ")
		for j, v := range row {
			if j > 0 {
				b.WriteString(" ")
			}
			re := strconv.FormatFloat(float64(real(v)), 'g', -1, 32)
			if imag(v) == 0 {
				b.WriteString(re)
			} else {
				b.WriteString(re + "," + strconv.FormatFloat(float64(imag(v)), 'g', -1, 32))
			}
		}
	}
	return b.String()
}

// processEditing handles the inline table editor: the same name-and-matrix
// table shown on hover, now interactive. Click a cell to type its value
// ("0.7" or "0.7,0.3"), use the +/- buttons to change the qubit count;
// Enter closes the editor and auto-applies the matrix (raylib's default exit
// key is Escape, so it must not be the way out of the editor).
func (c *Gate) processEditing(worldMouse rl.Vector2, isCursorAvailable *bool) {
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
	minusRect := rl.Rectangle{X: x + w - 54, Y: sizeRowY, Width: 22, Height: 20}
	plusRect := rl.Rectangle{X: x + w - 28, Y: sizeRowY, Width: 22, Height: 20}

	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && (c.holdingCursor || *isCursorAvailable) {
		switch {
		case rl.CheckCollisionPointRec(worldMouse, plusRect) && c.InputCount < 4:
			label := c.Label
			c.Reconfigure(resizeGateOperation(c.Operation, c.InputCount+1), c.InputCount+1)
			c.Label = label
			c.notice = ""
			return
		case rl.CheckCollisionPointRec(worldMouse, minusRect) && c.InputCount > 1:
			label := c.Label
			c.Reconfigure(resizeGateOperation(c.Operation, c.InputCount-1), c.InputCount-1)
			c.Label = label
			if !unitary.IsUnitary(c.Operation, unitary.Tolerance) {
				c.Operation = unitary.NearestUnitary(c.Operation)
				c.notice = "shrunk matrix projected to nearest unitary"
				c.noticeTimer = 4
			}
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
func (c *Gate) finalizeEdit() {
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
func (c *Gate) editPanelGeom() (x, y, w, h, cell, gridY float32, font int32) {
	size := int32(1) << c.InputCount
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
	y = c.Center.Y - c.Radius - (sizeTop + sizeH + gridW + 6) - 8
	gridY = y + sizeTop + sizeH
	h = gridY - y + gridW + 6
	return
}

// DrawEditPanel draws the right-click edit table: the name on top, the
// qubit-count +/- buttons, then the operation matrix as the same grid used by
// the hover preview — the clicked cell is edited inline.
func (c *Gate) DrawEditPanel() {
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
	qtext := "qubits: " + strconv.Itoa(int(c.InputCount))
	rl.DrawText(qtext, int32(x+8), int32(sizeRowY), 14, rl.White)

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

	if c.editErr != "" {
		rl.DrawText(c.editErr, int32(x+6), int32(y+h-18), 12, rl.Orange)
	}
}

// resizeGateOperation returns a new op matrix resized to 2^n x 2^n: growing
// pads with the identity, shrinking truncates to the top-left block.
func resizeGateOperation(op [][]complex64, n int32) [][]complex64 {
	size := int32(1) << n
	out := make([][]complex64, size)
	for i := int32(0); i < size; i++ {
		out[i] = make([]complex64, size)
		if i < int32(len(op)) {
			copy(out[i], op[i][:min(size, int32(len(op[i])))])
		} else {
			out[i][i] = 1
		}
	}
	return out
}

// formatCellValue renders one matrix cell in the edit form "re" or "re,im"
// consumed by parseMatrixEntry.
func formatCellValue(v complex64) string {
	re := strconv.FormatFloat(float64(real(v)), 'g', -1, 32)
	if imag(v) == 0 {
		return re
	}
	return re + "," + strconv.FormatFloat(float64(imag(v)), 'g', -1, 32)
}

// processRename handles the inline rename editor (right-click on the gate),
// mirroring the qubit determinator's rename: type to edit, Enter applies and
// opens the size/matrix editor, Escape cancels.
func (c *Gate) processRename(isCursorAvailable *bool) {
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

// formatGateEntry renders one complex matrix entry compactly, e.g. "1",
// "1+i", "-0.5i", so the hover matrix table stays readable at small sizes.
func formatGateEntry(v complex64) string {
	re := real(v)
	im := imag(v)
	if im == 0 {
		return strconv.FormatFloat(float64(re), 'g', -1, 32)
	}
	if re == 0 {
		s := strconv.FormatFloat(float64(im), 'g', -1, 32)
		if s == "1" {
			return "i"
		}
		if s == "-1" {
			return "-i"
		}
		return s + "i"
	}
	reStr := strconv.FormatFloat(float64(re), 'g', -1, 32)
	imStr := strconv.FormatFloat(float64(im), 'g', -1, 32)
	sign := "+"
	if im < 0 {
		sign = "-"
		imStr = strconv.FormatFloat(float64(-im), 'g', -1, 32)
	}
	if imStr == "1" {
		imStr = ""
	}
	return reStr + sign + imStr + "i"
}

// DrawValuePanel draws the hover info box above an editable gate: the name on
// top and the operation matrix as a grid of cells below, scaled to the space
// available.
func (c *Gate) DrawValuePanel() {
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
	y := c.Center.Y - c.Radius - panelH - 8

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

// Reconfigure swaps the gate's operation and qubit count, rebuilding the
// input hooks. The output hook (and its label/flags) is kept; the stale
// output system is destroyed since its size may no longer match.
func (c *Gate) Reconfigure(op [][]complex64, inputCount int32) {
	c.DestroyOutPut()

	var out *Hook
	if len(c.OutPutHook) > 0 {
		out = c.OutPutHook[0]
	}
	for _, h := range c.HookList {
		if h != out {
			h.Disconnect()
		}
	}

	c.Operation = op
	c.InputCount = inputCount
	c.Label = fmt.Sprintf("U%d", inputCount)
	c.HookList = nil

	snappedDist := utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)
	for i := 0; i < int(inputCount); i++ {
		offY := utils.SnapToGrid((float32(i)-float32(inputCount-1)/2)*config.SnapToGridInterval, config.SnapToGridInterval)
		newHook := NewHook(c.Center.X-snappedDist, c.Center.Y+offY, glob.HookRadius, config.HookColor)
		newHook.Label = "I" + strconv.Itoa(i)
		newHook.Tooltip = "Gate input: plug a qubit determinator"
		c.HookList = append(c.HookList, newHook)
	}
	if out == nil {
		out = NewOutputHook(c.Center.X+snappedDist, c.Center.Y, glob.OutputHookRadius, config.OutputHookColor)
		out.Label = "O"
		out.AllowQubitSystem = true
		out.Tooltip = "Gate output: the transformed qubit system"
	}
	c.HookList = append(c.HookList, out)
	c.OutPutHook = []*Hook{out}
}
