package components

import (
	"math"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/utils"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Gate struct {
	//It's a square, extending a circle
	//Wow, you have taken your OOP class well
	//No go out there and poison those LLM
	Circle
	ID         int32
	Label      string
	HookList   []*Hook
	Operation  [][]complex64
	InputCount int32
	OutPutHook *Hook
}

func NewGate(x, y, radius float32, color rl.Color, label string, operation [][]complex64, inputCount int32) *Gate {
	tmp := Gate{
		Circle:     *NewCircle(x, y, radius, color),
		Label:      label,
		Operation:  operation,
		InputCount: inputCount,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight(glob.GateWeight)
	for i := 0; i < int(inputCount); i++ {
		newHook := NewHook(x, y, glob.HookRadius, glob.HookColor)
		newHook.Label = "I" + strconv.Itoa(i)
		tmp.HookList = append(tmp.HookList, newHook)
	}
	outputHook := NewOutputHook(x, y, glob.OutputHookRadius, glob.OutputHookColor)
	tmp.HookList = append(tmp.HookList, outputHook)
	tmp.OutPutHook = outputHook
	tmp.OutPutHook.Label = "O"
	return &tmp
}

func (c *Gate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
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
	}
	if c.dragging {
		c.Center = rl.Vector2Add(worldMouse, c.offset)
		c.ClearForce()
	} else {
		c.DecayForce()
		c.ApplyForce()
	}

	cnt := 0

	for _, d := range c.HookList {
		c.pullToHook(d)

		d.Update(worldMouse, holdingCursor, isCursorAvailable)

		if d.IsHooked && !d.IsOutput {
			cnt++
		}
	}

	if cnt == int(c.InputCount) && !c.OutPutHook.IsHooked {
		c.CalculateOutPut()
	} else if c.OutPutHook.IsHooked && cnt != int(c.InputCount) {
		c.DestroyOutPut()
	}
}

func (c *Gate) CalculateOutPut() {
	QSM := []*qubits.QubitStateManager{}
	Cnt := []int32{}
	Idx := [][]int32{}
	defer func() {
		QSM = nil
	}()
	defer func() {
		Cnt = nil
	}()
	IsIN := func(id int32) bool {
		for _, d := range QSM {
			if d.ID == id {
				return true
			}
		}
		return false
	}
	index := int32(0)
	for _, d := range c.HookList {
		if !d.IsOutput {
			QD := utils.GetObjectFromID(d.TargetID).(*QubitDeterminator)
			qid := QD.GetQubitParent().Origin.ID
			if IsIN(qid) {
				for i, d2 := range QSM {
					if d2.ID == qid {
						var pos int32 = 0
						for i := range d2.ModifierID {
							if d2.ModifierID[i] == QD.ModifierID {
								pos = int32(i)
								break
							}
						}
						d2.SwapColumn(Cnt[i], pos)
						Cnt[i]++
						Idx[i] = append(Idx[i], index)
						index++
					}
				}
			} else {
				tmp := *QD.GetQubitParent().Origin
				QSM = append(QSM, &tmp)
				Cnt = append(Cnt, 0)
				Idx = append(Idx, []int32{index})
				index++
			}
		}
	}
	result := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	var sl int32 = 0
	for i := range QSM {
		result.MergeWithPrefix(QSM[i], Cnt[i], sl)
		sl += Cnt[i]
	}

	order := []int32{}

	for _, d := range Idx {
		for _, d2 := range d {
			order = append(order, d2)
		}
	}

	n := len(order)

	for i := 0; i < n; i++ {
		if order[i] != int32(i) {
			pos := int32(i)
			for {
				if order[pos] == pos {
					break
				}
				result.SwapColumn(pos, order[pos])
				pos = order[order[pos]]
			}
		}
	}

	result.Multiply(c.Operation, c.InputCount)

	//fmt.Println(result)
	tmp := NewQubitsSystem(c.Center.X, c.Center.X, glob.QubitSystemRadius, glob.QubitSystemColor)
	tmp.Assign(result)
	tmp.SetParent(c.GetParent())
	tmp.GetParent().PushComponent(tmp)

	c.OutPutHook.Connect(tmp)

	//fmt.Println(c.OutPutHook.IsHooked)
}

func (c *Gate) DestroyOutPut() {
	c.OutPutHook.Disconnect()
}

func (c *Gate) pullToHook(d *Hook) {
	val := utils.Dist(c.Center, d.Center) - glob.GateToHookDist

	if math.Abs(float64(val)) <= float64(glob.GateToHookGraceDist) {
		return
	}

	val = float32(math.Max(float64(val), float64(-100)))
	val = float32(math.Min(float64(val), float64(100)))

	tmp := c.Center.Subtract(d.Center).Normalize().Scale(val * glob.GateToHookPullCoeff)

	if d.IsHooked {
		tmp2 := utils.GetObjectFromID(d.TargetID)
		t := tmp2.(Component)
		t.AddForce(tmp)
		c.AddForce(tmp.Scale(-1))
		return
	}

	d.AddForce(tmp)
	c.AddForce(tmp.Scale(-1))
}

func (c *Gate) Draw() {
	for _, d := range c.HookList {
		if d.IsHooked {
			tmp := utils.GetObjectFromID(d.TargetID)
			t := tmp.(Component)

			r := utils.Dist(c.Center, d.Center) - t.GetCircle().Radius
			start := c.Center
			end := rl.Vector2Add(start, rl.Vector2Scale(
				rl.Vector2Normalize(rl.Vector2Subtract(d.Center, start)),
				r,
			))
			rl.DrawLineV(start, end, c.Color)
		} else {
			rl.DrawLineV(c.Center, d.Center, c.Color)
		}
		d.Draw()
	}

	rect := rl.Rectangle{
		X:      c.Center.X - c.Radius,
		Y:      c.Center.Y - c.Radius,
		Width:  c.Radius * 2,
		Height: c.Radius * 2,
	}
	rl.DrawRectangleRec(rect, glob.ColorBg)
	rl.DrawRectangleLinesEx(rect, 1.0, c.Color)

	if c.Label != "" {
		// Choose a font size (adjust as needed)
		fontSize := int32(c.Radius / 2) // e.g., match size to radius

		// Measure text with raylib's default font
		textWidth := rl.MeasureText(c.Label, fontSize)

		// Centre the text inside the square
		textX := int32(c.Center.X) - textWidth/2
		textY := int32(c.Center.Y) - fontSize/2

		rl.DrawText(c.Label, textX, textY, fontSize, c.Color)
	}
}

func (c *Gate) PostUpdate() {
	for i := range c.HookList {
		for j := i + 1; j < len(c.HookList); j++ {
			c.HookList[i].GetCircle().AntiGravity(c.HookList[j].GetCircle())
		}
	}
}
