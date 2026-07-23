package components

import (
	"fmt"
	"qsim/config"
	glob "qsim/globals"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// LogicalBit is a classical 0/1 bit produced by measurement gates. It looks
// like a qubit determinator but is rendered solid black for 0 and solid white
// for 1.
type LogicalBit struct {
	Circle
	ID     int32
	Value  int32 // 0 or 1
	HookID int32
}

func NewLogicalBit(x, y, radius float32, value int32) *LogicalBit {
	if value != 0 && value != 1 {
		value = 0
	}
	lb := &LogicalBit{
		Circle: *NewCircle(x, y, radius, rl.White),
		Value:  value,
	}
	lb.ID = utils.GenerateID(lb)
	lb.SetWeight(glob.QubitDeterminatorWeight)
	lb.updateColor()
	return lb
}

func (lb *LogicalBit) updateColor() {
	if lb.Value == 0 {
		lb.Color = rl.Black
	} else {
		lb.Color = rl.White
	}
}

func (lb *LogicalBit) GetID() int32 { return lb.ID }

func (lb *LogicalBit) SetValue(v int32) {
	if v != 0 && v != 1 {
		return
	}
	lb.Value = v
	lb.updateColor()
}

func (lb *LogicalBit) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateDetach):
		if lb.HookID != 0 {
			tmp := utils.GetObjectFromID(lb.HookID)
			if h, ok := tmp.(*Hook); ok {
				h.Disconnect()
			}
		}
	case utils.IsMouseState(glob.MouseStateErase):
		lb.Kill()
	default:
		lb.dragging = true
		*isCursorAvailable = false
		lb.holdingCursor = true
		lb.offset = rl.Vector2Subtract(lb.Center, worldMouse)
		lb.VirtualCenter = lb.Center
	}
}

func (lb *LogicalBit) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if rl.CheckCollisionPointCircle(worldMouse, lb.Center, lb.Radius) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			glob.TooltipText = fmt.Sprintf("%d", lb.Value)
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (lb.holdingCursor || *isCursorAvailable) {
			lb.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && lb.holdingCursor {
		lb.dragging = false
		lb.holdingCursor = false
		*isCursorAvailable = true
		lb.Center = rl.Vector2{
			X: utils.SnapToGrid(lb.VirtualCenter.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(lb.VirtualCenter.Y, config.SnapToGridInterval),
		}
		lb.VirtualCenter = lb.Center
		lb.zipToHook()
		lb.ClearForce()
	}
	if lb.dragging {
		raw := rl.Vector2Add(worldMouse, lb.offset)
		lb.VirtualCenter = rl.Vector2Lerp(lb.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		lb.Center = lb.VirtualCenter
		lb.ClearForce()
		// Keep the hook exactly at the bit so the wire stays connected even when
		// dragging fast (the hook update runs before the bit update in the gate).
		if lb.HookID != 0 {
			tmp := utils.GetObjectFromID(lb.HookID)
			if h, ok := tmp.(*Hook); ok {
				h.Center = lb.Center
				h.VirtualCenter = lb.Center
			}
		}
	} else {
		lb.DecayForce()
		lb.ApplyForce()
		// When hooked, the logical bit is anchored to its hook so it moves with
		// the gate or other owner.
		if lb.HookID != 0 {
			tmp := utils.GetObjectFromID(lb.HookID)
			if h, ok := tmp.(*Hook); ok {
				lb.Center = h.Center
				lb.VirtualCenter = h.Center
			}
		}
	}
}

func (lb *LogicalBit) zipToHook() {
	qp := lb.GetParent()
	if qp == nil {
		return
	}
	ele := qp.GetElement()
	gotHooked := false
	for _, d := range ele {
		switch v := d.(type) {
		case *Hook:
			// Skip the hook the bit is currently attached to: it follows the
			// bit while dragging, so it would always win the distance check
			// and the bit could never move to a different hook.
			if v.ID == lb.HookID {
				continue
			}
			if !v.AllowLogicalBit || v.Hidden || (v.IsHooked && v.TargetID != lb.ID) {
				continue
			}
			if utils.Dist(v.Center, lb.Center) <= glob.HookDist {
				gotHooked = true
				v.Connect(lb)
			}
		case hookOwner:
			for _, d2 := range v.GetHooks() {
				if d2.ID == lb.HookID {
					continue
				}
				if !d2.AllowLogicalBit || d2.Hidden || (d2.IsHooked && d2.TargetID != lb.ID) {
					continue
				}
				if utils.Dist(d2.Center, lb.Center) <= glob.HookDist {
					gotHooked = true
					d2.Connect(lb)
				}
			}
		default:
		}
		if gotHooked {
			break
		}
	}
	if !gotHooked && lb.HookID != 0 {
		lb.removeFromHook()
	}
}

func (lb *LogicalBit) removeFromHook() {
	if lb.HookID == 0 {
		return
	}
	tmp := utils.GetObjectFromID(lb.HookID)
	lb.HookID = 0
	lb.SetWeight(glob.QubitDeterminatorWeight)
	if h, ok := tmp.(*Hook); ok {
		h.IsHooked = false
		h.TargetID = 0
	}
}

func (lb *LogicalBit) Draw() {
	rl.DrawCircleV(lb.Center, lb.Radius, lb.Color)
	outline := rl.Gray
	if lb.Value == 0 {
		outline = rl.White
	}
	rl.DrawCircleLines(int32(lb.Center.X), int32(lb.Center.Y), lb.Radius, outline)
}

func (lb *LogicalBit) DrawGhost() {
	ghostColor := rl.Fade(lb.Color, 0.3)
	rl.DrawCircleLines(int32(lb.Center.X), int32(lb.Center.Y), lb.Radius, ghostColor)
}

func (lb *LogicalBit) GetChildCircles() []*Circle { return nil }

func (lb *LogicalBit) PostUpdate() {}

func (lb *LogicalBit) Kill() {
	if h, ok := utils.GetObjectFromID(lb.HookID).(*Hook); ok {
		h.Disconnect()
	}
	parent := lb.GetParent()
	if parent != nil {
		parent.DeleteChildWithID(lb.ID)
	}
}
