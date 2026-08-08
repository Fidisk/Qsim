package windows

import (
	"math"
	"qsim/animation"
	"qsim/components"
	"qsim/config"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/qubits/attributes"
	"qsim/utils"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// previewModID is a single qubit modifier ID registered once and shared by
// every qubit-system spawn preview. Previews need a real, registered ID
// because Assign resolves it while building the qubit visuals (a dummy ID
// crashes AttrManager.Get), but creating a fresh one per preview would burn
// a qubit name each time and make spawned-qubit IDs skip every other number.
var previewModID int32 = -1

type RenderWindow struct {
	Window
	Camera rl.Camera2D
	WComp  []components.Component

	// Panning state
	isPanning      bool
	panStartMouse  rl.Vector2 // world coordinate on press
	panStartTarget rl.Vector2 // camera target on press

	CanPan                bool
	CanSpawn              bool
	IsHorizontalScrolling bool
	IsVerticalScrolling   bool

	ShowGrid bool

	postEffect []func()

	pinPosition func() rl.Vector2

	IsTitleBarVisible bool
	IsTitleEditable   bool

	isEditingTitle   bool
	titleEditBuffer  string
	titleEditCounter int

	spawnPreview   components.Component
	spawnPrevState glob.SpawnType
}

// shade scales a color's RGB channels by f, clamped to [0,255].
func shade(c rl.Color, f float32) rl.Color {
	ch := func(v uint8) uint8 {
		x := float32(v) * f
		if x > 255 {
			x = 255
		}
		if x < 0 {
			x = 0
		}
		return uint8(x)
	}
	return rl.NewColor(ch(c.R), ch(c.G), ch(c.B), c.A)
}

func NewRenderWindow(x, y, width, height int32) *RenderWindow {
	rw := &RenderWindow{
		Window:            *NewWindow(x, y, width, height),
		WComp:             nil,
		CanPan:            true,
		CanSpawn:          true,
		ShowGrid:          true,
		postEffect:        nil,
		pinPosition:       nil,
		IsTitleBarVisible: true,
	}
	rw.Camera = rl.Camera2D{
		Offset:   rl.NewVector2(0, 0), // will be set each frame
		Target:   rl.NewVector2(0, 0), // world point the camera looks at
		Rotation: 0,
		Zoom:     1.0,
	}
	rw.updateCameraOffset()
	return rw
}

func (rw *RenderWindow) GetContentRect() rl.Rectangle {
	return rl.Rectangle{
		X:      float32(rw.X),
		Y:      float32(rw.Y + rw.TitleBarHeight),
		Width:  float32(rw.Width),
		Height: float32(rw.Height - rw.TitleBarHeight),
	}
}

func (rw *RenderWindow) updateCameraOffset() {
	contentWidth := float32(rw.Width)
	contentHeight := float32(rw.Height - rw.TitleBarHeight)
	rw.Camera.Offset = rl.NewVector2(
		float32(rw.X)+contentWidth/2,
		float32(rw.Y+rw.TitleBarHeight)+contentHeight/2,
	)
}

// GetWorldMouse converts the screen mouse position to world coordinates
func (rw *RenderWindow) GetWorldMouse() rl.Vector2 {
	return rl.GetScreenToWorld2D(rl.GetMousePosition(), rw.Camera)
}
func (rw *RenderWindow) Draw() {
	// 1. Draw window decorations in screen space (unclipped)
	// Drop shadow for depth
	rl.DrawRectangle(rw.X+5, rw.Y+7, rw.Width, rw.Height, rl.Fade(rl.Black, 0.3))
	// Subtle vertical gradient body. The color is read live from config so
	// qsimSetConfig("ColorBg", ...) repaints the window on the next frame.
	rl.DrawRectangleGradientV(rw.X, rw.Y, rw.Width, rw.Height, shade(config.ColorBg, 1.08), shade(config.ColorBg, 0.9))

	if rw.IsTitleBarVisible {
		rl.DrawRectangle(rw.X, rw.Y, rw.Width, rw.TitleBarHeight, rw.ColorTitleBar)
		// Accent line under the title bar
		rl.DrawRectangle(rw.X, rw.Y+rw.TitleBarHeight-2, rw.Width, 2, rl.Fade(rl.SkyBlue, 0.5))
		if rw.isEditingTitle {
			editBg := rl.NewColor(30, 30, 30, 255)
			rl.DrawRectangleRec(rl.NewRectangle(float32(rw.X+4), float32(rw.Y+3), float32(rw.Width-8), float32(rw.TitleBarHeight-6)), editBg)
			rl.DrawRectangleLinesEx(rl.NewRectangle(float32(rw.X+4), float32(rw.Y+3), float32(rw.Width-8), float32(rw.TitleBarHeight-6)), 1, rl.SkyBlue)
			rl.DrawText(rw.titleEditBuffer, rw.X+8, rw.Y+5, 16, rl.White)
			if (rw.titleEditCounter/20)%2 == 0 {
				textW := rl.MeasureText(rw.titleEditBuffer, 16)
				rl.DrawRectangle(rw.X+8+textW, rw.Y+5, 2, 16, rl.SkyBlue)
			}
		} else {
			rl.DrawText(rw.Name, rw.X+5, rw.Y+5, 16, rw.ColorText)
		}
	}

	// Resize handle
	handleColor := rw.ColorResize
	if rw.cursorOnResize {
		handleColor = rw.ColorResizeHover
	}

	// 2. Activate scissor to clip content to the window's content area
	contentRect := rw.GetContentRect()
	rl.BeginScissorMode(
		int32(contentRect.X),
		int32(contentRect.Y),
		int32(contentRect.Width),
		int32(contentRect.Height),
	)

	// 3. Draw grid overlay and components in world space (camera transforms, scissor clips)
	rl.BeginMode2D(rw.Camera)

	// Grid overlay
	if rw.ShowGrid {
		contentRect := rw.GetContentRect()
		topLeft := rl.GetScreenToWorld2D(
			rl.Vector2{X: contentRect.X, Y: contentRect.Y},
			rw.Camera,
		)
		bottomRight := rl.GetScreenToWorld2D(
			rl.Vector2{X: contentRect.X + contentRect.Width, Y: contentRect.Y + contentRect.Height},
			rw.Camera,
		)
		interval := config.SnapToGridInterval
		minorColor := rl.NewColor(80, 80, 80, 45)
		majorColor := rl.NewColor(100, 100, 100, 90)

		for x := float32(math.Floor(float64(topLeft.X/interval))) * interval; x <= bottomRight.X; x += interval {
			col, thick := minorColor, float32(1)
			if math.Mod(math.Abs(float64(x)), float64(interval*5)) < 0.01 {
				col, thick = majorColor, 1.5
			}
			rl.DrawLineEx(
				rl.Vector2{X: x, Y: topLeft.Y},
				rl.Vector2{X: x, Y: bottomRight.Y},
				thick, col,
			)
		}
		for y := float32(math.Floor(float64(topLeft.Y/interval))) * interval; y <= bottomRight.Y; y += interval {
			col, thick := minorColor, float32(1)
			if math.Mod(math.Abs(float64(y)), float64(interval*5)) < 0.01 {
				col, thick = majorColor, 1.5
			}
			rl.DrawLineEx(
				rl.Vector2{X: topLeft.X, Y: y},
				rl.Vector2{X: bottomRight.X, Y: y},
				thick, col,
			)
		}
	}

	for _, c := range rw.WComp {
		circle := c.GetCircle()
		if circle.IsDragging() {
			offset := circle.VirtualCenter.Subtract(circle.Center)
			children := c.GetChildCircles()
			for _, child := range children {
				child.Center = child.Center.Subtract(offset)
			}
			c.DrawGhost()
			for _, child := range children {
				child.Center = child.Center.Add(offset)
			}
			saved := circle.Center
			circle.Center = circle.VirtualCenter
			c.Draw()
			circle.Center = saved
		} else {
			c.Draw()
		}
	}
	// Qubit determinators draw above every other component so they stay
	// visible (and grabbable) even when overlapping e.g. a gate.
	for _, c := range rw.WComp {
		if qs, ok := c.(*components.QubitsSystem); ok {
			qs.DrawDeterminators()
		}
	}
	// When a qubit wire exits the visible content area its determinator is
	// gone, so draw a virtual one at the edge to show which qubit it is.
	rw.drawEdgeMarkers()
	if rw.CanSpawn && utils.IsMouseState(glob.MouseStateSpawn) {
		state := utils.GetSpawnState()
		if state != glob.None {
			if state != rw.spawnPrevState || rw.spawnPreview == nil {
				rw.spawnPrevState = state
				rw.spawnPreview = rw.makeSpawnPreview(state)
			}
			if rw.spawnPreview != nil {
				mouse := rw.GetWorldMouse()
				c := rw.spawnPreview.GetCircle()
				snapped := rl.Vector2{
					X: utils.SnapToGrid(mouse.X, config.SnapToGridInterval),
					Y: utils.SnapToGrid(mouse.Y, config.SnapToGridInterval),
				}
				delta := snapped.Subtract(c.Center)
				c.Center = snapped
				for _, child := range rw.spawnPreview.GetChildCircles() {
					child.Center = child.Center.Add(delta)
				}
				rw.spawnPreview.DrawGhost()
			}
		} else {
			rw.spawnPreview = nil
			rw.spawnPrevState = glob.None
		}
	} else {
		rw.spawnPreview = nil
		rw.spawnPrevState = glob.None
	}
	if rw.CanSpawn {
		animation.Draw()
	}
	rl.EndMode2D()

	// 4. Disable scissor
	rl.EndScissorMode()

	if rw.CanResize {
		rl.DrawTriangle(
			rl.Vector2{X: float32(rw.X + rw.Width), Y: float32(rw.Y + rw.Height - 15)},
			rl.Vector2{X: float32(rw.X + rw.Width - 15), Y: float32(rw.Y + rw.Height)},
			rl.Vector2{X: float32(rw.X + rw.Width), Y: float32(rw.Y + rw.Height)},
			handleColor,
		)
	}
	rl.DrawRectangleLines(rw.X, rw.Y, rw.Width, rw.Height, rw.ColorBorder)
}

// liangBarsky clips the segment (a,b) against the axis-aligned box
// [minX,maxX]x[minY,maxY] and returns the entry/exit parameters of the
// visible interval. ok is false when the segment misses the box entirely.
func liangBarsky(a, b rl.Vector2, minX, minY, maxX, maxY float32) (tIn, tOut float32, ok bool) {
	t0, t1 := float32(0), float32(1)
	dx, dy := b.X-a.X, b.Y-a.Y
	p := []float32{-dx, dx, -dy, dy}
	q := []float32{a.X - minX, maxX - a.X, a.Y - minY, maxY - a.Y}
	for i := 0; i < 4; i++ {
		if p[i] == 0 {
			if q[i] < 0 {
				return 0, 0, false
			}
			continue
		}
		r := q[i] / p[i]
		if p[i] < 0 {
			if r > t1 {
				return 0, 0, false
			}
			if r > t0 {
				t0 = r
			}
		} else {
			if r < t0 {
				return 0, 0, false
			}
			if r < t1 {
				t1 = r
			}
		}
	}
	return t0, t1, true
}

// drawEdgeMarkers shows a virtual qubit determinator where a wire exits the
// visible content area, so a qubit line leaving the window still says which
// qubit it belongs to. Runs inside the camera transform, in world space.
func (rw *RenderWindow) drawEdgeMarkers() {
	capture := rw.GetContentRect()
	if capture.Width <= 0 || capture.Height <= 0 {
		return
	}
	tl := rl.GetScreenToWorld2D(rl.Vector2{X: capture.X, Y: capture.Y}, rw.Camera)
	br := rl.GetScreenToWorld2D(rl.Vector2{X: capture.X + capture.Width, Y: capture.Y + capture.Height}, rw.Camera)
	minX := float32(math.Min(float64(tl.X), float64(br.X)))
	maxX := float32(math.Max(float64(tl.X), float64(br.X)))
	minY := float32(math.Min(float64(tl.Y), float64(br.Y)))
	maxY := float32(math.Max(float64(tl.Y), float64(br.Y)))
	if maxX <= minX || maxY <= minY {
		return
	}
	box := rl.Rectangle{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}

	for _, comp := range rw.WComp {
		qs, ok := comp.(*components.QubitsSystem)
		if !ok {
			continue
		}
		for _, d := range qs.QubitDeterminatorList {
			if !qs.DeterminatorWireVisible(d) {
				continue
			}
			// The real determinator is on screen; no virtual one needed.
			if rl.CheckCollisionPointRec(d.Center, box) {
				continue
			}

			a, b := qs.Center, d.Center
			_, tOut, ok := liangBarsky(a, b, minX, minY, maxX, maxY)
			if !ok || tOut >= 1 {
				continue
			}
			// Use the real wire curve at the exit parameter so the marker sits
			// exactly where the bezier leaves the window edge.
			exit := components.WirePointAt(a, b, tOut)

			// Nudge the ring just inside the boundary so it stays visible
			// even though it should sit on the edge.
			radius := float32(10)
			if circle := d.GetCircle(); circle != nil && circle.Radius > 0 {
				radius = circle.Radius
			}
			dot := rl.Vector2{
				X: float32(math.Max(float64(exit.X), float64(minX+radius))),
				Y: float32(math.Max(float64(exit.Y), float64(minY+radius))),
			}
			dot.X = float32(math.Min(float64(dot.X), float64(maxX-radius)))
			dot.Y = float32(math.Min(float64(dot.Y), float64(maxY-radius)))

			rl.DrawCircleV(dot, radius, config.ColorBg)
			rl.DrawCircleLinesV(dot, radius, d.Color)
			label := d.DisplayName()
			font := int32(14)
			if int32(radius)*2 < rl.MeasureText(label, font) {
				font = 10
			}
			rl.DrawText(label,
				int32(dot.X)-rl.MeasureText(label, font)/2,
				int32(dot.Y)-font/2,
				font, d.Color)
		}
	}
}

// EnableTitleBar toggles the visible title bar of the window.
func (rw *RenderWindow) EnableTitleBar(enable bool) {
	rw.IsTitleBarVisible = enable
}

func (rw *RenderWindow) Update() {

	// Cursor locking logic fixed: Check if THIS window is holding it to release it
	if !glob.CursorAvailable && !rw.holdingCursor {
		worldMouse := rw.GetWorldMouse()
		tmp := false
		for i := len(rw.WComp) - 1; i >= 0; i-- {
			if qs, ok := rw.WComp[i].(*components.QubitsSystem); ok {
				qs.UpdateDeterminators(worldMouse, rw.holdingCursor, &tmp)
			}
		}
		for i := len(rw.WComp) - 1; i >= 0; i-- {
			rw.WComp[i].Update(worldMouse, rw.holdingCursor, &tmp)
		}

		return
	}

	mousePos := rl.GetMousePosition()
	windowRect := rl.Rectangle{
		X: float32(rw.X), Y: float32(rw.Y),
		Width: float32(rw.Width), Height: float32(rw.Height),
	}

	if rl.CheckCollisionPointRec(mousePos, windowRect) && glob.CursorAvailable {
		rw.holdingCursor = true
		glob.CursorAvailable = false
	} else if glob.CursorAvailable {
		rw.holdingCursor = false
	}

	if rw.holdingCursor && rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		rw.activate = true
	}

	// Window resize/drag in screen space
	rw.handleResize(mousePos)
	if !rw.IsResizing {
		rw.handleDrag(mousePos)
	}

	// Title editing
	titleRect := rl.Rectangle{
		X: float32(rw.X), Y: float32(rw.Y),
		Width: float32(rw.Width), Height: float32(rw.TitleBarHeight),
	}
	if rw.isEditingTitle {
		rw.titleEditCounter++
		key := rl.GetCharPressed()
		for key > 0 {
			if key >= 32 && key <= 125 {
				rw.titleEditBuffer += string(rune(key))
			}
			key = rl.GetCharPressed()
		}
		if rl.IsKeyPressed(rl.KeyBackspace) && len(rw.titleEditBuffer) > 0 {
			rw.titleEditBuffer = rw.titleEditBuffer[:len(rw.titleEditBuffer)-1]
		}
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
			rw.isEditingTitle = false
			rw.Name = rw.titleEditBuffer
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && !rl.CheckCollisionPointRec(mousePos, titleRect) {
			rw.isEditingTitle = false
			rw.Name = rw.titleEditBuffer
		}
	} else if rw.IsTitleEditable {
		if rw.holdingCursor && rl.CheckCollisionPointRec(mousePos, titleRect) && rl.IsMouseButtonPressed(rl.MouseButtonRight) {
			rw.isEditingTitle = true
			rw.titleEditBuffer = rw.Name
			rw.titleEditCounter = 0
		}
	}

	// Update camera offset after possible resize/move
	rw.updateCameraOffset()

	// --- Pan, Scroll, and Zoom ---
	rw.handlePanning(mousePos)
	rw.handleScroll(mousePos)

	contentRect := rw.GetContentRect()

	if rw.pinPosition != nil {
		rw.Camera.Target = rw.pinPosition()
	}

	worldMouse := rw.GetWorldMouse()
	isCursorAvailable := false
	if rw.holdingCursor {
		isCursorAvailable = true
	}
	// Qubit determinators get first pick of the cursor so they stay
	// draggable even when they end up on top of another component
	// (e.g. a gate).
	for i := len(rw.WComp) - 1; i >= 0; i-- {
		if i >= len(rw.WComp) {
			continue
		}
		q, ok := rw.WComp[i].(*components.QubitsSystem)
		if !ok {
			continue
		}
		qs := q
		qs.UpdateDeterminators(worldMouse, rw.holdingCursor, &isCursorAvailable)
	}
	for i := len(rw.WComp) - 1; i >= 0; i-- {
		if i >= len(rw.WComp) {
			continue
		}
		rw.WComp[i].Update(worldMouse, rw.holdingCursor, &isCursorAvailable)
	}

	if rw.holdingCursor && rl.CheckCollisionPointRec(mousePos, contentRect) {

		// Mouse‑wheel zoom (towards cursor) (FIXED: Order of operations)
		rw.handleZoom()

		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			rw.OnClick(worldMouse)
		}
	}

}

