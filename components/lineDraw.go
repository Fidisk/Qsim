package components

import (
	"math"

	glob "qsim/globals"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type LineDraw struct {
	*Circle
	End     rl.Vector2
	Placing bool
	LineColor rl.Color
	ID      int32
}

func NewLineDraw(x, y float32) *LineDraw {
	ld := &LineDraw{
		Circle:    NewCircle(x, y, 1, rl.Blank),
		End:       rl.Vector2{X: x, Y: y},
		Placing:   true,
		LineColor: rl.White,
	}
	ld.ID = utils.GenerateID(ld)
	return ld
}

func (ld *LineDraw) GetID() int32 { return ld.ID }

func pointToLineDist(p, a, b rl.Vector2) float32 {
	ab := b.Subtract(a)
	ap := p.Subtract(a)
	dot := ab.X*ap.X + ab.Y*ap.Y
	lenSq := ab.X*ab.X + ab.Y*ab.Y
	if lenSq == 0 {
		return rl.Vector2Distance(p, a)
	}
	t := dot / lenSq
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cx := a.X + ab.X*t
	cy := a.Y + ab.Y*t
	return float32(math.Sqrt(float64((p.X-cx)*(p.X-cx) + (p.Y-cy)*(p.Y-cy))))
}

func (ld *LineDraw) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if ld.Placing {
		ld.End = worldMouse
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			dx := worldMouse.X - ld.Center.X
			dy := worldMouse.Y - ld.Center.Y
			angle := math.Atan2(float64(dy), float64(dx))
			snapped := math.Round(angle/(math.Pi/4)) * (math.Pi / 4)
			dist := rl.Vector2Distance(ld.Center, worldMouse)
			ld.End = rl.Vector2{
				X: ld.Center.X + float32(math.Cos(snapped)*float64(dist)),
				Y: ld.Center.Y + float32(math.Sin(snapped)*float64(dist)),
			}
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor {
			ld.Placing = false
		}
		return
	}

	if utils.IsMouseState(glob.MouseStateErase) && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor {
		if pointToLineDist(worldMouse, ld.Center, ld.End) < 10 {
			ld.GetParent().DeleteChildWithID(ld.ID)
		}
	}
}

func (ld *LineDraw) Draw() {
	col := ld.LineColor
	if ld.Placing {
		col = rl.Fade(col, 0.3)
	}
	if ld.Center.X != ld.End.X || ld.Center.Y != ld.End.Y {
		rl.DrawLineEx(ld.Center, ld.End, 2, col)
	}
}

func (ld *LineDraw) DrawGhost() {
	rl.DrawLineEx(ld.Center, ld.End, 2, rl.Fade(ld.LineColor, 0.15))
}
