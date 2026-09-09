package components

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand/v2"
	"qsim/config"
	glob "qsim/globals"
	qub "qsim/qubits"
	"qsim/qubits/attributes"
	"qsim/utils"
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
	// Probability is the branch weight of this system: 1 for fresh inputs,
	// the product of its inputs' probabilities after a gate, and the input
	// probability times the outcome probability after a measurement. Shown
	// on hover.
	Probability float64
	rows        int32
	cols        int32
	startX      float32
	startY      float32
	hovered     bool
	pressPos    rl.Vector2

	//Spagetti
	InfoHookID int32

	// IsLogical marks systems that should not expose individual qubit
	// determinators. Kept for save-file compatibility; current producers
	// (copies, collapse remainders) emit normal systems.
	IsLogical bool
}

func NewQubitsSystem(x, y, radius float32, color rl.Color) *QubitsSystem {
	tmp := QubitsSystem{
		Circle:      *NewCircle(x, y, radius, color),
		QubitList:   nil,
		Origin:      nil,
		HookID:      0,
		Probability: 1,
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

// OutlinePoint returns the point on the state grid's rectangle edge toward
// `from`: connection wires stop there instead of running over the drawn grid
// (the grid is the visible body, far larger than the 30px circle). It finds
// the first edge the segment from->center crosses.
func (c *QubitsSystem) OutlinePoint(from rl.Vector2) rl.Vector2 {
	if c.Origin == nil {
		return from
	}
	exp := c.Origin.Size
	w := float32(int32(1)<<((exp+1)/2)) * glob.QubitSystemCellWidth
	h := float32(int32(1)<<(exp/2)) * glob.QubitSystemCellHeight
	cx, cy := c.Center.X, c.Center.Y
	halfW, halfH := w/2, h/2
	dx := cx - from.X
	dy := cy - from.Y
	best := float32(1)
	if dx != 0 {
		for _, ex := range []float32{cx - halfW, cx + halfW} {
			t := (ex - from.X) / dx
			if t >= 0 && t <= 1 {
				py := from.Y + t*dy
				if py >= cy-halfH && py <= cy+halfH {
					best = min(best, t)
				}
			}
		}
	}
	if dy != 0 {
		for _, ey := range []float32{cy - halfH, cy + halfH} {
			t := (ey - from.Y) / dy
			if t >= 0 && t <= 1 {
				px := from.X + t*dx
				if px >= cx-halfW && px <= cx+halfW {
					best = min(best, t)
				}
			}
		}
	}
	return rl.Vector2{X: from.X + best*dx, Y: from.Y + best*dy}
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
		c.pressPos = worldMouse
	}
}

func (c *QubitsSystem) isDeterminatorVisible(d *QubitDeterminator) bool {
	// Hooked determinators stay visible. Free ones disappear only while the
	// gate holding a sibling determinator is calculating (all of its inputs
	// hooked) — until then they stay grabbable so more qubits of the system
	// can be fed into the same gate.
	if d.HookID != 0 {
		return true
	}
	parent := c.GetParent()
	for _, det := range c.QubitDeterminatorList {
		if det.HookID == 0 {
			continue
		}
		if gateCalculating(parent, det.HookID) {
			return false
		}
	}
	return true
}

// DeterminatorWireVisible reports whether the wire to a determinator is
// currently being drawn. Kept exported so render passes (e.g. edge markers)
// can agree on which determinators are on screen.
func (c *QubitsSystem) DeterminatorWireVisible(d *QubitDeterminator) bool {
	return c.isDeterminatorVisible(d)
}

// findHookOwner returns the component whose GetHooks list contains the hook
// with the given ID (e.g. the gate a determinator is plugged into), or nil
// for standalone hooks.
func findHookOwner(parent PlaceholderWindow, hookID int32) Component {
	h, ok := utils.GetObjectFromID(hookID).(*Hook)
	if !ok || parent == nil {
		return nil
	}
	for _, comp := range parent.GetElement() {
		owner, ok := comp.(hookOwner)
		if !ok {
			continue
		}
		for _, gh := range owner.GetHooks() {
			if gh == h {
				return comp
			}
		}
	}
	return nil
}

// gateCalculating reports whether the gate owning the given hook is currently
// calculating, i.e. all of its input hooks are connected.
func gateCalculating(parent PlaceholderWindow, hookID int32) bool {
	h, ok := utils.GetObjectFromID(hookID).(*Hook)
	if !ok {
		return false
	}
	var owner hookOwner
	if parent != nil {
		for _, comp := range parent.GetElement() {
			if o, ok2 := comp.(hookOwner); ok2 {
				for _, gh := range o.GetHooks() {
					if gh == h {
						owner = o
						break
					}
				}
				if owner != nil {
					break
				}
			}
		}
	}
	if owner == nil {
		for _, obj := range utils.ObjectList {
			if o, ok2 := obj.(hookOwner); ok2 {
				for _, gh := range o.GetHooks() {
					if gh == h {
						owner = o
						break
					}
				}
				if owner != nil {
					break
				}
			}
		}
	}
	if owner == nil {
		return false
	}
	var hookList []*Hook
	var inputCount int32
	switch g := owner.(type) {
	case *Gate:
		hookList = g.HookList
		inputCount = g.InputCount
	case *CollapseGate:
		hookList = g.HookList
		inputCount = g.InputCount
	case *M4Gate:
		hookList = g.HookList
		inputCount = g.InputCount
	case *ControlledUGate:
		hookList = g.QubitHooks
		inputCount = g.InputCount
	default:
		return false
	}
	cnt := 0
	for _, h := range hookList {
		if h.IsHooked && !h.IsOutput {
			cnt++
		}
	}
	return cnt == int(inputCount)
}

// qubitCycle is the click-cycle of single-qubit states, in order:
// |0> -> |1> -> |+> -> |-> -> |i> -> |-i>. Freshly spawned qubits start at
// |0> (see SpawnObject), so the first click goes |0> -> |1>.
var qubitCycle = []struct {
	name string
	amps []complex64
}{
	{"|0>", []complex64{1, 0}},
	{"|1>", []complex64{0, 1}},
	{"|+>", []complex64{h, h}},
	{"|->", []complex64{h, -h}},
	{"|i>", []complex64{h, ih}},
	{"|-i>", []complex64{h, -ih}},
}

var h = complex64(complex(float32(1/math.Sqrt(2)), 0))  // 1/√2
var ih = complex64(complex(0, float32(1/math.Sqrt(2)))) // i/√2

// qubitStateName returns the cycle name of the given single-qubit state, or
// "" when it is not one of the six cycle states.
func qubitStateName(amps []complex64) string {
	for _, s := range qubitCycle {
		ok := len(amps) == len(s.amps)
		for i := range s.amps {
			if ok && complexAbs(amps[i]-s.amps[i]) > 1e-4 {
				ok = false
			}
		}
		if ok {
			return s.name
		}
	}
	return ""
}

// nextQubitState returns the amplitude vector that follows the given one in
// the cycle. Unknown states cycle from |0>.
func nextQubitState(amps []complex64) []complex64 {
	for i, s := range qubitCycle {
		match := len(amps) == len(s.amps)
		for j := range s.amps {
			if match && complexAbs(amps[j]-s.amps[j]) > 1e-4 {
				match = false
			}
		}
		if match {
			return qubitCycle[(i+1)%len(qubitCycle)].amps
		}
	}
	return qubitCycle[1].amps // unknown -> |1> (next after |0>)
}

// cycleState advances a standalone single-qubit system to the next state in
// the cycle. Fresh qubits stay cyclable even when their determinator is
// plugged into a gate; systems emitted by a source or a gate (HookID != 0)
// are not. The amplitudes are updated in place so the modifier ID, the
// determinators and any hooked wiring survive the change.
func (c *QubitsSystem) cycleState() {
	if c.Origin == nil || c.Origin.Size != 1 || c.HookID != 0 {
		return // only normal qubits, not source/gate outputs
	}
	next := nextQubitState(c.Origin.Amptitude)
	copy(c.Origin.Amptitude, next)
}

func complexAbs(z complex64) float64 {
	return math.Hypot(float64(real(z)), float64(imag(z)))
}

// enforceSingleGate implements the "one gate at a time" rule: a qubit system
// feeds exactly one gate. Once any of its determinators sits in a gate that
// is filled (all inputs hooked), every other determinator is hidden, and any
// determinator still hooked to a DIFFERENT gate is disconnected so it cannot
// be used twice. Determinators of the same system plugged into the same gate
// (multi-input gates) are left alone.
func (c *QubitsSystem) enforceSingleGate() {
	parent := c.GetParent()
	if parent == nil {
		return
	}
	for _, det := range c.QubitDeterminatorList {
		if det.HookID == 0 || !gateCalculating(parent, det.HookID) {
			continue
		}
		owner := findHookOwner(parent, det.HookID)
		for _, other := range c.QubitDeterminatorList {
			if other == det || other.HookID == 0 {
				continue
			}
			if findHookOwner(parent, other.HookID) == owner {
				continue // same gate: multi-input gate sharing this system
			}
			if h, ok := utils.GetObjectFromID(other.HookID).(*Hook); ok {
				h.Disconnect()
			}
		}
		return
	}
}

func (c *QubitsSystem) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	for _, d := range c.QubitList {
		d.Update()
	}

	c.enforceSingleGate()

	// Hover state drives the ghost highlight in Draw.
	c.hovered = !c.dragging && c.CheckCollide(worldMouse)

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

		// A plain click (no drag) in normal mode on a standalone single
		// qubit cycles its state: |0> -> |1> -> |+> -> |-> -> |i> -> |-i>.
		if utils.IsMouseState(glob.MouseStateNormal) && utils.Dist(c.pressPos, worldMouse) < 5 {
			c.cycleState()
		}

		// Commit the drop position BEFORE zipping: during the drag only
		// VirtualCenter moved, so zipToHook must look for hooks around the
		// drop point, not the position the drag started from.
		c.Center = c.VirtualCenter

		// Dragging the system carries its determinators along, so let them
		// auto-connect first, from the positions they were dropped at. Then
		// the system zips; its snap shifts only the still-free determinators,
		// leaving freshly hooked ones anchored to their hooks.
		c.ZipDeterminatorsToHooks()
		c.zipToHook()
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
}