func (rw *RenderWindow) PinCamera(pos func() rl.Vector2) {
	rw.pinPosition = pos
}

func (rw *RenderWindow) UnpinCamera() {
	rw.pinPosition = nil
}

func (rw *RenderWindow) OnClick(worldMouse rl.Vector2) {
	switch {
	case utils.IsMouseState(glob.MouseStateSpawn):
		if rw.CanSpawn {
			// Use the preview's snapped position to match the ghost exactly
			pos := rl.Vector2{
				X: utils.SnapToGrid(worldMouse.X, config.SnapToGridInterval),
				Y: utils.SnapToGrid(worldMouse.Y, config.SnapToGridInterval),
			}
			if rw.spawnPreview != nil {
				pos = rw.spawnPreview.GetCircle().Center
			}
			if utils.IsSpawnState(glob.LineDraw) {
				ld := components.NewLineDraw(pos.X, pos.Y)
				rw.PushComponent(ld)
			} else {
				rw.SpawnObject(pos)
			}
			utils.ToggleMouseState(glob.MouseStateSpawn)
			utils.SetSpawnState(glob.None)
		}
	}
}

func (rw *RenderWindow) SpawnObject(snapped rl.Vector2) {
	// snapped is already grid-aligned by the caller
	switch {
	case utils.IsSpawnState(glob.Qubit):
		q := components.NewQubitsSystem(snapped.X, snapped.Y, glob.QubitSystemRadius, config.QubitSystemColor)

		// Fresh qubits start at |0>; clicking the system cycles
		// |0> -> |1> -> |+> -> |-> -> |i> -> |-i>.
		qState := qubits.NewQubitStateManager([]complex64{1, 0}, 1)

		q.Assign(qState)

		rw.PushComponent(q)
		// A determinator spawned on top of a hook auto-connects.
		q.ZipDeterminatorsToHooks()
	case utils.IsSpawnState(glob.Hadamard):
		t := complex(float32(1/math.Sqrt(2)), 0)
		H1 := components.NewGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "H", [][]complex64{{t, t}, {t, -t}}, 1)
		rw.PushComponent(H1)
	case utils.IsSpawnState(glob.X):
		X1 := components.NewGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "X", [][]complex64{{0, 1}, {1, 0}}, 1)
		rw.PushComponent(X1)
	case utils.IsSpawnState(glob.Y):
		Y1 := components.NewGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "Y", [][]complex64{{0, complex64(complex(0, -1))}, {complex64(complex(0, 1)), 0}}, 1)
		rw.PushComponent(Y1)
	case utils.IsSpawnState(glob.Z):
		Z1 := components.NewGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "Z", [][]complex64{{1, 0}, {0, -1}}, 1)
		rw.PushComponent(Z1)
	case utils.IsSpawnState(glob.CX):
		CX := components.NewGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "CX", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)
		rw.PushComponent(CX)
	case utils.IsSpawnState(glob.CY):
		t := complex64(complex(0, 1))
		CY := components.NewGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "CY", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, -t}, {0, 0, t, 0}}, 2)
		rw.PushComponent(CY)
	case utils.IsSpawnState(glob.CZ):
		CZ := components.NewGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "CZ", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1}}, 2)
		rw.PushComponent(CZ)
	case utils.IsSpawnState(glob.Measurement):
		m := components.NewMeasurementGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "M")
		rw.PushComponent(m)
	case utils.IsSpawnState(glob.Measure2):
		m := components.NewCollapseGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "M2")
		rw.PushComponent(m)
	case utils.IsSpawnState(glob.Measure3):
		m := components.NewCollapseGate3(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "M3")
		rw.PushComponent(m)
	case utils.IsSpawnState(glob.Measure4):
		m := components.NewM4Gate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor)
		rw.PushComponent(m)
	case utils.IsSpawnState(glob.Copy):
		cg := components.NewCopyGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "Get")
		rw.PushComponent(cg)
	case utils.IsSpawnState(glob.Compare):
		cmp := components.NewCompareGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor, "==")
		rw.PushComponent(cmp)
	case utils.IsSpawnState(glob.LogicButton):
		lb := components.NewLogicButton(snapped.X, snapped.Y, config.GateColor)
		rw.PushComponent(lb)
	case utils.IsSpawnState(glob.LogicNot):
		lg := components.NewLogicGate(snapped.X, snapped.Y, config.GateColor, components.LogicNot)
		rw.PushComponent(lg)
	case utils.IsSpawnState(glob.LogicAnd):
		lg := components.NewLogicGate(snapped.X, snapped.Y, config.GateColor, components.LogicAnd)
		rw.PushComponent(lg)
	case utils.IsSpawnState(glob.LogicOr):
		lg := components.NewLogicGate(snapped.X, snapped.Y, config.GateColor, components.LogicOr)
		rw.PushComponent(lg)
	case utils.IsSpawnState(glob.Light):
		l := components.NewLight(snapped.X, snapped.Y, config.GateColor)
		rw.PushComponent(l)
	case utils.IsSpawnState(glob.CBitX):
		cg := components.NewControlledGate(snapped.X, snapped.Y, config.GateColor, components.CtrlX)
		rw.PushComponent(cg)
	case utils.IsSpawnState(glob.CBitY):
		cg := components.NewControlledGate(snapped.X, snapped.Y, config.GateColor, components.CtrlY)
		rw.PushComponent(cg)
	case utils.IsSpawnState(glob.CBitZ):
		cg := components.NewControlledGate(snapped.X, snapped.Y, config.GateColor, components.CtrlZ)
		rw.PushComponent(cg)
	case utils.IsSpawnState(glob.ArbGate):
		g := components.NewArbGate(snapped.X, snapped.Y, glob.GateRadius, config.GateColor)
		rw.PushComponent(g)
	case utils.IsSpawnState(glob.Info):
		t1 := components.NewInfoTable(snapped.X, snapped.Y, 260, 40, config.ColorBg, []components.InfoRow{})
		rw.PushComponent(t1)
	case utils.IsSpawnState(glob.GQubit):
		testSource := components.NewSourceGate(snapped.X, snapped.Y, 100, config.GateColor, "Test", []complex64{0, 1})
		rw.PushComponent(testSource)
	case utils.IsSpawnState(glob.TextBox):
		tb := components.NewTextBox(snapped.X, snapped.Y, 200, 30, 16)
		rw.PushComponent(tb)
	}
}

