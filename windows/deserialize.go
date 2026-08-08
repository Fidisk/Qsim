package windows

import (
	"encoding/json"

	"qsim/components"
	glob "qsim/globals"
	"qsim/qubits"
	"qsim/qubits/attributes"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type loadCtx struct {
	oldToNew map[int32]interface{}
}

func parseColor(m map[string]interface{}) rl.Color {
	return rl.Color{
		R: uint8(m["r"].(float64)),
		G: uint8(m["g"].(float64)),
		B: uint8(m["b"].(float64)),
		A: uint8(m["a"].(float64)),
	}
}

func parseVec2(m map[string]interface{}) rl.Vector2 {
	return rl.Vector2{X: float32(m["x"].(float64)), Y: float32(m["y"].(float64))}
}

func parseComplex(m map[string]interface{}) complex64 {
	return complex(float32(m["real"].(float64)), float32(m["imag"].(float64)))
}

func LoadState(data string) []interface{} {
	var lines []string
	current := ""
	for _, c := range data {
		if c == '\n' || c == '\r' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}

	var result []interface{}
	for _, line := range lines {
		line = trimSpace(line)
		if line == "" {
			continue
		}
		w := unmarshalWindow(line)
		if w != nil {
			result = append(result, w)
		}
	}
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}

func unmarshalWindow(line string) interface{} {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil
	}
	typeName, _ := raw["type"].(string)
	switch typeName {
	case "Window":
		return unmarshalBaseWindow(raw)
	case "RenderWindow":
		return unmarshalRenderWindow(raw)
	case "TextWindow":
		return unmarshalTextWindow(raw)
	}
	return nil
}

func unmarshalBaseWindow(raw map[string]interface{}) *Window {
	w := NewWindow(0, 0, 100, 100)
	if v, ok := raw["name"]; ok {
		w.Name = v.(string)
	}
	if v, ok := raw["id"]; ok {
		w.ID = int32(v.(float64))
	}
	if v, ok := raw["x"]; ok {
		w.X = int32(v.(float64))
	}
	if v, ok := raw["y"]; ok {
		w.Y = int32(v.(float64))
	}
	if v, ok := raw["width"]; ok {
		w.Width = int32(v.(float64))
	}
	if v, ok := raw["height"]; ok {
		w.Height = int32(v.(float64))
	}
	if v, ok := raw["titleBarHeight"]; ok {
		w.TitleBarHeight = int32(v.(float64))
	}
	if v, ok := raw["resizeMinW"]; ok {
		w.ResizeMinW = int32(v.(float64))
	}
	if v, ok := raw["resizeMinH"]; ok {
		w.ResizeMinH = int32(v.(float64))
	}
	if v, ok := raw["colorBg"]; ok {
		w.ColorBg = parseColor(v.(map[string]interface{}))
	}
	if v, ok := raw["colorTitleBar"]; ok {
		w.ColorTitleBar = parseColor(v.(map[string]interface{}))
	}
	if v, ok := raw["colorText"]; ok {
		w.ColorText = parseColor(v.(map[string]interface{}))
	}
	if v, ok := raw["colorResize"]; ok {
		w.ColorResize = parseColor(v.(map[string]interface{}))
	}
	if v, ok := raw["colorResizeHover"]; ok {
		w.ColorResizeHover = parseColor(v.(map[string]interface{}))
	}
	if v, ok := raw["colorBorder"]; ok {
		w.ColorBorder = parseColor(v.(map[string]interface{}))
	}
	if v, ok := raw["canResize"]; ok {
		w.CanResize = v.(bool)
	}
	if v, ok := raw["canDrag"]; ok {
		w.CanDrag = v.(bool)
	}
	if v, ok := raw["canZoom"]; ok {
		w.CanZoom = v.(bool)
	}
	if v, ok := raw["priority"]; ok {
		w.Priority = int32(v.(float64))
	}
	return w
}

func unmarshalTextWindow(raw map[string]interface{}) *TextWindow {
	windowRaw, _ := raw["window"].(map[string]interface{})
	w := unmarshalBaseWindow(windowRaw)
	tw := &TextWindow{
		Window: *w,
	}
	if v, ok := raw["textBuffer"]; ok {
		tw.TextBuffer = v.(string)
	}
	if v, ok := raw["maxChars"]; ok {
		tw.MaxChars = int(v.(float64))
	}
	if v, ok := raw["isEditable"]; ok {
		tw.IsEditable = v.(bool)
	}
	return tw
}

