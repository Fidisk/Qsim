package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

func InitMainWindow(width, height int32, title string) {
	setWindowFlags()
	rl.InitWindow(int32(width), int32(height), title)
}
