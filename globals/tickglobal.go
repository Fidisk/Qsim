package globals

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var CursorAvailable bool
var MainWindowHeight int
var MainWindowWidth int
var WorldMouse rl.Vector2

func Refresh() {
	if !CursorLock {
		CursorAvailable = true
	}

	MainWindowHeight = rl.GetScreenHeight()
	MainWindowWidth = rl.GetScreenWidth()
}
