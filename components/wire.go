package components

import (
	"qsim/config"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawLogicalBitWire draws a double parallel line: a thick line in col and a
// thinner line in the background color centered on top, so the wire looks like
// two colored rails with a gap between them.
func DrawLogicalBitWire(a, b rl.Vector2, thick float32, col rl.Color) {
	DrawWire(a, b, thick+3, col)
	DrawWire(a, b, thick, config.ColorBg)
}

// IsLogicalTarget reports whether a wire endpoint is a logical carrier: a
// LogicalBit or a logical QubitSystem (IsLogical set). By convention such
// connections are drawn with the double-line wire.
func IsLogicalTarget(target interface{}) bool {
	switch t := target.(type) {
	case *LogicalBit:
		return true
	case *QubitsSystem:
		return t.IsLogical
	}
	return false
}

// DrawWireToTarget draws the wire to a hooked target, using the double-line
// style when the target is a logical carrier (see IsLogicalTarget).
func DrawWireToTarget(a, b rl.Vector2, thick float32, col rl.Color, target interface{}) {
	if IsLogicalTarget(target) {
		DrawLogicalBitWire(a, b, thick, col)
		return
	}
	DrawWire(a, b, thick, col)
}

// DrawHookLink draws the wire from a body edge to a hooked target. It aims at
// the target itself rather than at the hook: a target with several links
// floats between them while the hooks rest at their anchors, so aiming at the
// hook would leave the wire short of the target. The wire stops at the
// target's drawn outline — the state-grid rectangle edge for a qubit system,
// the component circle otherwise — and uses the double-line style for logical
// carriers.
func DrawHookLink(edge rl.Vector2, t Component, thick float32, col rl.Color) {
	DrawWireToTarget(edge, outlinePoint(t, edge), thick, col, t)
}

// outlinePoint returns the point on the target's drawn outline toward which a
// wire from `from` should stop: the grid rectangle edge for a qubit system
// (its grid is the visible body, far larger than its 30px circle), the
// component circle otherwise.
func outlinePoint(t Component, from rl.Vector2) rl.Vector2 {
	if qs, ok := t.(*QubitsSystem); ok {
		return qs.OutlinePoint(from)
	}
	c := t.GetCircle()
	d := utils.Dist(from, c.Center) - c.Radius
	if d < 0 {
		d = 0
	}
	return rl.Vector2Add(from, rl.Vector2Scale(
		rl.Vector2Normalize(rl.Vector2Subtract(c.Center, from)),
		d,
	))
}

// WirePointAt returns the point at parameter t (0..1) along the same cubic
// bezier DrawWire renders between endpoints a and b.
func WirePointAt(a, b rl.Vector2, t float32) rl.Vector2 {
	off := (b.X - a.X) * 0.5
	if off > 80 {
		off = 80
	}
	if off < -80 {
		off = -80
	}
	c1 := rl.NewVector2(a.X+off, a.Y)
	c2 := rl.NewVector2(b.X-off, b.Y)
	mt := 1 - t
	return rl.NewVector2(
		mt*mt*mt*a.X+3*mt*mt*t*c1.X+3*mt*t*t*c2.X+t*t*t*b.X,
		mt*mt*mt*a.Y+3*mt*mt*t*c1.Y+3*mt*t*t*c2.Y+t*t*t*b.Y,
	)
}

// DrawWire draws a connection line as a smooth cubic bezier that leaves its
// endpoints horizontally. It reads much better than a straight line for
// hook/component connections.
func DrawWire(a, b rl.Vector2, thick float32, col rl.Color) {
	const segs = 20
	prev := a
	for i := 1; i <= segs; i++ {
		cur := WirePointAt(a, b, float32(i)/segs)
		rl.DrawLineEx(prev, cur, thick, col)
		prev = cur
	}
}
