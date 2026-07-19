package animation

import (
	"fmt"

	"qsim/components"
	"qsim/config"
	"qsim/qubits"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ComputeDemo is a precomputed snapshot of a gate's matrix multiplication
//
//	result[(j<<shift)+l] += A[(i<<shift)+l] * val[j][i]
//
// used to render the step-by-step computation visualization that replaces the
// old "Compute:" box. The demo is drawn in world space above the gate.
type ComputeDemo struct {
	Size   int32 // total qubits in the merged input system
	N      int32 // qubits the gate acts on (gate matrix is 2^N x 2^N)
	Shift  int32 // Size - N
	Label  string
	Matrix [][]complex64

	// Merged input system before the column reorder.
	MergedAmps []complex64
	PreMods    []int32 // modifier IDs in display order (top -> bottom)

	// After the reorder the gate qubits occupy the top N display positions.
	PostMods  []int32
	DigitDest []int32 // DigitDest[oldPos] = new display position of a qubit
	Perm      []int32 // Perm[oldRow] = row the amplitude content moves to
	InAmps    []complex64
	OutAmps   []complex64

	// Per-source columns (pre-merge), for the merge phase.
	SrcAmps  [][]complex64
	SrcMods  [][]int32
	SrcSizes []int32

	Steps     []computeStep
	IterStart []float64
	IterTotal float64

	TMerge    float64
	TReorder  float64
	TGate     float64
	TCollapse float64
	Total     float64
}

type computeStep struct {
	I, L    int32
	Contrib []complex64 // per output state j: InAmps[(I<<Shift)+L] * Matrix[j][I]
}

// BuildComputeDemo snapshots the gate's inputs and precomputes the whole
// computation. It mirrors Gate.CalculateOutPut (merge -> swap -> multiply)
// without mutating any live state. Returns nil when the gate has no
// computable matrix input (e.g. measurement gates).
func BuildComputeDemo(g *components.Gate) *ComputeDemo {
	if g.IsMeasurementGate || g.InputCount <= 0 || len(g.Operation) == 0 {
		return nil
	}
	n := g.InputCount
	dim := int32(1) << uint(n)
	if int32(len(g.Operation)) != dim {
		return nil
	}
	for _, row := range g.Operation {
		if int32(len(row)) != dim {
			return nil
		}
	}

	var qsms []*qubits.QubitStateManager
	var idx []int32
	seen := map[int32]bool{}

	for _, h := range g.HookList {
		if h.IsOutput {
			continue
		}
		qd, ok := utils.GetObjectFromID(h.TargetID).(*components.QubitDeterminator)
		if !ok || qd == nil {
			return nil
		}
		qp := qd.GetQubitParent()
		if qp == nil || qp.Origin == nil {
			return nil
		}
		if seen[qp.Origin.ID] {
			idx = append(idx, qd.ModifierID)
			continue
		}
		seen[qp.Origin.ID] = true
		cp := *qp.Origin
		cp.Amptitude = append([]complex64(nil), cp.Amptitude...)
		cp.ModifierID = append([]int32(nil), cp.ModifierID...)
		qsms = append(qsms, &cp)
		idx = append(idx, qd.ModifierID)
	}
	if len(qsms) == 0 || int32(len(idx)) != n {
		return nil
	}

	d := &ComputeDemo{N: n, Label: g.Label, Matrix: g.Operation}

	for _, q := range qsms {
		d.SrcAmps = append(d.SrcAmps, append([]complex64(nil), q.Amptitude...))
		d.SrcMods = append(d.SrcMods, append([]int32(nil), q.ModifierID...))
		d.SrcSizes = append(d.SrcSizes, q.Size)
	}

	merged := qubits.NewQubitStateManagerFrom([]complex64{}, []int32{})
	for _, q := range qsms {
		merged.Merge(q)
	}
	d.Size = merged.Size
	d.Shift = d.Size - n
	if d.Shift < 0 || d.Size <= 0 || d.Size > 10 {
		return nil
	}
	d.MergedAmps = append([]complex64(nil), merged.Amptitude...)
	d.PreMods = append([]int32(nil), merged.ModifierID...)

	// Reorder: bring the gate input qubits to the top display positions.
	// The swap targets are chosen exactly like Gate.CalculateOutPut does
	// (FindID on the live QSM), so the demo reproduces the same amplitudes
	// the app computes - including any ModifierID quirks for Size >= 3.
	// post/dest/perm track the physical display permutation: SwapColumn(i, m)
	// swaps state bits (Size-1-i) and (Size-1-m), i.e. display positions
	// i and m.
	post := append([]int32(nil), d.PreMods...)
	dest := make([]int32, d.Size)
	at := make([]int32, d.Size) // at[pos] = original position of the qubit now at pos
	for p := range post {
		dest[p] = int32(p)
		at[p] = int32(p)
	}
	perm := make([]int32, 1<<uint(d.Size))
	for s := range perm {
		perm[s] = int32(s)
	}
	swapPos := func(a, b int32) {
		ba := d.Size - 1 - a
		bb := d.Size - 1 - b
		for s := range perm {
			cur := perm[s]
			bita := (cur >> uint(ba)) & 1
			bitb := (cur >> uint(bb)) & 1
			if bita != bitb {
				perm[s] = cur ^ (1 << uint(ba)) ^ (1 << uint(bb))
			}
		}
		post[a], post[b] = post[b], post[a]
		oa, ob := at[a], at[b]
		at[a], at[b] = ob, oa
		dest[oa], dest[ob] = b, a
	}
	for i := int32(0); i < n; i++ {
		m := merged.FindID(idx[i])
		if m < 0 {
			return nil
		}
		if m != i {
			swapPos(i, m)
		}
		merged.SwapColumn(i, m)
	}
	d.PostMods = post
	d.DigitDest = dest
	d.Perm = perm
	d.InAmps = append([]complex64(nil), merged.Amptitude...)

	out := qubits.NewQubitStateManagerFrom(
		append([]complex64(nil), merged.Amptitude...),
		append([]int32(nil), merged.ModifierID...),
	)
	out.Multiply(g.Operation, n)
	d.OutAmps = out.Amptitude

	// Iteration steps: i (gate input state) outer, l (remaining qubits) inner,
	// each step distributing over all output states j. Speed ramps up.
	dimI := int32(1) << uint(n)
	dimL := int32(1) << uint(d.Shift)
	count := dimI * dimL
	if count > config.ComputeIterCap {
		count = config.ComputeIterCap
	}
	t := 0.0
	dur := float64(config.ComputeIterBaseDur)
	for i := int32(0); i < dimI && int32(len(d.Steps)) < count; i++ {
		for l := int32(0); l < dimL && int32(len(d.Steps)) < count; l++ {
			st := computeStep{I: i, L: l, Contrib: make([]complex64, dimI)}
			a := d.InAmps[(i<<uint(d.Shift))+l]
			for j := int32(0); j < dimI; j++ {
				st.Contrib[j] = a * g.Operation[j][i]
			}
			d.Steps = append(d.Steps, st)
			d.IterStart = append(d.IterStart, t)
			t += dur
			dur *= float64(config.ComputeIterDecay)
			if dur < float64(config.ComputeIterFloorDur) {
				dur = float64(config.ComputeIterFloorDur)
			}
		}
	}
	d.IterTotal = t

	d.TMerge = config.ComputeMergeDur
	if len(d.SrcAmps) <= 1 {
		d.TMerge = 0.6
	}
	d.TReorder = config.ComputeReorderDur
	identity := true
	for p := range dest {
		if dest[p] != int32(p) {
			identity = false
			break
		}
	}
	if identity {
		d.TReorder = 0.5
	}
	d.TGate = config.ComputeGateAppearDur
	d.TCollapse = config.ComputeCollapseDur
	d.Total = d.TMerge + d.TReorder + d.TGate + d.IterTotal + d.TCollapse
	return d
}

// ---------------------------------------------------------------- layout ---

const (
	dRowH    = float32(26)
	dCellW   = float32(78)
	dSumColW = float32(124)
	dPlusW   = float32(26)
	dHeaderH = float32(30)
	dTagW    = float32(46)
	dTagH    = float32(24)
	dAmpW    = float32(108)
	dDigitW  = float32(13)
	dGap     = float32(44)
)

type demoLayout struct {
	top    float32
	tagX   float32
	ampX   float32
	ketX   float32
	matX   float32
	sumX   float32
	width  float32
	height float32
	colW   float32
}

func makeLayout(d *ComputeDemo, ga *GateAnim) demoLayout {
	dim := int32(1) << uint(d.N)
	ketW := 26 + float32(d.Size)*dDigitW
	colW := dAmpW + ketW
	matW := float32(dim) * dCellW
	sumW := float32(dim)*dSumColW + float32(dim-1)*dPlusW
	width := dTagW + 8 + colW + dGap + matW + dGap + sumW

	colRows := float32(visCount(int32(1) << uint(d.Size)))
	height := dHeaderH + colRows*dRowH
	if mh := dHeaderH + float32(dim)*dRowH; mh > height {
		height = mh
	}

	// The demo block is centered on the gate so it appears on top of it.
	ctr := ga.Gate.GetCenter()
	l := demoLayout{width: width, height: height, colW: colW}
	l.top = ctr.Y - height/2
	x := ctr.X - width/2
	l.tagX = x
	l.ampX = x + dTagW + 8
	l.ketX = l.ampX + dAmpW + 6
	l.matX = l.ketX + ketW + dGap
	l.sumX = l.matX + matW + dGap
	return l
}

func visCount(total int32) int32 {
	if total < config.ComputeMaxRows {
		return total
	}
	return config.ComputeMaxRows
}

// visRows returns the row indices to display, with -1 marking a "..." slot.
func visRows(total int32) []int32 {
	max := int32(config.ComputeMaxRows)
	if max < 3 {
		max = 3
	}
	if total <= max {
		out := make([]int32, total)
		for i := range out {
			out[i] = int32(i)
		}
		return out
	}
	out := make([]int32, 0, max)
	for i := int32(0); i < max-2; i++ {
		out = append(out, i)
	}
	out = append(out, -1, total-1)
	return out
}

// slotOf maps an actual row to its display slot; hidden rows map to the
// "..." slot.
func slotOf(rows []int32, r int32) int32 {
	for i, v := range rows {
		if v == r {
			return int32(i)
		}
	}
	return int32(len(rows) - 2)
}

// --------------------------------------------------------------- helpers ---

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func ease(t float32) float32 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float32) float32 { return a + (b-a)*t }