// UpdateDeterminators updates the qubit determinators. It runs as a
// priority pass before the other components so a determinator stays
// draggable even when it overlaps another component (e.g. a gate).
func (c *QubitsSystem) UpdateDeterminators(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	for i, d := range c.QubitDeterminatorList {
		if !c.isDeterminatorVisible(d) {
			continue
		}
		c.pullToQubitSystem(d, i)

		d.Update(worldMouse, holdingCursor, isCursorAvailable)
	}
}

// ZipDeterminatorsToHooks gives every free, visible determinator a chance to
// auto-connect when it sits on top of a hook. Used when a system spawns or is
// dropped. Visibility is snapshotted up front: connecting one determinator
// hides the others, and each visible one on a hook should still connect.
func (c *QubitsSystem) ZipDeterminatorsToHooks() {
	visible := make([]bool, len(c.QubitDeterminatorList))
	for i, d := range c.QubitDeterminatorList {
		visible[i] = d.HookID == 0 && c.isDeterminatorVisible(d)
	}
	for i, d := range c.QubitDeterminatorList {
		if visible[i] {
			d.zipToHook()
		}
	}
}

// modifierLabel returns the display label for a qubit modifier: the renamed
// display name when one is set, otherwise the raw numeric ID.
func modifierLabel(id int32) string {
	if id >= 0 && int(id) < attributes.AttributesManager.Len() {
		if n, ok := attributes.AttributesManager.Get(id).(*attributes.Name); ok {
			return n.Val
		}
	}
	return strconv.Itoa(int(id))
}

