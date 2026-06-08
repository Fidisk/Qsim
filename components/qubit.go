package components

import (
	"fmt"
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"qsim/config"
	attr "qsim/qubits/attributes"
)

//Why tf did I want to make a esclipse, a circle work lmao
//Greed doomed us all

// Lmao this is even dumber, basically I draw a circle and squeeze it to make a eclipse
// Sound chopped af
type Qubit struct {
	Rotation          float32
	RotationDelta     float32
	PathRotation      float32
	PathRotationDelta float32
	Radius            float32
	Angle             float32
	AngleDelta        float32
	Size              float32
	Ratio             float32

	r       int32
	g       int32
	b       int32
	sideCnt int32
}

func (c *Qubit) buildAttr(representation int32, modifierID []int32, n int32) {
	c.sideCnt = 0
	c.r, c.b, c.g = 0, 0, 0
	colCnt := 0
	for i := int32(0); i < n; i++ {
		tmp := attr.AttributesManager.Get(modifierID[i])
		state := (representation >> (n - i - 1)) & 1

		switch v := tmp.(type) {
		case *attr.Color:
			if state == 0 {
				continue
			}
			colCnt++
			c.r += v.R * state
			c.g += v.G * state
			c.b += v.B * state
		case *attr.Side:
			if state != 0 {
				c.sideCnt += v.SideCntPositive
			} else {
				c.sideCnt += v.SideCntNegative
			}
		default:
		}
	}
	if colCnt == 0 {
		colCnt = 1
	}
	//c.r /= int32(colCnt)
	//c.b /= int32(colCnt)
	//c.g /= int32(colCnt)
	fmt.Println("Qubit ", c.r, c.g, c.b, representation)
}

func NewQubit(rotation, rotationDelta, pathRotation, pathRotationDelta, radius, angle, angleDelta, size, ratio float32, representation int32, modifierID []int32, n int32) *Qubit {
	tmp := Qubit{
		Rotation:          rotation,
		RotationDelta:     rotationDelta,
		PathRotation:      pathRotation,
		PathRotationDelta: pathRotationDelta,
		Radius:            radius,
		Angle:             angle,
		AngleDelta:        angleDelta,
		Size:              size,
		Ratio:             ratio,
	}
	tmp.buildAttr(representation, modifierID, n)
	return &tmp
}

func (q *Qubit) Update() {
	if config.QubitSystemStatePause {
		return
	}
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

	q.PathRotation += q.PathRotationDelta
	if q.PathRotation < 0 {
		q.PathRotation += 2 * math.Pi
	}
	if q.PathRotation >= 2*math.Pi {
		q.PathRotation -= 2 * math.Pi
	}
}

func (q *Qubit) Draw(center rl.Vector2, radius float32) {
	// 1. Position on the circular path (center of the square)
	pathRadius := q.Radius * radius
	xOffset := pathRadius * float32(math.Cos(float64(q.Angle)))
	yOffset := pathRadius * float32(math.Sin(float64(q.Angle)))

	if q.Ratio < 1 {
		xOffset *= q.Ratio
	} else {
		yOffset /= q.Ratio
	}

	rot := float64(q.PathRotation)
	cosRot := float32(math.Cos(rot))
	sinRot := float32(math.Sin(rot))

	newX := xOffset*cosRot - yOffset*sinRot
	newY := xOffset*sinRot + yOffset*cosRot
	xOffset, yOffset = newX, newY

	pos := rl.Vector2{
		X: center.X + xOffset,
		Y: center.Y + yOffset,
	}

	// 2. Circumradius = q.Size * radius (the "perfect circle" radius)
	circumRadius := q.Size * radius

	// 3. Draw a 4‑sided regular polygon (square) centered at pos
	//    rotationDeg is q.Rotation in degrees (already matches DrawPoly's unit)
	rotationDeg := q.Rotation * 180 / float32(math.Pi)

	col := color.RGBA{
		R: uint8(q.r),
		G: uint8(q.g),
		B: uint8(q.b),
		A: 255,
	}

	rl.DrawPoly(pos, q.sideCnt, circumRadius, rotationDeg, col)
}
