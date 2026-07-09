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
var QubitDeterminatorWeight float32 = 100
var QubitSystemCellWidth float32 = 100
var QubitSystemCellHeight float32 = 100

var GateToHookDist float32 = 150
var InfoCardToHookDist float32 = 40
var GateToHookGraceDist float32 = 30
var GateToHookPullCoeff float32 = 0.1

var HookColor rl.Color = rl.Blue
var HookRadius float32 = 30

var OutputHookColor rl.Color = rl.Pink
var OutputHookRadius float32 = 30

var QubitSystemRadius float32 = 30
var QubitSystemColor rl.Color = rl.Blue

var GateRadius float32 = 30
var GateColor rl.Color = rl.Lime

var ForceCap float32 = 20

type MouseOperation int

const (
	MouseStateNormal MouseOperation = 1 << iota
	MouseStateDetach
	MouseStateFix
	MouseStateSpawn
	MouseStateErase
)

type SpawnType int

const (
	None SpawnType = 1 << iota
	Qubit
	Hadamard
	X
	Y
	Z
	CX
	CY
	CZ
	Measurement
)
