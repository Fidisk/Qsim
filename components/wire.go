package components

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawWire draws a connection line as a smooth cubic bezier that leaves its
// endpoints horizontally. It reads much better than a straight line for
// hook/component connections.
func DrawWire(a, b rl.Vector2, thick float32, col rl.Color) {
	off := (b.X - a.X) * 0.5
	if off > 80 {
		off = 80
	}
	if off < -80 {
		off = -80
	}
	c1 := rl.NewVector2(a.X+off, a.Y)
	c2 := rl.NewVector2(b.X-off, b.Y)

	const segs = 20
	prev := a
	for i := 1; i <= segs; i++ {
		t := float32(i) / segs
		mt := 1 - t
		cur := rl.NewVector2(
			mt*mt*mt*a.X+3*mt*mt*t*c1.X+3*mt*t*t*c2.X+t*t*t*b.X,
			mt*mt*mt*a.Y+3*mt*mt*t*c1.Y+3*mt*t*t*c2.Y+t*t*t*b.Y,
		)
		rl.DrawLineEx(prev, cur, thick, col)
		prev = cur
	}
}
