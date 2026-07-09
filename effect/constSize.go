package effect

import "qsim/windows"

var ConstSize = func(rw *windows.RenderWindow, w, h int32) {
	rw.Width = w
	rw.Height = h
}