func fadeA(c rl.Color, a float32) rl.Color { return rl.Fade(c, clamp01(a)) }

func lerpColor(a, b rl.Color, t float32) rl.Color {
	f := func(x, y uint8) uint8 {
		v := float32(x) + (float32(y)-float32(x))*clamp01(t)
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(v)
	}
	return rl.NewColor(f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), f(a.A, b.A))
}

// qColor returns the configured color of a qubit by its modifier ID.
func qColor(mod int32) rl.Color {
	n := int32(len(config.QubitColors))
	if n == 0 {
		return rl.White
	}
	m := mod % n
	if m < 0 {
		m += n
	}
	return config.QubitColors[m]
}

func fmtC(v complex64) string {
	return fmt.Sprintf("%.2f%+.2fi", real(v), imag(v))
}

func drawAmp(x, y float32, v complex64, alpha float32) {
	s := fmtC(v)
	w := rl.MeasureText(s, 13)
	rl.DrawText(s, int32(x+dAmpW-6)-w, int32(y+6), 13, fadeA(rl.White, alpha))
}

// drawKet draws |digits> with each digit colored by its qubit's color.
func drawKet(x, y float32, s int32, size int32, mods []int32, alpha float32) {
	rl.DrawText("|", int32(x), int32(y+4), 16, fadeA(rl.White, alpha))
	for p := int32(0); p < size; p++ {
		bit := (s >> uint(size-1-p)) & 1
		ch := "0"
		if bit == 1 {
			ch = "1"
		}
		col := rl.White
		if int(p) < len(mods) {
			col = qColor(mods[p])
		}
		rl.DrawText(ch, int32(x+10+float32(p)*dDigitW), int32(y+4), 16, fadeA(col, alpha))
	}
	rl.DrawText(">", int32(x+10+float32(size)*dDigitW), int32(y+4), 16, fadeA(rl.White, alpha))
}

