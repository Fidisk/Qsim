package components

import rl "github.com/gen2brain/raylib-go/raylib"

type Component interface {
	Draw()
	Update(rl.Vector2, bool, *bool)
	PostUpdate()
}

type WindowComponent struct {
	Component
	HoldingCursor bool
}
