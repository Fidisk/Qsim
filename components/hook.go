package components

import (
	"qsim/globals"
	glob "qsim/globals"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Hook struct {
	Circle
	IsHooked         bool
	ID               int32
	TargetID         int32
	IsOutput         bool
	Label            string
	AllowQubitSystem bool
}

func NewHook(x, y, radius float32, color rl.Color) *Hook {
	tmp := Hook{
		Circle: *NewCircle(x, y, radius, color),
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight((globals.HookWeight))
	return &tmp
}

func NewOutputHook(x, y, radius float32, color rl.Color) *Hook {
	tmp := Hook{
		Circle:   *NewCircle(x, y, radius, color),
		IsOutput: true,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight((globals.HookWeight))
	return &tmp
}

func (c *Hook) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		c.IsFixed = !c.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		if !c.IsOutput {
			c.Disconnect()
		}
	default:
		c.dragging = true
		*isCursorAvailable = false
		c.holdingCursor = true
		c.offset = rl.Vector2Subtract(c.Center, worldMouse)
	}
}

func (c *Hook) onRelease() {

}

func (c *Hook) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if c.IsHooked {
		if utils.GetObjectFromID(c.TargetID) == nil {
			c.Disconnect()
			//Should do smth about physics here, but meh
			return
		}

		tmp := utils.GetObjectFromID(c.TargetID)
		t := tmp.(Component)
		c.Center = t.GetCircle().Center
		//c.ClearForce()
		t.AddForce(c.GetCircle().GetForce())
		c.GetCircle().ClearForce()
		return
	}

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
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
		c.ClearForce()
	} else {
		c.DecayForce()
		c.ApplyForce()

	}
}

func (c *Hook) Draw() {
	if c.IsHooked {
		return
	}

	// Calculate the bounding lines based on Center and Radius
	left := rl.Vector2{X: c.Center.X - c.Radius, Y: c.Center.Y}
	right := rl.Vector2{X: c.Center.X + c.Radius, Y: c.Center.Y}
	top := rl.Vector2{X: c.Center.X, Y: c.Center.Y - c.Radius}
	bottom := rl.Vector2{X: c.Center.X, Y: c.Center.Y + c.Radius}

	// Define how bold you want the cross to be (in pixels)
	thickness := float32(4.0)

	// Draw the horizontal bar
	rl.DrawLineEx(left, right, thickness, c.Color)

	// Draw the vertical bar
	rl.DrawLineEx(top, bottom, thickness, c.Color)

	fontSize := int32(14) // adjust as needed
	offsetX := float32(8) // how far left from the center
	offsetY := float32(8) // how far up from the center
	textX := int32(c.Center.X - c.Radius - offsetX)
	textY := int32(c.Center.Y - c.Radius - offsetY - float32(fontSize))

	rl.DrawText(c.Label, textX, textY, fontSize, c.Color)
}

func (v *Hook) Connect(val Component) {
	switch c := val.(type) {
	case *QubitsSystem:
		c.removeFromHook()
		c.Center = v.Center
		v.IsHooked = true
		v.TargetID = c.ID
		c.HookID = v.ID
		c.SetWeight(0)
	case *QubitDeterminator:
		c.removeFromHook()
		c.Center = v.Center
		v.IsHooked = true
		v.TargetID = c.ID
		c.HookID = v.ID
		c.SetWeight(0)
	}
}

func (v *Hook) ConnectInfo(val Component) {
	switch c := val.(type) {
	case *QubitsSystem:
		//c.removeFromHook()
		c.Center = v.Center
		v.IsHooked = true
		v.TargetID = c.ID
		c.InfoHookID = v.ID
		c.SetWeight(0)
	}
}

func (v *Hook) Disconnect() {
	if v.TargetID == 0 {
		return
	}
	tmp := utils.GetObjectFromID(v.TargetID)
	v.IsHooked = false
	v.TargetID = 0
	switch c := tmp.(type) {
	case *QubitsSystem:
		c.HookID = 0
		c.SetWeight(glob.QubitDeterminatorWeight)
	case *QubitDeterminator:
		c.HookID = 0
		c.SetWeight(glob.QubitDeterminatorWeight)
	default:
	}
}

func (v *Hook) DisconnectAndKill() {
	if v.TargetID == 0 {
		return
	}
	tmp := utils.GetObjectFromID(v.TargetID)
	v.IsHooked = false
	v.TargetID = 0
	switch c := tmp.(type) {
	case *QubitsSystem:
		c.HookID = 0
		c.SetWeight(glob.QubitDeterminatorWeight)
		c.Kill()
	case *QubitDeterminator:
		c.HookID = 0
		c.SetWeight(glob.QubitDeterminatorWeight)
		c.Kill()
	default:
	}
}