// drawKetMix is drawKet with per-position colors crossfading pre -> post.
func drawKetMix(x, y float32, s int32, size int32, pre, post []int32, e float32) {
	rl.DrawText("|", int32(x), int32(y+4), 16, rl.White)
	for p := int32(0); p < size; p++ {
		bit := (s >> uint(size-1-p)) & 1
		ch := "0"
		if bit == 1 {
			ch = "1"
		}
		col := rl.White
		if int(p) < len(pre) && int(p) < len(post) {
			col = lerpColor(qColor(pre[p]), qColor(post[p]), e)
		}
		rl.DrawText(ch, int32(x+10+float32(p)*dDigitW), int32(y+4), 16, col)
	}
	rl.DrawText(">", int32(x+10+float32(size)*dDigitW), int32(y+4), 16, rl.White)
}

// drawMiniKet draws a small centered ket used for matrix/sum-column headers.
func drawMiniKet(cx, y float32, val int32, size int32, mods []int32, alpha float32) {
	w := 12 + float32(size)*9
	x := cx - w/2
	rl.DrawText("|", int32(x), int32(y), 11, fadeA(rl.White, alpha))
	for p := int32(0); p < size; p++ {
		bit := (val >> uint(size-1-p)) & 1
		ch := "0"
		if bit == 1 {
			ch = "1"
		}
		col := rl.White
		if int(p) < len(mods) {
			col = qColor(mods[p])
		}
		rl.DrawText(ch, int32(x+7+float32(p)*9), int32(y), 11, fadeA(col, alpha))
	}
	rl.DrawText(">", int32(x+7+float32(size)*9), int32(y), 11, fadeA(rl.White, alpha))
}

