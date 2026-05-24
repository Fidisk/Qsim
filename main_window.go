package main

import (
	glob "qsim/globals"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	edgeThickness = 6 // pixel width of the invisible resize border
)

// InitMainWindow creates an undecorated, manually resizable window
func InitMainWindow(width, height int32, title string) {
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(int32(width), int32(height), title)
}

// UpdateMainWindow checks for edge dragging and resizes the OS window accordingly
func UpdateMainWindow() {
	if !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		return
	}

	if !glob.CursorAvailable {
		return
	}

	mouse := rl.GetMousePosition()
	mx := int32(mouse.X)
	my := int32(mouse.Y)
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	newW, newH := sw, sh

	// Right edge
	if mx >= sw-edgeThickness && mx <= sw {
		newW = mx
	}
	// Bottom edge
	if my >= sh-edgeThickness && my <= sh {
		newH = my
	}

	// Only change if size actually differs (avoids unnecessary calls)
	if newW != sw || newH != sh {
		rl.SetWindowSize(int(newW), int(newH))
	}
}
