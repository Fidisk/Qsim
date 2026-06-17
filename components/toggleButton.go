package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type ToggleButton struct {
	*Circle
	Width, Height float32
	Label         string
	FontSize      int32
	OnClick       func() // called when toggled ON
	OffClick      func() // called when toggled OFF
	toggled       *bool  // current state: true = pressed/down
	held          bool   // mouse is currently pressed inside the button
}

func NewToggleButton(x, y, width, height float32, color rl.Color, label string, fontsize int32, onClick, offClick func()) *ToggleButton {
	radius := max(width, height) / 2
	tb := &ToggleButton{
		Circle:   NewCircle(x, y, radius, color),
		Width:    width,
		Height:   height,
		Label:    label,
		FontSize: fontsize,
		OnClick:  onClick,
		OffClick: offClick,
	}
	tb.SetWeight(0)
	return tb
}

// Update handles mouse interaction and toggles state on a complete click.
func (tb *ToggleButton) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	rect := rl.NewRectangle(
		tb.Center.X-tb.Width/2,
		tb.Center.Y-tb.Height/2,
		tb.Width,
		tb.Height,
	)

	if rl.CheckCollisionPointRec(worldMouse, rect) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (tb.held || *isCursorAvailable) {
			tb.held = true
			*isCursorAvailable = false
		}
	}

	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		if tb.held {
			// Toggle only if released inside the button
			if rl.CheckCollisionPointRec(worldMouse, rect) {
				*tb.toggled = !*tb.toggled
				if *tb.toggled {
					if tb.OnClick != nil {
						tb.OnClick()
					}
				} else {
					if tb.OffClick != nil {
						tb.OffClick()
					}
				}
			}
			tb.held = false
			*isCursorAvailable = true
		}
	}

	// Cancel press if mouse leaves the button while held
	if tb.held && !rl.CheckCollisionPointRec(worldMouse, rect) {
		tb.held = false
		*isCursorAvailable = true
	}
}

var toggleBevel float32 = 3.0
var togglePressOffset float32 = 1.5

// Draw renders the toggle button with a raised or depressed style based on toggled state.
func (tb *ToggleButton) Draw() {
	x := tb.Center.X - tb.Width/2
	y := tb.Center.Y - tb.Height/2
	w := tb.Width
	h := tb.Height

	if *tb.toggled {
		// Depressed look
		x += togglePressOffset
		y += togglePressOffset

		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), darken(tb.Color, 0.6))
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, toggleBevel), darken(tb.Color, 0.3))
		rl.DrawRectangleRec(rl.NewRectangle(x, y, toggleBevel, h), darken(tb.Color, 0.3))
		rl.DrawRectangleRec(rl.NewRectangle(x+w-toggleBevel, y, toggleBevel, h), lighten(tb.Color, 0.3))
		rl.DrawRectangleRec(rl.NewRectangle(x, y+h-toggleBevel, w, toggleBevel), lighten(tb.Color, 0.3))
	} else {
		// Raised look
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), tb.Color)
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, toggleBevel), lighten(tb.Color, 0.5))
		rl.DrawRectangleRec(rl.NewRectangle(x, y, toggleBevel, h), lighten(tb.Color, 0.5))
		rl.DrawRectangleRec(rl.NewRectangle(x+w-toggleBevel, y, toggleBevel, h), darken(tb.Color, 0.5))
		rl.DrawRectangleRec(rl.NewRectangle(x, y+h-toggleBevel, w, toggleBevel), darken(tb.Color, 0.5))
	}

	//rl.DrawRectangleLines(int32(x), int32(y), int32(w), int32(h), rl.Black)

	textWidth := rl.MeasureText(tb.Label, tb.FontSize)
	textX := int32(tb.Center.X) - textWidth/2
	textY := int32(tb.Center.Y) - tb.FontSize/2
	if *tb.toggled {
		textX += int32(togglePressOffset)
		textY += int32(togglePressOffset)
	}
	rl.DrawText(tb.Label, textX, textY, tb.FontSize, rl.White)
}