// trimFloat formats f with two decimals and strips trailing zeros:
// "1.00" -> "1", "0.50" -> "0.5", "0.00" -> "0".
func trimFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "-0" {
		s = "0"
	}
	return s
}

// formatCellAmplitude renders one state-grid amplitude compactly, dropping
// the part that is zero: 1+0i -> "1", 0+1i -> "i", 0 -> "0",
// 0.7+0.3i -> "0.7 + 0.3i", 1-i -> "1 - i". The sign is spaced so it reads
// as a separate element (and gets its own color in drawAmplitude).
func formatCellAmplitude(v complex64) string {
	re := trimFloat(float64(real(v)))
	im := trimFloat(float64(imag(v)))
	if im == "0" {
		return re
	}
	if re == "0" {
		switch im {
		case "1":
			return "i"
		case "-1":
			return "-i"
		}
		return im + "i"
	}
	sign := "+"
	if imag(v) < 0 {
		sign = "-"
		im = trimFloat(-float64(imag(v)))
	}
	if im == "1" {
		im = ""
	}
	return re + " " + sign + " " + im + "i"
}

// drawAmplitude renders an amplitude string left-to-right from (x, y),
// coloring every '+' sign green (e.g. 0.7 + 0.3i, 1 - i, -0.5i). The rest,
// including the minus sign, keeps the given base color.
func drawAmplitude(x, y int32, s string, fontSize int32, base rl.Color) {
	type seg struct {
		text string
		col  rl.Color
	}
	var segs []seg
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			segs = append(segs, seg{buf.String(), base})
			buf.Reset()
		}
	}
	for _, r := range s {
		switch r {
		case '+':
			flush()
			segs = append(segs, seg{"+", rl.Green})
		default:
			buf.WriteRune(r)
		}
	}
	flush()
	sx := x
	for _, g := range segs {
		rl.DrawText(g.text, sx, y, fontSize, g.col)
		sx += rl.MeasureText(g.text, fontSize)
	}
}

