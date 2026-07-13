package effect

import (
	"qsim/globals"
	"qsim/windows"
)

var PinToBottom = func(rw *windows.RenderWindow, margin int32) {
	rw.Y = int32(globals.MainWindowHeight) - margin - rw.Height
}