func unmarshalRenderWindow(raw map[string]interface{}) *RenderWindow {
	windowRaw, _ := raw["window"].(map[string]interface{})
	w := unmarshalBaseWindow(windowRaw)
	rw := &RenderWindow{
		Window: *w,
	}
	if v, ok := raw["cameraZoom"]; ok {
		rw.Camera.Zoom = float32(v.(float64))
	}
	if v, ok := raw["cameraTargetX"]; ok {
		rw.Camera.Target.X = float32(v.(float64))
	}
	if v, ok := raw["cameraTargetY"]; ok {
		rw.Camera.Target.Y = float32(v.(float64))
	}
	if v, ok := raw["cameraOffsetX"]; ok {
		rw.Camera.Offset.X = float32(v.(float64))
	}
	if v, ok := raw["cameraOffsetY"]; ok {
		rw.Camera.Offset.Y = float32(v.(float64))
	}
	if v, ok := raw["cameraRotation"]; ok {
		rw.Camera.Rotation = float32(v.(float64))
	}
	if v, ok := raw["canPan"]; ok {
		rw.CanPan = v.(bool)
	}
	if v, ok := raw["canSpawn"]; ok {
		rw.CanSpawn = v.(bool)
	}
	if v, ok := raw["isHorizontalScrolling"]; ok {
		rw.IsHorizontalScrolling = v.(bool)
	}
	if v, ok := raw["isVerticalScrolling"]; ok {
		rw.IsVerticalScrolling = v.(bool)
	}
	if v, ok := raw["isTitleBarVisible"]; ok {
		rw.IsTitleBarVisible = v.(bool)
	}
	if v, ok := raw["isTitleEditable"]; ok {
		rw.IsTitleEditable = v.(bool)
	}
	// The grid must survive a save/load round trip: NewRenderWindow defaults
	// it on, but the deserializer builds the window directly, so default it
	// on here too when the save predates showGrid.
	if v, ok := raw["showGrid"]; ok {
		rw.ShowGrid = v.(bool)
	} else {
		rw.ShowGrid = true
	}

	compList, ok := raw["components"].([]interface{})
	if !ok {
		return rw
	}

	ctx := &loadCtx{oldToNew: make(map[int32]interface{})}

	var created []components.Component
	for _, c := range compList {
		comp := unmarshalComponent(c.(map[string]interface{}), ctx)
		if comp != nil {
			created = append(created, comp)
		}
	}

	remapReferences(created, ctx)

	for _, comp := range created {
		rw.PushComponent(comp)
	}

	return rw
}

func unmarshalComponent(raw map[string]interface{}, ctx *loadCtx) components.Component {
	typeName, _ := raw["type"].(string)
	switch typeName {
	case "QubitsSystem":
		return unmarshalQubitsSystem(raw, ctx)
	case "Gate":
		return unmarshalGate(raw, ctx)
	case "CollapseGate":
		return unmarshalCollapseGate(raw, ctx)
	case "M4Gate":
		return unmarshalM4Gate(raw, ctx)
	case "CopyGate":
		return unmarshalCopyGate(raw, ctx)
	case "CompareGate":
		return unmarshalCompareGate(raw, ctx)
	case "ControlledGate":
		return unmarshalControlledGate(raw, ctx)
	case "LogicButton":
		return unmarshalLogicButton(raw, ctx)
	case "LogicGate":
		return unmarshalLogicGate(raw, ctx)
	case "Light":
		return unmarshalLight(raw, ctx)
	case "Hook":
		return unmarshalHook(raw, ctx)
	case "InfoTable":
		return unmarshalInfoTable(raw, ctx)
	case "SourceGate":
		return unmarshalSourceGate(raw, ctx)
	case "LogicalBit":
		return unmarshalLogicalBit(raw, ctx)
	case "Button":
		return unmarshalButton(raw)
	case "ToggleButton":
		return unmarshalToggleButton(raw)
	case "Input":
		return unmarshalInput(raw)
	case "Label":
		return unmarshalLabel(raw)
	case "TextBox":
		return unmarshalTextBox(raw)
	case "LineDraw":
		return unmarshalLineDraw(raw)
	case "Circle":
		return unmarshalCircle(raw)
	}
	return nil
}

func unmarshalCircle(raw map[string]interface{}) *components.Circle {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	c := components.NewCircle(center.X, center.Y, radius, color)
	if v, ok := raw["isFixed"]; ok {
		c.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		c.SetWeight(float32(v.(float64)))
	}
	return c
}