func (rw *RenderWindow) makeSpawnPreview(state glob.SpawnType) components.Component {
	switch {
	case state&glob.Qubit != 0:
		q := components.NewQubitsSystem(0, 0, glob.QubitSystemRadius, config.QubitSystemColor)
		// Assign the same 1-qubit layout the real spawn uses so DrawGhost
		// can render the grid and determinator, with the shared registered
		// preview modifier ID (see previewModID).
		if previewModID < 0 {
			previewModID = attributes.GenerateQubitModifierID()
		}
		q.Assign(qubits.NewQubitStateManagerFrom([]complex64{1, 0}, []int32{previewModID}))
		return q
	case state&glob.Hadamard != 0:
		t := complex(float32(1/math.Sqrt(2)), 0)
		return components.NewGate(0, 0, glob.GateRadius, config.GateColor, "H", [][]complex64{{t, t}, {t, -t}}, 1)
	case state&glob.X != 0:
		return components.NewGate(0, 0, glob.GateRadius, config.GateColor, "X", [][]complex64{{0, 1}, {1, 0}}, 1)
	case state&glob.Y != 0:
		return components.NewGate(0, 0, glob.GateRadius, config.GateColor, "Y", [][]complex64{{0, complex64(complex(0, -1))}, {complex64(complex(0, 1)), 0}}, 1)
	case state&glob.Z != 0:
		return components.NewGate(0, 0, glob.GateRadius, config.GateColor, "Z", [][]complex64{{1, 0}, {0, -1}}, 1)
	case state&glob.CX != 0:
		return components.NewGate(0, 0, glob.GateRadius, config.GateColor, "CX", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, 1}, {0, 0, 1, 0}}, 2)
	case state&glob.CY != 0:
		t := complex64(complex(0, 1))
		return components.NewGate(0, 0, glob.GateRadius, config.GateColor, "CY", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 0, -t}, {0, 0, t, 0}}, 2)
	case state&glob.CZ != 0:
		return components.NewGate(0, 0, glob.GateRadius, config.GateColor, "CZ", [][]complex64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, -1}}, 2)
	case state&glob.Measurement != 0:
		return components.NewMeasurementGate(0, 0, glob.GateRadius, config.GateColor, "M")
	case state&glob.Measure2 != 0:
		return components.NewCollapseGate(0, 0, glob.GateRadius, config.GateColor, "M2")
	case state&glob.Measure3 != 0:
		return components.NewCollapseGate3(0, 0, glob.GateRadius, config.GateColor, "M3")
	case state&glob.Measure4 != 0:
		return components.NewM4Gate(0, 0, glob.GateRadius, config.GateColor)
	case state&glob.Copy != 0:
		return components.NewCopyGate(0, 0, glob.GateRadius, config.GateColor, "Copy")
	case state&glob.Compare != 0:
		return components.NewCompareGate(0, 0, glob.GateRadius, config.GateColor, "==")
	case state&glob.LogicButton != 0:
		return components.NewLogicButton(0, 0, config.GateColor)
	case state&glob.LogicNot != 0:
		return components.NewLogicGate(0, 0, config.GateColor, components.LogicNot)
	case state&glob.LogicAnd != 0:
		return components.NewLogicGate(0, 0, config.GateColor, components.LogicAnd)
	case state&glob.LogicOr != 0:
		return components.NewLogicGate(0, 0, config.GateColor, components.LogicOr)
	case state&glob.Light != 0:
		return components.NewLight(0, 0, config.GateColor)
	case state&glob.CBitX != 0:
		return components.NewControlledGate(0, 0, config.GateColor, components.CtrlX)
	case state&glob.CBitY != 0:
		return components.NewControlledGate(0, 0, config.GateColor, components.CtrlY)
	case state&glob.CBitZ != 0:
		return components.NewControlledGate(0, 0, config.GateColor, components.CtrlZ)
	case state&glob.ArbGate != 0:
		return components.NewArbGate(0, 0, glob.GateRadius, config.GateColor)
	case state&glob.Info != 0:
		return components.NewInfoTable(0, 0, 260, 40, config.ColorBg, []components.InfoRow{})
	case state&glob.GQubit != 0:
		return components.NewSourceGate(0, 0, 100, config.GateColor, "Test", []complex64{0, 1})
	case state&glob.TextBox != 0:
		return components.NewTextBox(0, 0, 200, 30, 16)
	case state&glob.LineDraw != 0:
		return nil
	}
	return nil
}

