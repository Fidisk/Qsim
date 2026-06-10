package components

import (
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

func (c *QubitsSystem) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	for _, d := range c.QubitList {
		d.Update()
	}

	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
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

/*-----------------------------------------

Move out the way, under construction :3

                        (       )
                          (      )
                           (     )
                            (    )
                             (   )
                              (  )
          _____________         _
          | |     | |          | |
          | |     | |          | |
          | | --- | |__________|_|___________
        __|   ___          _____________     \
       /  |   |_|         |   ________  |    |
       \__|  (owo)        |__|_______|__|    |             _
          |______________________________ ___|            //
          _______________________________ ______________ //
         / _    ____________________   _  \ ____________||
        | (_)  |____________________| (_)  |             \\
         \_______________________________ /               \\
-------------------------------------------------------------------
Stolen from Dan Carrion
*/

func (c *QubitsSystem) Draw() {
	for _, d := range c.QubitDeterminatorList {
		rl.DrawLineV(c.Center, d.Center, c.Color)
		d.Draw()
	}

	// New: centered rectangle split into 2^x × 2^x cells of size n × m
	exp := c.Origin.Size // x – natural number
	n := 100             // width of each cell
	m := 100             // height of each cell

	cellsPerSide := int32(1 << exp) // 2^x
	totalWidth := float32(cellsPerSide * int32(n))
	totalHeight := float32(cellsPerSide * int32(m))

	startX := c.Center.X - totalWidth/2
	startY := c.Center.Y - totalHeight/2

	// Outer border
	rl.DrawRectangleLines(
		int32(startX), int32(startY),
		int32(totalWidth), int32(totalHeight),
		c.Color,
	)

	// Vertical grid lines
	for i := int32(1); i < cellsPerSide; i++ {
		x := startX + float32(i*int32(n))
		rl.DrawLineV(
			rl.Vector2{X: x, Y: startY},
			rl.Vector2{X: x, Y: startY + totalHeight},
			c.Color,
		)
	}

	for j := int32(1); j < cellsPerSide; j++ {
		y := startY + float32(j)*float32(m)
		rl.DrawLineV(
			rl.Vector2{X: startX, Y: y},
			rl.Vector2{X: startX + totalWidth, Y: y},
			c.Color,
		)
	}

	/*
	   rl.DrawCircleV(c.Center, c.Radius, glob.ColorBg)
	   rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)
	   for _, d := range c.QubitList {
	       d.Draw(c.Center, c.Radius)
	   }
	*/
}

/*
                                 [O]
                                 [O]
                                 [O]
          ____   /`  ___          ||
      ___/O|__\_/_  /___\         ||
     /_____\__/___/|x===o|        ||
______(o)_______(o)||___||________||________
JRO    ________  ~.>\`    '    ________
 .  '  \__[]__/  '  ^^   .  .  \[]____/ '
       //\__/\\  '    ((())    /_/o\\\
  '.   ( o__o )       aa((((    <_  )/
        \ __ /  .  '  \-(((( .   \__/   .
 .     __\__/___      / /\\(     /   \
      /| \||/  |\    ( ( //  .  ||PD|| '  .
     / |  \/  @| \    \ //      ||  ||
    /  |_  __  |\ \   /[__]   ' ||  ||  .
.   \____|//_| |/ /  /_____\    ||__||
       |       |\/    | | | .   | / \|
       |_______|/     | | |     |_\\\|   .
        |     | .   ' |_|_|      |   |
 .'     |  |  |      <-<--/      |   |
        |  |  |  '  '  .      '  |   |  '  '
  .     |__|__|                  |___|
       (__) (__)      .  '      (____)  .

	   Stolen from Jonathon R. Oglesbee
*/

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