func unmarshalQubitsSystem(raw map[string]interface{}, ctx *loadCtx) (qs *components.QubitsSystem) {
	defer func() {
		if r := recover(); r != nil {
			qs = components.NewQubitsSystem(100, 100, 30, rl.Purple)
			qs.Origin = qubits.NewQubitStateManagerFrom([]complex64{1, 0}, []int32{0})
		}
	}()
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	oldID := int32(raw["id"].(float64))

	qs = components.NewQubitsSystem(center.X, center.Y, radius, color)
	qs.Center = center
	qs.Color = color
	ctx.oldToNew[oldID] = qs

	if v, ok := raw["isLogical"]; ok {
		qs.IsLogical = v.(bool)
	}

	if v, ok := raw["isFixed"]; ok {
		qs.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		qs.SetWeight(float32(v.(float64)))
	}

	if v, ok := raw["hookID"]; ok {
		qs.HookID = int32(v.(float64))
	}
	if v, ok := raw["infoHookID"]; ok {
		qs.InfoHookID = int32(v.(float64))
	}

	originRaw, ok := raw["origin"].(map[string]interface{})
	if ok && originRaw != nil {
		qsm := unmarshalQubitStateManager(originRaw)
		qs.Assign(qsm)
	}

	detsRaw, ok := raw["qubitDeterminators"].([]interface{})
	if ok {
		for i, dRaw := range detsRaw {
			d := dRaw.(map[string]interface{})
			dPos := parseVec2(d["center"].(map[string]interface{}))
			var det *components.QubitDeterminator
			if i < len(qs.QubitDeterminatorList) {
				det = qs.QubitDeterminatorList[i]
				det.Center = dPos
			} else {
				modID := int32(0)
				if v, ok := d["modifierID"]; ok {
					modID = int32(v.(float64))
				}
				detColor := rl.Purple
				if v, ok := d["color"]; ok {
					detColor = parseColor(v.(map[string]interface{}))
				}
				detRadius := float32(10)
				if v, ok := d["radius"]; ok {
					detRadius = float32(v.(float64))
				}
				det = components.NewQubitDeterminator(dPos.X, dPos.Y, detRadius, detColor, modID)
				det.QubitSystemID = qs.ID
				qs.QubitDeterminatorList = append(qs.QubitDeterminatorList, det)
			}
			if v, ok := d["radius"]; ok {
				det.Radius = float32(v.(float64))
			}
			if v, ok := d["color"]; ok {
				det.Color = parseColor(v.(map[string]interface{}))
			}
			if v, ok := d["modifierID"]; ok {
				det.ModifierID = int32(v.(float64))
			}
			if v, ok := d["id"]; ok {
				ctx.oldToNew[int32(v.(float64))] = det
			}
			if v, ok := d["hookID"]; ok {
				det.HookID = int32(v.(float64))
			}
			if v, ok := d["qubitSystemID"]; ok {
				if newQS, found := ctx.oldToNew[int32(v.(float64))]; found {
					if qs2, ok2 := newQS.(*components.QubitsSystem); ok2 {
						det.QubitSystemID = qs2.ID
					}
				}
			}
		}
	}

	if qs.Origin == nil {
		if len(qs.QubitDeterminatorList) > 0 {
			modIDs := make([]int32, len(qs.QubitDeterminatorList))
			for i, det := range qs.QubitDeterminatorList {
				modIDs[i] = det.ModifierID
			}
			amps := make([]complex64, 1<<len(qs.QubitDeterminatorList))
			if len(amps) > 0 {
				amps[0] = 1
			}
			qs.Origin = qubits.NewQubitStateManagerFrom(amps, modIDs)
		} else {
			qs.Origin = qubits.NewQubitStateManagerFrom([]complex64{1, 0}, []int32{0})
		}
	}

	return
}

func unmarshalQubitStateManager(raw map[string]interface{}) *qubits.QubitStateManager {
	ampsRaw, _ := raw["amplitudes"].([]interface{})
	amps := make([]complex64, len(ampsRaw))
	for i, a := range ampsRaw {
		amps[i] = parseComplex(a.(map[string]interface{}))
	}
	modIDsRaw, _ := raw["modifierIDs"].([]interface{})
	modIDs := make([]int32, len(modIDsRaw))
	for i, m := range modIDsRaw {
		modIDs[i] = int32(m.(float64))
	}
	// Register any modifier IDs that don't exist yet in the AttrManager
	// (they're session-persistent but reset on program restart)
	for _, id := range modIDs {
		for attributes.QubitModifierID <= id {
			attributes.GenerateQubitModifierID()
		}
	}
	return qubits.NewQubitStateManagerFrom(amps, modIDs)
}

func unmarshalCollapseGate(raw map[string]interface{}, ctx *loadCtx) *components.CollapseGate {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	label, _ := raw["label"].(string)
	normalSystem, _ := raw["normalSystem"].(bool)

	var g *components.CollapseGate
	if normalSystem {
		g = components.NewCollapseGate3(center.X, center.Y, radius, color, label)
	} else {
		g = components.NewCollapseGate(center.X, center.Y, radius, color, label)
	}
	g.Center = center
	g.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = g

	if v, ok := raw["isFixed"]; ok {
		g.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		g.SetWeight(float32(v.(float64)))
	}
	if v, ok := raw["forceMode"]; ok {
		g.ForceMode = int32(v.(float64))
	}
	if v, ok := raw["outcomeProbs"]; ok {
		if arr, ok2 := v.([]interface{}); ok2 && len(arr) == 2 {
			g.OutcomeProbs[0] = arr[0].(float64)
			g.OutcomeProbs[1] = arr[1].(float64)
		}
	}

	hooksRaw, ok := raw["hooks"].([]interface{})
	if ok {
		for i, hRaw := range hooksRaw {
			if i >= len(g.HookList) {
				continue
			}
			unmarshalHookInto(g.HookList[i], hRaw.(map[string]interface{}), ctx)
		}
	}

	return g
}

