package components

import rl "github.com/gen2brain/raylib-go/raylib"

type PlaceholderWindow interface {
	GetElement() []Component
	PushComponent(...Component)
}

type Component interface {
	Draw()
	Update(rl.Vector2, bool, *bool)
	PostUpdate()
	IsHoldingCursor() bool
	GetParent() PlaceholderWindow
	SetParent(PlaceholderWindow)
	GetBounds() rl.Rectangle
	//Bloat -w-
	AddForce(rl.Vector2)
	DecayForce()
	ApplyForce()
	AntiGravity(Component)
	Gravity(Component)
	GetCenter() rl.Vector2
	SetCenter(rl.Vector2)
	GetCircle() *Circle
	SetWeight(float32)
	GetWeight() float32
}

type WindowComponent struct {
	Component
	holdingCursor bool
	parent        PlaceholderWindow
}

func (c *WindowComponent) IsHoldingCursor() bool {
	return c.holdingCursor
}

func (c *WindowComponent) SetParent(p PlaceholderWindow) {
	c.parent = p
}

func (c *WindowComponent) GetParent() PlaceholderWindow {
	return c.parent
}

// ============================================================================
// HOVER DETECTION UTILITY
// ============================================================================

func GetHoveredComponent(elements []Component) Component {
	mousePos := rl.GetMousePosition()

	for i := len(elements) - 1; i >= 0; i-- {
		el := elements[i]
		if el == nil {
			continue
		}

		if circlePtr := el.GetCircle(); circlePtr != nil {
			if rl.CheckCollisionPointCircle(mousePos, el.GetCenter(), circlePtr.Radius) {
				return el
			}
		} else {
			if rl.CheckCollisionPointRec(mousePos, el.GetBounds()) {
				return el
			}
		}
	}

	return nil
}