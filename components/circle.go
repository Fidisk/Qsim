package components

import (
	"math"
	"qsim/globals"
	"qsim/utils"

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

func (c *Circle) SetWeight(t float32) {
	c.weight = t
}

func (c *Circle) GetWeight() float32 {
	return c.weight
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

func (c *Circle) SetCenter(t rl.Vector2) {
	c.Center = t
}

func (c *Circle) GetCenter() rl.Vector2 {
	return c.Center
}

func (c *Circle) GetCircle() *Circle {
	return c
}

func (c *Circle) Draw() {
	rl.DrawCircleV(c.Center, c.Radius, c.Color)
}

func (c *Circle) ApplyForce() {
	c.Center = c.Center.Add(c.curForce)
}

func (c *Circle) AddForce(force rl.Vector2) {
	c.curForce = c.curForce.Add(force)
}

func (c *Circle) ClearForce() {
	c.curForce = rl.Vector2{X: 0, Y: 0}
}

func (c *Circle) DecayForce() {
	//Dumb ass way to do friction
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
	d := dtmp.GetCircle()

	//Another dumb ass way, non Newtonian
	tmp := d.Center.Subtract(c.Center)

	dist := utils.Dist(d.Center, c.Center)

	//Right so karma got me, need to rewrite this
	c.AddForce(tmp.Normalize().Scale(c.weight * d.weight).Scale(-float32(math.Min(float64(1/dist/dist), float64(10)))))
	d.AddForce(tmp.Normalize().Scale(c.weight * d.weight).Scale(float32(math.Min(float64(1/dist/dist), float64(10)))))
}

func (c *Circle) Gravity(dtmp Component) {
	d := dtmp.GetCircle()

	//Another dumb ass way, non Newtonian
	tmp := d.Center.Subtract(c.Center)

	dist := utils.Dist(d.Center, c.Center)

	//Right so karma got me, need to rewrite this
	c.AddForce(tmp.Normalize().Scale(c.weight * d.weight).Scale(float32(math.Min(float64(1/dist/dist), float64(10)))))
	d.AddForce(tmp.Normalize().Scale(c.weight * d.weight).Scale(-float32(math.Min(float64(1/dist/dist), float64(10)))))
}
