package components

import (
	"fmt"
	"math"
	"math/cmplx"
	"qsim/config"
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
	ID                int32
	Label             string
	HookList          []*Hook
	Operation         [][]complex64
	InputCount        int32
	OutputCount       int32
	OutPutHook        []*Hook
	IsMeasurementGate bool
	MeasureResult     int32
	Measured          bool
	OutcomeProbs      [2]float64
	OutcomeLabels     [2]string
}

func NewGate(x, y, radius float32, color rl.Color, label string, operation [][]complex64, inputCount int32) *Gate {
	tmp := Gate{
		Circle:      *NewCircle(x, y, radius, color),
		Label:       label,
		Operation:   operation,
		InputCount:  inputCount,
		OutputCount: 1,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight(glob.GateWeight)
	snappedDist := utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)
	for i := 0; i < int(inputCount); i++ {
		offY := utils.SnapToGrid((float32(i)-float32(inputCount-1)/2)*config.SnapToGridInterval, config.SnapToGridInterval)
		newHook := NewHook(x-snappedDist, y+offY, glob.HookRadius, glob.HookColor)
		newHook.Label = "I" + strconv.Itoa(i)
		tmp.HookList = append(tmp.HookList, newHook)
	}
	tmp.OutPutHook = nil
	outputHook := NewOutputHook(x+snappedDist, y, glob.OutputHookRadius, glob.OutputHookColor)
	tmp.HookList = append(tmp.HookList, outputHook)
	tmp.OutPutHook = append(tmp.OutPutHook, outputHook)
	tmp.OutPutHook[0].Label = "O"
	tmp.OutPutHook[0].AllowQubitSystem = true
	return &tmp
}

func NewMeasurementGate(x, y, radius float32, color rl.Color, label string) *Gate {
	tmp := Gate{
		Circle:            *NewCircle(x, y, radius, color),
		Label:             label,
		InputCount:        1,
		IsMeasurementGate: true,
		OutputCount:       2,
	}
	tmp.ID = utils.GenerateID(&tmp)
	tmp.SetWeight(glob.GateWeight)
	snappedDist := utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)
	newHook := NewHook(x-snappedDist, y, glob.HookRadius, glob.HookColor)
	newHook.Label = "I"
	tmp.HookList = append(tmp.HookList, newHook)

	outputHook := NewOutputHook(x+snappedDist, y-glob.HookRadius, glob.OutputHookRadius, glob.OutputHookColor)
	tmp.HookList = append(tmp.HookList, outputHook)
	tmp.OutPutHook = nil
	tmp.OutPutHook = append(tmp.OutPutHook, outputHook)
	tmp.OutPutHook[0].Label = "O"
	tmp.OutPutHook[0].AllowQubitSystem = true

	outputHook2 := NewOutputHook(x+snappedDist, y+glob.HookRadius, glob.OutputHookRadius, glob.OutputHookColor)
	tmp.HookList = append(tmp.HookList, outputHook2)
	tmp.OutPutHook = append(tmp.OutPutHook, outputHook2)
	tmp.OutPutHook[1].Label = "O"
	tmp.OutPutHook[1].AllowQubitSystem = true
	return &tmp
}

func (c *Gate) onClick(worldMouse rl.Vector2, isCursorAvailable *bool) {
	switch {
	case utils.IsMouseState(glob.MouseStateFix):
		c.IsFixed = !c.IsFixed
	case utils.IsMouseState(glob.MouseStateDetach):
		c.DestroyOutPut()
		for _, d := range c.HookList {
			d.Disconnect()
		}
	case utils.IsMouseState(glob.MouseStateErase):
		c.Destroy()
	default:
		c.dragging = true
		*isCursorAvailable = false
		c.holdingCursor = true
		c.offset = rl.Vector2Subtract(c.Center, worldMouse)
		c.VirtualCenter = c.Center
	}
}

