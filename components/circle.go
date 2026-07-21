package components

import (
	"math"
	"math/rand/v2"
	"qsim/config"
	"qsim/globals"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Circle struct {
	WindowComponent
	Center        rl.Vector2
	VirtualCenter rl.Vector2
	Radius        float32
	Color         rl.Color
	dragging      bool
	offset        rl.Vector2
	curForce      rl.Vector2
	weight        float32

	// double-click tracking for the detach shortcut
	lastClickTime float64
	lastClickPos  rl.Vector2

	IsFixed bool
}

// DoubleClicked reports whether the current left-press is a double-click
// (within 0.35s and 10 world units of the previous press) and re-arms the
// tracker. Call it only when the press actually lands on the component.
func (c *Circle) DoubleClicked(worldMouse rl.Vector2) bool {
	now := rl.GetTime()
	ok := now-c.lastClickTime < 0.35 && utils.Dist(c.lastClickPos, worldMouse) < 10
	c.lastClickTime = now
	c.lastClickPos = worldMouse
	return ok
}

func (c *Circle) SetWeight(t float32) {
	c.weight = t
}

func (c *Circle) GetWeight() float32 {
	return c.weight
}

func NewCircle(x, y, radius float32, color rl.Color) *Circle {
	return &Circle{
		Center:        rl.NewVector2(x, y),
		VirtualCenter: rl.NewVector2(x, y),
		Radius:        radius,
		Color:         color,
	}
}

func (c *Circle) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor {
			c.dragging = true
			c.offset = rl.Vector2Subtract(c.Center, worldMouse)
			c.VirtualCenter = c.Center
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		c.dragging = false
		if !c.IsFixed {
			c.Center = c.VirtualCenter
		}
		c.ClearForce()
	}
	if c.dragging {
		raw := rl.Vector2Add(worldMouse, c.offset)
		c.VirtualCenter = rl.Vector2Lerp(c.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
	}
}

func (c *Circle) SetCenter(t rl.Vector2) {
	c.Center = t
}

func (c *Circle) GetCenter() rl.Vector2 {
	return c.Center
}

func (c *Circle) GetCircle() *Circle {
	return c
}

func (c *Circle) IsDragging() bool {
	return c.dragging
}

func (c *Circle) Draw() {
	rl.DrawCircleV(c.Center, c.Radius, c.Color)
}

func (c *Circle) DrawGhost() {
	rl.DrawCircleLines(int32(c.Center.X), int32(c.Center.Y), c.Radius, rl.Fade(c.Color, 0.3))
}

func (c *Circle) GetChildCircles() []*Circle { return nil }

func (c *Circle) GetForce() rl.Vector2 {
	return c.curForce
}

func (c *Circle) ApplyForce() {
	if c.IsFixed {
		return
	}
	if !config.PhysicsEnabled {
		return
	}

	c.curForce = c.curForce.ClampValue(-globals.ForceCap, globals.ForceCap)
	c.Center = c.Center.Add(c.curForce)
}

func (c *Circle) AddForce(force rl.Vector2) {
	if !config.PhysicsEnabled {
		return
	}
	c.curForce = c.curForce.Add(force)
}

func (c *Circle) ClearForce() {
	c.curForce = rl.Vector2{X: 0, Y: 0}
}

func (c *Circle) DecayForce() {
	if !config.PhysicsEnabled {
		return
	}
	c.curForce = c.curForce.Scale(globals.ForceDecay)

	if math.Abs(float64(c.curForce.X)) < float64(globals.FrictionDelta) {
		c.curForce.X = 0
	}
	if math.Abs(float64(c.curForce.Y)) < float64(globals.FrictionDelta) {
		c.curForce.Y = 0
	}
}

// AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
func (c *Circle) AntiGravity(dtmp Component) {
	if !config.PhysicsEnabled {
		return
	}
	d := dtmp.GetCircle()

	//Another dumb ass way, non Newtonian
	tmp := d.Center.Subtract(c.Center)

	if (tmp == rl.Vector2{X: 0, Y: 0}) {
		tmp = rl.Vector2{X: rand.Float32() - 0.5*2, Y: rand.Float32() - 0.5*2}
	}

	dist := tmp.Length()

	//Right so karma got me, need to rewrite this
	c.AddForce(tmp.Normalize().Scale(c.weight*d.weight).Scale(-float32(math.Min(float64(1/dist/dist), float64(10)))).ClampValue(-100, 100))
	d.AddForce(tmp.Normalize().Scale(c.weight*d.weight).Scale(float32(math.Min(float64(1/dist/dist), float64(10)))).ClampValue(-100, 100))
}

func (c *Circle) Gravity(dtmp Component) {
	if !config.PhysicsEnabled {
		return
	}
	d := dtmp.GetCircle()

	//Another dumb ass way, non Newtonian
	tmp := d.Center.Subtract(c.Center)

	dist := utils.Dist(d.Center, c.Center)

	//Right so karma got me, need to rewrite this
	c.AddForce(tmp.Normalize().Scale(c.weight * d.weight).Scale(float32(math.Min(float64(1/dist/dist), float64(10)))))
	d.AddForce(tmp.Normalize().Scale(c.weight * d.weight).Scale(-float32(math.Min(float64(1/dist/dist), float64(10)))))
}

func (c *Circle) PostUpdate() {
	//To not cause crash
}

func (c *Circle) ChainUpdate() {
	//Bullshit to put this while realistically only a single gate function will need this, but oh welp
}
