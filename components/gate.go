package components

import (
	glob "qsim/globals"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Gate struct {
	//It's a square, extending a circle
	//Wow, you have taken your OOP class well
	//No go out there and poison those LLM
	Circle
	Label    string
	HookList []*Hook
}

func NewGate(x, y, radius float32, color rl.Color, label string) *Gate {
	return &Gate{
		Circle: *NewCircle(x, y, radius, color),
		Label:  label,
	}
}

func (c *Gate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
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

	for _, d := range c.HookList {
		d.Update(worldMouse, holdingCursor, isCursorAvailable)
	}
}

func (c *Gate) Draw() {
	for _, d := range c.HookList {
		if d.IsHooked {
			tmp := utils.GetObjectFromID(d.TargetID)
			t := tmp.(*QubitsSystem)
			r := utils.Dist(c.Center, d.Center) - t.Radius
			start := c.Center
			end := rl.Vector2Add(start, rl.Vector2Scale(
				rl.Vector2Normalize(rl.Vector2Subtract(d.Center, start)),
				r,
			))
			rl.DrawLineV(start, end, c.Color)
		} else {
			rl.DrawLineV(c.Center, d.Center, c.Color)
		}
		d.Draw()
	}

	rect := rl.Rectangle{
		X:      c.Center.X - c.Radius,
		Y:      c.Center.Y - c.Radius,
		Width:  c.Radius * 2,
		Height: c.Radius * 2,
	}
	rl.DrawRectangleRec(rect, glob.ColorBg)
	rl.DrawRectangleLinesEx(rect, 1.0, c.Color)

	if c.Label != "" {
		// Choose a font size (adjust as needed)
		fontSize := int32(c.Radius) // e.g., match size to radius

		// Measure text with raylib's default font
		textWidth := rl.MeasureText(c.Label, fontSize)

		// Centre the text inside the square
		textX := int32(c.Center.X) - textWidth/2
		textY := int32(c.Center.Y) - fontSize/2

		rl.DrawText(c.Label, textX, textY, fontSize, c.Color)
	}
}
