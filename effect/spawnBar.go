package effect

import (
	"qsim/globals"
	"qsim/windows"
)

var PinToSide = func(rw *windows.RenderWindow) {
	w := float32(globals.MainWindowWidth)
	center := float32(rw.X) + float32(rw.Width)/2
	if center < w/4 {
		rw.X = 0
	} else if center > w*3/4 {
		rw.X = int32(w - float32(rw.Width))
	}
	rw.Width = 100
}
