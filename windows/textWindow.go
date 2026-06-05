package windows

import (
	"fmt"
	glob "qsim/globals"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type InputWindow struct {
	Window              
	TextBuffer   string
	MaxChars     int
	Active       bool 
	OnSubmit     func(string)
	
	frameCounter int
}

func NewInputWindow(title string, x, y, width, height int32, maxChars int, onSubmit func(string)) *InputWindow {
	iw := &InputWindow{
		Window:     *NewWindow(x, y, width, height),
		MaxChars:   maxChars,
		Active:     true, 
		OnSubmit:   onSubmit,
		TextBuffer: "",
	}
	iw.Name = title 
	return iw
}

func (iw *InputWindow) Update() {
	mousePos := rl.GetMousePosition()
	windowRect := rl.Rectangle{
		X: float32(iw.X), Y: float32(iw.Y),
		Width: float32(iw.Width), Height: float32(iw.Height),
	}

	// FIX: Check if we are inside bounds and whether cursor state is accessible to us
	if rl.CheckCollisionPointRec(mousePos, windowRect) && (glob.CursorAvailable || iw.holdingCursor) {
		iw.holdingCursor = true
		glob.CursorAvailable = false
	} else {
		iw.holdingCursor = false
	}

	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		if iw.holdingCursor {
			iw.Active = true
			iw.activate = true
		} else {
			iw.Active = false
		}
	}

	iw.handleResize(mousePos)
	if !iw.IsResizing {
		iw.handleDrag(mousePos)
	}

	if !iw.Active {
		return
	}

	iw.frameCounter++

	key := rl.GetCharPressed()
	for key > 0 {
		if (key >= 32) && (key <= 125) && (len(iw.TextBuffer) < iw.MaxChars) {
			iw.TextBuffer += string(rune(key))
		}
		key = rl.GetCharPressed() 
	}

	if rl.IsKeyPressed(rl.KeyBackspace) {
		if len(iw.TextBuffer) > 0 {
			iw.TextBuffer = iw.TextBuffer[:len(iw.TextBuffer)-1]
		}
	} else if rl.IsKeyDown(rl.KeyBackspace) {
		if iw.frameCounter%6 == 0 { 
			if len(iw.TextBuffer) > 0 {
				iw.TextBuffer = iw.TextBuffer[:len(iw.TextBuffer)-1]
			}
		}
	}

	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		if iw.OnSubmit != nil {
			iw.OnSubmit(iw.TextBuffer)
		}
	}
}

func (iw *InputWindow) Draw() {
	rl.DrawRectangle(iw.X, iw.Y, iw.Width, iw.Height, iw.ColorBg)
	rl.DrawRectangle(iw.X, iw.Y, iw.Width, iw.TitleBarHeight, iw.ColorTitleBar)
	rl.DrawText(iw.Name, iw.X+5, iw.Y+5, 16, iw.ColorText)

	handleColor := iw.ColorResize
	if iw.cursorOnResize {
		handleColor = iw.ColorResizeHover
	}

	inputFieldX := float32(iw.X) + 12
	inputFieldY := float32(iw.Y + iw.TitleBarHeight) + 12
	inputFieldW := float32(iw.Width) - 24
	inputFieldH := float32(iw.Height - iw.TitleBarHeight) - 30

	inputBox := rl.NewRectangle(inputFieldX, inputFieldY, inputFieldW, inputFieldH)
	rl.DrawRectangleRec(inputBox, rl.NewColor(20, 20, 20, 255))
	
	borderColor := rl.DarkGray
	if iw.Active {
		borderColor = rl.SkyBlue
	}
	rl.DrawRectangleLinesEx(inputBox, 1, borderColor)

	rl.DrawText(iw.TextBuffer, int32(inputBox.X)+10, int32(inputBox.Y)+14, 16, rl.RayWhite)

	if iw.Active {
		if (iw.frameCounter/20)%2 == 0 {
			textWidth := rl.MeasureText(iw.TextBuffer, 16)
			rl.DrawRectangle(int32(inputBox.X)+10+textWidth+2, int32(inputBox.Y)+14, 2, 16, rl.SkyBlue)
		}
	}

	charCountStr := fmt.Sprintf("%d/%d", len(iw.TextBuffer), iw.MaxChars)
	rl.DrawText(charCountStr, int32(inputBox.X+inputBox.Width)-rl.MeasureText(charCountStr, 12)-8, int32(inputBox.Y+inputBox.Height)-18, 12, rl.DarkGray)

	rl.DrawTriangle(
		rl.Vector2{X: float32(iw.X + iw.Width), Y: float32(iw.Y + iw.Height - 15)},
		rl.Vector2{X: float32(iw.X + iw.Width - 15), Y: float32(iw.Y + iw.Height)},
		rl.Vector2{X: float32(iw.X + iw.Width), Y: float32(iw.Y + iw.Height)},
		handleColor,
	)
	rl.DrawRectangleLines(iw.X, iw.Y, iw.Width, iw.Height, iw.ColorBorder)
}

func (iw *InputWindow) PostUpdate() {}