package components

import (
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Hook struct {
	Circle
	IsHooked bool
	ID       int32
	TargetID int32
}

func NewHook(x, y, radius float32, color rl.Color) *Hook {
	tmp := Hook{
		Circle: *NewCircle(x, y, radius, color),
	}
	tmp.ID = utils.GenerateID(&tmp)
	return &tmp
}

func (c *Hook) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if c.IsHooked {
		tmp := utils.GetObjectFromID(c.TargetID)
		t := tmp.(*QubitsSystem)
		c.Center = t.Center
		return
	}

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
		c.holdingCursor = false
		*isCursorAvailable = true
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
	}
}

func (c *Hook) Draw() {
	if c.IsHooked {
		return
	}
	rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)
}
