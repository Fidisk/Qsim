package globals

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var CursorLock bool = false
var ColorBg color.RGBA = rl.NewColor(50, 50, 50, 255)
var ColorTitleBar color.RGBA = rl.NewColor(70, 70, 70, 255)
var HookDist float32 = 50
var CurrentID int32 = 0
