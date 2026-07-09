package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Label struct {
	*Circle
	Text     string
	FontSize int32
	Color    rl.Color
}

func NewLabel(x, y float32, text string, fontSize int32, color rl.Color) *Label {
	l := &Label{
		Circle:   NewCircle(x, y, 1, rl.Blank),
		Text:     text,
		FontSize: fontSize,
		Color:    color,
	}
	l.SetWeight(0)
	return l
}

func (l *Label) Draw() {
	rl.DrawText(l.Text, int32(l.Center.X), int32(l.Center.Y), l.FontSize, l.Color)
}

func (l *Label) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {}
