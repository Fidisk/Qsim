//go:build !js || !wasm

package main

import (
	glob "qsim/globals"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	edgeThickness = 6 // pixel width of the invisible resize border
)

// runMainLoop runs the game loop on desktop platforms.
func runMainLoop(update func()) {
	for !rl.WindowShouldClose() {
		update()
	}
}

// setWindowFlags makes the desktop OS window resizable and enables
// anti-aliasing for smoother lines and shapes.
func setWindowFlags() {
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
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
