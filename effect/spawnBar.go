package effect

import (
	"qsim/globals"
	"qsim/windows"
)

var PinToRight = func(rw *windows.RenderWindow) {
	rw.X = int32(float32(globals.MainWindowWidth) - float32(rw.Width))
	rw.Y = 0
}
