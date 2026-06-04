package components

import (
	"fmt"
	"math"
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
	fmt.Println(c.curForce)
	c.Center = c.Center.Add(c.curForce)
}

func (c *Circle) AddForce(force rl.Vector2) {
	c.curForce = c.curForce.Add(force)
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

	fmt.Println(c, "pppp", d)

	//Another dumb ass way, non Newtonian
	tmp := d.Center.Subtract(c.Center)

	fmt.Println("Fuckkkkkkkkkkkkkkkk: ", tmp)

	if tmp.X > 1000 || tmp.Y > 1000 {
		return
	}
	cmpWeight := c.weight * d.weight

	if tmp.X <= 0 && tmp.X >= -1 {
		tmp.X = -1
	}

	if tmp.X >= 0 && tmp.X <= 1 {
		tmp.X = 1
	}

	if tmp.Y <= 0 && tmp.Y >= -1 {
		tmp.Y = -1
	}

	if tmp.Y >= 0 && tmp.Y <= 1 {
		tmp.Y = 1
	}

	if float32(math.Abs(float64(tmp.X))) <= c.Radius+d.Radius {
		tmp.X = tmp.X / float32(math.Abs(float64(tmp.X)))
	} else {
		if tmp.X > 0 {
			tmp.X -= c.Radius + d.Radius
		} else {
			tmp.X += c.Radius + d.Radius
		}
	}

	if float32(math.Abs(float64(tmp.Y))) <= c.Radius+d.Radius {
		tmp.Y = tmp.Y / float32(math.Abs(float64(tmp.Y)))
	} else {
		if tmp.Y > 0 {
			tmp.Y -= c.Radius + d.Radius
		} else {
			tmp.Y += c.Radius + d.Radius
		}
	}

	fmt.Println("Val", c.weight, d.weight, cmpWeight, rl.Vector2{X: cmpWeight / tmp.X, Y: cmpWeight / tmp.Y})

	c.AddForce(rl.Vector2{X: -cmpWeight / tmp.X, Y: -cmpWeight / tmp.Y})
	d.AddForce(rl.Vector2{X: cmpWeight / tmp.X, Y: cmpWeight / tmp.Y})

	fmt.Println("Res", c.curForce, d.curForce, rl.Vector2{X: cmpWeight / tmp.X, Y: cmpWeight / tmp.Y})
}

func (c *Circle) Gravity(dtmp Component) {
	d := dtmp.GetCircle()

	//Hi there, Daniel
	tmp := d.Center.Subtract(c.Center)

	if tmp.X > 1000 || tmp.Y > 1000 {
		return
	}

	cmpWeight := c.weight * d.weight

	if tmp.X <= 0 && tmp.X >= -1 {
		tmp.X = -1
	}

	if tmp.X >= 0 && tmp.X <= 1 {
		tmp.X = 1
	}

	if tmp.Y <= 0 && tmp.Y >= -1 {
		tmp.Y = -1
	}

	if tmp.Y >= 0 && tmp.Y <= 1 {
		tmp.Y = 1
	}

	c.AddForce(rl.Vector2{X: cmpWeight / tmp.X, Y: cmpWeight / tmp.Y})
	d.AddForce(rl.Vector2{X: -cmpWeight / tmp.X, Y: -cmpWeight / tmp.Y})
}
