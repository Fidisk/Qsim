package components

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand/v2"
	glob "qsim/globals"
	qub "qsim/qubits"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type QubitsSystem struct {
	Circle
	QubitList             []*Qubit
	Origin                *qub.QubitStateManager
	BitList               []int32 //Overkill
	ID                    int32
	HookID                int32
	QubitDeterminatorList []*QubitDeterminator
	rows                  int32
	cols                  int32
	startX                float32
	startY                float32
}

func NewQubitsSystem(x, y, radius float32, color rl.Color) *QubitsSystem {
	tmp := QubitsSystem{
		Circle:    *NewCircle(x, y, radius, color),
		QubitList: nil,
		Origin:    nil,
		HookID:    0,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight(glob.QubitSystemWeight)
	return &tmp
}

func (c *QubitsSystem) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, rl.Rectangle{X: c.startX, Y: c.startY, Width: float32(c.cols) * glob.QubitSystemCellWidth, Height: float32(c.rows) * glob.QubitSystemCellHeight})
}

func (c *QubitsSystem) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	for _, d := range c.QubitList {
		d.Update()
	}

	// collision check using world coordinates
	if c.CheckCollide(worldMouse) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			c.dragging = true
			*isCursorAvailable = false
			c.holdingCursor = true
			c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		c.dragging = false
		c.holdingCursor = false
		*isCursorAvailable = true

		c.zipToHook()
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
		c.ClearForce()
	} else {
		c.DecayForce()
		c.ApplyForce()
	}

	for _, d := range c.QubitDeterminatorList {
		c.pullToQubitSystem(d)

		d.Update(worldMouse, holdingCursor, isCursorAvailable)
	}
}

func (c *QubitsSystem) Draw() {
	for _, d := range c.QubitDeterminatorList {
		rl.DrawLineV(c.Center, d.Center, c.Color)
	}

	// New: rectangle split into 2^ceil(exp/2) columns × 2^floor(exp/2) c.rows
	// total cells = 2^(ceil+floor) = 2^exp
	exp := c.Origin.Size            // natural number
	n := glob.QubitSystemCellWidth  // width of each cell
	m := glob.QubitSystemCellHeight // height of each cell

	// Compute exponents for width and height
	widthExp := (exp + 1) / 2 // ceil(exp/2)
	heightExp := exp / 2      // floor(exp/2)

	c.cols = int32(1 << widthExp)  // 2^ceil(exp/2)
	c.rows = int32(1 << heightExp) // 2^floor(exp/2)

	totalWidth := float32(c.cols * int32(n))
	totalHeight := float32(c.rows * int32(m))

	c.startX = c.Center.X - totalWidth/2
	c.startY = c.Center.Y - totalHeight/2

	rl.DrawRectangle(
		int32(c.startX), int32(c.startY),
		int32(totalWidth), int32(totalHeight),
		glob.ColorBg,
	)

	// Outer border
	rl.DrawRectangleLines(
		int32(c.startX), int32(c.startY),
		int32(totalWidth), int32(totalHeight),
		c.Color,
	)

	// Vertical grid lines (between columns)
	for i := int32(1); i < c.cols; i++ {
		x := c.startX + float32(i*int32(n))
		rl.DrawLineV(
			rl.Vector2{X: x, Y: c.startY},
			rl.Vector2{X: x, Y: c.startY + totalHeight},
			c.Color,
		)
	}

	// Horizontal grid lines (between c.rows)
	for j := int32(1); j < c.rows; j++ {
		y := c.startY + float32(j)*float32(m)
		rl.DrawLineV(
			rl.Vector2{X: c.startX, Y: y},
			rl.Vector2{X: c.startX + totalWidth, Y: y},
			c.Color,
		)
	}

	// Draw number and name in each cell
	for row := int32(0); row < c.rows; row++ {
		for col := int32(0); col < c.cols; col++ {
			// Cell center
			cellCenterX := c.startX + float32(col)*float32(n) + float32(n)/2
			cellCenterY := c.startY + float32(row)*float32(m) + float32(m)/2

			// Linear index (row-major order)
			index := row*c.cols + col
			numberStr := fmt.Sprintf("%v", c.Origin.Amptitude[index]) // now properly formatted
			nameStr := fmt.Sprintf("Q%d", index)

			// Draw number slightly above center
			numFontSize := int32(20)
			numWidth := rl.MeasureText(numberStr, numFontSize)
			rl.DrawText(numberStr,
				int32(cellCenterX)-numWidth/2,
				int32(cellCenterY)-numFontSize-2, // 2px gap
				numFontSize,
				c.Color,
			)

			// Draw name slightly below center
			nameFontSize := int32(14)
			nameWidth := rl.MeasureText(nameStr, nameFontSize)
			rl.DrawText(nameStr,
				int32(cellCenterX)-nameWidth/2,
				int32(cellCenterY)+2,
				nameFontSize,
				c.Color,
			)
		}
	}

	// (Commented‑out old drawing code remains unchanged)
	/*
	   rl.DrawCircleV(c.Center, c.Radius, glob.ColorBg)
	   rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)
	   for _, d := range c.QubitList {
	       d.Draw(c.Center, c.Radius)
	   }
	*/

	for _, d := range c.QubitDeterminatorList {
		d.Draw()
	}
}