func drawTag(x, y float32, m int32, alpha float32, glow bool) {
	col := qColor(m)
	rect := rl.NewRectangle(x, y, dTagW-6, dTagH-4)
	rl.DrawRectangleRec(rect, fadeA(col, 0.25*alpha))
	thick := float32(1.5)
	if glow {
		thick = 3
	}
	rl.DrawRectangleLinesEx(rect, thick, fadeA(col, alpha))
	if glow {
		rl.DrawRectangleLinesEx(rect, 1, fadeA(rl.Gold, 0.9*alpha))
	}
	name := fmt.Sprintf("Q%d", m)
	rl.DrawText(name, int32(x+6), int32(y+4), 12, fadeA(rl.White, alpha))
}

// drawStateColumn draws amplitude + ket rows of a state at (ampX, ketX, top).
// hiRow >= 0 highlights that row.
func drawStateColumn(d *ComputeDemo, ampX, ketX, top float32, amps []complex64, mods []int32, alpha float32, hiRow int32) {
	rows := visRows(int32(1) << uint(d.Size))
	ketW := 26 + float32(d.Size)*dDigitW
	for slot, r := range rows {
		y := top + dHeaderH + float32(slot)*dRowH
		if r < 0 {
			rl.DrawText("...", int32(ampX+dAmpW-20), int32(y+6), 13, fadeA(rl.White, alpha*0.7))
			continue
		}
		if r == hiRow {
			rect := rl.NewRectangle(ampX-6, y, dAmpW+ketW+6, dRowH)
			rl.DrawRectangleRec(rect, rl.Fade(rl.White, 0.10*clamp01(alpha)))
			rl.DrawRectangleLinesEx(rect, 1, rl.Fade(rl.Gold, 0.6*clamp01(alpha)))
		}
		drawAmp(ampX, y, amps[r], alpha)
		drawKet(ketX, y, r, d.Size, mods, alpha)
	}
}

