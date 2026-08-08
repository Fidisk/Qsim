package components

import (
	"strconv"
	"strings"

	glob "qsim/globals"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type TextBox struct {
	Circle
	Width, Height float32
	Text          string
	FontSize      int32
	Active        bool
	ID            int32
	holdingCursor bool
	frameCounter  int
	btnHeld       int     // 0 = none, 1 = minus, 2 = plus (while mouse held)
	btnHover      int     // control under the mouse, for drawing highlight
	holdTime      float32 // seconds the current button has been held
	rampTicks     int     // repeats in the current hold session (step growth)
	sizeEditing   bool    // typing a font size directly in the size field
	sizeStr       string  // buffer for the size field while editing
}

func NewTextBox(x, y, width, height float32, fontSize int32) *TextBox {
	tmp := &TextBox{
		Circle:   *NewCircle(x, y, 1, rl.Blank),
		Width:    width,
		Height:   height,
		Text:     "",
		FontSize: fontSize,
		Active:   false,
	}
	tmp.ID = utils.GenerateID(tmp)
	return tmp
}

func (tb *TextBox) GetID() int32 { return tb.ID }

// fontControls returns the +/- buttons and the editable size field above
// the box: [-] [ size ] [+].
func (tb *TextBox) fontControls(rect rl.Rectangle) (minus, sizeField, plus rl.Rectangle) {
	btnW, btnH := float32(22), float32(18)
	sizeW := float32(40)
	by := rect.Y - btnH - 3
	minus = rl.NewRectangle(rect.X+2, by, btnW, btnH)
	sizeField = rl.NewRectangle(rect.X+btnW+5, by, sizeW, btnH)
	plus = rl.NewRectangle(rect.X+btnW+5+sizeW+4, by, btnW, btnH)
	return
}

// stepFont adjusts the font size by delta (clamped at the small end) and
// refits the box. There is no upper clamp: the size is only bounded by what
// fits on the canvas.
func (tb *TextBox) stepFont(delta int32) {
	tb.FontSize = max(tb.FontSize+delta, 6)
	tb.fitToText()
}

func (tb *TextBox) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	rect := rl.NewRectangle(
		tb.Center.X-tb.Width/2,
		tb.Center.Y-tb.Height/2,
		tb.Width,
		tb.Height,
	)

	over := rl.CheckCollisionPointRec(worldMouse, rect)

	// Font-size controls above the box, visible only while editing:
	// [-] [size] [+]. Clicking +/- steps once; holding ramps the rate; the
	// middle field shows the current size and can be clicked to type a new
	// one directly.
	tb.btnHover = 0
	if tb.Active && utils.IsMouseState(glob.MouseStateNormal) {
		minus, sizeField, plus := tb.fontControls(rect)
		if tb.sizeEditing {
			key := rl.GetCharPressed()
			for key > 0 {
				if key >= '0' && key <= '9' && len(tb.sizeStr) < 4 {
					tb.sizeStr += string(rune(key))
				}
				key = rl.GetCharPressed()
			}
			if rl.IsKeyPressed(rl.KeyBackspace) && len(tb.sizeStr) > 0 {
				tb.sizeStr = tb.sizeStr[:len(tb.sizeStr)-1]
			}
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
				if v, err := strconv.Atoi(tb.sizeStr); err == nil && v >= 1 {
					tb.FontSize = int32(v)
					tb.fitToText()
				}
				tb.sizeEditing = false
			}
			if rl.IsKeyPressed(rl.KeyEscape) {
				tb.sizeEditing = false
			}
		} else {
			if rl.CheckCollisionPointRec(worldMouse, sizeField) {
				tb.btnHover = 3
			}
			if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (*isCursorAvailable || tb.holdingCursor) {
				if rl.CheckCollisionPointRec(worldMouse, sizeField) {
					tb.sizeEditing = true
					tb.sizeStr = strconv.Itoa(int(tb.FontSize))
				}
			}
		}
		if rl.CheckCollisionPointRec(worldMouse, minus) {
			tb.btnHover = 1
		}
		if rl.CheckCollisionPointRec(worldMouse, plus) {
			tb.btnHover = 2
		}
		if rl.IsMouseButtonDown(rl.MouseButtonLeft) && holdingCursor && (*isCursorAvailable || tb.holdingCursor) {
			overMinus := rl.CheckCollisionPointRec(worldMouse, minus)
			overPlus := rl.CheckCollisionPointRec(worldMouse, plus)
			if tb.btnHeld == 0 {
				switch {
				case overMinus:
					tb.btnHeld, tb.holdTime, tb.rampTicks = 1, 0, 0
					tb.stepFont(-1)
				case overPlus:
					tb.btnHeld, tb.holdTime, tb.rampTicks = 2, 0, 0
					tb.stepFont(1)
				}
			} else {
				tb.holdTime += rl.GetFrameTime()
				interval := float32(0.25)
				for tb.holdTime >= interval {
					tb.holdTime -= interval
					tb.rampTicks++
					// Exponential scaling: every repeat doubles the step
					// size (1, 2, 4, 8, ...) up to a 64px cap, so holding
					// scales faster and faster.
					step := int32(1) << min(tb.rampTicks, 6)
					if tb.btnHeld == 1 {
						tb.stepFont(-step)
					} else {
						tb.stepFont(step)
					}
					interval = max(interval*0.6, 0.04)
				}
			}
		} else {
			tb.btnHeld = 0
			tb.holdTime = 0
			tb.rampTicks = 0
		}
	}

	if utils.IsMouseState(glob.MouseStateErase) && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && over {
		tb.GetParent().DeleteChildWithID(tb.ID)
		return
	}

	if tb.Active && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && !over {
		// A click on the font controls must not end the edit session.
		minus, sizeField, plus := tb.fontControls(rect)
		if !rl.CheckCollisionPointRec(worldMouse, minus) &&
			!rl.CheckCollisionPointRec(worldMouse, sizeField) &&
			!rl.CheckCollisionPointRec(worldMouse, plus) {
			tb.Active = false
		}
	}

	if !utils.IsMouseState(glob.MouseStateErase) && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && over && (*isCursorAvailable || tb.holdingCursor) {
		tb.dragging = true
		*isCursorAvailable = false
		tb.holdingCursor = true
		tb.offset = rl.Vector2Subtract(tb.Center, worldMouse)
		tb.VirtualCenter = tb.Center
	}

	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && tb.holdingCursor {
		tb.dragging = false
		tb.holdingCursor = false
		*isCursorAvailable = true
		tb.Center = tb.VirtualCenter
		tb.ClearForce()
	}

	if tb.dragging {
		raw := rl.Vector2Add(worldMouse, tb.offset)
		tb.VirtualCenter = rl.Vector2Lerp(tb.VirtualCenter, raw, 0.25)
		tb.ClearForce()
		return
	}

	if rl.IsMouseButtonPressed(rl.MouseButtonRight) && holdingCursor && (*isCursorAvailable || over) {
		if over {
			tb.Active = true
			*isCursorAvailable = false
		}
	}

	if !tb.Active {
		return
	}

	tb.frameCounter++
	if tb.sizeEditing {
		// Typing goes to the size field; skip text input this frame.
		return
	}
	key := rl.GetCharPressed()
	for key > 0 {
		if key >= 32 && key <= 125 {
			tb.Text += string(rune(key))
		}
		key = rl.GetCharPressed()
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(tb.Text) > 0 {
		tb.Text = tb.Text[:len(tb.Text)-1]
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		tb.Text += "\n"
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		tb.Active = false
	}

	tb.fitToText()
}

