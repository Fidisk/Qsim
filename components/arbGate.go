package components

import (
	"fmt"
	"qsim/config"
	glob "qsim/globals"
	"qsim/utils"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// NewArbGate creates an editable gate: right-clicking the body opens an
// inline editor where the user sets the qubit count and the full operation
// matrix. It spawns as the 1-qubit identity labeled "U".
func NewArbGate(x, y, radius float32, color rl.Color) *Gate {
	g := NewGate(x, y, radius, color, "U", [][]complex64{{1, 0}, {0, 1}}, 1)
	g.Editable = true
	g.Tooltip = "Arbitrary gate: right-click to set the qubit count and the operation matrix"
	return g
}

// ParseMatrixSpec parses the inline matrix spec:
//
//	n ; row ; row ; ...      (exactly 2^n rows)
//
// where each row holds 2^n whitespace-separated entries and each entry is
// "re" or "re,im". Example (1-qubit X):  "1; 0 1; 1 0".
func ParseMatrixSpec(spec string) (int32, [][]complex64, error) {
	parts := strings.Split(spec, ";")
	if len(parts) < 2 {
		return 0, nil, fmt.Errorf("expected 'n; row; row; ...'")
	}
	n64, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil || n64 < 1 || n64 > 4 {
		return 0, nil, fmt.Errorf("qubit count must be 1-4")
	}
	n := int32(n64)
	size := 1 << n
	rows := parts[1:]
	if len(rows) != size {
		return 0, nil, fmt.Errorf("expected %d rows, got %d", size, len(rows))
	}
	op := make([][]complex64, size)
	for i, row := range rows {
		fields := strings.Fields(row)
		if len(fields) != size {
			return 0, nil, fmt.Errorf("row %d: expected %d entries, got %d", i, size, len(fields))
		}
		op[i] = make([]complex64, size)
		for j, f := range fields {
			v, err := parseMatrixEntry(f)
			if err != nil {
				return 0, nil, fmt.Errorf("row %d entry %d: %v", i, j, err)
			}
			op[i][j] = v
		}
	}
	return n, op, nil
}

func parseMatrixEntry(s string) (complex64, error) {
	parts := strings.Split(s, ",")
	if len(parts) > 2 {
		return 0, fmt.Errorf("bad entry %q (want re or re,im)", s)
	}
	re, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("bad entry %q", s)
	}
	im := float64(0)
	if len(parts) == 2 {
		im, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, fmt.Errorf("bad entry %q", s)
		}
	}
	return complex64(complex(re, im)), nil
}

// formatMatrixSpec renders the gate's current matrix in the edit-buffer
// format, so right-clicking starts from the existing operation.
func formatMatrixSpec(n int32, op [][]complex64) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(int(n)))
	for _, row := range op {
		b.WriteString("; ")
		for j, v := range row {
			if j > 0 {
				b.WriteString(" ")
			}
			re := strconv.FormatFloat(float64(real(v)), 'g', -1, 32)
			if imag(v) == 0 {
				b.WriteString(re)
			} else {
				b.WriteString(re + "," + strconv.FormatFloat(float64(imag(v)), 'g', -1, 32))
			}
		}
	}
	return b.String()
}

// processEditing handles the inline matrix editor: type to edit, Enter
// parses and applies the new operation, Escape cancels. A parse error is
// shown and editing continues.
func (c *Gate) processEditing(isCursorAvailable *bool) {
	c.cursorBlink += rl.GetFrameTime()
	if c.cursorBlink > 0.5 {
		c.cursorShow = !c.cursorShow
		c.cursorBlink = 0
	}

	key := rl.GetCharPressed()
	for key > 0 {
		if strings.ContainsRune("0123456789.,;-+eE ", key) && len(c.editBuffer) < 4096 {
			c.editBuffer += string(key)
		}
		key = rl.GetCharPressed()
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(c.editBuffer) > 0 {
		c.editBuffer = c.editBuffer[:len(c.editBuffer)-1]
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		n, op, err := ParseMatrixSpec(c.editBuffer)
		if err != nil {
			c.editErr = err.Error()
			return
		}
		c.Reconfigure(op, n)
		c.editing = false
		c.holdingCursor = false
		*isCursorAvailable = true
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		c.editing = false
		c.holdingCursor = false
		*isCursorAvailable = true
	}
}

// Reconfigure swaps the gate's operation and qubit count, rebuilding the
// input hooks. The output hook (and its label/flags) is kept; the stale
// output system is destroyed since its size may no longer match.
func (c *Gate) Reconfigure(op [][]complex64, inputCount int32) {
	c.DestroyOutPut()

	var out *Hook
	if len(c.OutPutHook) > 0 {
		out = c.OutPutHook[0]
	}
	for _, h := range c.HookList {
		if h != out {
			h.Disconnect()
		}
	}

	c.Operation = op
	c.InputCount = inputCount
	c.Label = fmt.Sprintf("U%d", inputCount)
	c.HookList = nil

	snappedDist := utils.SnapToGrid(glob.GateToHookDist, config.SnapToGridInterval)
	for i := 0; i < int(inputCount); i++ {
		offY := utils.SnapToGrid((float32(i)-float32(inputCount-1)/2)*config.SnapToGridInterval, config.SnapToGridInterval)
		newHook := NewHook(c.Center.X-snappedDist, c.Center.Y+offY, glob.HookRadius, config.HookColor)
		newHook.Label = "I" + strconv.Itoa(i)
		newHook.Tooltip = "Gate input: plug a qubit determinator"
		c.HookList = append(c.HookList, newHook)
	}
	if out == nil {
		out = NewOutputHook(c.Center.X+snappedDist, c.Center.Y, glob.OutputHookRadius, config.OutputHookColor)
		out.Label = "O"
		out.AllowQubitSystem = true
		out.Tooltip = "Gate output: the transformed qubit system"
	}
	c.HookList = append(c.HookList, out)
	c.OutPutHook = []*Hook{out}
}
