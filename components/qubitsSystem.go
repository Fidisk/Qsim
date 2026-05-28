package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type QubitsSystem struct {
	Circle
	QubitList []*Qubit
}

func NewQubitsSystem(x, y, radius float32, color rl.Color) *QubitsSystem {
	return &QubitsSystem{
		Circle:    *NewCircle(x, y, radius, color),
		QubitList: nil,
	}
}

func (c *QubitsSystem) Update(worldMouse rl.Vector2, holdingCursor bool) {
	for _, d := range c.QubitList {
		d.Update()
	}

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

func (c *QubitsSystem) Draw() {
	for _, d := range c.QubitList {
		d.Draw(c.Center, c.Radius)
	}

	rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)
}
