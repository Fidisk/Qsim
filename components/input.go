package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Input struct {
	*Circle
	Width, Height float32
	Text          string
	FontSize      int32
	MaxChars      int
	Active        bool
	OnSubmit      func(string)
	frameCounter  int
}

func NewInput(x, y, width, height float32, fontSize int32, maxChars int, onSubmit func(string)) *Input {
	radius := max(width, height) / 2
	return &Input{
		Circle:    NewCircle(x, y, radius, rl.Blank),
		Width:     width,
		Height:    height,
		Text:      "",
		FontSize:  fontSize,
		MaxChars:  maxChars,
		OnSubmit:  onSubmit,
		Active:    false,
	}
}

func (in *Input) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	rect := rl.NewRectangle(
		in.Center.X-in.Width/2,
		in.Center.Y-in.Height/2,
		in.Width,
		in.Height,
	)

	over := rl.CheckCollisionPointRec(worldMouse, rect)
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (*isCursorAvailable || over) {
		if over {
			in.Active = true
			*isCursorAvailable = false
		} else {
			in.Active = false
		}
	}

	if !in.Active {
		return
	}

	in.frameCounter++
	key := rl.GetCharPressed()
	for key > 0 {
		if key >= 32 && key <= 125 && len(in.Text) < in.MaxChars {
			in.Text += string(rune(key))
		}
		key = rl.GetCharPressed()
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(in.Text) > 0 {
		in.Text = in.Text[:len(in.Text)-1]
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		if in.OnSubmit != nil {
			in.OnSubmit(in.Text)
		}
	}
}

func (in *Input) Draw() {
	x := in.Center.X - in.Width/2
	y := in.Center.Y - in.Height/2
	w := in.Width
	h := in.Height

	border := rl.DarkGray
	bg := rl.NewColor(30, 30, 30, 255)
	if in.Active {
		border = rl.SkyBlue
	}
	box := rl.NewRectangle(x, y, w, h)
	rl.DrawRectangleRounded(box, 0.2, 4, bg)
	rl.DrawRectangleRoundedLinesEx(box, 0.2, 4, 2, border)

	textX := int32(x + 8)
	textY := int32(y + (h-float32(in.FontSize))/2)
	rl.DrawText(in.Text, textX, textY, in.FontSize, rl.White)

	if in.Active && (in.frameCounter/20)%2 == 0 {
		textW := rl.MeasureText(in.Text, in.FontSize)
		rl.DrawRectangle(textX+textW, textY, 4, in.FontSize, rl.SkyBlue)
	}
}

func (in *Input) SetText(t string) {
	in.Text = t
}
