package components

import (
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/utils"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// M4Gate is a measurement gate like M2 (CollapseGate) that outputs the measured
// outcome as a logical bit plus the conditioned remainder state. Unlike M2 it
// does not eat the input qubit: the plugged determinator stays hooked and
// visible, and the system it belongs to is committed to this gate (its other
// determinators hide / disconnect — a qubit system feeds one gate at a time).
// The gate then toggles its output value every N frames. Right-click the gate
// body to set the flip interval.
type M4Gate struct {
	CollapseGate
	FrameCount    int
	InputConsumed bool // input snapshot taken (and current)
	StoredInput   *qubits.QubitStateManager
	StoredPos     int32
	StoredSysID   int32 // parent system the snapshot was taken from
	SwapInterval  int

	// inline editor state for the flip interval
	editing     bool
	editBuffer  string
	editErr     string
	cursorBlink float32
	cursorShow  bool
}

const defaultM4SwapInterval = 20

func NewM4Gate(x, y, radius float32, color rl.Color) *M4Gate {
	tmp := &M4Gate{}
	tmp.Circle = *NewCircle(x, y, radius, color)
	tmp.Label = "M4"
	tmp.InputCount = 1
	tmp.Tooltip = "Oscillating measurement: behaves like M2, locks the input qubit (one gate per qubit system), then toggles the measured output. Right-click to set the flip interval."
	tmp.ID = utils.GenerateID(tmp)
	tmp.SetWeight(glob.GateWeight)
	tmp.ConsumeInput = true
	tmp.SkipRandom = true
	tmp.ForceMode = 1 // deterministic start at |0>
	tmp.SwapInterval = defaultM4SwapInterval

	snappedDist := utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)

	in := NewHook(x-snappedDist, y, glob.HookRadius, config.HookColor)
	in.Label = "I"
	in.Tooltip = "Measure input: plug a qubit determinator"
	tmp.HookList = append(tmp.HookList, in)

	outCollapsed := NewLogicalOutputHook(x+snappedDist, y-glob.HookRadius)
	outCollapsed.Label = "C"
	outCollapsed.Tooltip = "Measured logical bit (toggles every N frames)"
	tmp.HookList = append(tmp.HookList, outCollapsed)
	tmp.OutPutHook = append(tmp.OutPutHook, outCollapsed)

	outRest := NewOutputHook(x+snappedDist, y+glob.HookRadius, glob.OutputHookRadius, config.OutputHookColor)
	outRest.Label = "R"
	outRest.AllowQubitSystem = true
	outRest.Tooltip = "Remaining state output (multi-qubit inputs only)"
	outRest.Hidden = true
	tmp.HookList = append(tmp.HookList, outRest)
	tmp.OutPutHook = append(tmp.OutPutHook, outRest)

	return tmp
}

func (m *M4Gate) Update(worldMouse rl.Vector2, holdingCursor bool, isCursorAvailable *bool) {
	if m.editing {
		m.processEditing(isCursorAvailable)
		return
	}

	// Right-click opens the flip-time editor (only in normal mouse mode).
	if utils.IsMouseState(glob.MouseStateNormal) && rl.CheckCollisionPointCircle(worldMouse, m.Center, m.Radius) {
		if !rl.IsMouseButtonDown(rl.MouseButtonLeft) && m.Tooltip != "" {
			glob.TooltipText = m.Tooltip
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonRight) && holdingCursor && (m.holdingCursor || *isCursorAvailable) {
			m.editing = true
			m.editBuffer = strconv.Itoa(m.SwapInterval)
			m.editErr = ""
			m.holdingCursor = true
			*isCursorAvailable = false
			return
		}
	}

	m.CollapseGate.Update(worldMouse, holdingCursor, isCursorAvailable)

	if !m.Measured {
		m.FrameCount = 0
		return
	}

	// The input qubit is NOT consumed: it stays hooked (and visible). When
	// the plugged qubit changes (different parent system), re-measure
	// against the new system instead of toggling a stale snapshot.
	if !m.InputConsumed || m.inputChanged() {
		m.storeInput()
		m.InputConsumed = true
		m.Measured = false
		m.FrameCount = 0
		return
	}

	m.FrameCount++
	if m.FrameCount >= m.SwapInterval {
		m.FrameCount = 0
		m.swapOutput()
	}
}

