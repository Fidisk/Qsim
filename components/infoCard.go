package components

import (
	"fmt"
	"math"
	"math/cmplx"
	"qsim/config"
	"qsim/globals"
	glob "qsim/globals"
	qub "qsim/qubits"
	"qsim/utils"
	"sort"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	tablePaddingTop  = 8.0
	tableRowHeight   = 24.0
	tableFontSize    = 14
	tablePaddingLeft = 8.0
)

// InfoRow holds one line of the table.
type InfoRow struct {
	Label    string
	Progress float32 // 0.0 to 1.0
	Complex  complex128
}

// InfoTable draws a rectangular table with multiple rows.
// It embeds Circle (like Button/QubitsSystem) so it can be dragged.
type InfoTable struct {
	Circle
	ID            int32
	Width, Height float32
	Rows          []InfoRow
	Color         rl.Color

	Hook *Hook

	// dragging state (same as QubitsSystem/Button)
	dragging      bool
	holdingCursor bool
	offset        rl.Vector2

	isResizing     bool
	resizeTL       rl.Vector2 // fixed top‑left corner during resize
	cursorOnResize bool
}

// NewInfoTable creates a table widget.
func NewInfoTable(x, y, width, height float32, color rl.Color, rows []InfoRow) *InfoTable {
	radius := maxF(width, height) / 2
	tmp := InfoTable{
		Circle: *NewCircle(x, y, radius, color),
		Width:  width,
		Height: height,
		Color:  color,
		Rows:   rows,
	}
	tmp.ID = utils.GenerateID(tmp)
	tmp.Hook = NewHook(utils.SnapToGrid(x+width/2+10, config.SnapToGridInterval), y, globals.HookRadius, globals.HookColor)
	tmp.Hook.Label = "Input"
	tmp.Hook.AllowQubitSystem = true
	return &tmp
}

func (it *InfoTable) pullToHook() {
	margin := float32(math.Max(math.Max(float64(it.Width), float64(it.Height)), float64((it.Width+it.Height)/2)))
	val := utils.Dist(it.Center, it.Hook.Center) - utils.SnapToGrid(glob.InfoCardToHookDist+margin, config.SnapToGridInterval)
	if math.Abs(float64(val)) <= float64(glob.GateToHookGraceDist) {
		return
	}
	val = float32(math.Max(float64(val), float64(-100)))
	val = float32(math.Min(float64(val), float64(100)))
	dir := it.Center.Subtract(it.Hook.Center).Normalize().Scale(val * glob.GateToHookPullCoeff)
	if it.Hook.IsHooked {
		target := utils.GetObjectFromID(it.Hook.TargetID)
		if target == nil {
			it.Hook.Disconnect()
			return
		}
		if !config.PhysicsEnabled {
			return
		}
		t := target.(Component)
		t.AddForce(dir)
		it.AddForce(dir.Scale(-1))
	} else if config.PhysicsEnabled {
		it.Hook.AddForce(dir)
		it.AddForce(dir.Scale(-1))
	}
}

func (c *InfoTable) GetID() int32 {
	return c.ID
}

// CheckCollide returns true if the mouse is inside the table rectangle.
func (it *InfoTable) CheckCollide(worldMouse rl.Vector2) bool {
	rect := rl.NewRectangle(
		it.Center.X-it.Width/2,
		it.Center.Y-it.Height/2,
		it.Width,
		it.Height,
	)
	return rl.CheckCollisionPointRec(worldMouse, rect)
}