func (c *QubitsSystem) Assign(p *qub.QubitStateManager) {
	//This do not defer the Origin
	c.Origin = p

	//The downfall of OOP
	c.QubitList = nil

	for i := range p.Size {
		q := NewQubitDeterminator(c.Center.X, c.Center.Y, c.Radius/float32(2), rl.Purple, p.ModifierID[i])

		q.QubitSystemID = c.ID

		c.QubitDeterminatorList = append(c.QubitDeterminatorList, q)
	}

	for i := range p.Amptitude {
		rotation := rand.Float32() * 2 * math.Pi
		angle := rand.Float32() * 2 * math.Pi

		magnitude := rand.Float32()*0.05 + 0.05 // 0.05 … 0.1
		if rand.IntN(2) == 0 {
			magnitude = -magnitude
		}
		rotationDelta := magnitude

		magnitude = rand.Float32()*0.05 + 0.05
		if rand.IntN(2) == 0 {
			magnitude = -magnitude
		}
		angleDelta := magnitude

		ratio := rand.Float32()*2.5 + 0.5 // 0.5 … 10
		if rand.IntN(2) == 0 {
			ratio = 1.0 / ratio
		}

		pathRotation := rand.Float32() * 2 * math.Pi
		pathRotationDelta := rand.Float32()*0.001 + 0.001

		size := float32(cmplx.Abs(complex128(p.Amptitude[i]))) / 2.0
		radius := 0.95 - size

		q := NewQubit(rotation, rotationDelta, pathRotation, pathRotationDelta, radius, angle, angleDelta, size, ratio, int32(i), p.ModifierID, int32(len(p.ModifierID)))
		// Use q (e.g., append to QubitList)
		c.QubitList = append(c.QubitList, q)
	}
}

func (c *QubitsSystem) zipToHook() {
	return
	tmp := c.GetParent()
	ele := tmp.GetElement()
	gotHooked := false
	for _, d := range ele {
		switch v := d.(type) {
		case *Hook:
			if utils.Dist(v.Center, c.Center) <= glob.HookDist && (!v.IsHooked || v.TargetID == c.ID) && !gotHooked {
				gotHooked = true
				v.Connect(c)
			}
		case *Gate:
			for _, d2 := range v.HookList {
				if utils.Dist(d2.Center, c.Center) <= glob.HookDist && (!d2.IsHooked || d2.TargetID == c.ID) && !gotHooked {
					gotHooked = true
					d2.Connect(c)
				}
			}
		default:
		}
		if gotHooked {
			break
		}
	}
	if !gotHooked && c.HookID != 0 {
		c.removeFromHook()
	}
}

func (c *QubitsSystem) removeFromHook() {
	if c.HookID == 0 {
		return
	}
	tmp := utils.GetObjectFromID(c.HookID)
	c.HookID = 0
	c.SetWeight(glob.QubitSystemWeight)
	switch t := tmp.(type) {
	case *Hook:
		t.IsHooked = false
		t.TargetID = 0
	default:
	}
}

func (c *QubitsSystem) split() {

}

func (c *QubitsSystem) PostUpdate() {
	for i := range c.QubitDeterminatorList {
		for j := i + 1; j < len(c.QubitDeterminatorList); j++ {
			c.QubitDeterminatorList[i].GetCircle().AntiGravity(c.QubitDeterminatorList[j].GetCircle())
		}
	}
}

func (c *QubitsSystem) pullToQubitSystem(d *QubitDeterminator) {
	val := utils.Dist(c.Center, d.Center) - glob.GateToHookDist

	if math.Abs(float64(val)) <= float64(glob.GateToHookGraceDist) {
		return
	}

	val = float32(math.Max(float64(val), float64(-100)))
	val = float32(math.Min(float64(val), float64(100)))

	tmp := c.Center.Subtract(d.Center).Normalize().Scale(val * glob.GateToHookPullCoeff)

	d.AddForce(tmp)
	c.AddForce(tmp.Scale(-1))
}
