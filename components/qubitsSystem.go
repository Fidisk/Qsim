package components

import (
	"math"
	"math/rand/v2"
	glob "qsim/globals"
	qub "qsim/qubits"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	StateNormal   = 0
	StateExpanded = 1
	StateEjected  = 2
)

type QubitsSystem struct {
	Circle
	QubitList []*Qubit
	Origin    *qub.QubitStateManager
	BitList   []int32 
	ID        int32
	HookID    int32

	ViewState     int    
	EjectedIndex  int    
	BaseRadius    float32 // Remembers the unexpanded layout dimension size
}

func NewQubitsSystem(x, y, radius float32, color rl.Color) *QubitsSystem {
	tmp := QubitsSystem{
		Circle:       *NewCircle(x, y, radius, color),
		QubitList:    nil,
		Origin:       nil,
		HookID:       0,
		ViewState:    StateNormal,
		EjectedIndex: -1,
		BaseRadius:   radius,
	}
	tmp.ID = utils.GenerateID(&tmp)
	return &tmp
}

// Dynamically calculates a system container radius that completely encloses all inner circles
func (c *QubitsSystem) GetCurrentRadius() float32 {
	if c.ViewState == StateExpanded {
		// spreadRadius is BaseRadius * 1.1. Sub-circles are 45.0. 
		// We add them together plus a 15px safe inner boundary padding.
		return (c.BaseRadius * 1.1) + 45.0 + 15.0
	}
	// If a qubit is ejected, the main cluster goes back to containing just the remaining qubits
	if c.ViewState == StateEjected {
		return (c.BaseRadius * 1.1) + 45.0 + 15.0
	}
	return c.BaseRadius
}

func (c *QubitsSystem) getSubCircleCenter(index int) rl.Vector2 {
	q := c.QubitList[index]
	currentRadius := c.GetCurrentRadius()
	if c.ViewState == StateEjected && c.EjectedIndex == index {
		// Positions the single inspected target safely out to the right of the expanded main container wall
		return rl.Vector2{
			X: c.Center.X + currentRadius + q.RenderRadius + 25,
			Y: c.Center.Y,
		}
	}
	return rl.Vector2Add(c.Center, q.ExpandedOffset)
}

func (c *QubitsSystem) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	currentRadius := c.GetCurrentRadius()

	for i, d := range c.QubitList {
		d.Update()
		isThisQubitEjected := (c.ViewState == StateEjected && c.EjectedIndex == i)
		d.CalculateCenter(c.Center, c.BaseRadius, c.ViewState, isThisQubitEjected, currentRadius)
	}

	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		clickedInsideSystem := rl.CheckCollisionPointCircle(worldMouse, c.Center, currentRadius)

		switch c.ViewState {
		case StateNormal:
			if clickedInsideSystem && !c.dragging {
				c.ViewState = StateExpanded
				return 
			}

		case StateExpanded:
			clickedAQubit := false
			for i, d := range c.QubitList {
				subCenter := c.getSubCircleCenter(i)
				if rl.CheckCollisionPointCircle(worldMouse, subCenter, d.RenderRadius) {
					c.ViewState = StateEjected
					c.EjectedIndex = i
					clickedAQubit = true
					break
				}
			}

			if !clickedInsideSystem && !clickedAQubit {
				c.ViewState = StateNormal
				return
			}

		case StateEjected:
			clickedTarget := false
			if c.EjectedIndex >= 0 && c.EjectedIndex < len(c.QubitList) {
				subCenter := c.getSubCircleCenter(c.EjectedIndex)
				d := c.QubitList[c.EjectedIndex]
				if rl.CheckCollisionPointCircle(worldMouse, subCenter, d.RenderRadius) {
					clickedTarget = true
				}
			}

			if !clickedTarget {
				c.ViewState = StateExpanded
				c.EjectedIndex = -1
				return
			}
		}
	}

	if rl.CheckCollisionPointCircle(worldMouse, c.Center, currentRadius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			c.dragging = true
			*isCursorAvailable = false
			c.holdingCursor = true
			c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		c.dragging = false
		c.holdingCursor = false
		*isCursorAvailable = true
		c.zipToHook()
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
	}
}

