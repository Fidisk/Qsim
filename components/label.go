package components

import (
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Label struct {
	*Circle
	ID       int32
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
	l.ID = utils.GenerateID(l)
	l.SetWeight(0)
	return l
}

func (l *Label) GetID() int32 { return l.ID }

func (l *Label) Draw() {
	rl.DrawText(l.Text, int32(l.Center.X), int32(l.Center.Y), l.FontSize, l.Color)
}

func (l *Label) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {}