func (it *InfoTable) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// --- Resize handle (bottom‑right corner) ---
	const resizeHandleSize = 15.0
	resizeRect := rl.NewRectangle(
		it.Center.X+it.Width/2-resizeHandleSize,
		it.Center.Y+it.Height/2-resizeHandleSize,
		resizeHandleSize, resizeHandleSize,
	)
	hoverResize := rl.CheckCollisionPointRec(worldMouse, resizeRect)

	// Cursor icon
	if hoverResize && holdingCursor && *isCursorAvailable {
		rl.SetMouseCursor(rl.MouseCursorResizeNWSE)
	} else if !hoverResize && it.cursorOnResize {
		rl.SetMouseCursor(rl.MouseCursorDefault)
		it.cursorOnResize = false
	}
	if hoverResize {
		it.cursorOnResize = true
	}

	// Start resizing
	if hoverResize && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (it.holdingCursor || *isCursorAvailable) {
		it.isResizing = true
		it.holdingCursor = true
		*isCursorAvailable = false
		// Remember top‑left corner
		it.resizeTL = rl.Vector2{X: it.Center.X - it.Width/2, Y: it.Center.Y - it.Height/2}
	}

	// While resizing
	if it.isResizing {
		if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			newW := worldMouse.X - it.resizeTL.X
			newH := worldMouse.Y - it.resizeTL.Y
			if newW < 80 {
				newW = 80
			} // minimum width
			if newH < 40 {
				newH = 40
			} // minimum height
			it.Width = newW
			it.Height = newH
			it.Center = rl.Vector2{X: it.resizeTL.X + newW/2, Y: it.resizeTL.Y + newH/2}
		} else {
			it.isResizing = false
			it.holdingCursor = false
			*isCursorAvailable = true
			rl.SetMouseCursor(rl.MouseCursorDefault)
		}
		return // don't process dragging while resizing
	}

	// --- Dragging (unchanged) ---
	if it.CheckCollide(worldMouse) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (it.holdingCursor || *isCursorAvailable) {
			it.dragging = true
			*isCursorAvailable = false
			it.holdingCursor = true
			it.offset = rl.Vector2Subtract(it.Center, worldMouse)
			it.VirtualCenter = it.Center
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && it.holdingCursor {
		it.dragging = false
		it.holdingCursor = false
		*isCursorAvailable = true
		it.Center = it.VirtualCenter
		it.ClearForce()
	}
	if it.dragging {
		raw := rl.Vector2Add(worldMouse, it.offset)
		oldVC := it.VirtualCenter
		it.VirtualCenter = rl.Vector2Lerp(it.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		delta := it.VirtualCenter.Subtract(oldVC)
		it.Hook.Center = it.Hook.Center.Add(delta)
	}

	it.pullToHook()
	it.Hook.Update(worldMouse, holdingCursor, isCursorAvailable)

	if it.Hook.IsHooked {
		target := utils.GetObjectFromID(it.Hook.TargetID)
		if target == nil {
			return
		}
		qs, ok := target.(*QubitsSystem)
		if !ok {
			return
		}
		it.RebuildRowsFromStateManager(qs.Origin)

		it.Height = 2*tablePaddingTop + float32(len(it.Rows))*tableRowHeight
	}
}

func (it *InfoTable) RebuildRowsFromStateManager(sm *qub.QubitStateManager) {
	if sm == nil {
		it.Rows = nil
		return
	}
	n := int(sm.Size) // number of modifier IDs = n
	total := 1 << n   // 2^n amplitudes
	rows := make([]InfoRow, 0, total)

	for i := 0; i < total; i++ {
		amp := complex128(sm.Amptitude[i])
		progress := float32(cmplx.Abs(amp)) // |amplitude|
		progress *= progress

		// Build label: join modifier IDs where the i-th bit is set
		var parts []string

		s := ""

		for bit := 0; bit < n; bit++ {
			if (i>>bit)&1 == 1 {
				parts = append(parts, strconv.Itoa(int(sm.ModifierID[bit])))
				s += "1"
			} else {
				s += "0"
			}
		}
		// Sort parts numerically (same as in QubitsSystem)
		sort.Slice(parts, func(a, b int) bool {
			ai, _ := strconv.Atoi(parts[a])
			bi, _ := strconv.Atoi(parts[b])
			return ai < bi
		})
		label := strings.Join(parts, " + ")

		if label == "" {
			label = "None"
		}

		label += "("
		label += s + ")"

		rows = append(rows, InfoRow{
			Label:    label,
			Progress: progress,
			Complex:  amp,
		})
	}

	// Sort rows by progress descending
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Progress > rows[j].Progress
	})

	it.Rows = rows
}