func unmarshalM4Gate(raw map[string]interface{}, ctx *loadCtx) *components.M4Gate {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))

	g := components.NewM4Gate(center.X, center.Y, radius, color)
	g.Center = center
	g.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = g

	if v, ok := raw["isFixed"]; ok {
		g.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		g.SetWeight(float32(v.(float64)))
	}
	if v, ok := raw["forceMode"]; ok {
		g.ForceMode = int32(v.(float64))
	}
	if v, ok := raw["skipRandom"]; ok {
		g.SkipRandom = v.(bool)
	}
	if v, ok := raw["frameCount"]; ok {
		g.FrameCount = int(v.(float64))
	}
	if v, ok := raw["swapInterval"]; ok {
		g.SwapInterval = int(v.(float64))
	}
	if v, ok := raw["measured"]; ok {
		g.Measured = v.(bool)
	}
	if v, ok := raw["result"]; ok {
		g.Result = int32(v.(float64))
	}
	if v, ok := raw["hasRemainder"]; ok {
		g.HasRemainder = v.(bool)
	}
	if v, ok := raw["outcomeProbs"]; ok {
		if arr, ok2 := v.([]interface{}); ok2 && len(arr) == 2 {
			g.OutcomeProbs[0] = arr[0].(float64)
			g.OutcomeProbs[1] = arr[1].(float64)
		}
	}
	if v, ok := raw["inputConsumed"]; ok {
		g.InputConsumed = v.(bool)
	}
	if v, ok := raw["storedPos"]; ok {
		g.StoredPos = int32(v.(float64))
	}
	if v, ok := raw["storedSysID"]; ok {
		g.StoredSysID = int32(v.(float64))
	}
	if v, ok := raw["storedInput"]; ok {
		if si, ok2 := v.(map[string]interface{}); ok2 {
			g.StoredInput = unmarshalQubitStateManager(si)
		}
	}

	hooksRaw, ok := raw["hooks"].([]interface{})
	if ok {
		for i, hRaw := range hooksRaw {
			if i >= len(g.HookList) {
				continue
			}
			unmarshalHookInto(g.HookList[i], hRaw.(map[string]interface{}), ctx)
		}
	}

	return g
}

func unmarshalCopyGate(raw map[string]interface{}, ctx *loadCtx) *components.CopyGate {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	label, _ := raw["label"].(string)

	cg := components.NewCopyGate(center.X, center.Y, radius, color, label)
	cg.Center = center
	cg.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = cg

	if v, ok := raw["isFixed"]; ok {
		cg.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		cg.SetWeight(float32(v.(float64)))
	}
	if v, ok := raw["copyID"]; ok {
		cg.CopyID = int32(v.(float64))
	}

	if hookRaw, ok := raw["inHook"].(map[string]interface{}); ok {
		unmarshalHookInto(cg.InHook, hookRaw, ctx)
	}
	if hookRaw, ok := raw["outHook"].(map[string]interface{}); ok {
		unmarshalHookInto(cg.OutHook, hookRaw, ctx)
	}

	return cg
}

func unmarshalCompareGate(raw map[string]interface{}, ctx *loadCtx) *components.CompareGate {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	label, _ := raw["label"].(string)

	cg := components.NewCompareGate(center.X, center.Y, radius, color, label)
	cg.Center = center
	cg.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = cg

	if v, ok := raw["isFixed"]; ok {
		cg.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		cg.SetWeight(float32(v.(float64)))
	}
	if v, ok := raw["outputID"]; ok {
		cg.OutputID = int32(v.(float64))
	}

	if hookRaw, ok := raw["inA"].(map[string]interface{}); ok {
		unmarshalHookInto(cg.InA, hookRaw, ctx)
	}
	if hookRaw, ok := raw["inB"].(map[string]interface{}); ok {
		unmarshalHookInto(cg.InB, hookRaw, ctx)
	}
	if hookRaw, ok := raw["outHook"].(map[string]interface{}); ok {
		unmarshalHookInto(cg.OutHook, hookRaw, ctx)
	}

	return cg
}

func unmarshalControlledGate(raw map[string]interface{}, ctx *loadCtx) *components.ControlledGate {
	center := parseVec2(raw["center"].(map[string]interface{}))
	color := parseColor(raw["color"].(map[string]interface{}))
	kind := int32(0)
	if v, ok := raw["kind"]; ok {
		kind = int32(v.(float64))
	}

	cg := components.NewControlledGate(center.X, center.Y, color, kind)
	cg.Center = center
	cg.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = cg

	if v, ok := raw["isFixed"]; ok {
		cg.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		cg.SetWeight(float32(v.(float64)))
	}

	if hookRaw, ok := raw["inQubit"].(map[string]interface{}); ok {
		unmarshalHookInto(cg.InQubit, hookRaw, ctx)
	}
	if hookRaw, ok := raw["inControl"].(map[string]interface{}); ok {
		unmarshalHookInto(cg.InControl, hookRaw, ctx)
	}
	if hookRaw, ok := raw["outHook"].(map[string]interface{}); ok {
		unmarshalHookInto(cg.OutHook, hookRaw, ctx)
	}

	return cg
}

func unmarshalLogicButton(raw map[string]interface{}, ctx *loadCtx) *components.LogicButton {
	center := parseVec2(raw["center"].(map[string]interface{}))
	color := parseColor(raw["color"].(map[string]interface{}))

	lb := components.NewLogicButton(center.X, center.Y, color)
	lb.Center = center
	lb.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = lb

	if v, ok := raw["isFixed"]; ok {
		lb.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		lb.SetWeight(float32(v.(float64)))
	}
	if v, ok := raw["value"]; ok {
		lb.Value = int32(v.(float64))
	}
	if v, ok := raw["outputID"]; ok {
		lb.OutputID = int32(v.(float64))
	}

	if hookRaw, ok := raw["outHook"].(map[string]interface{}); ok {
		unmarshalHookInto(lb.OutHook, hookRaw, ctx)
	}

	return lb
}

