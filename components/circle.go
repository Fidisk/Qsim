package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	glob "qsim/globals"
)

// Circle is a draggable circle in world space.
type Circle struct {
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

func (c *Circle) Update() {
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		if rl.CheckCollisionPointCircle(glob.WorldMouse, c.Center, c.Radius) {
			c.dragging = true
			c.offset = rl.Vector2Subtract(c.Center, glob.WorldMouse)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		c.dragging = false
	}
	if c.dragging {
		c.Center = rl.Vector2Add(glob.WorldMouse, c.offset)
	}
}

func (c *Circle) Draw() {
	rl.DrawCircleV(c.Center, c.Radius, c.Color)
	// optional outline
	// rl.DrawCircleLines(int32(c.Center.X), int32(c.Center.Y), c.Radius, rl.Black)
}
