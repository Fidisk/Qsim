package components

import (
	"qsim/globals"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Circle struct {
	WindowComponent
	Center   rl.Vector2
	Radius   float32
	Color    rl.Color
	dragging bool
	offset   rl.Vector2
	curForce rl.Vector2
	weight   float32
}

func NewCircle(x, y, radius float32, color rl.Color) *Circle {
	return &Circle{
		Center: rl.NewVector2(x, y),
		Radius: radius,
		Color:  color,
	}
}

func (c *Circle) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor {
			c.dragging = true
			c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		c.dragging = false
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
	}
}

func (c *Circle) Draw() {
	rl.DrawCircleV(c.Center, c.Radius, c.Color)
}

func (c *Circle) ApplyForce() {
	c.Center.Add(c.curForce)
}

func (c *Circle) AddForce(force rl.Vector2) {
	c.curForce.Add(force)
}

func (c *Circle) DecayForce() {
	//Dumb ass way to do friction
	c.curForce.Scale(globals.ForceDecay)
}

// AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
func (c *Circle) AntiGravity(d *Circle) {
	//Another dumb ass way, non Newtonian
	cmpWeight := c.weight * d.weight
	tmp := d.Center.Subtract(c.Center).ClampValue(1, 1000000) //Magic Number
	c.AddForce(rl.Vector2{X: cmpWeight / tmp.X, Y: cmpWeight / tmp.Y})
}

func (c *Circle) Gravity(d *Circle) {
	//Hi there, Daniel
	cmpWeight := c.weight * d.weight
	tmp := d.Center.Subtract(c.Center).ClampValue(1, 10)
	c.AddForce(rl.Vector2{X: -cmpWeight / tmp.X, Y: -cmpWeight / tmp.Y})
}
