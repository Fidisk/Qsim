package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// InitMainWindow creates an undecorated, manually resizable window.
// Window flags are platform-specific: desktop gets a resizable OS window,
// the web build pins the canvas to the given resolution.
func InitMainWindow(width, height int32, title string) {
	setWindowFlags()
	rl.InitWindow(int32(width), int32(height), title)
}