func (c *QubitsSystem) Draw() {
	if c.Origin == nil {
		return
	}
	for _, d := range c.QubitDeterminatorList {
		if c.isDeterminatorVisible(d) {
			DrawWire(c.Center, d.Center, 4, c.Color)
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
		config.ColorBg,
	)

	// Probability heat-map: brighter cell = larger |amplitude|^2
	for row := int32(0); row < c.rows; row++ {
		for col := int32(0); col < c.cols; col++ {
			index := row*c.cols + col
			v := c.Origin.Amptitude[index]
			p := real(v)*real(v) + imag(v)*imag(v)
			if p > 1 {
				p = 1
			}
			if p > 0.001 {
				cellRect := rl.NewRectangle(
					c.startX+float32(col)*n, c.startY+float32(row)*m, n, m,
				)
				rl.DrawRectangleRec(cellRect, rl.Fade(rl.SkyBlue, p*0.28))
			}
		}
	}

	// Outer border
	borderRect := rl.NewRectangle(c.startX, c.startY, totalWidth, totalHeight)
	borderThickness := float32(4.0)
	if c.IsFixed {
		borderThickness = 5.0
	}
	rl.DrawRectangleRoundedLinesEx(borderRect, 0.03, 6, borderThickness, c.Color)

	// Hover ghost: faint bright outline + wash so the system reads as grabbable
	if c.hovered {
		rl.DrawRectangleRec(borderRect, rl.Fade(rl.SkyBlue, 0.08))
		rl.DrawRectangleRoundedLinesEx(borderRect, 0.03, 6, borderThickness+2, rl.Fade(rl.SkyBlue, 0.45))
	}

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
			numberStr := formatCellAmplitude(complex(r, img))

			// Fade out states with negligible probability
			p := r*r + img*img
			textA := float32(0.35)
			if p > 0 {
				textA = 0.35 + 0.65*min(p*2, 1)
			}
			textCol := rl.Fade(c.Color, textA)

			// Bit representation of the basis state (e.g. 000, 001, 110),
			// shown where the modifier names used to be.
			nameStr := fmt.Sprintf("%0*b", exp, index)

			// Draw number slightly above center. Font stays at 20 and only
			// scales down when the entry is too wide for the cell.
			numFontSize := int32(20)
			numWidth := rl.MeasureText(numberStr, numFontSize)
			for numWidth > int32(n)-10 && numFontSize > 8 {
				numFontSize--
				numWidth = rl.MeasureText(numberStr, numFontSize)
			}
			drawAmplitude(int32(cellCenterX)-numWidth/2,
				int32(cellCenterY)-numFontSize-2, // 2px gap
				numberStr, numFontSize, textCol)

			// Draw name slightly below center
			nameFontSize := int32(14)
			nameWidth := rl.MeasureText(nameStr, nameFontSize)
			rl.DrawText(nameStr,
				int32(cellCenterX)-nameWidth/2,
				int32(cellCenterY)+2,
				nameFontSize,
				textCol,
			)
		}
	}

	// Hover: small label above the system naming its qubit determinators
	// (e.g. "Q0 + Q1 + Q2"); standalone single qubits also show their cycle
	// state (|0>, |1>, |+>, |->, |i>, |-i>) and every system shows its
	// branch probability (the product of its inputs' probabilities through
	// gates, times the outcome probability after a measurement).
	if c.hovered && c.Origin != nil {
		names := make([]string, len(c.Origin.ModifierID))
		for i, id := range c.Origin.ModifierID {
			names[i] = modifierLabel(id)
		}
		label := strings.Join(names, " + ")
		if c.Origin.Size == 1 && c.HookID == 0 {
			if s := qubitStateName(c.Origin.Amptitude); s != "" {
				label += "  " + s
			}
		}
		label += fmt.Sprintf("  p=%.3f", c.Probability)
		fontSize := int32(16)
		textWidth := rl.MeasureText(label, fontSize)
		boxW := float32(textWidth) + 12
		boxH := float32(fontSize) + 8
		box := rl.NewRectangle(c.Center.X-boxW/2, c.startY-boxH-6, boxW, boxH)
		rl.DrawRectangleRec(box, config.ColorBg)
		rl.DrawRectangleLinesEx(box, 2, c.Color)
		rl.DrawText(label, int32(box.X+6), int32(box.Y+4), fontSize, c.Color)
	}

	// (Commented‑out old drawing code remains unchanged)
	/*
	   rl.DrawCircleV(c.Center, c.Radius, config.ColorBg)
	   rl.DrawCircleLinesV(c.Center, c.Radius, c.Color)
	   for _, d := range c.QubitList {
	       d.Draw(c.Center, c.Radius)
	   }
	*/
}

