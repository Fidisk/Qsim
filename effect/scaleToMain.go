package effect

import (
	"qsim/globals"
	"qsim/windows"
)

var ScaleWidthToScreen = func(w *windows.RenderWindow) {
	w.Width = int32(globals.MainWindowWidth)
}

var ScaleHeightToScreen = func(w *windows.RenderWindow) {
	w.Height = int32(globals.MainWindowHeight)
}