func unmarshalLogicGate(raw map[string]interface{}, ctx *loadCtx) *components.LogicGate {
	center := parseVec2(raw["center"].(map[string]interface{}))
	color := parseColor(raw["color"].(map[string]interface{}))
	kind := int32(0)
	if v, ok := raw["kind"]; ok {
		kind = int32(v.(float64))
	}

	lg := components.NewLogicGate(center.X, center.Y, color, kind)
	lg.Center = center
	lg.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = lg

	if v, ok := raw["isFixed"]; ok {
		lg.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		lg.SetWeight(float32(v.(float64)))
	}
	if v, ok := raw["outputID"]; ok {
		lg.OutputID = int32(v.(float64))
	}

	if hookRaw, ok := raw["inA"].(map[string]interface{}); ok {
		unmarshalHookInto(lg.InA, hookRaw, ctx)
	}
	if hookRaw, ok := raw["inB"].(map[string]interface{}); ok && lg.InB != nil {
		unmarshalHookInto(lg.InB, hookRaw, ctx)
	}
	if hookRaw, ok := raw["outHook"].(map[string]interface{}); ok {
		unmarshalHookInto(lg.OutHook, hookRaw, ctx)
	}

	return lg
}

func unmarshalLight(raw map[string]interface{}, ctx *loadCtx) *components.Light {
	center := parseVec2(raw["center"].(map[string]interface{}))
	color := parseColor(raw["color"].(map[string]interface{}))

	l := components.NewLight(center.X, center.Y, color)
	l.Center = center
	l.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = l

	if v, ok := raw["isFixed"]; ok {
		l.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		l.SetWeight(float32(v.(float64)))
	}

	if hookRaw, ok := raw["inHook"].(map[string]interface{}); ok {
		unmarshalHookInto(l.InHook, hookRaw, ctx)
	}

	return l
}

func unmarshalGate(raw map[string]interface{}, ctx *loadCtx) *components.Gate {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	label, _ := raw["label"].(string)
	isMeas, _ := raw["isMeasurementGate"].(bool)
	inputCount := int32(raw["inputCount"].(float64))

	var g *components.Gate
	if isMeas {
		g = components.NewMeasurementGate(center.X, center.Y, radius, color, label)
	} else {
		opRaw, ok := raw["operation"].([]interface{})
		var op [][]complex64
		if ok {
			op = make([][]complex64, len(opRaw))
			for i, row := range opRaw {
				rowVals := row.([]interface{})
				op[i] = make([]complex64, len(rowVals))
				for j, val := range rowVals {
					op[i][j] = parseComplex(val.(map[string]interface{}))
				}
			}
		}
		if !isMeas && (op == nil || len(op) == 0) {
			sz := 1 << inputCount
			op = make([][]complex64, sz)
			for i := 0; i < sz; i++ {
				op[i] = make([]complex64, sz)
				op[i][i] = 1
			}
		}
		g = components.NewGate(center.X, center.Y, radius, color, label, op, inputCount)
	}

	g.Center = center
	g.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = g

	if v, ok := raw["isFixed"]; ok {
		g.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		g.SetWeight(float32(v.(float64)))
	}
	if v, ok := raw["outputCount"]; ok {
		g.OutputCount = int32(v.(float64))
	}
	if v, ok := raw["measureResult"]; ok {
		g.MeasureResult = int32(v.(float64))
	}
	if v, ok := raw["editable"]; ok {
		g.Editable = v.(bool)
	}

	hooksRaw, ok := raw["hooks"].([]interface{})
	if ok {
		for i, hRaw := range hooksRaw {
			if i >= len(g.HookList) {
				continue
			}
			unmarshalHookInto(g.HookList[i], hRaw.(map[string]interface{}), ctx)
		}
	}

	return g
}

func unmarshalHook(raw map[string]interface{}, ctx *loadCtx) *components.Hook {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	isOutput, _ := raw["isOutput"].(bool)

	var h *components.Hook
	if isOutput {
		h = components.NewOutputHook(center.X, center.Y, radius, color)
	} else {
		h = components.NewHook(center.X, center.Y, radius, color)
	}
	unmarshalHookInto(h, raw, ctx)
	return h
}

