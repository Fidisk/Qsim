package windows

import (
	"fmt"
<<<<<<< HEAD
	glob "qsim/globals"

=======
>>>>>>> parent of 0df1919 (fix bug)
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

<<<<<<< HEAD
	// ADVISOR FIX: If cursor is not available and this window isn't already holding it,
	// do not let it interact or falsely click through from underneath another window.
	if !glob.CursorAvailable && !iw.holdingCursor {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			iw.Active = false
		}
		return
	}

	// Evaluate if the mouse is hovering over this window
	if rl.CheckCollisionPointRec(mousePos, windowRect) {
		iw.holdingCursor = true
		glob.CursorAvailable = false
	} else if glob.CursorAvailable {
		iw.holdingCursor = false
	}

=======
>>>>>>> parent of 0df1919 (fix bug)
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		if rl.CheckCollisionPointRec(mousePos, windowRect) {
			iw.Active = true
<<<<<<< HEAD
			iw.activate = true
			glob.CursorLock = true // Lock cursor to prioritize this window
=======
>>>>>>> parent of 0df1919 (fix bug)
		} else {
			iw.Active = false
		}
	}

	// Delegate resizing and dragging using parent window logic
	iw.handleResize(mousePos)
	if !iw.IsResizing {
		iw.handleDrag(mousePos)
	}

	// Release CursorLock if the user stops dragging or resizing
	if !iw.IsDragging && !iw.IsResizing && glob.CursorLock && !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		glob.CursorLock = false
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

	// === ADJUSTED SPACING FOR A LARGER INTERNAL BOX ===
	inputFieldX := float32(iw.X) + 12
	inputFieldY := float32(iw.Y+iw.TitleBarHeight) + 12
	inputFieldW := float32(iw.Width) - 24
<<<<<<< HEAD
	inputFieldH := float32(iw.Height-iw.TitleBarHeight) - 30
=======
	// Adjusted height calculations so it utilizes the maximum bottom canvas area
	inputFieldH := float32(iw.Height - iw.TitleBarHeight) - 30
>>>>>>> parent of 0df1919 (fix bug)

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

<<<<<<< HEAD
func (iw *InputWindow) PostUpdate() {
	if !glob.CursorLock {
		iw.holdingCursor = false
	}
}
=======
func (iw *InputWindow) PostUpdate() {}
>>>>>>> parent of 0df1919 (fix bug)

func (iw *InputWindow) IsActive() bool {
	return iw.Active
}

func (iw *InputWindow) SetActive(active bool) {
	iw.Active = active
<<<<<<< HEAD
}
=======
}
>>>>>>> parent of 0df1919 (fix bug)
