package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Circle struct {
	WindowComponent
	Center   rl.Vector2
	Radius   float32
	Color    rl.Color
	dragging bool
	offset   rl.Vector2
}

func NewCircle(x, y, radius float32, color rl.Color) *Circle {
	return &Circle{
		Center: rl.NewVector2(x, y),
		Radius: radius,
		Color:  color,
	}
}

func (c *Circle) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor {
			c.dragging = true
			c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		c.dragging = false
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
	}
}

func (c *Circle) Draw() {
	rl.DrawCircleV(c.Center, c.Radius, c.Color)
}