// DrawDeterminators draws the qubit determinators. It is called after all
// other components so determinators stay visible (and grabbable) on top of
// whatever they overlap (e.g. a gate).
func (c *QubitsSystem) DrawDeterminators() {
	if c.Origin == nil {
		return
	}
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
			rl.DrawCircleLines(int32(d.Center.X), int32(d.Center.Y), d.Radius, ghostColor)
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
	widthExp := (exp + 1) / 2
	cols := int32(1 << widthExp)
	totalWidth := float32(cols * int32(n))
	detX := determinatorColumnX(c.Center.X, totalWidth)

	if !c.IsLogical {
		for i := range p.Size {
			y := determinatorSlotY(c.Center.Y, int(i), int(p.Size))
			q := NewQubitDeterminator(detX, y, c.Radius/float32(2), rl.Purple, p.ModifierID[i])

			q.QubitSystemID = c.ID

			c.QubitDeterminatorList = append(c.QubitDeterminatorList, q)
		}
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
			if utils.Dist(v.Center, c.Center) <= glob.HookDist && (!v.IsHooked || v.TargetID == c.ID) && !gotHooked && v.AllowQubitSystem && !v.Hidden && !v.AllowLogicalBit {
				gotHooked = true
				v.Connect(c)
			}
		case *CompareGate:
			// Compare inputs only read the state: attach via InfoHookID so the
			// system's source connection (HookID) survives, like InfoTable and
			// CopyGate. Pick the closest available input hook.
			var closest *Hook
			closestDist := float32(math.MaxFloat32)
			for _, h := range []*Hook{v.InA, v.InB} {
				if h.Hidden || !h.AllowQubitSystem || h.AllowLogicalBit {
					continue
				}
				if h.IsHooked && h.TargetID != c.ID {
					continue
				}
				d := utils.Dist(h.Center, c.Center)
				if d <= glob.HookDist && d < closestDist {
					closest = h
					closestDist = d
				}
			}
			if closest != nil {
				gotHooked = true
				closest.ConnectInfo(c)
			}
		case hookOwner:
			for _, d2 := range v.GetHooks() {
				if d2.Hidden || !d2.AllowQubitSystem || d2.AllowLogicalBit {
					continue
				}
				if utils.Dist(d2.Center, c.Center) <= glob.HookDist && (!d2.IsHooked || d2.TargetID == c.ID) && d2.AllowQubitSystem {
					gotHooked = true
					d2.Connect(c)
				}
			}
		case *InfoTable:
			tmp := v.Hook
			if utils.Dist(tmp.Center, c.Center) <= glob.HookDist && (!tmp.IsHooked || tmp.TargetID == c.ID) && tmp.AllowQubitSystem && !tmp.AllowLogicalBit {
				gotHooked = true
				tmp.ConnectInfo(c)
			}
		case *CopyGate:
			tmp := v.InHook
			if utils.Dist(tmp.Center, c.Center) <= glob.HookDist && (!tmp.IsHooked || tmp.TargetID == c.ID) && tmp.AllowQubitSystem && !tmp.AllowLogicalBit {
				gotHooked = true
				tmp.ConnectInfo(c)
			}
		case *SourceGate:
			tmp := v.OutHook
			if utils.Dist(tmp.Center, c.Center) <= glob.HookDist && (!tmp.IsHooked || tmp.TargetID == c.ID) && !gotHooked && tmp.AllowQubitSystem && !tmp.AllowLogicalBit {
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
		// Dragging away from an output hook (gate/source) only repositions
		// the system — the link must survive, otherwise the gate sees a free
		// output hook and respawns a duplicate system. Standalone hooks
		// release normally.
		keep := false
		if h, ok := utils.GetObjectFromID(c.HookID).(*Hook); ok && h.IsOutput {
			keep = true
		}
		if !keep {
			c.removeFromHook()
		}
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

// determinatorColumnX returns the grid-snapped X of the determinator column:
// one cell to the right of the state grid.
func determinatorColumnX(centerX, totalWidth float32) float32 {
	return utils.SnapToGrid(centerX+totalWidth/2+glob.QubitSystemCellWidth, config.SnapToGridInterval)
}

// determinatorSlotY returns the grid-snapped Y of the i-th determinator slot.
// Slots are spaced one snap interval apart, centered on the system, so a
// determinator always rests on a grid point.
func determinatorSlotY(centerY float32, i, size int) float32 {
	stride := config.SnapToGridInterval
	return utils.SnapToGrid(centerY+(float32(i)-float32(size-1)/2)*stride, stride)
}

// determinatorHome returns the laid-out position for the i-th determinator:
// a column one cell to the right of the state grid, matching Assign.
func (c *QubitsSystem) determinatorHome(i int) rl.Vector2 {
	exp := int32(1)
	if c.Origin != nil {
		exp = c.Origin.Size
	}
	widthExp := (exp + 1) / 2
	cols := int32(1 << widthExp)
	totalWidth := float32(cols * int32(glob.QubitSystemCellWidth))
	size := len(c.QubitDeterminatorList)
	if size < 1 {
		size = 1
	}
	return rl.Vector2{
		X: determinatorColumnX(c.Center.X, totalWidth),
		Y: determinatorSlotY(c.Center.Y, i, size),
	}
}

// pullToQubitSystem springs a free determinator back toward its laid-out
// home slot next to the grid, so determinators keep their column instead of
// drifting off at random angles.
func (c *QubitsSystem) pullToQubitSystem(d *QubitDeterminator, i int) {
	if !config.PhysicsEnabled {
		return
	}
	if !c.isDeterminatorVisible(d) {
		return
	}
	if d.HookID != 0 {
		return
	}
	disp := c.determinatorHome(i).Subtract(d.Center)
	dist := disp.Length()

	if dist <= glob.GateToHookGraceDist {
		return
	}

	val := float32(math.Min(float64(dist), 100))

	tmp := disp.Normalize().Scale(val * glob.GateToHookPullCoeff)

	d.AddForce(tmp)
	c.AddForce(tmp.Scale(-1))
}

// CopyFromState replaces this system's quantum state with the supplied one and
// then invalidates any downstream gates so they recalculate with the new state.
func (c *QubitsSystem) CopyFromState(state *qub.QubitStateManager) {
	if c.Origin == nil {
		c.Origin = qub.NewQubitStateManagerFrom([]complex64{}, []int32{})
	}
	c.Origin.CopyFrom(state)
	c.InvalidateDownstreamGates()
}

// InvalidateDownstreamGates marks every cached Gate/CollapseGate fed by this
// system as unmeasured so it recalculates on the next frame, and recursively
// propagates through the qubit-system outputs of those gates.
func (c *QubitsSystem) InvalidateDownstreamGates() {
	parent := c.GetParent()
	if parent == nil {
		return
	}
	for _, d := range c.QubitDeterminatorList {
		if d.HookID == 0 {
			continue
		}
		owner := findHookOwner(parent, d.HookID)
		if owner == nil {
			continue
		}
		var outputs []*Hook
		switch g := owner.(type) {
		case *Gate:
			if g.Measured {
				g.Measured = false
				outputs = g.OutPutHook
			}
		case *CollapseGate:
			if g.Measured {
				g.Measured = false
				outputs = g.OutPutHook
			}
		}
		for _, oh := range outputs {
			if oh.IsHooked {
				if qs, ok := utils.GetObjectFromID(oh.TargetID).(*QubitsSystem); ok {
					qs.InvalidateDownstreamGates()
				}
			}
		}
	}
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