func drawMatrix(d *ComputeDemo, l demoLayout, alpha float32, hiCol int32) {
	dim := int32(1) << uint(d.N)
	for i := int32(0); i < dim; i++ {
		drawMiniKet(l.matX+float32(i)*dCellW+dCellW/2, l.top+6, i, d.N, d.PostMods, alpha)
	}
	for j := int32(0); j < dim; j++ {
		drawMiniKet(l.matX-10-(12+float32(d.N)*9)/2, l.top+dHeaderH+float32(j)*dRowH+7, j, d.N, d.PostMods, alpha)
		for i := int32(0); i < dim; i++ {
			x := l.matX + float32(i)*dCellW
			y := l.top + dHeaderH + float32(j)*dRowH
			rect := rl.NewRectangle(x, y, dCellW, dRowH)
			bg := rl.Fade(rl.White, 0.05*clamp01(alpha))
			if i == hiCol {
				bg = rl.Fade(rl.Gold, 0.18*clamp01(alpha))
			}
			rl.DrawRectangleRec(rect, bg)
			rl.DrawRectangleLinesEx(rect, 1, rl.Fade(rl.LightGray, 0.5*clamp01(alpha)))
			s := fmtC(d.Matrix[j][i])
			w := rl.MeasureText(s, 10)
			rl.DrawText(s, int32(x+dCellW/2)-w/2, int32(y+8), 10, fadeA(rl.White, alpha))
		}
	}
}

// drawSumColumnAt draws one sum column (output state j) at (x, colTop).
// grid/written hold the accumulated values; pend (if non-nil) is the step
// currently arriving, crossfading into row pend.L with factor pop.
// headerAlpha controls the mini-ket header independently (collapsed early).
func drawSumColumnAt(d *ComputeDemo, x, colTop float32, j int32, grid []complex64, written []bool, alpha, headerAlpha float32, pend *computeStep, pop float32) {
	dimL := int32(1) << uint(d.Shift)
	rows := visRows(dimL)
	drawMiniKet(x+dSumColW/2, colTop+6, j, d.N, d.PostMods, headerAlpha)
	for slot, r := range rows {
		y := colTop + dHeaderH + float32(slot)*dRowH
		if r < 0 {
			rl.DrawText("...", int32(x+dSumColW/2-8), int32(y+6), 13, fadeA(rl.White, alpha*0.7))
			continue
		}
		idx := j*dimL + r
		a := alpha
		if !written[idx] {
			a = alpha * 0.25
		}
		if pend != nil && r == pend.L && pop > 0 {
			rect := rl.NewRectangle(x+2, y, dSumColW-4, dRowH)
			rl.DrawRectangleRec(rect, rl.Fade(rl.Gold, 0.15*clamp01(pop)))
			if written[idx] {
				s := fmtC(grid[idx])
				w := rl.MeasureText(s, 11)
				rl.DrawText(s, int32(x+dSumColW/2)-w/2, int32(y+7), 11, fadeA(rl.White, alpha*(1-pop)))
			}
			s := fmtC(grid[idx] + pend.Contrib[j])
			w := rl.MeasureText(s, 11)
			rl.DrawText(s, int32(x+dSumColW/2)-w/2, int32(y+7), 11, fadeA(rl.White, alpha*pop))
			continue
		}
		s := fmtC(grid[idx])
		w := rl.MeasureText(s, 11)
		rl.DrawText(s, int32(x+dSumColW/2)-w/2, int32(y+7), 11, fadeA(rl.White, a))
	}
}

// drawRays draws the light rays for the current iteration step: from the
// active state row through each row of the gate matrix column I into the
// sum columns. u in [0,1) is the intra-step progress.
func drawRays(d *ComputeDemo, l demoLayout, st computeStep, u float32) {
	dimJ := int32(1) << uint(d.N)
	dimL := int32(1) << uint(d.Shift)
	rowsIn := visRows(int32(1) << uint(d.Size))
	rowsL := visRows(dimL)

	srcIdx := (st.I << uint(d.Shift)) + st.L
	srcSlot := slotOf(rowsIn, srcIdx)
	y1 := l.top + dHeaderH + float32(srcSlot)*dRowH + dRowH/2
	p1 := rl.NewVector2(l.ketX+16+float32(d.Size)*dDigitW, y1)

	for j := int32(0); j < dimJ; j++ {
		v := st.Contrib[j]
		if real(v) == 0 && imag(v) == 0 {
			continue
		}
		cellX := l.matX + float32(st.I)*dCellW
		cellY := l.top + dHeaderH + float32(j)*dRowH
		p2 := rl.NewVector2(cellX, cellY+dRowH/2)
		mid := rl.NewVector2(cellX+dCellW, cellY+dRowH/2)
		dstSlot := slotOf(rowsL, st.L)
		y3 := l.top + dHeaderH + float32(dstSlot)*dRowH + dRowH/2
		p3 := rl.NewVector2(l.sumX+float32(j)*(dSumColW+dPlusW), y3)

		rayCol := rl.Fade(rl.Gold, 0.35)
		rl.DrawLineEx(p1, p2, 2.5, rayCol)
		rl.DrawLineEx(mid, p3, 2.5, rayCol)

		var pos rl.Vector2
		if u < 0.5 {
			pos = rl.Vector2Lerp(p1, p2, u*2)
		} else {
			pos = rl.Vector2Lerp(mid, p3, (u-0.5)*2)
		}
		rl.DrawCircleV(pos, 5, rl.Fade(rl.White, 0.9))
		rl.DrawCircleV(pos, 9, rl.Fade(rl.Gold, 0.35))
	}
}