func (c *QubitsSystem) Draw() {
	currentRadius := c.GetCurrentRadius()

	// Draw the expanded container circle bounds background
	rl.DrawCircleV(c.Center, currentRadius, glob.ColorBg)
	rl.DrawCircleLinesV(c.Center, currentRadius, c.Color)
	
	if c.ViewState > StateNormal {
		for i, d := range c.QubitList {
			// If a qubit is ejected out, do not render its tracking ring inside the container
			if c.ViewState == StateEjected && c.EjectedIndex == i {
				continue
			}
			subCenter := c.getSubCircleCenter(i)
			rl.DrawCircleLinesV(subCenter, d.RenderRadius, rl.LightGray)
		}

		// Draw the single ejected focus ring cleanly outside the system
		if c.ViewState == StateEjected && c.EjectedIndex >= 0 && c.EjectedIndex < len(c.QubitList) {
			subCenter := c.getSubCircleCenter(c.EjectedIndex)
			d := c.QubitList[c.EjectedIndex]
			rl.DrawCircleLinesV(subCenter, d.RenderRadius, rl.Gold)
		}
	}

	// Render moving orbital particles
	for _, d := range c.QubitList {
		d.Draw()
	}
}

func (c *QubitsSystem) Assign(p *qub.QubitStateManager) {
	c.Origin = p
	c.QubitList = nil

	totalQubits := len(p.Representation)

	for i := range p.Representation {
		rotation := rand.Float32() * 2 * math.Pi
		angle := rand.Float32() * 2 * math.Pi

		magnitude := rand.Float32()*0.05 + 0.05
		if rand.IntN(2) == 0 {
			magnitude = -magnitude
		}
		rotationDelta := magnitude

		magnitude = rand.Float32()*0.05 + 0.05
		if rand.IntN(2) == 0 {
			magnitude = -magnitude
		}
		angleDelta := magnitude

		ratio := rand.Float32()*2.5 + 0.5
		if rand.IntN(2) == 0 {
			ratio = 1.0 / ratio
		}

		pathRotation := rand.Float32() * 2 * math.Pi
		pathRotationDelta := rand.Float32()*0.001 + 0.001

		size := p.Amptitude[i] / 2.0
		radius := 0.95 - size

		q := NewQubit(rotation, rotationDelta, pathRotation, pathRotationDelta, radius, angle, angleDelta, size, ratio, p.Representation[i], p.ModifierID, int32(len(p.ModifierID)))

		if totalQubits > 1 {
			distributionAngle := (float64(i) * 2 * math.Pi) / float64(totalQubits)
			// Spread centers out using BaseRadius as a reference scalar
			spreadRadius := c.BaseRadius * 1.1
			q.ExpandedOffset = rl.Vector2{
				X: spreadRadius * float32(math.Cos(distributionAngle)),
				Y: spreadRadius * float32(math.Sin(distributionAngle)),
			}
		} else {
			q.ExpandedOffset = rl.Vector2{X: 0, Y: 0}
		}

		c.QubitList = append(c.QubitList, q)
	}
}

func (c *QubitsSystem) zipToHook() {
	tmp := c.GetParent()
	ele := tmp.GetElement()
	gotHooked := false
	for _, d := range ele {
		switch v := d.(type) {
		case *Hook:
			if utils.Dist(v.Center, c.Center) <= glob.HookDist && (!v.IsHooked || v.TargetID == c.ID) && !gotHooked {
				c.removeFromHook()
				c.Center = v.Center
				v.IsHooked = true
				v.TargetID = c.ID
				gotHooked = true
				c.HookID = v.ID
			}
		case *Gate:
			for _, d2 := range v.HookList {
				if utils.Dist(d2.Center, c.Center) <= glob.HookDist && (!d2.IsHooked || d2.TargetID == c.ID) && !gotHooked {
					c.removeFromHook()
					c.Center = d2.Center
					d2.IsHooked = true
					d2.TargetID = c.ID
					gotHooked = true
					c.HookID = d2.ID
				}
			}
		default:
		}
		if gotHooked {
			break
		}
	}
	if !gotHooked && c.HookID != 0 {
		c.removeFromHook()
	}
}

func (c *QubitsSystem) removeFromHook() {
	if c.HookID == 0 {
		return
	}
	tmp := utils.GetObjectFromID(c.HookID)
	c.HookID = 0
	switch t := tmp.(type) {
	case *Hook:
		t.IsHooked = false
		t.TargetID = 0
	default:
	}
}