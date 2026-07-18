package components

import (
	"qsim/config"
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

func (c *QubitDeterminator) GetID() int32 {
	return c.ID
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

func (c *QubitDeterminator) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateDetach):
		if c.HookID != 0 {
			tmp := utils.GetObjectFromID(c.HookID)
			if h, ok := tmp.(*Hook); ok {
				h.Disconnect()
			}
		}
	default:
		c.dragging = true
		*isCursorAvailable = false
		c.holdingCursor = true
		c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		c.VirtualCenter = c.Center
	}
}

func (c *QubitDeterminator) onRelease() {

}

func (c *QubitDeterminator) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			c.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		c.dragging = false
		c.holdingCursor = false
		*isCursorAvailable = true

		c.zipToHook()
	}
	if c.dragging {
		raw := rl.Vector2Add(worldMouse, c.offset)
		c.VirtualCenter = rl.Vector2Lerp(c.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		c.Center = c.VirtualCenter
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
	case *attributes.Name:
		rl.DrawText(t.Val, int32(c.Center.X)-int32(rl.MeasureText(t.Val, 20)/2), int32(c.Center.Y)-10, 20, c.Color)
	}
}

func (c *QubitDeterminator) GetQubitParent() *QubitsSystem {
	//Evil
	tmp, ok := utils.GetObjectFromID(c.QubitSystemID).(*QubitsSystem)
	if ok {
		return tmp
	} else {
		return nil
	}
}

func (c *QubitDeterminator) GetParent() PlaceholderWindow {
	//AHHHHHHHHHHHHH
	return nil
}

func (c *QubitDeterminator) zipToHook() {
	qp := c.GetQubitParent()
	if qp == nil {
		return
	}
	tmp := qp.GetParent()
	if tmp == nil {
		return
	}
	ele := tmp.GetElement()
	gotHooked := false
	for _, d := range ele {
		switch v := d.(type) {
		case *Hook:
			if utils.Dist(v.Center, c.Center) <= glob.HookDist && (!v.IsHooked || v.TargetID == c.ID) && !gotHooked {
				gotHooked = true
				v.Connect(c)
			}
		case *Gate:
			for _, d2 := range v.HookList {
				if utils.Dist(d2.Center, c.Center) <= glob.HookDist && (!d2.IsHooked || d2.TargetID == c.ID) && !gotHooked {
					gotHooked = true
					d2.Connect(c)
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

func (c *QubitDeterminator) Kill() {
	h, ok := utils.GetObjectFromID(c.HookID).(*Hook)

	if ok {
		h.Disconnect()
	}

	qp := c.GetQubitParent()
	if qp != nil {
		parent := qp.GetParent()
		if parent != nil {
			parent.DeleteChildWithID(c.ID)
		}
	}
}
