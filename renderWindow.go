package main

import (
	comp "qsim/components"
	glob "qsim/globals"

	conf "qsim/config"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type RenderWindow struct {
	Window
	Camera rl.Camera2D
	wComp  []comp.Component

	// Panning state
	isPanning      bool
	panStartMouse  rl.Vector2 // world coordinate on press
	panStartTarget rl.Vector2 // camera target on press
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

func (rw *RenderWindow) GetContentRect() rl.Rectangle {
	return rl.Rectangle{
		X:      float32(rw.X),
		Y:      float32(rw.Y + rw.TitleBarHeight),
		Width:  float32(rw.Width),
		Height: float32(rw.Height - rw.TitleBarHeight),
	}
}

func (rw *RenderWindow) updateCameraOffset() {
	contentWidth := float32(rw.Width)
	contentHeight := float32(rw.Height - rw.TitleBarHeight)
	rw.Camera.Offset = rl.NewVector2(
		float32(rw.X)+contentWidth/2,
		float32(rw.Y+rw.TitleBarHeight)+contentHeight/2,
	)
}

// GetWorldMouse converts the screen mouse position to world coordinates
func (rw *RenderWindow) GetWorldMouse() rl.Vector2 {
	return rl.GetScreenToWorld2D(rl.GetMousePosition(), rw.Camera)
}
func (rw *RenderWindow) Draw() {
	// 1. Draw window decorations in screen space (unclipped)
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

	// 2. Activate scissor to clip content to the window's content area
	contentRect := rw.GetContentRect()
	rl.BeginScissorMode(
		int32(contentRect.X),
		int32(contentRect.Y),
		int32(contentRect.Width),
		int32(contentRect.Height),
	)

	// 3. Draw components in world space (camera transforms, scissor clips)
	rl.BeginMode2D(rw.Camera)
	for _, c := range rw.wComp {
		c.Draw()
	}
	rl.EndMode2D()

	// 4. Disable scissor
	rl.EndScissorMode()
}

func (rw *RenderWindow) Update() {
	worldMouse := rw.GetWorldMouse()
	for _, c := range rw.wComp {
		c.Update(worldMouse, rw.holdingCursor)
	}

	// Cursor locking logic fixed: Check if THIS window is holding it to release it
	if !glob.CursorAvailable && !rw.holdingCursor {
		return
	}

	mousePos := rl.GetMousePosition()
	windowRect := rl.Rectangle{
		X: float32(rw.X), Y: float32(rw.Y),
		Width: float32(rw.Width), Height: float32(rw.Height),
	}

	if rl.CheckCollisionPointRec(mousePos, windowRect) {
		rw.holdingCursor = true
		glob.CursorAvailable = false
	} else if glob.CursorAvailable {
		rw.holdingCursor = false
	}

	if rw.holdingCursor && rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		rw.activate = true
	}

	// Window resize/drag in screen space
	rw.handleResize(mousePos)
	if !rw.IsResizing {
		rw.handleDrag(mousePos)
	}

	// Update camera offset after possible resize/move
	rw.updateCameraOffset()

	// --- Pan and Zoom ---
	contentRect := rw.GetContentRect()
	if rw.holdingCursor && rl.CheckCollisionPointRec(mousePos, contentRect) {

		// Middle‑mouse panning (FIXED: Using Screen Space to avoid feedback loops)
		if rl.IsMouseButtonPressed(rl.MouseButtonMiddle) {
			rw.isPanning = true
			rw.panStartMouse = rl.GetMousePosition() // FIX: Store SCREEN position, not world
			rw.panStartTarget = rw.Camera.Target
		}
		if rl.IsMouseButtonReleased(rl.MouseButtonMiddle) {
			rw.isPanning = false
		}
		if rw.isPanning {
			currentMouse := rl.GetMousePosition()
			// Calculate delta in screen pixels
			deltaScreen := rl.Vector2Subtract(rw.panStartMouse, currentMouse)
			// FIX: Convert screen delta to world delta by dividing by zoom
			deltaWorld := rl.Vector2Scale(deltaScreen, 1.0/rw.Camera.Zoom)
			rw.Camera.Target = rl.Vector2Add(rw.panStartTarget, deltaWorld)
		}

		// Mouse‑wheel zoom (towards cursor) (FIXED: Order of operations)
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			// 1. Get the world position under mouse BEFORE zooming
			worldBefore := rw.GetWorldMouse()

			oldZoom := rw.Camera.Zoom
			newZoom := oldZoom + wheel*0.1
			if newZoom < conf.MinZoom {
				newZoom = conf.MinZoom
			} else if newZoom > conf.MaxZoom {
				newZoom = conf.MaxZoom
			}

			// 2. Apply the zoom factor
			rw.Camera.Zoom = newZoom

			// 3. Get the world position under mouse AFTER zooming
			worldAfter := rw.GetWorldMouse()

			// 4. Adjust target by the true difference to anchor zoom to the cursor
			rw.Camera.Target = rl.Vector2Add(rw.Camera.Target,
				rl.Vector2Subtract(worldBefore, worldAfter))
		}
	}

	//fmt.Println(rw.Camera)

	// Now update components with world mouse (only if mouse in content area)
	//if rw.holdingCursor && rl.CheckCollisionPointRec(mousePos, contentRect) {

	//}
}
