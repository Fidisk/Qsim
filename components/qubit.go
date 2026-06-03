package components

import (
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	attr "qsim/qubits/attributes"
)

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

	// --- FIELDS FOR SPLITTING MECHANIC ---
	ExpandedOffset rl.Vector2 // Center point of this sub-circle relative to system center
	CurrentPos     rl.Vector2 // Calculated visual center frame-by-frame
	RenderRadius   float32    // Calculated visual inspection radius (sub-circle radius)
}

func (c *Qubit) buildAttr(representation int32, modifierID []int32, n int32) {
	c.sideCnt = 0
	c.r, c.b, c.g = 0, 0, 0
	colCnt := 0
	for i := int32(0); i < n; i++ {
		tmp := attr.AttributesManager.Get(modifierID[i])
		state := (representation >> i) & 1

		switch v := tmp.(type) {
		case *attr.Color:
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
	if colCnt > 0 {
		c.r /= int32(colCnt)
		c.b /= int32(colCnt)
		c.g /= int32(colCnt)
	}
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
	// Rotation on self
	q.Rotation += q.RotationDelta
	if q.Rotation < 0 {
		q.Rotation += 2 * math.Pi
	}
	if q.Rotation >= 2*math.Pi {
		q.Rotation -= 2 * math.Pi
	}

	// Orbital Angle speed loops
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

// CalculateCenter updates coordinate trajectories cleanly
func (q *Qubit) CalculateCenter(systemCenter rl.Vector2, baseSystemRadius float32, state int, isEjected bool, dynamicSystemRadius float32) {
	if state == 0 { // StateNormal
		q.RenderRadius = q.Size * baseSystemRadius
		
		pathRadius := q.Radius * baseSystemRadius
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

		q.CurrentPos = rl.Vector2{X: systemCenter.X + newX, Y: systemCenter.Y + newY}
	} else {
		// Fixed tracking circle radius when split open
		q.RenderRadius = 45.0

		var subCircleCenter rl.Vector2
		if state == 2 && isEjected {
			// Position exactly outside the dynamic boundary ring of the system container
			subCircleCenter = rl.Vector2{
				X: systemCenter.X + dynamicSystemRadius + q.RenderRadius + 25,
				Y: systemCenter.Y,
			}
		} else {
			subCircleCenter = rl.Vector2Add(systemCenter, q.ExpandedOffset)
		}

		// FORCED ROUND ORBIT: No distortion ratio vectors applied
		orbitRadius := q.RenderRadius * 0.55
		newX := orbitRadius * float32(math.Cos(float64(q.Angle)))
		newY := orbitRadius * float32(math.Sin(float64(q.Angle)))

		q.CurrentPos = rl.Vector2{X: subCircleCenter.X + newX, Y: subCircleCenter.Y + newY}
	}
}

func (q *Qubit) Draw() {
	rotationDeg := q.Rotation * 180 / float32(math.Pi)
	col := color.RGBA{
		R: uint8(q.r),
		G: uint8(q.g),
		B: uint8(q.b),
		A: 255,
	}

	particleVisualSize := q.Size * 25.0 
	if particleVisualSize < 10.0 {
		particleVisualSize = 10.0
	}
	rl.DrawPoly(q.CurrentPos, q.sideCnt, particleVisualSize, rotationDeg, col)
}