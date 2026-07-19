//go:build js && wasm

package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// runMainLoop hands the game loop to the browser. rl.SetMainLoop never
// returns on wasm (WindowShouldClose is unsupported on the web platform).
func runMainLoop(update func()) {
	rl.SetMainLoop(update)
}

// setWindowFlags is a no-op on the web: the canvas must stay at the exact
// InitWindow resolution (1600x900). A resizable canvas would track the
// browser window size and break the in-app scaling, which reads the screen
// size every frame.
func setWindowFlags() {}

// UpdateMainWindow is a no-op on the web: there is no OS window to resize,
// and calling rl.SetWindowSize would change the canvas resolution the
// layout depends on.
func UpdateMainWindow() {}
