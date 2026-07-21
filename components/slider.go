package components

import (
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Slider is a horizontal draggable track reporting values in [0,1]
// (e.g. a timeline scrubber). Ticks optionally marks positions along it.
type Slider struct {
	*Circle
	ID            int32
	Width, Height float32
	Value         float32
	Ticks         []float32
	OnChange      func(v float32)
	OnRelease     func(v float32)
	held          bool
	hovered       bool
}

// NewSlider creates a slider centered at (x, y).
func NewSlider(x, y, width, height float32, color rl.Color, onChange, onRelease func(float32)) *Slider {
	s := &Slider{
		Circle:    NewCircle(x, y, max(width, height)/2, color),
		Width:     width,
		Height:    height,
		OnChange:  onChange,
		OnRelease: onRelease,
	}
	s.ID = utils.GenerateID(s)
	s.SetWeight(0)
	return s
}

func (s *Slider) GetID() int32 {
	return s.ID
}

func (s *Slider) trackRect() rl.Rectangle {
	return rl.NewRectangle(s.Center.X-s.Width/2, s.Center.Y-s.Height/2, s.Width, s.Height)
}

// Update handles mouse interaction, respecting cursor availability.
func (s *Slider) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	rect := s.trackRect()
	// Slightly larger hitbox for easier grabbing
	hit := rl.NewRectangle(rect.X, rect.Y-6, rect.Width, rect.Height+12)
	over := rl.CheckCollisionPointRec(worldMouse, hit)
	s.hovered = over && !s.held

	if over {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (s.held || *isCursorAvailable) {
			s.held = true
			*isCursorAvailable = false
		}
	}

	if s.held {
		v := (worldMouse.X - rect.X) / s.Width
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		if v != s.Value {
			s.Value = v
			if s.OnChange != nil {
				s.OnChange(v)
			}
		}
	}

	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && s.held {
		s.held = false
		*isCursorAvailable = true
		if s.OnRelease != nil {
			s.OnRelease(s.Value)
		}
	}
}

func (s *Slider) Draw() {
	rect := s.trackRect()

	// Track background and filled portion (rounded)
	rl.DrawRectangleRounded(rect, 0.5, 4, darken(s.Color, 0.7))
	if s.Value > 0 {
		fill := rl.NewRectangle(rect.X, rect.Y, rect.Width*s.Value, rect.Height)
		rl.DrawRectangleRounded(fill, 0.5, 4, rl.Fade(rl.Gold, 0.9))
	}

	// Tick marks (e.g. gate boundaries)
	for _, t := range s.Ticks {
		if t <= 0 || t >= 1 {
			continue
		}
		x := rect.X + rect.Width*t
		rl.DrawLineEx(
			rl.NewVector2(x, rect.Y-3),
			rl.NewVector2(x, rect.Y+rect.Height+3),
			1, rl.Fade(rl.White, 0.5),
		)
	}

	// Border
	rl.DrawRectangleRoundedLinesEx(rect, 0.5, 4, 1, darken(s.Color, 0.3))

	// Playhead
	kx := rect.X + rect.Width*s.Value
	rl.DrawLineEx(
		rl.NewVector2(kx, rect.Y-4),
		rl.NewVector2(kx, rect.Y+rect.Height+4),
		1.5, rl.Fade(rl.Gold, 0.6),
	)

	// Knob
	kr := s.Height * 0.9
	if s.hovered || s.held {
		kr = s.Height * 1.2
	}
	knob := rl.NewVector2(kx, s.Center.Y)
	rl.DrawCircleV(knob, kr, rl.Gold)
	ring := rl.Fade(rl.White, 0.4)
	if s.hovered || s.held {
		ring = rl.Fade(rl.White, 0.9)
	}
	rl.DrawCircleLines(int32(knob.X), int32(knob.Y), kr, ring)
}