func unmarshalHookInto(h *components.Hook, raw map[string]interface{}, ctx *loadCtx) {
	h.Center = parseVec2(raw["center"].(map[string]interface{}))
	h.Radius = float32(raw["radius"].(float64))
	h.Color = parseColor(raw["color"].(map[string]interface{}))
	if v, ok := raw["id"]; ok {
		ctx.oldToNew[int32(v.(float64))] = h
	}
	if v, ok := raw["isFixed"]; ok {
		h.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		h.SetWeight(float32(v.(float64)))
	}
	if v, ok := raw["isHooked"]; ok {
		h.IsHooked = v.(bool)
	}
	if v, ok := raw["targetID"]; ok {
		h.TargetID = int32(v.(float64))
	}
	if v, ok := raw["isOutput"]; ok {
		h.IsOutput = v.(bool)
	}
	if v, ok := raw["hidden"]; ok {
		h.Hidden = v.(bool)
	}
	if v, ok := raw["label"]; ok {
		h.Label = v.(string)
	}
	if v, ok := raw["allowQubitSystem"]; ok {
		h.AllowQubitSystem = v.(bool)
	}
	if v, ok := raw["allowLogicalBit"]; ok {
		h.AllowLogicalBit = v.(bool)
	}
}

func unmarshalInfoTable(raw map[string]interface{}, ctx *loadCtx) *components.InfoTable {
	center := parseVec2(raw["center"].(map[string]interface{}))
	width := float32(raw["width"].(float64))
	height := float32(raw["height"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))

	it := components.NewInfoTable(center.X, center.Y, width, height, color, nil)
	it.Center = center
	it.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = it

	if hookRaw, ok := raw["hook"].(map[string]interface{}); ok {
		unmarshalHookInto(it.Hook, hookRaw, ctx)
	}

	return it
}

func unmarshalSourceGate(raw map[string]interface{}, ctx *loadCtx) *components.SourceGate {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	label, _ := raw["label"].(string)

	ampsRaw, ok := raw["amplitude"].([]interface{})
	var amps []complex64
	if ok {
		amps = make([]complex64, len(ampsRaw))
		for i, a := range ampsRaw {
			amps[i] = parseComplex(a.(map[string]interface{}))
		}
	} else {
		amps = []complex64{1, 0}
	}

	modID := int32(0)
	if v, ok := raw["modifierID"]; ok {
		modID = int32(v.(float64))
	}

	sg := components.NewSourceGateWithID(center.X, center.Y, radius, color, label, amps, modID)
	sg.Center = center
	sg.Color = color
	ctx.oldToNew[int32(raw["id"].(float64))] = sg

	if v, ok := raw["isFixed"]; ok {
		sg.IsFixed = v.(bool)
	}
	if v, ok := raw["weight"]; ok {
		sg.SetWeight(float32(v.(float64)))
	}

	if hookRaw, ok := raw["outHook"].(map[string]interface{}); ok {
		unmarshalHookInto(sg.OutHook, hookRaw, ctx)
	}

	return sg
}

func unmarshalLogicalBit(raw map[string]interface{}, ctx *loadCtx) *components.LogicalBit {
	center := parseVec2(raw["center"].(map[string]interface{}))
	radius := float32(raw["radius"].(float64))
	value := int32(0)
	if v, ok := raw["value"]; ok {
		value = int32(v.(float64))
	}

	lb := components.NewLogicalBit(center.X, center.Y, radius, value)
	lb.Center = center
	ctx.oldToNew[int32(raw["id"].(float64))] = lb

	// The serialized hookID is not restored: hook links are rebuilt from the
	// hook side during remapReferences (a bit may fan out to several hooks).

	return lb
}

