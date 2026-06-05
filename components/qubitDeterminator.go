package components

import (
	glob "qsim/globals"
	"qsim/qubits/attributes"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type QubitDeterminator struct {
	Circle
	ID            int32
	ModifierID    int32
	HookID        int32
	QubitSystemID int32
}

func NewQubitDeterminator(x, y, radius float32, color rl.Color, qubitID int32) *QubitDeterminator {
	tmp := QubitDeterminator{
		Circle:     *NewCircle(x, y, radius, color),
		ModifierID: qubitID,
		HookID:     0,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight(glob.QubitDeterminatorWeight)
	return &tmp
}

func (c *QubitDeterminator) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
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
		c.ClearForce()
	} else {
		c.DecayForce()
		c.ApplyForce()
	}
}

func (c *QubitDeterminator) Draw() {
	rl.DrawCircleV(c.Center, c.Radius, glob.ColorBg)
	rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)

	tmp := attributes.AttributesManager.Get(c.ModifierID)
	switch t := tmp.(type) {
	case *attributes.Side:
		rl.DrawPoly(c.Center, t.SideCntPositive, c.Radius*0.75, 0, c.Color)
	case *attributes.Color:
		rl.DrawCircleV(c.Center, c.Radius, rl.NewColor(uint8(t.R), uint8(t.G), uint8(t.B), uint8(0255)))
		rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)
	}
}

func (c *QubitDeterminator) GetQubitParent() *QubitsSystem {
	//Evil
	tmp := utils.GetObjectFromID(c.QubitSystemID)
	return tmp.(*QubitsSystem)
}

func (c *QubitDeterminator) GetParent() PlaceholderWindow {
	//AHHHHHHHHHHHHH
	return nil
}

func (c *QubitDeterminator) zipToHook() {
	tmp := c.GetQubitParent().GetParent()
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
				c.SetWeight(0)
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
					c.SetWeight(0)
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

func (c *QubitDeterminator) removeFromHook() {
	if c.HookID == 0 {
		return
	}
	tmp := utils.GetObjectFromID(c.HookID)
	c.HookID = 0
	c.SetWeight(glob.QubitDeterminatorWeight)
	switch t := tmp.(type) {
	case *Hook:
		t.IsHooked = false
		t.TargetID = 0
	default:
	}
}

func (c *QubitDeterminator) PostUpdate() {

}
