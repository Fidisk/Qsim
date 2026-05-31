package components

import (
	"fmt"
	"math"
	"math/rand/v2"
	glob "qsim/globals"
	qub "qsim/qubits"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type QubitsSystem struct {
	Circle
	QubitList []*Qubit
	Origin    *qub.QubitStateManager
	BitList   []int32 //Overkill
}

func NewQubitsSystem(x, y, radius float32, color rl.Color) *QubitsSystem {
	return &QubitsSystem{
		Circle:    *NewCircle(x, y, radius, color),
		QubitList: nil,
		Origin:    nil,
	}
}

func (c *QubitsSystem) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	for _, d := range c.QubitList {
		d.Update()
	}

	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.HoldingCursor || *isCursorAvailable) {
			c.dragging = true
			*isCursorAvailable = false
			c.HoldingCursor = true
			c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.HoldingCursor {
		c.dragging = false
		c.HoldingCursor = true
		*isCursorAvailable = true
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
	}
}

func (c *QubitsSystem) Draw() {
	rl.DrawCircleV(c.Center, c.Radius, glob.ColorBg)
	rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)
	for _, d := range c.QubitList {
		d.Draw(c.Center, c.Radius)
	}
}

func (c *QubitsSystem) Assign(p *qub.QubitStateManager) {
	//This do not defer the Origin
	c.Origin = p

	//The downfall of OOP
	c.QubitList = nil

	for i := range p.Representation {
		rotation := rand.Float32() * 2 * math.Pi
		angle := rand.Float32() * 2 * math.Pi

		magnitude := rand.Float32()*0.05 + 0.05 // 0.05 … 0.1
		if rand.IntN(2) == 0 {
			magnitude = -magnitude
		}
		rotationDelta := magnitude

		magnitude = rand.Float32()*0.05 + 0.05
		if rand.IntN(2) == 0 {
			magnitude = -magnitude
		}
		angleDelta := magnitude

		ratio := rand.Float32()*2.5 + 0.5 // 0.5 … 10
		if rand.IntN(2) == 0 {
			ratio = 1.0 / ratio
		}

		pathRotation := rand.Float32() * 2 * math.Pi
		pathRotationDelta := rand.Float32()*0.001 + 0.001

		size := p.Amptitude[i] / 2.0
		radius := 0.95 - size

		fmt.Println(ratio)

		q := NewQubit(rotation, rotationDelta, pathRotation, pathRotationDelta, radius, angle, angleDelta, size, ratio, p.Representation[i], p.ModifierID, int32(len(p.ModifierID)))

		// Use q (e.g., append to QubitList)
		c.QubitList = append(c.QubitList, q)
	}
}
