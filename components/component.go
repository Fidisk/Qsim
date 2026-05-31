package components

import rl "github.com/gen2brain/raylib-go/raylib"

type PlaceholderWindow interface {
	GetElement() []Component
}

type Component interface {
	Draw()
	Update(rl.Vector2, bool, *bool)
	PostUpdate()
	IsHoldingCursor() bool
	GetParent() PlaceholderWindow
	SetParent(PlaceholderWindow)
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