func (it *InfoTable) Draw() {
	edge := utils.RectEdgePoint(it.Center, it.Hook.Center, it.Width/2, it.Height/2)
	if it.Hook.IsHooked {
		target := utils.GetObjectFromID(it.Hook.TargetID)
		if target == nil {
			it.Hook.Disconnect()
			return
		}
		t := target.(Component)
		end := rl.Vector2Add(edge, rl.Vector2Scale(
			rl.Vector2Normalize(rl.Vector2Subtract(it.Hook.Center, edge)),
			utils.Dist(edge, t.GetCircle().Center)-t.GetCircle().Radius,
		))
		rl.DrawLineEx(edge, end, 4, globals.HookColor)
	} else {
		rl.DrawLineEx(edge, it.Hook.Center, 4, globals.HookColor)
	}
	it.Hook.Draw()

	// --- Outer rectangle (sharp corners) ---
	rect := rl.NewRectangle(
		it.Center.X-it.Width/2,
		it.Center.Y-it.Height/2,
		it.Width,
		it.Height,
	)
	rl.DrawRectangleRec(rect, it.Color)
	rl.DrawRectangleLinesEx(rect, 4, rl.Black)

	const paddingLeft = 8
	const paddingTop = 8
	const fontSize int32 = 14
	const rowHeight float32 = 24

	// Column divider positions
	col1Right := it.Center.X - it.Width/2 + paddingLeft + it.Width*0.4
	col2Right := it.Center.X - it.Width/2 + paddingLeft + it.Width*0.75

	// --- Draw column separators (like QubitsSystem grid lines) ---
	rl.DrawLineEx(rl.Vector2{X: col1Right, Y: rect.Y}, rl.Vector2{X: col1Right, Y: rect.Y + rect.Height}, 4, rl.Black)
	rl.DrawLineEx(rl.Vector2{X: col2Right, Y: rect.Y}, rl.Vector2{X: col2Right, Y: rect.Y + rect.Height}, 4, rl.Black)

	// Available width for the label column
	labelMaxWidth := col1Right - (it.Center.X - it.Width/2) - paddingLeft

	visibleRows := int((it.Height - 2*tablePaddingTop) / tableRowHeight)
	if visibleRows < 0 {
		visibleRows = 0
	}

	for i, row := range it.Rows {
		if i >= visibleRows {
			break
		}
		rowY := it.Center.Y - it.Height/2 + tablePaddingTop + float32(i)*tableRowHeight

		// Column 1 – label (truncated if needed, original data stays intact)
		label := truncateText(row.Label, fontSize, labelMaxWidth)
		labelW := float32(rl.MeasureText(label, fontSize))
		labelX := int32(col1Right - paddingLeft - labelW) // right‑aligned
		labelY := int32(rowY + (rowHeight-float32(fontSize))/2)
		rl.DrawText(label, labelX, labelY, fontSize, rl.White)

		// Column 2 – progress bar with rounded ends
		barX := col1Right + paddingLeft
		barW := col2Right - barX - paddingLeft
		barH := rowHeight * 0.5
		barY := rowY + (rowHeight-barH)/2

		// Background (darker)
		//rl.DrawRectangleRec(rl.NewRectangle(barX, barY, barW, barH), darken(it.Color, 0.4))

		progress := clampF(row.Progress, 0, 1)
		fillW := barW * progress
		if fillW >= 0 {
			radius := barH / 2
			rl.DrawRectangleRounded(rl.NewRectangle(barX, barY, barW, barH), radius, 8, rl.LightGray)
			if fillW > 0 {
				rl.DrawRectangleRounded(rl.NewRectangle(barX, barY, fillW, barH), radius, 8, progressColor(progress))
			}
		}

		// Column 3 – complex number (centred in remaining space)
		complexStr := formatComplex(row.Complex)
		cTextW := rl.MeasureText(complexStr, fontSize)
		remainingWidth := (it.Center.X + it.Width/2) - col2Right - 2*paddingLeft
		cTextX := int32(col2Right + paddingLeft + (remainingWidth-float32(cTextW))/2)
		rl.DrawText(complexStr, cTextX, labelY, fontSize, rl.White)

		if i < len(it.Rows)-1 {
			rl.DrawLineEx(
				rl.Vector2{X: rect.X, Y: rowY + rowHeight},
				rl.Vector2{X: rect.X + rect.Width, Y: rowY + rowHeight},
				4, rl.Black,
			)
		}
	}
}

func (it *InfoTable) DrawGhost() {
	ghostColor := rl.Fade(it.Color, 0.3)
	rect := rl.NewRectangle(
		it.Center.X-it.Width/2,
		it.Center.Y-it.Height/2,
		it.Width,
		it.Height,
	)
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)
	it.Hook.DrawGhost()
}

func (it *InfoTable) GetChildCircles() []*Circle {
	if it.Hook == nil || it.Hook.IsHooked {
		return nil
	}
	return []*Circle{it.Hook.GetCircle()}
}

// ---------- helpers ----------
func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func clampF(v, low, high float32) float32 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func progressColor(t float32) rl.Color {
	if t < 0.5 {
		return rl.Color{R: uint8(255 * t * 2), G: 255, B: 0, A: 255}
	}
	return rl.Color{R: 255, G: uint8(255 * (2 - t*2)), B: 0, A: 255}
}

func formatComplex(c complex128) string {
	r := real(c)
	im := imag(c)
	if im >= 0 {
		return fmt.Sprintf("%.2f+%.2fi", r, im)
	}
	return fmt.Sprintf("%.2f-%.2fi", r, -im)
}

func truncateText(s string, fontSize int32, maxWidth float32) string {
	if maxWidth <= 0 {
		return ""
	}
	if float32(rl.MeasureText(s, fontSize)) <= maxWidth {
		return s
	}
	ellipsis := "..."
	ellipsisWidth := float32(rl.MeasureText(ellipsis, fontSize))
	if maxWidth <= ellipsisWidth {
		return ellipsis
	}
	runes := []rune(s)
	for i := len(runes) - 1; i >= 1; i-- {
		candidate := string(runes[:i])
		if float32(rl.MeasureText(candidate, fontSize))+ellipsisWidth <= maxWidth {
			return candidate + ellipsis
		}
	}
	return ellipsis
}