func (tb *TextBox) fitToText() {
	lines := strings.Split(tb.Text, "\n")
	lineH := float32(tb.FontSize) + 4
	paddingX := float32(24)
	paddingY := float32(12)
	minW := float32(80)
	minH := float32(lineH + paddingY)

	maxW := float32(0)
	for _, line := range lines {
		w := float32(rl.MeasureText(line, tb.FontSize))
		if w > maxW {
			maxW = w
		}
	}
	newW := maxW + paddingX
	newH := float32(len(lines))*lineH + paddingY
	if newW < minW {
		newW = minW
	}
	if newH < minH {
		newH = minH
	}
	tb.Width = newW
	tb.Height = newH
}

func (tb *TextBox) Draw() {
	// Always fit the box to its text, editing or not, so a text box never
	// shows text sticking out of its border.
	tb.fitToText()

	x := tb.Center.X - tb.Width/2
	y := tb.Center.Y - tb.Height/2
	w := tb.Width
	h := tb.Height

	border := rl.DarkGray
	bg := rl.NewColor(30, 30, 30, 255)
	if tb.Active {
		border = rl.SkyBlue
	}
	box := rl.NewRectangle(x, y, w, h)
	rl.DrawRectangleRounded(box, 0.1, 4, bg)
	rl.DrawRectangleRoundedLinesEx(box, 0.1, 4, 2, border)

	// Font-size controls above the box, only while editing: [-] [size] [+].
	// The control under the mouse is highlighted (btnHover is set during
	// Update); the size field shows the live value, or the typed buffer with
	// a cursor while sizeEditing.
	if tb.Active && !utils.IsMouseState(glob.MouseStateErase) {
		minus, sizeField, plus := tb.fontControls(box)
		for _, c := range []struct {
			rect rl.Rectangle
			text string
			hot  int
		}{
			{minus, "-", 1},
			{sizeField, "", 3},
			{plus, "+", 2},
		} {
			bg := rl.NewColor(45, 45, 45, 255)
			if tb.btnHover == c.hot {
				bg = rl.NewColor(70, 80, 110, 255)
			}
			if c.hot == 3 && tb.sizeEditing {
				bg = rl.NewColor(60, 70, 100, 255)
			}
			rl.DrawRectangleRounded(c.rect, 0.2, 4, bg)
			rl.DrawRectangleRoundedLinesEx(c.rect, 0.2, 4, 1.5, rl.SkyBlue)
			txt := c.text
			if c.hot == 3 {
				if tb.sizeEditing {
					txt = tb.sizeStr
				} else {
					txt = strconv.Itoa(int(tb.FontSize))
				}
			}
			tw := rl.MeasureText(txt, 12)
			rl.DrawText(txt, int32(c.rect.X)+int32(c.rect.Width)/2-tw/2, int32(c.rect.Y)+3, 12, rl.White)
			if c.hot == 3 && tb.sizeEditing && (tb.frameCounter/20)%2 == 0 {
				rl.DrawRectangle(int32(c.rect.X)+int32(c.rect.Width)/2+tw/2+1, int32(c.rect.Y)+4, 2, 12, rl.SkyBlue)
			}
		}
	}

	lines := strings.Split(tb.Text, "\n")
	lineH := float32(tb.FontSize) + 4
	textX := int32(x + 8)
	textY := y + (h-float32(len(lines))*lineH)/2

	for i, line := range lines {
		rl.DrawText(line, textX, int32(textY+float32(i)*lineH), tb.FontSize, rl.White)
	}

	if tb.Active && (tb.frameCounter/20)%2 == 0 {
		lastLine := lines[len(lines)-1]
		cursorX := int32(x+8) + rl.MeasureText(lastLine, tb.FontSize)
		cursorY := int32(textY + float32(len(lines)-1)*lineH)
		rl.DrawRectangle(cursorX, cursorY, 4, tb.FontSize, rl.SkyBlue)
	}
}

func (tb *TextBox) DrawGhost() {
	x := tb.Center.X - tb.Width/2
	y := tb.Center.Y - tb.Height/2
	rl.DrawRectangleLinesEx(rl.NewRectangle(x, y, tb.Width, tb.Height), 4, rl.Fade(rl.White, 0.3))
}
