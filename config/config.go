package config

import (
	"fmt"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// These are vars (not consts) so the dev bar on the web build can tweak them
// at runtime via window.qsimSetConfig(key, value).
var MaxZoom float32 = 10.0
var MinZoom float32 = 0.1

var QubitSystemStatePause bool = true

var SnapToGridInterval float32 = 100.0
var PhysicsEnabled bool = false

// UI colors. Settable at runtime via qsimSetConfig with either a color name
// ("blue", "lime", ...) or a hex code (#RRGGBB / #RRGGBBAA); see ParseColor.
var ColorBg rl.Color = rl.NewColor(50, 50, 50, 255)
var ColorTitleBar rl.Color = rl.NewColor(70, 70, 70, 255)
var HookColor rl.Color = rl.Blue
var OutputHookColor rl.Color = rl.Pink
var QubitSystemColor rl.Color = rl.Blue
var GateColor rl.Color = rl.Lime

// ParseColor converts a user-supplied string into a color. A leading '#'
// selects hex parsing (#RRGGBB or #RRGGBBAA); anything else is looked up as
// a case-insensitive raylib color name.
func ParseColor(s string) (rl.Color, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		return parseHexColor(s)
	}
	if c, ok := colorNames[strings.ToLower(s)]; ok {
		return c, nil
	}
	return rl.Color{}, fmt.Errorf("unknown color name: %s", s)
}

func parseHexColor(s string) (rl.Color, error) {
	hex := s[1:]
	if len(hex) != 6 && len(hex) != 8 {
		return rl.Color{}, fmt.Errorf("invalid hex color (want #RRGGBB or #RRGGBBAA): %s", s)
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return rl.Color{}, fmt.Errorf("invalid hex color: %s", s)
	}
	if len(hex) == 6 {
		v = v<<8 | 0xff
	}
	return rl.NewColor(uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v)), nil
}

// ColorToHex renders a color as #RRGGBBAA, a form ParseColor accepts.
func ColorToHex(c rl.Color) string {
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

var colorNames = map[string]rl.Color{
	"lightgray":  rl.LightGray,
	"gray":       rl.Gray,
	"darkgray":   rl.DarkGray,
	"yellow":     rl.Yellow,
	"gold":       rl.Gold,
	"orange":     rl.Orange,
	"pink":       rl.Pink,
	"red":        rl.Red,
	"maroon":     rl.Maroon,
	"green":      rl.Green,
	"lime":       rl.Lime,
	"darkgreen":  rl.DarkGreen,
	"skyblue":    rl.SkyBlue,
	"blue":       rl.Blue,
	"darkblue":   rl.DarkBlue,
	"purple":     rl.Purple,
	"violet":     rl.Violet,
	"darkpurple": rl.DarkPurple,
	"beige":      rl.Beige,
	"brown":      rl.Brown,
	"darkbrown":  rl.DarkBrown,
	"white":      rl.White,
	"black":      rl.Black,
	"blank":      rl.Blank,
	"magenta":    rl.Magenta,
	"raywhite":   rl.RayWhite,
}

// Gate-computation demo visualization.

// QubitColors is the demo color palette. In the compute demo each input
// system gets one palette color shared by all its qubits (indexed by source
// order); qColor falls back to per-modifier coloring when the source is
// unknown.
var QubitColors = []rl.Color{
	rl.SkyBlue,
	rl.Orange,
	rl.Lime,
	rl.Pink,
	rl.Yellow,
	rl.Purple,
	rl.Red,
	rl.Green,
}

var ComputeDotArriveDur = 1.0  // seconds for the input dots to fly into the gate
var ComputeMergeDur = 1.6      // source columns merging into one Dirac column
var ComputeReorderDur = 1.4    // gate qubits moving to the top of the column
var ComputeGateAppearDur = 1.0 // gate matrix fade-in
var ComputeIterBaseDur = 1.5   // seconds for the first iteration
var ComputeIterDecay = 0.7     // per-iteration speed-up factor
var ComputeIterFloorDur = 0.15 // fastest per-iteration duration
var ComputeIterCap int32 = 64   // max visualized iterations before fast-forward
var ComputeCollapseDur = 1.6    // sum columns collapsing into the result column
var ComputeMaxRows int32 = 8    // visible rows per column before "..." is used