func drawPhaseLabel(l demoLayout, s string) {
	rl.DrawText(s, int32(l.tagX), int32(l.top-24), 14, rl.Fade(rl.Gold, 0.9))
}

// fullGrid accumulates every (i, l) contribution, used once iterating ends.
func fullGrid(d *ComputeDemo) ([]complex64, []bool) {
	dimI := int32(1) << uint(d.N)
	dimL := int32(1) << uint(d.Shift)
	grid := make([]complex64, dimI*dimL)
	written := make([]bool, dimI*dimL)
	for i := int32(0); i < dimI; i++ {
		for l := int32(0); l < dimL; l++ {
			a := d.InAmps[(i<<uint(d.Shift))+l]
			for j := int32(0); j < dimI; j++ {
				v := a * d.Matrix[j][i]
				grid[j*dimL+l] += v
				if real(v) != 0 || imag(v) != 0 {
					written[j*dimL+l] = true
				}
			}
		}
	}
	return grid, written
}

// ---------------------------------------------------------------- phases ---

// DrawCompute renders the computation demo t seconds after it started.
func DrawCompute(ga *GateAnim, t float64) {
	d := ga.Demo
	if d == nil {
		return
	}
	l := makeLayout(d, ga)
	ft := float32(t)
	m1 := float32(d.TMerge)
	m2 := m1 + float32(d.TReorder)
	m3 := m2 + float32(d.TGate)
	m4 := float32(d.Total) - float32(d.TCollapse)

	// Backdrop: the demo sits on top of the gate, so give it a panel that
	// covers the circuit underneath (phase label included). During the
	// collapse phase the sum columns stack downward, so grow the panel to
	// keep them covered.
	pad := float32(14)
	bgH := l.height
	if ft >= m4 {
		stackH := dHeaderH + float32(int32(1)<<uint(d.N))*float32(visCount(int32(1)<<uint(d.Shift)))*dRowH
		if stackH > bgH {
			bgH = stackH
		}
	}
	bg := rl.NewRectangle(l.tagX-pad, l.top-30-pad, l.width+2*pad, bgH+30+2*pad)
	rl.DrawRectangleRec(bg, rl.Fade(rl.Black, 0.88))
	rl.DrawRectangleLinesEx(bg, 2, rl.Fade(rl.Gold, 0.6))

	switch {
	case ft < m1:
		drawPhaseLabel(l, "Merge the input systems into one column")
		drawMerge(d, l, ft/float32(d.TMerge))
	case ft < m2:
		drawPhaseLabel(l, "Qubits in the gate move to the top")
		drawReorder(d, l, (ft-m1)/float32(d.TReorder))
	case ft < m3:
		drawPhaseLabel(l, "Gate "+d.Label)
		drawGatePhase(d, l, ease((ft-m2)/float32(d.TGate)))
	case ft < m4:
		drawPhaseLabel(l, "Compute: each state spreads over the gate rows")
		drawIterate(d, l, ft-m3)
	default:
		drawPhaseLabel(l, "The sum collapses into the new qubit system")
		drawCollapse(d, l, ease((ft-m4)/float32(d.TCollapse)))
	}
}

