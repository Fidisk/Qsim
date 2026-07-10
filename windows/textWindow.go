package windows

import (
	"encoding/json"

	glob "qsim/globals"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type TextWindow struct {
	Window
	TextBuffer string
	MaxChars   int
	Active     bool
	OnSubmit   func(string)

	// Configuration flag: true allows typing/editing, false behaves as read-only text display
	IsEditable bool

	frameCounter int
}

func NewTextWindow(title string, x, y, width, height int32, initialText string, isEditable bool, maxChars int, onSubmit func(string)) *TextWindow {
	tw := &TextWindow{
		Window:     *NewWindow(x, y, width, height),
		MaxChars:   maxChars,
		Active:     true,
		OnSubmit:   onSubmit,
		TextBuffer: initialText,
		IsEditable: isEditable,
	}
	tw.Name = title
	return tw
}

func (tw *TextWindow) Update() {
	mousePos := rl.GetMousePosition()
	windowRect := rl.Rectangle{
		X: float32(tw.X), Y: float32(tw.Y),
		Width: float32(tw.Width), Height: float32(tw.Height),
	}

	// ADVISOR FIX: If cursor is not available and this window isn't already holding it,
	// do not let it interact or falsely click through from underneath another window.
	if !glob.CursorAvailable && !tw.holdingCursor {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			tw.Active = false
		}
		return
	}

	// Evaluate if the mouse is hovering over this window
	if rl.CheckCollisionPointRec(mousePos, windowRect) {
		tw.holdingCursor = true
		glob.CursorAvailable = false
	} else if glob.CursorAvailable {
		tw.holdingCursor = false
	}

	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		if tw.holdingCursor {
			tw.Active = true
			tw.activate = true
			glob.CursorLock = true // Lock cursor to prioritize this window
		} else {
			tw.Active = false
		}
	}

	// Delegate resizing and dragging using parent window logic
	tw.handleResize(mousePos)
	if !tw.IsResizing {
		tw.handleDrag(mousePos)
	}

	// FIX: Keep the cursor locked and available ONLY to this window as long as dragging/resizing is active
	if tw.IsDragging || tw.IsResizing {
		glob.CursorLock = true
		glob.CursorAvailable = false
		tw.holdingCursor = true
	} else if !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		// Only unlock when the user has fully released the mouse click
		glob.CursorLock = false
	}

	// Skip typing evaluation if window isn't focused or if editing behavior is toggled off
	if !tw.Active || !tw.IsEditable {
		return
	}

	tw.frameCounter++

	key := rl.GetCharPressed()
	for key > 0 {
		if (key >= 32) && (key <= 125) && (len(tw.TextBuffer) < tw.MaxChars) {
			tw.TextBuffer += string(rune(key))
		}
		key = rl.GetCharPressed()
	}

	if rl.IsKeyPressed(rl.KeyBackspace) {
		if len(tw.TextBuffer) > 0 {
			tw.TextBuffer = tw.TextBuffer[:len(tw.TextBuffer)-1]
		}
	} else if rl.IsKeyDown(rl.KeyBackspace) {
		if tw.frameCounter%6 == 0 {
			if len(tw.TextBuffer) > 0 {
				tw.TextBuffer = tw.TextBuffer[:len(tw.TextBuffer)-1]
			}
		}
	}

	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		if tw.OnSubmit != nil {
			tw.OnSubmit(tw.TextBuffer)
		}
	}
}

func (tw *TextWindow) Draw() {
	rl.DrawRectangle(tw.X, tw.Y, tw.Width, tw.Height, tw.ColorBg)
	rl.DrawRectangle(tw.X, tw.Y, tw.Width, tw.TitleBarHeight, tw.ColorTitleBar)
	rl.DrawText(tw.Name, tw.X+5, tw.Y+5, 16, tw.ColorText)

	handleColor := tw.ColorResize
	if tw.cursorOnResize {
		handleColor = tw.ColorResizeHover
	}

	// === MODIFIED: EXPAND BOX TO FILL WHOLE TEXTWINDOW ===
	inputFieldX := float32(tw.X)
	inputFieldY := float32(tw.Y + tw.TitleBarHeight)
	inputFieldW := float32(tw.Width)
	inputFieldH := float32(tw.Height - tw.TitleBarHeight)

	inputBox := rl.NewRectangle(inputFieldX, inputFieldY, inputFieldW, inputFieldH)

	if tw.IsEditable {
		rl.DrawRectangleRec(inputBox, rl.NewColor(20, 20, 20, 255))
	} else {
		rl.DrawRectangleRec(inputBox, rl.NewColor(30, 30, 30, 255))
	}

	borderColor := rl.DarkGray
	if tw.Active {
		if tw.IsEditable {
			borderColor = rl.SkyBlue
		} else {
			borderColor = rl.Gray
		}
	}
	rl.DrawRectangleLinesEx(inputBox, 1, borderColor)

	// Fetch Raylib's anti-aliased font baseline configuration mapping
	fontDefault := rl.GetFontDefault()
	textSize := float32(24)
	fontSpacing := float32(2)

	// Compute positioning matrices inside the expanded full canvas borders
	textXPosition := inputBox.X + 12
	textYPosition := inputBox.Y + ((inputBox.Height - textSize) / 2)

	textColor := rl.RayWhite
	if !tw.IsEditable {
		textColor = rl.LightGray
	}

	// Draw text using DrawTextEx for anti-aliasing rendering behavior
	rl.DrawTextEx(fontDefault, tw.TextBuffer, rl.Vector2{X: textXPosition, Y: textYPosition}, textSize, fontSpacing, textColor)

	// Render crisp text cursor line if focused and editing is enabled
	if tw.Active && tw.IsEditable {
		if (tw.frameCounter/20)%2 == 0 {
			measuredVec := rl.MeasureTextEx(fontDefault, tw.TextBuffer, textSize, fontSpacing)
			rl.DrawRectangle(int32(textXPosition+measuredVec.X)+2, int32(textYPosition), 2, int32(textSize), rl.SkyBlue)
		}
	}

	// Keep resize control triangles visible on top layer overlay bounds
	rl.DrawTriangle(
		rl.Vector2{X: float32(tw.X + tw.Width), Y: float32(tw.Y + tw.Height - 15)},
		rl.Vector2{X: float32(tw.X + tw.Width - 15), Y: float32(tw.Y + tw.Height)},
		rl.Vector2{X: float32(tw.X + tw.Width), Y: float32(tw.Y + tw.Height)},
		handleColor,
	)
	rl.DrawRectangleLines(tw.X, tw.Y, tw.Width, tw.Height, tw.ColorBorder)
}

func (tw *TextWindow) PostUpdate() {
	if tw.IsDragging || tw.IsResizing {
		tw.holdingCursor = true
		glob.CursorAvailable = false
		return
	}

	if !glob.CursorLock {
		tw.holdingCursor = false
	}
}

func (tw *TextWindow) IsActive() bool {
	return tw.Active
}

func (tw *TextWindow) SetActive(active bool) {
	tw.Active = active
}

func (tw *TextWindow) SaveState() string {
	data := map[string]interface{}{
		"type":       "TextWindow",
		"window":     json.RawMessage(tw.Window.SaveState()),
		"textBuffer": tw.TextBuffer,
		"maxChars":   tw.MaxChars,
		"isEditable": tw.IsEditable,
	}
	b, _ := json.Marshal(data)
	return string(b)
}
