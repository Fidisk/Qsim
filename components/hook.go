package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Hook struct {
	Circle
}

func NewHook(x, y, radius float32, color rl.Color) *Hook {
	return &Hook{
		Circle: *NewCircle(x, y, radius, color),
	}
}

func (c *Hook) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			c.dragging = true
			*isCursorAvailable = false
			c.holdingCursor = true
			c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		c.dragging = false
		c.holdingCursor = true
		*isCursorAvailable = true
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
	}
}

func (c *Hook) Draw() {
	rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)
}
