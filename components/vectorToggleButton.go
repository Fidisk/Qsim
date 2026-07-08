package components

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type VectorToggleButton struct {
	*Circle
	Width, Height float32
	Path          *Path // the vector shape to draw
	StrokeColor   rl.Color
	StrokeWidth   float32
	OnClick       func()
	OffClick      func()
	toggled       func() bool
	held          bool
}

func NewVectorToggleButton(
	x, y, width, height float32,
	path *Path,
	strokeColor rl.Color,
	strokeWidth float32,
	state func() bool,
	onClick, offClick func(),
) *VectorToggleButton {
	fmt.Println("Fuckkckckkckck")
	radius := max(width, height) / 2
	tb := &VectorToggleButton{
		Circle:      NewCircle(x, y, radius, strokeColor),
		Width:       width,
		Height:      height,
		Path:        path,
		StrokeColor: strokeColor,
		StrokeWidth: strokeWidth,
		OnClick:     onClick,
		OffClick:    offClick,
		toggled:     state,
	}
	tb.SetWeight(0)
	return tb
}

func (tb *VectorToggleButton) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
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
			if rl.CheckCollisionPointRec(worldMouse, rect) {
				if !tb.toggled() {
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
	if tb.held && !rl.CheckCollisionPointRec(worldMouse, rect) {
		tb.held = false
		*isCursorAvailable = true
	}
}

func (tb *VectorToggleButton) Draw() {
	fmt.Println("AIIJJSJSJSJ")
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
	} else {
		// Raised look
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), tb.Color)
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, toggleBevel), lighten(tb.Color, 0.5))
		rl.DrawRectangleRec(rl.NewRectangle(x, y, toggleBevel, h), lighten(tb.Color, 0.5))
		rl.DrawRectangleRec(rl.NewRectangle(x+w-toggleBevel, y, toggleBevel, h), darken(tb.Color, 0.5))
		rl.DrawRectangleRec(rl.NewRectangle(x, y+h-toggleBevel, w, toggleBevel), darken(tb.Color, 0.5))
	}

	offset := float32(0)
	if tb.toggled() {
		offset = togglePressOffset
	}

	subpaths := tb.Path.Flatten(0.01) // tolerance in pixels
	fmt.Println(len(subpaths), "subpaths")

	if len(subpaths) == 0 {
		return
	}

	for _, subpath := range subpaths {
		if len(subpath) < 2 {
			continue
		}

		translated := make([]rl.Vector2, len(subpath))
		for i, p := range subpath {
			translated[i] = rl.Vector2{
				X: p.X + tb.Center.X - tb.Width/2 + offset,
				Y: p.Y + tb.Center.Y - tb.Height/2 + offset,
			}
		}

		// Draw all segments including the closing one
		for i := 0; i < len(translated); i++ {
			next := (i + 1) % len(translated)
			rl.DrawLineEx(translated[i], translated[next], tb.StrokeWidth, rl.Black)
		}
	}
}
