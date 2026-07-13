package components

import (
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
}

func NewTextBox(x, y, width, height float32, fontSize int32) *TextBox {
	tmp := &TextBox{
		Circle:    *NewCircle(x, y, 1, rl.Blank),
		Width:     width,
		Height:    height,
		Text:      "",
		FontSize:  fontSize,
		Active:    false,
	}
	tmp.ID = utils.GenerateID(tmp)
	return tmp
}

func (tb *TextBox) GetID() int32 { return tb.ID }

func (tb *TextBox) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	rect := rl.NewRectangle(
		tb.Center.X-tb.Width/2,
		tb.Center.Y-tb.Height/2,
		tb.Width,
		tb.Height,
	)

	over := rl.CheckCollisionPointRec(worldMouse, rect)

	if utils.IsMouseState(glob.MouseStateErase) && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && over {
		tb.GetParent().DeleteChildWithID(tb.ID)
		return
	}

	if tb.Active && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && !over {
		tb.Active = false
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
	x := tb.Center.X - tb.Width/2
	y := tb.Center.Y - tb.Height/2
	w := tb.Width
	h := tb.Height

	border := rl.DarkGray
	bg := rl.NewColor(30, 30, 30, 255)
	if tb.Active {
		border = rl.SkyBlue
	}
	rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), bg)
	rl.DrawRectangleLinesEx(rl.NewRectangle(x, y, w, h), 1, border)

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
		rl.DrawRectangle(cursorX, cursorY, 2, tb.FontSize, rl.SkyBlue)
	}
}

func (tb *TextBox) DrawGhost() {
	x := tb.Center.X - tb.Width/2
	y := tb.Center.Y - tb.Height/2
	rl.DrawRectangleLinesEx(rl.NewRectangle(x, y, tb.Width, tb.Height), 2, rl.Fade(rl.White, 0.3))
}
