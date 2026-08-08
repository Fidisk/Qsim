package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	glob "qsim/globals"
)

type ToggleButton struct {
	*Circle
	Width, Height float32
	Label         string
	FontSize      int32
	OnClick       func()      // called when toggled ON
	OffClick      func()      // called when toggled OFF
	toggled       func() bool // current state: true = pressed/down
	// OnPress is called the moment the press begins, so drag-out buttons can
	// activate their mode immediately (button stays held down and the spawn
	// ghost follows the mouse while dragging to the canvas).
	OnPress func()
	// OnDragOut is called with the screen position when the button is pressed
	// and released somewhere else (drag-out). When set, the press is not
	// cancelled when the mouse leaves the button, so the button can be
	// dragged out of a toolbar; leave nil to cancel the press on mouse-leave.
	OnDragOut func(screenPos rl.Vector2)
	// Tooltip is shown in the floating tooltip window while the button is
	// hovered. Leave empty for no tooltip.
	Tooltip string
	held    bool // mouse is currently pressed inside the button
	hovered bool
	// wasToggled records the state at press time, so clicking an already
	// selected drag-out button unselects it instead of keeping it on.
	wasToggled bool
}

func NewToggleButton(x, y, width, height float32, color rl.Color, label string, state func() bool, fontsize int32, onClick, offClick func()) *ToggleButton {
	radius := max(width, height) / 2
	tb := &ToggleButton{
		Circle:   NewCircle(x, y, radius, color),
		Width:    width,
		Height:   height,
		Label:    label,
		FontSize: fontsize,
		OnClick:  onClick,
		OffClick: offClick,
		toggled:  state,
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

	over := rl.CheckCollisionPointRec(worldMouse, rect)
	tb.hovered = over && !tb.held

	if tb.hovered && tb.Tooltip != "" {
		glob.TooltipText = tb.Tooltip
	}

	if over {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (tb.held || *isCursorAvailable) {
			tb.held = true
			*isCursorAvailable = false
			// Capture the state before OnPress flips it, so a click on an
			// already selected button can unselect it on release.
			tb.wasToggled = tb.toggled()
			if tb.OnPress != nil {
				tb.OnPress()
			}
		}
	}

	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		if tb.held {
			// Released inside the button: plain toggle buttons flip their
			// state; drag-out buttons already activated on press, so a simple
			// click leaves the mode on — unless the button was already
			// selected, in which case the click unselects it.
			if over {
				if tb.OnDragOut == nil {
					if !tb.toggled() {
						if tb.OnClick != nil {
							tb.OnClick()
						}
					} else {
						if tb.OffClick != nil {
							tb.OffClick()
						}
					}
				} else if tb.wasToggled {
					if tb.OffClick != nil {
						tb.OffClick()
					}
				}
			} else if tb.OnDragOut != nil {
				tb.OnDragOut(rl.GetMousePosition())
			}
			tb.held = false
			*isCursorAvailable = true
		}
	}

	// Cancel press if mouse leaves the button while held — unless the button
	// supports drag-out, which keeps the press alive until release.
	if tb.held && !over && tb.OnDragOut == nil {
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

	if tb.toggled() {
		// Depressed look
		x += togglePressOffset
		y += togglePressOffset

		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), darken(tb.Color, 0.6))
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, toggleBevel), darken(tb.Color, 0.3))
		rl.DrawRectangleRec(rl.NewRectangle(x, y, toggleBevel, h), darken(tb.Color, 0.3))
		rl.DrawRectangleRec(rl.NewRectangle(x+w-toggleBevel, y, toggleBevel, h), lighten(tb.Color, 0.3))
		rl.DrawRectangleRec(rl.NewRectangle(x, y+h-toggleBevel, w, toggleBevel), lighten(tb.Color, 0.3))
	} else if tb.hovered {
		// Hovered: subtle darken with lower factor
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), darken(tb.Color, 0.15))
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, toggleBevel), lighten(tb.Color, 0.4))
		rl.DrawRectangleRec(rl.NewRectangle(x, y, toggleBevel, h), lighten(tb.Color, 0.4))
		rl.DrawRectangleRec(rl.NewRectangle(x+w-toggleBevel, y, toggleBevel, h), darken(tb.Color, 0.45))
		rl.DrawRectangleRec(rl.NewRectangle(x, y+h-toggleBevel, w, toggleBevel), darken(tb.Color, 0.45))
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
	if tb.toggled() {
		textX += int32(togglePressOffset)
		textY += int32(togglePressOffset)
	}
	rl.DrawText(tb.Label, textX, textY, tb.FontSize, rl.White)
}
