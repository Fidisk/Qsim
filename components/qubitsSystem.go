package components

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand/v2"
	"qsim/config"
	glob "qsim/globals"
	qub "qsim/qubits"
	"qsim/utils"
	"sort"
	"strconv"
	"strings"

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

	//Spagetti
	InfoHookID int32
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

func (c *QubitsSystem) GetID() int32 {
	return c.ID
}

func (c *QubitsSystem) CheckCollide(worldMouse rl.Vector2) bool {
	return rl.CheckCollisionPointRec(worldMouse, rl.Rectangle{X: c.startX, Y: c.startY, Width: float32(c.cols) * glob.QubitSystemCellWidth, Height: float32(c.rows) * glob.QubitSystemCellHeight})
}

func (c *QubitsSystem) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		c.IsFixed = !c.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		//c.Disconnect()
	case utils.IsMouseState(glob.MouseStateErase):
		c.Kill()
	default:
		c.dragging = true
		*isCursorAvailable = false
		c.holdingCursor = true
		c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		c.VirtualCenter = c.Center
	}
}

func (c *QubitsSystem) isDeterminatorVisible(d *QubitDeterminator) bool {
	for _, det := range c.QubitDeterminatorList {
		if det.HookID != 0 {
			return d.HookID != 0
		}
	}
	return true
}

func (c *QubitsSystem) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	for _, d := range c.QubitList {
		d.Update()
	}

	// collision check using world coordinates
	if c.CheckCollide(worldMouse) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			c.onClick(worldMouse, isCursorAvailable)
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		c.dragging = false
		c.holdingCursor = false
		*isCursorAvailable = true

		c.zipToHook()
		c.Center = c.VirtualCenter
		c.ClearForce()
	}
	if c.dragging {
		raw := rl.Vector2Add(worldMouse, c.offset)
		oldVC := c.VirtualCenter
		c.VirtualCenter = rl.Vector2Lerp(c.VirtualCenter, rl.Vector2{
			X: utils.SnapToGrid(raw.X, config.SnapToGridInterval),
			Y: utils.SnapToGrid(raw.Y, config.SnapToGridInterval),
		}, 0.25)
		delta := c.VirtualCenter.Subtract(oldVC)
		for _, d := range c.QubitDeterminatorList {
			d.Center = d.Center.Add(delta)
		}
		c.ClearForce()
	} else {
		c.DecayForce()
		c.ApplyForce()
	}

	for _, d := range c.QubitDeterminatorList {
		if !c.isDeterminatorVisible(d) {
			continue
		}
		c.pullToQubitSystem(d)

		d.Update(worldMouse, holdingCursor, isCursorAvailable)
	}
}