func (c *Gate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	// collision check using world coordinates
	if rl.CheckCollisionPointCircle(worldMouse, c.Center, c.Radius) {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && holdingCursor && (c.holdingCursor || *isCursorAvailable) {
			if utils.IsMouseState(glob.MouseStateNormal) && c.DoubleClicked(worldMouse) {
				// double-click: detach everything from the gate
				c.DestroyOutPut()
				for _, d := range c.HookList {
					d.Disconnect()
				}
				*isCursorAvailable = false
			} else {
				c.onClick(worldMouse, isCursorAvailable)
			}
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && c.holdingCursor {
		c.dragging = false
		c.holdingCursor = false
		*isCursorAvailable = true
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
		for _, d := range c.HookList {
			d.Center = d.Center.Add(delta)
		}
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

	if cnt == int(c.InputCount) {
		if c.IsMeasurementGate {
			if !c.Measured {
				c.MeasureOutput()
			}
		} else {
			if !c.CalculateOutPut() {
				c.DestroyOutPut()
			}
		}
	} else if len(c.OutPutHook) > 0 && c.OutPutHook[0].IsHooked && cnt != int(c.InputCount) {
		c.DestroyOutPut()
	}
}
func (c *Gate) MeasureOutput() {
	if len(c.HookList) == 0 || !c.HookList[0].IsHooked {
		return
	}
	target := utils.GetObjectFromID(c.HookList[0].TargetID)
	QD, ok := target.(*QubitDeterminator)
	if !ok {
		return
	}
	qp := QD.GetQubitParent()
	if qp == nil || qp.Origin == nil {
		return
	}
	QSM := qp.Origin

	var pos int32
	for i, d := range QSM.ModifierID {
		if QD.ModifierID == d {
			pos = int32(i)
			break
		}
	}

	var l complex64
	n := QSM.Size - 1
	for i, d := range QSM.Amptitude {
		if ((i >> (n - pos)) & 1) == 0 {
			l += complex(real(d)*real(d)+imag(d)*imag(d), 0)
		}
	}

	for hit := 0; hit <= 1; hit++ {
		var prob float64
		if hit == 0 {
			prob = cmplx.Abs(complex128(l))
		} else {
			prob = cmplx.Abs(complex128(complex(1, 0) - l))
		}
		c.OutcomeProbs[hit] = prob
		c.OutcomeLabels[hit] = fmt.Sprintf("|%d>", hit)

		if hit >= len(c.OutPutHook) {
			continue
		}

		if prob == 0 {
			amps := make([]complex64, 1<<QSM.Size)
			amps[hit<<(n-pos)] = 1
			result := qubits.NewQubitStateManagerFrom(amps, QSM.ModifierID)
			hook := c.OutPutHook[hit]
			if hook.IsHooked {
				tmp := utils.GetObjectFromID(hook.TargetID)
				if tmp != nil {
					if qs, ok2 := tmp.(*QubitsSystem); ok2 && qs.Origin != nil {
						qs.Origin.CopyFrom(result)
						continue
					}
				}
			}
			tmp := NewQubitsSystem(c.Center.X, c.Center.Y, glob.QubitSystemRadius, glob.QubitSystemColor)
			tmp.Assign(result)
			parent := c.GetParent()
			if parent != nil {
				tmp.SetParent(parent)
				parent.PushComponent(tmp)
			}
			hook.Connect(tmp)
			tmp.ZipDeterminatorsToHooks()
			continue
		}

		coeff := complex64(complex(1/prob, 0))
		coeff = complex64(cmplx.Sqrt(complex128(coeff)))

		result := qubits.NewQubitStateManagerFrom([]complex64{}, QSM.ModifierID)

		for i, d := range QSM.Amptitude {
			if ((i >> (n - pos)) & 1) == hit {
				result.Amptitude = append(result.Amptitude, d*coeff)
			} else {
				result.Amptitude = append(result.Amptitude, 0)
			}
		}

		hook := c.OutPutHook[hit]
		if hook.IsHooked {
			tmp := utils.GetObjectFromID(hook.TargetID)
			if tmp == nil {
				continue
			}
			if qs, ok2 := tmp.(*QubitsSystem); ok2 && qs.Origin != nil {
				qs.Origin.CopyFrom(result)
			}
		} else {
			tmp := NewQubitsSystem(c.Center.X, c.Center.Y, glob.QubitSystemRadius, glob.QubitSystemColor)
			tmp.Assign(result)
			parent := c.GetParent()
			if parent != nil {
				tmp.SetParent(parent)
				parent.PushComponent(tmp)
			}
			hook.Connect(tmp)
			tmp.ZipDeterminatorsToHooks()
		}
	}
}

func (c *Gate) CalculateOutPut() bool {
	QSM := []*qubits.QubitStateManager{}
	Idx := []int32{}

	defer func() {
		QSM = nil
	}()

	IsIN := func(id int32) bool {
		for _, d := range QSM {
			if d.ID == id {
				return true
			}
		}
		return false
	}

	for _, d := range c.HookList {
		if !d.IsOutput {
			QD, ok := utils.GetObjectFromID(d.TargetID).(*QubitDeterminator)
			if !ok {
				return false
			}
			qidObject := QD.GetQubitParent()
			if qidObject == nil {
				return false
			}
			qid := qidObject.Origin.ID
			if IsIN(qid) {
				Idx = append(Idx, QD.ModifierID)
			} else {
				tmp := *QD.GetQubitParent().Origin
				QSM = append(QSM, &tmp)
				Idx = append(Idx, QD.ModifierID)
		}
	}
	c.Measured = true
}
	result := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	for i := range QSM {
		result.Merge(QSM[i])
	}

	for i := range Idx {
		result.SwapColumn(int32(i), result.FindID(Idx[i]))
	}

	result.Multiply(c.Operation, c.InputCount)

	if len(c.OutPutHook) > 0 && c.OutPutHook[0].IsHooked {
		tmp := utils.GetObjectFromID(c.OutPutHook[0].TargetID)
		if qs, ok := tmp.(*QubitsSystem); ok && qs.Origin != nil {
			qs.Origin.CopyFrom(result)
			return true
		}
	}

	tmp := NewQubitsSystem(c.Center.X, c.Center.Y, glob.QubitSystemRadius, glob.QubitSystemColor)
	tmp.Assign(result)
	parent := c.GetParent()
	if parent != nil {
		tmp.SetParent(parent)
		parent.PushComponent(tmp)
	}

	c.OutPutHook[0].Connect(tmp)
	tmp.ZipDeterminatorsToHooks()
	return true
}

func (c *Gate) DestroyOutPut() {
	c.Measured = false
	for _, d := range c.OutPutHook {
		d.DisconnectAndKill()
	}
}

func (c *Gate) pullToHook(d *Hook) {
	val := utils.Dist(c.Center, d.Center) - utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)

	if math.Abs(float64(val)) <= float64(glob.GateToHookGraceDist) {
		return
	}

	val = float32(math.Max(float64(val), float64(-100)))
	val = float32(math.Min(float64(val), float64(100)))

	tmp := c.Center.Subtract(d.Center).Normalize().Scale(val * glob.GateToHookPullCoeff)

	if d.IsHooked {
		tmp2 := utils.GetObjectFromID(d.TargetID)
		if tmp2 == nil {
			d.Disconnect()
			return
		}
		if !config.PhysicsEnabled {
			return
		}
		t, ok := tmp2.(Component)
		if !ok {
			d.Disconnect()
			return
		}
		t.AddForce(tmp)
		c.AddForce(tmp.Scale(-1))
		return
	}

	if !config.PhysicsEnabled {
		return
	}

	d.AddForce(tmp)
	c.AddForce(tmp.Scale(-1))
}

func (c *Gate) Draw() {
	for _, d := range c.HookList {
		start := utils.RectEdgePoint(c.Center, d.Center, c.Radius, c.Radius)
		if d.IsHooked {
			tmp := utils.GetObjectFromID(d.TargetID)
			if tmp == nil {
				d.Disconnect()
				continue
			}
			t, ok := tmp.(Component)
		if !ok {
			d.Disconnect()
			continue
		}

			r := utils.Dist(start, d.Center) - t.GetCircle().Radius
			if r < 0 {
				r = 0
			}
			end := rl.Vector2Add(start, rl.Vector2Scale(
				rl.Vector2Normalize(rl.Vector2Subtract(d.Center, start)),
				r,
			))
			DrawWire(start, end, 4, c.Color)
		} else {
			DrawWire(start, d.Center, 4, c.Color)
		}
		d.Draw()
	}

	if c.IsMeasurementGate {
		for i, oh := range c.OutPutHook {
			mid := rl.Vector2Scale(rl.Vector2Add(c.Center, oh.Center), 0.5)
			label := fmt.Sprintf("%s, probability: %.2f", c.OutcomeLabels[i], c.OutcomeProbs[i])
			textWidth := rl.MeasureText(label, 14)
			// Backdrop so the label stays readable over wires and the grid
			bg := rl.NewRectangle(mid.X-float32(textWidth)/2-4, mid.Y-22, float32(textWidth)+8, 18)
			rl.DrawRectangleRec(bg, rl.Fade(rl.Black, 0.6))
			rl.DrawText(label, int32(mid.X)-textWidth/2, int32(mid.Y)-20, 14, rl.White)
		}
	}

	rect := rl.Rectangle{
		X:      c.Center.X - c.Radius,
		Y:      c.Center.Y - c.Radius,
		Width:  c.Radius * 2,
		Height: c.Radius * 2,
	}
	rl.DrawRectangleRounded(rect, 0.15, 6, glob.ColorBg)
	thickness := float32(4.0)
	if c.IsFixed {
		thickness = 5.0
	}
	rl.DrawRectangleRoundedLinesEx(rect, 0.15, 6, thickness, c.Color)

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

	if c.MeasureResult != 0 {
		rl.DrawText("Hit", int32(c.Center.X-30), int32(c.Center.Y-30), 14, c.Color)
	}
}

func (c *Gate) DrawGhost() {
	ghostColor := rl.Fade(c.Color, 0.3)
	ghostBg := rl.Fade(glob.ColorBg, 0.3)

	rect := rl.Rectangle{
		X:      c.Center.X - c.Radius,
		Y:      c.Center.Y - c.Radius,
		Width:  c.Radius * 2,
		Height: c.Radius * 2,
	}
	rl.DrawRectangleRec(rect, ghostBg)
	rl.DrawRectangleLinesEx(rect, 4, ghostColor)

	if c.Label != "" {
		fontSize := int32(c.Radius / 2)
		textWidth := rl.MeasureText(c.Label, fontSize)
		textX := int32(c.Center.X) - textWidth/2
		textY := int32(c.Center.Y) - fontSize/2
		rl.DrawText(c.Label, textX, textY, fontSize, ghostColor)
	}

	for _, d := range c.HookList {
		start := utils.RectEdgePoint(c.Center, d.Center, c.Radius, c.Radius)
		rl.DrawLineEx(start, d.Center, 4, ghostColor)
		d.DrawGhost()
	}
}

func (c *Gate) GetChildCircles() []*Circle {
	var children []*Circle
	for _, h := range c.HookList {
		if !h.IsHooked {
			children = append(children, h.GetCircle())
		}
	}
	return children
}

func (c *Gate) PostUpdate() {
	for i := range c.HookList {
		for j := i + 1; j < len(c.HookList); j++ {
			c.HookList[i].GetCircle().AntiGravity(c.HookList[j].GetCircle())
		}
	}
}

func (c *Gate) GetID() int32 {
	return c.ID
}

func (c *Gate) Destroy() {
	c.DestroyOutPut()
	for _, d := range c.HookList {
		d.DisconnectAndKill()
	}
	c.GetParent().DeleteChildWithID(c.ID)
}