func (rw *RenderWindow) handleZoom() {
	if !rw.CanZoom {
		return
	}

	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		worldBefore := rw.GetWorldMouse()

		oldZoom := rw.Camera.Zoom
		newZoom := oldZoom + wheel*0.1
		if newZoom < config.MinZoom {
			newZoom = config.MinZoom
		} else if newZoom > config.MaxZoom {
			newZoom = config.MaxZoom
		}

		rw.Camera.Zoom = newZoom

		worldAfter := rw.GetWorldMouse()

		rw.Camera.Target = rl.Vector2Add(rw.Camera.Target,
			rl.Vector2Subtract(worldBefore, worldAfter))
	}
}

// handlePanning processes middle‑mouse panning while the cursor is held inside the content area.
func (rw *RenderWindow) handlePanning(mousePos rl.Vector2) {
	if !rw.CanPan {
		return
	}

	contentRect := rw.GetContentRect()
	if rw.holdingCursor && rl.CheckCollisionPointRec(mousePos, contentRect) {

		// Middle‑mouse panning (FIXED: Using Screen Space to avoid feedback loops)
		if rl.IsMouseButtonPressed(rl.MouseButtonMiddle) {
			rw.isPanning = true
			rw.panStartMouse = rl.GetMousePosition()
			rw.panStartTarget = rw.Camera.Target
		}
		if rl.IsMouseButtonReleased(rl.MouseButtonMiddle) {
			rw.isPanning = false
		}
		if rw.isPanning {
			currentMouse := rl.GetMousePosition()
			deltaScreen := rl.Vector2Subtract(rw.panStartMouse, currentMouse)
			deltaWorld := rl.Vector2Scale(deltaScreen, 1.0/rw.Camera.Zoom)
			rw.Camera.Target = rl.Vector2Add(rw.panStartTarget, deltaWorld)
		}
	}
}

