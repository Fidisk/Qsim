package main

import (
	comp "qsim/components"

	glob "qsim/globals"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type RenderWindow struct {
	Window
	wComp []comp.Component
}

func NewRenderWindow(x, y, width, height int32) *RenderWindow {
	return &RenderWindow{
		Window: *NewWindow(x, y, width, height),
		wComp:  nil,
	}
}

func (w *RenderWindow) Draw() {
	// Main body
	rl.DrawRectangle(w.X, w.Y, w.Width, w.Height, w.ColorBg)
	// Title bar
	rl.DrawRectangle(w.X, w.Y, w.Width, w.TitleBarHeight, w.ColorTitleBar)
	// Title text (you can customise this)
	rl.DrawText(w.Name, w.X+5, w.Y+5, 16, w.ColorText)

	for _, d := range w.wComp {
		d.Draw(
			float32(w.X),
			float32(w.Y)+float32(w.TitleBarHeight),
			float32(w.Width),
			float32(w.Height)-float32(w.TitleBarHeight),
		)
	}

	// Resize handle – use hover color if mouse is over it
	handleColor := w.ColorResize
	if w.cursorOnResize {
		handleColor = w.ColorResizeHover
	}
	rl.DrawTriangle(
		rl.Vector2{X: float32(w.X + w.Width), Y: float32(w.Y + w.Height - 15)},
		rl.Vector2{X: float32(w.X + w.Width - 15), Y: float32(w.Y + w.Height)},
		rl.Vector2{X: float32(w.X + w.Width), Y: float32(w.Y + w.Height)},
		handleColor,
	)

	rl.DrawRectangleLines(w.X, w.Y, w.Width, w.Height, w.ColorBorder)
}

func (w *RenderWindow) Update() {
	if !glob.CursorAvailable && !w.holdingCursor {
		return
	}

	mousePos := rl.GetMousePosition()

	if rl.CheckCollisionPointRec(mousePos, rl.Rectangle{X: float32(w.X), Y: float32(w.Y), Width: float32(w.Width), Height: float32(w.Height)}) {
		w.holdingCursor = true
		glob.CursorAvailable = false
	} else if glob.CursorAvailable {
		w.holdingCursor = false
	}

	if w.holdingCursor && rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		w.activate = true
	}

	w.handleResize(mousePos)
	if !w.IsResizing {
		w.handleDrag(mousePos)
	}

	for _, d := range w.wComp {
		d.Update(
			float32(w.X),
			float32(w.Y)+float32(w.TitleBarHeight),
			float32(w.Width),
			float32(w.Height)-float32(w.TitleBarHeight),
		)
	}
}
