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
	ID    int32
	Value int32 // 0 or 1
	// HookID is the primary hook link; LinkedHookIDs holds the additional
	// fan-out links (see logicalHook.go).
	HookID        int32
	LinkedHookIDs []int32
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
		lb.disconnectAllHooks()
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
		// Keep every linked hook exactly at the bit so the wires stay connected
		// even when dragging fast (the hook update runs before the bit update
		// in the gate).
		for _, h := range lb.LinkedHooks() {
			h.Center = lb.Center
			h.VirtualCenter = lb.Center
		}
	} else {
		lb.DecayForce()
		lb.ApplyForce()
		// With a single link the bit is anchored to its hook so it moves with
		// the gate. With several links it floats freely: each gate's hook pull
		// positions it between its drivers and consumers.
		if lb.HookID != 0 && len(lb.LinkedHookIDs) == 0 {
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
			// Skip hooks the bit is already linked to: they follow the bit
			// while dragging and would always win the distance check.
			if lb.HasHook(v.ID) {
				continue
			}
			if !v.AllowLogicalBit || v.Hidden || v.IsHooked {
				continue
			}
			// A bit is driven by at most one gate output.
			if v.IsOutput && lb.HasOutputLink() {
				continue
			}
			if utils.Dist(v.Center, lb.Center) <= glob.HookDist {
				gotHooked = true
				v.Connect(lb)
			}
		case hookOwner:
			for _, d2 := range v.GetHooks() {
				if lb.HasHook(d2.ID) {
					continue
				}
				if !d2.AllowLogicalBit || d2.Hidden || d2.IsHooked {
					continue
				}
				if d2.IsOutput && lb.HasOutputLink() {
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
	// Releasing the bit anywhere but on a different hook keeps its current
	// connection: while not dragging, the bit is anchored to its hook and
	// simply snaps back. Detaching is done explicitly via the detach mouse
	// mode, not by dragging.
	if !gotHooked {
		lb.zipToBit()
	}
}

// isOutputBit reports whether the bit is actively driven by a gate, i.e. any
// of its linked hooks is a gate output.
func (lb *LogicalBit) isOutputBit() bool {
	return lb.HasOutputLink()
}

// zipToBit merges the bit into a nearby logical bit: bits accept other bits
// the way hooks do. The output bit (if any) survives with its value and takes
// over the loser's hooks; two output bits refuse to merge.
func (lb *LogicalBit) zipToBit() bool {
	qp := lb.GetParent()
	if qp == nil {
		return false
	}
	for _, d := range qp.GetElement() {
		other, ok := d.(*LogicalBit)
		if !ok || other == nil || other.ID == lb.ID {
			continue
		}
		if utils.Dist(other.Center, lb.Center) <= glob.HookDist {
			return lb.mergeInto(other)
		}
	}
	return false
}

// mergeInto merges the dragged bit with other. The output bit wins and keeps
// its value; the loser is removed and the winner inherits all of its hooks.
// When neither bit is an output, the target (other) wins. Two output bits
// cannot merge.
func (lb *LogicalBit) mergeInto(other *LogicalBit) bool {
	if lb.isOutputBit() && other.isOutputBit() {
		return false
	}
	winner, loser := other, lb
	if lb.isOutputBit() {
		winner, loser = lb, other
	}

	// Remember the loser's hooks before killing it: the winner inherits them.
	loserHooks := loser.LinkedHooks()
	loserCenter := loser.Center
	loser.Kill()
	for _, h := range loserHooks {
		h.Connect(winner)
	}
	winner.Center = loserCenter
	winner.VirtualCenter = loserCenter
	return true
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
	lb.disconnectAllHooks()
	parent := lb.GetParent()
	if parent != nil {
		parent.DeleteChildWithID(lb.ID)
	}
}