// handleScroll processes mouse‑wheel axis‑constrained scrolling.
func (rw *RenderWindow) handleScroll(mousePos rl.Vector2) {
	if !rw.IsHorizontalScrolling && !rw.IsVerticalScrolling {
		return
	}

	const scrollSpeed float32 = 20.0
	contentRect := rw.GetContentRect()
	if rw.holdingCursor && rl.CheckCollisionPointRec(mousePos, contentRect) {
		wheel := rl.GetMouseWheelMoveV()
		if rw.IsHorizontalScrolling {
			rw.Camera.Target.X += wheel.X * scrollSpeed / rw.Camera.Zoom
		}
		if rw.IsVerticalScrolling {
			rw.Camera.Target.Y -= wheel.Y * scrollSpeed / rw.Camera.Zoom
		}
	}
}

func (rw *RenderWindow) IsPanAllow(val bool) {
	rw.CanPan = true
}

func (rw *RenderWindow) IsHorizontalScrollAllow(val bool) {
	rw.IsHorizontalScrolling = val
}

func (rw *RenderWindow) IsVerticalScrollAllow(val bool) {
	rw.IsVerticalScrolling = val
}

func (rw *RenderWindow) IsSpawnAllow(val bool) {
	rw.CanSpawn = val
}

func (rw *RenderWindow) IsTitleEditableEnable(val bool) {
	rw.IsTitleEditable = val
}

