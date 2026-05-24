package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	glob "qsim/globals"
)

// Window is a draggable and resizable panel
type Window struct {
	Name string

	X, Y, Width, Height int32
	TitleBarHeight      int32

	IsDragging               bool
	IsResizing               bool
	DragOffsetX, DragOffsetY int32
	ResizeMinW, ResizeMinH   int32

	// Visual
	ColorBg          rl.Color
	ColorTitleBar    rl.Color
	ColorText        rl.Color
	ColorResize      rl.Color
	ColorResizeHover rl.Color // highlight color when mouse is over the handle

	// Cursor state to avoid calling SetMouseCursor every frame if unchanged
	cursorOnResize bool

	//Check if window is using cursor for anything
	holdingCursor bool

	activate bool

	ColorBorder rl.Color
}

// NewWindow creates a new Window with default styling
func NewWindow(x, y, width, height int32) *Window {
	return &Window{
		Name: "Panel",
		X:    x, Y: y, Width: width, Height: height,
		TitleBarHeight:   25,
		ResizeMinW:       100,
		ResizeMinH:       25,
		ColorBg:          rl.NewColor(50, 50, 50, 255),
		ColorTitleBar:    rl.NewColor(70, 70, 70, 255),
		ColorText:        rl.White,
		ColorResize:      rl.Black,
		ColorResizeHover: rl.White,

		cursorOnResize: false,
		holdingCursor:  false,
		activate:       false,

		ColorBorder: rl.NewColor(100, 100, 100, 255),
	}
}

// IsActive returns the current activation state
func (w *Window) IsActive() bool {
	return w.activate
}

// SetActive sets the activation state to true
func (w *Window) SetActive(b bool) {
	w.activate = b
}

// Update handles resize (and cursor) and delegates dragging
func (w *Window) Update() {
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
}

func (w *Window) handleResize(mousePos rl.Vector2) {
	mouseX := int32(mousePos.X)
	mouseY := int32(mousePos.Y)

	// --- Resize handle area ---
	resizeRect := rl.Rectangle{
		X:      float32(w.X + w.Width - 15),
		Y:      float32(w.Y + w.Height - 15),
		Width:  15,
		Height: 15,
	}
	hoverResize := rl.CheckCollisionPointRec(mousePos, resizeRect)

	// --- Cursor management ---
	if hoverResize && !w.cursorOnResize {
		rl.SetMouseCursor(rl.MouseCursorResizeNESW)
		w.cursorOnResize = true
	} else if !hoverResize && w.cursorOnResize {
		rl.SetMouseCursor(rl.MouseCursorDefault)
		w.cursorOnResize = false
	}

	// --- Start resizing ---
	if hoverResize && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {

		w.IsResizing = true
		glob.CursorLock = true
		w.activate = true
		w.DragOffsetX = mouseX - (w.X + w.Width)
		w.DragOffsetY = mouseY - (w.Y + w.Height)
	}

	// --- Continue resizing ---
	if w.IsResizing {
		if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			newW := mouseX - w.X - w.DragOffsetX
			newH := mouseY - w.Y - w.DragOffsetY
			if newW < w.ResizeMinW {
				newW = w.ResizeMinW
			}
			if newH < w.ResizeMinH {
				newH = w.ResizeMinH
			}
			w.Width = newW
			w.Height = newH
		} else {
			w.IsResizing = false
			glob.CursorLock = false
		} // skip dragging while resizing
	}

	// --- Dragging is handled in its own method --

	w.Height = max(0, w.Height)
	w.Width = max(0, w.Width)

	w.Height = min(w.Height, int32(glob.MainWindowHeight))
	w.Width = min(w.Width, int32(glob.MainWindowWidth))

	w.Height = min(w.Height, int32(glob.MainWindowHeight)-w.Y)
	w.Width = min(w.Width, int32(glob.MainWindowWidth)-w.X)
}

// handleDrag manages the title‑bar dragging logic
func (w *Window) handleDrag(mousePos rl.Vector2) {
	mouseX := int32(mousePos.X)
	mouseY := int32(mousePos.Y)

	titleRect := rl.Rectangle{
		X:      float32(w.X),
		Y:      float32(w.Y),
		Width:  float32(w.Width),
		Height: float32(w.TitleBarHeight),
	}

	if rl.CheckCollisionPointRec(mousePos, titleRect) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			w.IsDragging = true
			glob.CursorLock = true
			w.activate = true
			w.DragOffsetX = mouseX - w.X
			w.DragOffsetY = mouseY - w.Y
		}
	}

	if w.IsDragging {
		if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			w.X = mouseX - w.DragOffsetX
			w.Y = mouseY - w.DragOffsetY
		} else {
			w.IsDragging = false
			glob.CursorLock = false
		}
	}

	w.X = max(0, w.X)
	w.Y = max(0, w.Y)

	w.Y = min(w.Y, int32(glob.MainWindowHeight)-w.Height)
	w.X = min(w.X, int32(glob.MainWindowWidth)-w.Width)
}

// Draw renders the window
func (w *Window) Draw() {
	// Main body
	rl.DrawRectangle(w.X, w.Y, w.Width, w.Height, w.ColorBg)
	// Title bar
	rl.DrawRectangle(w.X, w.Y, w.Width, w.TitleBarHeight, w.ColorTitleBar)
	// Title text (you can customise this)
	rl.DrawText(w.Name, w.X+5, w.Y+5, 16, w.ColorText)

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

func (w *Window) PostUpdate() {
	if !glob.CursorLock {
		w.holdingCursor = false
	}
}
