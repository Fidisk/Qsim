package main

import (
	comp "qsim/components"
	glob "qsim/globals"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type RenderWindow struct {
	Window
	Camera rl.Camera2D
	wComp  []comp.Component
}

func NewRenderWindow(x, y, width, height int32) *RenderWindow {
	rw := &RenderWindow{
		Window: *NewWindow(x, y, width, height),
		wComp:  nil,
	}
	rw.Camera = rl.Camera2D{
		Offset:   rl.NewVector2(0, 0), // will be set each frame
		Target:   rl.NewVector2(0, 0), // world point the camera looks at
		Rotation: 0,
		Zoom:     1.0,
	}
	rw.updateCameraOffset()
	return rw
}

// updateCameraOffset sets the camera offset to the center of the content area
func (rw *RenderWindow) updateCameraOffset() {
	contentWidth := float32(rw.Width)
	contentHeight := float32(rw.Height - rw.TitleBarHeight)
	rw.Camera.Offset = rl.NewVector2(contentWidth/2, contentHeight/2)
}

// GetWorldMouse converts the screen mouse position to world coordinates
func (rw *RenderWindow) GetWorldMouse() rl.Vector2 {
	return rl.GetScreenToWorld2D(rl.GetMousePosition(), rw.Camera)
}

func (rw *RenderWindow) Draw() {
	// 1. Draw the window itself (title bar, border, resize handle) in screen space
	rl.DrawRectangle(rw.X, rw.Y, rw.Width, rw.Height, rw.ColorBg)
	rl.DrawRectangle(rw.X, rw.Y, rw.Width, rw.TitleBarHeight, rw.ColorTitleBar)
	rl.DrawText(rw.Name, rw.X+5, rw.Y+5, 16, rw.ColorText)

	// Resize handle
	handleColor := rw.ColorResize
	if rw.cursorOnResize {
		handleColor = rw.ColorResizeHover
	}
	rl.DrawTriangle(
		rl.Vector2{X: float32(rw.X + rw.Width), Y: float32(rw.Y + rw.Height - 15)},
		rl.Vector2{X: float32(rw.X + rw.Width - 15), Y: float32(rw.Y + rw.Height)},
		rl.Vector2{X: float32(rw.X + rw.Width), Y: float32(rw.Y + rw.Height)},
		handleColor,
	)
	rl.DrawRectangleLines(rw.X, rw.Y, rw.Width, rw.Height, rw.ColorBorder)

	// 2. Activate camera and draw components in world space
	rl.BeginMode2D(rw.Camera)
	for _, c := range rw.wComp {
		c.Draw() // no arguments needed – components draw at their own world coords
	}
	rl.EndMode2D()
}

func (rw *RenderWindow) Update() {
	// Cursor locking logic (unchanged, but using the window's screen rect)
	if !glob.CursorAvailable && !rw.holdingCursor {
		// If cursor is locked globally and we aren't already holding it, skip further updates?
		// You might still want to update the window geometry (resize/drag) – decide accordingly.
		// But for simplicity, we return early if we don't have focus and not holding.
		return
	}

	mousePos := rl.GetMousePosition()
	windowRect := rl.Rectangle{
		X: float32(rw.X), Y: float32(rw.Y),
		Width: float32(rw.Width), Height: float32(rw.Height),
	}

	// Track if mouse is over this window
	if rl.CheckCollisionPointRec(mousePos, windowRect) {
		rw.holdingCursor = true
		glob.CursorAvailable = false
	} else if glob.CursorAvailable {
		rw.holdingCursor = false
	}

	if rw.holdingCursor && rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		rw.activate = true
	}

	// Handle window resize & drag (in screen space)
	rw.handleResize(mousePos)
	if !rw.IsResizing {
		rw.handleDrag(mousePos)
	}

	// Update camera offset (content area may have changed after resize)
	rw.updateCameraOffset()

	// Convert mouse to world coordinates for components
	worldMouse := rw.GetWorldMouse()

	// Update each component, passing world mouse
	for _, c := range rw.wComp {
		// New component interface: Update(worldMouse rl.Vector2)
		c.Update(worldMouse)
	}
}