// drawMerge: source system columns slide together into one Dirac column.
func drawMerge(d *ComputeDemo, l demoLayout, u float32) {
	e := ease(u)
	if len(d.SrcAmps) <= 1 {
		drawStateColumn(d, l.ampX, l.ketX, l.top, d.MergedAmps, d.PreMods, e, -1)
		for p := int32(0); p < d.Size; p++ {
			drawTag(l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PreMods[p], e, false)
		}
		return
	}

	srcAlpha := float32(1)
	mergeAlpha := float32(0)
	if e > 0.6 {
		srcAlpha = 1 - (e-0.6)/0.4
		mergeAlpha = (e - 0.6) / 0.4
	}
	spread := float32(140)
	for si := range d.SrcAmps {
		off := (float32(si) - float32(len(d.SrcAmps)-1)/2) * spread * (1 - e)
		size := d.SrcSizes[si]
		rows := visRows(int32(1) << uint(size))
		for slot, r := range rows {
			y := l.top + dHeaderH + float32(slot)*dRowH
			if r < 0 {
				rl.DrawText("...", int32(l.ampX+off+dAmpW-20), int32(y+6), 13, fadeA(rl.White, srcAlpha*0.7))
				continue
			}
			drawAmp(l.ampX+off, y, d.SrcAmps[si][r], srcAlpha)
			drawKet(l.ketX+off, y, r, size, d.SrcMods[si], srcAlpha)
		}
	}
	drawStateColumn(d, l.ampX, l.ketX, l.top, d.MergedAmps, d.PreMods, mergeAlpha, -1)
	for p := int32(0); p < d.Size; p++ {
		drawTag(l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PreMods[p], mergeAlpha, false)
	}
}

// drawReorder: gate qubits move to the top of the column. Amplitudes travel
// to their permuted rows, qubit tags slide vertically, digit colors fade.
func drawReorder(d *ComputeDemo, l demoLayout, u float32) {
	e := ease(u)
	rows := visRows(int32(1) << uint(d.Size))

	for slot, r := range rows {
		if r < 0 {
			continue
		}
		dstSlot := slotOf(rows, d.Perm[r])
		y1 := l.top + dHeaderH + float32(slot)*dRowH
		y2 := l.top + dHeaderH + float32(dstSlot)*dRowH
		drawAmp(l.ampX, lerp(y1, y2, e), d.MergedAmps[r], 1)
	}
	for slot, r := range rows {
		y := l.top + dHeaderH + float32(slot)*dRowH
		if r < 0 {
			rl.DrawText("...", int32(l.ampX+dAmpW-20), int32(y+6), 13, rl.Fade(rl.White, 0.7))
			continue
		}
		drawKetMix(l.ketX, y, r, d.Size, d.PreMods, d.PostMods, e)
	}
	for p := int32(0); p < d.Size; p++ {
		y1 := l.top + dHeaderH + float32(p)*dTagH
		y2 := l.top + dHeaderH + float32(d.DigitDest[p])*dTagH
		drawTag(l.tagX, lerp(y1, y2, e), d.PreMods[p], 1, d.DigitDest[p] < d.N)
	}
}

// drawGatePhase: the reordered column stays while the gate matrix fades in.
func drawGatePhase(d *ComputeDemo, l demoLayout, e float32) {
	drawStateColumn(d, l.ampX, l.ketX, l.top, d.InAmps, d.PostMods, 1, -1)
	for p := int32(0); p < d.Size; p++ {
		drawTag(l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PostMods[p], 1, p < d.N)
	}
	drawMatrix(d, l, e, -1)
}

