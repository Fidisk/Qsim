//go:build js && wasm

package main

import (
	"fmt"
	"qsim/config"
	"strconv"
	"syscall/js"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// devtools exposes runtime-tweakable config values to the page's dev bar as
// window.qsimSetConfig(key, value) and window.qsimGetConfig(key). Values are
// passed as strings and parsed per target type; the setter returns a short
// status string that the dev bar shows as feedback, the getter returns the
// current value (or nil for unknown keys).
func init() {
	js.Global().Set("qsimGetConfig", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 1 {
			return nil
		}
		f32 := func(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) }
		f64 := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
		i32 := func(v int32) string { return strconv.FormatInt(int64(v), 10) }
		b := func(v bool) string { return strconv.FormatBool(v) }

		switch args[0].String() {
		case "SnapToGridInterval":
			return f32(config.SnapToGridInterval)
		case "PhysicsEnabled":
			return b(config.PhysicsEnabled)
		case "MinZoom":
			return f32(config.MinZoom)
		case "MaxZoom":
			return f32(config.MaxZoom)
		case "QubitSystemStatePause":
			return b(config.QubitSystemStatePause)
		case "ColorBg":
			return config.ColorToHex(config.ColorBg)
		case "ColorTitleBar":
			return config.ColorToHex(config.ColorTitleBar)
		case "HookColor":
			return config.ColorToHex(config.HookColor)
		case "OutputHookColor":
			return config.ColorToHex(config.OutputHookColor)
		case "QubitSystemColor":
			return config.ColorToHex(config.QubitSystemColor)
		case "GateColor":
			return config.ColorToHex(config.GateColor)
		case "ComputeDotArriveDur":
			return f64(config.ComputeDotArriveDur)
		case "ComputeMergeDur":
			return f64(config.ComputeMergeDur)
		case "ComputeReorderDur":
			return f64(config.ComputeReorderDur)
		case "ComputeGateAppearDur":
			return f64(config.ComputeGateAppearDur)
		case "ComputeIterBaseDur":
			return f64(config.ComputeIterBaseDur)
		case "ComputeIterDecay":
			return f64(config.ComputeIterDecay)
		case "ComputeIterFloorDur":
			return f64(config.ComputeIterFloorDur)
		case "ComputeIterCap":
			return i32(config.ComputeIterCap)
		case "ComputeCollapseDur":
			return f64(config.ComputeCollapseDur)
		case "ComputeMaxRows":
			return i32(config.ComputeMaxRows)
		}
		return nil
	}))

	js.Global().Set("qsimSetConfig", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 {
			return "usage: qsimSetConfig(key, value)"
		}
		key := args[0].String()
		raw := args[1].String()

		setF32 := func(p *float32) string {
			f, err := strconv.ParseFloat(raw, 32)
			if err != nil {
				return "not a number: " + raw
			}
			*p = float32(f)
			return "ok"
		}
		setF64 := func(p *float64) string {
			f, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return "not a number: " + raw
			}
			*p = f
			return "ok"
		}
		setInt := func(p *int32) string {
			n, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return "not an int: " + raw
			}
			*p = int32(n)
			return "ok"
		}
		setBool := func(p *bool) string {
			b, err := strconv.ParseBool(raw)
			if err != nil {
				return "not a bool (true/false): " + raw
			}
			*p = b
			return "ok"
		}
		setColor := func(p *rl.Color) string {
			c, err := config.ParseColor(raw)
			if err != nil {
				return err.Error()
			}
			*p = c
			return "ok"
		}

		switch key {
		case "SnapToGridInterval":
			return setF32(&config.SnapToGridInterval)
		case "PhysicsEnabled":
			return setBool(&config.PhysicsEnabled)
		case "MinZoom":
			return setF32(&config.MinZoom)
		case "MaxZoom":
			return setF32(&config.MaxZoom)
		case "QubitSystemStatePause":
			return setBool(&config.QubitSystemStatePause)
		case "ColorBg":
			return setColor(&config.ColorBg)
		case "ColorTitleBar":
			return setColor(&config.ColorTitleBar)
		case "HookColor":
			return setColor(&config.HookColor)
		case "OutputHookColor":
			return setColor(&config.OutputHookColor)
		case "QubitSystemColor":
			return setColor(&config.QubitSystemColor)
		case "GateColor":
			return setColor(&config.GateColor)
		case "ComputeDotArriveDur":
			return setF64(&config.ComputeDotArriveDur)
		case "ComputeMergeDur":
			return setF64(&config.ComputeMergeDur)
		case "ComputeReorderDur":
			return setF64(&config.ComputeReorderDur)
		case "ComputeGateAppearDur":
			return setF64(&config.ComputeGateAppearDur)
		case "ComputeIterBaseDur":
			return setF64(&config.ComputeIterBaseDur)
		case "ComputeIterDecay":
			return setF64(&config.ComputeIterDecay)
		case "ComputeIterFloorDur":
			return setF64(&config.ComputeIterFloorDur)
		case "ComputeIterCap":
			return setInt(&config.ComputeIterCap)
		case "ComputeCollapseDur":
			return setF64(&config.ComputeCollapseDur)
		case "ComputeMaxRows":
			return setInt(&config.ComputeMaxRows)
		}
		return fmt.Sprintf("unknown key: %s", key)
	}))
}