func (c *QubitsSystem) Draw() {
	if c.Origin == nil {
		return
	}
	for _, d := range c.QubitDeterminatorList {
		if c.isDeterminatorVisible(d) {
			rl.DrawLineEx(c.Center, d.Center, 4, c.Color)
		}
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
	borderRect := rl.NewRectangle(c.startX, c.startY, totalWidth, totalHeight)
	borderThickness := float32(4.0)
	if c.IsFixed {
		borderThickness = 5.0
	}
	rl.DrawRectangleLinesEx(borderRect, borderThickness, c.Color)

	// Vertical grid lines (between columns)
	for i := int32(1); i < c.cols; i++ {
		x := c.startX + float32(i*int32(n))
		rl.DrawLineEx(
			rl.Vector2{X: x, Y: c.startY},
			rl.Vector2{X: x, Y: c.startY + totalHeight},
			4, c.Color,
		)
	}

	// Horizontal grid lines (between c.rows)
	for j := int32(1); j < c.rows; j++ {
		y := c.startY + float32(j)*float32(m)
		rl.DrawLineEx(
			rl.Vector2{X: c.startX, Y: y},
			rl.Vector2{X: c.startX + totalWidth, Y: y},
			4, c.Color,
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
			r := real(c.Origin.Amptitude[index])
			img := imag(c.Origin.Amptitude[index])
			numberStr := fmt.Sprintf("%.2f%+.2fi", r, img)

			//Lmao why doesnt the AI just make a temp arr lol
			var parts []string
			temp := index
			for i := 0; temp > 0; i++ {
				if temp&1 == 1 {
					parts = append(parts, strconv.Itoa(int(c.Origin.ModifierID[i])))
				}
				temp >>= 1
			}

			// Sort numerically, not alphabetically
			sort.Slice(parts, func(i, j int) bool {
				a, _ := strconv.Atoi(parts[i])
				b, _ := strconv.Atoi(parts[j])
				return a < b
			})

			nameStr := strings.Join(parts, " + ")

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
		if c.isDeterminatorVisible(d) {
			d.Draw()
		}
	}
}

func (c *QubitsSystem) DrawGhost() {
	if c.Origin == nil {
		return
	}
	ghostColor := rl.Fade(c.Color, 0.3)

	for _, d := range c.QubitDeterminatorList {
		if c.isDeterminatorVisible(d) {
			rl.DrawLineEx(c.Center, d.Center, 4, ghostColor)
		}
	}

	exp := c.Origin.Size
	n := glob.QubitSystemCellWidth
	m := glob.QubitSystemCellHeight
	widthExp := (exp + 1) / 2
	heightExp := exp / 2
	cols := int32(1 << widthExp)
	rows := int32(1 << heightExp)
	totalWidth := float32(cols * int32(n))
	totalHeight := float32(rows * int32(m))
	startX := c.Center.X - totalWidth/2
	startY := c.Center.Y - totalHeight/2

	borderRect := rl.NewRectangle(startX, startY, totalWidth, totalHeight)
	rl.DrawRectangleLinesEx(borderRect, 4, ghostColor)

	for i := int32(1); i < cols; i++ {
		x := startX + float32(i*int32(n))
		rl.DrawLineEx(rl.Vector2{X: x, Y: startY}, rl.Vector2{X: x, Y: startY + totalHeight}, 4, ghostColor)
	}
	for j := int32(1); j < rows; j++ {
		y := startY + float32(j)*float32(m)
		rl.DrawLineEx(rl.Vector2{X: startX, Y: y}, rl.Vector2{X: startX + totalWidth, Y: y}, 4, ghostColor)
	}
}

func (c *QubitsSystem) GetChildCircles() []*Circle {
	var children []*Circle
	for _, d := range c.QubitDeterminatorList {
		if c.isDeterminatorVisible(d) {
			children = append(children, d.GetCircle())
		}
	}
	return children
}

func (c *QubitsSystem) Assign(p *qub.QubitStateManager) {
	//This do not defer the Origin
	c.Origin = p

	//The downfall of OOP
	c.QubitList = nil

	// Compute grid layout for right-side determinator placement
	exp := p.Size
	n := glob.QubitSystemCellWidth
	m := glob.QubitSystemCellHeight
	widthExp := (exp + 1) / 2
	heightExp := exp / 2
	cols := int32(1 << widthExp)
	rows := int32(1 << heightExp)
	totalWidth := float32(cols * int32(n))
	totalHeight := float32(rows * int32(m))
	startX := c.Center.X - totalWidth/2
	startY := c.Center.Y - totalHeight/2
	rightX := startX + totalWidth + n

	for i := range p.Size {
		y := startY + (float32(i)+0.5)*totalHeight/float32(p.Size)
		q := NewQubitDeterminator(rightX, y, c.Radius/float32(2), rl.Purple, p.ModifierID[i])

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
	tmp := c.GetParent()
	if tmp == nil {
		return
	}
	ele := tmp.GetElement()
	gotHooked := false
	for _, d := range ele {
		switch v := d.(type) {
		case *Hook:
			if utils.Dist(v.Center, c.Center) <= glob.HookDist && (!v.IsHooked || v.TargetID == c.ID) && !gotHooked && v.AllowQubitSystem {
				gotHooked = true
				v.Connect(c)
			}
		case *Gate:
			for _, d2 := range v.HookList {
				if utils.Dist(d2.Center, c.Center) <= glob.HookDist && (!d2.IsHooked || d2.TargetID == c.ID) && d2.AllowQubitSystem {
					gotHooked = true
					d2.Connect(c)
				}
			}
		case *InfoTable:
			tmp := v.Hook
			if utils.Dist(tmp.Center, c.Center) <= glob.HookDist && (!tmp.IsHooked || tmp.TargetID == c.ID) && tmp.AllowQubitSystem {
				gotHooked = true
				tmp.ConnectInfo(c)
			}
		case *SourceGate:
			tmp := v.OutHook
			if utils.Dist(tmp.Center, c.Center) <= glob.HookDist && (!tmp.IsHooked || tmp.TargetID == c.ID) && !gotHooked && tmp.AllowQubitSystem {
				gotHooked = true
				tmp.ConnectInfo(c)
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
		if !c.isDeterminatorVisible(c.QubitDeterminatorList[i]) {
			continue
		}
		for j := i + 1; j < len(c.QubitDeterminatorList); j++ {
			if !c.isDeterminatorVisible(c.QubitDeterminatorList[j]) {
				continue
			}
			c.QubitDeterminatorList[i].GetCircle().AntiGravity(c.QubitDeterminatorList[j].GetCircle())
		}
	}
}

func (c *QubitsSystem) pullToQubitSystem(d *QubitDeterminator) {
	if !config.PhysicsEnabled {
		return
	}
	if !c.isDeterminatorVisible(d) {
		return
	}
	if d.HookID != 0 {
		return
	}
	val := utils.Dist(c.Center, d.Center) - glob.GateToHookDist*float32(c.cols)

	if math.Abs(float64(val)) <= float64(glob.GateToHookGraceDist) {
		return
	}

	val = float32(math.Max(float64(val), float64(-100)))
	val = float32(math.Min(float64(val), float64(100)))

	tmp := c.Center.Subtract(d.Center).Normalize().Scale(val * glob.GateToHookPullCoeff)

	d.AddForce(tmp)
	c.AddForce(tmp.Scale(-1))
}

func (c *QubitsSystem) Kill() {
	for _, d := range c.QubitDeterminatorList {
		d.Kill()
	}

	parent := c.GetParent()
	if parent != nil {
		parent.DeleteChildWithID(c.ID)
	}
}