func (rw *RenderWindow) PostUpdate() {
	rw.Window.PostUpdate()

	for _, d := range rw.WComp {
		d.PostUpdate()
	}

	for i := 1; i < len(rw.WComp); i++ {
		if rw.WComp[i-1].IsHoldingCursor() == true {
			rw.WComp[i-1], rw.WComp[i] = rw.WComp[i], rw.WComp[i-1]
		}
	}

	//Bull shat
	for i := range rw.WComp {
		for j := i + 1; j < len(rw.WComp); j++ {
			rw.WComp[i].GetCircle().AntiGravity(rw.WComp[j].GetCircle())
		}
	}

	//AHhhhhhhh
	for _, d := range rw.postEffect {
		d()
	}
}

func (rw *RenderWindow) GetElement() []components.Component {
	//Danger zone
	return rw.WComp
}

func (rw *RenderWindow) PushComponent(val ...components.Component) {
	for _, d := range val {
		d.SetParent(rw)
		rw.WComp = append(rw.WComp, d)
	}
}

// No rolling cause I'm lazy
func (rw *RenderWindow) RemoveComponent(index int) {
	if index < 0 || index >= len(rw.WComp) {
		return
	}

	// 1. Shift elements over
	copy(rw.WComp[index:], rw.WComp[index+1:])

	// 2. Erase the duplicated pointer at the end to prevent memory leaks
	rw.WComp[len(rw.WComp)-1] = nil

	// 3. Shrink the slice
	rw.WComp = rw.WComp[:len(rw.WComp)-1]
}

func (rw *RenderWindow) DeleteChildWithID(idList ...int32) {
	for _, id := range idList {
		for i, d := range rw.WComp {
			if d.GetID() == id {
				rw.RemoveComponent(i)
				utils.DeleteObjectWithID(id)
				break
			}
		}
	}
}

func (rw *RenderWindow) AddEffect(f ...func()) {
	for _, fun := range f {
		rw.postEffect = append(rw.postEffect, fun)
	}
}
