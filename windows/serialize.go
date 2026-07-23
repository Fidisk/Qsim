package windows

import (
	"encoding/json"

	"qsim/components"
	"qsim/qubits"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func (rw *RenderWindow) SaveState() string {
	compList := make([]map[string]interface{}, 0, len(rw.WComp))
	for _, c := range rw.WComp {
		if m := serializeComponent(c); m != nil {
			compList = append(compList, m)
		}
	}
	data := map[string]interface{}{
		"type":                   "RenderWindow",
		"window":                 json.RawMessage(rw.Window.SaveState()),
		"cameraZoom":             rw.Camera.Zoom,
		"cameraTargetX":          rw.Camera.Target.X,
		"cameraTargetY":          rw.Camera.Target.Y,
		"cameraOffsetX":          rw.Camera.Offset.X,
		"cameraOffsetY":          rw.Camera.Offset.Y,
		"cameraRotation":         rw.Camera.Rotation,
		"canPan":                 rw.CanPan,
		"canSpawn":               rw.CanSpawn,
		"isHorizontalScrolling":  rw.IsHorizontalScrolling,
		"isVerticalScrolling":    rw.IsVerticalScrolling,
		"isTitleBarVisible":      rw.IsTitleBarVisible,
		"isTitleEditable":        rw.IsTitleEditable,
		"components":             compList,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func serializeComponent(c components.Component) map[string]interface{} {
	switch v := c.(type) {
	case *components.QubitsSystem:
		return serializeQubitsSystem(v)
	case *components.Gate:
		return serializeGate(v)
	case *components.CollapseGate:
		return serializeCollapseGate(v)
	case *components.CopyGate:
		return serializeCopyGate(v)
	case *components.Hook:
		return serializeHook(v)
	case *components.InfoTable:
		return serializeInfoTable(v)
	case *components.SourceGate:
		return serializeSourceGate(v)
	case *components.Button:
		return serializeButton(v)
	case *components.ToggleButton:
		return serializeToggleButton(v)
	case *components.Input:
		return serializeInput(v)
	case *components.Label:
		return serializeLabel(v)
	case *components.TextBox:
		return serializeTextBox(v)
	case *components.LineDraw:
		return serializeLineDraw(v)
	case *components.Circle:
		return serializeCircle(v)
	default:
		return nil
	}
}

func colorMap(c rl.Color) map[string]uint8 {
	return map[string]uint8{"r": c.R, "g": c.G, "b": c.B, "a": c.A}
}

func vec2Map(v rl.Vector2) map[string]float32 {
	return map[string]float32{"x": v.X, "y": v.Y}
}

func complexMap(c complex64) map[string]float32 {
	return map[string]float32{"real": real(c), "imag": imag(c)}
}

func serializeCircle(c *components.Circle) map[string]interface{} {
	return map[string]interface{}{
		"type":    "Circle",
		"center":  vec2Map(c.Center),
		"radius":  c.Radius,
		"color":   colorMap(c.Color),
		"isFixed": c.IsFixed,
		"weight":  c.GetWeight(),
	}
}

func serializeQubitsSystem(qs *components.QubitsSystem) map[string]interface{} {
	dets := make([]map[string]interface{}, len(qs.QubitDeterminatorList))
	for i, d := range qs.QubitDeterminatorList {
		dets[i] = map[string]interface{}{
			"center":        vec2Map(d.Center),
			"radius":        d.Radius,
			"color":         colorMap(d.Color),
			"modifierID":    d.ModifierID,
			"id":            d.ID,
			"hookID":        d.HookID,
			"qubitSystemID": d.QubitSystemID,
		}
	}
	return map[string]interface{}{
		"type":               "QubitsSystem",
		"id":                 qs.ID,
		"center":             vec2Map(qs.Center),
		"radius":             qs.Radius,
		"color":              colorMap(qs.Color),
		"isFixed":            qs.IsFixed,
		"weight":             qs.GetWeight(),
		"hookID":             qs.HookID,
		"infoHookID":         qs.InfoHookID,
		"isLogical":          qs.IsLogical,
		"origin":             serializeQubitStateManager(qs.Origin),
		"qubitDeterminators": dets,
	}
}

func serializeQubitStateManager(qm *qubits.QubitStateManager) map[string]interface{} {
	if qm == nil {
		return nil
	}
	amps := make([]map[string]float32, len(qm.Amptitude))
	for i, a := range qm.Amptitude {
		amps[i] = complexMap(a)
	}
	return map[string]interface{}{
		"amplitudes":  amps,
		"modifierIDs": qm.ModifierID,
		"size":        qm.Size,
	}
}

func serializeQubit(q *components.Qubit) map[string]interface{} {
	return map[string]interface{}{
		"rotation":          q.Rotation,
		"rotationDelta":     q.RotationDelta,
		"pathRotation":      q.PathRotation,
		"pathRotationDelta": q.PathRotationDelta,
		"radius":            q.Radius,
		"angle":             q.Angle,
		"angleDelta":        q.AngleDelta,
		"size":              q.Size,
		"ratio":             q.Ratio,
	}
}

func serializeGate(g *components.Gate) map[string]interface{} {
	hooks := make([]map[string]interface{}, len(g.HookList))
	for i, h := range g.HookList {
		hooks[i] = serializeHook(h)
	}
	op := make([][]map[string]float32, len(g.Operation))
	for i, row := range g.Operation {
		opRow := make([]map[string]float32, len(row))
		for j, val := range row {
			opRow[j] = complexMap(val)
		}
		op[i] = opRow
	}
	return map[string]interface{}{
		"type":              "Gate",
		"id":                g.ID,
		"center":            vec2Map(g.Center),
		"radius":            g.Radius,
		"color":             colorMap(g.Color),
		"isFixed":           g.IsFixed,
		"weight":            g.GetWeight(),
		"label":             g.Label,
		"inputCount":        g.InputCount,
		"outputCount":       g.OutputCount,
		"isMeasurementGate": g.IsMeasurementGate,
		"measureResult":     g.MeasureResult,
		"operation":         op,
		"hooks":             hooks,
	}
}

func serializeCollapseGate(g *components.CollapseGate) map[string]interface{} {
	hooks := make([]map[string]interface{}, len(g.HookList))
	for i, h := range g.HookList {
		hooks[i] = serializeHook(h)
	}
	return map[string]interface{}{
		"type":       "CollapseGate",
		"id":         g.ID,
		"center":     vec2Map(g.Center),
		"radius":     g.Radius,
		"color":      colorMap(g.Color),
		"isFixed":    g.IsFixed,
		"weight":     g.GetWeight(),
		"label":      g.Label,
		"inputCount": g.InputCount,
		"forceMode":  g.ForceMode,
		"hooks":      hooks,
	}
}

func serializeCopyGate(cg *components.CopyGate) map[string]interface{} {
	return map[string]interface{}{
		"type":   "CopyGate",
		"id":     cg.ID,
		"center": vec2Map(cg.Center),
		"radius": cg.Radius,
		"color":  colorMap(cg.Color),
		"isFixed": cg.IsFixed,
		"weight": cg.GetWeight(),
		"label":  cg.Label,
		"width":  cg.Width,
		"height": cg.Height,
		"inHook":  serializeHook(cg.InHook),
		"outHook": serializeHook(cg.OutHook),
		"copyID":  cg.CopyID,
	}
}

func serializeHook(h *components.Hook) map[string]interface{} {
	return map[string]interface{}{
		"type":             "Hook",
		"id":               h.ID,
		"center":           vec2Map(h.Center),
		"radius":           h.Radius,
		"color":            colorMap(h.Color),
		"isFixed":          h.IsFixed,
		"weight":           h.GetWeight(),
		"isHooked":         h.IsHooked,
		"targetID":         h.TargetID,
		"isOutput":         h.IsOutput,
		"label":            h.Label,
		"allowQubitSystem": h.AllowQubitSystem,
	}
}

func serializeInfoTable(it *components.InfoTable) map[string]interface{} {
	return map[string]interface{}{
		"type":   "InfoTable",
		"id":     it.ID,
		"center": vec2Map(it.Center),
		"radius": it.Radius,
		"color":  colorMap(it.Color),
		"width":  it.Width,
		"height": it.Height,
		"hook":   serializeHook(it.Hook),
	}
}

func serializeSourceGate(sg *components.SourceGate) map[string]interface{} {
	amps := make([]map[string]float32, len(sg.Amplitude))
	for i, a := range sg.Amplitude {
		amps[i] = complexMap(a)
	}
	return map[string]interface{}{
		"type":       "SourceGate",
		"id":         sg.ID,
		"center":     vec2Map(sg.Center),
		"radius":     sg.Radius,
		"color":      colorMap(sg.Color),
		"isFixed":    sg.IsFixed,
		"weight":     sg.GetWeight(),
		"label":      sg.Label,
		"amplitude":  amps,
		"modifierID": sg.ModifierID,
		"outHook":    serializeHook(sg.OutHook),
	}
}

func serializeButton(b *components.Button) map[string]interface{} {
	return map[string]interface{}{
		"type":     "Button",
		"center":   vec2Map(b.Center),
		"radius":   b.Radius,
		"color":    colorMap(b.Color),
		"width":    b.Width,
		"height":   b.Height,
		"label":    b.Label,
		"fontSize": b.FontSize,
	}
}

func serializeToggleButton(tb *components.ToggleButton) map[string]interface{} {
	return map[string]interface{}{
		"type":     "ToggleButton",
		"center":   vec2Map(tb.Center),
		"radius":   tb.Radius,
		"color":    colorMap(tb.Color),
		"width":    tb.Width,
		"height":   tb.Height,
		"label":    tb.Label,
		"fontSize": tb.FontSize,
	}
}

func serializeInput(in *components.Input) map[string]interface{} {
	return map[string]interface{}{
		"type":     "Input",
		"center":   vec2Map(in.Center),
		"radius":   in.Radius,
		"color":    colorMap(in.Color),
		"width":    in.Width,
		"height":   in.Height,
		"text":     in.Text,
		"fontSize": in.FontSize,
		"maxChars": in.MaxChars,
	}
}

func serializeLabel(l *components.Label) map[string]interface{} {
	return map[string]interface{}{
		"type":     "Label",
		"center":   vec2Map(l.Center),
		"text":     l.Text,
		"fontSize": l.FontSize,
		"color":    colorMap(l.Color),
	}
}

func serializeTextBox(tb *components.TextBox) map[string]interface{} {
	return map[string]interface{}{
		"type":     "TextBox",
		"center":   vec2Map(tb.Center),
		"width":    tb.Width,
		"height":   tb.Height,
		"text":     tb.Text,
		"fontSize": tb.FontSize,
	}
}

func serializeLineDraw(ld *components.LineDraw) map[string]interface{} {
	return map[string]interface{}{
		"type":   "LineDraw",
		"start":  vec2Map(ld.Center),
		"end":    vec2Map(ld.End),
		"lineColor":  colorMap(ld.LineColor),
		"placed": !ld.Placing,
	}
}
