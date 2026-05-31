package components

import rl "github.com/gen2brain/raylib-go/raylib"

type Component interface {
	Draw()
	Update(rl.Vector2, bool, *bool)
	PostUpdate()
	IsHoldingCursor() bool
}

type WindowComponent struct {
	Component
	holdingCursor bool
}

func (c *WindowComponent) IsHoldingCursor() bool {
	return c.holdingCursor
}
