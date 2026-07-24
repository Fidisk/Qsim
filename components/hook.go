package components

import (
	"qsim/config"
	"qsim/globals"
	glob "qsim/globals"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Hook struct {
	Circle
	IsHooked          bool
	ID                int32
	TargetID          int32
	IsOutput          bool
	Label             string
	AllowQubitSystem  bool
	AllowLogicalBit   bool

	// Hidden hooks are not drawn and cannot be interacted with (e.g. the
	// collapse gate's unused remainder output).
	Hidden bool
	// Tooltip shows in the tooltip window when an empty hook is hovered
	// with nothing held. Leave empty for no tooltip.
	Tooltip string

	// justSpawned triggers a one-shot auto-connect attempt on the first
	// Update, so a hook spawned on top of a qubit determinator hooks up.
	justSpawned bool
	// skipNextZip prevents auto-reconnection after a double-click detach.
	skipNextZip bool
}

func NewHook(x, y, radius float32, color rl.Color) *Hook {
	tmp := Hook{
		Circle:      *NewCircle(x, y, radius, color),
		justSpawned: true,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight((globals.HookWeight))
	return &tmp
}

func NewOutputHook(x, y, radius float32, color rl.Color) *Hook {
	tmp := Hook{
		Circle:      *NewCircle(x, y, radius, color),
		IsOutput:    true,
		justSpawned: true,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight((globals.HookWeight))
	return &tmp
}

// NewLogicalHook creates an input hook that only accepts logical bits.
func NewLogicalHook(x, y float32) *Hook {
	h := NewHook(x, y, globals.HookRadius, config.HookColor)
	h.AllowLogicalBit = true
	return h
}

// NewLogicalOutputHook creates an output hook that emits logical bits.
func NewLogicalOutputHook(x, y float32) *Hook {
	h := NewOutputHook(x, y, globals.OutputHookRadius, config.OutputHookColor)
	h.AllowLogicalBit = true
	return h
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
		c.VirtualCenter = c.Center
	}
}

func (c *Hook) onRelease() {

}

func (c *Hook) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if c.Hidden {
		return
	}
	if c.justSpawned {
		c.justSpawned = false
		c.zipToDeterminator()
		c.zipToQubitSystem()
		c.zipToLogicalBit()
	}

	// Guide tooltip: hovering an empty hook with nothing held explains
	// what to plug in.
	if !c.IsHooked && c.Tooltip != "" && !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
			glob.TooltipText = c.Tooltip
		}
	}
	if c.IsHooked {
		if utils.GetObjectFromID(c.TargetID) == nil {
			c.Disconnect()
			//Should do smth about physics here, but meh
			return
		}

		tmp := utils.GetObjectFromID(c.TargetID)
		t, ok := tmp.(Component)
		if !ok {
			c.Disconnect()
			return
		}
		// Logical bits are normally anchored to their hook so they move with the
		// gate, but when the user is actively dragging the bit, the hook follows
		// the bit instead so the wire stays connected.
		if _, isLogicalBit := tmp.(*LogicalBit); !isLogicalBit || t.GetCircle().IsDragging() {
			c.Center = t.GetCircle().Center
		}
		t.AddForce(c.GetCircle().GetForce())
		c.GetCircle().ClearForce()
		return
	}

	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			if utils.IsMouseState(glob.MouseStateNormal) && c.DoubleClicked(worldMouse) {
				// double-click: detach only if this hook is connected to a QubitDeterminator
				if c.IsHooked && !c.IsOutput {
					if _, ok := utils.GetObjectFromID(c.TargetID).(*QubitDeterminator); ok {
						c.Disconnect()
						c.skipNextZip = true
						*isCursorAvailable = false
						return
					}
				}
			}
			c.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		c.dragging = false
		c.holdingCursor = false
		*isCursorAvailable = true
		c.Center = c.VirtualCenter
		if !c.skipNextZip {
			c.zipToDeterminator()
			c.zipToQubitSystem()
			c.zipToLogicalBit()
		} else {
			c.skipNextZip = false
		}
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

func (c *Hook) Draw() {
	if c.Hidden {
		return
	}
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

	// Port outline, then the "+" cross. Logical hooks (logical-bit only) get a
	// double ring, following the double-line convention for logical carriers.
	if c.AllowLogicalBit {
		rl.DrawCircleLines(int32(c.Center.X), int32(c.Center.Y), c.Radius, rl.Fade(c.Color, 0.6))
		rl.DrawCircleLines(int32(c.Center.X), int32(c.Center.Y), c.Radius-4, rl.Fade(c.Color, 0.6))
	} else {
		rl.DrawCircleLines(int32(c.Center.X), int32(c.Center.Y), c.Radius, rl.Fade(c.Color, 0.6))
	}

	// Draw the horizontal bar
	rl.DrawLineEx(left, right, thickness, c.Color)

	// Draw the vertical bar
	rl.DrawLineEx(top, bottom, thickness, c.Color)

	fontSize := int32(14) // adjust as needed
	offsetX := float32(8) // how far left from the center
	offsetY := float32(8) // how far up from the center
	textX := int32(c.Center.X - c.Radius - offsetX)
	textY := int32(c.Center.Y - c.Radius - offsetY - float32(fontSize))

	// Backdrop so the label stays readable over wires and the grid
	labelW := rl.MeasureText(c.Label, fontSize)
	bg := rl.NewRectangle(float32(textX)-3, float32(textY)-2, float32(labelW)+6, float32(fontSize)+4)
	rl.DrawRectangleRec(bg, rl.Fade(rl.Black, 0.55))
	rl.DrawText(c.Label, textX, textY, fontSize, c.Color)
}

