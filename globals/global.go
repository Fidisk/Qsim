package globals

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var CursorLock bool = false
var ColorBg color.RGBA = rl.NewColor(50, 50, 50, 255)
var ColorTitleBar color.RGBA = rl.NewColor(70, 70, 70, 255)
var HookDist float32 = 80
var CurrentID int32 = 0
var ForceDecay float32 = 0.85
var FrictionDelta float32 = 1

var QubitSystemWeight float32 = 100
var HookWeight float32 = 100
var GateWeight float32 = 100

var GateToHookDist float32 = 150
var GateToHookGraceDist float32 = 30
var GateToHookPullCoeff float32 = 0.1