func unmarshalButton(raw map[string]interface{}) *components.Button {
	center := parseVec2(raw["center"].(map[string]interface{}))
	width := float32(raw["width"].(float64))
	height := float32(raw["height"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	label, _ := raw["label"].(string)
	fontSize := int32(raw["fontSize"].(float64))
	return components.NewButton(center.X, center.Y, width, height, color, label, fontSize, nil)
}

func unmarshalToggleButton(raw map[string]interface{}) *components.ToggleButton {
	center := parseVec2(raw["center"].(map[string]interface{}))
	width := float32(raw["width"].(float64))
	height := float32(raw["height"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	label, _ := raw["label"].(string)
	fontSize := int32(raw["fontSize"].(float64))
	return components.NewToggleButton(center.X, center.Y, width, height, color, label, func() bool { return false }, fontSize, nil, nil)
}

func unmarshalInput(raw map[string]interface{}) *components.Input {
	center := parseVec2(raw["center"].(map[string]interface{}))
	width := float32(raw["width"].(float64))
	height := float32(raw["height"].(float64))
	fontSize := int32(raw["fontSize"].(float64))
	maxChars := int(raw["maxChars"].(float64))
	text, _ := raw["text"].(string)
	in := components.NewInput(center.X, center.Y, width, height, fontSize, maxChars, nil)
	in.Text = text
	return in
}

func unmarshalLabel(raw map[string]interface{}) *components.Label {
	center := parseVec2(raw["center"].(map[string]interface{}))
	text, _ := raw["text"].(string)
	fontSize := int32(raw["fontSize"].(float64))
	color := parseColor(raw["color"].(map[string]interface{}))
	return components.NewLabel(center.X, center.Y, text, fontSize, color)
}

func unmarshalTextBox(raw map[string]interface{}) *components.TextBox {
	center := parseVec2(raw["center"].(map[string]interface{}))
	width := float32(raw["width"].(float64))
	height := float32(raw["height"].(float64))
	fontSize := int32(raw["fontSize"].(float64))
	text, _ := raw["text"].(string)
	tb := components.NewTextBox(center.X, center.Y, width, height, fontSize)
	tb.Text = text
	return tb
}

func unmarshalLineDraw(raw map[string]interface{}) *components.LineDraw {
	start := parseVec2(raw["start"].(map[string]interface{}))
	end := parseVec2(raw["end"].(map[string]interface{}))
	lineColor := parseColor(raw["lineColor"].(map[string]interface{}))
	placed, _ := raw["placed"].(bool)
	ld := components.NewLineDraw(start.X, start.Y)
	ld.End = end
	ld.LineColor = lineColor
	ld.Placing = !placed
	return ld
}

func remapReferences(comps []components.Component, ctx *loadCtx) {
	for _, comp := range comps {
		switch v := comp.(type) {
		case *components.QubitsSystem:
			remapQubitsSystemRefs(v, ctx)
		case *components.Gate:
			remapGateRefs(v, ctx)
		case *components.M4Gate:
			remapM4GateRefs(v, ctx)
		case *components.CollapseGate:
			remapCollapseGateRefs(v, ctx)
		case *components.InfoTable:
			remapInfoTableRefs(v, ctx)
		case *components.SourceGate:
			remapSourceGateRefs(v, ctx)
		case *components.CopyGate:
			remapCopyGateRefs(v, ctx)
		case *components.CompareGate:
			remapCompareGateRefs(v, ctx)
		case *components.ControlledGate:
			remapControlledGateRefs(v, ctx)
		case *components.LogicButton:
			remapLogicButtonRefs(v, ctx)
		case *components.LogicGate:
			remapLogicGateRefs(v, ctx)
		case *components.Light:
			remapLightRefs(v, ctx)
		}
	}
}

func remapQubitsSystemRefs(qs *components.QubitsSystem, ctx *loadCtx) {
	if qs.HookID != 0 {
		if newObj, found := ctx.oldToNew[qs.HookID]; found {
			if h, ok := newObj.(*components.Hook); ok {
				qs.HookID = 0
				qs.SetWeight(glob.QubitSystemWeight)
				h.IsHooked = true
				h.TargetID = qs.ID
				qs.HookID = h.ID
				qs.SetWeight(0)
			}
		}
	}
	if qs.InfoHookID != 0 {
		if newObj, found := ctx.oldToNew[qs.InfoHookID]; found {
			if h, ok := newObj.(*components.Hook); ok {
				h.IsHooked = true
				h.TargetID = qs.ID
				qs.InfoHookID = h.ID
				qs.SetWeight(0)
			}
		}
	}
	for _, d := range qs.QubitDeterminatorList {
		if d.HookID != 0 {
			if newObj, found := ctx.oldToNew[d.HookID]; found {
				if h, ok := newObj.(*components.Hook); ok {
					d.HookID = 0
					d.SetWeight(glob.QubitDeterminatorWeight)
					h.IsHooked = true
					h.TargetID = d.ID
					d.HookID = h.ID
					d.SetWeight(0)
				}
			}
		}
	}
}

func remapGateRefs(g *components.Gate, ctx *loadCtx) {
	for _, h := range g.HookList {
		if h.TargetID != 0 {
			if newObj, found := ctx.oldToNew[h.TargetID]; found {
				h.TargetID = 0
				switch t := newObj.(type) {
				case *components.QubitsSystem:
					t.HookID = 0
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.HookID = h.ID
					t.SetWeight(0)
				case *components.QubitDeterminator:
					t.HookID = 0
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.HookID = h.ID
					t.SetWeight(0)
				case *components.LogicalBit:
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.AddHook(h.ID)
					t.SetWeight(0)
				}
			}
		}
	}
}

func remapCollapseGateRefs(g *components.CollapseGate, ctx *loadCtx) {
	for _, h := range g.HookList {
		if h.TargetID != 0 {
			if newObj, found := ctx.oldToNew[h.TargetID]; found {
				h.TargetID = 0
				switch t := newObj.(type) {
				case *components.QubitsSystem:
					t.HookID = 0
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.HookID = h.ID
					t.SetWeight(0)
				case *components.QubitDeterminator:
					t.HookID = 0
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.HookID = h.ID
					t.SetWeight(0)
				case *components.LogicalBit:
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.AddHook(h.ID)
					t.SetWeight(0)
				}
			}
		}
	}
}

func remapM4GateRefs(g *components.M4Gate, ctx *loadCtx) {
	remapCollapseGateRefs(&g.CollapseGate, ctx)
	if g.StoredSysID != 0 {
		if newObj, found := ctx.oldToNew[g.StoredSysID]; found {
			if qs, ok := newObj.(*components.QubitsSystem); ok {
				g.StoredSysID = qs.ID
			}
		}
	}
}

func remapInfoTableRefs(it *components.InfoTable, ctx *loadCtx) {
	h := it.Hook
	if h.TargetID != 0 {
		if newObj, found := ctx.oldToNew[h.TargetID]; found {
			if qs, ok := newObj.(*components.QubitsSystem); ok {
				qs.InfoHookID = 0
				qs.Center = h.Center
				h.IsHooked = true
				h.TargetID = qs.ID
				qs.InfoHookID = h.ID
				qs.SetWeight(0)
			}
		}
	}
}

func remapSourceGateRefs(sg *components.SourceGate, ctx *loadCtx) {
	h := sg.OutHook
	if h.TargetID != 0 {
		if newObj, found := ctx.oldToNew[h.TargetID]; found {
			if qs, ok := newObj.(*components.QubitsSystem); ok {
				qs.InfoHookID = 0
				qs.Center = h.Center
				h.IsHooked = true
				h.TargetID = qs.ID
				qs.InfoHookID = h.ID
				qs.SetWeight(0)
			}
		}
	}
}

func remapCopyGateRefs(cg *components.CopyGate, ctx *loadCtx) {
	if cg.CopyID != 0 {
		if newObj, found := ctx.oldToNew[cg.CopyID]; found {
			if qs, ok := newObj.(*components.QubitsSystem); ok {
				cg.CopyID = qs.ID
			}
		} else {
			cg.CopyID = 0
		}
	}
}

func remapCompareGateRefs(cg *components.CompareGate, ctx *loadCtx) {
	for _, h := range []*components.Hook{cg.InA, cg.InB, cg.OutHook} {
		isInput := h == cg.InA || h == cg.InB
		if h.TargetID != 0 {
			if newObj, found := ctx.oldToNew[h.TargetID]; found {
				h.TargetID = 0
				switch t := newObj.(type) {
				case *components.QubitsSystem:
					// Compare inputs are read-only: they attach through
					// InfoHookID so the system's source link (HookID) survives.
					if isInput {
						t.InfoHookID = 0
						t.Center = h.Center
						h.IsHooked = true
						h.TargetID = t.ID
						t.InfoHookID = h.ID
						t.SetWeight(0)
						break
					}
					t.HookID = 0
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.HookID = h.ID
					t.SetWeight(0)
				case *components.QubitDeterminator:
					t.HookID = 0
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.HookID = h.ID
					t.SetWeight(0)
				case *components.LogicalBit:
					t.Center = h.Center
					h.IsHooked = true
					h.TargetID = t.ID
					t.AddHook(h.ID)
					t.SetWeight(0)
				}
			}
		}
	}
	if cg.OutputID != 0 {
		if newObj, found := ctx.oldToNew[cg.OutputID]; found {
			if lb, ok := newObj.(*components.LogicalBit); ok {
				cg.OutputID = lb.ID
			}
		} else {
			cg.OutputID = 0
		}
	}
}

func remapControlledGateRefs(cg *components.ControlledGate, ctx *loadCtx) {
	for _, h := range []*components.Hook{cg.InQubit, cg.InControl, cg.OutHook} {
		if h.TargetID == 0 {
			continue
		}
		newObj, found := ctx.oldToNew[h.TargetID]
		if !found {
			continue
		}
		h.TargetID = 0
		switch t := newObj.(type) {
		case *components.QubitsSystem:
			t.HookID = 0
			t.Center = h.Center
			h.IsHooked = true
			h.TargetID = t.ID
			t.HookID = h.ID
			t.SetWeight(0)
		case *components.QubitDeterminator:
			t.HookID = 0
			t.Center = h.Center
			h.IsHooked = true
			h.TargetID = t.ID
			t.HookID = h.ID
			t.SetWeight(0)
		case *components.LogicalBit:
			t.Center = h.Center
			h.IsHooked = true
			h.TargetID = t.ID
			t.AddHook(h.ID)
			t.SetWeight(0)
		}
	}
}

// remapLogicalBitHook reconnects a hook whose target is a LogicalBit: the bit
// anchors to the hook and the hook points at the new bit ID.
func remapLogicalBitHook(h *components.Hook, ctx *loadCtx) {
	if h.TargetID == 0 {
		return
	}
	newObj, found := ctx.oldToNew[h.TargetID]
	if !found {
		return
	}
	h.TargetID = 0
	if t, ok := newObj.(*components.LogicalBit); ok {
		t.Center = h.Center
		h.IsHooked = true
		h.TargetID = t.ID
		t.AddHook(h.ID)
		t.SetWeight(0)
	}
}

func remapOutputID(outputID *int32, ctx *loadCtx) {
	if *outputID == 0 {
		return
	}
	if newObj, found := ctx.oldToNew[*outputID]; found {
		if lb, ok := newObj.(*components.LogicalBit); ok {
			*outputID = lb.ID
		}
	} else {
		*outputID = 0
	}
}

func remapLogicButtonRefs(lb *components.LogicButton, ctx *loadCtx) {
	remapLogicalBitHook(lb.OutHook, ctx)
	remapOutputID(&lb.OutputID, ctx)
}

func remapLogicGateRefs(lg *components.LogicGate, ctx *loadCtx) {
	remapLogicalBitHook(lg.InA, ctx)
	if lg.InB != nil {
		remapLogicalBitHook(lg.InB, ctx)
	}
	remapLogicalBitHook(lg.OutHook, ctx)
	remapOutputID(&lg.OutputID, ctx)
}

func remapLightRefs(l *components.Light, ctx *loadCtx) {
	remapLogicalBitHook(l.InHook, ctx)
}