// processEditing handles the inline flip-interval editor: type an integer,
// Enter applies, Escape cancels.
func (m *M4Gate) processEditing(isCursorAvailable *bool) {
	m.cursorBlink += rl.GetFrameTime()
	if m.cursorBlink > 0.5 {
		m.cursorShow = !m.cursorShow
		m.cursorBlink = 0
	}

	key := rl.GetCharPressed()
	for key > 0 {
		if key >= '0' && key <= '9' && len(m.editBuffer) < 8 {
			m.editBuffer += string(rune(key))
		}
		key = rl.GetCharPressed()
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(m.editBuffer) > 0 {
		m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		val, err := strconv.Atoi(m.editBuffer)
		if err != nil || val < 1 {
			m.editErr = "interval must be >= 1"
			return
		}
		m.SwapInterval = val
		m.editing = false
		m.holdingCursor = false
		*isCursorAvailable = true
		m.FrameCount = 0
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		m.editing = false
		m.holdingCursor = false
		*isCursorAvailable = true
	}
}

// Draw renders the gate and, when active, the inline flip-interval editor.
func (m *M4Gate) Draw() {
	m.CollapseGate.Draw()

	if m.editing {
		fontSize := int32(14)
		hint := "flip interval (frames): type integer, Enter=apply, Esc=cancel"
		text := m.editBuffer
		w := rl.MeasureText(text, fontSize) + 12
		hw := rl.MeasureText(hint, 10) + 12
		if hw > w {
			w = hw
		}
		if w < 120 {
			w = 120
		}
		height := float32(46)
		if m.editErr != "" {
			height = 64
		}
		box := rl.NewRectangle(m.Center.X-float32(w)/2, m.Center.Y-m.Radius-height-8, float32(w), height)
		rl.DrawRectangleRounded(box, 0.15, 4, rl.NewColor(30, 30, 30, 255))
		rl.DrawRectangleRoundedLinesEx(box, 0.15, 4, 1.5, rl.SkyBlue)
		rl.DrawText(hint, int32(box.X+6), int32(box.Y+4), 10, rl.Gray)
		rl.DrawText(text, int32(box.X+6), int32(box.Y+18), fontSize, rl.White)
		if m.cursorShow {
			tw := rl.MeasureText(text, fontSize)
			rl.DrawText("|", int32(box.X+6)+tw, int32(box.Y+18), fontSize, rl.SkyBlue)
		}
		if m.editErr != "" {
			rl.DrawText(m.editErr, int32(box.X+6), int32(box.Y+38), 12, rl.Red)
		}
	}
}

// storeInput copies the input state and measured-qubit position so the gate can
// keep toggling its output even while the input stays hooked. The parent system
// ID is remembered so an input swap can be detected and re-measured.
func (m *M4Gate) storeInput() {
	_, _, qp, pos, ok := m.getMeasuredInput()
	if !ok || qp.Origin == nil {
		return
	}
	m.StoredPos = pos
	m.StoredSysID = qp.ID
	m.StoredInput = qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	m.StoredInput.CopyFrom(qp.Origin)
}

// inputChanged reports whether the determinator currently plugged into the
// gate belongs to a different parent system than the stored snapshot.
func (m *M4Gate) inputChanged() bool {
	_, _, qp, _, ok := m.getMeasuredInput()
	return !ok || qp == nil || qp.Origin == nil || qp.ID != m.StoredSysID
}

// swapOutput toggles the realized measurement result and re-emits the outputs
// so that the logical bit (and the conditioned remainder) flip to the opposite
// outcome. Downstream propagation is handled automatically by QubitsSystem.
func (m *M4Gate) swapOutput() {
	if m.StoredInput == nil {
		return
	}
	m.Result = 1 - m.Result
	m.emitOutcome(m.StoredInput, m.StoredPos, m.Result)
}

// DrawGhost draws the ghost preview; keep the M4 label but skip the editor.
func (m *M4Gate) DrawGhost() {
	m.CollapseGate.DrawGhost()
}