func (c *Hook) DrawGhost() {
	if c.Hidden {
		return
	}
	if c.IsHooked {
		return
	}
	ghostColor := rl.Fade(c.Color, 0.3)

	if c.AllowLogicalBit {
		rl.DrawCircleLines(int32(c.Center.X), int32(c.Center.Y), c.Radius, ghostColor)
		rl.DrawCircleLines(int32(c.Center.X), int32(c.Center.Y), c.Radius-4, ghostColor)
	} else {
		rl.DrawCircleLines(int32(c.Center.X), int32(c.Center.Y), c.Radius, ghostColor)
	}

	left := rl.Vector2{X: c.Center.X - c.Radius, Y: c.Center.Y}
	right := rl.Vector2{X: c.Center.X + c.Radius, Y: c.Center.Y}
	top := rl.Vector2{X: c.Center.X, Y: c.Center.Y - c.Radius}
	bottom := rl.Vector2{X: c.Center.X, Y: c.Center.Y + c.Radius}

	rl.DrawLineEx(left, right, 4, ghostColor)
	rl.DrawLineEx(top, bottom, 4, ghostColor)

	fontSize := int32(14)
	offsetX := float32(8)
	offsetY := float32(8)
	textX := int32(c.Center.X - c.Radius - offsetX)
	textY := int32(c.Center.Y - c.Radius - offsetY - float32(fontSize))
	rl.DrawText(c.Label, textX, textY, fontSize, ghostColor)
}

// hookOwner is any component that owns a set of hooks (Gate, CollapseGate),
// so the zip helpers can treat them uniformly.
type hookOwner interface {
	GetHooks() []*Hook
}

// zipToDeterminator connects the hook to a nearby free qubit determinator.
// It mirrors QubitDeterminator.zipToHook from the hook side and runs when the
// hook is released after a drag or right after it spawns. Output-style hooks
// (IsOutput or AllowQubitSystem) are skipped: those connect to qubit systems.
func (c *Hook) zipToDeterminator() {
	if c.IsOutput || c.AllowQubitSystem || c.AllowLogicalBit || c.IsHooked {
		return
	}
	for _, obj := range utils.ObjectList {
		d, ok := obj.(*QubitDeterminator)
		if !ok || d == nil {
			continue
		}
		if d.HookID != 0 && d.HookID != c.ID {
			continue
		}
		// Skip determinators that are hidden or not placed in a window
		// (e.g. spawn previews, which never get a parent).
		qp := d.GetQubitParent()
		if qp == nil || qp.GetParent() == nil || !qp.isDeterminatorVisible(d) {
			continue
		}
		if utils.Dist(d.Center, c.Center) <= glob.HookDist {
			c.Connect(d)
			return
		}
	}
}

// zipToQubitSystem connects an info-style hook (AllowQubitSystem, not an
// output) to a nearby qubit system. It mirrors QubitsSystem.zipToHook from
// the hook side and runs when the hook is released after a drag or right
// after it spawns, so dragging the hook onto a system auto-connects.
func (c *Hook) zipToQubitSystem() {
	if !c.AllowQubitSystem || c.AllowLogicalBit || c.IsOutput || c.IsHooked {
		return
	}
	for _, obj := range utils.ObjectList {
		qs, ok := obj.(*QubitsSystem)
		if !ok || qs == nil {
			continue
		}
		// One info link per system; re-connecting to the same hook is fine.
		if qs.InfoHookID != 0 && qs.InfoHookID != c.ID {
			continue
		}
		// Skip systems that are not placed in a window (spawn previews).
		if qs.GetParent() == nil {
			continue
		}
		if utils.Dist(qs.Center, c.Center) <= glob.HookDist {
			c.ConnectInfo(qs)
			return
		}
	}
}

// zipToLogicalBit connects this hook to a nearby free logical bit.
func (c *Hook) zipToLogicalBit() {
	if !c.AllowLogicalBit || c.IsHooked {
		return
	}
	for _, obj := range utils.ObjectList {
		lb, ok := obj.(*LogicalBit)
		if !ok || lb == nil {
			continue
		}
		if lb.HasHook(c.ID) {
			continue
		}
		// A bit is driven by at most one gate output.
		if c.IsOutput && lb.HasOutputLink() {
			continue
		}
		if lb.GetParent() == nil {
			continue
		}
		if utils.Dist(lb.Center, c.Center) <= glob.HookDist {
			c.Connect(lb)
			return
		}
	}
}

func (v *Hook) Connect(val Component) {
	switch c := val.(type) {
	case *QubitsSystem:
		c.removeFromHook()
		delta := v.Center.Subtract(c.Center)
		c.Center = v.Center
		c.VirtualCenter = v.Center
		// Move the determinators along with the system so they keep their
		// layout (right of the state grid) relative to the new position.
		// Hooked determinators stay anchored to their own hooks.
		for _, d := range c.QubitDeterminatorList {
			if d.HookID != 0 {
				continue
			}
			d.Center = d.Center.Add(delta)
			d.VirtualCenter = d.Center
		}
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
	case *LogicalBit:
		// Logical bits fan out: linking another hook never severs the
		// bit's existing connections.
		c.AddHook(v.ID)
		c.Center = v.Center
		v.IsHooked = true
		v.TargetID = c.ID
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
		// Also drop any info link (e.g. InfoTable hook) so the system can be
		// auto-connected again later.
		if c.InfoHookID == v.ID {
			c.InfoHookID = 0
		}
		c.SetWeight(glob.QubitDeterminatorWeight)
	case *QubitDeterminator:
		c.HookID = 0
		c.SetWeight(glob.QubitDeterminatorWeight)
	case *LogicalBit:
		c.RemoveHook(v.ID)
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
	case *LogicalBit:
		c.SetWeight(glob.QubitDeterminatorWeight)
		c.Kill()
	default:
	}
}
