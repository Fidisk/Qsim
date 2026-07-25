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

	// ModColor colors every qubit by the source system it came from:
	// qubits of the same input system share one color.
	ModColor map[int32]rl.Color

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
	I, L, J int32
	Contrib complex64 // InAmps[(I<<Shift)+L] * Matrix[J][I]
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

	// One color per source system: every qubit of the same input system
	// shares that system's color throughout the demo.
	d.ModColor = map[int32]rl.Color{}
	for si, mods := range d.SrcMods {
		col := config.QubitColors[si%len(config.QubitColors)]
		for _, m := range mods {
			d.ModColor[m] = col
		}
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
	// j innermost - the matrix is walked cell by cell down each column, one
	// step per cell. Speed ramps up.
	dimI := int32(1) << uint(n)
	dimL := int32(1) << uint(d.Shift)
	count := dimI * dimL * dimI
	if count > config.ComputeIterCap {
		count = config.ComputeIterCap
	}
	t := 0.0
	dur := float64(config.ComputeIterBaseDur)
	for i := int32(0); i < dimI && int32(len(d.Steps)) < count; i++ {
		for l := int32(0); l < dimL && int32(len(d.Steps)) < count; l++ {
			a := d.InAmps[(i<<uint(d.Shift))+l]
			for j := int32(0); j < dimI && int32(len(d.Steps)) < count; j++ {
				d.Steps = append(d.Steps, computeStep{I: i, L: l, J: j, Contrib: a * g.Operation[j][i]})
				d.IterStart = append(d.IterStart, t)
				t += dur
				dur *= float64(config.ComputeIterDecay)
				if dur < float64(config.ComputeIterFloorDur) {
					dur = float64(config.ComputeIterFloorDur)
				}
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
	// input column + gate matrix + result column (same shape as the input)
	width := dTagW + 8 + colW + dGap + matW + dGap + colW

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

// easeOut decelerates toward the end: fast start, gentle settle.
func easeOut(t float32) float32 {
	t = clamp01(t)
	u := 1 - t
	return 1 - u*u*u*u
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

// colorOf returns the qubit's color: the color of the source system it came
// from, falling back to the per-modifier color when unknown.
func (d *ComputeDemo) colorOf(mod int32) rl.Color {
	if c, ok := d.ModColor[mod]; ok {
		return c
	}
	return qColor(mod)
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// drawBrackets draws a tall "|" left and a tall ">" right of a state column
// so the whole stack reads as one Dirac ket. Each bracket is segmented per
// qubit and carries that qubit's (source system) color.
func drawBrackets(d *ComputeDemo, leftX, rightX, top, height float32, mods []int32, alpha float32) {
	n := len(mods)
	if n == 0 || alpha <= 0 {
		return
	}
	const angleW = float32(10)
	const thick = float32(3)
	segH := height / float32(n)
	// right angle bracket: apex points right at the column's middle
	xAt := func(y float32) float32 {
		f := (y - top) / height
		return rightX + angleW*(1-abs32(2*f-1))
	}
	for i, m := range mods {
		col := fadeA(d.colorOf(m), alpha)
		y0 := top + float32(i)*segH
		y1 := y0 + segH
		rl.DrawLineEx(rl.NewVector2(leftX, y0), rl.NewVector2(leftX, y1), thick, col)
		// split at the midpoint so the apex survives even a single segment
		ym := (y0 + y1) / 2
		//rl.DrawLineEx(rl.NewVector2(xAt(y0), y0), rl.NewVector2(xAt(ym), ym), thick, col)
		//rl.DrawLineEx(rl.NewVector2(xAt(ym), ym), rl.NewVector2(xAt(y1), y1), thick, col)
		_, _ = xAt, ym // keep declared while the right-bracket lines are commented out
	}
}

func fmtC(v complex64) string {
	return fmt.Sprintf("%.2f%+.2fi", real(v), imag(v))
}

func drawAmp(x, y float32, v complex64, alpha float32) {
	s := fmtC(v)
	w := rl.MeasureText(s, 13)
	rl.DrawText(s, int32(x+dAmpW-6)-w, int32(y+6), 13, fadeA(rl.White, alpha))
	// probability bar under the amplitude: width ~ |amp|^2
	p := real(v)*real(v) + imag(v)*imag(v)
	if p > 1 {
		p = 1
	}
	if p > 0.001 {
		bw := (dAmpW - 8) * p
		rl.DrawRectangleRec(rl.NewRectangle(x+dAmpW-6-bw, y+21, bw, 2.5), rl.Fade(rl.SkyBlue, 0.5*clamp01(alpha)))
	}
}

// drawKet draws |digits> with each digit colored by its qubit's color.
// 0 digits are dimmed so the 1s stand out.
func drawKet(d *ComputeDemo, x, y float32, s int32, size int32, mods []int32, alpha float32) {
	rl.DrawText("|", int32(x), int32(y+4), 16, fadeA(rl.White, alpha))
	for p := int32(0); p < size; p++ {
		bit := (s >> uint(size-1-p)) & 1
		ch := "0"
		if bit == 1 {
			ch = "1"
		}
		col := rl.White
		if int(p) < len(mods) {
			col = d.colorOf(mods[p])
		}
		a := alpha
		if bit == 0 {
			a = alpha * 0.55
		}
		rl.DrawText(ch, int32(x+10+float32(p)*dDigitW), int32(y+4), 16, fadeA(col, a))
	}
	rl.DrawText(">", int32(x+10+float32(size)*dDigitW), int32(y+4), 16, fadeA(rl.White, alpha))
}

// drawKetMix is drawKet with per-position colors crossfading pre -> post.
func drawKetMix(d *ComputeDemo, x, y float32, s int32, size int32, pre, post []int32, e float32) {
	rl.DrawText("|", int32(x), int32(y+4), 16, rl.White)
	for p := int32(0); p < size; p++ {
		bit := (s >> uint(size-1-p)) & 1
		ch := "0"
		if bit == 1 {
			ch = "1"
		}
		col := rl.White
		if int(p) < len(pre) && int(p) < len(post) {
			col = lerpColor(d.colorOf(pre[p]), d.colorOf(post[p]), e)
		}
		rl.DrawText(ch, int32(x+10+float32(p)*dDigitW), int32(y+4), 16, col)
	}
	rl.DrawText(">", int32(x+10+float32(size)*dDigitW), int32(y+4), 16, rl.White)
}

// drawMiniKet draws a small centered ket used for matrix/sum-column headers.
func drawMiniKet(d *ComputeDemo, cx, y float32, val int32, size int32, mods []int32, alpha float32) {
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
			col = d.colorOf(mods[p])
		}
		rl.DrawText(ch, int32(x+7+float32(p)*9), int32(y), 11, fadeA(col, alpha))
	}
	rl.DrawText(">", int32(x+7+float32(size)*9), int32(y), 11, fadeA(rl.White, alpha))
}

func drawTag(d *ComputeDemo, x, y float32, m int32, alpha float32, glow bool) {
	col := d.colorOf(m)
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

// drawStateColumn draws amplitude + ket rows of a state at (ampX, ketX, top),
// wrapped in tall Dirac brackets so the stack reads as one ket.
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
			// underline the top-N (gate) qubit digits: they drive this step
			for p := int32(0); p < d.N; p++ {
				col := rl.Gold
				if int(p) < len(mods) {
					col = d.colorOf(mods[p])
				}
				dx := ketX + 10 + float32(p)*dDigitW
				rl.DrawRectangleRec(rl.NewRectangle(dx-1, y+22, dDigitW-2, 2.5), fadeA(col, 0.85*alpha))
			}
		}
		drawAmp(ampX, y, amps[r], alpha)
		drawKet(d, ketX, y, r, d.Size, mods, alpha)
	}
	drawBrackets(d, ampX-16, ketX+ketW+6, top+dHeaderH, float32(len(rows))*dRowH, mods, alpha)
}

func drawMatrix(d *ComputeDemo, l demoLayout, alpha float32, hiRow, hiCol int32) {
	dim := int32(1) << uint(d.N)
	for i := int32(0); i < dim; i++ {
		drawMiniKet(d, l.matX+float32(i)*dCellW+dCellW/2, l.top+6, i, d.N, d.PostMods, alpha)
	}
	// underline the active input header
	if hiCol >= 0 && hiCol < dim {
		hx := l.matX + float32(hiCol)*dCellW
		rl.DrawRectangleRec(rl.NewRectangle(hx+4, l.top+22, dCellW-8, 2.5), rl.Fade(rl.Gold, 0.8*clamp01(alpha)))
	}
	for j := int32(0); j < dim; j++ {
		drawMiniKet(d, l.matX-10-(12+float32(d.N)*9)/2, l.top+dHeaderH+float32(j)*dRowH+7, j, d.N, d.PostMods, alpha)
		for i := int32(0); i < dim; i++ {
			x := l.matX + float32(i)*dCellW
			y := l.top + dHeaderH + float32(j)*dRowH
			rect := rl.NewRectangle(x, y, dCellW, dRowH)
			bg := rl.Fade(rl.White, 0.05*clamp01(alpha))
			if i == hiCol && j == hiRow {
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

// drawResultColumn draws the accumulating output state as a single column -
// the 1xn input column times the nxn gate matrix gives another 1xn column:
// result[(j<<shift)+l] += InAmps[(i<<shift)+l] * Matrix[j][i]. Rows that have
// received no contribution yet are dimmed; pend (if non-nil) is the step
// currently arriving, crossfading into its target row with factor pop.
func drawResultColumn(d *ComputeDemo, ampX, ketX, top float32, grid []complex64, written []bool, alpha float32, pend *computeStep, pop float32) {
	rows := visRows(int32(1) << uint(d.Size))
	ketW := 26 + float32(d.Size)*dDigitW
	for slot, r := range rows {
		y := top + dHeaderH + float32(slot)*dRowH
		if r < 0 {
			rl.DrawText("...", int32(ampX+dAmpW-20), int32(y+6), 13, fadeA(rl.White, alpha*0.7))
			continue
		}
		a := alpha
		if !written[r] {
			a = alpha * 0.25
		}
		if pend != nil && (pend.J<<uint(d.Shift))+pend.L == r && pop > 0 {
			rect := rl.NewRectangle(ampX-6, y, dAmpW+ketW+6, dRowH)
			rl.DrawRectangleRec(rect, rl.Fade(rl.Gold, 0.15*clamp01(pop)))
			rl.DrawRectangleLinesEx(rect, 1.5, rl.Fade(rl.Gold, 0.7*clamp01(pop)))
			if written[r] {
				drawAmp(ampX, y, grid[r], alpha*(1-pop))
			}
			drawAmp(ampX, y, grid[r]+pend.Contrib, alpha*pop)
			drawKet(d, ketX, y, r, d.Size, d.PostMods, alpha)
			continue
		}
		drawAmp(ampX, y, grid[r], a)
		drawKet(d, ketX, y, r, d.Size, d.PostMods, a)
	}
	drawBrackets(d, ampX-16, ketX+ketW+6, top+dHeaderH, float32(len(rows))*dRowH, d.PostMods, alpha)
}

// drawRays draws the light ray for the current step: from the active state
// row through the gate matrix cell (J, I) into the result column's row
// (J<<shift)+L. u in [0,1) is the intra-step progress.
func drawRays(d *ComputeDemo, l demoLayout, st computeStep, u float32) {
	rows := visRows(int32(1) << uint(d.Size))

	srcIdx := (st.I << uint(d.Shift)) + st.L
	srcSlot := slotOf(rows, srcIdx)
	y1 := l.top + dHeaderH + float32(srcSlot)*dRowH + dRowH/2
	p1 := rl.NewVector2(l.ketX+16+float32(d.Size)*dDigitW, y1)

	cellX := l.matX + float32(st.I)*dCellW
	cellY := l.top + dHeaderH + float32(st.J)*dRowH
	p2 := rl.NewVector2(cellX, cellY+dRowH/2)
	mid := rl.NewVector2(cellX+dCellW, cellY+dRowH/2)
	dstSlot := slotOf(rows, (st.J<<uint(d.Shift))+st.L)
	y3 := l.top + dHeaderH + float32(dstSlot)*dRowH + dRowH/2
	p3 := rl.NewVector2(l.sumX, y3)

	// light beam: wide soft glow, medium halo, bright core
	rl.DrawLineEx(p1, p2, 7, rl.Fade(rl.Gold, 0.10))
	rl.DrawLineEx(mid, p3, 7, rl.Fade(rl.Gold, 0.10))
	rl.DrawLineEx(p1, p2, 3, rl.Fade(rl.Gold, 0.28))
	rl.DrawLineEx(mid, p3, 3, rl.Fade(rl.Gold, 0.28))
	rl.DrawLineEx(p1, p2, 1.2, rl.Fade(rl.White, 0.55))
	rl.DrawLineEx(mid, p3, 1.2, rl.Fade(rl.White, 0.55))

	var pos rl.Vector2
	if u < 0.5 {
		pos = rl.Vector2Lerp(p1, p2, u*2)
	} else {
		pos = rl.Vector2Lerp(mid, p3, (u-0.5)*2)
	}
	rl.DrawCircleV(pos, 8, rl.Fade(rl.Gold, 0.25))
	rl.DrawCircleV(pos, 5, rl.Fade(rl.White, 0.95))
	rl.DrawCircleLines(int32(pos.X), int32(pos.Y), 8, rl.Fade(rl.Gold, 0.7))
}

func drawPhaseLabel(l demoLayout, s string) {
	rl.DrawText(s, int32(l.tagX), int32(l.top-24), 14, rl.Fade(rl.Gold, 0.9))
}

// drawDemoProgress draws a thin progress bar under the phase label showing
// how far the demo has played, with tick marks at the phase boundaries.
func drawDemoProgress(l demoLayout, total, ft float32, bounds []float32, alpha float32) {
	x := l.tagX
	y := l.top - 7
	w := l.width
	rl.DrawRectangleRec(rl.NewRectangle(x, y, w, 3), rl.Fade(rl.White, 0.12*alpha))
	frac := clamp01(ft / total)
	rl.DrawRectangleRec(rl.NewRectangle(x, y, w*frac, 3), rl.Fade(rl.Gold, 0.85*alpha))
	for _, b := range bounds {
		tx := x + w*clamp01(b/total)
		rl.DrawLineEx(rl.NewVector2(tx, y-2), rl.NewVector2(tx, y+5), 1, rl.Fade(rl.White, 0.5*alpha))
	}
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
	// covers the circuit underneath (phase label included). The panel fades
	// in over the first 0.3s.
	panelA := ease(ft / 0.3)
	pad := float32(14)
	bg := rl.NewRectangle(l.tagX-pad, l.top-30-pad, l.width+2*pad, l.height+30+2*pad)
	rl.DrawRectangleRounded(bg, 0.06, 6, rl.Fade(rl.Black, 0.88*panelA))
	rl.DrawRectangleRoundedLinesEx(bg, 0.06, 6, 2, rl.Fade(rl.Gold, 0.6*panelA))
	drawDemoProgress(l, float32(d.Total), ft, []float32{m1, m2, m3, m4}, panelA)

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
		drawPhaseLabel(l, "Compute: each state flows through the gate, cell by cell")
		drawIterate(d, l, ft-m3)
	default:
		drawPhaseLabel(l, "The result becomes the new qubit system")
		drawCollapse(d, l, ease((ft-m4)/float32(d.TCollapse)))
	}
}

// drawMerge: source system columns slide together into one Dirac column.
// The slide eases out so the columns slow down as they settle into place.
func drawMerge(d *ComputeDemo, l demoLayout, u float32) {
	e := easeOut(u)
	if len(d.SrcAmps) <= 1 {
		a := ease(u)
		drawStateColumn(d, l.ampX, l.ketX, l.top, d.MergedAmps, d.PreMods, a, -1)
		for p := int32(0); p < d.Size; p++ {
			drawTag(d, l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PreMods[p], a, false)
		}
		return
	}

	srcAlpha := float32(1)
	mergeAlpha := float32(0)
	if u > 0.6 {
		srcAlpha = 1 - (u-0.6)/0.4
		mergeAlpha = (u - 0.6) / 0.4
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
			drawKet(d, l.ketX+off, y, r, size, d.SrcMods[si], srcAlpha)
		}
		srcKetW := 26 + float32(size)*dDigitW
		drawBrackets(d, l.ampX+off-16, l.ketX+off+srcKetW+6, l.top+dHeaderH, float32(len(rows))*dRowH, d.SrcMods[si], srcAlpha)
	}
	drawStateColumn(d, l.ampX, l.ketX, l.top, d.MergedAmps, d.PreMods, mergeAlpha, -1)
	for p := int32(0); p < d.Size; p++ {
		drawTag(d, l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PreMods[p], mergeAlpha, false)
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
		drawKetMix(d, l.ketX, y, r, d.Size, d.PreMods, d.PostMods, e)
	}
	for p := int32(0); p < d.Size; p++ {
		y1 := l.top + dHeaderH + float32(p)*dTagH
		y2 := l.top + dHeaderH + float32(d.DigitDest[p])*dTagH
		drawTag(d, l.tagX, lerp(y1, y2, e), d.PreMods[p], 1, d.DigitDest[p] < d.N)
	}
}

// drawGatePhase: the reordered column stays while the gate matrix fades in.
func drawGatePhase(d *ComputeDemo, l demoLayout, e float32) {
	drawStateColumn(d, l.ampX, l.ketX, l.top, d.InAmps, d.PostMods, 1, -1)
	for p := int32(0); p < d.Size; p++ {
		drawTag(d, l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PostMods[p], 1, p < d.N)
	}
	drawMatrix(d, l, e, -1, -1)
}

// drawIterate: the main loop visualization. The top qubits iterate i from 0
// to 2^N-1, the qubits below traverse l; for each (i, l) the gate matrix is
// walked cell by cell down column i, each cell firing a light ray into the
// accumulating result column.
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
		v := prev.Contrib
		grid[prev.J*dimL+prev.L] += v
		if real(v) != 0 || imag(v) != 0 {
			written[prev.J*dimL+prev.L] = true
		}
	}
	pop := clamp01((u - 0.85) / 0.15)

	drawStateColumn(d, l.ampX, l.ketX, l.top, d.InAmps, d.PostMods, 1, (st.I<<uint(d.Shift))+st.L)
	for p := int32(0); p < d.Size; p++ {
		drawTag(d, l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PostMods[p], 1, p < d.N)
	}
	drawMatrix(d, l, 1, st.J, st.I)

	// The result builds up as one column: the 1xn input state times the nxn
	// gate matrix yields another 1xn column.
	drawResultColumn(d, l.sumX, l.sumX+dAmpW+6, l.top, grid, written, 1, &st, pop)
	drawRays(d, l, st, u)

	counter := fmt.Sprintf("step %d/%d", k+1, len(d.Steps))
	cw := rl.MeasureText(counter, 12)
	rl.DrawText(counter, int32(l.tagX+l.width)-cw, int32(l.top-24), 12, rl.Fade(rl.White, 0.7))
}

// drawCollapse: the input column and gate fade away while the result column
// slides to the center and settles as the new qubit system state.
func drawCollapse(d *ComputeDemo, l demoLayout, e float32) {
	sceneA := clamp01(1 - e*1.4)
	if sceneA > 0 {
		drawStateColumn(d, l.ampX, l.ketX, l.top, d.InAmps, d.PostMods, sceneA, -1)
		for p := int32(0); p < d.Size; p++ {
			drawTag(d, l.tagX, l.top+dHeaderH+float32(p)*dTagH, d.PostMods[p], sceneA, false)
		}
		drawMatrix(d, l, sceneA*0.6, -1, -1)
	}

	ketW := 26 + float32(d.Size)*dDigitW
	colW := dAmpW + ketW
	matW := float32(int32(1)<<uint(d.N)) * dCellW
	finalX := l.matX + matW/2 - colW/2

	// The result column is already a single column; it just slides over.
	x := lerp(l.sumX, finalX, e)
	drawStateColumn(d, x, x+dAmpW+6, l.top, d.OutAmps, d.PostMods, 1, -1)

	fa := clamp01((e - 0.7) / 0.3)
	if fa > 0 {
		// settle glow that peaks as the new state materializes
		glow := fa * (1 - fa) * 2
		if glow > 0.01 {
			grect := rl.NewRectangle(finalX-8, l.top, colW+16, dHeaderH+float32(visCount(int32(1)<<uint(d.Size)))*dRowH)
			rl.DrawRectangleRec(grect, rl.Fade(rl.Gold, 0.10*glow))
		}
		rl.DrawText("new state", int32(finalX), int32(l.top+4), 13, fadeA(rl.Gold, fa))
		for p := int32(0); p < d.Size; p++ {
			drawTag(d, finalX-dTagW-8, l.top+dHeaderH+float32(p)*dTagH, d.PostMods[p], fa, false)
		}
	}
}
