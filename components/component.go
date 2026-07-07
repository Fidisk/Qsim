package components

import rl "github.com/gen2brain/raylib-go/raylib"

type PlaceholderWindow interface {
	GetElement() []Component
	PushComponent(...Component)
	DeleteChildWithID(...int32)
}

type Component interface {
	Draw()
	Update(rl.Vector2, bool, *bool)
	PostUpdate()
	IsHoldingCursor() bool
	GetParent() PlaceholderWindow
	SetParent(PlaceholderWindow)

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
	GetID() int32
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
