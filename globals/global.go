package globals

var CursorLock bool = false
var HookDist float32 = 80
var CurrentID int32 = 0
var ForceDecay float32 = 0.85
var FrictionDelta float32 = 1

// TooltipText carries the hover tooltip for the current frame. Buttons write
// it while hovered; the tooltip window reads and clears it once per frame.
var TooltipText string = ""

var QubitSystemWeight float32 = 100
var HookWeight float32 = 100
var GateWeight float32 = 100
var QubitDeterminatorWeight float32 = 100
var QubitSystemCellWidth float32 = 100
var QubitSystemCellHeight float32 = 100

// GateToHookDist is the design distance between a gate/source body center
// and its hooks (snapped to the 100px grid -> 300px). It is large enough
// that the state grid of the hooked system (2^ceil(n/2) cells of 100px,
// centered on the hook) clears the gate body instead of overlapping it.
var GateToHookDist float32 = 250
var InfoCardToHookDist float32 = 40
var GateToHookGraceDist float32 = 30
var GateToHookPullCoeff float32 = 0.1

var HookRadius float32 = 30

var OutputHookRadius float32 = 30

var QubitSystemRadius float32 = 30

var GateRadius float32 = 30

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
	Info
	GQubit
	TextBox
	LineDraw
	Measure2
	Measure3
	Copy
	Compare
	LogicButton
	LogicNot
	LogicAnd
	LogicOr
	Light
	CBitX
	CBitY
	CBitZ
	ArbGate
	Measure4
)
