package components

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//Why tf did I want to make a esclipse, a circle work lmao
//Greed doomed us all

// Lmao this is even dumber, basically I draw a circle and squeeze it to make a eclipse
// Sound chopped af
type Qubit struct {
	Rotation      float32
	RotationDelta float32
	Radius        float32
	Angle         float32
	AngleDelta    float32
	Size          float32
	Ratio         float32
}

func NewQubit(rotation, rotationDelta, radius, angle, angleDelta, size, ratio float32) *Qubit {
	return &Qubit{
		Rotation:      rotation,
		RotationDelta: rotationDelta,
		Radius:        radius,
		Angle:         angle,
		AngleDelta:    angleDelta,
		Size:          size,
		Ratio:         ratio,
	}
}

func (q *Qubit) Update() {
	// Rotation
	q.Rotation += q.RotationDelta
	if q.Rotation < 0 {
		q.Rotation += 2 * math.Pi
	}
	if q.Rotation >= 2*math.Pi {
		q.Rotation -= 2 * math.Pi
	}

	// Angle
	q.Angle += q.AngleDelta
	if q.Angle < 0 {
		q.Angle += 2 * math.Pi
	}
	if q.Angle >= 2*math.Pi {
		q.Angle -= 2 * math.Pi
	}
}

func (q *Qubit) Draw(center rl.Vector2, radius float32) {
	// 1. Position on the circular path (center of the square)
	pathRadius := q.Radius
	xOffset := pathRadius * float32(math.Cos(float64(q.Angle)))
	yOffset := pathRadius * float32(math.Sin(float64(q.Angle)))
	pos := rl.Vector2{
		X: center.X + xOffset,
		Y: center.Y + yOffset,
	}

	// 2. Circumradius = q.Size * radius (the "perfect circle" radius)
	circumRadius := q.Size * radius

	// 3. Draw a 4‑sided regular polygon (square) centered at pos
	//    rotationDeg is q.Rotation in degrees (already matches DrawPoly's unit)
	rotationDeg := q.Rotation * 180 / float32(math.Pi)

	rl.DrawPoly(pos, 6, circumRadius, rotationDeg, rl.Red)
}