// drawIterate: the main loop visualization. The top qubits iterate i from 0
// to 2^N-1, the qubits below traverse l; each step fires light rays through
// the gate matrix into the accumulating sum columns joined by "+".
func drawIterate(d *ComputeDemo, l demoLayout, it float32) {
	dimJ := int32(1) << uint(d.N)
	dimL := int32(1) << uint(d.Shift)

	k := 0
	for i := range d.Steps {
		if it >= float32(d.IterStart[i]) {
			k = i
		} else {
			break
		}
	}
	st := d.Steps[k]
	start := float32(d.IterStart[k])
	end := float32(d.IterTotal)
	if k+1 < len(d.Steps) {
		end = float32(d.IterStart[k+1])
	}
	u := float32(0)
	if end > start {
		u = clamp01((it - start) / (end - start))
	}

	// Accumulate the completed steps into the sum grid.
	grid := make([]complex64, dimJ*dimL)
	written := make([]bool, dimJ*dimL)
	for s := 0; s < k; s++ {
		prev := d.Steps[s]
		for j := int32(0); j < dimJ; j++ {
			v := prev.Contrib[j]
			grid[j*dimL+prev.L] += v
			if real(v) != 0 || imag(v) != 0 {
				written[j*dimL+prev.L] = true
			}
		}
	}
	pop := clamp01((u - 0.85) / 0.15)

	drawStateColumn(d, l.ampX, l.ketX, l.top, d.InAmps, d.PostMods, 1, (st.I<<uint(d.Shift))+st.L)
	for p := int32(0); p < d.Size; p++ {
		drawTag(l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PostMods[p], 1, p < d.N)
	}
	drawMatrix(d, l, 1, st.I)

	for j := int32(0); j < dimJ; j++ {
		x := l.sumX + float32(j)*(dSumColW+dPlusW)
		drawSumColumnAt(d, x, l.top, j, grid, written, 1, 1, &st, pop)
		if j < dimJ-1 {
			plusY := l.top + dHeaderH + float32(visCount(dimL))*dRowH/2 - 10
			rl.DrawText("+", int32(x+dSumColW+dPlusW/2-4), int32(plusY), 20, rl.White)
		}
	}
	drawRays(d, l, st, u)

	counter := fmt.Sprintf("step %d/%d", k+1, len(d.Steps))
	cw := rl.MeasureText(counter, 12)
	rl.DrawText(counter, int32(l.tagX+l.width)-cw, int32(l.top-24), 12, rl.Fade(rl.White, 0.7))
}

// drawCollapse: the sum columns slide together and stack into a single
// column - the new qubit system state.
func drawCollapse(d *ComputeDemo, l demoLayout, e float32) {
	dimJ := int32(1) << uint(d.N)
	dimL := int32(1) << uint(d.Shift)
	grid, written := fullGrid(d)

	sceneA := clamp01(1 - e*1.4)
	if sceneA > 0 {
		drawStateColumn(d, l.ampX, l.ketX, l.top, d.InAmps, d.PostMods, sceneA, -1)
		for p := int32(0); p < d.Size; p++ {
			drawTag(l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PostMods[p], sceneA, false)
		}
		drawMatrix(d, l, sceneA*0.6, -1)
	}

	ketW := 26 + float32(d.Size)*dDigitW
	colW := dAmpW + ketW
	matW := float32(dimJ) * dCellW
	finalX := l.matX + matW/2 - colW/2
	blockH := float32(visCount(dimL)) * dRowH

	colA := clamp01(1 - clamp01((e-0.7)/0.3))
	headA := colA * clamp01(1-e*2.5)
	for j := int32(0); j < dimJ; j++ {
		x := lerp(l.sumX+float32(j)*(dSumColW+dPlusW), finalX, e)
		yOff := lerp(0, float32(j)*blockH, e)
		drawSumColumnAt(d, x, l.top+yOff, j, grid, written, colA, headA, nil, 0)
		if colA > 0 && j < dimJ-1 {
			plusX := lerp(l.sumX+float32(j)*(dSumColW+dPlusW)+dSumColW+dPlusW/2-4, finalX+colW/2-4, e)
			plusY := l.top + dHeaderH + float32(visCount(dimL))*dRowH/2 - 10 + yOff
			rl.DrawText("+", int32(plusX), int32(plusY), 20, fadeA(rl.White, colA))
		}
	}

	fa := clamp01((e - 0.7) / 0.3)
	if fa > 0 {
		rl.DrawText("new state", int32(finalX), int32(l.top+4), 13, fadeA(rl.Gold, fa))
		drawStateColumn(d, finalX, finalX+dAmpW+6, l.top, d.OutAmps, d.PostMods, fa, -1)
		for p := int32(0); p < d.Size; p++ {
			drawTag(finalX-dTagW-8, l.top+dHeaderH+float32(p)*dTagH, d.PostMods[p], fa, false)
		}
	}
}
