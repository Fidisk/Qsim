package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Button is a clickable rectangle with a label and an optional drag ability.
type Button struct {
	*Circle
	Width, Height float32
	Label         string
	FontSize      int32
	OnClick       func()
	held          bool
	hovered       bool
}

// NewButton creates a rectangular button.
func NewButton(x, y, width, height float32, color rl.Color, label string, fontSize int32, onClick func()) *Button {
	radius := max(width, height) / 2
	b := &Button{
		Circle:   NewCircle(x, y, radius, color),
		Width:    width,
		Height:   height,
		Label:    label,
		FontSize: fontSize,
		OnClick:  onClick,
	}
	b.SetWeight(0)
	return b
}

// Update handles mouse interaction, respecting cursor availability.
func (b *Button) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	rect := rl.NewRectangle(
		b.Center.X-b.Width/2,
		b.Center.Y-b.Height/2,
		b.Width,
		b.Height,
	)

	over := rl.CheckCollisionPointRec(worldMouse, rect)
	b.hovered = over && !b.held

	if over {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (b.held || *isCursorAvailable) {
			b.held = true
			*isCursorAvailable = false
		}
	}

	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		if b.held {
			// Fire OnClick only if released while still inside the button.
			if over {
				if b.OnClick != nil {
					b.OnClick()
				}
			}
			b.held = false
			*isCursorAvailable = true
		}
	}

	// Cancel press if mouse leaves the button while held.
	if b.held && !over {
		b.held = false
		*isCursorAvailable = true
	}
}

	var bevel float32 = 3.0 // thickness of the 3D edges
	var pressOffset float32 = 1.5

	func (b *Button) Draw() {
		// Base rectangle coordinates
		x := b.Center.X - b.Width/2
		y := b.Center.Y - b.Height/2
		w := b.Width
		h := b.Height

		if b.held {
			// Depressed: shift down/right and darken
			x += pressOffset
			y += pressOffset

			// Background
			rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), darken(b.Color, 0.6))
			// Top/left inner shadow (darker than background)
			rl.DrawRectangleRec(rl.NewRectangle(x, y, w, bevel), darken(b.Color, 0.3))
			rl.DrawRectangleRec(rl.NewRectangle(x, y, bevel, h), darken(b.Color, 0.3))
			// Bottom/right highlight (slightly lighter than background)
			rl.DrawRectangleRec(rl.NewRectangle(x+w-bevel, y, bevel, h), lighten(b.Color, 0.3))
			rl.DrawRectangleRec(rl.NewRectangle(x, y+h-bevel, w, bevel), lighten(b.Color, 0.3))
		} else if b.hovered {
			// Hovered: subtle darken with lower factor
			rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), darken(b.Color, 0.15))
			rl.DrawRectangleRec(rl.NewRectangle(x, y, w, bevel), lighten(b.Color, 0.4))
			rl.DrawRectangleRec(rl.NewRectangle(x, y, bevel, h), lighten(b.Color, 0.4))
			rl.DrawRectangleRec(rl.NewRectangle(x+w-bevel, y, bevel, h), darken(b.Color, 0.45))
			rl.DrawRectangleRec(rl.NewRectangle(x, y+h-bevel, w, bevel), darken(b.Color, 0.45))
		} else {
			// Raised: normal background
			rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), b.Color)
			// Top/left highlight (lighter)
			rl.DrawRectangleRec(rl.NewRectangle(x, y, w, bevel), lighten(b.Color, 0.5))
			rl.DrawRectangleRec(rl.NewRectangle(x, y, bevel, h), lighten(b.Color, 0.5))
			// Bottom/right shadow (darker)
			rl.DrawRectangleRec(rl.NewRectangle(x+w-bevel, y, bevel, h), darken(b.Color, 0.5))
			rl.DrawRectangleRec(rl.NewRectangle(x, y+h-bevel, w, bevel), darken(b.Color, 0.5))
		}

	// Border (optional)
	//rl.DrawRectangleLines(int32(x), int32(y), int32(w), int32(h), rl.Black)

	// Centered label
	textWidth := rl.MeasureText(b.Label, b.FontSize)
	textX := int32(b.Center.X) - textWidth/2
	textY := int32(b.Center.Y) - b.FontSize/2
	if b.held {
		textX += int32(pressOffset)
		textY += int32(pressOffset)
	}
	rl.DrawText(b.Label, textX, textY, b.FontSize, rl.White)
}

// Simple color helpers (define these somewhere accessible or inline)
func lighten(c rl.Color, factor float32) rl.Color {
	return rl.Color{
		R: uint8(min(255, float32(c.R)*(1+factor))),
		G: uint8(min(255, float32(c.G)*(1+factor))),
		B: uint8(min(255, float32(c.B)*(1+factor))),
		A: c.A,
	}
}

func darken(c rl.Color, factor float32) rl.Color {
	return rl.Color{
		R: uint8(float32(c.R) * (1 - factor)),
		G: uint8(float32(c.G) * (1 - factor)),
		B: uint8(float32(c.B) * (1 - factor)),
		A: c.A,
	}
}
